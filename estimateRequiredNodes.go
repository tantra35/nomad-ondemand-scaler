package main

import (
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/helper/uuid"
	"github.com/hashicorp/nomad/nomad/state"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/scheduler"
)

func estimateRequiredNodes(_pool *Pool, _en []*structs.Node, _ea []*structs.Allocation, _job *structs.Job, _tg *structs.TaskGroup, _unallocatedCount int) ([]*structs.Node, []*structs.Allocation) {
	var lephemeralNodes []*structs.Node
	var lephemeralAllocations []*structs.Allocation

	plan := &structs.Plan{
		EvalID:          uuid.Generate(),
		NodeUpdate:      make(map[string][]*structs.Allocation),
		NodeAllocation:  make(map[string][]*structs.Allocation),
		NodePreemptions: make(map[string][]*structs.Allocation),
	}

	logger := hclog.L().Named("binpaking")

	config := &state.StateStoreConfig{Logger: logger, Region: "global"}
	state, _ := state.NewStateStore(config)
	evlCtx := scheduler.NewEvalContext(nil, state, plan, logger)

	stack := scheduler.NewGenericStack(false, evlCtx)
	ljob := _job.Copy()
	ljob.Status = structs.JobStatusPending

	_pool.lock.Lock()

	lnomadNodes := make([]*structs.Node, 0, len(_pool.ephemeralnomadNodes)+len(_en))
	for _, ephNode := range _pool.ephemeralnomadNodes {
		lnomadNodes = append(lnomadNodes, ephNode)
	}

	for _, ephNode := range _en {
		lnomadNodes = append(lnomadNodes, ephNode)
	}

	stack.SetNodes(lnomadNodes)
	stack.SetJob(_job)

	for _, ephAlloc := range _pool.ephemeralnomadAllocs {
		plan.AppendAlloc(ephAlloc, nil)
	}

	for _, ephAlloc := range _ea {
		plan.AppendAlloc(ephAlloc, nil)
	}

	_pool.lock.Unlock()

	deploymentId := uuid.Generate()

	for i := 0; i < _unallocatedCount; {
		selectOptions := &scheduler.SelectOptions{}
		rnode := stack.Select(_tg, selectOptions)

		if rnode == nil {
			lepheralNode := _pool.poolnodespec.GetNode(uuid.Generate())
			lephemeralNodes = append(lephemeralNodes, lepheralNode)

			lnomadNodes = append(lnomadNodes, lepheralNode)
			stack.SetNodes(lnomadNodes)

			continue
		}

		i += 1
		resources := &structs.AllocatedResources{
			Tasks:          rnode.TaskResources,
			TaskLifecycles: rnode.TaskLifecycles,
			Shared: structs.AllocatedSharedResources{
				DiskMB: int64(_tg.EphemeralDisk.SizeMB),
			},
		}
		if rnode.AllocResources != nil {
			resources.Shared.Networks = rnode.AllocResources.Networks
			resources.Shared.Ports = rnode.AllocResources.Ports
		}

		// Create an allocation for this
		alloc := &structs.Allocation{
			ID:                 uuid.Generate(),
			Namespace:          ljob.Namespace,
			EvalID:             plan.EvalID,
			Name:               fmt.Sprintf("%s[%d]", _tg.Name, i),
			JobID:              ljob.ID,
			Job:                ljob,
			TaskGroup:          _tg.Name,
			Metrics:            evlCtx.Metrics(),
			NodeID:             rnode.Node.ID,
			NodeName:           rnode.Node.Name,
			DeploymentID:       deploymentId,
			TaskResources:      resources.OldTaskResources(),
			AllocatedResources: resources,
			DesiredStatus:      structs.AllocDesiredStatusRun,
			ClientStatus:       structs.AllocClientStatusPending,
			// SharedResources is considered deprecated, will be removed in 0.11.
			// It is only set for compat reasons.
			SharedResources: &structs.Resources{
				DiskMB:   _tg.EphemeralDisk.SizeMB,
				Networks: resources.Shared.Networks,
			},
		}

		plan.AppendAlloc(alloc, nil)
		lephemeralAllocations = append(lephemeralAllocations, alloc)
	}

	return lephemeralNodes, lephemeralAllocations
}
