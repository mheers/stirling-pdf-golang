package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mheers/stirling-pdf-golang/internal/ops"
	"github.com/mheers/stirling-pdf-golang/stirling"
	"github.com/spf13/cobra"
)

func writeResult(cmd *cobra.Command, operation ops.Operation, result *stirling.Result) error {
	defer func() { _ = result.Close() }()

	output, _ := cmd.Flags().GetString("output")
	force, _ := cmd.Flags().GetBool("force")
	toStdout, _ := cmd.Flags().GetBool("stdout")

	if operation.Filter {
		matched := !result.IsEmpty()
		if !matched {
			fmt.Fprintln(cmd.OutOrStdout(), false)
			return nil
		}
		if toStdout || output == "-" {
			return writeBinary(cmd, operation, result, output, toStdout, force)
		}
		fmt.Fprintln(cmd.OutOrStdout(), true)
		if output == "" {
			return nil
		}
		return writeBinary(cmd, operation, result, output, toStdout, force)
	}

	if result.IsEmpty() {
		return nil
	}

	switch operation.Response {
	case "json":
		body, err := result.Bytes()
		if err != nil {
			return err
		}
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, body, "", "  "); err != nil {
			pretty.Write(body)
		}
		pretty.WriteByte('\n')
		return writeData(cmd, output, force, pretty.Bytes())
	case "text":
		body, err := result.Bytes()
		if err != nil {
			return err
		}
		return writeData(cmd, output, force, body)
	default:
		return writeBinary(cmd, operation, result, output, toStdout, force)
	}
}

func writeBinary(cmd *cobra.Command, operation ops.Operation, result *stirling.Result, output string, toStdout, force bool) error {
	if output != "" && toStdout {
		return errors.New("--output and --stdout cannot be used together")
	}
	if toStdout || output == "-" {
		writer := cmd.OutOrStdout()
		if _, err := result.SaveTo(writer); err != nil {
			return err
		}
		return nil
	}
	if output == "" {
		if writer, ok := cmd.OutOrStdout().(*os.File); !ok || !isTerminal(writer) {
			_, err := result.SaveTo(cmd.OutOrStdout())
			return err
		}
		output = defaultFilename(operation, result)
	}
	if info, err := os.Stat(output); err == nil && info.IsDir() {
		output = filepath.Join(output, defaultFilename(operation, result))
	}
	if !force {
		if _, err := os.Stat(output); err == nil {
			return fmt.Errorf("%s already exists (use --force to overwrite)", output)
		}
	}
	if err := result.Save(output); err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), output)
	return nil
}

func writeData(cmd *cobra.Command, output string, force bool, data []byte) error {
	if output == "" || output == "-" {
		_, err := cmd.OutOrStdout().Write(data)
		return err
	}
	if !force {
		if _, err := os.Stat(output); err == nil {
			return fmt.Errorf("%s already exists (use --force to overwrite)", output)
		}
	}
	if err := os.WriteFile(output, data, 0o644); err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), output)
	return nil
}

func defaultFilename(operation ops.Operation, result *stirling.Result) string {
	if name := result.Filename(); name != "" {
		return filepath.Base(name)
	}
	extension := result.Extension()
	if extension == "" {
		extension = ".bin"
	}
	return operation.Name + extension
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func parseObjectList(raw string) ([]map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var objects []map[string]any
	if err := json.Unmarshal([]byte(raw), &objects); err != nil {
		return nil, fmt.Errorf("expected a JSON array of objects: %w", err)
	}
	return objects, nil
}

func registerOutputFlags(command *cobra.Command) {
	flags := command.Flags()
	flags.StringP("output", "o", "", "write the result to this file ('-' for stdout)")
	flags.Bool("stdout", false, "write the result to stdout")
	flags.Bool("force", false, "overwrite an existing output file")
}
