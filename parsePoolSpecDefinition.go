package main

import (
	"fmt"
	"os"

	"github.com/dustin/go-humanize"
	"gopkg.in/yaml.v3"
)

func parsePoolDifinitionCastWrapedToInterface(_from map[string]*VariantYamlUnmarshaled) map[string]Variant {
	to := map[string]Variant{}
	for k, v := range _from {
		to[k] = v
	}

	return to
}

func parsePoolDifinition(_yamlPath string) ([]*PoolNodeSpec, error) {
	fileData, lerr := os.ReadFile(_yamlPath)
	if lerr != nil {
		return nil, fmt.Errorf("can't read file due: %s", lerr)
	}

	data := []map[string]*VariantYamlUnmarshaled{}
	lerr = yaml.Unmarshal([]byte(fileData), &data)
	if lerr != nil {
		return nil, fmt.Errorf("can't unmarshal err: %s", lerr)
	}

	var pools []*PoolNodeSpec
	for _, poolData := range data {
		poolDataCasted := parsePoolDifinitionCastWrapedToInterface(poolData)
		if value, lok := poolData["mem"]; lok {
			if value.GetType() == VariantTypeString {
				bytesValue, lerr := humanize.ParseBytes(*value.GetStringValue())
				if lerr != nil {
					return nil, fmt.Errorf("can't parse mem field due: %s", lerr)
				}

				poolDataCasted["mem"] = NewVariantIntValue(int(bytesValue / 1024 / 1024))
			}
		}

		if value, lok := poolData["disk"]; lok {
			if value.GetType() == VariantTypeString {
				bytesValue, lerr := humanize.ParseBytes(*value.GetStringValue())
				if lerr != nil {
					return nil, fmt.Errorf("can't parse disk field due: %s", lerr)
				}

				poolDataCasted["disk"] = NewVariantIntValue(int(bytesValue / 1024 / 1024))
			}
		}

		if reserved, lok := poolDataCasted["reserved"]; lok {
			if reserved.GetType() == VariantTypeMap {
				reservedMap := reserved.GetMapValue()

				if value, lok := reservedMap["mem"]; lok {
					if value.GetType() == VariantTypeString {
						bytesValue, lerr := humanize.ParseBytes(*value.GetStringValue())
						if lerr != nil {
							return nil, fmt.Errorf("can't parse mem field of reserved due: %s", lerr)
						}

						reservedMap["mem"] = NewVariantIntValue(int(bytesValue / 1024 / 1024))
					}
				}

				if value, lok := reservedMap["disk"]; lok {
					if value.GetType() == VariantTypeString {
						bytesValue, lerr := humanize.ParseBytes(*value.GetStringValue())
						if lerr != nil {
							return nil, fmt.Errorf("can't parse disk field of reserved due: %s", lerr)
						}

						reservedMap["disk"] = NewVariantIntValue(int(bytesValue / 1024 / 1024))
					}
				}
			}
		}

		if ldevices, lok := poolDataCasted["devices"]; lok {
			if ldevices.GetType() == VariantTypeSlice {
				devicesList := ldevices.GetSliceValue()

				for _, ldevice := range devicesList {
					if ldevice.GetType() != VariantTypeMap {
						continue
					}

					ldeviceMap := ldevice.GetMapValue()
					if _, lok := ldeviceMap["count"]; !lok {
						ldeviceMap["count"] = NewVariantIntValue(1)
					}
				}
			}
		}

		newpool := NewPoolNodeSpec(poolDataCasted)
		pools = append(pools, newpool)
	}

	return pools, nil
}
