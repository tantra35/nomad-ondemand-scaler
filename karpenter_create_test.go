package main

import (
	"testing"

	"github.com/tantra35/nomad-ondemand-scaler/nodeprovider"
)

func TestCreatekarpenterprovider(t *testing.T) {
	pools, lerr := parsePoolDifinition("./pools.devices.karpenter.yml")
	if lerr != nil {
		t.Fatal(lerr)
	}

	poolWithKarpenter := pools[1]
	lpoolProviderAttr, lok := poolWithKarpenter.Attributes["provider"]
	if !lok {
		t.Fatal("provider attribute must be set in pool")
	}

	lproviderInfo := lpoolProviderAttr.GetMapValue()
	if lproviderInfo == nil {
		t.Fatal("provider info is nil")
	}

	lname, lok := lproviderInfo["name"]
	if !lok {
		t.Fatal("provider info have no name attribute")
	}

	if *lname.GetStringValue() != "karpenter" {
		t.Fatal("provider name is not karpenter")
	}

	lparamsVariant, lok := lproviderInfo["params"]
	if !lok {
		t.Fatal("provider info have no params attribute")
	}

	lresources := poolWithKarpenter.GetResources()
	lproviderResources := &nodeprovider.K8sKapenterProviderResources{
		Cpu:    lresources.Cpu,
		MemMB:  lresources.MemMB,
		DiskMB: lresources.DiskMB,
	}
	llprovider, lerr := nodeprovider.Createkarpenterprovider(variantToTypes(lparamsVariant), lproviderResources)
	if lerr != nil {
		t.Fatal(lerr)
	}

	t.Logf("provider: %v", llprovider)
}

func TestCreatekarpenterproviderOnlyLaunchTemplate(t *testing.T) {
	pools, lerr := parsePoolDifinition("./pools.devices.karpenter.yml")
	if lerr != nil {
		t.Fatal(lerr)
	}

	poolWithKarpenter := pools[2]
	lpoolProviderAttr, lok := poolWithKarpenter.Attributes["provider"]
	if !lok {
		t.Fatal("provider attribute must be set in pool")
	}

	lproviderInfo := lpoolProviderAttr.GetMapValue()
	if lproviderInfo == nil {
		t.Fatal("provider info is nil")
	}

	lname, lok := lproviderInfo["name"]
	if !lok {
		t.Fatal("provider info have no name attribute")
	}

	if *lname.GetStringValue() != "karpenter" {
		t.Fatal("provider name is not karpenter")
	}

	lparamsVariant, lok := lproviderInfo["params"]
	if !lok {
		t.Fatal("provider info have no params attribute")
	}

	lresources := poolWithKarpenter.GetResources()
	lproviderResources := &nodeprovider.K8sKapenterProviderResources{
		Cpu:    lresources.Cpu,
		MemMB:  lresources.MemMB,
		DiskMB: lresources.DiskMB,
	}
	llprovider, lerr := nodeprovider.Createkarpenterprovider(variantToTypes(lparamsVariant), lproviderResources)
	if lerr != nil {
		t.Fatal(lerr)
	}

	t.Logf("provider: %v", llprovider)
}
