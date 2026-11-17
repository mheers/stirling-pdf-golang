package stirling

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (t *trackingReadCloser) Close() error {
	t.closed = true
	return nil
}

func TestResultFilename(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{"quoted", `attachment; filename="merged.pdf"`, "merged.pdf"},
		{"unquoted", `attachment; filename=result.zip`, "result.zip"},
		{"rfc5987", `attachment; filename*=UTF-8''r%C3%A9sum%C3%A9.pdf`, "résumé.pdf"},
		{"none", "", ""},
		{"garbage", `;;;`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := http.Header{}
			if tt.header != "" {
				h.Set("Content-Disposition", tt.header)
			}
			r := &Result{Header: h}
			if got := r.Filename(); got != tt.want {
				t.Fatalf("Filename() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResultBytesClosesBody(t *testing.T) {
	body := &trackingReadCloser{Reader: bytes.NewBufferString("hello")}
	r := &Result{StatusCode: 200, Header: http.Header{}, Body: body, ContentType: "text/plain"}
	data, err := r.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("Bytes() = %q", data)
	}
	if !body.closed {
		t.Fatal("Bytes() did not close the body")
	}
}

func TestResultSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.pdf")
	r := &Result{
		StatusCode:  200,
		Header:      http.Header{},
		Body:        io.NopCloser(bytes.NewBufferString("pdf-bytes")),
		ContentType: "application/pdf",
	}
	if err := r.Save(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "pdf-bytes" {
		t.Fatalf("saved %q", data)
	}
	if err := r.Save(""); err == nil {
		t.Fatal("expected an error for an empty path")
	}
}

func TestResultSaveTo(t *testing.T) {
	r := &Result{
		StatusCode: 200,
		Header:     http.Header{},
		Body:       io.NopCloser(bytes.NewBufferString("copy")),
	}
	var dst bytes.Buffer
	n, err := r.SaveTo(&dst)
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 || dst.String() != "copy" {
		t.Fatalf("SaveTo() = %d, %q", n, dst.String())
	}
}

func TestResultExtensionAndContentType(t *testing.T) {
	tests := []struct {
		ct   string
		ext  string
		kind string
	}{
		{"application/pdf", ".pdf", "pdf"},
		{"application/zip", ".zip", "zip"},
		{"image/png", ".png", "png"},
		{"image/jpeg", ".jpg", "jpeg"},
		{"text/plain; charset=utf-8", ".txt", "text"},
		{"application/json", ".json", "json"},
		{"text/markdown", ".md", "markdown"},
		{"application/octet-stream", "", "binary"},
	}
	for _, tt := range tests {
		r := &Result{ContentType: tt.ct}
		if got := r.Extension(); got != tt.ext {
			t.Errorf("Extension(%q) = %q, want %q", tt.ct, got, tt.ext)
		}
		if got := r.Kind(); got != tt.kind {
			t.Errorf("Kind(%q) = %q, want %q", tt.ct, got, tt.kind)
		}
	}
}

func TestResultIsEmpty(t *testing.T) {
	if !(&Result{StatusCode: 204}).IsEmpty() {
		t.Fatal("204 should be empty")
	}
	if (&Result{StatusCode: 200}).IsEmpty() {
		t.Fatal("200 should not be empty")
	}
}

func TestResultCloseNilBody(t *testing.T) {
	r := &Result{}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	var nilResult *Result
	if err := nilResult.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestResultReadError(t *testing.T) {
	boom := errors.New("boom")
	r := &Result{Body: io.NopCloser(errReader{err: boom})}
	if _, err := r.Bytes(); !errors.Is(err, boom) {
		t.Fatalf("Bytes() error = %v", err)
	}
}

type errReader struct{ err error }

func (e errReader) Read([]byte) (int, error) { return 0, e.err }
