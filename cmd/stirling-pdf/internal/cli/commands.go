package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mheers/stirling-pdf-golang/internal/ops"
	"github.com/mheers/stirling-pdf-golang/stirling"
	"github.com/spf13/cobra"
)

func groupOrder() []string {
	var order []string
	seen := map[string]bool{}
	for _, operation := range ops.All {
		if !seen[operation.Group] {
			seen[operation.Group] = true
			order = append(order, operation.Group)
		}
	}
	return order
}

func operationsByGroup(group string) []ops.Operation {
	var operations []ops.Operation
	for _, operation := range ops.All {
		if operation.Group == group {
			operations = append(operations, operation)
		}
	}
	return operations
}

func groupDescription(group string) string {
	switch group {
	case "General":
		return "Merge, split, rotate, crop, scale, and rearrange pages"
	case "Convert":
		return "Convert PDFs to and from other formats"
	case "Security":
		return "Protect, redact, sign, sanitize, and watermark PDFs"
	case "Misc":
		return "Compress, repair, OCR, stamp, and other utilities"
	case "Analysis":
		return "Inspect PDF structure and metadata as JSON"
	case "Filter":
		return "Test PDF properties and print true or false"
	case "Pipeline":
		return "Run several operations in one request"
	}
	return group
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func newStatusCommand(opts *options) *cobra.Command {
	var asJSON bool
	command := &cobra.Command{
		Use:           "status",
		Short:         "Check that a Stirling PDF server is reachable",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := opts.client()
			if err != nil {
				return err
			}
			status, err := client.Status(cmd.Context())
			if err != nil {
				return err
			}
			if asJSON {
				body, err := json.MarshalIndent(status, "", "  ")
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), string(body))
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", status.Version, status.Status)
			return err
		},
	}
	command.Flags().BoolVar(&asJSON, "json", false, "print the raw JSON response")
	return command
}

func newListCommand() *cobra.Command {
	var group string
	var asJSON bool
	command := &cobra.Command{
		Use:           "list",
		Short:         "List available operations",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			type entry struct {
				Group   string `json:"group"`
				Command string `json:"command"`
				Path    string `json:"path"`
				Summary string `json:"summary"`
			}
			var entries []entry
			for _, operation := range ops.All {
				if group != "" && !strings.EqualFold(group, operation.Group) {
					continue
				}
				entries = append(entries, entry{
					Group:   operation.Group,
					Command: strings.ToLower(operation.Group) + " " + operation.Name,
					Path:    operation.Path,
					Summary: operation.Summary,
				})
			}
			if len(entries) == 0 {
				return fmt.Errorf("unknown group %q", group)
			}
			if asJSON {
				body, err := json.MarshalIndent(entries, "", "  ")
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), string(body))
				return err
			}
			current := ""
			for _, entry := range entries {
				if entry.Group != current {
					current = entry.Group
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\n", current); err != nil {
						return err
					}
				}
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  %-34s %s\n", entry.Command, entry.Summary); err != nil {
					return err
				}
			}
			return nil
		},
	}
	command.Flags().StringVar(&group, "group", "", "only list this group")
	command.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return command
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:           "version",
		Short:         "Print the version",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "stirling-pdf %s\n", stirling.Version)
			return err
		},
	}
}
