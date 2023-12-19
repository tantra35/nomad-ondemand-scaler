# Node provider awsautoscale
## Описание
Manages the creation of nodes through management [`aws autoscale group`](https://docs.aws.amazon.com/autoscaling/ec2/userguide/auto-scaling-groups.html)

## Configuration

```
provider:
  name: awsautoscale
  params:
    - "some-aws-autoscale-t3.medium" 
```

Receives only one parameter for input, which specifies the name of the `aws autoscale group`