package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/helper/uuid"
	"github.com/hashicorp/nomad/nomad/state"
	"github.com/hashicorp/nomad/nomad/structs"
	psstructs "github.com/hashicorp/nomad/plugins/shared/structs"
	"github.com/hashicorp/nomad/scheduler"
	"github.com/mitchellh/hashstructure"
)

// -----------------------------------------------------------------------------
type PoolNodeSpecResources struct {
	Cpu     int
	MemMB   int
	DiskMB  int
	Devices map[string]string
}

type PoolNodeSpec struct {
	FullName   string
	Attributes map[string]Variant
}

func NewPoolNodeSpec(_a map[string]Variant) *PoolNodeSpec {
	lpoolSpec := &PoolNodeSpec{Attributes: _a}

	lpoolComputeClass, _ := lpoolSpec.ComputeClass()
	lpoolName := lpoolSpec.GetName() + "-" + lpoolComputeClass
	lpoolSpec.FullName = lpoolName

	return lpoolSpec
}

func (n *PoolNodeSpec) GetResources() *PoolNodeSpecResources {
	lresources := &PoolNodeSpecResources{}

	for _, resName := range []string{"cpu", "mem", "disk"} {
		if resVal, lok := n.Attributes[resName]; lok {
			if resVal.GetType() == VariantTypeInt {
				switch resName {
				case "cpu":
					lresources.Cpu = *resVal.GetIntValue()
				case "mem":
					lresources.MemMB = *resVal.GetIntValue()
				case "disk":
					lresources.DiskMB = *resVal.GetIntValue()
				}
			}
		}
	}

	return lresources
}

func (n *PoolNodeSpec) GetFullName() string {
	return n.FullName
}

func (n PoolNodeSpec) HashIncludeMap(field string, k, v interface{}) (bool, error) {
	switch field {
	case "Attributes":
		as := k.(string)
		switch as {
		case "cpu", "mem", "disk":
			return false, nil
		}
		return true, nil
	default:
		return false, nil
	}
}

func (n *PoolNodeSpec) ComputeClass() (string, error) {
	hash, err := hashstructure.Hash(n, nil)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("v1:%d", hash), nil
}

func (n *PoolNodeSpec) GetName() string {
	lName := ""

	for _, resName := range []string{"cpu", "mem", "disk"} {
		if resVal, lok := n.Attributes[resName]; lok {
			if resVal.GetType() == VariantTypeInt {
				if len(lName) > 0 {
					lName += ";"
				}

				lName += fmt.Sprintf("%s:%d", resName, *resVal.GetIntValue())
			}
		}
	}

	return lName
}

