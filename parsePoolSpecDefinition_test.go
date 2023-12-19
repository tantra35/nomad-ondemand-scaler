package main

import "testing"

func TestParsePoolDifinition(t *testing.T) {
	pools, err := parsePoolDifinition("./pools.yml")
	if err != nil {
		t.Logf("can't parce pool yaml due: %s", err)
		t.FailNow()
	}

	t.Logf("pools: %v", pools[0].GetName())
}

func TestParsePoolDifinitionWithReserved(t *testing.T) {
	pools, err := parsePoolDifinition("./pools.yml")
	if err != nil {
		t.Logf("can't parce pool yaml due: %s", err)
	}

	if pools[0].Attributes["reserved"].GetType() != VariantTypeMap {
		t.Logf("pool have wront type to reserved")
		t.FailNow()
	}

	reserved := pools[0].Attributes["reserved"].GetMapValue()
	if _, lok := reserved["mem"]; !lok {
		t.Logf("pool have have not mem in reserved")
		t.FailNow()
	}

	mem := reserved["mem"].GetIntValue()
	if mem == nil {
		t.Logf("pool have have wront type for mem in reserved")
		t.FailNow()
	}

	if *mem != 128*1024*1024 {
		t.Logf("mem in reserved have wrong value: %d", *mem)
		t.FailNow()
	}
}
