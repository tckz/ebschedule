//go:generate mockgen -source=$GOFILE -destination=./mock/$GOFILE

package ebschedule

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/scheduler"
)

type SchedulerClient interface {
	GetScheduleGroup(ctx context.Context, params *scheduler.GetScheduleGroupInput, optFns ...func(*scheduler.Options)) (*scheduler.GetScheduleGroupOutput, error)
	CreateScheduleGroup(ctx context.Context, params *scheduler.CreateScheduleGroupInput, optFns ...func(*scheduler.Options)) (*scheduler.CreateScheduleGroupOutput, error)

	GetSchedule(ctx context.Context, params *scheduler.GetScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.GetScheduleOutput, error)
	CreateSchedule(ctx context.Context, params *scheduler.CreateScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.CreateScheduleOutput, error)
	UpdateSchedule(ctx context.Context, params *scheduler.UpdateScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.UpdateScheduleOutput, error)
}
