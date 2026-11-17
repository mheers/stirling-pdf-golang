# SDK guide

The `stirling` package is a typed, streaming client for the Stirling PDF
Processing API. It is safe for concurrent use; create one `*stirling.Client` per
server and reuse it.

```go
import "github.com/mheers/stirling-pdf-golang/stirling"
```

## Creating a client

```go
client, err := stirling.New(
	stirling.WithBaseURL("http://localhost:8080"),
	stirling.WithAPIKey(os.Getenv("STIRLING_API_KEY")),
	stirling.WithTimeout(2*time.Minute),
	stirling.WithRetries(3),
)
```

With no options the client targets `https://api.stirling.com`. The recommended
setup for scripts and services is `NewFromEnv`, which reads the environment and
still accepts overrides:

```go
client, err := stirling.NewFromEnv(stirling.WithTimeout(10*time.Minute))
```

| Option | Default | Purpose |
|---|---|---|
| `WithBaseURL(url)` | `https://api.stirling.com` | Server root. A path prefix such as `https://example.com/pdf` is preserved. |
| `WithAPIKey(key)` | empty | Sends `X-API-KEY`. |
| `WithHTTPClient(client)` | `http.Client{Timeout: 5m}` | Custom transport, proxy, TLS, or connection pool. |
| `WithTimeout(d)` | `5m` | Overall request timeout. `0` disables it. |
| `WithRetries(n)` | `2` | Retries for transport errors and 408/425/429/500/502/503/504. |
| `WithRetryWait(min, max)` | `500ms`, `5s` | Exponential backoff bounds. |
| `WithRetryPolicy(policy)` | — | Replaces the complete policy. |
| `WithHeader(name, value)` | — | Adds a header to every request. |
| `WithUserAgent(ua)` | `stirling-pdf-golang/<version>` | Overrides the user agent. |
| `WithRequestEditor(fn)` | — | Mutates each request, for tracing or proxy auth. |

Environment variables: `STIRLING_URL`, `STIRLING_API_KEY`, `STIRLING_TIMEOUT`
(Go duration), `STIRLING_RETRIES` (integer).

## File inputs

All uploads use `stirling.FileInput`:

| Constructor | Reopenable | Use when |
|---|---|---|
| `FromBytes(name, data)` | yes | You already have the document in memory. |
| `FromString(name, text)` | yes | Markdown, HTML, text, or JSON input. |
| `FromPath(path)` | yes | The file is on disk. Nothing stays open between requests. |
| `FromFS(fsys, name)` | yes | The file is in an `fs.FS` (embedded assets, tests). |
| `FromReader(name, reader)` | no | You have a stream. Requests with it are never retried. |
| `FromReaderWithSize(name, reader, size)` | no | Same, with a known size hint. |
| `FromFileID(id)` | yes | The file already exists on the server; sends `fileId`. |

Reopenable inputs let the client retry safely. With a one-shot reader the client
sends exactly one attempt, because the bytes cannot be rewound.

`FromPath` and `FromFS` open the file only while the multipart body is being
written, so large uploads do not sit in memory.

## Results

Binary endpoints return `*stirling.Result`:

```go
res, err := client.Merge(ctx, files, nil)
if err != nil {
	return err
}
defer res.Close()

fmt.Println(res.StatusCode, res.ContentType, res.Filename(), res.Kind())
if err := res.Save("merged.pdf"); err != nil {
	return err
}
```

| Method | Behavior |
|---|---|
| `Bytes()` | Reads the whole body and closes it. |
| `Save(path)` | Streams to a new file and closes the body. |
| `SaveTo(w)` | Streams to a writer without closing the body. |
| `Close()` | Closes the body; nil-safe. |
| `Filename()` | Parses `Content-Disposition` (including RFC 5987). |
| `ContentType` | Media type without parameters. |
| `Kind()` | `pdf`, `zip`, `png`, `jpeg`, `json`, `text`, and so on. |
| `Extension()` | Conventional extension including the dot. |
| `IsEmpty()` | True for `204 No Content` (used by filters). |

Always close a result, even when you do not read the body, so the connection can
be reused.

## Errors

Non-2xx responses become `*stirling.APIError` with the HTTP status, the API's
message, the request path, response headers, and a capped copy of the body:

```go
res, err := client.Compress(ctx, input, nil)
if err != nil {
	var apiErr *stirling.APIError
	if errors.As(err, &apiErr) {
		log.Printf("HTTP %d: %s (request %s)", apiErr.StatusCode, apiErr.Message, apiErr.RequestID)
	}
	if stirling.StatusCode(err) == http.StatusUnauthorized {
		// bad or missing API key
	}
	return err
}
defer res.Close()
```