func (n *PoolNodeSpec) GetNode(uid string) *structs.Node {
	node := &structs.Node{
		ID:                    uid,
		SecretID:              uid,
		Name:                  uid,
		Drivers:               map[string]*structs.DriverInfo{},
		Attributes:            map[string]string{},
		Links:                 map[string]string{},
		Meta:                  map[string]string{},
		Status:                structs.NodeStatusReady,
		SchedulingEligibility: structs.NodeSchedulingEligible,
		Resources: &structs.Resources{
			DiskMB: 100 * 1024,
		},
		NodeResources: &structs.NodeResources{
			Disk: structs.NodeDiskResources{
				DiskMB: 100 * 1024,
			},
			Networks: []*structs.NetworkResource{
				{
					Mode:          "host",
					Device:        "eth0",
					CIDR:          "192.168.0.100/32",
					MBits:         1000,
					ReservedPorts: []structs.Port{{Label: "ssh", Value: 22}},
				},
			},
			NodeNetworks: []*structs.NodeNetworkResource{
				{
					Mode:   "host",
					Device: "eth0",
					Speed:  1000,
					Addresses: []structs.NodeNetworkAddress{
						{
							Alias:   "default",
							Address: "192.168.0.100",
							Family:  structs.NodeNetworkAF_IPv4,
						},
					},
				},
			},
		},
	}

	if reservedVariant, lok := n.Attributes["reserved"]; lok {
		if reservedVariant.GetType() == VariantTypeMap {
			reserved := &structs.NodeReservedResources{}
			reservedDeprecated := &structs.Resources{}

			for reservedName, reservedValue := range reservedVariant.GetMapValue() {
				switch reservedName {
				case "mem":
					if reservedValue.GetType() == VariantTypeInt {
						reserved.Memory.MemoryMB = int64(*reservedValue.GetIntValue())
						reservedDeprecated.MemoryMB = *reservedValue.GetIntValue()
					}

				case "cpu":
					if reservedValue.GetType() == VariantTypeInt {
						reserved.Cpu.CpuShares = int64(*reservedValue.GetIntValue())
						reservedDeprecated.CPU = *reservedValue.GetIntValue()
					}

				case "disk":
					if reservedValue.GetType() == VariantTypeInt {
						reserved.Disk.DiskMB = int64(*reservedValue.GetIntValue())
						reservedDeprecated.DiskMB = *reservedValue.GetIntValue()
					}
				}
			}

			node.Reserved = reservedDeprecated
			node.ReservedResources = reserved
		}
	}

	for ak, av := range n.Attributes {
		if strings.Index(ak, "attr.") == 0 {
			switch av.GetType() {
			case VariantTypeString:
				node.Attributes[ak[5:]] = *av.GetStringValue()
			case VariantTypeInt:
				node.Attributes[ak[5:]] = fmt.Sprintf("%d", *av.GetIntValue())
			case VariantTypeBool:
				node.Attributes[ak[5:]] = fmt.Sprintf("%v", *av.GetBoolValue())
			}

		} else if strings.Index(ak, "meta.") == 0 {
			if av.GetType() == VariantTypeString {
				node.Meta[ak[5:]] = *av.GetStringValue()
			}
		} else if strings.Index(ak, "links.") == 0 {
			if av.GetType() == VariantTypeString {
				node.Links[ak[6:]] = *av.GetStringValue()
			}
		}
	}

	if ldrivers, lok := n.Attributes["drivers"]; lok {
		if ldrivers.GetType() == VariantTypeSlice {
			for _, ldriverv := range ldrivers.GetSliceValue() {
				if ldriverv.GetType() != VariantTypeString {
					continue
				}

				ldriverName := ldriverv.GetStringValue()
				node.Drivers[*ldriverName] = &structs.DriverInfo{
					Detected: true,
					Healthy:  true,
				}
			}
		}
	}

	if ldevices, lok := n.Attributes["devices"]; lok {
		if ldevices.GetType() == VariantTypeSlice {
			for _, ldeviceVariant := range ldevices.GetSliceValue() {
				if ldeviceVariant.GetType() != VariantTypeMap {
					continue
				}

				ldevice := ldeviceVariant.GetMapValue()
				lnomadDevice := &structs.NodeDeviceResource{}

				for ldevAttr, ldevAttrValuev := range ldevice {
					switch ldevAttr {
					case "name":
						if ldevAttrValuev.GetType() == VariantTypeString {
							lnomadDevice.Name = *ldevAttrValuev.GetStringValue()
						}
					case "type":
						if ldevAttrValuev.GetType() == VariantTypeString {
							lnomadDevice.Type = *ldevAttrValuev.GetStringValue()
						}
					case "vendor":
						if ldevAttrValuev.GetType() == VariantTypeString {
							lnomadDevice.Vendor = *ldevAttrValuev.GetStringValue()
						}
					case "count":
						if ldevAttrValuev.GetType() == VariantTypeInt {
							for li := 0; li < *ldevAttrValuev.GetIntValue(); li++ {
								linstance := &structs.NodeDevice{
									ID:      uuid.Generate(),
									Healthy: true,
									Locality: &structs.NodeDeviceLocality{
										PciBusID: fmt.Sprintf("0000:00:%02X.0", li+10), // domain:bus:device.function
									},
								}
								lnomadDevice.Instances = append(lnomadDevice.Instances, linstance)
							}
						}

					case "attr":
						if ldevAttrValuev.GetType() == VariantTypeMap {
							ldeviceAttrsVariant := ldevAttrValuev.GetMapValue()
							ldeviceAttrs := map[string]*psstructs.Attribute{}
							for lattrName, lattrNameValuev := range ldeviceAttrsVariant {
								if lattrNameValuev.GetType() != VariantTypeString {
									continue
								}

								ldeviceAttrs[lattrName] = psstructs.ParseAttribute(*lattrNameValuev.GetStringValue())
							}

							lnomadDevice.Attributes = ldeviceAttrs
						}
					}
				}

				node.NodeResources.Devices = append(node.NodeResources.Devices, lnomadDevice)
			}
		}
	}

	if lcpu, lok := n.Attributes["cpu"]; lok {
		if lcpu.GetType() == VariantTypeInt {
			node.Resources.CPU = *lcpu.GetIntValue()
			node.NodeResources.Cpu.CpuShares = int64(*lcpu.GetIntValue())
		}
	}

	if lmem, lok := n.Attributes["mem"]; lok {
		if lmem.GetType() == VariantTypeInt {
			node.Resources.MemoryMB = *lmem.GetIntValue()
			node.NodeResources.Memory.MemoryMB = int64(*lmem.GetIntValue())
		}
	}

	if ldisk, lok := n.Attributes["disk"]; lok {
		if ldisk.GetType() == VariantTypeInt {
			node.NodeResources.Disk.DiskMB = int64(*ldisk.GetIntValue())
		}
	}

	if ldatacenter, lok := n.Attributes["datacenter"]; lok {
		if ldatacenter.GetType() == VariantTypeString {
			node.Datacenter = *ldatacenter.GetStringValue()
		}
	}

	if lnodeclass, lok := n.Attributes["nodeclass"]; lok {
		if lnodeclass.GetType() == VariantTypeString {
			node.NodeClass = *lnodeclass.GetStringValue()
		}
	}

	node.Canonicalize()
	node.ComputeClass()

	return node
}

