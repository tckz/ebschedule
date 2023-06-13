package ebschedule

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/goccy/go-yaml"
	"github.com/kayac/go-config"
	"github.com/spf13/cobra"
)

const (
	OptSchedule            = "schedule"
	OptCreateScheduleGroup = "create-schedule-group"
)

type CommandInput struct {
	AppName         string
	Version         string
	SchedulerClient SchedulerClient
	OutWriter       io.Writer
}

func NewCommand(in *CommandInput) *cobra.Command {
	root := wrapCobra(&cobra.Command{
		Use:           in.AppName,
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
			fmt.Fprintf(in.OutWriter, "%s\n", in.Version)
			return nil
		},
	}, func(cmd *cobra.Command) {
		root.AddCommand(cmd)
	})

	root.AddCommand(newUpdateCommand(in))
	root.AddCommand(newDiffCommand(in))

	return root
}

func outputResultAsYAML(out any, w io.Writer) error {
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
