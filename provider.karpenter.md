# Node provider karpenter
## Synopsis
Manages node creation using part of the k8s scaler [karpenter](https://karpenter.sh/), it only works with the aws cloud. [Aditional plugin](https://github.com/tantra35/nomad-ondemand-scaler-karpenter-plugin) required for correct work.

## Configuration
Sample usage:
```
provider:
  name: karpenter
  params:
    name: "sparkdriver"
    freqPerCpuCore: 2500
    launchtemplate: "launchtemplate-t3.2xlarge"
    subnets:
      "Name": "routed-us-east-1b,routed-us-east-2b"
    reqs:
      - Key: "node.kubernetes.io/instance-type"
        Op: "In"
        Values: ["t3a.2xlarge", "t3.2xlarge"]

      - Key: "karpenter.sh/capacity-type"
        Op: "In"
        Values: ["on-demand"]
 ```

* `name` - a very important parameter, it sets the name that will be used when labeling instances in aws, so it is important that the name is unique for each pool, those must match 1 to 1 (1 scaler pool = 1 unique name), if this condition is not met, instances will not be able to spread correctly across pools, which will lead to incorrect operation the scaler. It was not possible to come up with a reliable way to persistently generate a name from a description, which would make this parameter redundant.

* `freqPerCpuCore` - sets the frequency of one instance core, it is necessary in order to calculate the number of instance cores based on the cpu shares in the pool description(a simple formula is used: cpu/freqPerCpuCore), it is used to smooth out the differences between how nomad and k8s cpu provides(nomad operates cpu shares(this is an abstract parameter and it is calculated by nomad `<cpu cores count> * <cpu core freq>`), and k8s operates on the number of cores available on the instance(`<cpu cores count>`))

* `launchtemplate` - name of [`launch template`](https://docs.aws.amazon.com/autoscaling/ec2/userguide/launch-templates.html ), which will be used when creating the instance. If you pass it, then krapenter will not try to create its own custom launch template. The launch template must specify:
  * [`ami`](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AMIs.html) which will be used when creating nodes
  * [`security group`](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-security-groups.html), and although they can be specified in the provider's `karpenter` parameters, the SG in the `launch template` is not overridden (this is not implemented in the karpenter code), perhaps this will be fixed in subsequent versions of carpenter
  * [`instance profile`](https://docs.aws.amazon.com/managedservices/latest/userguide/defaults-instance-profile.html)- it is also not overridden, it is only used to create a new `launch template`

* `profile` - name of [`instance profile`](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-metadata.html#instancedata-iam), it is used only when creating a custom `launch template` and does not override the `instance profile` in the launch template, which is passed in the `launchtemplate` parameter

* `securitygroups` - list of [`security groups`](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-security-groups.html), which will be used when creating a custom `launch template` and does not override the `security groups` in the launch template that is passed in the `launchtemplate` parameter

* `subnets` - subnets in which nodes will be created, understands both [`теги`](https://docs.aws.amazon.com/tag-editor/latest/userguide/tagging.html)(in sample `Name` is a tag), and a special name `aws-ids` - позволяющее задать список идентификаторов подсетей. Allows to set a list of subnet IDs. If you need to specify several values, they are separated by commas, for example: `Name: "routed-us-east-1b,routed-us-east-2b"`(будут отобраны подсети у которых тег `Name` равен `routed-us-east-1b` или `routed-us-east-2b`)

* `reqs` - the list of requirements for nodes is set as a list and individual requirements are combined according to the rule `AND(&)`
  * `Key` - it can take one of the following values:
    ```
    "node.kubernetes.io/instance-type" <-- instance type, for example t3.medium
    "karpenter.k8s.aws/instance-cpu"
    "karpenter.k8s.aws/instance-network-bandwidth"
    "karpenter.k8s.aws/instance-gpu-name"
    "karpenter.k8s.aws/instance-gpu-count"
    "karpenter.k8s.aws/instance-gpu-memory"
    "karpenter.k8s.aws/instance-accelerator-name"
    "karpenter.k8s.aws/instance-hypervisor"
    "karpenter.k8s.aws/instance-encryption-in-transit-support"
    "kubernetes.io/arch"
    "topology.kubernetes.io/region"
    "karpenter.k8s.aws/instance-memory"
    "karpenter.k8s.aws/instance-generation"
    "karpenter.k8s.aws/instance-local-nvme"
    "karpenter.k8s.aws/instance-size"
    "karpenter.k8s.aws/instance-accelerator-manufacturer"
    "topology.kubernetes.io/zone"
    "node.kubernetes.io/windows-build"
    "karpenter.k8s.aws/instance-category"
    "kubernetes.io/os"
    "karpenter.sh/capacity-type"
    "karpenter.k8s.aws/instance-pods"
    "karpenter.k8s.aws/instance-family"  <---  instance family, for example t3
    "karpenter.k8s.aws/instance-gpu-manufacturer"
    "karpenter.k8s.aws/instance-accelerator-count"
    ```
  * `OP` Takes one of the following values (https://github.com/kubernetes/api/blob/v0.25.4/core/v1/types.go#L2781-L2788):
      * `In`
      * `NotIn`
      * `Exists`
      * `DoesNotExist`
      * `Gt`
      * `Lt`
  * `Values` - values depends on `Key`
