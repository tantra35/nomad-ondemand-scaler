package nodeprovider

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	nomad "github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/tantra35/nomad-ondemand-scaler/nodeprovider/karpenterprovidergrpc"
)

type K8sKapenterProviderResources struct {
	Cpu    int
	MemMB  int
	DiskMB int
}

type K8sKapenterProviderPLuginSingleton struct {
	once      sync.Once
	rpcClient plugin.ClientProtocol
	err       error
}

var gK8sKapenterProviderPluginSingleton *K8sKapenterProviderPLuginSingleton = &K8sKapenterProviderPLuginSingleton{}

func getK8sKapenterProviderPLuginSingleton() (plugin.ClientProtocol, error) {
	gK8sKapenterProviderPluginSingleton.once.Do(func() {
		ex, lerr := os.Executable()
		if lerr != nil {
			gK8sKapenterProviderPluginSingleton.err = lerr
			return
		}

		exPath := filepath.Dir(ex)
		client := plugin.NewClient(&plugin.ClientConfig{
			HandshakeConfig: plugin.HandshakeConfig{
				ProtocolVersion:  1,
				MagicCookieKey:   "BASIC_PLUGIN",
				MagicCookieValue: "hello",
			},
			Plugins: map[string]plugin.Plugin{
				"grpc": &K8sKapenterProviderPlugin{},
			},
			Cmd: exec.Command(filepath.Join(exPath, "nomad-cluster-scalerv2-karpenter-plugin"), os.Getenv("NOMAD_JOB_ID")),
			AllowedProtocols: []plugin.Protocol{
				plugin.ProtocolNetRPC, plugin.ProtocolGRPC,
			},
			Logger: hclog.L().Named("karpenterprovider_plugin"),
		})

		rpcClient, lerr := client.Client()
		if lerr != nil {
			gK8sKapenterProviderPluginSingleton.err = lerr
		}

		gK8sKapenterProviderPluginSingleton.rpcClient = rpcClient
	})

	return gK8sKapenterProviderPluginSingleton.rpcClient, gK8sKapenterProviderPluginSingleton.err
}

type K8sKapenterProvider struct {
	lock   sync.Mutex
	logger hclog.Logger

	plugin       K8sKapenterProviderPluginInterface
	instanseSpec *karpenterprovidergrpc.AddInstancesSpec

	name           string
	state          map[string]bool
	incephemeral   map[string]int32
	lastUpdatetime time.Time
}

func updateStateFromKapenterPlugin(_poolName string, _k K8sKapenterProviderPluginInterface, _state map[string]bool) (int32, error) {
	var desiredCapacity int32

	linstances, lerr := _k.ListInstances(_poolName)
	if lerr != nil {
		return 0, lerr
	}

	for _, instanseId := range linstances {
		_state[instanseId] = false
	}

	return desiredCapacity, nil
}

func NewK8sKapenterProvider(_name string, _ami map[string]string, _securityGroups map[string]string, _subnets map[string]string, _instanceProfile string, _launchTemplate *string, _reqs []*karpenterprovidergrpc.Requirement, _res *K8sKapenterProviderResources) (INodeProvider, error) {
	rpcClient, lerr := getK8sKapenterProviderPLuginSingleton()
	if lerr != nil {
		return nil, lerr
	}

	raw, lerr := rpcClient.Dispense("grpc")
	if lerr != nil {
		return nil, lerr
	}

	newstate := map[string]bool{}
	svc := raw.(K8sKapenterProviderPluginInterface)

	_, lerr = updateStateFromKapenterPlugin(_name, svc, newstate)
	if lerr != nil {
		return nil, lerr
	}

	var lres map[string]string
	if _res != nil {
		lres = map[string]string{}
		if _res.Cpu != 0 {
			lres["cpu"] = fmt.Sprintf("%dm", _res.Cpu)
		}
		if _res.MemMB != 0 {
			lres["memory"] = fmt.Sprintf("%dMi", _res.MemMB)
		}
		if _res.DiskMB != 0 {
			lres["storage"] = fmt.Sprintf("%dMi", _res.DiskMB)
		}
	}

	return &K8sKapenterProvider{
		logger: hclog.L().Named("K8sKapenterProvider").With("name", _name),
		name:   _name,
		plugin: raw.(K8sKapenterProviderPluginInterface),
		instanseSpec: &karpenterprovidergrpc.AddInstancesSpec{
			Ami:             _ami,
			SecurityGroups:  _securityGroups,
			Subnets:         _subnets,
			InstanceProfile: _instanceProfile,
			LaunchTemplate:  _launchTemplate,
			Requirements:    _reqs,
			Resources:       lres,
		},
		state:          newstate,
		incephemeral:   map[string]int32{},
		lastUpdatetime: time.Now(),
	}, nil
}

