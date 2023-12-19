package main

import (
	"testing"
	"time"
)

func TestParseConfig(t *testing.T) {
	var config Config
	lerr := configParse("./config.hcl", &config)
	if lerr != nil {
		t.Fatalf("Can't parse config due: %s", lerr)
	}
}

func TestParseConfigWithoutStale(t *testing.T) {
	var config Config
	lerr := configParse("./config.hcl", &config)
	if lerr != nil {
		t.Fatalf("Can't parse config due: %s", lerr)
	}

	if len(config.StaleNomadApi) != 1 {
		t.Fatalf("Wrong stale config")
	}

	if !(config.StaleNomadApi[0].Allow == false && config.StaleNomadApi[0].StaleAllowedDuration == 0) {
		t.Fatalf("Wrong stale config: %v", config.StaleNomadApi[0])
	}
}

func TestParseConfigWithStale(t *testing.T) {
	var config Config
	lerr := configParse("./config.withallowestale.hcl", &config)
	if lerr != nil {
		t.Fatalf("Can't parse config due: %s", lerr)
	}

	if len(config.StaleNomadApi) != 1 {
		t.Fatalf("Wrong stale config")
	}

	if !(config.StaleNomadApi[0].Allow == true && config.StaleNomadApi[0].StaleAllowedDuration == time.Duration(30*time.Millisecond)) {
		t.Fatalf("Wrong stale config: %v", config.StaleNomadApi[0])
	}
}

func TestParseConfigAllowedFreexpr(t *testing.T) {
	var config Config
	lerr := configParse("./config.withgcexpression.hcl", &config)
	if lerr != nil {
		t.Fatalf("Can't parse config due: %s", lerr)
	}

	if config.GC == nil && len(config.GC) == 0 {
		t.Fatalf("GC config must not be nil")
	}

	if config.GC[0].AllowedFreexpr == nil {
		t.Fatalf("AllowedFreexpr must not be nil")
	}

	config.GC[0].AllowedFreexpr.SetDoubleVariableValue("totalnodes", float64(20))
	t.Logf("result: %d", int(config.GC[0].AllowedFreexpr.GetEvaluatedValue())) // int(config.GC[0].AllowedFreexpr.GetEvaluatedValue())
}
