package stirling

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
)

// ErrNotReopenable is returned by a one-shot [FileInput] when it is opened more
// than once. Streaming inputs cannot be retried safely, so the client disables
// automatic retries for requests that contain them.
var ErrNotReopenable = errors.New("stirling: input cannot be opened more than once")

// FileInput is a document that can be uploaded to Stirling PDF.
//
// Implementations are provided for byte slices, files on disk, files in an
// [fs.FS], and one-shot readers. A FileInput can also refer to a file that
// already exists on the server by using [FromFileID].
type FileInput interface {
	// Name is the filename presented to the server. It may be empty for
	// server-side file references.
	Name() string

	// Size is the number of bytes in the input, or -1 when unknown.
	Size() int64

	// Open returns a fresh reader for the input. Implementations backed by
	// bytes, a path, or an fs.FS can be opened repeatedly; reader-backed
	// inputs return [ErrNotReopenable] after the first call.
	Open() (io.ReadCloser, error)
}

type bytesInput struct {
	name string
	data []byte
}

func (b bytesInput) Name() string { return b.name }
func (b bytesInput) Size() int64  { return int64(len(b.data)) }
func (b bytesInput) Open() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(b.data)), nil
}

// FromBytes returns a FileInput backed by an in-memory byte slice. It is safe
// to open repeatedly and is therefore retry-friendly.
func FromBytes(name string, data []byte) FileInput {
	return bytesInput{name: name, data: data}
}

// FromString is like [FromBytes] for string content.
func FromString(name, content string) FileInput {
	return bytesInput{name: name, data: []byte(content)}
}

type pathInput struct {
	path string
	size int64
}

func (p pathInput) Name() string { return filepath.Base(p.path) }
func (p pathInput) Size() int64  { return p.size }
func (p pathInput) Open() (io.ReadCloser, error) {
	return os.Open(p.path)
}

// FromPath opens a file from the local filesystem lazily. The file is not held
// open between requests, which makes this input retry-friendly and suitable for
// large documents.
func FromPath(filename string) (FileInput, error) {
	info, err := os.Stat(filename)
	if err != nil {
		return nil, fmt.Errorf("stirling: open input: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("stirling: open input %q: is a directory", filename)
	}
	return pathInput{path: filename, size: info.Size()}, nil
}

type fsInput struct {
	fsys fs.FS
	name string
	size int64
}

func (f fsInput) Name() string { return path.Base(f.name) }
func (f fsInput) Size() int64  { return f.size }
func (f fsInput) Open() (io.ReadCloser, error) {
	return f.fsys.Open(f.name)
}

// FromFS returns a FileInput for a file inside an [fs.FS]. The base name of
// name is used as the upload filename.
func FromFS(fsys fs.FS, name string) (FileInput, error) {
	info, err := fs.Stat(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("stirling: open input: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("stirling: open input %q: is a directory", name)
	}
	return fsInput{fsys: fsys, name: name, size: info.Size()}, nil
}

type readerInput struct {
	name   string
	reader io.Reader
	size   int64
	used   bool
}

func (r *readerInput) Name() string { return r.name }
func (r *readerInput) Size() int64  { return r.size }
func (r *readerInput) Open() (io.ReadCloser, error) {
	if r.used {
		return nil, ErrNotReopenable
	}
	r.used = true
	return io.NopCloser(r.reader), nil
}

// FromReader wraps a streaming reader. The input can only be opened once, so
// requests containing it are never retried. Use [FromBytes] or [FromPath] when
// retries matter.
func FromReader(name string, reader io.Reader) FileInput {
	return &readerInput{name: name, reader: reader, size: -1}
}

// FromReaderWithSize is [FromReader] with a known size. The size is advisory
// and is not enforced while reading.
func FromReaderWithSize(name string, reader io.Reader, size int64) FileInput {
	return &readerInput{name: name, reader: reader, size: size}
}

type fileIDInput struct {
	id string
}

func (f fileIDInput) Name() string { return "" }
func (f fileIDInput) Size() int64  { return 0 }
func (f fileIDInput) Open() (io.ReadCloser, error) {
	return nil, errors.New("stirling: server-side file references cannot be opened")
}

// FromFileID references a file that already exists on the server. The value is
// sent as the fileId form field instead of a file upload. This is useful with
// Stirling PDF deployments that expose server-side file storage.
func FromFileID(id string) FileInput {
	return fileIDInput{id: id}
}

// FileID reports whether in is a server-side file reference and, if so,
// returns its identifier.
func FileID(in FileInput) (string, bool) {
	if id, ok := in.(fileIDInput); ok {
		return id.id, true
	}
	return "", false
}

func isNilFile(in FileInput) bool {
	if in == nil {
		return true
	}
	value := reflect.ValueOf(in)
	return value.Kind() == reflect.Pointer && value.IsNil()
}
