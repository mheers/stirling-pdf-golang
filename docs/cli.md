# CLI guide

`stirling-pdf` exposes every Processing API operation as a Cobra subcommand. It
uses the same `stirling` SDK as the library, so authentication, retries, and
streaming behave identically.

## Build and install

From a checkout:

```bash
go build -o bin/stirling-pdf ./cmd/stirling-pdf
```

Or install the latest published version:

```bash
go install github.com/mheers/stirling-pdf-golang/cmd/stirling-pdf@latest
```

## Configuration

| Flag | Environment | Default | Purpose |
|---|---|---|---|
| `--url` | `STIRLING_URL` | `https://api.stirling.com` | Server root, including any path prefix. |
| `--api-key` | `STIRLING_API_KEY` | empty | Sent as `X-API-KEY`. |
| `--timeout` | `STIRLING_TIMEOUT` | `5m` | Overall request timeout. |
| `--retries` | `STIRLING_RETRIES` | `2` | Retries for network and 5xx errors. |
| `--verbose`, `-v` | — | off | Print each request to stderr. |

```bash
export STIRLING_URL=http://localhost:8080
export STIRLING_API_KEY=...
stirling-pdf status
```

## Command shape

```
stirling-pdf [global flags] <group> <command> [flags]
```

Groups:

| Group | Purpose |
|---|---|
| `general` | Merge, split, rotate, crop, scale, rearrange, overlay, booklet |
| `convert` | PDF ↔ Word, text, Markdown, HTML, CSV, XML, images, PDF/A, EML |
| `security` | Passwords, watermark, redaction, signing, sanitizing |
| `misc` | Compress, OCR, repair, stamps, page numbers, attachments, metadata |
| `analysis` | Page count, dimensions, fonts, form fields, properties |
| `filter` | Tests that print `true` or `false` |
| `pipeline` | Multi-operation requests |

`stirling-pdf list` prints the catalog; `stirling-pdf list --group convert`
narrows it. Every command has `--help` with endpoint, description, and flags.

Common aliases: `compress` → `misc compress-pdf`, `merge` →
`general merge-pdfs`, `rotate` → `general rotate-pdf`, `split` →
`general split-pages`, `watermark` → `security add-watermark`, `encrypt` /
`decrypt` → add/remove password.

## Files

`--file` is a path. Repeat it for operations that accept several documents:

```bash
stirling-pdf general merge-pdfs --file cover.pdf --file body.pdf -o merged.pdf
```

Use `-` to read from stdin. This is supported wherever a file is accepted:

```bash
cat report.pdf | stirling-pdf convert pdf-text --file - -o -
```

Use `--file-id` for a file that already exists on the server (self-hosted file
storage or cloud):

```bash
stirling-pdf general rotate-pdf --file-id a1b2c3d4... --angle 90 -o rotated.pdf
```

## Output

Output flags are available on every operation:

| Flag | Behavior |
|---|---|
| `-o, --output path` | Write to `path`. |
| `-o -` | Write binary/text/JSON output to stdout. |
| `--stdout` | Same as `-o -`, without a path. |
| `--force` | Overwrite an existing file. Without it, existing files are protected. |

When no output is given:

- **Binary result, stdout is a pipe or file** — bytes go to stdout. This makes
  `stirling-pdf … | zcat` and shell pipelines work.
- **Binary result, stdout is a terminal** — the CLI derives a filename from
  `Content-Disposition` or the operation name, saves it in the current
  directory, and prints the path.
- **JSON result** — pretty-printed JSON goes to stdout.
- **Text result** — raw text goes to stdout.

Examples:

```bash
# Save with an explicit name
stirling-pdf misc compress-pdf --file big.pdf --optimize-level 7 -o small.pdf

# Rotate and pipe the PDF to another tool
stirling-pdf general rotate-pdf --file in.pdf --angle 90 -o - | pdfinfo -

# JSON into jq
stirling-pdf analysis page-count --file report.pdf | jq .pageCount
```

## Filters

Filter commands print `true` or `false` and always exit 0 for a successful
request. A `204 No Content` response means `false`. To keep the matched
document, pass an output:

```bash
stirling-pdf filter filter-page-count --file report.pdf --comparator Greater --page-count 5
# false

stirling-pdf filter filter-contains-text --file report.pdf --text Invoice -o matched.pdf
# true
# matched.pdf contains the original document

stirling-pdf filter filter-page-count --file report.pdf --comparator Greater --page-count 5 -o - | pdfinfo -
# the matched PDF goes to stdout without a boolean line
```

## Passing parameters

Every documented form field is a typed flag. Names are kebab-case versions of
the API fields:

```bash
stirling-pdf security add-watermark --file in.pdf \
  --watermark-type text --watermark-text DRAFT --opacity 0.25 \
  --font-size 48 --rotation 45 -o watermarked.pdf
```

Defaults from the OpenAPI document are shown in `--help`. Flags you do not pass
are omitted, letting the server apply its own defaults.

For endpoints or parameters that are newer than the vendored spec, use the
generic escape hatches:

```bash
stirling-pdf misc repair --file in.pdf \
  --field customKey1=value \
  --file-field customAttachment=notes.txt \
  -o repaired.pdf
```

Bracket notation is supported for maps and object arrays through `--field` if
needed (`--field 'allRequestParams[k]=v'`).

## Examples

```bash
# Merge with a generated table of contents
stirling-pdf general merge-pdfs --file a.pdf --file b.pdf --generate-toc -o merged.pdf

# Split a document into one PDF per chapter
stirling-pdf general split-pdf-by-chapters --file book.pdf -o chapters.zip

# OCR a scan in English and German
stirling-pdf misc ocr-pdf --file scan.pdf --languages eng,deu --ocr-type skip-text -o searchable.pdf

# Redact a list of words
stirling-pdf security auto-redact --file contract.pdf --list-of-text "secret,internal" -o redacted.pdf

# Redact rectangular areas (object arrays are passed as JSON)
stirling-pdf security redact --file contract.pdf \
  --redactions '[{"x":100,"y":200,"width":300,"height":50,"page":1,"color":"#000000"}]' \
  -o redacted.pdf

# Encrypt and later decrypt
stirling-pdf security add-password --file report.pdf --password reader --owner-password owner --key-length 256 -o locked.pdf
stirling-pdf security remove-password --file locked.pdf --password reader -o unlocked.pdf

# Convert to images and to Markdown
stirling-pdf convert pdf-img --file report.pdf --image-format png --single-or-multiple multiple -o pages.zip
stirling-pdf convert pdf-markdown --file report.pdf -o report.md
```

## Status, version, completion

```bash
stirling-pdf --url http://localhost:8080 status
# 3.3.0 UP

stirling-pdf --url http://localhost:8080 status --json
# {"version":"3.3.0","status":"UP"}

stirling-pdf version
stirling-pdf completion bash > /etc/bash_completion.d/stirling-pdf
```

`status` calls `GET /api/v1/info/status`. It is a self-hosted convenience and is
not part of the Processing API document.

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Success, including a filter result of `false`. |
| `1` | Request, validation, or I/O error. The message is printed to stderr. |
| `130` | Interrupted with Ctrl-C or SIGTERM. |

Usage errors from Cobra (unknown flag, missing required file) also return `1`.
