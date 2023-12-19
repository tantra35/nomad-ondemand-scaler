package main

import (
	"fmt"
	"math/rand"
	"reflect"
	"time"

	"github.com/hashicorp/go-hclog"
	nomad "github.com/hashicorp/nomad/api"
	nomadapi "github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/jinzhu/copier"
)

func containsInSlice[T comparable](slice []T, element T) bool {
	for _, item := range slice {
		if reflect.DeepEqual(item, element) {
			return true
		}
	}
	return false
}

func addIfNotExists[T comparable](list []T, elem T) []T {
	for _, e := range list {
		if e == elem {
			return list
		}
	}

	list = append(list, elem)

	return list
}

func Min[T int | float64 | int32 | int64](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func apiNomadJobToStructsJobV2(aj *nomadapi.Job) *structs.Job {
	var sj structs.Job
	copier.Copy(&sj, aj)

	sj.Canonicalize()
	return &sj
}

func apiNomadNodeToStructsNode(an *nomadapi.Node) *structs.Node {
	var sn structs.Node
	copier.Copy(&sn, an)

	sn.Canonicalize()
	return &sn
}

func structsNomadNodeToApiNode(sn *structs.Node) *nomadapi.Node {
	var sa nomadapi.Node
	copier.Copy(&sa, sn)

	return &sa
}

func apiNomadAllocToStructsAlloc(aa *nomadapi.Allocation) *structs.Allocation {
	var sa structs.Allocation
	copier.Copy(&sa, aa)

	sa.Canonicalize()
	return &sa
}

func variantToTypes(_v Variant) interface{} {
	var retval interface{}

	switch _v.GetType() {
	case VariantTypeInt:
		retval = *_v.GetIntValue()
	case VariantTypeString:
		retval = *_v.GetStringValue()
	case VariantTypeSlice:
		lvs := _v.GetSliceValue()
		retvals := make([]interface{}, 0, len(lvs))
		for _, lv := range lvs {
			retvals = append(retvals, variantToTypes(lv))
		}
		retval = retvals
	case VariantTypeMap:
		lvm := _v.GetMapValue()
		retvalm := make(map[string]interface{})
		for k, v := range lvm {
			retvalm[k] = variantToTypes(v)
		}

		retval = retvalm
	}

	return retval
}

func getJobInfoFromEvalWithRetry(_stalecnf *StaleApiConfig, _logger hclog.Logger, _nc *nomad.Client, _eval *nomad.Evaluation) *nomad.Job {
	var jobInfo *nomad.Job
	lnqoptions := nomad.QueryOptions{Namespace: _eval.Namespace, AllowStale: _stalecnf.Allow}

	for {
		var lerr error
		var lmeta *nomad.QueryMeta

		jobInfo, lmeta, lerr = _nc.Jobs().Info(_eval.JobID, &lnqoptions)
		if lerr == nil {
			if _stalecnf.Allow {
				if lmeta.LastContact > _stalecnf.StaleAllowedDuration {
					_logger.Warn(fmt.Sprintf("too stale responce: %s, retrying, as fully consistent", lmeta.LastContact))
					lnqoptions.AllowStale = false

					continue
				}
			}

			break
		}

		_logger.Error(fmt.Sprintf("can't get allocations for job: %s/%s due: %s", _eval.Namespace, _eval.JobID, lerr))
		time.Sleep(10 * time.Second)
	}

	jobInfo.Canonicalize()
	return jobInfo
}

func getJobAllocationsFromEvalWithRetry(_stalecnf *StaleApiConfig, _logger hclog.Logger, _nc *nomad.Client, _eval *nomad.Evaluation) []*nomad.AllocationListStub {
	var allocs []*nomad.AllocationListStub
	lnqoptions := nomad.QueryOptions{Namespace: _eval.Namespace, AllowStale: _stalecnf.Allow}

	for {
		var lerr error
		var lmeta *nomad.QueryMeta

		allocs, lmeta, lerr = _nc.Jobs().Allocations(_eval.JobID, false, &lnqoptions)
		if lerr == nil {
			if _stalecnf.Allow {
				if lmeta.LastContact > _stalecnf.StaleAllowedDuration {
					_logger.Warn(fmt.Sprintf("too stale responce: %s, retrying, as fully consistent", lmeta.LastContact))
					lnqoptions.AllowStale = false

					continue
				}
			}

			break
		}

		_logger.Error(fmt.Sprintf("can't get allocations for job: %s/%s due: %s", _eval.Namespace, _eval.JobID, lerr))
		time.Sleep(10 * time.Second)
	}

	return allocs
}

func getJobEvalsFromEvalWithRetry(_stalecnf *StaleApiConfig, _logger hclog.Logger, _nc *nomad.Client, _eval *nomad.Evaluation) []*nomad.Evaluation {
	var evals []*nomad.Evaluation
	lnqoptions := nomad.QueryOptions{Namespace: _eval.Namespace, AllowStale: _stalecnf.Allow}

	for {
		var lerr error
		var lmeta *nomad.QueryMeta

		evals, lmeta, lerr = _nc.Jobs().Evaluations(_eval.JobID, &lnqoptions)
		if lerr == nil {
			if _stalecnf.Allow {
				if lmeta.LastContact > _stalecnf.StaleAllowedDuration {
					_logger.Warn(fmt.Sprintf("too stale responce: %s, retrying, as fully consistent", lmeta.LastContact))
					lnqoptions.AllowStale = false

					continue
				}
			}

			break
		}

		_logger.Error(fmt.Sprintf("can't get allocations for job: %s/%s due: %s", _eval.Namespace, _eval.JobID, lerr))
		time.Sleep(10 * time.Second)
	}

	return evals
}

func getJobSummaryWithRetry(_stalecnf *StaleApiConfig, _logger hclog.Logger, _nc *nomad.Client, _eval *nomad.Evaluation) *nomad.JobSummary {
	var jobsummry *nomad.JobSummary
	lnqoptions := &nomad.QueryOptions{Namespace: _eval.Namespace, AllowStale: _stalecnf.Allow}

	for {
		var lerr error
		var lmeta *nomad.QueryMeta

		jobsummry, lmeta, lerr = _nc.Jobs().Summary(_eval.JobID, lnqoptions)
		if lerr == nil {
			if _stalecnf.Allow {
				if lmeta.LastContact > _stalecnf.StaleAllowedDuration {
					_logger.Warn(fmt.Sprintf("too stale responce: %s, retrying, as fully consistent", lmeta.LastContact))
					lnqoptions.AllowStale = false

					continue
				}
			}

			break
		}

		_logger.Error(fmt.Sprintf("can't get allocations for job: %s/%s due: %s", _eval.Namespace, _eval.JobID, lerr))
		time.Sleep(10 * time.Second)
	}

	return jobsummry
}

func GetRandomElementOfMap[K comparable, V any](m map[K]V) V {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	randomIndex := rand.Intn(len(keys))
	randomKey := keys[randomIndex]

	return m[randomKey]
}

func AbbreviateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}

	halfLength := (maxLength - 3) / 2
	abbreviated := fmt.Sprintf("%s...%s", s[:halfLength], s[len(s)-halfLength:])
	return abbreviated
}
