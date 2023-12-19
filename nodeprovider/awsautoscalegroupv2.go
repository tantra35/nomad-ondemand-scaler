package nodeprovider

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"github.com/hashicorp/go-hclog"
	nomad "github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/nomad/structs"
)

type AwsAutoscaleGroupProvider struct {
	lock           sync.Mutex
	logger         hclog.Logger
	asgName        string
	lastUpdatetime time.Time
	asgClient      *autoscaling.Client
	state          map[string]bool
}

func updateStateFromAws(_ctx context.Context, _svc *autoscaling.Client, _asgName string, _state map[string]bool) (int32, error) {
	var nextToken *string
	var desiredCapacity int32

	for {
		describeASGOutput, lerr := _svc.DescribeAutoScalingGroups(_ctx, &autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []string{_asgName},
			NextToken:             nextToken,
		})
		if lerr != nil {
			return -1, fmt.Errorf("can't update autoscale group %s state due: %s", _asgName, lerr)
		}
		if len(describeASGOutput.AutoScalingGroups) == 0 {
			return -1, fmt.Errorf("now such autoscale group %s", _asgName)
		}

		lasg := describeASGOutput.AutoScalingGroups[0]

		for _, instanceAsg := range lasg.Instances {
			if *instanceAsg.HealthStatus == "Unhealthy" {
				delete(_state, *instanceAsg.InstanceId)
				continue
			}

			_state[*instanceAsg.InstanceId] = false
		}

		nextToken = describeASGOutput.NextToken
		if nextToken == nil {
			desiredCapacity = *lasg.DesiredCapacity
			break
		}
	}

	return desiredCapacity, nil
}

func NewAwsAutoscaleGroupProvider(_asgName string) (INodeProvider, error) {
	cfg, _ := config.LoadDefaultConfig(context.TODO())
	svc := autoscaling.NewFromConfig(cfg)

	lstate := map[string]bool{}
	_, lerr := updateStateFromAws(context.TODO(), svc, _asgName, lstate)
	if lerr != nil {
		return nil, fmt.Errorf("can't create aws autoscale provider due: %s", lerr)
	}

	return &AwsAutoscaleGroupProvider{
		logger:         hclog.L().Named("AwsAutoscaleGroupProvider").With("asg", _asgName),
		asgName:        _asgName,
		asgClient:      svc,
		state:          lstate,
		lastUpdatetime: time.Now(),
	}, nil
}

func (c *AwsAutoscaleGroupProvider) IsNodeExists(_nomadNode *nomad.Node) bool {
	lloger := c.logger.Named("IsNodeExists")
	llastregisterevnt := GetLastRegisterEvent(_nomadNode.Events)
	lastTime := llastregisterevnt.Timestamp
	instanceId := _nomadNode.Attributes["unique.platform.aws.instance-id"]
	lNodeExists := false

	c.lock.Lock()
	if lastTime.After(c.lastUpdatetime) {
		c.lock.Unlock()
		updatetime := time.Now()
		newstate := map[string]bool{}

		for {
			_, lerr := updateStateFromAws(context.TODO(), c.asgClient, c.asgName, newstate)
			if lerr == nil {
				lloger.Debug(fmt.Sprintf("successed updated state when check insatnceId: %s(nomadnodeid: %s)", instanceId, _nomadNode.ID))
				break
			}

			lloger.Error(fmt.Sprintf("can't update state due: %s", lerr))
			time.Sleep(10 * time.Second)
		}

		c.lock.Lock()
		for linstanceid := range newstate {
			if seenbypool, lok := c.state[linstanceid]; lok {
				newstate[linstanceid] = seenbypool
			}
		}
		c.state = newstate
		c.lastUpdatetime = updatetime
	}

	_, lNodeExists = c.state[instanceId]
	if lNodeExists {
		c.state[instanceId] = true
	}
	c.lock.Unlock()

	return lNodeExists
}

func extractInstanceIDs(input string) []string {
	instanceIDs := []string{}

	// Проверяем, содержит ли входная строка фразу "The instances"
	if regexMatch := regexp.MustCompile(`The instance(?:s{0,1}) (.+) (?:are|is) not part of Auto Scaling group`).FindStringSubmatch(input); len(regexMatch) > 0 {
		// Извлекаем список инстансов из регулярного выражения
		instanceStr := regexMatch[1]

		// Извлекаем идентификаторы инстансов
		re := regexp.MustCompile(`i-[0-9a-fA-F]+`)
		instanceIDs = re.FindAllString(instanceStr, -1)
	}

	return instanceIDs
}

