package main

import (
	"context"
	"time"

	"github.com/Pramod-Devireddy/go-exprtk"
	"github.com/hashicorp/nomad/nomad/structs"
)

type ScalingEventTgInfo struct {
	UnAllocCount int
	PoolInfo     *PoolNodeSpec
}

type ScalingEvent struct {
	Id            string
	Job           *structs.Job
	FireTime      time.Time
	UnAllocatedTg map[string]*ScalingEventTgInfo
	Ctx           context.Context
	CtxCancelFn   context.CancelFunc
}

type Config struct {
	PoolConfig     string                    `mapstructure:"poolconfig" hcl:"poolconfig,label"`
	GC             []*GarbageCollectorConfig `hcl:"gc,block"`
	Telemetry      []*TelemetryConfig        `hcl:"telemetry,block"`
	StaleNomadApi  []*StaleApiConfig         `hcl:"stalenomadapi,block"`
	HungPrevention []*HungPreventionConfig   `hcl:"hungprevention,block"`
}

type HungPreventionConfig struct {
	Allow        bool          `mapstructure:"allow" hcl:"allow"`
	DetectPeriod time.Duration `mapstructure:"detect_period" hcl:"detect_period"`
}

type StaleApiConfig struct {
	Allow                bool          `mapstructure:"allow" hcl:"allow"`
	StaleAllowedDuration time.Duration `mapstructure:"duration" hcl:"duration"`
}

type GarbageCollectorConfig struct {
	CiclesToGc     int              `mapstructure:"cicles_to_gc" hcl:"cicles_to_gc,label"`
	CiclePeriod    time.Duration    `mapstructure:"cicle_period" hcl:"cicle_period"`
	AllowedFreexpr *exprtk.GoExprtk `mapstructure:"allowed_freexpr" hcl:"allowed_freexpr,label"`
}

type TelemetryConfig struct {
	StatSiteAddr string `mapstructure:"statsiteaddr" hcl:"statsiteaddr,label"`
	Prefix       string `mapstructure:"prefix" hcl:"prefix,label"`
}

type Opts struct {
	Verbose      []bool `short:"v" description:"Verbosity level"`
	ScaleThreads int    `long:"scalethreads" default:"1" description:"Number of scale threads"`
	ConfigPath   string `short:"c" long:"configpath" description:"Path to config file"`
}
