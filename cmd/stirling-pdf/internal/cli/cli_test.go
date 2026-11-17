package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/mheers/stirling-pdf-golang/internal/ops"
	"github.com/mheers/stirling-pdf-golang/stirling"
)

type request struct {
	Method string
	Path   string
	Values url.Values
	Files  map[string][]string
}

type fakeServer struct {
	mu       sync.Mutex
	requests []request
	server   *httptest.Server
	handler  func(w http.ResponseWriter, r *http.Request, req request)
}

func newFakeServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request, req request)) *fakeServer {
	t.Helper()
	fake := &fakeServer{handler: handler}
	fake.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := request{Method: r.Method, Path: r.URL.Path, Files: map[string][]string{}}
		if r.Method == http.MethodPost {
			if err := r.ParseMultipartForm(8 << 20); err != nil {
				t.Errorf("ParseMultipartForm: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			req.Values = r.MultipartForm.Value
			for name, headers := range r.MultipartForm.File {
				for _, header := range headers {
					req.Files[name] = append(req.Files[name], header.Filename)
				}
			}
		}
		fake.mu.Lock()
		fake.requests = append(fake.requests, req)
		fake.mu.Unlock()
		if fake.handler != nil {
			fake.handler(w, r, req)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF-1.7 result"))
	}))
	t.Cleanup(fake.server.Close)
	return fake
}

func (f *fakeServer) last(t *testing.T) request {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.requests) == 0 {
		t.Fatal("no request captured")
	}
	return f.requests[len(f.requests)-1]
}

