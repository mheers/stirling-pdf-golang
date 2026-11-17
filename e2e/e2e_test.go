//go:build e2e

// Package e2e verifies the SDK and CLI against a real Stirling PDF server in a
// container. Run it with:
//
//	go test -tags=e2e ./e2e -v
//
// Set STIRLING_E2E_URL to test an already running server instead of starting a
// container.
package e2e

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mheers/stirling-pdf-golang/internal/testutil"
	"github.com/mheers/stirling-pdf-golang/stirling"
)

var baseURL string

func TestMain(m *testing.M) {
	if existing := os.Getenv("STIRLING_E2E_URL"); existing != "" {
		baseURL = existing
		os.Exit(m.Run())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	container, endpoint, err := testutil.StartStirling(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "e2e:", err)
		os.Exit(1)
	}
	baseURL = endpoint
	code := m.Run()
	_ = container.Terminate(context.Background())
	os.Exit(code)
}

func newClient(t *testing.T) *stirling.Client {
	t.Helper()
	client, err := stirling.New(
		stirling.WithBaseURL(baseURL),
		stirling.WithTimeout(2*time.Minute),
		stirling.WithRetries(1),
	)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func requirePDF(t *testing.T, res *stirling.Result) []byte {
	t.Helper()
	data, err := res.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF")) {
		t.Fatalf("response is not a PDF: %q", data[:min(len(data), 40)])
	}
	return data
}

func TestStatus(t *testing.T) {
	status, err := newClient(t).Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "UP" {
		t.Fatalf("status = %+v", status)
	}
	if status.Version == "" {
		t.Fatal("version is empty")
	}
}

func TestMergeAndAnalyze(t *testing.T) {
	client := newClient(t)
	a := stirling.FromBytes("alpha.pdf", testutil.PDF("Alpha document"))
	b := stirling.FromBytes("beta.pdf", testutil.PDF("Beta document", "Beta page two"))

	merged, err := client.Merge(context.Background(), []stirling.FileInput{a, b}, &stirling.MergeOptions{GenerateTOC: stirling.Ptr(true)})
	if err != nil {
		t.Fatal(err)
	}
	data := requirePDF(t, merged)
	if len(data) < 200 {
		t.Fatalf("merged PDF is suspiciously small: %d bytes", len(data))
	}

	count, err := client.PageCount(context.Background(), stirling.FromBytes("merged.pdf", data))
	if err != nil {
		t.Fatal(err)
	}
	if got := count["pageCount"]; got != float64(3) {
		t.Fatalf("pageCount = %v, want 3", got)
	}
}

func TestSplitPagesProducesZip(t *testing.T) {
	client := newClient(t)
	res, err := client.SplitPages(context.Background(), stirling.FromBytes("split.pdf", testutil.PDF("One", "Two", "Three")), "1")
	if err != nil {
		t.Fatal(err)
	}
	data, err := res.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("response is not a ZIP: %v", err)
	}
	pdfs := 0
	for _, file := range reader.File {
		if strings.HasSuffix(strings.ToLower(file.Name), ".pdf") {
			pdfs++
		}
	}
	if pdfs < 2 {
		t.Fatalf("ZIP contains %d PDFs, want at least 2", pdfs)
	}
}

func TestCompressPreservesPages(t *testing.T) {
	client := newClient(t)
	original := testutil.PDF("Compress me", "Second page")
	res, err := client.Compress(context.Background(), stirling.FromBytes("input.pdf", original), &stirling.CompressOptions{
		OptimizeLevel:      stirling.Ptr(5),
		ExpectedOutputSize: "100KB",
	})
	if err != nil {
		t.Fatal(err)
	}
	compressed := requirePDF(t, res)
	count, err := client.PageCount(context.Background(), stirling.FromBytes("compressed.pdf", compressed))
	if err != nil {
		t.Fatal(err)
	}
	if got := count["pageCount"]; got != float64(2) {
		t.Fatalf("pageCount = %v, want 2", got)
	}
}

func TestRotateAndCrop(t *testing.T) {
	client := newClient(t)
	rotated, err := client.Rotate(context.Background(), stirling.FromBytes("input.pdf", testutil.PDF("Rotate me")), 90)
	if err != nil {
		t.Fatal(err)
	}
	rotatedData := requirePDF(t, rotated)

	cropped, err := client.Crop(context.Background(), stirling.FromBytes("rotated.pdf", rotatedData), stirling.CropOptions{X: 10, Y: 10, Width: 300, Height: 400})
	if err != nil {
		t.Fatal(err)
	}
	requirePDF(t, cropped)
}

