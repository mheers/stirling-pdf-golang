package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Execute runs the command line against ctx and returns a process exit code.
func Execute(ctx context.Context) int {
	root := newRootCommand()
	root.SetContext(ctx)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return exitCode(err)
	}
	return 0
}

// NewRootCommand exposes the command tree for tests and embedders.
func NewRootCommand() *cobra.Command { return newRootCommand() }
