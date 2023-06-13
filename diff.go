package ebschedule

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/fatih/color"
	"github.com/goccy/go-yaml"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

func newDiffCommand(in *CommandInput) *cobra.Command {
	return wrapCobra(&cobra.Command{
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

			curSch, err := in.SchedulerClient.GetSchedule(ctx, &scheduler.GetScheduleInput{
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
			fmt.Fprint(in.OutWriter, coloredDiff(fmt.Sprint(gotextdiff.ToUnified(fromName, fn, fromYAML, edits))))
			return nil
		},
	}, func(cmd *cobra.Command) {
		cmd.Flags().String(OptSchedule, "", "path/to/schedule.yaml")
		lo.Must0(cmd.MarkFlagRequired(OptSchedule))
	})
}

func normalizeJSON(js []byte) ([]byte, error) {
	var v any
	dec := json.NewDecoder(bytes.NewReader(js))
	dec.UseNumber()
	err := dec.Decode(&v)
	if err != nil {
		return nil, fmt.Errorf("json.Decode: %w", err)
	}

	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetIndent("", "  ")
	err = enc.Encode(v)
	if err != nil {
		return nil, fmt.Errorf("json.Encode: %w", err)
	}
	return buf.Bytes(), nil
}

func marshalYAMLForDiff(src any) (string, error) {
	// yaml.Marshal which compliant with encoding/yaml with types without yaml tag such as GetScheduleOutput outputs keys as lowercase.
	// To avoid it, we marshal it to JSON and decode it again.
	js, err := json.Marshal(src)
	if err != nil {
		return "", fmt.Errorf("json.Marshal: %w", err)
	}

	dec := json.NewDecoder(bytes.NewReader(js))
	dec.UseNumber()
	var v any
	err = dec.Decode(&v)
	if err != nil {
		return "", fmt.Errorf("json.Decode: %w", err)
	}

	for _, p := range []string{
		"/Arn",
		"/CreationDate",
		"/ClientToken",
		"/LastModificationDate",
		"/ResultMetadata",
	} {
		v, _, err = removeValue(v, p)
		if err != nil {
			return "", fmt.Errorf("removeValue(%s): %w", p, err)
		}
	}

	// Try to normalize Target.Input for diff
	var input *string
	const targetInput = "/Target/Input"
	found, err := getValue(v, targetInput, &input)
	if err != nil {
		return "", fmt.Errorf("getValue(%s): %w", targetInput, err)
	}
	if found && input != nil {
		js, err := normalizeJSON([]byte(*input))
		if err == nil {
			err := setValue(v, targetInput, string(js))
			if err != nil {
				return "", fmt.Errorf("setValue(%s): %w", targetInput, err)
			}
		}
	}

	out, err := yaml.MarshalWithOptions(v, yaml.UseLiteralStyleIfMultiline(true))
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// https://github.com/kayac/ecspresso/blob/v2/diff.go
func coloredDiff(src string) string {
	var b strings.Builder
	for _, line := range strings.Split(src, "\n") {
		if strings.HasPrefix(line, "-") {
			b.WriteString(color.RedString(line) + "\n")
		} else if strings.HasPrefix(line, "+") {
			b.WriteString(color.GreenString(line) + "\n")
		} else {
			b.WriteString(line + "\n")
		}
	}
	return b.String()
}