func TestScannerEffect(t *testing.T) {
	client := newClient(t)
	res, err := client.ScannerEffect(context.Background(), stirling.FromBytes("scan.pdf", testutil.PDF("Scanned page")), stirling.ScannerEffectOptions{
		Quality:    stirling.ScanQualityMedium,
		Rotation:   stirling.ScanRotationNone,
		Colorspace: stirling.ScannerColorspaceGrayscale,
	})
	if err != nil {
		t.Fatal(err)
	}
	requirePDF(t, res)
}

func TestRedactManual(t *testing.T) {
	client := newClient(t)
	res, err := client.RedactManual(context.Background(), stirling.FromBytes("redact.pdf", testutil.PDF("Secret content")), stirling.RedactManualOptions{
		Pages: "1",
		Redactions: []stirling.RedactionArea{
			{X: 20, Y: 20, Width: 120, Height: 60, Page: 1, Color: "#000000"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	requirePDF(t, res)
}

func TestSanitizeDefaults(t *testing.T) {
	client := newClient(t)
	res, err := client.Sanitize(context.Background(), stirling.FromBytes("sanitize.pdf", testutil.PDF("Sanitize me")), nil)
	if err != nil {
		t.Fatal(err)
	}
	requirePDF(t, res)
}

func TestConvertToImagesAndText(t *testing.T) {
	client := newClient(t)
	input := stirling.FromBytes("convert.pdf", testutil.PDF("Hello from Stirling PDF"))

	imageResult, err := client.PDFToImages(context.Background(), input, stirling.PDFToImagesOptions{
		Format:    stirling.ImagePNG,
		Output:    stirling.ImageSingle,
		ColorType: stirling.ColorModeColor,
		DPI:       stirling.Ptr(72),
	})
	if err != nil {
		t.Fatal(err)
	}
	imageData, err := imageResult.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(imageData))
	if err != nil {
		t.Fatalf("decode image: %v", err)
	}
	if format != "png" || config.Width == 0 || config.Height == 0 {
		t.Fatalf("image = %s %dx%d", format, config.Width, config.Height)
	}

	textResult, err := client.PDFToText(context.Background(), input, stirling.PDFToTXT)
	if err != nil {
		t.Fatal(err)
	}
	text, err := textResult.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(text), "Hello from Stirling PDF") {
		t.Fatalf("extracted text = %q", text)
	}
}

func TestWatermarkAndPageNumbers(t *testing.T) {
	client := newClient(t)
	input := stirling.FromBytes("input.pdf", testutil.PDF("Watermark me", "Second page"))

	watermarked, err := client.AddWatermark(context.Background(), input, stirling.WatermarkOptions{
		Type:     stirling.WatermarkText,
		Text:     "CONFIDENTIAL",
		Opacity:  stirling.Ptr(0.3),
		FontSize: stirling.Ptr(float64(40)),
		Alphabet: stirling.AlphabetRoman,
	})
	if err != nil {
		t.Fatal(err)
	}
	watermarkedData := requirePDF(t, watermarked)

	numbered, err := client.AddPageNumbers(context.Background(), stirling.FromBytes("watermarked.pdf", watermarkedData), stirling.PageNumbersOptions{
		Position:       stirling.Ptr(stirling.PositionBottomCenter),
		FontType:       stirling.FontHelvetica,
		StartingNumber: stirling.Ptr(1),
	})
	if err != nil {
		t.Fatal(err)
	}
	requirePDF(t, numbered)
}

func TestEncryptAndDecrypt(t *testing.T) {
	client := newClient(t)
	original := testutil.PDF("Secret", "Second secret page")
	encrypted, err := client.AddPassword(context.Background(), stirling.FromBytes("secret.pdf", original), stirling.PasswordOptions{
		UserPassword:  "user-secret",
		OwnerPassword: "owner-secret",
		KeyLength:     stirling.Ptr(256),
	})
	if err != nil {
		t.Fatal(err)
	}
	encryptedData := requirePDF(t, encrypted)

	if _, err := client.PageCount(context.Background(), stirling.FromBytes("encrypted.pdf", encryptedData)); err == nil {
		t.Fatal("expected page analysis of an encrypted PDF to fail without a password")
	}

	decrypted, err := client.RemovePassword(context.Background(), stirling.FromBytes("encrypted.pdf", encryptedData), "user-secret")
	if err != nil {
		t.Fatal(err)
	}
	decryptedData := requirePDF(t, decrypted)
	count, err := client.PageCount(context.Background(), stirling.FromBytes("decrypted.pdf", decryptedData))
	if err != nil {
		t.Fatal(err)
	}
	if got := count["pageCount"]; got != float64(2) {
		t.Fatalf("pageCount = %v, want 2", got)
	}
}

func TestFilters(t *testing.T) {
	client := newClient(t)
	input := stirling.FromBytes("filter.pdf", testutil.PDF("Invoice number 42"))

	matched, err := client.FilterPageCount(context.Background(), input, stirling.FilterPageCountOptions{Comparator: stirling.Greater, Count: 0})
	if err != nil {
		t.Fatal(err)
	}
	if !matched.Matched {
		t.Fatal("page-count filter should match")
	}
	requirePDF(t, matched.Result)

	notMatched, err := client.FilterPageCount(context.Background(), input, stirling.FilterPageCountOptions{Comparator: stirling.Greater, Count: 10})
	if err != nil {
		t.Fatal(err)
	}
	if notMatched.Matched {
		t.Fatal("page-count filter should not match")
	}

	textMatch, err := client.FilterContainsText(context.Background(), input, stirling.FilterContainsTextOptions{Pages: "all", Text: "Invoice"})
	if err != nil {
		t.Fatal(err)
	}
	if !textMatch.Matched {
		t.Fatal("contains-text filter should match")
	}
	_ = textMatch.Close()
}

func TestPipeline(t *testing.T) {
	client := newClient(t)
	pipeline := stirling.Pipeline{Name: "sanitize-and-compress"}
	pipeline.Step("/api/v1/security/sanitize-pdf", map[string]any{"removeJavaScript": true})
	pipeline.Step("/api/v1/misc/compress-pdf", map[string]any{"optimizeLevel": 5})
	res, err := client.RunPipeline(context.Background(), []stirling.FileInput{
		stirling.FromBytes("input.pdf", testutil.PDF("Pipeline page one", "Pipeline page two")),
	}, pipeline)
	if err != nil {
		t.Fatal(err)
	}
	requirePDF(t, res)
}

func TestCLIEndToEnd(t *testing.T) {
	binary := buildCLI(t)
	dir := t.TempDir()
	a := filepath.Join(dir, "a.pdf")
	b := filepath.Join(dir, "b.pdf")
	if err := os.WriteFile(a, testutil.PDF("CLI alpha"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, testutil.PDF("CLI beta", "CLI beta two"), 0o600); err != nil {
		t.Fatal(err)
	}
	merged := filepath.Join(dir, "merged.pdf")

	output := run(t, nil, binary, "--url", baseURL, "general", "merge-pdfs", "--file", a, "--file", b, "-o", merged)
	if strings.TrimSpace(output) != merged {
		t.Fatalf("merge output = %q", output)
	}

	redacted := filepath.Join(dir, "redacted.pdf")
	run(t, nil, binary, "--url", baseURL, "security", "redact", "--file", a,
		"--redactions", `[{"x":20,"y":20,"width":80,"height":40,"page":1,"color":"#000000"}]`,
		"-o", redacted)
	redactedData, err := os.ReadFile(redacted)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(redactedData, []byte("%PDF")) {
		t.Fatalf("redacted output is not a PDF: %q", redactedData[:min(len(redactedData), 40)])
	}

	analysis := run(t, nil, binary, "--url", baseURL, "analysis", "page-count", "--file", merged)
	if !strings.Contains(analysis, `"pageCount": 3`) {
		t.Fatalf("analysis = %q", analysis)
	}

	if got := strings.TrimSpace(run(t, nil, binary, "--url", baseURL, "filter", "filter-page-count", "--file", merged, "--comparator", "Greater", "--page-count", "5")); got != "false" {
		t.Fatalf("filter output = %q", got)
	}

	stdin := testutil.PDF("Text from stdin")
	markdown := run(t, stdin, binary, "--url", baseURL, "convert", "pdf-markdown", "--file", "-", "-o", "-")
	if !strings.Contains(markdown, "Text from stdin") {
		t.Fatalf("markdown = %q", markdown)
	}

	status := run(t, nil, binary, "--url", baseURL, "status")
	if !strings.HasSuffix(strings.TrimSpace(status), "UP") {
		t.Fatalf("status = %q", status)
	}
}

func buildCLI(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "stirling-pdf")
	command := exec.Command("go", "build", "-o", binary, "./cmd/stirling-pdf")
	command.Dir = repoRoot(t)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	return binary
}

func run(t *testing.T, stdin []byte, binary string, args ...string) string {
	t.Helper()
	command := exec.Command(binary, args...)
	if stdin != nil {
		command.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("%s: %v\nstderr: %s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String()
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	return filepath.Dir(filepath.Dir(file))
}
