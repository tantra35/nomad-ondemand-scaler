package main

import (
	"context"
	"fmt"

	"github.com/armon/go-metrics"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/nomad/structs"
)

type PoolToScale struct {
	pool *Pool
	en   []*structs.Node
	ea   []*structs.Allocation
}

func scalingAction(_stat *StateStat, _pools map[string]*Pool, scalingRequireCh *Queue[*ScalingEvent], scalingDoneCh chan<- string) {
	logger := hclog.L().Named("scaling")

	for {
		scalingEvent := scalingRequireCh.Dequeue()
		_stat.DecFreeScalingThreads()

		lJobToScale := scalingEvent.Job
		lpoolsToScale := map[string]*PoolToScale{}

		for tgName, unAllocInfo := range scalingEvent.UnAllocatedTg {
			var tg *structs.TaskGroup

			for _, tg = range lJobToScale.TaskGroups {
				if tg.Name == tgName {
					break
				}
			}

			lpoolName := unAllocInfo.PoolInfo.GetFullName()
			var lpoolToScale *PoolToScale
			var lok bool
			if lpoolToScale, lok = lpoolsToScale[lpoolName]; !lok {
				lpoolToScale = &PoolToScale{
					pool: _pools[lpoolName],
				}
				lpoolsToScale[lpoolName] = lpoolToScale
			}
			logger.Debug(fmt.Sprintf("for task group %s/%s.%s using pool %s", lJobToScale.Namespace, lJobToScale.ID, tgName, lpoolName))

			en, ea := estimateRequiredNodes(lpoolToScale.pool, lpoolToScale.en, lpoolToScale.ea, lJobToScale, tg, unAllocInfo.UnAllocCount)
			logger.Info(fmt.Sprintf("estimate required nodes for %s/%s.%s: %d", lJobToScale.Namespace, lJobToScale.Name, tgName, len(en)), "pool", lpoolName)

			lpoolToScale.en = append(lpoolToScale.en, en...)
			lpoolToScale.ea = append(lpoolToScale.ea, ea...)
		}

		lestimatedNodes := 0
		for lpoolName, lpoolToScale := range lpoolsToScale {
			lestimatedNodes += len(lpoolToScale.en)
			lerr := lpoolToScale.pool.Update(scalingEvent.Ctx, lpoolToScale.en, lpoolToScale.ea)
			if lerr != nil {
				if lerr == context.DeadlineExceeded {
					logger.Error("Waiting for pool update canceled by timeout", "pool", lpoolName)
					_stat.IncScalingTimeouts()
				} else if lerr == context.Canceled {
					logger.Info("Waiting for pool update canceled, due reported that resources fully satisfied or not needed", "pool", lpoolName)
				} else {
					logger.Info(fmt.Sprintf("can't update pool, due: %s", lerr), "pool", lpoolName)
				}
			}
		}

		metrics.IncrCounter([]string{"scaleup", "estimatedNodes"}, float32(lestimatedNodes))
		metrics.MeasureSince([]string{"scaleup", "executiontime"}, scalingEvent.FireTime)

		scalingDoneCh <- scalingEvent.Id
		_stat.IncFreeScalingThreads()
	}
}
