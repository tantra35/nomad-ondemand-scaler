package main

import (
	"os"
	"testing"

	nomad "github.com/hashicorp/nomad/api"
)

func TestFeasiblePool(t *testing.T) {
	testNodes := []*PoolNodeSpec{
		NewPoolNodeSpec(map[string]Variant{
			"cpu":              NewVariantIntValue(1000),
			"mem":              NewVariantIntValue(100),
			"attr.cpu.arch":    NewVariantStringValue("arm64"),
			"attr.kernel.name": NewVariantStringValue("linux"),
			"datacenter":       NewVariantStringValue("test"),
			"drivers": NewVariantSliceValue([]Variant{
				NewVariantStringValue("docker"),
			}),
			"meta.project": NewVariantStringValue("homescapes"),
		}),

		NewPoolNodeSpec(map[string]Variant{
			"cpu":              NewVariantIntValue(1000),
			"mem":              NewVariantIntValue(100),
			"attr.cpu.arch":    NewVariantStringValue("x86"),
			"attr.kernel.name": NewVariantStringValue("linux"),
			"datacenter":       NewVariantStringValue("test"),
			"drivers": NewVariantSliceValue([]Variant{
				NewVariantStringValue("docker"),
			}),
			"meta.project": NewVariantStringValue("homescapes"),
		}),
	}

	apiJob, lerr := parseJobFile("./carbon_local.atf01-multiarch.nomad")
	if lerr != nil {
		t.Logf("Can't parse job file due: %s", lerr)
		t.FailNow()
	}
	structsJob := apiNomadJobToStructsJobV2(apiJob)

	feasible := feasiblePoolByConstraint(testNodes, structsJob)
	if !feasible {
		t.Logf("not Feasible pool")
		t.FailNow()
	}
}

func TestFeasiblePoolFromRealAmrv01(t *testing.T) {
	os.Setenv("NOMAD_TOKEN", "b15bd264-e047-aab3-78f9-f709f1782511")
	os.Setenv("NOMAD_ADDR", "http://nomad.service.consul:4646")

	cnf := nomad.DefaultConfig()
	nclient, _ := nomad.NewClient(cnf)

	qoptions := &nomad.QueryOptions{
		Region: "amrv01",
	}
	apiJob, _, lerr := nclient.Jobs().Info("appsflyer-spark-daily/periodic-1695956400", qoptions)
	if lerr != nil {
		t.Fatalf("can't get nomad job from amrv01 due: %s", lerr)
	}

	pools, lerr := parsePoolDifinition("./pools.devices.amrv01.yml")
	if lerr != nil {
		t.Fatalf("can't parse pools definition due: %s", lerr)
	}

	structsJob := apiNomadJobToStructsJobV2(apiJob)
	is_feasible := feasiblePoolByConstraint(pools, structsJob)
	if is_feasible {
		t.Logf("not feasible pool")
	}
}

func TestFeasiblePoolFromRealAtf01(t *testing.T) {
	os.Setenv("NOMAD_TOKEN", "89f54fb8-edef-d2bb-656a-00392f3ef717")
	os.Setenv("NOMAD_ADDR", "http://nomad.service.consul:4646")

	cnf := nomad.DefaultConfig()
	nclient, _ := nomad.NewClient(cnf)

	qoptions := &nomad.QueryOptions{
		Region: "atf01",
	}
	apiJob, _, lerr := nclient.Jobs().Info("fixsaves-dev", qoptions)
	if lerr != nil {
		t.Fatalf("can't get nomad job from atf01 due: %s", lerr)
	}

	pools, lerr := parsePoolDifinition("./pools.atf01.yml")
	if lerr != nil {
		t.Fatalf("can't parse pools definition due: %s", lerr)
	}

	structsJob := apiNomadJobToStructsJobV2(apiJob)
	is_feasible := feasiblePoolByConstraint(pools, structsJob)
	if is_feasible {
		t.Logf("not feasible pool")
	}
}

func TestFeasiblePoolFromRealAtf01StableDiffusionHub(t *testing.T) {
	//vault read secrets/nomad/plr/atf01/creds/submit
	os.Setenv("NOMAD_TOKEN", "1d8a2c3f-432f-5ffa-874b-ba524fd8a4f4")
	os.Setenv("NOMAD_ADDR", "http://nomad.service.consul:4646")

	cnf := nomad.DefaultConfig()
	nclient, _ := nomad.NewClient(cnf)

	qoptions := &nomad.QueryOptions{
		Region: "atf01",
	}
	apiJob, _, lerr := nclient.Jobs().Info("stablediffusionhub/liashun-y", qoptions)
	if lerr != nil {
		t.Fatalf("can't get nomad job from atf01 due: %s", lerr)
	}

	pools, lerr := parsePoolDifinition("./pools.stablediffusionhub.yml")
	if lerr != nil {
		t.Fatalf("can't parse pools definition due: %s", lerr)
	}

	structsJob := apiNomadJobToStructsJobV2(apiJob)
	is_feasible := feasiblePoolByConstraint(pools, structsJob)
	if is_feasible {
		t.Logf("not feasible pool")
	}
}

func TestFeasiblePoolFromSsshbastion(t *testing.T) {
	apiJob, lerr := parseJobFile("./ruslan-sshbastion.nomad")
	if lerr != nil {
		t.Logf("Can't parse job file due: %s", lerr)
		t.FailNow()
	}
	structsJob := apiNomadJobToStructsJobV2(apiJob)

	pools, lerr := parsePoolDifinition("./pools.sshbastion.yml")
	if lerr != nil {
		t.Fatalf("can't parse pools definition due: %s", lerr)
	}

	is_feasible := feasiblePoolByConstraint(pools, structsJob)
	if !is_feasible {
		t.Logf("not feasible pool")
		t.FailNow()
	}
}

func TestFeasiblePoolFromSsshbastionWrongPoolDescription(t *testing.T) {
	apiJob, lerr := parseJobFile("./ruslan-sshbastion-withdevice.nomad")
	if lerr != nil {
		t.Logf("Can't parse job file due: %s", lerr)
		t.FailNow()
	}
	structsJob := apiNomadJobToStructsJobV2(apiJob)

	pools, lerr := parsePoolDifinition("./pools.sshbastion.wrongdescr.yml")
	if lerr != nil {
		t.Fatalf("can't parse pools definition due: %s", lerr)
	}

	is_feasible := feasiblePoolByConstraint(pools, structsJob)
	if is_feasible {
		t.Logf("pools must by non feasible")
		t.FailNow()
	}
}

func TestFeasiblePoolFromSsshbastionWithoutDevices(t *testing.T) {
	apiJob, lerr := parseJobFile("./ruslan-sshbastion-withdevice.nomad")
	if lerr != nil {
		t.Logf("Can't parse job file due: %s", lerr)
		t.FailNow()
	}
	structsJob := apiNomadJobToStructsJobV2(apiJob)

	pools, lerr := parsePoolDifinition("./pools.sshbastion.withoutdevices.yml")
	if lerr != nil {
		t.Fatalf("can't parse pools definition due: %s", lerr)
	}

	is_feasible := feasiblePoolByConstraint(pools, structsJob)
	if is_feasible {
		t.Logf("pools must by non feasible")
		t.FailNow()
	}
}
