package nodeprovider

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
)

func TestNewAwsAutoscaleGroupProvider(t *testing.T) {
	// vault read secrets/aws/plr/atf01/creds/full
	os.Setenv("AWS_ACCESS_KEY_ID", "insert here")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "insert here")
	os.Setenv("AWS_DEFAULT_REGION", "eu-central-1")

	provider, lerr := NewAwsAutoscaleGroupProvider("dockerworker-bastion-t3.medium")
	if lerr != nil {
		t.Errorf("can't create aws autoscale provider due: %s", lerr)
		t.FailNow()
	}

	t.Logf("%v", provider)
}

func TestNewAwsAutoscaleGroupProviderSetNodeCount(t *testing.T) {
	// vault read secrets/aws/plr/atf01/creds/full
	os.Setenv("AWS_ACCESS_KEY_ID", "insert here")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "insert here")
	os.Setenv("AWS_DEFAULT_REGION", "eu-central-1")

	provider, lerr := NewAwsAutoscaleGroupProvider("dockerworker-bastion-t3.medium")
	if lerr != nil {
		t.Errorf("can't create aws autoscale provider due: %s", lerr)
		t.FailNow()
	}

	lerr = provider.UpdateNode(context.TODO(), nil, 2)
	if lerr != nil {
		t.Errorf("can't update due: %s", lerr)
		t.FailNow()
	}
}

func TestAwsApiDescribeInstances(t *testing.T) {
	// vault read secrets/aws/plr/atf01/creds/full
	os.Setenv("AWS_ACCESS_KEY_ID", "insert here")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "insert here")
	os.Setenv("AWS_DEFAULT_REGION", "eu-central-1")

	cfg, _ := config.LoadDefaultConfig(context.TODO())
	ec2Client := ec2.NewFromConfig(cfg)

	instncesIdsBatch := []string{
		"i-003da42a905996f7d",
		"i-0909319a27d4bc626",
		"i-087170dc77ad8bef9",
		"i-0861e046a309adacf", //not terminated
		"i-0861e046a409adacf", //not existent
	}

	resp, lerr := ec2Client.DescribeInstances(context.TODO(), &ec2.DescribeInstancesInput{
		InstanceIds: instncesIdsBatch,
	})
	if lerr != nil {
		t.Logf("failed to describe instances due: %s", lerr)
		t.FailNow()
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

	if !allTerminated {
		t.Logf("not all instances terminated")
		t.FailNow()
	}
}

func TestAwsDetachInvalidInstance(t *testing.T) {
	// vault read secrets/aws/plr/atf01/creds/full
	os.Setenv("AWS_ACCESS_KEY_ID", "insert here")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "insert here")
	os.Setenv("AWS_DEFAULT_REGION", "eu-central-1")

	cfg, _ := config.LoadDefaultConfig(context.TODO())
	asgClient := autoscaling.NewFromConfig(cfg)

	instncesIdsBatch := []string{"i-0fb5fd0e762dda7ae", "i-0fb5fd0e762dd37ae"}
	_, lerr := asgClient.DetachInstances(context.TODO(), &autoscaling.DetachInstancesInput{
		InstanceIds:                    instncesIdsBatch,
		AutoScalingGroupName:           aws.String("stablediffusion-artist-g5.4xlarge"),
		ShouldDecrementDesiredCapacity: aws.Bool(true),
	})

	t.Logf("%v", lerr)

	var validationErr *smithy.GenericAPIError
	if errors.As(lerr, &validationErr) {
		if validationErr.ErrorCode() == "ValidationError" {
			notexistentInstances := extractInstanceIDs(validationErr.ErrorMessage())
			if !CompareSlices(notexistentInstances, []string{"i-0fb5fd0e762dda7ae", "i-0fb5fd0e762dd37ae"}) {
				t.Fatalf("wrong instnces ids: %v", notexistentInstances)
			}

			instncesIdsBatch = RemoveSliceElements(instncesIdsBatch, notexistentInstances)
			if len(instncesIdsBatch) != 0 {
				t.Fatalf("wrong instances after remove, must be nil, but got: %v", instncesIdsBatch)
			}
		}
	}
}
