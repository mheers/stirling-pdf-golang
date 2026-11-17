package stirling

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Result is a successful Stirling PDF response.
//
// Responses can be binary documents, ZIP archives, images, plain text, or
// JSON. Body is streamed; call [Result.Close] when done, or use [Result.Bytes],
// [Result.Save], or [Result.SaveTo], which consume and close it.
type Result struct {
	// StatusCode is the HTTP status code. A filter operation that does not
	// match returns 204 and an empty body.
	StatusCode int

	// Header contains the response headers.
	Header http.Header

	// ContentType is the media type without parameters, for example
	// "application/pdf".
	ContentType string

	// Body is the response body stream.
	Body io.ReadCloser
}

func newResult(resp *http.Response) *Result {
	contentType := resp.Header.Get("Content-Type")
	if mediaType, _, err := mime.ParseMediaType(contentType); err == nil {
		contentType = mediaType
	}
	return &Result{
		StatusCode:  resp.StatusCode,
		Header:      resp.Header,
		ContentType: contentType,
		Body:        resp.Body,
	}
}

// Bytes reads the complete body and closes it.
func (r *Result) Bytes() ([]byte, error) {
	if r == nil || r.Body == nil {
		return nil, nil
	}
	defer func() { _ = r.Body.Close() }()
	return io.ReadAll(r.Body)
}

// Save streams the body to path and closes it. Existing files are truncated.
func (r *Result) Save(path string) error {
	if path == "" {
		return errors.New("stirling: save: empty path")
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := r.SaveTo(file); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

// SaveTo streams the body into w without closing it.
func (r *Result) SaveTo(w io.Writer) (int64, error) {
	if r == nil || r.Body == nil {
		return 0, nil
	}
	return io.Copy(w, r.Body)
}

// Close closes the response body. It is safe to call on a nil result.
func (r *Result) Close() error {
	if r == nil || r.Body == nil {
		return nil
	}
	return r.Body.Close()
}

// IsEmpty reports whether the server returned 204 No Content, which the
// filter endpoints use to say "the document does not match".
func (r *Result) IsEmpty() bool {
	return r == nil || r.StatusCode == http.StatusNoContent
}

// Filename returns the filename advertised in the Content-Disposition header,
// or "" when the server did not provide one.
func (r *Result) Filename() string {
	if r == nil || r.Header == nil {
		return ""
	}
	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Disposition"))
	if err != nil {
		return ""
	}
	return params["filename"]
}

// Kind classifies the response for display and default output naming.
func (r *Result) Kind() string {
	if r == nil {
		return "binary"
	}
	if r.Filename() != "" {
		if kind := kindFromExtension(filepath.Ext(r.Filename())); kind != "" {
			return kind
		}
	}
	switch r.ContentType {
	case "application/pdf":
		return "pdf"
	case "application/zip", "application/x-zip-compressed":
		return "zip"
	case "image/png":
		return "png"
	case "image/jpeg":
		return "jpeg"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	case "application/json":
		return "json"
	case "text/plain":
		return "text"
	case "text/markdown", "text/x-markdown":
		return "markdown"
	case "text/html":
		return "html"
	case "text/csv":
		return "csv"
	case "application/xml", "text/xml":
		return "xml"
	case "application/rtf", "text/rtf":
		return "rtf"
	case "application/msword":
		return "doc"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return "docx"
	case "application/vnd.oasis.opendocument.text":
		return "odt"
	case "application/vnd.ms-powerpoint":
		return "ppt"
	case "application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return "pptx"
	case "application/vnd.oasis.opendocument.presentation":
		return "odp"
	}
	if strings.HasPrefix(r.ContentType, "text/") {
		return "text"
	}
	return "binary"
}

// Extension returns a conventional file extension for the response, including
// the leading dot, or "" when unknown.
func (r *Result) Extension() string {
	if r == nil {
		return ""
	}
	if name := r.Filename(); name != "" {
		if ext := filepath.Ext(name); ext != "" {
			return ext
		}
	}
	switch r.Kind() {
	case "pdf":
		return ".pdf"
	case "zip":
		return ".zip"
	case "png":
		return ".png"
	case "jpeg":
		return ".jpg"
	case "gif":
		return ".gif"
	case "webp":
		return ".webp"
	case "json":
		return ".json"
	case "text":
		return ".txt"
	case "markdown":
		return ".md"
	case "html":
		return ".html"
	case "csv":
		return ".csv"
	case "xml":
		return ".xml"
	case "rtf":
		return ".rtf"
	case "doc":
		return ".doc"
	case "docx":
		return ".docx"
	case "odt":
		return ".odt"
	case "ppt":
		return ".ppt"
	case "pptx":
		return ".pptx"
	case "odp":
		return ".odp"
	}
	return ""
}

func kindFromExtension(ext string) string {
	switch strings.ToLower(ext) {
	case ".pdf":
		return "pdf"
	case ".zip":
		return "zip"
	case ".png":
		return "png"
	case ".jpg", ".jpeg":
		return "jpeg"
	case ".gif":
		return "gif"
	case ".webp":
		return "webp"
	case ".json":
		return "json"
	case ".txt":
		return "text"
	case ".md", ".markdown":
		return "markdown"
	case ".html", ".htm":
		return "html"
	case ".csv":
		return "csv"
	case ".xml":
		return "xml"
	case ".rtf":
		return "rtf"
	case ".doc":
		return "doc"
	case ".docx":
		return "docx"
	case ".ppt":
		return "ppt"
	case ".pptx":
		return "pptx"
	case ".odt":
		return "odt"
	case ".odp":
		return "odp"
	}
	return ""
}
