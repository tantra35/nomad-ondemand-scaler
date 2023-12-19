package nodeprovider

import (
	"fmt"
	"reflect"

	"github.com/spf13/cast"
	"github.com/tantra35/nomad-ondemand-scaler/nodeprovider/karpenterprovidergrpc"
)

func Createkarpenterprovider(_params interface{}, _res *K8sKapenterProviderResources) (INodeProvider, error) {
	params, lok := _params.(map[string]interface{})
	if !lok {
		return nil, fmt.Errorf("params is not map[string]interface{}")
	}

	argValues := make([]reflect.Value, 0)

	if lnameIntf, lok := params["name"]; !lok {
		return nil, fmt.Errorf("params have no name attribute")
	} else {
		lname, lerr := cast.ToStringE(lnameIntf)
		if lerr != nil {
			return nil, fmt.Errorf("param name of wrong type")
		}

		argValues = append(argValues, reflect.ValueOf(lname))
	}

	if lfreqPerCpuCoreIntf, lok := params["freqPerCpuCore"]; _res != nil && !lok {
		return nil, fmt.Errorf("params have no freqPerCpuCore attribute")
	} else {
		lfreqPerCpuCore, lerr := cast.ToIntE(lfreqPerCpuCoreIntf)
		if lerr != nil {
			return nil, fmt.Errorf("param freqPerCpuCore of wrong type")
		}

		if _res != nil {
			_res.Cpu /= lfreqPerCpuCore
		}
	}

	if lamiIntf, lok := params["ami"]; lok {
		lami, lerr := cast.ToStringMapStringE(lamiIntf)
		if lerr != nil {
			return nil, fmt.Errorf("params have no ami of wrong type")
		}

		argValues = append(argValues, reflect.ValueOf(lami))
	} else {
		argValues = append(argValues, reflect.MakeMap(reflect.TypeOf(map[string]string{})))
	}

	if lsecuritygroupsIntf, lok := params["securitygroups"]; lok {
		lsecuritygroups, lerr := cast.ToStringMapStringE(lsecuritygroupsIntf)
		if lerr == nil {
			argValues = append(argValues, reflect.ValueOf(lsecuritygroups))
		} else {
			return nil, fmt.Errorf("params have no securitygroups of wrong type")
		}
	} else {
		argValues = append(argValues, reflect.MakeMap(reflect.TypeOf(map[string]string{})))
	}

	if lsubnetsIntf, lok := params["subnets"]; lok {
		lsubnets, lerr := cast.ToStringMapStringE(lsubnetsIntf)
		if lerr == nil {
			argValues = append(argValues, reflect.ValueOf(lsubnets))
		} else {
			return nil, fmt.Errorf("params have no subnets of wrong type")
		}
	} else {
		argValues = append(argValues, reflect.MakeMap(reflect.TypeOf(map[string]string{})))
	}

	if linstanceProfileIntf, lok := params["profile"]; lok {
		linstanceProfile, lok := linstanceProfileIntf.(string)
		if !lok {
			return nil, fmt.Errorf("params have no profile of wrong type")
		}

		argValues = append(argValues, reflect.ValueOf(linstanceProfile))
	} else {
		argValues = append(argValues, reflect.ValueOf(""))
	}

	if llaunchtemplateintf, lok := params["launchtemplate"]; lok {
		llaunchtemplate, lok := llaunchtemplateintf.(string)
		if !lok {
			return nil, fmt.Errorf("params have no launchtemplate of wrong type")
		}

		llaunchtemplateptr := &llaunchtemplate
		argValues = append(argValues, reflect.ValueOf(llaunchtemplateptr))
	} else {
		var ptr *string
		argValues = append(argValues, reflect.Zero(reflect.TypeOf(ptr)))
	}

	if lreqsIntf, lok := params["reqs"]; lok {
		lreqsSlice, lerr := cast.ToSliceE(lreqsIntf)
		if lerr != nil {
			return nil, lerr
		}

		lparamsReq := make([]*karpenterprovidergrpc.Requirement, 0, len(lreqsSlice))

		for _, lreqIntf := range lreqsSlice {
			lreq, lerr := cast.ToStringMapE(lreqIntf)
			if lerr != nil {
				return nil, lerr
			}

			var lkey string
			if lkeyIntf, lok := lreq["Key"]; lok {
				lkey = cast.ToString(lkeyIntf)
			} else {
				continue
			}

			var lvalues []string
			if lvaluesIntf, lok := lreq["Values"]; lok {
				lvalues = cast.ToStringSlice(lvaluesIntf)
			} else {
				continue
			}

			var lop string
			if lopIntf, lok := lreq["Op"]; lok {
				lop = cast.ToString(lopIntf)
			} else {
				lop = "In"
			}

			lparamsReq = append(lparamsReq, &karpenterprovidergrpc.Requirement{
				Key:      lkey,
				Operator: lop,
				Values:   lvalues,
			})
		}

		argValues = append(argValues, reflect.ValueOf(lparamsReq))
	} else {
		return nil, fmt.Errorf("params have no reqs")
	}

	argValues = append(argValues, reflect.ValueOf(_res))
	result := reflect.ValueOf(NewK8sKapenterProvider).Call(argValues)
	if !result[1].IsNil() {
		return nil, fmt.Errorf("aws karenter provider returned error: %v", result[1].Interface().(error))
	}

	lprovider := result[0].Interface().(INodeProvider)
	return lprovider, nil
}
