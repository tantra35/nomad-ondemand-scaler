package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/chrusty/go-tableprinter"
)

func TestDumpPoolSpectToYml(t *testing.T) {
	pools, err := parsePoolDifinition("./pools.devices.yml")
	if err != nil {
		t.Logf("can't parse pool yaml due: %s", err)
	}

	result := dumpPoolSpectToYml(NewVariantMapValue(pools[1].Attributes), "", 0, []string{"provider"})

	buf := bytes.NewBuffer(nil)
	printer := tableprinter.New().WithOutput(buf).WithSortedHeaders(true)

	printer.Print(map[string]interface{}{
		"Orig":       result,
		"OneOfNodes": "N/A",
	})

	lreport := ""

	for _, lline := range strings.Split(buf.String(), "\n") {
		lreport += "\t\t1" + lline + "\n"
	}

	t.Log("\n" + lreport)
}
