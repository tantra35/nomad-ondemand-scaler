package main

import (
	"testing"

	"github.com/hashicorp/nomad/helper/uuid"
	"github.com/hashicorp/nomad/nomad/structs"
)

func TestPoolNodeSpecDifferentDrivers(t *testing.T) {
	_a1 := map[string]Variant{
		"cpu":              NewVariantIntValue(1000),
		"mem":              NewVariantIntValue(2000),
		"attr.arch":        NewVariantStringValue("amd64"),
		"attr.kernel.name": NewVariantStringValue("linux"),
		"datacenter":       NewVariantStringValue("test"),
		"drivers": NewVariantSliceValue([]Variant{
			NewVariantStringValue("docker"),
		}),
	}

	pna1 := NewPoolNodeSpec(_a1)
	csa1, _ := pna1.ComputeClass()
	t.Logf("compute class a1: %s", csa1)

	_a2 := map[string]Variant{
		"cpu":        NewVariantIntValue(2000),
		"mem":        NewVariantIntValue(4000),
		"attr.arch":  NewVariantStringValue("amd64"),
		"attr.os":    NewVariantStringValue("linux"),
		"datacenter": NewVariantStringValue("test"),
		"drivers": NewVariantSliceValue([]Variant{
			NewVariantStringValue("exec"),
		}),
	}

	pna2 := NewPoolNodeSpec(_a2)
	csa2, _ := pna2.ComputeClass()
	t.Logf("compute class a2: %s", csa2)

	if csa1 == csa2 {
		t.FailNow()
	}
}

func TestPoolNodeDifferentProjects(t *testing.T) {
	_a1 := map[string]Variant{
		"cpu":          NewVariantIntValue(1000),
		"mem":          NewVariantIntValue(2000),
		"attr.arch":    NewVariantStringValue("amd64"),
		"attr.os":      NewVariantStringValue("linux"),
		"datacenter":   NewVariantStringValue("test"),
		"meta.project": NewVariantStringValue("homescapes"),
	}

	pna1 := NewPoolNodeSpec(_a1)
	csa1, _ := pna1.ComputeClass()
	t.Logf("compute class a1: %s", csa1)

	_a2 := map[string]Variant{
		"cpu":          NewVariantIntValue(2000),
		"mem":          NewVariantIntValue(4000),
		"attr.arch":    NewVariantStringValue("amd64"),
		"attr.os":      NewVariantStringValue("linux"),
		"datacenter":   NewVariantStringValue("test"),
		"meta.project": NewVariantStringValue("gardenscapes"),
	}

	pna2 := NewPoolNodeSpec(_a2)
	csa2, _ := pna2.ComputeClass()
	t.Logf("compute class a2: %s", csa2)

	if csa1 == csa2 {
		t.FailNow()
	}
}

func mokeNodePoolNode(memAmount int, arch string) *PoolNodeSpec {
	_a1 := map[string]Variant{
		"cpu":              NewVariantIntValue(1000),
		"mem":              NewVariantIntValue(memAmount),
		"attr.arch":        NewVariantStringValue(arch),
		"attr.kernel.name": NewVariantStringValue("linux"),
		"datacenter":       NewVariantStringValue("test"),
		"drivers": NewVariantSliceValue([]Variant{
			NewVariantStringValue("docker"),
		}),
		"meta.project": NewVariantStringValue("homescapes"),
	}

	return NewPoolNodeSpec(_a1)
}

func TestGetSuitableNodes(t *testing.T) {
	//apiJob, lerr := parseJobFile("./carbon_local.atf01-x86.nomad")
	apiJob, lerr := parseJobFile("./carbon_local.atf01-multiarch.nomad")
	//	apiJob, lerr := parseJobFile("./carbon_local.atf01-arm64.nomad")
	if lerr != nil {
		t.Logf("Can't parse job file due: %s", lerr)
		t.FailNow()
	}

	apiJob.Canonicalize()
	structsJob := apiNomadJobToStructsJobV2(apiJob)
	structsJob.Status = structs.JobStatusPending

	lid := uuid.Generate()
	testNodes := []*structs.Node{}

	testNodes = append(testNodes, mokeNodeX86FromPoolNode("x86node-2048-"+lid, 2048))
	testNodes = append(testNodes, mokeNodeArmFromPoolNode("armnodepoll-512-"+lid, 512))
	testNodes = append(testNodes, mokeNodeArmFromPoolNode("armnodepoll-1024-"+lid, 1024))
	testNodes = append(testNodes, mokeNodeArmFromPoolNode("armnodepoll-2048-"+lid, 2048))

	result := getSuitableNodes("", structsJob, testNodes)
	t.Logf("result: %s", result)
}

