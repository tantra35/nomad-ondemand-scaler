package main

import (
	"fmt"
	"sort"
	"strings"
)

func dumpPoolSpectToYml(_v Variant, _newField string, depth int, _lastfield []string) string {
	result := _newField

	switch _v.GetType() {
	case VariantTypeInt:
		result += fmt.Sprintf("%d", *_v.GetIntValue())

	case VariantTypeString:
		result += AbbreviateString(*_v.GetStringValue(), 30)

	case VariantTypeBool:
		result += fmt.Sprintf("%v", *_v.GetBoolValue())

	case VariantTypeSlice:
		list := _v.GetSliceValue()
		for _, lv := range list {
			result += dumpPoolSpectToYml(lv, fmt.Sprintf("\n%s  - ", strings.Repeat("  ", depth)), depth+1, nil)
		}

	case VariantTypeMap:
		lmap := _v.GetMapValue()
		lkeys := make([]string, 0, len(lmap))
		lkeysLast := make([]string, 0, len(_lastfield))
		for lkey := range lmap {
			if !containsInSlice(_lastfield, lkey) {
				lkeys = append(lkeys, lkey)
			} else {
				lkeysLast = append(lkeysLast, lkey)
			}
		}
		sort.Strings(lkeys)
		sort.Strings(lkeysLast)

		lindex := 0
		for _, lkey := range append(lkeys, lkeysLast...) {
			lv := lmap[lkey]
			if lindex == 0 && strings.Trim(_newField, "\n ") == "-" {
				result += dumpPoolSpectToYml(lv, fmt.Sprintf("%s: ", lkey), depth+1, nil)
			} else {
				result += dumpPoolSpectToYml(lv, fmt.Sprintf("\n%s  %s: ", strings.Repeat("  ", depth), lkey), depth+1, nil)
			}

			lindex += 1
		}
	}

	return result
}
