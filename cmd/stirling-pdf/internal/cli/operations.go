package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mheers/stirling-pdf-golang/internal/ops"
	"github.com/mheers/stirling-pdf-golang/stirling"
	"github.com/spf13/cobra"
)

var errCanceled = errors.New("canceled")

var operationAliases = map[string][]string{
	"merge-pdfs":           {"merge"},
	"rotate-pdf":           {"rotate"},
	"compress-pdf":         {"compress"},
	"split-pages":          {"split"},
	"pdf-img":              {"to-images"},
	"pdf-text":             {"to-text"},
	"pdf-word":             {"to-word"},
	"pdf-markdown":         {"to-markdown"},
	"img-pdf":              {"from-images"},
	"html-pdf":             {"from-html"},
	"markdown-pdf":         {"from-markdown"},
	"add-watermark":        {"watermark"},
	"add-password":         {"encrypt"},
	"remove-password":      {"decrypt"},
	"filter-page-count":    {"page-count-is"},
	"filter-contains-text": {"contains-text"},
}

func newOperationCommand(operation ops.Operation, opts *options) *cobra.Command {
	command := &cobra.Command{
		Use:           operation.Name,
		Aliases:       operationAliases[operation.Name],
		Short:         operation.Summary,
		Long:          operationLong(operation),
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := opts.client()
			if err != nil {
				return err
			}
			params, err := buildParams(cmd, operation)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			result, err := client.Call(ctx, operation.Path, params...)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return fmt.Errorf("%w: %v", errCanceled, err)
				}
				return err
			}
			return writeResult(cmd, operation, result)
		},
	}
	registerFlags(command, operation)
	registerOutputFlags(command)
	return command
}

func registerFlags(command *cobra.Command, operation ops.Operation) {
	flags := command.Flags()
	for _, field := range operation.Fields {
		description := fieldDescription(field)
		switch field.Kind {
		case ops.FileKind:
			flags.String(field.Flag, "", description)
			_ = command.MarkFlagFilename(field.Flag)
		case ops.FileListKind:
			flags.StringArray(field.Flag, nil, description)
			_ = command.MarkFlagFilename(field.Flag)
		case ops.StringKind:
			flags.String(field.Flag, stringDefault(field), description)
		case ops.IntegerKind:
			flags.Int64(field.Flag, intDefault(field), description)
		case ops.NumberKind:
			flags.Float64(field.Flag, floatDefault(field), description)
		case ops.BooleanKind:
			flags.Bool(field.Flag, boolDefault(field), description)
		case ops.StringListKind:
			flags.StringSlice(field.Flag, nil, description)
		case ops.IntegerListKind:
			flags.IntSlice(field.Flag, nil, description)
		case ops.NumberListKind:
			flags.Float64Slice(field.Flag, nil, description)
		case ops.StringMapKind:
			flags.StringArray(field.Flag, nil, description)
		case ops.ObjectListKind:
			flags.String(field.Flag, "", description)
		}
		if len(field.Enum) > 0 {
			values := field.Enum
			_ = command.RegisterFlagCompletionFunc(field.Flag, func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
				return values, cobra.ShellCompDirectiveNoFileComp
			})
		}
		if field.Required && (field.Kind == ops.FileKind || field.Kind == ops.FileListKind) {
			_ = command.MarkFlagRequired(field.Flag)
		}
	}
	flags.StringArray("field", nil, "extra form field as name=value (repeatable)")
	flags.StringArray("file-field", nil, "extra file field as name=path (repeatable)")
}

func buildParams(cmd *cobra.Command, operation ops.Operation) ([]stirling.Param, error) {
	var params []stirling.Param
	for _, field := range operation.Fields {
		if !cmd.Flags().Changed(field.Flag) {
			continue
		}
		param, err := readParam(cmd, field)
		if err != nil {
			return nil, err
		}
		if param != nil {
			params = append(params, *param)
		}
	}

	extraFields, _ := cmd.Flags().GetStringArray("field")
	for _, assignment := range extraFields {
		name, value, ok := strings.Cut(assignment, "=")
		if !ok || name == "" {
			return nil, fmt.Errorf("--field %q: expected name=value", assignment)
		}
		params = append(params, stirling.Param{Name: name, Value: value})
	}
	extraFiles, _ := cmd.Flags().GetStringArray("file-field")
	for _, assignment := range extraFiles {
		name, value, ok := strings.Cut(assignment, "=")
		if !ok || name == "" {
			return nil, fmt.Errorf("--file-field %q: expected name=path", assignment)
		}
		input, err := openInput(cmd, value)
		if err != nil {
			return nil, err
		}
		params = append(params, stirling.Param{Name: name, Value: input})
	}
	return params, nil
}

