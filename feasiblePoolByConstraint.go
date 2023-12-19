package main

import (
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/helper/uuid"
	"github.com/hashicorp/nomad/nomad/state"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/scheduler"
)

func feasiblePoolByConstraint(_pools []*PoolNodeSpec, _job *structs.Job) bool {
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

	lpoolNodes := make([]*structs.Node, 0, len(_pools))
	constraintChecker := scheduler.NewConstraintChecker(evlCtx, _job.Constraints)
	for _, lpool := range _pools {
		lnode := lpool.GetNode("")

		if !containsInSlice(_job.Datacenters, lnode.Datacenter) {
			continue
		}

		if constraintChecker.Feasible(lnode) {
			lpoolNodes = append(lpoolNodes, lnode)
		}
	}

	feasible := false
	if len(lpoolNodes) > 0 {
		driversChecker := scheduler.NewDriverChecker(evlCtx, nil)
		deviceChecker := scheduler.NewDeviceChecker(evlCtx)

		for _, ltg := range _job.TaskGroups {
			deviceChecker.SetTaskGroup(ltg)

			constraints := make([]*structs.Constraint, 0, len(ltg.Constraints))
			constraints = append(constraints, ltg.Constraints...)
			drivers := make(map[string]struct{})

			for _, task := range ltg.Tasks {
				drivers[task.Driver] = struct{}{}
				constraints = append(constraints, task.Constraints...)
			}

			constraintChecker.SetConstraints(constraints)
			driversChecker.SetDrivers(drivers)
			feasible = false

			for _, lpoolNode := range lpoolNodes {
				feasible = constraintChecker.Feasible(lpoolNode) && driversChecker.Feasible(lpoolNode) && deviceChecker.Feasible(lpoolNode)
				if feasible {
					break
				}
			}

			if !feasible {
				break
			}
		}
	}

	return feasible
}
