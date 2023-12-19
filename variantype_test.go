package main

import "testing"

func TestVariantType(t *testing.T) {
	var vslices []Variant
	for i := 0; i < 10; i++ {
		vi := NewVariantIntValue(i)
		vslices = append(vslices, vi)
	}

	v := NewVariantSliceValue(vslices)
	if v.GetType() != VariantTypeSlice {
		t.Log("Illegal Type in variant must by slice")
		t.FailNow()
	}

	for _, vv := range v.GetSliceValue() {
		if vv.GetType() != VariantTypeInt {
			t.Log("Illegal Type in variant slice item, must by int")
			t.FailNow()
		}

		t.Logf("v: %d", *vv.GetIntValue())
	}
}
