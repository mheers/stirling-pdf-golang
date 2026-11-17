# Development

## Requirements

- Go 1.25+
- Docker (only for the end-to-end suite)
- Python 3 (only for `hack/update-spec.sh`)
- `oapi-codegen` is downloaded on demand by `go generate`; no global install is
  required

## Make targets

```bash
make generate      # regenerate the OpenAPI code and the CLI operation registry
make test          # unit tests
make test-race     # unit tests with the race detector
make e2e           # end-to-end tests against a Testcontainers server
make lint          # gofmt check and go vet
make build         # build bin/stirling-pdf
make cover         # coverage report in coverage.html
```

## Code generation

Two generated layers exist:

1. **`api/types.gen.go` and `api/client.gen.go`** — produced by
   `oapi-codegen v2.5.1` from `api/openapi.json`.
2. **`internal/ops/registry.gen.go` and `docs/operations.md`** — produced by
   `hack/genregistry` from the same spec.

Run both:

```bash
make generate
```

Do not edit `*.gen.go` or `docs/operations.md` by hand. CI checks that
generated files are current:

```bash
make check-generate
```

### Updating the OpenAPI document

`hack/update-spec.sh` fetches the current document from the Scalar registry and
regenerates everything:

```bash
./hack/update-spec.sh
go test ./...
```

The vendored document is the source of truth for this repository. Pin a review
of the generated diff whenever it changes: operation paths, form field names,
and requiredness can drift between Stirling PDF releases and the cloud API.

## Tests

### Unit tests

```bash
go test ./...
```

The SDK tests use `httptest` and cover:

- file inputs, reopenability, and multipart encoding
- API error parsing and status helpers
- result streaming, filenames, and content types
- retry behavior, `Retry-After`, and one-shot input safety
- every typed operation's path and key form fields (`operations_test.go`)
- CLI flags, output behavior, filters, and aliases

Tests are white-box (`package stirling`) where they need to exercise the
multipart encoder and retry loop directly.

### End-to-end tests

The `e2e` package is behind the `e2e` build tag. It starts
`stirlingtools/stirling-pdf:3.3.0` with Testcontainers, waits for
`/api/v1/info/status`, and runs SDK and CLI scenarios against it:

```bash
make e2e
# or
go test -tags=e2e ./e2e -v
```

Useful environment variables:

| Variable | Purpose |
|---|---|
| `STIRLING_PDF_IMAGE` | Use another Stirling PDF image tag. |
| `STIRLING_E2E_URL` | Skip Testcontainers and test an existing server. |

```bash
STIRLING_E2E_URL=http://localhost:8080 go test -tags=e2e ./e2e -run TestMergeAndAnalyze -v
```

The suite covers status, merge and page analysis, ZIP splitting, compression,
rotation and cropping, scanner effects, manual redaction, sanitization,
image/text conversion, watermarking, page numbers, encryption and decryption,
filters, pipelines, and the built CLI (including stdin/stdout).

## Adding an operation

1. Update `api/openapi.json` (preferably with `hack/update-spec.sh`) and run
   `make generate`.
2. Add a typed method and an options struct in the matching `stirling/*.go`
   file. Follow the existing pattern: `inputParams`, pointer options, explicit
   form field names.
3. Add a case to `TestOperationPathsAndFields` in
   `stirling/operations_test.go` asserting the path, form values, and uploaded
   filenames.
4. Add an end-to-end test if the operation is central or has unusual output.
5. Add the method to the reference table in `docs/sdk.md`. The CLI command is
   generated automatically and appears in `docs/operations.md`.

The CLI needs no per-operation code: it reads `internal/ops` and builds flags,
typed values, files, and output handling from the registry.

## Style

- `gofmt` is mandatory; `make lint` checks it.
- `go vet ./...` must pass.
- Exported identifiers carry doc comments, because the SDK is consumed through
  `go doc`.
- Keep user-facing errors lowercase and prefixed with `stirling:` where the
  error originates in the SDK.
- Do not log API keys. The CLI never prints them, even with `--verbose`.

## Release

The module path is `github.com/mheers/stirling-pdf-golang`. Version tags use
semantic versioning. The SDK version defaults to `0.1.0` and can be injected at
build time:

```bash
go build -ldflags "-X github.com/mheers/stirling-pdf-golang/stirling.Version=v0.2.0" ./cmd/stirling-pdf
```

Publishing the GitHub repository and the module proxy is manual: the repository
is public at `github.com/mheers/stirling-pdf-golang`, and version tags are pushed
by the maintainer.

## Continuous integration

`.github/workflows/ci.yml` runs:

1. `gofmt` check and `go vet`
2. a check that the generated operation registry and docs are current
3. unit tests with the race detector
4. the `e2e` suite with Docker (GitHub-hosted runners provide it)
5. a CLI build and `--help` smoke test

The e2e job pulls the Stirling PDF image, which dominates its runtime. Cache
Docker layers or point `STIRLING_PDF_IMAGE` at a mirror if that becomes a
problem.
