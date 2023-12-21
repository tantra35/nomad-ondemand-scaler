package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	nomad "github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/tantra35/nomad-ondemand-scaler/nodeprovider"
)

type PoolNodeConsume struct {
	consummer    chan int
	closemonitor chan struct{}
}

type PoolAllocConsume struct {
	consummer    chan string
	closemonitor chan struct{}
}

type Pool struct {
	lock       sync.Mutex
	updrmvlock sync.RWMutex

	logger   hclog.Logger
	fullName string

	nomadNodes  map[string]*structs.Node
	nomadAllocs map[string]*structs.Allocation

	ephemeralnomadNodes  []*structs.Node
	ephemeralnomadAllocs []*structs.Allocation

	poolnodespec *PoolNodeSpec
	nodeProvider nodeprovider.INodeProvider

	countNodesPubCh  []*PoolNodeConsume
	countAllocsPubCh []*PoolAllocConsume
}

func NewPool(_poolnodespec *PoolNodeSpec) (*Pool, error) {
	lpoolProviderAttr, lok := _poolnodespec.Attributes["provider"]
	if !lok {
		return nil, fmt.Errorf("provider attribute must be set in pool")
	}

	var lprovider nodeprovider.INodeProvider
	lproviderInfo := lpoolProviderAttr.GetMapValue()
	if lproviderInfo == nil {
		return nil, fmt.Errorf("provider info is nil")
	}

	lname, lok := lproviderInfo["name"]
	if !lok {
		return nil, fmt.Errorf("provider info have no name attribute")
	}

	switch *lname.GetStringValue() {
	case "awsautoscale":
		lparamsVariant, lok := lproviderInfo["params"]
		if !lok {
			return nil, fmt.Errorf("provider info have no params attribute")
		}

		llprovider, lerr := nodeprovider.Createawsautoscalegroupv2(variantToTypes(lparamsVariant))
		if lerr != nil {
			return nil, fmt.Errorf("can't create awsautoscale provider due: %s", lerr)
		}

		lprovider = llprovider

	case "anynode":
		lprovider, _ = nodeprovider.NewAnyNodeProvider()

	case "karpenter":
		lparamsVariant, lok := lproviderInfo["params"]
		if !lok {
			return nil, fmt.Errorf("provider info have no params attribute")
		}

		lres := _poolnodespec.GetResources()
		lproviderRes := &nodeprovider.K8sKapenterProviderResources{
			Cpu:    lres.Cpu,
			MemMB:  lres.MemMB,
			DiskMB: lres.DiskMB,
		}

		llprovider, lerr := nodeprovider.Createkarpenterprovider(variantToTypes(lparamsVariant), lproviderRes)
		if lerr != nil {
			return nil, fmt.Errorf("can't create karpenter provider due: %s", lerr)
		}

		lprovider = llprovider
	}

	lpoolName := _poolnodespec.GetFullName()

	pool := &Pool{
		logger:       hclog.L().Named("pool").With("pool", lpoolName),
		fullName:     lpoolName,
		poolnodespec: _poolnodespec,
		nodeProvider: lprovider,

		nomadNodes:  map[string]*structs.Node{},
		nomadAllocs: map[string]*structs.Allocation{},
	}

	return pool, nil
}

func (p *Pool) GetName() string {
	return p.fullName
}

func (p *Pool) tryNomadNode(_nomadNode *nomad.Node) bool {
	p.lock.Lock()
	_, alreadyInPool := p.nomadNodes[_nomadNode.ID]
	p.lock.Unlock()

	if alreadyInPool || p.nodeProvider.IsNodeExists(_nomadNode) {
		if _nomadNode.Status != nomad.NodeStatusDown {
			p.lock.Lock()

			p.nomadNodes[_nomadNode.ID] = apiNomadNodeToStructsNode(_nomadNode)
			lnomadNodesCount := len(p.nomadNodes)
			lnomadAllocsCount := len(p.nomadAllocs)

			p.lock.Unlock()

			if !alreadyInPool {
				p.logger.Info(fmt.Sprintf("nomad node %s, added with status: %s(%s), so pool have %d nodes, and %d allocations",
					_nomadNode.ID, _nomadNode.Status, _nomadNode.SchedulingEligibility,
					lnomadNodesCount, lnomadAllocsCount))

				//сообщаем что у нас добавилась новая нода
				p.lock.Lock()
				countNodesPubCh := []*PoolNodeConsume{}

				for _, lcountCh := range p.countNodesPubCh {
					select {
					case lcountCh.consummer <- lnomadNodesCount:
						countNodesPubCh = append(countNodesPubCh, lcountCh)
					case <-lcountCh.closemonitor:
						close(lcountCh.consummer)
					}
				}

				p.countNodesPubCh = countNodesPubCh
				p.lock.Unlock()
			} else {
				p.logger.Debug(fmt.Sprintf("nomad node %s, updated with status: %s(%s)",
					_nomadNode.ID, _nomadNode.Status, _nomadNode.SchedulingEligibility))
			}
		} else {
			p.lock.Lock()
			delete(p.nomadNodes, _nomadNode.ID)
			allocsToRemove := []string{}
			for _, lalloc := range p.nomadAllocs {
				if lalloc.NodeID == _nomadNode.ID {
					allocsToRemove = append(allocsToRemove, lalloc.ID)
				}
			}

			for _, allocToRemove := range allocsToRemove {
				delete(p.nomadAllocs, allocToRemove)
			}

			p.logger.Info(fmt.Sprintf("nomad node %s, deleted with status: %s, so pool have %d nodes, and %d allocations",
				_nomadNode.ID, _nomadNode.Status,
				len(p.nomadNodes), len(p.nomadAllocs)))

			p.lock.Unlock()
		}

		return true
	}

	return false
}

