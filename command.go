package ebschedule

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/goccy/go-yaml"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/kayac/go-config"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

const (
	OptSchedule            = "schedule"
	OptCreateScheduleGroup = "create-schedule-group"
)

func NewCommand(myName, version string, schedulerClient SchedulerClient, outWriter io.Writer) *cobra.Command {
	root := wrapCobra(&cobra.Command{
		Use:           myName,
		Short:         "update/diff schedule of Amazon EventBridge Scheduler",
		SilenceErrors: true,
		SilenceUsage:  true,
	}, func(cmd *cobra.Command) {
		cmd.SetOut(os.Stderr)
	})

	wrapCobra(&cobra.Command{
		Use:   "version",
		Short: "Show version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(version)
			return nil
		},
	}, func(cmd *cobra.Command) {
		root.AddCommand(cmd)
	})

	wrapCobra(&cobra.Command{
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

			_, err = schedulerClient.GetScheduleGroup(ctx, &scheduler.GetScheduleGroupInput{
				Name: sch.GroupName,
			})
			if err != nil {
				var notFound *types.ResourceNotFoundException
				if !errors.As(err, &notFound) || !optCreateScheduleGroup {
					return fmt.Errorf("scheduler.GetScheduleGroup: %w", err)
				}
				log.Printf("ScheduleGroup %s does not exist, try to create", *sch.GroupName)
				out, err := schedulerClient.CreateScheduleGroup(ctx, &scheduler.CreateScheduleGroupInput{
					Name: sch.GroupName,
				})
				if err != nil {
					return fmt.Errorf("scheduler.CreateScheduleGroup: %w", err)
				}
				_ = outputResult(out, outWriter)
			}

			var out any
			_, err = schedulerClient.GetSchedule(ctx, &scheduler.GetScheduleInput{
				Name:      sch.Name,
				GroupName: sch.GroupName,
			})
			if err != nil {
				var notFound *types.ResourceNotFoundException
				if !errors.As(err, &notFound) {
					return fmt.Errorf("scheduler.GetSchedule: %w", err)
				}

				out, err = schedulerClient.CreateSchedule(ctx, sch)
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

				out, err = schedulerClient.UpdateSchedule(ctx, &updateInput)
				if err != nil {
					return err
				}
			}
			_ = outputResult(out, outWriter)
			return nil
		},
	}, func(cmd *cobra.Command) {
		cmd.Flags().String(OptSchedule, "", "path/to/schedule.yaml")
		lo.Must0(cmd.MarkFlagRequired(OptSchedule))
		cmd.Flags().Bool(OptCreateScheduleGroup, true, "create schedule group if not exist")
		root.AddCommand(cmd)
	})

	wrapCobra(&cobra.Command{
		Use:   "diff",
		Short: "Diff schedule configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			fn := cmd.Flag(OptSchedule).Value.String()

			sch, err := prepareInputSchedule(fn)
			if err != nil {
				return fmt.Errorf("prepareInputSchedule: %w", err)
			}

			fromYAML := ""
			fromName := "/dev/null"

			curSch, err := schedulerClient.GetSchedule(ctx, &scheduler.GetScheduleInput{
				Name:      sch.Name,
				GroupName: sch.GroupName,
			})
			if err != nil {
				var notFound *types.ResourceNotFoundException
				if !errors.As(err, &notFound) {
					return fmt.Errorf("scheduler.GetSchedule: %w", err)
				}
			} else {
				fromYAML, err = marshalYAMLForDiff(&curSch)
				if err != nil {
					return fmt.Errorf("marshalYAMLForDiff.currentSchedule: %w", err)
				}
				fromName = *curSch.Arn
			}

			toYAML, err := marshalYAMLForDiff(&sch)
			if err != nil {
				return fmt.Errorf("marshalYAMLForDiff.specifiedSchedule: %w", err)
			}

			edits := myers.ComputeEdits(span.URIFromPath(fromName), fromYAML, toYAML)
			fmt.Fprint(outWriter, coloredDiff(fmt.Sprint(gotextdiff.ToUnified(fromName, fn, fromYAML, edits))))
			return nil
		},
	}, func(cmd *cobra.Command) {
		cmd.Flags().String(OptSchedule, "", "path/to/schedule.yaml")
		lo.Must0(cmd.MarkFlagRequired(OptSchedule))
		root.AddCommand(cmd)
	})

	return root
}

func outputResult(out any, w io.Writer) error {
	b, err := marshalYAML(out)
	if err != nil {
		return err
	}
	fmt.Fprintln(w, "---")
	fmt.Fprint(w, string(b))
	return nil
}

func marshalYAML(s any) ([]byte, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	b, err = yaml.JSONToYAML(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func prepareInputSchedule(fn string) (*scheduler.CreateScheduleInput, error) {
	b, err := config.ReadWithEnv(fn)
	if err != nil {
		return nil, err
	}

	var sch scheduler.CreateScheduleInput
	if err := unmarshalYAML(b, &sch); err != nil {
		return nil, fmt.Errorf("unmarshalYAML: %w", err)
	}
	if sch.GroupName == nil {
		sch.GroupName = aws.String("default")
	}
	if sch.Name == nil {
		return nil, fmt.Errorf("Name must be specified")
	}

	return &sch, nil
}

func unmarshalYAML(b []byte, out any) error {
	// yaml.Unmarshal which compliant with encoding/yaml with types without yaml tag such as CreateScheduleInput assumes all keys are lowercase.
	// It results there is no matches yaml key and fields of the type.
	// To avoid it, we unmarshal from JSON.
	js, err := yaml.YAMLToJSON(b)
	if err != nil {
		return err
	}

	return json.Unmarshal(js, out)
}

func wrapCobra(cmd *cobra.Command, f func(*cobra.Command)) *cobra.Command {
	f(cmd)
	return cmd
}
