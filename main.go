package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/armon/go-metrics"
	"github.com/hashicorp/go-hclog"
	nomad "github.com/hashicorp/nomad/api"
	"github.com/jessevdk/go-flags"
)

func processNodes(_pools map[string]*Pool, nodeCh <-chan *nomad.Node) {
	for _node := range nodeCh {
		for _, pool := range _pools {
			if pool.tryNomadNode(_node) {
				break
			}
		}
	}
}

func processAllocs(_pools map[string]*Pool, allocsCh <-chan *nomad.Allocation) {
	for _alloc := range allocsCh {
		for _, pool := range _pools {
			if pool.tryNomadAllocation(_alloc) {
				break
			}
		}
	}
}

func main() {
	appLogger := hclog.New(&hclog.LoggerOptions{
		Name:       "nomad-ondemand-scaler",
		TimeFormat: "[2006-01-02 15:04:05.000]",
		Level:      hclog.Info,
	})

	log.SetOutput(appLogger.StandardWriter(&hclog.StandardLoggerOptions{InferLevels: true}))
	log.SetPrefix("")
	log.SetFlags(0)

	hclog.SetDefault(appLogger)
	var opts Opts

	_, err := flags.Parse(&opts)
	if err != nil {
		if lperr, ok := err.(*flags.Error); ok {
			switch lperr.Type {
			case flags.ErrHelp:
				return
			case flags.ErrUnknown:
				log.Fatal(err)
			case flags.ErrTag:
				log.Fatal(err)
			}

			return
		} else {
			log.Fatal(err)
		}
	}

	switch len(opts.Verbose) {
	case 0:
		appLogger.SetLevel(hclog.Warn)
	case 1:
		appLogger.SetLevel(hclog.Info)
	case 2:
		fallthrough
	case 3:
		appLogger.SetLevel(hclog.Debug)
	default:
		appLogger.SetLevel(hclog.Trace)
	}

	var config Config
	lerr := configParse(opts.ConfigPath, &config)
	if lerr != nil {
		log.Fatalf("[ERROR] can't parse config due: %s", lerr)
	}

	poolSpecs, lerr := parsePoolDifinition(config.PoolConfig)
	if lerr != nil {
		log.Fatalf("[ERROR] can't parce pool yaml due: %s", lerr)
	}

	lstateStat := NewStateStat() // объект статистической инфы

	if config.Telemetry != nil {
		l_metricsConfig := metrics.DefaultConfig(config.Telemetry[0].Prefix)
		l_metricsConfig.EnableHostname = false
		l_metricsConfig.EnableRuntimeMetrics = false

		lstatsSink, lerr := metrics.NewStatsiteSink(config.Telemetry[0].StatSiteAddr)
		if lerr != nil {
			log.Fatalf("[ERROR] can't init statsite sink due: %s", lerr)
		}
		metrics.NewGlobal(l_metricsConfig, lstatsSink)
		go sendStats(lstateStat)
	}

	cnf := nomad.DefaultConfig()
	nclient, lerr := nomad.NewClient(cnf)
	if lerr != nil {
		log.Fatalf("[ERROR] can't init nomad client due: %s", lerr)
	}

	lpools, lnomadLastIndex, lerr := createPools(config.StaleNomadApi[0], nclient, poolSpecs)
	if lerr != nil {
		log.Fatalf("[ERROR] can't create pools due: %s", lerr)
	}

	scalingRequireCh := NewQueue[*ScalingEvent]()
	scalingDoneCh := make(chan string, 10)

	evalCh := make(chan *nomad.Evaluation)
	go processEvals(lstateStat, config.StaleNomadApi[0], config.HungPrevention[0], evalCh, scalingRequireCh, scalingDoneCh, poolSpecs)

	lstateStat.SetTotalScalingThreads(opts.ScaleThreads)
	for i := 0; i < opts.ScaleThreads; i++ {
		hclog.L().Info(fmt.Sprintf("Start scale thread %d", i+1))
		lstateStat.IncFreeScalingThreads()
		go scalingAction(lstateStat, lpools, scalingRequireCh, scalingDoneCh)
	}

	if opts.ScaleThreads > 0 {
		hclog.L().Info("Start gc thread")
		lstateStat.SetTotalGcThreads(1)
		lstateStat.IncFreeGcThreads()
		go gcAction(lstateStat, config.GC[0], nclient, lpools)
	}

	nodeCh := make(chan *nomad.Node)
	go processNodes(lpools, nodeCh)

	allocCh := make(chan *nomad.Allocation)
	go processAllocs(lpools, allocCh)

	go dumpSateonSignal(lstateStat, lpools)

	//в nomad 1.1.x evals не поддерживают зведочку в качестве wildcard для всех неймспейсов, в 1.3 это уже поправлено, и этот код можно будет упростить
	namespcpaces, _, _ := nclient.Namespaces().List(nil)
	for _, namespace := range namespcpaces {
		lnqoptions := nomad.QueryOptions{Namespace: namespace.Name, AllowStale: config.StaleNomadApi[0].Allow}
		evals, _, lerr := nclient.Evaluations().List(&lnqoptions)
		if lerr != nil {
			log.Fatalf("can't get evals due: %s", lerr)
		}

		for _, leval := range evals {
			hclog.L().Trace(fmt.Sprintf("list eval: %s, status: %s", leval.ID, leval.Status))
			evalCh <- leval
		}
	}

	var streamretrytime time.Duration = 5
	hclog.L().Info("Start main loop")

	for {
		lnetopics := map[nomad.Topic][]string{
			nomad.TopicEvaluation: {"*"},
			nomad.TopicNode:       {"*"},
			nomad.TopicAllocation: {"*"},
		}
		lnqoptions := nomad.QueryOptions{Namespace: nomad.AllNamespacesNamespace, AllowStale: false}
		streamcancelCtx, streamcancelFn := context.WithCancel(context.Background())

		lech, lerr := nclient.EventStream().Stream(streamcancelCtx, lnetopics, lnomadLastIndex, &lnqoptions)
		if lerr != nil {
			hclog.L().Warn(fmt.Sprintf("can't init nomad event stream due: %s. Retry after %d seconds", lerr, streamretrytime))

			time.Sleep(streamretrytime * time.Second)
			streamretrytime += 5

			if streamretrytime > 60 {
				streamretrytime = 5
			}

			continue
		}

		for lei := range lech {
			if lei.Err != nil {
				hclog.L().Error(fmt.Sprintf("can't get events due: %s", lei.Err))
				break
			}

			lnomadLastIndex = lei.Index

			for _, le := range lei.Events {
				switch le.Topic {
				case nomad.TopicEvaluation:
					lEval, lerr := le.Evaluation()
					if lerr != nil {
						hclog.L().Error(fmt.Sprintf("can't get evals due: %s", lerr))
						continue
					}

					if hclog.L().GetLevel() == hclog.Trace {
						hclog.L().Trace(fmt.Sprintf("eval: %s, status: %s", lEval.ID, lEval.Status))
					}
					lstateStat.IncAcceptedEvals()
					evalCh <- lEval

				case nomad.TopicNode:
					lNode, lerr := le.Node()
					if lerr != nil {
						hclog.L().Error(fmt.Sprintf("can't get nodes due: %s", lerr))
						continue
					}

					if hclog.L().GetLevel() == hclog.Trace {
						hclog.L().Trace(fmt.Sprintf("node event type: %s, node status: %s", le.Type, lNode.Status))
					}
					nodeCh <- lNode

				case nomad.TopicAllocation:
					lAlloc, lerr := le.Allocation()
					if lerr != nil {
						hclog.L().Error(fmt.Sprintf("can't get alloc due: %s", lerr))
						continue
					}

					lstateStat.IncAcceptedAllocs()
					allocCh <- lAlloc
				}
			}
		}

		streamcancelFn()
		hclog.L().Warn("Event stream from nomad was closed for some reason...")
	}
}
