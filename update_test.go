package ebschedule

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	mock_ebschedule "github.com/tckz/ebschedule/mock"
)

var _ gomock.Matcher = (*CmpDiffMatcher)(nil)
var _ gomock.GotFormatter = (*CmpDiffMatcher)(nil)

type CmpDiffMatcher struct {
	expected interface{}
	opts     []cmp.Option
}

func (m CmpDiffMatcher) Got(got any) string {
	return cmp.Diff(m.expected, got, m.opts...)
}

func (m CmpDiffMatcher) String() string {
	return fmt.Sprintf("%T%+v", m.expected, m.expected)
}

func (m CmpDiffMatcher) Matches(x any) bool {
	return cmp.Diff(m.expected, x, m.opts...) == ""
}

func CmpDiff(expected any, opts ...cmp.Option) gomock.Matcher {
	return &CmpDiffMatcher{
		opts:     opts,
		expected: expected,
	}
}

func Test_update(t *testing.T) {

	optsIgnoreUnexported := cmpopts.IgnoreUnexported(
		scheduler.CreateScheduleInput{},
		scheduler.UpdateScheduleInput{},
		scheduler.GetScheduleGroupInput{},
		types.FlexibleTimeWindow{},
		types.Target{},
		types.RetryPolicy{},
		types.DeadLetterConfig{},
		types.EcsParameters{},
		types.NetworkConfiguration{},
		types.AwsVpcConfiguration{},
	)

	t.Run("update-sch", func(t *testing.T) {
		assert := assert.New(t)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		out := bytes.NewBuffer(nil)
		cl := mock_ebschedule.NewMockSchedulerClient(ctrl)

		cl.EXPECT().GetScheduleGroup(gomock.Any(),
			&scheduler.GetScheduleGroupInput{Name: aws.String("some-group")}).
			Return(&scheduler.GetScheduleGroupOutput{
				Arn:                  aws.String("arn:aws:scheduler:ap-northeast-1:99999:schedule-group/some-group"),
				CreationDate:         nil,
				LastModificationDate: nil,
				Name:                 aws.String("some-group"),
			}, nil)

		cl.EXPECT().GetSchedule(gomock.Any(), &scheduler.GetScheduleInput{
			Name:      aws.String("some-schedule"),
			GroupName: aws.String("some-group"),
		}).Return(&scheduler.GetScheduleOutput{}, nil)

		cl.EXPECT().UpdateSchedule(gomock.Any(), CmpDiff(&scheduler.UpdateScheduleInput{
			Name: aws.String("some-schedule"),
			FlexibleTimeWindow: &types.FlexibleTimeWindow{
				Mode: types.FlexibleTimeWindowModeOff,
			},
			ScheduleExpression: aws.String("cron(*/3 * * * ? *)"),
			Target: &types.Target{
				Arn:     aws.String("arn:aws:ecs:ap-northeast-1:99999:cluster/some-cluster"),
				RoleArn: aws.String("arn:aws:iam::99999:role/some-scheduler-role"),
				DeadLetterConfig: &types.DeadLetterConfig{
					Arn: aws.String("arn:aws:sqs:ap-northeast-1:99999:some-dlq"),
				},
				EcsParameters: &types.EcsParameters{
					TaskDefinitionArn:    aws.String("arn:aws:ecs:ap-northeast-1:99999:task-definition/some-def"),
					EnableECSManagedTags: aws.Bool(true),
					EnableExecuteCommand: aws.Bool(false),
					LaunchType:           types.LaunchTypeFargate,
					NetworkConfiguration: &types.NetworkConfiguration{
						AwsvpcConfiguration: &types.AwsVpcConfiguration{
							Subnets:        []string{"subnet-xxxxx", "subnet-yyyyy"},
							AssignPublicIp: types.AssignPublicIpEnabled,
							SecurityGroups: []string{"sg-xxxxx"},
						},
					},
					TaskCount: aws.Int32(1),
				},
				Input: aws.String(`{"containerOverrides":[{"name":"hello-task","command":["ya","yo"]}]}
`),
				RetryPolicy: &types.RetryPolicy{
					MaximumEventAgeInSeconds: aws.Int32(600),
					MaximumRetryAttempts:     aws.Int32(2),
				},
			},
			GroupName:                  aws.String("some-group"),
			ScheduleExpressionTimezone: aws.String("Asia/Tokyo"),
			State:                      types.ScheduleStateEnabled,
		}, optsIgnoreUnexported)).
			Return(&scheduler.UpdateScheduleOutput{
				ScheduleArn: aws.String("arn:aws:scheduler:ap-northeast-1:99999:schedule/some-group/some-schedule"),
			}, nil)

		cmd := NewCommand(&CommandInput{
			AppName:         "ut",
			Version:         "v0.0.1",
			SchedulerClient: cl,
			OutWriter:       out,
		})
		ctx := context.Background()
		cmd.SetArgs([]string{"update", "--schedule", "testdata/update/normal.yml"})
		err := cmd.ExecuteContext(ctx)

		assert.NoError(err)
		assert.Equal(`---
ScheduleArn: arn:aws:scheduler:ap-northeast-1:99999:schedule/some-group/some-schedule
ResultMetadata: {}
`, out.String())
	})

	t.Run("create-sch", func(t *testing.T) {
		assert := assert.New(t)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		out := bytes.NewBuffer(nil)
		cl := mock_ebschedule.NewMockSchedulerClient(ctrl)

		cl.EXPECT().GetScheduleGroup(gomock.Any(),
			&scheduler.GetScheduleGroupInput{Name: aws.String("some-group")}).
			Return(&scheduler.GetScheduleGroupOutput{
				Arn:                  aws.String("arn:aws:scheduler:ap-northeast-1:99999:schedule-group/some-group"),
				CreationDate:         nil,
				LastModificationDate: nil,
				Name:                 aws.String("some-group"),
			}, nil)

		cl.EXPECT().GetSchedule(gomock.Any(), &scheduler.GetScheduleInput{
			Name:      aws.String("some-schedule"),
			GroupName: aws.String("some-group"),
		}).Return(nil, &types.ResourceNotFoundException{})

		cl.EXPECT().CreateSchedule(gomock.Any(), CmpDiff(&scheduler.CreateScheduleInput{
			Name: aws.String("some-schedule"),
			FlexibleTimeWindow: &types.FlexibleTimeWindow{
				Mode: types.FlexibleTimeWindowModeOff,
			},
			ScheduleExpression: aws.String("cron(*/3 * * * ? *)"),
			Target: &types.Target{
				Arn:     aws.String("arn:aws:ecs:ap-northeast-1:99999:cluster/some-cluster"),
				RoleArn: aws.String("arn:aws:iam::99999:role/some-scheduler-role"),
				DeadLetterConfig: &types.DeadLetterConfig{
					Arn: aws.String("arn:aws:sqs:ap-northeast-1:99999:some-dlq"),
				},
				EcsParameters: &types.EcsParameters{
					TaskDefinitionArn:    aws.String("arn:aws:ecs:ap-northeast-1:99999:task-definition/some-def"),
					EnableECSManagedTags: aws.Bool(true),
					EnableExecuteCommand: aws.Bool(false),
					LaunchType:           types.LaunchTypeFargate,
					NetworkConfiguration: &types.NetworkConfiguration{
						AwsvpcConfiguration: &types.AwsVpcConfiguration{
							Subnets:        []string{"subnet-xxxxx", "subnet-yyyyy"},
							AssignPublicIp: types.AssignPublicIpEnabled,
							SecurityGroups: []string{"sg-xxxxx"},
						},
					},
					TaskCount: aws.Int32(1),
				},
				Input: aws.String(`{"containerOverrides":[{"name":"hello-task","command":["ya","yo"]}]}
`),
				RetryPolicy: &types.RetryPolicy{
					MaximumEventAgeInSeconds: aws.Int32(600),
					MaximumRetryAttempts:     aws.Int32(2),
				},
			},
			GroupName:                  aws.String("some-group"),
			ScheduleExpressionTimezone: aws.String("Asia/Tokyo"),
			State:                      types.ScheduleStateEnabled,
		}, optsIgnoreUnexported)).
			Return(&scheduler.CreateScheduleOutput{
				ScheduleArn: aws.String("arn:aws:scheduler:ap-northeast-1:99999:schedule/some-group/some-schedule"),
			}, nil)

		cmd := NewCommand(&CommandInput{
			AppName:         "ut",
			Version:         "v0.0.1",
			SchedulerClient: cl,
			OutWriter:       out,
		})
		ctx := context.Background()
		cmd.SetArgs([]string{"update", "--schedule", "testdata/update/normal.yml"})
		err := cmd.ExecuteContext(ctx)

		assert.NoError(err)
		assert.Equal(`---
ScheduleArn: arn:aws:scheduler:ap-northeast-1:99999:schedule/some-group/some-schedule
ResultMetadata: {}
`, out.String())
	})

	t.Run("create-group", func(t *testing.T) {
		assert := assert.New(t)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		out := bytes.NewBuffer(nil)
		cl := mock_ebschedule.NewMockSchedulerClient(ctrl)

		cl.EXPECT().GetScheduleGroup(gomock.Any(),
			&scheduler.GetScheduleGroupInput{Name: aws.String("some-group")}).
			Return(nil, &types.ResourceNotFoundException{})

		cl.EXPECT().CreateScheduleGroup(gomock.Any(), &scheduler.CreateScheduleGroupInput{
			Name: aws.String("some-group"),
		}).
			Return(&scheduler.CreateScheduleGroupOutput{
				ScheduleGroupArn: aws.String("arn:aws:scheduler:ap-northeast-1:99999:schedule-group/some-group"),
			}, nil)

		cl.EXPECT().GetSchedule(gomock.Any(), &scheduler.GetScheduleInput{
			Name:      aws.String("some-schedule"),
			GroupName: aws.String("some-group"),
		}).Return(nil, &types.ResourceNotFoundException{})

		cl.EXPECT().CreateSchedule(gomock.Any(), CmpDiff(&scheduler.CreateScheduleInput{
			Name: aws.String("some-schedule"),
			FlexibleTimeWindow: &types.FlexibleTimeWindow{
				Mode: types.FlexibleTimeWindowModeOff,
			},
			ScheduleExpression: aws.String("cron(*/3 * * * ? *)"),
			Target: &types.Target{
				Arn:     aws.String("arn:aws:ecs:ap-northeast-1:99999:cluster/some-cluster"),
				RoleArn: aws.String("arn:aws:iam::99999:role/some-scheduler-role"),
				DeadLetterConfig: &types.DeadLetterConfig{
					Arn: aws.String("arn:aws:sqs:ap-northeast-1:99999:some-dlq"),
				},
				EcsParameters: &types.EcsParameters{
					TaskDefinitionArn:    aws.String("arn:aws:ecs:ap-northeast-1:99999:task-definition/some-def"),
					EnableECSManagedTags: aws.Bool(true),
					EnableExecuteCommand: aws.Bool(false),
					LaunchType:           types.LaunchTypeFargate,
					NetworkConfiguration: &types.NetworkConfiguration{
						AwsvpcConfiguration: &types.AwsVpcConfiguration{
							Subnets:        []string{"subnet-xxxxx", "subnet-yyyyy"},
							AssignPublicIp: types.AssignPublicIpEnabled,
							SecurityGroups: []string{"sg-xxxxx"},
						},
					},
					TaskCount: aws.Int32(1),
				},
				Input: aws.String(`{"containerOverrides":[{"name":"hello-task","command":["ya","yo"]}]}
`),
				RetryPolicy: &types.RetryPolicy{
					MaximumEventAgeInSeconds: aws.Int32(600),
					MaximumRetryAttempts:     aws.Int32(2),
				},
			},
			GroupName:                  aws.String("some-group"),
			ScheduleExpressionTimezone: aws.String("Asia/Tokyo"),
			State:                      types.ScheduleStateEnabled,
		}, optsIgnoreUnexported)).
			Return(&scheduler.CreateScheduleOutput{
				ScheduleArn: aws.String("arn:aws:scheduler:ap-northeast-1:99999:schedule/some-group/some-schedule"),
			}, nil)

		cmd := NewCommand(&CommandInput{
			AppName:         "ut",
			Version:         "v0.0.1",
			SchedulerClient: cl,
			OutWriter:       out,
		})
		ctx := context.Background()
		cmd.SetArgs([]string{"update", "--schedule", "testdata/update/normal.yml"})
		err := cmd.ExecuteContext(ctx)

		assert.NoError(err)
		assert.Equal(`---
ScheduleGroupArn: arn:aws:scheduler:ap-northeast-1:99999:schedule-group/some-group
ResultMetadata: {}
---
ScheduleArn: arn:aws:scheduler:ap-northeast-1:99999:schedule/some-group/some-schedule
ResultMetadata: {}
`, out.String())
	})

	t.Run("without-create-group", func(t *testing.T) {
		assert := assert.New(t)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		out := bytes.NewBuffer(nil)
		cl := mock_ebschedule.NewMockSchedulerClient(ctrl)

		cl.EXPECT().GetScheduleGroup(gomock.Any(),
			&scheduler.GetScheduleGroupInput{Name: aws.String("some-group")}).
			Return(nil, &types.ResourceNotFoundException{})

		cmd := NewCommand(&CommandInput{
			AppName:         "ut",
			Version:         "v0.0.1",
			SchedulerClient: cl,
			OutWriter:       out,
		})
		ctx := context.Background()
		cmd.SetArgs([]string{"update",
			"--schedule", "testdata/update/normal.yml",
			"--create-schedule-group=false",
		})
		err := cmd.ExecuteContext(ctx)

		assert.EqualError(err, `scheduler.GetScheduleGroup: ResourceNotFoundException: `)
		assert.Equal(``, out.String())
	})

	t.Run("err-wo-name", func(t *testing.T) {
		assert := assert.New(t)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		out := bytes.NewBuffer(nil)
		cl := mock_ebschedule.NewMockSchedulerClient(ctrl)

		cmd := NewCommand(&CommandInput{
			AppName:         "ut",
			Version:         "v0.0.1",
			SchedulerClient: cl,
			OutWriter:       out,
		})
		ctx := context.Background()
		cmd.SetArgs([]string{"update", "--schedule", "testdata/update/err-wo-name.yml"})
		err := cmd.ExecuteContext(ctx)

		assert.EqualError(err, `prepareInputSchedule: Name must be specified`)
		assert.Equal(``, out.String())
	})

	t.Run("err@GetScheduleGroup", func(t *testing.T) {
		assert := assert.New(t)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		out := bytes.NewBuffer(nil)
		cl := mock_ebschedule.NewMockSchedulerClient(ctrl)

		cl.EXPECT().GetScheduleGroup(gomock.Any(),
			CmpDiff(&scheduler.GetScheduleGroupInput{Name: aws.String("default")}, optsIgnoreUnexported)).
			Return(nil, errors.New("err@GetScheduleGroup"))

		cmd := NewCommand(&CommandInput{
			AppName:         "ut",
			Version:         "v0.0.1",
			SchedulerClient: cl,
			OutWriter:       out,
		})
		ctx := context.Background()
		cmd.SetArgs([]string{"update", "--schedule", "testdata/update/omit-group.yml"})
		err := cmd.ExecuteContext(ctx)

		assert.EqualError(err, `scheduler.GetScheduleGroup: err@GetScheduleGroup`)
		assert.Equal(``, out.String())
	})

}
