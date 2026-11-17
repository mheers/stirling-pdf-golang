// Command stirling-pdf is a Cobra command-line client for the Stirling PDF
// Processing API. It mirrors the stirling SDK and exposes every documented
// operation as a subcommand.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/mheers/stirling-pdf-golang/cmd/stirling-pdf/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(cli.Execute(ctx))
}