`StatusCode(err)` unwraps wrapped errors and returns `0` for non-API errors.
`IsNotFound(err)` covers the common 404 case.

Network and timeout errors are wrapped with the method and path, so
`errors.Is(err, context.Canceled)` and `errors.Is(err, context.DeadlineExceeded)`
keep working.

## Retries

Retries are off for requests that contain a one-shot reader. For everything else
the client retries transport failures and these statuses:

`408 Request Timeout`, `425 Too Early`, `429 Too Many Requests`,
`500 Internal Server Error`, `502 Bad Gateway`, `503 Service Unavailable`,
`504 Gateway Timeout`.

`Retry-After` is honored in both seconds and HTTP-date form. Backoff is
deterministic exponential (`min`, `2×min`, `4×min`, … capped at `max`) so tests
and scripts stay predictable. Retries never reuse a partially consumed body:
each attempt builds a fresh multipart stream.

## Analysis and JSON

The Processing API documents analysis responses as free-form JSON, so those
methods return `map[string]any`:

```go
count, err := client.PageCount(ctx, input)
if err != nil {
	return err
}
fmt.Println(count["pageCount"])
```

Methods: `BasicInfo`, `DocumentProperties`, `FontInfo`, `FormFields`,
`PageCount`, `PageDimensions`, `SecurityInfo`, `AnnotationInfo`,
`ExtractBookmarks`, `PDFInfo`, `ValidateSignature`.

Use `Client.Call` and decode the body yourself when you need a typed result
instead of a map.

## Filters

Filter endpoints answer `200` with the original document when the condition
matches and `204 No Content` when it does not. The SDK turns that into a
`FilterResult`:

```go
result, err := client.FilterContainsText(ctx, input, stirling.FilterContainsTextOptions{
	Pages: "1-3",
	Text:  "invoice",
})
if err != nil {
	return err
}
defer result.Close()
if !result.Matched {
	return nil // 204: no match
}
// result.Result is the original PDF when matched.
```

Filters: `FilterPageSize`, `FilterPageRotation`, `FilterPageCount`,
`FilterFileSize`, `FilterContainsText`, `FilterContainsImage`.

## Pipelines

A pipeline runs several operations in one request. Use the typed builder:

```go
pipeline := stirling.Pipeline{Name: "prepare-for-email"}
pipeline.Step("/api/v1/security/sanitize-pdf", map[string]any{
	"removeJavaScript":    true,
	"removeEmbeddedFiles": true,
})
pipeline.Step("/api/v1/misc/compress-pdf", map[string]any{"optimizeLevel": 2})

res, err := client.RunPipeline(ctx, files, pipeline)
```

`RunPipelineJSON` accepts a pre-encoded definition. `operation` values are full
API paths, not tool names.

## Self-hosted status

`Status` calls `GET /api/v1/info/status`, which self-hosted servers expose but
the Processing API document does not describe:

```go
status, err := client.Status(ctx)
// status.Version == "3.3.0", status.Status == "UP"
```

Use it as a readiness gate before a batch job.

## Generic escape hatch

`Call` sends a multipart POST to any path. `CallMethod` selects the HTTP method.
This is how the CLI reaches every operation and how you can use endpoints that
are newer than the SDK:

```go
res, err := client.Call(ctx, "/api/v1/misc/compress-pdf",
	stirling.Param{Name: "fileInput", Value: input},
	stirling.Param{Name: "optimizeLevel", Value: 5},
	stirling.Param{Name: "linearize", Value: true},
)
```

Supported `Param.Value` types: strings, booleans, integers, floats, `time.Time`,
`[]string`, `[]int`, `[]int64`, `[]float64`, `map[string]string`
(encoded as `name[key]`), `[]map[string]any` (encoded as `name[0][key]`),
`FileInput`, and `[]FileInput`. Query strings in the path are preserved.

## Method reference

### General

