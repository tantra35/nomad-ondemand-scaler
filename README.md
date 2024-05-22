# Hashicorp nomad ondemand horizontal cluster autoscaler

## Purpose
The nomad-ondemand-scaler is used to automatically adjust the cluster size when cluster size not sufficient to satisfy workload requirements

The need for this project arose due to the lack of such scaling in the original [`nomad-autoscaler`](https://github.com/hashicorp/nomad-autoscaler)

This scaler monitor blocked [`evals`](https://developer.hashicorp.com/nomad/docs/v1.4.x/concepts/architecture#evaluation), and if it detect this, begin scaling action(selects the most suitable pool, calculate required amount of nodes to place required workload)

As most of autoscalers this project also, have such abstraction as pools of nodes - which is a set of instances (nodes) combined by one or more parameters (these can be attributes, resources or devices available on pool instances). Pools are unique relative to each other and should not overlap

Using pools allows you to more granularly allocate resources for workload, for example, it makes no sense to allocate instances with gpu for loads that do not require gpu, etc.



## Config
```
poolconfig="./pools.yml" # <-- Setup location of yaml file, that describes node pools

gc {
  cicles_to_gc = 3
  cicle_period = "1m"
  allowed_freexpr = "min(round(totalnodes * 0.1), 2)"
}

stalenomadapi {
  allow = true
  duration = "30ms"
}

telemetry {
  statsiteaddr = "statesitelocal.service.consul:8125"
  prefix = "telemetry stats prefix"
}

hungprevention {
 allow = true
 detect_period = "30m"
}
```

Config consist from 4 sections:
  * <h6><code id="login-optional-fields">gc</code></h6> describes garbage collection:
  * `cicles_to_gc` how many GC cycles instance must exist in idle state(without allocations) before it will be garbage collected
  * `cicle_period` periodically of GC cycle(should be specified in form that understands [ParseDuration](https://pkg.go.dev/time#ParseDuration) function)
  * `allowed_freexpr` expression that understands [exprtk](http://www.partow.net/programming/exprtk/). This expression defines allowed free nodes count in each pool (instances that will not garbage collected)) this is usefull to organize Hot pools
  **Important** in expression can be used predefined variables:
    * `totalnodes` - total nodes in pool
    * `busynodes` - busy nodes in pool

* `stalenomadapi` allow use [_inconsistent nomad api_](https://developer.hashicorp.com/nomad/api-docs#consistency-modes)
  * `allow` allow using inconsistent nomad api(true|false)
  * `duration` max allowed interval of inconsistency, within which response from nomad api will be considered as valid (should be specified in form that understands [ParseDuration](https://pkg.go.dev/time#ParseDuration) function), in other case request will be repeated with fully consistent requirements

* `telemetry` - allow telemetry, now supports only statsite, but due library https://github.com/hashicorp/go-metrics used, no any problems to add other collectors, for example Prometheus, Datadog etc

* `hungprevention` - describes parameters that prevents scale action hung(they can be caused by errors in the code of the scaler itself, as well as external reasons - for example, the cloud provider cannot allocate the requested resources)
  * `allow` - prohibits or not setting of a global timeout for scaleup actions (default: `false`)
  * `detect_period` - timeout for scaleup action(should be specified in form that understands [ParseDuration](https://pkg.go.dev/time#ParseDuration) function)


## Pool configuration
Pool configuration is a yaml file, something like this: 
```yaml
- datacenter: <some datacenter>
  nodeclass: <node class> 
  cpu: 1000
  mem: 25Gib
  reserved:
    cpu: 100
    mem: 100Mib
  drivers:
    - docker
    - exec
  devices:
    - name: "NVIDIA A10G"
      type: gpu
      vendor: nvidia
      attr:
        memory: "23028 MiB"
  attr.cpu.arch: x86
  attr.kernel.name: linux
  provider:
    name: anynode
```

In such config `provider` field describe `node provider`, which is used to create pool instances, for now 3 types of providers are supported:
  * [`anynode`](./provider.anynode.md)
  * [`awsautoscale`](./provider.awsautoscale.md)
  * [`karpenter`](./provider.karpenter.md)