func (c *AwsAutoscaleGroupProvider) _removeNode(lloger hclog.Logger, instncesIds []string) error {
	cfg, _ := config.LoadDefaultConfig(context.TODO())
	ec2Client := ec2.NewFromConfig(cfg)

	// https://docs.aws.amazon.com/cli/latest/reference/autoscaling/detach-instances.html#options
	// Датач экземпляра из Auto Scaling группы и затем их терминирование, далать это нужно батчами по 20, см ссылку выше(detach принимает не большее 20 идентификаторов)
	for i := 0; i < len(instncesIds); i += 20 {
		instncesIdsBatch := instncesIds[i:Min(i+20, len(instncesIds))]

		// Датач экземпляра из Auto Scaling группы
		for {
			_, lerr := c.asgClient.DetachInstances(context.TODO(), &autoscaling.DetachInstancesInput{
				InstanceIds:                    instncesIdsBatch,
				AutoScalingGroupName:           aws.String(c.asgName),
				ShouldDecrementDesiredCapacity: aws.Bool(true),
			})
			if lerr == nil {
				break
			}

			// https://github.com/aws/aws-sdk-go-v2/blob/main/CHANGELOG.md#error-handling
			var validationErr *smithy.GenericAPIError
			if errors.As(lerr, &validationErr) {
				if validationErr.ErrorCode() == "ValidationError" {
					notexistentInstances := extractInstanceIDs(validationErr.ErrorMessage())
					instncesIdsBatch = RemoveSliceElements(instncesIdsBatch, notexistentInstances)
				}
			}

			if len(instncesIdsBatch) == 0 {
				break
			}

			lloger.Error(fmt.Sprintf("failed to detach instance from Auto Scaling group: %v", lerr))
			time.Sleep(10 * time.Second)
		}

		if len(instncesIdsBatch) == 0 {
			continue
		}

		// Ожидание завершения открепления экземпляров
		var newstate map[string]bool
		for {
			updatetime := time.Now()
			newstate = map[string]bool{}
			_, lerr := updateStateFromAws(context.TODO(), c.asgClient, c.asgName, newstate)
			if lerr != nil {
				lloger.Error(fmt.Sprintf("can't update state due: %s", lerr))
				time.Sleep(10 * time.Second)

				continue
			}

			var someInstancesExists bool
			for _, instanceId := range instncesIds {
				if _, lok := newstate[instanceId]; lok {
					someInstancesExists = true
					break
				}
			}

			if !someInstancesExists {
				c.lock.Lock()

				for linstanceid := range newstate {
					if seenbypool, lok := c.state[linstanceid]; lok {
						newstate[linstanceid] = seenbypool
					}
				}

				c.state = newstate
				c.lastUpdatetime = updatetime

				c.lock.Unlock()

				break
			}

			time.Sleep(5 * time.Second)
		}

		// терминирование инстансов
		for {
			_, lerr := ec2Client.TerminateInstances(context.TODO(), &ec2.TerminateInstancesInput{
				InstanceIds: instncesIdsBatch,
			})
			if lerr == nil {
				break
			}

			lloger.Error(fmt.Sprintf("failed to terminate instance due: %s", lerr))
			time.Sleep(10 * time.Second)
		}

		// ожидание завершения терминирования
		for {
			resp, lerr := ec2Client.DescribeInstances(context.TODO(), &ec2.DescribeInstancesInput{
				InstanceIds: instncesIdsBatch,
			})
			if lerr != nil {
				lloger.Error(fmt.Sprintf("failed to describe instances due: %s", lerr))
				time.Sleep(10 * time.Second)
				continue
			}

			allTerminated := true
			for _, reservation := range resp.Reservations {
				for _, instance := range reservation.Instances {
					if instance.State.Name != ec2types.InstanceStateNameTerminated {
						allTerminated = false
						break
					}
				}
			}

			if allTerminated {
				break
			}

			time.Sleep(10 * time.Second)
		}
	}

	return nil
}

func (c *AwsAutoscaleGroupProvider) RemoveNode(_nomadNodes []*nomad.Node) error {
	lloger := c.logger.Named("RemoveNode")
	var instncesIds []string

	for _, lnode := range _nomadNodes {
		instanceId := lnode.Attributes["unique.platform.aws.instance-id"]
		instncesIds = append(instncesIds, instanceId)
	}

	return c._removeNode(lloger, instncesIds)
}

func (c *AwsAutoscaleGroupProvider) UpdateNode(_ctx context.Context, _nodes []*structs.Node, _totalcount int32) error {
	lloger := c.logger.Named("UpdateNode")

	c.lock.Lock()

	lnodesinpool := map[string]struct{}{}
	for _, lnode := range _nodes {
		instanceId := lnode.Attributes["unique.platform.aws.instance-id"]
		lnodesinpool[instanceId] = struct{}{}
	}

	instancetoremove := []string{}
	for instanceId, seenbypool := range c.state {
		if seenbypool {
			if _, lok := lnodesinpool[instanceId]; !lok {
				instancetoremove = append(instancetoremove, instanceId)
			}
		}
	}

	lmynodescount := len(c.state)
	c.lock.Unlock()

	//обнаружили рассинхрон, сначала пытаемся просто обновить стейт
	if len(instancetoremove) > 0 {
		lloger.Warn(fmt.Sprintf("pool reported about differense in nodes %d(my) -> %d(pool oppinion), so, remove unexisten: %v", lmynodescount, len(_nodes), instancetoremove))
		c._removeNode(lloger, instancetoremove)
	}

	for {
		_, lerr := c.asgClient.UpdateAutoScalingGroup(_ctx, &autoscaling.UpdateAutoScalingGroupInput{
			AutoScalingGroupName: aws.String(c.asgName),
			DesiredCapacity:      aws.Int32(_totalcount),
		})
		if lerr == nil {
			lloger.Info(fmt.Sprintf("Set asg disiresize size to: %d", _totalcount))
			break
		}

		if _ctx.Err() != nil {
			return _ctx.Err()
		}

		lloger.Error(fmt.Sprintf("can't set asg disiresize size to: %d due: %s", _totalcount, lerr))
		time.Sleep(10 * time.Second)
	}

	return nil
}