func (n *PoolNodeSpec) GetSpectFromNode(node *structs.Node) *PoolNodeSpec {
	poolSpecAttibutes := map[string]Variant{}

	for ak, av := range n.Attributes {
		if strings.Index(ak, "attr.") == 0 {
			ltype := av.GetType()
			if ltype == VariantTypeString || ltype == VariantTypeInt || ltype == VariantTypeBool {
				if na, lok := node.Attributes[ak[5:]]; lok {
					poolSpecAttibutes[ak] = NewVariantStringValue(na)
				}
			}
		} else if strings.Index(ak, "meta.") == 0 {
			ltype := av.GetType()
			if ltype == VariantTypeString || ltype == VariantTypeInt || ltype == VariantTypeBool {
				if nm, lok := node.Meta[ak[5:]]; lok {
					poolSpecAttibutes[ak] = NewVariantStringValue(nm)
				}
			}
		} else if strings.Index(ak, "links.") == 0 {
			if av.GetType() == VariantTypeString {
				if nl, lok := node.Meta[ak[6:]]; lok {
					poolSpecAttibutes[ak] = NewVariantStringValue(nl)
				}
			}
		}
	}

	if reservedVariant, lok := n.Attributes["reserved"]; lok {
		reserved := node.ReservedResources

		if reservedVariant.GetType() == VariantTypeMap && reserved != nil {
			lreserved := map[string]Variant{}

			for reservedName, reservedValue := range reservedVariant.GetMapValue() {
				switch reservedName {
				case "mem":
					if reservedValue.GetType() == VariantTypeInt {
						lreserved["mem"] = NewVariantIntValue(int(reserved.Memory.MemoryMB))
					}

				case "cpu":
					if reservedValue.GetType() == VariantTypeInt {
						lreserved["cpu"] = NewVariantIntValue(int(reserved.Cpu.CpuShares))
					}

				case "disk":
					if reservedValue.GetType() == VariantTypeInt {
						lreserved["disk"] = NewVariantIntValue(int(reserved.Disk.DiskMB))
					}
				}
			}

			poolSpecAttibutes["reserved"] = NewVariantMapValue(lreserved)
		}
	}

	if ldrivers, lok := n.Attributes["drivers"]; lok {
		if ldrivers.GetType() == VariantTypeSlice {
			lDriversList := []Variant{}

			for ldriverName, ldrv := range node.Drivers {
				if ldrv.Detected {
					lDriversList = append(lDriversList, NewVariantStringValue(ldriverName))
				}
			}

			poolSpecAttibutes["drivers"] = NewVariantSliceValue(lDriversList)
		}
	}

	if ldevicesVariant, lok := n.Attributes["devices"]; lok {
		lndevices := node.NodeResources.Devices
		if len(lndevices) > 0 && ldevicesVariant.GetType() == VariantTypeSlice {
			ldevices := []Variant{}
			ladevices := ldevicesVariant.GetSliceValue()

			for ldi, lndevice := range lndevices {
				if len(ladevices) <= ldi {
					break
				}

				ladeviceVariant := ladevices[ldi]
				if ladeviceVariant.GetType() == VariantTypeMap {
					ladeviceVariantMap := ladeviceVariant.GetMapValue()
					ldevice := map[string]Variant{}

					if _, lok := ladeviceVariantMap["name"]; lok {
						ldevice["name"] = NewVariantStringValue(lndevice.Name)
					}

					if _, lok := ladeviceVariantMap["type"]; lok {
						ldevice["type"] = NewVariantStringValue(lndevice.Type)
					}

					if _, lok := ladeviceVariantMap["vendor"]; lok {
						ldevice["vendor"] = NewVariantStringValue(lndevice.Vendor)
					}

					if ladeviceAttrs, lok := ladeviceVariantMap["attr"]; lok && len(lndevice.Attributes) > 0 {
						if ladeviceAttrs.GetType() == VariantTypeMap {
							ldeviceAttrsVariant := ladeviceAttrs.GetMapValue()
							ldeviceAttrs := map[string]Variant{}

							for lattrName := range ldeviceAttrsVariant {
								if lattrValue, lok := lndevice.Attributes[lattrName]; lok {
									if lvalueString, lok := lattrValue.GetString(); lok {
										ldeviceAttrs[lattrName] = NewVariantStringValue(lvalueString)
									}
								}
							}

							ldevice["attr"] = NewVariantMapValue(ldeviceAttrs)
						}
					}

					ldevice["count"] = NewVariantIntValue(len(lndevice.Instances))

					ldevices = append(ldevices, NewVariantMapValue(ldevice))
				}
			}

			poolSpecAttibutes["devices"] = NewVariantSliceValue(ldevices)
		}
	}

	if lcpu, lok := n.Attributes["cpu"]; lok {
		if lcpu.GetType() == VariantTypeInt {
			poolSpecAttibutes["cpu"] = NewVariantIntValue(int(node.NodeResources.Cpu.CpuShares))
		}
	}

	if lmem, lok := n.Attributes["mem"]; lok {
		if lmem.GetType() == VariantTypeInt {
			poolSpecAttibutes["mem"] = NewVariantIntValue(int(node.NodeResources.Memory.MemoryMB))
		}
	}

	if ldisk, lok := n.Attributes["disk"]; lok {
		if ldisk.GetType() == VariantTypeInt {
			poolSpecAttibutes["disk"] = NewVariantIntValue(int(node.NodeResources.Disk.DiskMB))
		}
	}

	if ldatacenter, lok := n.Attributes["datacenter"]; lok {
		if ldatacenter.GetType() == VariantTypeString {
			poolSpecAttibutes["datacenter"] = NewVariantStringValue(node.Datacenter)
		}
	}

	if lnodeclass, lok := n.Attributes["nodeclass"]; lok {
		if lnodeclass.GetType() == VariantTypeString {
			poolSpecAttibutes["nodeclass"] = NewVariantStringValue(node.NodeClass)
		}
	}

	return NewPoolNodeSpec(poolSpecAttibutes)
}

