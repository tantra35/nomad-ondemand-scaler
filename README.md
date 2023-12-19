# Hashicorp Nomad ondemand cluster scaler

## Purpose
The cluster scaler is used to automatically adjust the cluster size to the existing workload according to the specified rules

In his work, he uses an abstraction such as pools of nodes, which is a set of instances (nodes) combined by one or more parameters (these can be attributes, resources or devices available on pool instances). Pools are unique relative to each other and should not overlap

Using pools allows you to more granularly allocate resources for workload, for example, it makes no sense to allocate instances with gpu for loads that do not require gpu, etc.