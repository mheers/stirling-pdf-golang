# stirling-pdf-golang

A Go SDK and command-line client for the [Stirling PDF Processing API](https://registry.scalar.com/@stirlingpdf/apis/stirling-pdf-processing-api/).
It targets both Stirling Cloud (`https://api.stirling.com`) and self-hosted
Stirling PDF servers.

- **Complete API coverage.** All 76 documented processing operations, from
  merging and splitting to OCR, redaction, signing, and pipelines.
- **Typed and streaming.** Every operation has an idiomatic Go method. Documents
  are streamed as multipart uploads and downloads, never buffered twice.
- **Generated from OpenAPI.** Request and response models come from
  `oapi-codegen`; the checked-in spec keeps the SDK reproducible.
- **Production plumbing included.** API keys, base URLs with path prefixes,
  timeouts, context cancellation, retries with `Retry-After`, and structured API
  errors.
- **A real CLI.** Every operation is available as a Cobra subcommand, plus a
  generic escape hatch for new or self-hosted endpoints.
- **Tested against a real server.** Unit tests run on `httptest`; end-to-end
  tests start Stirling PDF in a container with Testcontainers.

## Requirements

- Go 1.25 or newer
- Docker for the end-to-end tests

## Install

```bash
go get github.com/mheers/stirling-pdf-golang
```

The CLI can be built from a checkout or installed directly:

```bash
go build -o bin/stirling-pdf ./cmd/stirling-pdf
# or
go install github.com/mheers/stirling-pdf-golang/cmd/stirling-pdf@latest
```

## SDK quick start

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/mheers/stirling-pdf-golang/stirling"
)

func main() {
	client, err := stirling.New(
		stirling.WithBaseURL("http://localhost:8080"),
	)
	if err != nil {
		log.Fatal(err)
	}

	input, err := stirling.FromPath("invoice.pdf")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	compressed, err := client.Compress(ctx, input, &stirling.CompressOptions{
		OptimizeLevel:      stirling.Ptr(5),
		ExpectedOutputSize: "100KB",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer compressed.Close()

	if err := compressed.Save("invoice-small.pdf"); err != nil {
		log.Fatal(err)
	}
}
```

For Stirling Cloud, pass an API key. `NewFromEnv` reads `STIRLING_URL`,
`STIRLING_API_KEY`, `STIRLING_TIMEOUT`, and `STIRLING_RETRIES`:

```go
client, err := stirling.NewFromEnv()
```

Merging several documents:

```go
merged, err := client.Merge(ctx, []stirling.FileInput{
	stirling.FromBytes("cover.pdf", cover),
	stirling.FromBytes("body.pdf", body),
}, &stirling.MergeOptions{
	SortType:    stirling.SortOrderProvided,
	GenerateTOC: stirling.Ptr(true),
})
if err != nil {
	return err
}
defer merged.Close()
return merged.Save("report.pdf")
```

## CLI quick start

```bash
# Check the server
stirling-pdf --url http://localhost:8080 status

# Merge, compress, rotate, split
stirling-pdf --url http://localhost:8080 general merge-pdfs \
  --file cover.pdf --file body.pdf -o report.pdf

stirling-pdf --url http://localhost:8080 misc compress-pdf \
  --file report.pdf --optimize-level 5 -o report-small.pdf

# JSON analysis goes to stdout
stirling-pdf --url http://localhost:8080 analysis page-count --file report.pdf

# Filters print true or false and exit 0
stirling-pdf --url http://localhost:8080 filter filter-page-count \
  --file report.pdf --comparator Greater --page-count 5

# Read from stdin, write to stdout
cat document.pdf | stirling-pdf --url http://localhost:8080 convert pdf-markdown --file - -o -
```

The CLI defaults to `https://api.stirling.com`. Set `STIRLING_URL` and
`STIRLING_API_KEY` instead of repeating flags:

```bash
export STIRLING_URL=http://localhost:8080
export STIRLING_API_KEY=...
stirling-pdf status
stirling-pdf misc repair --file broken.pdf -o repaired.pdf
```

Run `stirling-pdf list` for the full command catalog and
`stirling-pdf <group> <command> --help` for flags.

## Authentication

| Server | Header | Notes |
|---|---|---|
| Stirling Cloud | `X-API-KEY` | Required. Create a key in Settings → Preferences → API Keys. |
| Self-hosted, login on | `X-API-KEY` | Create a key in Settings → Preferences → API Keys. A global key can be configured with `security.customGlobalAPIKey`. |
| Self-hosted, login off | none | Leave the API key empty. |

Use `stirling.WithAPIKey` or `--api-key`; the SDK never logs the key.

## What is covered

| Group | Operations |
|---|---|
| General | merge, split by pages/chapters/sections/size, rotate, crop, scale, remove pages/images, rearrange, overlay, booklet, table of contents |
| Convert | PDF ↔ Word, text, Markdown, HTML, CSV, XML, images, PDF/A, EML, office files, URLs |
| Security | add/remove password, watermark, sanitize, redact, sign, validate signature, certificate removal |
| Misc | compress, repair, OCR, flatten, stamp, page numbers, attachments, images, metadata, scanner effect |
| Analysis | page count, dimensions, fonts, form fields, annotations, security, document properties |
| Filter | page size, rotation, count, file size, contains text/image |
| Pipeline | run multiple operations in one request |

The generated reference is in [docs/operations.md](docs/operations.md). The
OpenAPI document is vendored at `api/openapi.json`.

## Documentation

- [SDK guide](docs/sdk.md) — clients, inputs, results, errors, retries, filters,
  pipelines, and the generic escape hatch.
- [CLI guide](docs/cli.md) — groups, flags, output behavior, scripting.
- [Operations](docs/operations.md) — generated command and path catalog.
- [Development](docs/development.md) — code generation, tests, Testcontainers,
  releases.

## Development

```bash
make test          # unit tests
make test-race     # unit tests with the race detector
make e2e           # end-to-end tests in a real container
make generate      # regenerate the OpenAPI code and command registry
make lint          # gofmt and go vet
make build         # build bin/stirling-pdf
```

The end-to-end suite starts `stirlingtools/stirling-pdf:3.3.0` unless
`STIRLING_PDF_IMAGE` points elsewhere. To run it against an already running
server instead:

```bash
STIRLING_E2E_URL=http://localhost:8080 go test -tags=e2e ./e2e -v
```

## Project layout

```
api/                     generated oapi-codegen models and low-level client
  openapi.json           vendored Stirling PDF Processing API document
stirling/                typed SDK
cmd/stirling-pdf/        Cobra CLI
internal/ops/            generated operation registry used by the CLI
internal/testutil/       PDF and container helpers for tests
e2e/                     end-to-end tests against a real server
hack/genregistry/        registry generator
docs/                    user and developer documentation
```

## License

MIT. Copyright (c) 2026 Marcel Heers &lt;marcel@heers.it&gt;.