func readParam(cmd *cobra.Command, field ops.Field) (*stirling.Param, error) {
	flags := cmd.Flags()
	switch field.Kind {
	case ops.StringKind:
		value, _ := flags.GetString(field.Flag)
		return &stirling.Param{Name: field.Name, Value: value}, nil
	case ops.IntegerKind:
		value, _ := flags.GetInt64(field.Flag)
		return &stirling.Param{Name: field.Name, Value: value}, nil
	case ops.NumberKind:
		value, _ := flags.GetFloat64(field.Flag)
		return &stirling.Param{Name: field.Name, Value: value}, nil
	case ops.BooleanKind:
		value, _ := flags.GetBool(field.Flag)
		return &stirling.Param{Name: field.Name, Value: value}, nil
	case ops.StringListKind:
		value, _ := flags.GetStringSlice(field.Flag)
		return &stirling.Param{Name: field.Name, Value: value}, nil
	case ops.IntegerListKind:
		value, _ := flags.GetIntSlice(field.Flag)
		return &stirling.Param{Name: field.Name, Value: value}, nil
	case ops.NumberListKind:
		value, _ := flags.GetFloat64Slice(field.Flag)
		return &stirling.Param{Name: field.Name, Value: value}, nil
	case ops.FileKind:
		value, _ := flags.GetString(field.Flag)
		input, err := openInput(cmd, value)
		if err != nil {
			return nil, err
		}
		return &stirling.Param{Name: field.Name, Value: input}, nil
	case ops.FileListKind:
		values, _ := flags.GetStringArray(field.Flag)
		inputs := make([]stirling.FileInput, 0, len(values))
		for _, value := range values {
			input, err := openInput(cmd, value)
			if err != nil {
				return nil, err
			}
			inputs = append(inputs, input)
		}
		return &stirling.Param{Name: field.Name, Value: inputs}, nil
	case ops.StringMapKind:
		values, _ := flags.GetStringArray(field.Flag)
		mapped := make(map[string]string, len(values))
		for _, assignment := range values {
			key, value, ok := strings.Cut(assignment, "=")
			if !ok || key == "" {
				return nil, fmt.Errorf("--%s %q: expected key=value", field.Flag, assignment)
			}
			mapped[key] = value
		}
		return &stirling.Param{Name: field.Name, Value: mapped}, nil
	case ops.ObjectListKind:
		raw, _ := flags.GetString(field.Flag)
		objects, err := parseObjectList(raw)
		if err != nil {
			return nil, fmt.Errorf("--%s: %w", field.Flag, err)
		}
		return &stirling.Param{Name: field.Name, Value: objects}, nil
	default:
		return nil, fmt.Errorf("unsupported field kind %q", field.Kind)
	}
}

func openInput(cmd *cobra.Command, value string) (stirling.FileInput, error) {
	if value == "-" {
		return stirling.FromReader("stdin", cmd.InOrStdin()), nil
	}
	input, err := stirling.FromPath(value)
	if err != nil {
		return nil, err
	}
	return input, nil
}

func fieldDescription(field ops.Field) string {
	description := field.Description
	if description == "" {
		switch field.Kind {
		case ops.FileKind, ops.FileListKind:
			description = "input file"
		default:
			description = "form field"
		}
	}
	if len(field.Enum) > 0 {
		if description != "" {
			description += " "
		}
		description += "(one of: " + strings.Join(field.Enum, ", ") + ")"
	}
	switch field.Kind {
	case ops.FileKind:
		description += " (path or '-')"
	case ops.FileListKind:
		description += " (repeatable path)"
	case ops.StringListKind, ops.IntegerListKind, ops.NumberListKind:
		description += " (comma-separated or repeatable)"
	case ops.StringMapKind:
		description += " (repeatable key=value)"
	case ops.ObjectListKind:
		description += " (JSON array of objects)"
	}
	return strings.TrimSpace(description)
}

func operationLong(operation ops.Operation) string {
	var builder strings.Builder
	builder.WriteString(operation.Summary)
	builder.WriteString("\n\n")
	builder.WriteString(operation.Path)
	if operation.Description != "" {
		builder.WriteString("\n\n")
		builder.WriteString(operation.Description)
	}
	builder.WriteString("\n\nFields that are not passed are left to the server's defaults. Use --field and --file-field to send parameters that are absent from the OpenAPI document.")
	return builder.String()
}

func stringDefault(field ops.Field) string {
	if !field.HasDefault || strings.HasPrefix(field.Default, "[") {
		return ""
	}
	return field.Default
}

func intDefault(field ops.Field) int64 {
	if !field.HasDefault {
		return 0
	}
	value, _ := strconv.ParseInt(field.Default, 10, 64)
	return value
}

func floatDefault(field ops.Field) float64 {
	if !field.HasDefault {
		return 0
	}
	value, _ := strconv.ParseFloat(field.Default, 64)
	return value
}

func boolDefault(field ops.Field) bool {
	return field.HasDefault && field.Default == "true"
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func envDuration(name string, fallback time.Duration) time.Duration {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