func (p *Pool) tryNomadAllocation(_nomadAllocation *nomad.Allocation) bool {
	p.lock.Lock()
	_, lnodeExists := p.nomadNodes[_nomadAllocation.NodeID]
	p.lock.Unlock()

	if lnodeExists {
		if _nomadAllocation.DesiredStatus == nomad.AllocDesiredStatusRun &&
			(_nomadAllocation.ClientStatus == nomad.AllocClientStatusPending || _nomadAllocation.ClientStatus == nomad.AllocClientStatusRunning) {

			p.lock.Lock()

			_, isexistingAlloc := p.nomadAllocs[_nomadAllocation.ID]
			p.nomadAllocs[_nomadAllocation.ID] = apiNomadAllocToStructsAlloc(_nomadAllocation)

			lnomadNodesCount := len(p.nomadNodes)
			lnomadAllocsCount := len(p.nomadAllocs)

			p.lock.Unlock()

			if !isexistingAlloc {
				p.logger.Debug(fmt.Sprintf("nomad alloc %s, added with status: %s(%s), so pool have %d nodes, and %d allocations",
					_nomadAllocation.ID, _nomadAllocation.ClientStatus, _nomadAllocation.DesiredStatus,
					lnomadNodesCount, lnomadAllocsCount))

				lexistingEphemeralNodes := map[string]struct{}{}
				var ephemeralToDelete int = -1
				var lephemeralAlloc *structs.Allocation

				p.lock.Lock()

				for i, ephemeralAlloc := range p.ephemeralnomadAllocs {
					if ephemeralToDelete < 0 &&
						ephemeralAlloc.Namespace == _nomadAllocation.Namespace &&
						ephemeralAlloc.JobID == _nomadAllocation.JobID &&
						ephemeralAlloc.TaskGroup == _nomadAllocation.TaskGroup {
						ephemeralToDelete = i
						lephemeralAlloc = ephemeralAlloc
						continue
					}

					lexistingEphemeralNodes[ephemeralAlloc.NodeID] = struct{}{}
				}

				for i, ephemeralNode := range p.ephemeralnomadNodes {
					if _, lok := lexistingEphemeralNodes[ephemeralNode.ID]; !lok {
						p.ephemeralnomadNodes = append(p.ephemeralnomadNodes[:i], p.ephemeralnomadNodes[i+1:]...)
						break
					}
				}

				if ephemeralToDelete >= 0 {
					p.ephemeralnomadAllocs = append(p.ephemeralnomadAllocs[:ephemeralToDelete], p.ephemeralnomadAllocs[ephemeralToDelete+1:]...)
				}

				if lephemeralAlloc != nil {
					countAllocsPubCh := []*PoolAllocConsume{}

					for _, lallocCh := range p.countAllocsPubCh {
						select {
						case lallocCh.consummer <- lephemeralAlloc.ID:
							countAllocsPubCh = append(countAllocsPubCh, lallocCh)
						case <-lallocCh.closemonitor:
							close(lallocCh.consummer)
						}
					}

					p.countAllocsPubCh = countAllocsPubCh
				}

				p.lock.Unlock()
			} else {
				p.logger.Debug(fmt.Sprintf("nomad alloc %s, updated with status: %s(%s)",
					_nomadAllocation.ID, _nomadAllocation.ClientStatus, _nomadAllocation.DesiredStatus))
			}
		} else {
			p.lock.Lock()
			delete(p.nomadAllocs, _nomadAllocation.ID)
			p.logger.Debug(fmt.Sprintf(`nomad alloc %s, deleted with status: %s(%s)", so pool have %d nodes, and %d allocations`,
				_nomadAllocation.ID, _nomadAllocation.ClientStatus, _nomadAllocation.DesiredStatus,
				len(p.nomadNodes), len(p.nomadAllocs)))
			p.lock.Unlock()
		}

		return true
	}

	return false
}

