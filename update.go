package ebschedule

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

func newUpdateCommand(in *CommandInput) *cobra.Command {
	return wrapCobra(&cobra.Command{
		Use:   "update",
		Short: "Update or create schedule",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			fn := cmd.Flag(OptSchedule).Value.String()
			optCreateScheduleGroup, _ := cmd.Flags().GetBool(OptCreateScheduleGroup)

			sch, err := prepareInputSchedule(fn)
			if err != nil {
				return fmt.Errorf("prepareInputSchedule: %w", err)
			}

			_, err = in.SchedulerClient.GetScheduleGroup(ctx, &scheduler.GetScheduleGroupInput{
				Name: sch.GroupName,
			})
			if err != nil {
				var notFound *types.ResourceNotFoundException
				if !errors.As(err, &notFound) || !optCreateScheduleGroup {
					return fmt.Errorf("scheduler.GetScheduleGroup: %w", err)
				}
				log.Printf("ScheduleGroup %s does not exist, try to create", *sch.GroupName)
				out, err := in.SchedulerClient.CreateScheduleGroup(ctx, &scheduler.CreateScheduleGroupInput{
					Name: sch.GroupName,
				})
				if err != nil {
					return fmt.Errorf("scheduler.CreateScheduleGroup: %w", err)
				}
				_ = outputResultAsYAML(out, in.OutWriter)
			}

			var out any
			_, err = in.SchedulerClient.GetSchedule(ctx, &scheduler.GetScheduleInput{
				Name:      sch.Name,
				GroupName: sch.GroupName,
			})
			if err != nil {
				var notFound *types.ResourceNotFoundException
				if !errors.As(err, &notFound) {
					return fmt.Errorf("scheduler.GetSchedule: %w", err)
				}

				out, err = in.SchedulerClient.CreateSchedule(ctx, sch)
				if err != nil {
					return err
				}
			} else {
				b, err := json.Marshal(sch)
				if err != nil {
					return err
				}
				var updateInput scheduler.UpdateScheduleInput
				err = json.Unmarshal(b, &updateInput)
				if err != nil {
					return err
				}

				out, err = in.SchedulerClient.UpdateSchedule(ctx, &updateInput)
				if err != nil {
					return err
				}
			}
			_ = outputResultAsYAML(out, in.OutWriter)
			return nil
		},
	}, func(cmd *cobra.Command) {
		cmd.Flags().String(OptSchedule, "", "path/to/schedule.yaml")
		lo.Must0(cmd.MarkFlagRequired(OptSchedule))
		cmd.Flags().Bool(OptCreateScheduleGroup, true, "create schedule group if not exist")
	})
}