func getSuitableNodes(_tgName string, _job *structs.Job, _nodes []*structs.Node) map[string]string {
	tgToNode := make(map[string]string)
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
	stack.SetJob(_job)

	nodeWithoutDevices := make([]*structs.Node, 0, len(_nodes))
	for _, lnode := range _nodes {
		if lnode.NodeResources.Devices == nil || len(lnode.NodeResources.Devices) == 0 {
			nodeWithoutDevices = append(nodeWithoutDevices, lnode)
		}
	}

	selectOptions := &scheduler.SelectOptions{}
	for _, ltg := range _job.TaskGroups {
		if _tgName == "" || ltg.Name == _tgName {
			if len(nodeWithoutDevices) > 0 { // сначала пробуем ноды без девайсов, вдруг подойдут
				stack.SetNodes(nodeWithoutDevices)
			} else {
				stack.SetNodes(_nodes)
			}

			rnode := stack.Select(ltg, selectOptions)

			if rnode == nil && len(nodeWithoutDevices) > 0 {
				stack.SetNodes(_nodes)
				rnode = stack.Select(ltg, selectOptions)
			}

			if rnode != nil {
				tgToNode[ltg.Name] = rnode.Node.ID
			} else {
				tgToNode[ltg.Name] = ""
			}
		}
	}

	return tgToNode
}