func (p *Pool) Update(_ctx context.Context, _en []*structs.Node, _ea []*structs.Allocation) error {
	var returnerr error
	p.updrmvlock.RLock()
	defer p.updrmvlock.RUnlock()

	logger := p.logger.Named("update")

	p.lock.Lock()

	p.ephemeralnomadNodes = append(p.ephemeralnomadNodes, _en...)
	p.ephemeralnomadAllocs = append(p.ephemeralnomadAllocs, _ea...)
	waitCount := len(p.nomadNodes) + len(p.ephemeralnomadNodes)

	lnomadnodes := make([]*structs.Node, 0, len(p.nomadNodes))
	for _, lnode := range p.nomadNodes {
		lnomadnodes = append(lnomadnodes, lnode)
	}

	logger.Info(fmt.Sprintf("setting size to %d nodes", waitCount))
	lerr := p.nodeProvider.UpdateNode(_ctx, lnomadnodes, int32(waitCount))
	if lerr != nil {
		for _, lea := range _ea {
			for li, noAllocated := range p.ephemeralnomadAllocs {
				if noAllocated.ID == lea.ID {
					p.ephemeralnomadAllocs = append(p.ephemeralnomadAllocs[:li], p.ephemeralnomadAllocs[li+1:]...)
					break
				}
			}
		}

		for _, len := range _en {
			for li, notExistentNode := range p.ephemeralnomadNodes {
				if len.ID == notExistentNode.ID {
					p.ephemeralnomadNodes = append(p.ephemeralnomadNodes[:li], p.ephemeralnomadNodes[li+1:]...)
					break
				}
			}
		}

		p.lock.Unlock()

		return fmt.Errorf("can't set node count due: %s", lerr)
	}

	countNodesCh := &PoolNodeConsume{
		make(chan int),
		make(chan struct{}),
	}
	p.countNodesPubCh = append(p.countNodesPubCh, countNodesCh)

	countAllocsCh := &PoolAllocConsume{
		make(chan string),
		make(chan struct{}),
	}
	p.countAllocsPubCh = append(p.countAllocsPubCh, countAllocsCh)

	p.lock.Unlock()

	var allocationsPlaced []*structs.Allocation
	waitAllocsTimer := time.NewTimer(10 * time.Second)
	waitAllocsTimer.Stop()

WAITLOOP:
	for {
		select {
		case <-_ctx.Done():
			returnerr = _ctx.Err()
			break WAITLOOP

		case <-waitAllocsTimer.C:
			logger.Info(fmt.Sprintf("Waiting for pool update done, only %d ephemeral allocations(%d) are placed", len(allocationsPlaced), len(_ea)))
			break WAITLOOP

		case curCount := <-countNodesCh.consummer:
			if curCount >= waitCount {
				waitAllocsTimer.Reset(10 * time.Second)
			}

		case ephemeralAllocID := <-countAllocsCh.consummer:
			for _, lea := range _ea {
				if lea.ID == ephemeralAllocID {
					allocationsPlaced = append(allocationsPlaced, lea)
					waitAllocsTimer.Reset(10 * time.Second)

					if len(allocationsPlaced) == len(_ea) {
						logger.Info(fmt.Sprintf("Waiting for pool update done, all ephemeral allocations(%d) are placed", len(_ea)))
						break WAITLOOP
					}

					break
				}
			}
		}
	}

	close(countNodesCh.closemonitor)
	close(countAllocsCh.closemonitor)

	p.lock.Lock()

	//в текущем раунде не удалось заплейсить все аллокации сбрасываем их
	for _, lea := range _ea {
		for li, noAllocated := range p.ephemeralnomadAllocs {
			if noAllocated.ID == lea.ID {
				p.ephemeralnomadAllocs = append(p.ephemeralnomadAllocs[:li], p.ephemeralnomadAllocs[li+1:]...)
				break
			}
		}
	}

	for _, len := range _en {
		for li, notExistentNode := range p.ephemeralnomadNodes {
			if len.ID == notExistentNode.ID {
				p.ephemeralnomadNodes = append(p.ephemeralnomadNodes[:li], p.ephemeralnomadNodes[li+1:]...)
				break
			}
		}
	}

	logger.Info(fmt.Sprintf("now pool have %d nodes and %d allocs", len(p.nomadNodes), len(p.nomadAllocs)))
	p.lock.Unlock()

	return returnerr
}

