package main

import (
	"fmt"
	"os"
	"reflect"

	"github.com/Pramod-Devireddy/go-exprtk"
	"github.com/hashicorp/hcl"
	"github.com/mitchellh/mapstructure"
)

func decodeVariable(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t != reflect.TypeOf(&exprtk.GoExprtk{}) {
		return decodeToTimeDuration(f, t, data)
	}

	if f.Kind() != reflect.String {
		return nil, fmt.Errorf("wrong type in value for exprtk.GoExprtk")
	}

	exprtkObj := exprtk.NewExprtk()
	exprtkObj.SetExpression(data.(string))
	exprtkObj.AddDoubleVariable("totalnodes")
	exprtkObj.AddDoubleVariable("busynodes")

	lerr := exprtkObj.CompileExpression()
	if lerr != nil {
		exprtkObj.Delete()
		return nil, fmt.Errorf("can't compile expression: %s", lerr)
	}

	return &exprtkObj, nil
}

func configParse(_cnfPath string, _opts *Config) error {
	lconfigbytes, lerr := os.ReadFile(_cnfPath)
	if lerr != nil {
		return fmt.Errorf("can't parse configfile due: %s", lerr)
	}

	lconfigast, lerr := hcl.ParseBytes(lconfigbytes)
	if lerr != nil {
		return fmt.Errorf("can't parse configfile due: %s", lerr)
	}

	var m map[string]interface{}
	lerr = hcl.DecodeObject(&m, lconfigast.Node)
	if lerr != nil {
		return fmt.Errorf("can't decode parsed configfile due: %s", lerr)
	}

	config := &mapstructure.DecoderConfig{
		DecodeHook:       decodeVariable,
		Metadata:         nil,
		Result:           _opts,
		WeaklyTypedInput: true,
	}

	decoder, lerr := mapstructure.NewDecoder(config)
	if lerr == nil {
		lerr = decoder.Decode(m)
	}

	if len(_opts.StaleNomadApi) == 0 {
		_opts.StaleNomadApi = []*StaleApiConfig{
			{
				Allow:                false,
				StaleAllowedDuration: 0,
			},
		}
	}

	if len(_opts.HungPrevention) == 0 {
		_opts.HungPrevention = []*HungPreventionConfig{
			{
				Allow: false,
			},
		}
	}

	return lerr
}
