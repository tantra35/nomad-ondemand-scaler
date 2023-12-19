package main

import (
	"fmt"
	"time"

	"github.com/armon/go-metrics"
	"github.com/hashicorp/go-hclog"
	nomad "github.com/hashicorp/nomad/api"
)

type GCInfo struct {
	poolName             string
	seenEmptyCiclesCount int
	Collecting           bool
}

func gcAction(_state *StateStat, _gcconfig *GarbageCollectorConfig, _nc *nomad.Client, _pools map[string]*Pool) {
	lnodesToGC := make(map[string]*GCInfo)
	lticker := time.NewTicker(_gcconfig.CiclePeriod)
	logger := hclog.L().Named("gc")

	for range lticker.C {
		lnodesToGCCurrentCicle := make(map[string]*GCInfo)
		_state.DecFreeGcThreads()

		//TODO по идее каждый цикл сборки мусора нужно стопать и скейлинг, тее вводить stop the world паузу
		allowedfreeByPools := make(map[string]int)

		for _, lpool := range _pools {
			lallocsByNodes := make(map[string]int)
			lpoolTotalNodes := 0
			lpoolBusyNodes := 0

			lpool.lock.Lock()

			for _, lnode := range lpool.nomadNodes {
				lallocsByNodes[lnode.ID] = 0
				lpoolTotalNodes += 1
			}

			for _, lalloc := range lpool.nomadAllocs {
				lallocsByNodes[lalloc.NodeID] += 1
			}

			lpool.lock.Unlock()

			for lnodeId, lnodeAllocCount := range lallocsByNodes {
				if lnodeAllocCount == 0 {
					lgcInfo := lnodesToGC[lnodeId]
					if lgcInfo == nil {
						lgcInfo = &GCInfo{}
						lnodesToGC[lnodeId] = lgcInfo
					}

					lgcInfo.poolName = lpool.GetName()
					lgcInfo.seenEmptyCiclesCount += 1

					lnodesToGCCurrentCicle[lnodeId] = lgcInfo
				} else {
					lpoolBusyNodes += 1
				}
			}

			if _gcconfig.AllowedFreexpr != nil {
				_gcconfig.AllowedFreexpr.SetDoubleVariableValue("totalnodes", float64(lpoolTotalNodes))
				_gcconfig.AllowedFreexpr.SetDoubleVariableValue("busynodes", float64(lpoolBusyNodes))

				allowedFreenodes := int(_gcconfig.AllowedFreexpr.GetEvaluatedValue())
				allowedfreeByPools[lpool.GetName()] = allowedFreenodes

				nodesTolaunch := allowedFreenodes - (lpoolTotalNodes - lpoolBusyNodes)
				if nodesTolaunch > 0 {
					go lpool.WarmUp(nodesTolaunch)
					logger.Info(fmt.Sprintf("warming up pool %s allowered_free: %d, total_nodes: %d, busy_nodes: %d", lpool.GetName(), allowedFreenodes, lpoolTotalNodes, lpoolBusyNodes))
					metrics.IncrCounter([]string{"gc", "nodestoadd"}, float32(nodesTolaunch))
				}
			}
		}

		//ишем ноды которые в текущем цикле gc не появились(значит они или уже удалены или же пеерстали быть годными для gc)
		cleanupnodes := []string{}
		for lnodeId := range lnodesToGC {
			if _, lok := lnodesToGCCurrentCicle[lnodeId]; !lok {
				cleanupnodes = append(cleanupnodes, lnodeId)
			}
		}

		for _, lnodeId := range cleanupnodes {
			delete(lnodesToGC, lnodeId)
		}

		// Теперь полностью определили ноды которые следует удалить - удаляем
		gcnodedesdeleted := make(map[string][]string)

		for lnodeId, gcInfo := range lnodesToGC {
			if gcInfo.seenEmptyCiclesCount >= _gcconfig.CiclesToGc && !gcInfo.Collecting {
				if allowedfreeByPools[gcInfo.poolName] > 0 {
					allowedfreeByPools[gcInfo.poolName] -= 1
					continue
				}

				logger.Info(fmt.Sprintf("garbage colected node: %s in pool %s after %d gc cicles", lnodeId, gcInfo.poolName, gcInfo.seenEmptyCiclesCount))

				ldrainSpec := &nomad.DrainSpec{}
				_, lerr := _nc.Nodes().UpdateDrain(lnodeId, ldrainSpec, false, nil)
				if lerr != nil {
					logger.Error(fmt.Sprintf("can't drain node %s: %s", lnodeId, lerr))
				}

				gcnodedesdeleted[gcInfo.poolName] = append(gcnodedesdeleted[gcInfo.poolName], lnodeId)
				gcInfo.Collecting = true
			}
		}

		//TODO здесь stop the world паузу можно уже отпускать

		for lpoolName, lnodeIds := range gcnodedesdeleted {
			_pools[lpoolName].RemoveNode(lnodeIds)
			metrics.IncrCounter([]string{"gc", "nodestoremove"}, float32(len(lnodeIds)))
		}

		_state.IncFreeGcThreads()
	}
}
