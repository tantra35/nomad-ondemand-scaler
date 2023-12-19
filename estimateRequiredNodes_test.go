package main

import (
	"context"
	"os"
	"testing"
	"time"

	nomad "github.com/hashicorp/nomad/api"
)

func TestEstimateRequiredNodes(t *testing.T) {
	poolSpecs, lerr := parsePoolDifinition("./pools.yml")
	if lerr != nil {
		t.Logf("Can't parse pools spec file due: %s", lerr)
		t.FailNow()
	}

	lpool, lerr := NewPool(poolSpecs[1])
	if lerr != nil {
		t.Logf("Can't create pool from  spec due: %s", lerr)
		t.FailNow()
	}

	apiJob, lerr := parseJobFile("./carbon_local.atf01-multiarch.nomad")
	if lerr != nil {
		t.Logf("Can't parse job file due: %s", lerr)
		t.FailNow()
	}

	structsJob := apiNomadJobToStructsJobV2(apiJob)

	ephemeralnomadNodes, ephemeralnomadAllocs := estimateRequiredNodes(lpool, nil, nil, structsJob, structsJob.TaskGroups[1], structsJob.TaskGroups[1].Count)
	t.Logf("nodesCount: %d, allocsCount: %d", len(ephemeralnomadNodes), len(ephemeralnomadAllocs))
	t.FailNow()
}

func TestEstimateRequiredNodesWithReservedInNodes(t *testing.T) {
	poolSpecs, lerr := parsePoolDifinition("./pools.yml")
	if lerr != nil {
		t.Logf("Can't parse pools spec file due: %s", lerr)
		t.FailNow()
	}

	lpool, lerr := NewPool(poolSpecs[0])
	if lerr != nil {
		t.Logf("Can't create pool from  spec due: %s", lerr)
		t.FailNow()
	}

	apiJob, lerr := parseJobFile("./test-escape-parametes.nomad")
	if lerr != nil {
		t.Logf("Can't parse job file due: %s", lerr)
		t.FailNow()
	}

	structsJob := apiNomadJobToStructsJobV2(apiJob)

	ephemeralnomadNodes, ephemeralnomadAllocs := estimateRequiredNodes(lpool, nil, nil, structsJob, structsJob.TaskGroups[0], structsJob.TaskGroups[0].Count)
	t.Logf("nodesCount: %d, allocsCount: %d", len(ephemeralnomadNodes), len(ephemeralnomadAllocs))
	t.FailNow()
}

func TestEstimateRequiredNodesVagrant(t *testing.T) {
	poolSpecs, lerr := parsePoolDifinition("./pools.vagrant.yml")
	if lerr != nil {
		t.Logf("Can't parse pools spec file due: %s", lerr)
		t.FailNow()
	}

	lpool, lerr := NewPool(poolSpecs[0])
	if lerr != nil {
		t.Logf("Can't create pool from  spec due: %s", lerr)
		t.FailNow()
	}

	apiJob01, lerr := parseJobFile("./test-escape-parametes.nomad")
	if lerr != nil {
		t.Logf("Can't parse job file due: %s", lerr)
		t.FailNow()
	}
	structsJob01 := apiNomadJobToStructsJobV2(apiJob01)
	ephemeralnomadNodes, ephemeralnomadAllocs := estimateRequiredNodes(lpool, nil, nil, structsJob01, structsJob01.TaskGroups[0], structsJob01.TaskGroups[0].Count)
	t.Logf("nodesCount: %d, allocsCount: %d", len(ephemeralnomadNodes), len(ephemeralnomadAllocs))

	lctx := context.TODO()
	go lpool.Update(lctx, ephemeralnomadNodes, ephemeralnomadAllocs)
	time.Sleep(1 * time.Second)

	apiJob02, lerr := parseJobFile("./test-escape-parametes.namespace.nomad")
	if lerr != nil {
		t.Logf("Can't parse job file due: %s", lerr)
		t.FailNow()
	}
	structsJob02 := apiNomadJobToStructsJobV2(apiJob02)
	ephemeralnomadNodes, ephemeralnomadAllocs = estimateRequiredNodes(lpool, nil, nil, structsJob02, structsJob02.TaskGroups[0], structsJob02.TaskGroups[0].Count)
	t.Logf("nodesCount: %d, allocsCount: %d", len(ephemeralnomadNodes), len(ephemeralnomadAllocs))

	t.FailNow()
}

func TestEstimateRequiredNodesAtf01StabbleDiffusion(t *testing.T) {
	//vault read secrets/nomad/plr/atf01/creds/submit
	os.Setenv("NOMAD_TOKEN", "1d8a2c3f-432f-5ffa-874b-ba524fd8a4f4")
	os.Setenv("NOMAD_ADDR", "http://nomad.service.consul:4646")

	poolSpecs, lerr := parsePoolDifinition("./pools.stablediffusionhub.yml")
	if lerr != nil {
		t.Logf("Can't parse pools spec file due: %s", lerr)
		t.FailNow()
	}

	lpool, lerr := NewPool(poolSpecs[0])
	if lerr != nil {
		t.Fatalf("Can't create pool from  spec due: %s", lerr)
	}

	cnf := nomad.DefaultConfig()
	nclient, _ := nomad.NewClient(cnf)

	qoptions := &nomad.QueryOptions{
		Region: "atf01",
	}
	apiJob01, _, lerr := nclient.Jobs().Info("stablediffusionhub/liashun-y", qoptions)
	if lerr != nil {
		t.Fatalf("can't get nomad job from atf01 due: %s", lerr)
	}

	structsJob01 := apiNomadJobToStructsJobV2(apiJob01)
	ephemeralnomadNodes, ephemeralnomadAllocs := estimateRequiredNodes(lpool, nil, nil, structsJob01, structsJob01.TaskGroups[0], structsJob01.TaskGroups[0].Count)
	t.Logf("nodesCount: %d, allocsCount: %d", len(ephemeralnomadNodes), len(ephemeralnomadAllocs))

	t.FailNow()
}
