# Hashicorp Nomad ondemand cluster scaler

## Purpose
The cluster scaler is used to automatically adjust the cluster size to the existing workload according to the specified rules. 

The need for this project arose due to the lack of such scaling in the original [`nomad-autoscaler`](https://github.com/hashicorp/nomad-autoscaler)

In his work, he uses an abstraction such as pools of nodes, which is a set of instances (nodes) combined by one or more parameters (these can be attributes, resources or devices available on pool instances). Pools are unique relative to each other and should not overlap

Using pools allows you to more granularly allocate resources for workload, for example, it makes no sense to allocate instances with gpu for loads that do not require gpu, etc.

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
