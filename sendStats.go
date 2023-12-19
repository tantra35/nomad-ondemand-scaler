package main

import (
	"time"

	"github.com/armon/go-metrics"
)

func sendStats(_state *StateStat) {
	tiker := time.NewTicker(10 * time.Second)

	for range tiker.C {
		metrics.SetGauge([]string{"state", "freeGcThreads"}, float32(_state.GetFreeGcThreads()))
		metrics.SetGauge([]string{"state", "NumGcThreads"}, float32(_state.GetTotalGcThreads()))
		metrics.SetGauge([]string{"state", "freeScalingThreads"}, float32(_state.GetFreeScalingThreads()))
		metrics.SetGauge([]string{"state", "NumScalingThreads"}, float32(_state.GetTotalScalingThreads()))
		metrics.SetGauge([]string{"state", "AcceptedEvals"}, float32(_state.GetAcceptedEvals()))
		metrics.SetGauge([]string{"state", "AcceptedAllocs"}, float32(_state.GetAcceptedAllocs()))
		metrics.SetGauge([]string{"state", "ScalingTimeouts"}, float32(_state.GetScalingTimeouts()))
		metrics.SetGauge([]string{"state", "NoSuitedScalingEvents"}, float32(_state.GetNosuitedEvents()))
	}
}
