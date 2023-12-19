//go:build !windows

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/chrusty/go-tableprinter"
	"github.com/hashicorp/go-hclog"
)

func dumpSateonSignal(_stat *StateStat, _pools map[string]*Pool) {
	sigchnl := make(chan os.Signal, 1)
	signal.Notify(sigchnl, syscall.SIGUSR1)

	for range sigchnl {
		lreport := fmt.Sprintf("free scaling threads: %d(%d)\n", _stat.GetFreeScalingThreads(), _stat.GetTotalScalingThreads())
		lreport += fmt.Sprintf("free gc threads: %d(%d)\n", _stat.GetFreeGcThreads(), _stat.GetTotalGcThreads())
		lreport += fmt.Sprintf("accepted evals: %d\n", _stat.GetAcceptedEvals())
		lreport += fmt.Sprintf("accepted allocs: %d\n", _stat.GetAcceptedAllocs())
		lreport += fmt.Sprintf("timeouted scalings: %d\n", _stat.GetScalingTimeouts())
		lreport += fmt.Sprintf("nosuited scaling events: %d\n", _stat.GetNosuitedEvents())

		for _, lpool := range _pools {
			lpool.lock.Lock()

			if len(lreport) > 0 {
				lreport += "\n"
			}
			lreport += fmt.Sprintf("pool: %s have total nodes: %d, and allocs: %d\n", lpool.GetName(), len(lpool.nomadNodes), len(lpool.nomadAllocs))
			lreport += "  spec:\n"

			lpecbuf := bytes.NewBuffer(nil)
			printer := tableprinter.New().WithOutput(lpecbuf)

			if len(lpool.nomadNodes) > 0 {
				oneOfNodes := GetRandomElementOfMap(lpool.nomadNodes)

				printer.Print(map[string]interface{}{
					"Orig": dumpPoolSpectToYml(NewVariantMapValue(
						lpool.poolnodespec.Attributes), "", 0, []string{"provider"}),
					"OneOfNodes(" + strings.Split(oneOfNodes.ID, "-")[0] + ")": dumpPoolSpectToYml(NewVariantMapValue(
						lpool.poolnodespec.GetSpectFromNode(oneOfNodes).Attributes), "", 0, nil),
				})
			} else {
				printer.Print(map[string]interface{}{
					"Orig": dumpPoolSpectToYml(NewVariantMapValue(
						lpool.poolnodespec.Attributes), "", 0, []string{"provider"}),
					"OneOfNodes": "N/A",
				})
			}

			for _, lline := range strings.Split(lpecbuf.String(), "\n") {
				lreport += "    " + lline + "\n"
			}
			lreport += "\n"

			lreport += "  nodes:\n"
			for _, lnode := range lpool.nomadNodes {
				lallocCount := 0
				for _, lalloc := range lpool.nomadAllocs {
					if lnode.ID == lalloc.NodeID {
						lallocCount += 1
					}
				}

				lreport += fmt.Sprintf("    %s(%s), first event time: %s, allocCount: %d\n", lnode.ID, lnode.Status, lnode.Events[0].Timestamp, lallocCount)
			}

			lreport += "\n  allocs:\n"
			for _, lalloc := range lpool.nomadAllocs {
				lreport += fmt.Sprintf("    %s(%s)\n", lalloc.ID, lalloc.ClientStatus)
			}

			lpool.lock.Unlock()
		}

		hclog.L().Named("dumpState").Info(fmt.Sprintf("\n%s", lreport))
	}
}
