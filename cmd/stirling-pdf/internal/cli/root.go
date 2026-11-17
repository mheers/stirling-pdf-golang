package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/mheers/stirling-pdf-golang/stirling"
	"github.com/spf13/cobra"
)

// options are the global command-line settings shared by all commands.
type options struct {
	baseURL string
	apiKey  string
	timeout time.Duration
	retries int
	verbose bool
}

func newRootCommand() *cobra.Command {
	opts := &options{}
	root := &cobra.Command{
		Use:   "stirling-pdf",
		Short: "Process PDFs with a Stirling PDF server",
		Long: `stirling-pdf is a command-line client for the Stirling PDF Processing API.

It can talk to Stirling Cloud (https://api.stirling.com) or to a self-hosted
server. Set STIRLING_URL and STIRLING_API_KEY, or pass --url and --api-key.
Every processing operation is available below a group such as general, convert,
security, misc, analysis, or filter. Run "stirling-pdf list" for a compact
catalog and "stirling-pdf <group> <command> --help" for flags.`,
		Example: `  # Compress a PDF against a local server
  stirling-pdf --url http://localhost:8080 misc compress-pdf --file invoice.pdf --output invoice-small.pdf

  # Merge two documents from Stirling Cloud
  STIRLING_API_KEY=... stirling-pdf general merge-pdfs --file a.pdf --file b.pdf -o merged.pdf

  # Inspect a document as JSON
  stirling-pdf analysis page-count --file report.pdf

  # Read a PDF from stdin and write the result to stdout
  cat report.pdf | stirling-pdf convert pdf-markdown --file - -o -`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&opts.baseURL, "url", envOr("STIRLING_URL", stirling.DefaultBaseURL), "Stirling PDF server URL")
	root.PersistentFlags().StringVar(&opts.apiKey, "api-key", os.Getenv("STIRLING_API_KEY"), "API key sent as X-API-KEY")
	root.PersistentFlags().DurationVar(&opts.timeout, "timeout", envDuration("STIRLING_TIMEOUT", 5*time.Minute), "overall request timeout")
	root.PersistentFlags().IntVar(&opts.retries, "retries", envInt("STIRLING_RETRIES", 2), "retries for network and 5xx errors")
	root.PersistentFlags().BoolVarP(&opts.verbose, "verbose", "v", false, "log requests to stderr")

	for _, group := range groupOrder() {
		root.AddCommand(newGroupCommand(group, opts))
	}
	root.AddCommand(newStatusCommand(opts))
	root.AddCommand(newListCommand())
	root.AddCommand(newVersionCommand())
	return root
}

func newGroupCommand(group string, opts *options) *cobra.Command {
	command := &cobra.Command{
		Use:           lowerFirst(group),
		Short:         groupDescription(group),
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	for _, operation := range operationsByGroup(group) {
		command.AddCommand(newOperationCommand(operation, opts))
	}
	return command
}

func (o *options) client() (*stirling.Client, error) {
	clientOptions := []stirling.Option{
		stirling.WithBaseURL(o.baseURL),
		stirling.WithTimeout(o.timeout),
		stirling.WithRetries(o.retries),
	}
	if o.apiKey != "" {
		clientOptions = append(clientOptions, stirling.WithAPIKey(o.apiKey))
	}
	if o.verbose {
		clientOptions = append(clientOptions, stirling.WithRequestEditor(func(r *http.Request) error {
			fmt.Fprintf(os.Stderr, "> %s %s\n", r.Method, r.URL)
			return nil
		}))
	}
	return stirling.New(clientOptions...)
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	if errors.Is(err, errCanceled) || errors.Is(err, context.Canceled) {
		return 130
	}
	return 1
}
