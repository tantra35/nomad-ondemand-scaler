//go:build windows

package main

import (
	"github.com/hashicorp/go-hclog"
)

func dumpSateonSignal(_stat *StateStat, _pools map[string]*Pool) {
	hclog.L().Named("dumpState").Info("Not supported on windows")
}