func runCLI(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	root := newRootCommand()
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return stdout.String(), stderr.String(), err
}

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestMergeCommand(t *testing.T) {
	server := newFakeServer(t, nil)
	out := filepath.Join(t.TempDir(), "merged.pdf")
	a := writeTempFile(t, "a.pdf", "%PDF-1.7 a")
	b := writeTempFile(t, "b.pdf", "%PDF-1.7 b")

	stdout, _, err := runCLI(t,
		"--url", server.server.URL,
		"general", "merge-pdfs",
		"--file", a, "--file", b,
		"--sort-type", "byFileName",
		"--generate-toc",
		"-o", out,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(stdout); got != out {
		t.Fatalf("stdout = %q", got)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "%PDF-1.7 result" {
		t.Fatalf("output = %q", data)
	}
	req := server.last(t)
	if req.Path != "/api/v1/general/merge-pdfs" {
		t.Fatalf("path = %q", req.Path)
	}
	if got := req.Values["sortType"]; len(got) != 1 || got[0] != "byFileName" {
		t.Errorf("sortType = %v", got)
	}
	if got := req.Values["generateToc"]; len(got) != 1 || got[0] != "true" {
		t.Errorf("generateToc = %v", got)
	}
	if got := req.Files["fileInput"]; strings.Join(got, ",") != "a.pdf,b.pdf" {
		t.Errorf("files = %v", got)
	}
}

func TestCommandRefusesOverwrite(t *testing.T) {
	server := newFakeServer(t, nil)
	out := filepath.Join(t.TempDir(), "merged.pdf")
	if err := os.WriteFile(out, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	pdf := writeTempFile(t, "a.pdf", "%PDF-1.7")

	_, _, err := runCLI(t, "--url", server.server.URL, "general", "merge-pdfs", "--file", pdf, "-o", out)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %v", err)
	}
	if _, _, err := runCLI(t, "--url", server.server.URL, "general", "merge-pdfs", "--file", pdf, "-o", out, "--force"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(out)
	if string(data) != "%PDF-1.7 result" {
		t.Fatalf("output = %q", data)
	}
}

func TestJSONAnalysis(t *testing.T) {
	server := newFakeServer(t, func(w http.ResponseWriter, r *http.Request, _ request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"pageCount":2}`))
	})
	pdf := writeTempFile(t, "a.pdf", "%PDF-1.7")
	stdout, _, err := runCLI(t, "--url", server.server.URL, "analysis", "page-count", "--file", pdf)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, `"pageCount": 2`) {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestFilterPrintsBoolean(t *testing.T) {
	var matched bool
	server := newFakeServer(t, func(w http.ResponseWriter, r *http.Request, _ request) {
		if matched {
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = w.Write([]byte("%PDF-1.7"))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	pdf := writeTempFile(t, "a.pdf", "%PDF-1.7")
	args := []string{"--url", server.server.URL, "filter", "filter-page-count", "--file", pdf, "--comparator", "Greater", "--page-count", "1"}

	stdout, _, err := runCLI(t, args...)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout) != "false" {
		t.Fatalf("stdout = %q", stdout)
	}

	matched = true
	stdout, _, err = runCLI(t, args...)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout) != "true" {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestFilterWritesMatchedDocument(t *testing.T) {
	server := newFakeServer(t, func(w http.ResponseWriter, r *http.Request, _ request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF-1.7 matched"))
	})
	pdf := writeTempFile(t, "a.pdf", "%PDF-1.7")
	out := filepath.Join(t.TempDir(), "matched.pdf")
	stdout, _, err := runCLI(t, "--url", server.server.URL, "filter", "filter-page-count", "--file", pdf, "--comparator", "Greater", "--page-count", "1", "-o", out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "true") {
		t.Fatalf("stdout = %q", stdout)
	}
	data, _ := os.ReadFile(out)
	if string(data) != "%PDF-1.7 matched" {
		t.Fatalf("output = %q", data)
	}
}

func TestFilterStreamsMatchedDocumentWithoutBoolean(t *testing.T) {
	server := newFakeServer(t, func(w http.ResponseWriter, r *http.Request, _ request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF-1.7 matched"))
	})
	pdf := writeTempFile(t, "a.pdf", "%PDF-1.7")
	stdout, _, err := runCLI(t, "--url", server.server.URL, "filter", "filter-page-count", "--file", pdf, "--comparator", "Greater", "--page-count", "1", "-o", "-")
	if err != nil {
		t.Fatal(err)
	}
	if stdout != "%PDF-1.7 matched" {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestStdoutOutput(t *testing.T) {
	server := newFakeServer(t, nil)
	pdf := writeTempFile(t, "a.pdf", "%PDF-1.7")
	stdout, _, err := runCLI(t, "--url", server.server.URL, "general", "rotate-pdf", "--file", pdf, "--angle", "90", "-o", "-")
	if err != nil {
		t.Fatal(err)
	}
	if stdout != "%PDF-1.7 result" {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestObjectListFlag(t *testing.T) {
	server := newFakeServer(t, nil)
	pdf := writeTempFile(t, "a.pdf", "%PDF-1.7")
	redactions := `[{"x":10,"y":20,"width":30,"height":40,"page":1,"color":"#000000"}]`
	_, _, err := runCLI(t, "--url", server.server.URL, "security", "redact", "--file", pdf, "--redactions", redactions, "-o", "-")
	if err != nil {
		t.Fatal(err)
	}
	values := server.last(t).Values
	for name, want := range map[string]string{
		"redactions[0].x":      "10",
		"redactions[0].y":      "20",
		"redactions[0].width":  "30",
		"redactions[0].height": "40",
		"redactions[0].page":   "1",
		"redactions[0].color":  "#000000",
	} {
		if got := values[name]; len(got) != 1 || got[0] != want {
			t.Errorf("%s = %v, want %q", name, got, want)
		}
	}
}

func TestExtraFields(t *testing.T) {
	server := newFakeServer(t, nil)
	pdf := writeTempFile(t, "a.pdf", "%PDF-1.7")
	attachment := writeTempFile(t, "notes.txt", "hi")
	_, _, err := runCLI(t,
		"--url", server.server.URL,
		"general", "rotate-pdf",
		"--file", pdf,
		"--angle", "90",
		"--field", "customFlag=1",
		"--file-field", "extraFile="+attachment,
		"-o", "-",
	)
	if err != nil {
		t.Fatal(err)
	}
	req := server.last(t)
	if got := req.Values["customFlag"]; len(got) != 1 || got[0] != "1" {
		t.Errorf("customFlag = %v", got)
	}
	if got := req.Files["extraFile"]; len(got) != 1 || got[0] != "notes.txt" {
		t.Errorf("extraFile = %v", got)
	}
}

func TestRequiredFileFlag(t *testing.T) {
	_, _, err := runCLI(t, "--url", "http://127.0.0.1:1", "general", "merge-pdfs")
	if err == nil || !strings.Contains(err.Error(), `required flag(s) "file" not set`) {
		t.Fatalf("error = %v", err)
	}
}

func TestFileIDFlag(t *testing.T) {
	server := newFakeServer(t, nil)
	_, _, err := runCLI(t, "--url", server.server.URL, "general", "rotate-pdf", "--file-id", "abc", "--angle", "90", "-o", "-")
	if err != nil {
		t.Fatal(err)
	}
	if got := server.last(t).Values["fileId"]; len(got) != 1 || got[0] != "abc" {
		t.Fatalf("fileId = %v", got)
	}
}

func TestFileIDSatisfiesRequiredFile(t *testing.T) {
	operation := ops.Operation{Fields: []ops.Field{
		{Name: "fileInput", Flag: "file", Kind: ops.FileKind, Required: true},
		{Name: "fileId", Flag: "file-id", Kind: ops.StringKind},
	}}
	if err := validateFileParams(operation, []stirling.Param{{Name: "fileId", Value: "abc"}}); err != nil {
		t.Fatalf("file ID should satisfy a required fileInput: %v", err)
	}
	if err := validateFileParams(operation, nil); err == nil || !strings.Contains(err.Error(), `required flag(s) "file" not set`) {
		t.Fatalf("error = %v", err)
	}
}

func TestFileAndFileIDConflict(t *testing.T) {
	operation := ops.Operation{Fields: []ops.Field{
		{Name: "fileInput", Flag: "file", Kind: ops.FileKind, Required: true},
		{Name: "fileId", Flag: "file-id", Kind: ops.StringKind},
	}}
	err := validateFileParams(operation, []stirling.Param{{Name: "fileInput"}, {Name: "fileId"}})
	if err == nil || !strings.Contains(err.Error(), "cannot be used together") {
		t.Fatalf("error = %v", err)
	}
}

func TestFileIDCannotSatisfyFileList(t *testing.T) {
	operation := ops.Operation{Fields: []ops.Field{
		{Name: "fileInput", Flag: "file", Kind: ops.FileListKind, Required: true},
		{Name: "fileId", Flag: "file-id", Kind: ops.StringKind},
	}}
	err := validateFileParams(operation, []stirling.Param{{Name: "fileId", Value: "abc"}})
	if err == nil || !strings.Contains(err.Error(), `required flag(s) "file" not set`) {
		t.Fatalf("error = %v", err)
	}
}

func TestOutputAndStdoutConflict(t *testing.T) {
	server := newFakeServer(t, nil)
	pdf := writeTempFile(t, "a.pdf", "%PDF-1.7")
	_, _, err := runCLI(t, "--url", server.server.URL, "analysis", "page-count", "--file", pdf, "--stdout", "-o", "out.json")
	if err == nil || !strings.Contains(err.Error(), "cannot be used together") {
		t.Fatalf("error = %v", err)
	}
}

func TestIsTerminal(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	if isTerminal(file) {
		t.Fatal("regular file reported as a terminal")
	}
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = devNull.Close() }()
	if isTerminal(devNull) {
		t.Fatal(os.DevNull + " reported as a terminal")
	}
}

func TestExitCode(t *testing.T) {
	if got := exitCode(nil); got != 0 {
		t.Fatalf("exitCode(nil) = %d", got)
	}
	if got := exitCode(context.Canceled); got != 130 {
		t.Fatalf("exitCode(canceled) = %d", got)
	}
	if got := exitCode(fmt.Errorf("wrapped: %w", context.Canceled)); got != 130 {
		t.Fatalf("exitCode(wrapped canceled) = %d", got)
	}
	if got := exitCode(errors.New("boom")); got != 1 {
		t.Fatalf("exitCode(error) = %d", got)
	}
}

func TestStatusCommand(t *testing.T) {
	server := newFakeServer(t, func(w http.ResponseWriter, r *http.Request, _ request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"version":"3.3.0","status":"UP"}`))
	})
	stdout, _, err := runCLI(t, "--url", server.server.URL, "status")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout) != "3.3.0 UP" {
		t.Fatalf("stdout = %q", stdout)
	}
	stdout, _, err = runCLI(t, "--url", server.server.URL, "status", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var status map[string]any
	if err := json.Unmarshal([]byte(stdout), &status); err != nil {
		t.Fatal(err)
	}
	if status["version"] != "3.3.0" {
		t.Fatalf("status = %v", status)
	}
}