func (p *K8sKapenterProvider) IsNodeExists(_nomadNode *nomad.Node) bool {
	lloger := p.logger.Named("IsNodeExists")
	llastregisterevnt := GetLastRegisterEvent(_nomadNode.Events)
	lastTime := llastregisterevnt.Timestamp
	instanceId := _nomadNode.Attributes["unique.platform.aws.instance-id"]
	lNodeExists := false

	p.lock.Lock()
	if lastTime.After(p.lastUpdatetime) {
		p.lock.Unlock()
		updatetime := time.Now()
		newstate := map[string]bool{}

		for {
			_, lerr := updateStateFromKapenterPlugin(p.name, p.plugin, newstate)
			if lerr == nil {
				lloger.Debug(fmt.Sprintf("successed updated state when check insatnceId: %s(nomadnodeid: %s)", instanceId, _nomadNode.ID))
				break
			}

			lloger.Error(fmt.Sprintf("can't update state due: %s", lerr))
			time.Sleep(10 * time.Second)
		}

		p.lock.Lock()
		for linstanceid := range newstate {
			if seenbypool, lok := p.state[linstanceid]; lok {
				newstate[linstanceid] = seenbypool
			}
		}
		p.state = newstate
		p.lastUpdatetime = updatetime
	}

	_, lNodeExists = p.state[instanceId]
	if lNodeExists {
		p.state[instanceId] = true
	}
	p.lock.Unlock()

	return lNodeExists
}

func (p *K8sKapenterProvider) _removeNode(lloger hclog.Logger, instncesIds []string) error {
	for i := 0; i < len(instncesIds); i += 20 {
		instncesIdsBatch := instncesIds[i:Min(i+20, len(instncesIds))]

		//TODO надо переделать и удалять в плагине все таки по одном инстансу,
		// потому как удалить срузу пачку иснтансов из стейта до удаления, выглядит как то ненадежно, по одному то тоже не очень,
		//но для этого провайдера это не страшно, так как инстанс поднимется в любом случае
		p.lock.Lock()
		for _, linstanceId := range instncesIds {
			delete(p.state, linstanceId)
		}
		p.lock.Unlock()

		for {
			lerr := p.plugin.RemoveInstances(p.name, instncesIdsBatch)
			if lerr == nil {
				break
			}

			lloger.Error(fmt.Sprintf("failed to remove instanses by karpenter due: %s", lerr))
			time.Sleep(10 * time.Second)
		}
	}

	return nil
}

func (p *K8sKapenterProvider) RemoveNode(_nomadNodes []*nomad.Node) error {
	lloger := p.logger.Named("RemoveNode")
	var instncesIds []string

	for _, lnode := range _nomadNodes {
		instanceId := lnode.Attributes["unique.platform.aws.instance-id"]
		instncesIds = append(instncesIds, instanceId)
	}

	return p._removeNode(lloger, instncesIds)
}

func (p *K8sKapenterProvider) UpdateNode(_ctx context.Context, _nodes []*structs.Node, _totalcount int32) error {
	lloger := p.logger.Named("UpdateNode")

	p.lock.Lock()

	lnodesinpool := map[string]interface{}{}
	for _, lnode := range _nodes {
		instanceId := lnode.Attributes["unique.platform.aws.instance-id"]
		lnodesinpool[instanceId] = struct{}{}
	}

	instancetoremove := []string{}
	for instanceId, seenbypool := range p.state {
		if seenbypool {
			if _, lok := lnodesinpool[instanceId]; !lok {
				instancetoremove = append(instancetoremove, instanceId)
			}
		}
	}

	lmynodescount := len(p.state)
	p.lock.Unlock()

	if len(instancetoremove) > 0 {
		lloger.Warn(fmt.Sprintf("pool reported about differense in nodes %d(my) -> %d(pool oppinion), so, remove unexisten", lmynodescount, len(_nodes)))
		p._removeNode(lloger, instancetoremove)
		lmynodescount -= len(instancetoremove)
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	var laditinc int32 = 0
	for _, linc := range p.incephemeral {
		laditinc += linc
	}

	inccount := _totalcount - int32(lmynodescount) - laditinc

	if inccount > 0 {
		linckey := generateRandomString(19)
		p.incephemeral[linckey] = inccount

		go func(_ctx context.Context, _inccount int32, _p *K8sKapenterProvider) {
			for {
				instances, lreason, lerr := _p.plugin.AddInstances(_ctx, _p.name, int(_inccount), _p.instanseSpec)
				if len(instances) == int(_inccount) {
					break
				}

				if _ctx.Err() != nil {
					break
				}

				if lerr != nil {
					lloger.Error(fmt.Sprintf("can't set karpenter disiresize size to: %d(inc: %d), add only: %d due: %s", _totalcount, _inccount, 0, lerr))
				} else {
					lloger.Error(fmt.Sprintf("can't set karpenter disiresize size to: %d(inc: %d), add only: %d due: %s", _totalcount, _inccount, len(instances), lreason))
				}

				time.Sleep(10 * time.Second)
				_inccount -= int32(len(instances))
			}

			_p.lock.Lock()

			delete(_p.incephemeral, linckey)

			_p.lock.Unlock()
		}(_ctx, inccount, p)
	} else if inccount < 0 {
		return fmt.Errorf("incrementing karpenter disiresize size to: %d, that too bad", inccount)
	}

	return nil
}
