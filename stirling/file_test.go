package stirling

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestFromBytes(t *testing.T) {
	in := FromBytes("document.pdf", []byte("%PDF-1.7"))
	if in.Name() != "document.pdf" {
		t.Fatalf("Name() = %q", in.Name())
	}
	if in.Size() != 8 {
		t.Fatalf("Size() = %d", in.Size())
	}
	for i := 0; i < 2; i++ {
		rc, err := in.Open()
		if err != nil {
			t.Fatalf("Open() attempt %d: %v", i, err)
		}
		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatal(err)
		}
		if err := rc.Close(); err != nil {
			t.Fatal(err)
		}
		if string(data) != "%PDF-1.7" {
			t.Fatalf("read %q", data)
		}
	}
}

func TestFromString(t *testing.T) {
	in := FromString("readme.md", "# hello")
	if in.Name() != "readme.md" || in.Size() != 7 {
		t.Fatalf("unexpected input: %q %d", in.Name(), in.Size())
	}
}

func TestFromPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	if err := os.WriteFile(path, []byte("file-on-disk"), 0o600); err != nil {
		t.Fatal(err)
	}

	in, err := FromPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if in.Name() != "sample.pdf" {
		t.Fatalf("Name() = %q", in.Name())
	}
	if in.Size() != 12 {
		t.Fatalf("Size() = %d", in.Size())
	}

	rc, err := in.Open()
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	_ = rc.Close()
	if string(data) != "file-on-disk" {
		t.Fatalf("read %q", data)
	}
}

func TestFromPathErrors(t *testing.T) {
	if _, err := FromPath(filepath.Join(t.TempDir(), "missing.pdf")); err == nil {
		t.Fatal("expected an error for a missing file")
	}
	if _, err := FromPath(t.TempDir()); err == nil {
		t.Fatal("expected an error for a directory")
	}
}

func TestFromFS(t *testing.T) {
	fsys := fstest.MapFS{
		"nested/report.pdf": &fstest.MapFile{Data: []byte("from-fs")},
	}
	in, err := FromFS(fsys, "nested/report.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if in.Name() != "report.pdf" {
		t.Fatalf("Name() = %q", in.Name())
	}
	if in.Size() != 7 {
		t.Fatalf("Size() = %d", in.Size())
	}
	rc, err := in.Open()
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(rc)
	_ = rc.Close()
	if string(data) != "from-fs" {
		t.Fatalf("read %q", data)
	}

	if _, err := FromFS(fsys, "missing.pdf"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected fs.ErrNotExist, got %v", err)
	}
}

func TestFromReaderIsOneShot(t *testing.T) {
	in := FromReader("stream.pdf", strings.NewReader("streamed"))
	if in.Size() != -1 {
		t.Fatalf("Size() = %d, want -1", in.Size())
	}
	rc, err := in.Open()
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(rc)
	_ = rc.Close()
	if string(data) != "streamed" {
		t.Fatalf("read %q", data)
	}
	if _, err := in.Open(); !errors.Is(err, ErrNotReopenable) {
		t.Fatalf("second Open() error = %v, want ErrNotReopenable", err)
	}
}

func TestFromReaderWithSize(t *testing.T) {
	in := FromReaderWithSize("stream.pdf", strings.NewReader("abc"), 3)
	if in.Size() != 3 {
		t.Fatalf("Size() = %d", in.Size())
	}
}

func TestFromFileID(t *testing.T) {
	in := FromFileID("file-123")
	got, ok := FileID(in)
	if !ok || got != "file-123" {
		t.Fatalf("FileID() = %q, %v", got, ok)
	}
	if in.Name() != "" {
		t.Fatalf("Name() = %q", in.Name())
	}
	if _, err := in.Open(); err == nil {
		t.Fatal("expected an error opening a server-side file reference")
	}
}