| Method | Endpoint |
|---|---|
| `SplitBySections(ctx, in, SplitSectionsOptions)` | `/api/v1/general/split-pdf-by-sections` |
| `SplitByChapters(ctx, in, SplitChaptersOptions)` | `/api/v1/general/split-pdf-by-chapters` |
| `SplitPages(ctx, in, pages)` | `/api/v1/general/split-pages` |
| `SplitBySizeOrCount(ctx, in, SplitSizeOptions)` | `/api/v1/general/split-by-size-or-count` |
| `ScalePages(ctx, in, ScaleOptions)` | `/api/v1/general/scale-pages` |
| `Rotate(ctx, in, angle)` | `/api/v1/general/rotate-pdf` |
| `RemovePages(ctx, in, pages)` | `/api/v1/general/remove-pages` |
| `RemoveImages(ctx, in)` | `/api/v1/general/remove-image-pdf` |
| `RearrangePages(ctx, in, RearrangeOptions)` | `/api/v1/general/rearrange-pages` |
| `ToSinglePage(ctx, in)` | `/api/v1/general/pdf-to-single-page` |
| `OverlayPDFs(ctx, base, overlays, OverlayOptions)` | `/api/v1/general/overlay-pdfs` |
| `MultiPageLayout(ctx, in, MultiPageOptions)` | `/api/v1/general/multi-page-layout` |
| `Merge(ctx, files, *MergeOptions)` | `/api/v1/general/merge-pdfs` |
| `ExtractBookmarks(ctx, in)` | `/api/v1/general/extract-bookmarks` |
| `EditTableOfContents(ctx, in, data, replace)` | `/api/v1/general/edit-table-of-contents` |
| `Crop(ctx, in, CropOptions)` | `/api/v1/general/crop` |
| `Booklet(ctx, in, BookletOptions)` | `/api/v1/general/booklet-imposition` |

### Convert

| Method | Endpoint |
|---|---|
| `ConvertURL(ctx, url)` | `/api/v1/convert/url/pdf` |
| `PDFToXML(ctx, in)` | `/api/v1/convert/pdf/xml` |
| `PDFToWord(ctx, in, PDFToWordFormat)` | `/api/v1/convert/pdf/word` |
| `PDFToText(ctx, in, PDFToTextFormat)` | `/api/v1/convert/pdf/text` |
| `PDFToPresentation(ctx, in, PDFToPresentationFormat)` | `/api/v1/convert/pdf/presentation` |
| `PDFToPDFA(ctx, in, PDFToPDFAFormat)` | `/api/v1/convert/pdf/pdfa` |
| `PDFToMarkdown(ctx, in)` | `/api/v1/convert/pdf/markdown` |
| `PDFToImages(ctx, in, PDFToImagesOptions)` | `/api/v1/convert/pdf/img` |
| `PDFToHTML(ctx, in)` | `/api/v1/convert/pdf/html` |
| `PDFToCSV(ctx, in, pages)` | `/api/v1/convert/pdf/csv` |
| `MarkdownToPDF(ctx, in)` | `/api/v1/convert/markdown/pdf` |
| `ImagesToPDF(ctx, files, ImagesToPDFOptions)` | `/api/v1/convert/img/pdf` |
| `HTMLToPDF(ctx, in, *HTMLToPDFOptions)` | `/api/v1/convert/html/pdf` |
| `FileToPDF(ctx, in)` | `/api/v1/convert/file/pdf` |
| `EMLToPDF(ctx, in, *EMLToPDFOptions)` | `/api/v1/convert/eml/pdf` |

### Security

| Method | Endpoint |
|---|---|
| `ValidateSignature(ctx, in, *SignatureValidationOptions)` | `/api/v1/security/validate-signature` |
| `Sanitize(ctx, in, *SanitizeOptions)` | `/api/v1/security/sanitize-pdf` |
| `RemovePassword(ctx, in, password)` | `/api/v1/security/remove-password` |
| `RemoveCertSign(ctx, in)` | `/api/v1/security/remove-cert-sign` |
| `RedactManual(ctx, in, RedactManualOptions)` | `/api/v1/security/redact` |
| `PDFInfo(ctx, in)` | `/api/v1/security/get-info-on-pdf` |
| `SignPDF(ctx, in, SignPDFOptions)` | `/api/v1/security/cert-sign` |
| `RedactAuto(ctx, in, texts, RedactAutoOptions)` | `/api/v1/security/auto-redact` |
| `AddWatermark(ctx, in, WatermarkOptions)` | `/api/v1/security/add-watermark` |
| `AddPassword(ctx, in, PasswordOptions)` | `/api/v1/security/add-password` |

### Misc

