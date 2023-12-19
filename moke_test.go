package main

import "github.com/hashicorp/nomad/nomad/structs"

func mokeNodeArmFromPoolNode(uid string, memAmount int) *structs.Node {
	_a1 := map[string]Variant{
		"cpu":              NewVariantIntValue(1000),
		"mem":              NewVariantIntValue(memAmount),
		"attr.arch":        NewVariantStringValue("arm64"),
		"attr.kernel.name": NewVariantStringValue("linux"),
		"datacenter":       NewVariantStringValue("test"),
		"drivers": NewVariantSliceValue([]Variant{
			NewVariantStringValue("docker"),
		}),
		"meta.project": NewVariantStringValue("homescapes"),
	}

	pn := NewPoolNodeSpec(_a1)
	return pn.GetNode(uid)
}

func mokeNodeX86FromPoolNode(uid string, memAmount int) *structs.Node {
	_a1 := map[string]Variant{
		"cpu":              NewVariantIntValue(1000),
		"mem":              NewVariantIntValue(memAmount),
		"attr.arch":        NewVariantStringValue("x86"),
		"attr.kernel.name": NewVariantStringValue("linux"),
		"datacenter":       NewVariantStringValue("test"),
		"drivers": NewVariantSliceValue([]Variant{
			NewVariantStringValue("docker"),
		}),
		"meta.project": NewVariantStringValue("homescapes"),
	}

	pn := NewPoolNodeSpec(_a1)
	return pn.GetNode(uid)
}