func TestGetOptimalPool(t *testing.T) {
	//apiJob, lerr := parseJobFile("./carbon_local.atf01-arm64.nomad")
	apiJob, lerr := parseJobFile("./carbon_local.atf01-multiarch.nomad")
	if lerr != nil {
		t.Logf("Can't parse job file due: %s", lerr)
		t.FailNow()
	}

	structsJob := apiNomadJobToStructsJobV2(apiJob)
	structsJob.Status = structs.JobStatusPending

	nodes := []*PoolNodeSpec{}
	nodes = append(nodes, mokeNodePoolNode(2048, "x86"))
	nodes = append(nodes, mokeNodePoolNode(2048, "arm64"))
	nodes = append(nodes, mokeNodePoolNode(512, "arm64"))
	nodes = append(nodes, mokeNodePoolNode(1024, "arm64"))

	optimalNodes := GetOptimalPoolSpec(structsJob, nodes)

	for tgName, pool := range optimalNodes {
		t.Logf("tg %s: %s", tgName, pool.GetName())
	}
}

func TestGetOptimalPoolForBadDatacenter(t *testing.T) {
	//apiJob, lerr := parseJobFile("./carbon_local.atf01-arm64.nomad")
	apiJob, lerr := parseJobFile("./carbon_local.atf01-multiarch.dc-fake.nomad")
	if lerr != nil {
		t.Logf("Can't parse job file due: %s", lerr)
		t.FailNow()
	}

	structsJob := apiNomadJobToStructsJobV2(apiJob)

	nodes := []*PoolNodeSpec{}
	nodes = append(nodes, mokeNodePoolNode(2048, "x86"))
	nodes = append(nodes, mokeNodePoolNode(2048, "arm64"))
	nodes = append(nodes, mokeNodePoolNode(512, "arm64"))
	nodes = append(nodes, mokeNodePoolNode(1024, "arm64"))

	optimalNodes := GetOptimalPoolSpec(structsJob, nodes)

	for tgName, pool := range optimalNodes {
		t.Logf("tg %s: %s", tgName, pool.GetName())
	}
}

func TestParsePoolWithDevices(t *testing.T) {
	poolSpec, lerr := parsePoolDifinition("./pools.devices.yml")
	if lerr != nil {
		t.Logf("Can't parse pools spec file due: %s", lerr)
		t.FailNow()
	}

	lnomadNode := poolSpec[1].GetNode("testnode")

	if len(lnomadNode.NodeResources.Devices) == 0 {
		t.Logf("node must hold devices")
		t.FailNow()
	}

	ldevice := lnomadNode.NodeResources.Devices[0]
	if ldevice.Name != "NVIDIA A10G" {
		t.Logf("device name wrong")
		t.FailNow()
	}

	if len(ldevice.Attributes) == 0 {
		t.Logf("device must have attributes")
		t.FailNow()
	}

	t.Logf("device: %v", ldevice)
}

func TestGetOptimalPoolSpecWithDevices(t *testing.T) {
	poolSpecs, lerr := parsePoolDifinition("./pools.sshbastion.yml")
	if lerr != nil {
		t.Logf("Can't parse pools spec file due: %s", lerr)
		t.FailNow()
	}

	apiJob, lerr := parseJobFile("./ruslan-sshbastion-withdevice.nomad")
	if lerr != nil {
		t.Logf("Can't parse job file due: %s", lerr)
		t.FailNow()
	}

	structsJob := apiNomadJobToStructsJobV2(apiJob)
	optimal := GetOptimalPoolSpec(structsJob, poolSpecs)

	if len(optimal) == 0 {
		t.Fatalf("no optimal pool found")
	}

	for tgName, pool := range optimal {
		if _, lok := pool.Attributes["devices"]; !lok {
			t.Fatalf("tg %s must have pool with devices", tgName)
		}
	}

	t.Logf("optimal: %v", optimal)
}

func TestGetOptimalPoolSpecWithoutDevices(t *testing.T) {
	poolSpecs, lerr := parsePoolDifinition("./pools.sshbastion.yml")
	if lerr != nil {
		t.Logf("Can't parse pools spec file due: %s", lerr)
		t.FailNow()
	}

	apiJob, lerr := parseJobFile("./ruslan-sshbastion.nomad")
	if lerr != nil {
		t.Logf("Can't parse job file due: %s", lerr)
		t.FailNow()
	}

	structsJob := apiNomadJobToStructsJobV2(apiJob)
	optimal := GetOptimalPoolSpec(structsJob, poolSpecs)

	if len(optimal) == 0 {
		t.Fatalf("no optimal pool found")
	}

	for tgName, pool := range optimal {
		if _, lok := pool.Attributes["devices"]; lok {
			t.Fatalf("tg %s must not have pool with devices", tgName)
		}
	}

	t.Logf("optimal: %v", optimal)
}