func GetOptimalPoolSpec(_job *structs.Job, _poolList []*PoolNodeSpec) map[string]*PoolNodeSpec {
	lprevJobStatus := _job.Status
	_job.Status = structs.JobStatusPending
	defer func() {
		_job.Status = lprevJobStatus
	}()

	testNodes := make([]*structs.Node, 0, len(_poolList))
	testNodesByGroups := map[string][]*structs.Node{}
	poolsByNames := map[string]*PoolNodeSpec{}

	for _, lpool := range _poolList {
		lcs, _ := lpool.ComputeClass()
		lpoolName := lpool.GetName() + "|" + lcs
		lnode := lpool.GetNode(lpoolName)

		if !containsInSlice(_job.Datacenters, lnode.Datacenter) {
			continue
		}

		testNodes = append(testNodes, lnode)
		testNodesByGroups[lcs] = append(testNodesByGroups[lcs], lnode)
		poolsByNames[lpoolName] = lpool
	}

	for _, tnbg := range testNodesByGroups {
		sort.Slice(tnbg, func(i, j int) bool {
			return tnbg[i].NodeResources.Cpu.CpuShares <= tnbg[j].NodeResources.Cpu.CpuShares &&
				tnbg[i].NodeResources.Memory.MemoryMB <= tnbg[j].NodeResources.Memory.MemoryMB
		})
	}

	lreturnPoolsNames := map[string]string{}
	for tgName, lpoolName := range getSuitableNodes("", _job, testNodes) {
		if lpoolName == "" {
			continue
		}

		lcs := strings.SplitN(lpoolName, "|", 2)[1]
		if len(testNodesByGroups[lcs]) == 1 {
			lreturnPoolsNames[tgName] = lpoolName
		} else {
		LOOP:
			for _, lpool := range testNodesByGroups[lcs] {
				for tgName, lpoolName := range getSuitableNodes(tgName, _job, []*structs.Node{lpool}) {
					lreturnPoolsNames[tgName] = lpoolName
					break LOOP
				}
			}
		}
	}

	retval := map[string]*PoolNodeSpec{}
	for tgName, lpoolName := range lreturnPoolsNames {
		retval[tgName] = nil
		if lpoolName != "" {
			retval[tgName] = poolsByNames[lpoolName]
		}
	}

	return retval
}
