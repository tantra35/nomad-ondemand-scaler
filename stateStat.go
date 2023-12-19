package main

import (
	"sync/atomic"
)

type StateStat struct {
	evalsCount      atomic.Uint64
	allocsCount     atomic.Uint64
	scalingTimeouts atomic.Uint64
	nosuitedEvents  atomic.Uint64

	freeScalingThreads  atomic.Int64
	freeGcThreads       atomic.Int64
	totalScalingThreads int
	totalGcThreads      int
}

func NewStateStat() *StateStat {
	return &StateStat{}
}

func (s *StateStat) IncFreeScalingThreads() {
	s.freeScalingThreads.Add(1)
}

func (s *StateStat) DecFreeScalingThreads() {
	s.freeScalingThreads.Add(-1)
}

func (s *StateStat) GetFreeScalingThreads() int {
	return int(s.freeScalingThreads.Load())
}

func (s *StateStat) IncFreeGcThreads() {
	s.freeGcThreads.Add(1)
}

func (s *StateStat) DecFreeGcThreads() {
	s.freeGcThreads.Add(-1)
}

func (s *StateStat) GetFreeGcThreads() int {
	return int(s.freeGcThreads.Load())
}

func (s *StateStat) SetTotalScalingThreads(_scalingthreadsCount int) {
	s.totalScalingThreads = _scalingthreadsCount
}

func (s *StateStat) GetTotalScalingThreads() int {
	return s.totalScalingThreads
}

func (s *StateStat) SetTotalGcThreads(_gcthreadsCount int) {
	s.totalGcThreads = _gcthreadsCount
}

func (s *StateStat) GetTotalGcThreads() int {
	return s.totalGcThreads
}

func (s *StateStat) IncAcceptedEvals() {
	s.evalsCount.Add(1)
}

func (s *StateStat) IncAcceptedAllocs() {
	s.allocsCount.Add(1)
}

func (s *StateStat) GetAcceptedEvals() uint64 {
	return s.evalsCount.Load()
}

func (s *StateStat) GetAcceptedAllocs() uint64 {
	return s.allocsCount.Load()
}

func (s *StateStat) IncScalingTimeouts() {
	s.scalingTimeouts.Add(1)
}

func (s *StateStat) GetScalingTimeouts() uint64 {
	return s.scalingTimeouts.Load()
}

func (s *StateStat) IncNosuitedEvents() {
	s.nosuitedEvents.Add(1)
}

func (s *StateStat) GetNosuitedEvents() uint64 {
	return s.nosuitedEvents.Load()
}