func (p *Pool) WarmUp(_nodesTolaunch int) bool {
	p.updrmvlock.RLock()
	defer p.updrmvlock.RUnlock()

	p.lock.Lock()

	lnomadnodes := make([]*structs.Node, 0, len(p.nomadNodes))
	for _, lnode := range p.nomadNodes {
		lnomadnodes = append(lnomadnodes, lnode)
	}

	p.logger.Info(fmt.Sprintf("adding %d nodes to pool due warmup", _nodesTolaunch))
	countNodesCh := &PoolNodeConsume{
		make(chan int),
		make(chan struct{}),
	}

	p.countNodesPubCh = append(p.countNodesPubCh, countNodesCh)
	waitCount := len(lnomadnodes) + _nodesTolaunch
	p.nodeProvider.UpdateNode(context.TODO(), lnomadnodes, int32(waitCount))
	p.lock.Unlock()

	for curCount := range countNodesCh.consummer {
		if curCount >= waitCount {
			break
		}
	}
	close(countNodesCh.closemonitor)

	return true
}

func (p *Pool) RemoveNode(_nomadNodeIds []string) bool {
	p.updrmvlock.Lock()
	defer p.updrmvlock.Unlock()

	logger := p.logger.Named("remove")

	p.lock.Lock()
	nomadNodes := make([]*nomad.Node, 0, len(_nomadNodeIds))
	for _, lnomadNodeId := range _nomadNodeIds {
		if nomadNode, lok := p.nomadNodes[lnomadNodeId]; lok {
			nomadNodes = append(nomadNodes, structsNomadNodeToApiNode(nomadNode))
		}
	}
	p.lock.Unlock()

	lerr := p.nodeProvider.RemoveNode(nomadNodes)
	if lerr != nil {
		logger.Error(fmt.Sprintf("can't remove nomad nodes due: %s", lerr))
	}

	logger.Info(fmt.Sprintf("removed %s nomad nodes", _nomadNodeIds))
	return true
}

// ----------------------------------------------------------------------------
func createPools(_stalecnf *StaleApiConfig, _nomadClient *nomad.Client, _poolSpecs []*PoolNodeSpec) (map[string]*Pool, uint64, error) {
	lpools := map[string]*Pool{}

	lnqoptions := &nomad.QueryOptions{AllowStale: _stalecnf.Allow}
	nodeList, queryMeta, lerr := _nomadClient.Nodes().List(lnqoptions)
	if lerr == nil {
		if _stalecnf.Allow {
			if queryMeta.LastContact > _stalecnf.StaleAllowedDuration {
				lnqoptions.AllowStale = false
				nodeList, queryMeta, lerr = _nomadClient.Nodes().List(lnqoptions)
			}
		}
	}
	if lerr != nil {
		return nil, 0, fmt.Errorf("can't get nomad nodes due: %s", lerr)
	}

	for _, poolSpec := range _poolSpecs {
		pool, lerr := NewPool(poolSpec)
		if lerr != nil {
			return nil, 0, fmt.Errorf("can't create pool due: %s", lerr)
		}

		lpools[poolSpec.GetFullName()] = pool

		for _, lnomadNodeStub := range nodeList {
			lnqoptions := &nomad.QueryOptions{AllowStale: _stalecnf.Allow}
			nomadNode, lmeta, lerr := _nomadClient.Nodes().Info(lnomadNodeStub.ID, nil)
			if lerr == nil {
				if _stalecnf.Allow {
					if lmeta.LastContact > _stalecnf.StaleAllowedDuration {
						lnqoptions.AllowStale = false
						nomadNode, _, lerr = _nomadClient.Nodes().Info(lnomadNodeStub.ID, nil)
					}
				}
			}
			if lerr != nil {
				return nil, 0, fmt.Errorf("can't get nomad node %s info due: %s", lnomadNodeStub.ID, lerr)
			}

			if pool.tryNomadNode(nomadNode) {
				lnqoptions := nomad.QueryOptions{Namespace: nomad.AllNamespacesNamespace, AllowStale: _stalecnf.Allow}
				nomadNodeAllocations, lmeta, lerr := _nomadClient.Nodes().Allocations(lnomadNodeStub.ID, &lnqoptions)
				if lerr == nil {
					if _stalecnf.Allow {
						if lmeta.LastContact > _stalecnf.StaleAllowedDuration {
							lnqoptions.AllowStale = false
							nomadNodeAllocations, _, lerr = _nomadClient.Nodes().Allocations(lnomadNodeStub.ID, &lnqoptions)
						}
					}
				}
				if lerr != nil {
					return nil, 0, fmt.Errorf("can't get nomad allocations on node %s due: %s", lnomadNodeStub.ID, lerr)
				}

				for _, alloc := range nomadNodeAllocations {
					pool.tryNomadAllocation(alloc)
				}
			}
		}
	}

	return lpools, queryMeta.LastIndex, nil
}
