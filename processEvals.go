package main

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	nomad "github.com/hashicorp/nomad/api"
)

func processEvals(_stateStat *StateStat, _stalecnf *StaleApiConfig, _preventhung *HungPreventionConfig, evalsCh <-chan *nomad.Evaluation, scalingRequireCh *Queue[*ScalingEvent], scalingDoneCh <-chan string, _poolSpecs []*PoolNodeSpec) {
	logger := hclog.L().Named("evals")

	cnf := nomad.DefaultConfig()
	nclient, _ := nomad.NewClient(cnf)

	evals := map[string]*nomad.Evaluation{}
	blockedEvalsChains := map[string][]string{} // цепочка блокированных евалов с головой в которой всегда блокированный  eval
	firedEvents := map[string]*ScalingEvent{}

	waitCompleteEvents := time.NewTimer(10 * time.Second)

	for {
		select {
		case lEval := <-evalsCh:
			evals[lEval.ID] = lEval
			resetTimer := false

			lchainId := fmt.Sprintf("%s/%s", lEval.Namespace, lEval.JobID)
			if lEvalsChain, lok := blockedEvalsChains[lchainId]; lok {
				lIsnewEval := true
				for _, leId := range lEvalsChain {
					if leId == lEval.ID { //евал уже есть в нашей цепочке
						lIsnewEval = false
						break
					}
				}

				if lIsnewEval {
					if lEval.CreateIndex > evals[lEvalsChain[len(lEvalsChain)-1]].CreateIndex {
						lEvalsChain = append(lEvalsChain, lEval.ID)
						blockedEvalsChains[lchainId] = lEvalsChain
						logger.Debug(fmt.Sprintf("add new eval %s to chain %s, result chain: %v", lEval.ID, lchainId, lEvalsChain))
					}
				}

				resetTimer = true // нужно подвести итог так как изменились евалы в блокированных цепочках
			}

			if lEval.Status == "blocked" {
				if _, lok := blockedEvalsChains[lchainId]; !lok { //создаем новую цепочку блокированных евалов, если такой еще нет
					lEvalsChain := []string{lEval.ID}
					blockedEvalsChains[lchainId] = lEvalsChain
					logger.Debug(fmt.Sprintf("crate new chain %s, result chain: %v", lchainId, lEvalsChain))
					resetTimer = true // нужно подвести итог так как добавилась блокированная цепочка
				}
			}

			if resetTimer {
				waitCompleteEvents.Reset(10 * time.Second) // Тут очень подлый момент если оочень много поступает блокированных евалов, может сулчится так что таймер будет постоянной сбрасываться и мы никогда не можем подвести итог и бросить события
			}

		case lchainId := <-scalingDoneCh:
			delete(firedEvents, lchainId)

		case <-waitCompleteEvents.C:
			newevals := map[string]*nomad.Evaluation{}
			removedChains := []string{}

			for lchainId := range blockedEvalsChains {
				var li int = 0
				for ; li < len(blockedEvalsChains[lchainId]); li++ {
					lEvalId := blockedEvalsChains[lchainId][li]
					lEval := evals[lEvalId]
					if lEval.Status == "blocked" {
						break
					}
				}

				if li != 0 {
					blockedEvalsChains[lchainId] = blockedEvalsChains[lchainId][li:]
					logger.Debug(fmt.Sprintf("Set new result for chain %s, result chain: %v", lchainId, blockedEvalsChains[lchainId]))
				}

				if len(blockedEvalsChains[lchainId]) == 0 { // в цепочке нет блокированных эвалов ее нужно удалить
					if scalingRequireCh.Remove(lchainId) { //cancel scaling event
						delete(firedEvents, lchainId)
						logger.Info(fmt.Sprintf("cancel scaling event %s", lchainId))
					} else if firedEvent, lok := firedEvents[lchainId]; lok {
						firedEvent.CtxCancelFn()
					}

					removedChains = append(removedChains, lchainId)
					continue
				}

				for _, lEvalId := range blockedEvalsChains[lchainId] {
					newevals[lEvalId] = evals[lEvalId]
				}

				if _, lok := firedEvents[lchainId]; lok {
					continue
				}

				blockedEval := evals[blockedEvalsChains[lchainId][0]]
				jobDescription := getJobInfoFromEvalWithRetry(_stalecnf, logger, nclient, blockedEval)
				structsJob := apiNomadJobToStructsJobV2(jobDescription)

				if !feasiblePoolByConstraint(_poolSpecs, structsJob) {
					logger.Debug(fmt.Sprintf("eval chain %s for job: %s/%s not fully feasible for my pools so skip it", lchainId, blockedEval.Namespace, blockedEval.JobID))
					removedChains = append(removedChains, lchainId)
					continue
				}

				unAllocatedTg := map[string]*ScalingEventTgInfo{}
				tgPools := GetOptimalPoolSpec(structsJob, _poolSpecs)
				logMsg := fmt.Sprintf("Fire \"no enough resources\" event(%s) for job %s/%s evalschain %v\n", lchainId, blockedEval.Namespace, blockedEval.JobID, blockedEvalsChains[lchainId])

				var latestFailedPlacement *nomad.Evaluation
				for _, eval := range getJobEvalsFromEvalWithRetry(_stalecnf, logger, nclient, blockedEval) {
					if latestFailedPlacement == nil || latestFailedPlacement.CreateIndex < eval.CreateIndex {
						latestFailedPlacement = eval
					}
				}

				summary := getJobSummaryWithRetry(_stalecnf, logger, nclient, blockedEval)
				for tgName, tgSummary := range summary.Summary {
					if _, lok := latestFailedPlacement.FailedTGAllocs[tgName]; lok {
						if _, lok := tgPools[tgName]; !lok {
							logger.Warn(fmt.Sprintf("No any pools suited for job: %s/%s taskgroup:%s, so skip \"no enough resources\" event", blockedEval.Namespace, blockedEval.JobID, tgName))
							_stateStat.IncNosuitedEvents()
							continue
						}

						if tgSummary.Queued > 0 {
							unAllocatedTg[tgName] = &ScalingEventTgInfo{PoolInfo: tgPools[tgName], UnAllocCount: tgSummary.Queued}
							logMsg += fmt.Sprintf("\ttaskgroup %s have queued allocs with failed placement: %d\n", tgName, tgSummary.Queued)
						}
					}
				}

				if len(unAllocatedTg) > 0 { //бросить событие на автоскейл
					var evntCtx context.Context
					var cancelFn context.CancelFunc
					if !_preventhung.Allow {
						evntCtx, cancelFn = context.WithCancel(context.Background())
					} else {
						evntCtx, cancelFn = context.WithTimeout(context.Background(), _preventhung.DetectPeriod)
					}
					lscalingEvent := &ScalingEvent{
						Id:            lchainId,
						Job:           structsJob,
						FireTime:      time.Now(),
						UnAllocatedTg: unAllocatedTg,
						Ctx:           evntCtx,
						CtxCancelFn:   cancelFn,
					}

					scalingRequireCh.Enqueue(lscalingEvent, lchainId)

					firedEvents[lchainId] = lscalingEvent

					logger.Info(logMsg)
				} else {
					logger.Warn(fmt.Sprintf("It's strange but can't find any unallocated taskgroup for event: %s", lchainId))
					removedChains = append(removedChains, lchainId)
				}
			}

			evals = newevals
			for _, lchainId := range removedChains {
				delete(blockedEvalsChains, lchainId)
				logger.Debug(fmt.Sprintf("removed chain %s", lchainId))
			}
		}
	}
}