| Method | Endpoint |
|---|---|
| `UpdateMetadata(ctx, in, MetadataOptions)` | `/api/v1/misc/update-metadata` |
| `UnlockForms(ctx, in)` | `/api/v1/misc/unlock-pdf-forms` |
| `ShowJavaScript(ctx, in)` | `/api/v1/misc/show-javascript` |
| `ScannerEffect(ctx, in, ScannerEffectOptions)` | `/api/v1/misc/scanner-effect` |
| `ReplaceInvertColors(ctx, in, ReplaceInvertColorsOptions)` | `/api/v1/misc/replace-invert-pdf` |
| `Repair(ctx, in)` | `/api/v1/misc/repair` |
| `RemoveBlankPages(ctx, in, RemoveBlanksOptions)` | `/api/v1/misc/remove-blanks` |
| `OCR(ctx, in, OCROptions)` | `/api/v1/misc/ocr-pdf` |
| `Flatten(ctx, in, onlyForms)` | `/api/v1/misc/flatten` |
| `ExtractImages(ctx, in, ExtractImagesOptions)` | `/api/v1/misc/extract-images` |
| `ExtractImageScans(ctx, in, ExtractImageScansOptions)` | `/api/v1/misc/extract-image-scans` |
| `Decompress(ctx, in)` | `/api/v1/misc/decompress-pdf` |
| `Compress(ctx, in, *CompressOptions)` | `/api/v1/misc/compress-pdf` |
| `AutoSplit(ctx, in, *AutoSplitOptions)` | `/api/v1/misc/auto-split-pdf` |
| `AutoRename(ctx, in, *AutoRenameOptions)` | `/api/v1/misc/auto-rename` |
| `AddStamp(ctx, in, StampOptions)` | `/api/v1/misc/add-stamp` |
| `AddPageNumbers(ctx, in, PageNumbersOptions)` | `/api/v1/misc/add-page-numbers` |
| `AddImage(ctx, in, AddImageOptions)` | `/api/v1/misc/add-image` |
| `AddAttachments(ctx, in, attachments)` | `/api/v1/misc/add-attachments` |

`ScannerEffect` uses `ScannerColorspaceColor` or `ScannerColorspaceGrayscale`.
The scanner endpoint spells greyscale `grayscale`, while the conversion
endpoints use `ColorModeGreyscale`.

### Analysis and filter

| Method | Endpoint |
|---|---|
| `SecurityInfo`, `PageDimensions`, `PageCount`, `FormFields` | `/api/v1/analysis/...` |
| `FontInfo`, `DocumentProperties`, `BasicInfo`, `AnnotationInfo` | `/api/v1/analysis/...` |
| `FilterPageSize`, `FilterPageRotation`, `FilterPageCount` | `/api/v1/filter/...` |
| `FilterFileSize`, `FilterContainsText`, `FilterContainsImage` | `/api/v1/filter/...` |

### Pipeline

| Method | Endpoint |
|---|---|
| `RunPipeline(ctx, files, Pipeline)` | `/api/v1/pipeline/handleData` |
| `RunPipelineJSON(ctx, files, json)` | `/api/v1/pipeline/handleData` |

## Low-level generated API

`api` contains the unmodified `oapi-codegen` output and the vendored OpenAPI
document. It is useful when you need a request type exactly as generated, or a
`WithBody` method that the high-level SDK does not expose:

```go
import "github.com/mheers/stirling-pdf-golang/api"

lowLevel := api.NewClient(server)
```

The high-level `stirling` package is the recommended entry point.

## Examples

### Rotate and watermark in one pipeline

```go
pipeline := stirling.Pipeline{}
pipeline.Step("/api/v1/general/rotate-pdf", map[string]any{"angle": 90})
pipeline.Step("/api/v1/security/add-watermark", map[string]any{
	"watermarkType": "text",
	"watermarkText": "DRAFT",
	"opacity":       0.25,
})

res, err := client.RunPipeline(ctx, []stirling.FileInput{input}, pipeline)
```

### Encrypt, then decrypt later

```go
encrypted, err := client.AddPassword(ctx, input, stirling.PasswordOptions{
	UserPassword:  "reader-password",
	OwnerPassword: "owner-password",
	KeyLength:     stirling.Ptr(256),
})
// ...
decrypted, err := client.RemovePassword(ctx, stirling.FromBytes("locked.pdf", data), "reader-password")
```

### Save every page as a PNG

```go
res, err := client.PDFToImages(ctx, input, stirling.PDFToImagesOptions{
	Format:    stirling.ImagePNG,
	Output:    stirling.ImageMultiple,
	ColorType: stirling.ColorModeColor,
	DPI:       stirling.Ptr(150),
})
// res is a ZIP archive; res.Extension() == ".zip".
```