func TestListCommand(t *testing.T) {
	stdout, _, err := runCLI(t, "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "general merge-pdfs") {
		t.Fatalf("stdout missing merge: %q", stdout)
	}
	stdout, _, err = runCLI(t, "list", "--group", "filter", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var entries []map[string]any
	if err := json.Unmarshal([]byte(stdout), &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 6 {
		t.Fatalf("entries = %d", len(entries))
	}
	if _, _, err := runCLI(t, "list", "--group", "nope", "--json"); err == nil || !strings.Contains(err.Error(), "unknown group") {
		t.Fatalf("error = %v", err)
	}
}

func TestVersionCommand(t *testing.T) {
	stdout, _, err := runCLI(t, "version")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stdout, "stirling-pdf ") {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestTextResponse(t *testing.T) {
	server := newFakeServer(t, func(w http.ResponseWriter, r *http.Request, _ request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("console.log('hi')"))
	})
	pdf := writeTempFile(t, "a.pdf", "%PDF-1.7")
	stdout, _, err := runCLI(t, "--url", server.server.URL, "misc", "show-javascript", "--file", pdf)
	if err != nil {
		t.Fatal(err)
	}
	if stdout != "console.log('hi')" {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestPipelineCommand(t *testing.T) {
	server := newFakeServer(t, nil)
	pdf := writeTempFile(t, "a.pdf", "%PDF-1.7")
	pipeline := `{"name":"test","pipeline":[{"operation":"/api/v1/misc/repair"}]}`
	_, _, err := runCLI(t, "--url", server.server.URL, "pipeline", "handle-data", "--file", pdf, "--json", pipeline, "-o", "-")
	if err != nil {
		t.Fatal(err)
	}
	req := server.last(t)
	if got := req.Values["json"]; len(got) != 1 || got[0] != pipeline {
		t.Fatalf("json = %v", got)
	}
}

func TestInvalidURL(t *testing.T) {
	_, _, err := runCLI(t, "--url", "not-a-url", "list")
	if err != nil {
		t.Fatalf("list should not need a server: %v", err)
	}
	if _, _, err := runCLI(t, "--url", "not-a-url", "status"); err == nil {
		t.Fatal("expected a URL error")
	}
}

func TestMultipartValuesAreStrings(t *testing.T) {
	server := newFakeServer(t, nil)
	pdf := writeTempFile(t, "a.pdf", "%PDF-1.7")
	_, _, err := runCLI(t, "--url", server.server.URL, "misc", "compress-pdf", "--file", pdf, "--optimize-level", "9", "--linearize", "-o", "-")
	if err != nil {
		t.Fatal(err)
	}
	req := server.last(t)
	if got := req.Values["optimizeLevel"]; len(got) != 1 || got[0] != "9" {
		t.Fatalf("optimizeLevel = %v", got)
	}
	if got := req.Values["linearize"]; len(got) != 1 || got[0] != "true" {
		t.Fatalf("linearize = %v", got)
	}
}
