package stirling

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
)

// Param is a single multipart form field for [Client.Call]. Most callers use
// the typed methods instead.
//
// Value may be a string, bool, integer, float, [time.Time], []string, []int,
// []int64, []float64, map[string]string, []map[string]any, [FileInput], or
// []FileInput. Nil values are skipped.
type Param struct {
	Name  string
	Value any
}

type form struct {
	fields []Param
}

func newForm() *form { return &form{} }

func (f *form) add(name string, value any) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case FileInput:
		if isNilFile(v) {
			return nil
		}
		if id, ok := FileID(v); ok {
			f.fields = append(f.fields, Param{Name: "fileId", Value: id})
			return nil
		}
		f.fields = append(f.fields, Param{Name: name, Value: v})
		return nil
	case []FileInput:
		for _, file := range v {
			if isNilFile(file) {
				return fmt.Errorf("stirling: form field %q contains a nil file", name)
			}
		}
		if len(v) > 0 {
			f.fields = append(f.fields, Param{Name: name, Value: v})
		}
		return nil
	}
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		return f.add(name, rv.Elem().Interface())
	}
	switch v := value.(type) {
	case string, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, json.Number, time.Time:
		f.fields = append(f.fields, Param{Name: name, Value: v})
	case []string:
		for _, item := range v {
			f.fields = append(f.fields, Param{Name: name, Value: item})
		}
	case []int:
		for _, item := range v {
			f.fields = append(f.fields, Param{Name: name, Value: item})
		}
	case []int32:
		for _, item := range v {
			f.fields = append(f.fields, Param{Name: name, Value: item})
		}
	case []int64:
		for _, item := range v {
			f.fields = append(f.fields, Param{Name: name, Value: item})
		}
	case []float64:
		for _, item := range v {
			f.fields = append(f.fields, Param{Name: name, Value: item})
		}
	case map[string]string:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			f.fields = append(f.fields, Param{Name: name + "[" + key + "]", Value: v[key]})
		}
	case []map[string]any:
		for i, object := range v {
			keys := make([]string, 0, len(object))
			for key := range object {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				field := fmt.Sprintf("%s[%d][%s]", name, i, key)
				f.fields = append(f.fields, Param{Name: field, Value: object[key]})
			}
		}
	default:
		return fmt.Errorf("stirling: form field %q: unsupported value type %T", name, value)
	}
	return nil
}

func (f *form) reopenable() bool {
	for _, field := range f.fields {
		switch v := field.Value.(type) {
		case FileInput:
			if !isReopenable(v) {
				return false
			}
		case []FileInput:
			for _, file := range v {
				if !isReopenable(file) {
					return false
				}
			}
		}
	}
	return true
}

func isReopenable(in FileInput) bool {
	if isNilFile(in) {
		return false
	}
	if _, ok := FileID(in); ok {
		return true
	}
	if _, ok := in.(*readerInput); ok {
		return false
	}
	return true
}

func (f *form) body() (io.ReadCloser, string, error) {
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	contentType := writer.FormDataContentType()
	go func() {
		err := f.write(writer)
		if closeErr := writer.Close(); err == nil {
			err = closeErr
		}
		_ = pw.CloseWithError(err)
	}()
	return &formBody{PipeReader: pr, writer: pw}, contentType, nil
}

// formBody closes both ends of the pipe. The HTTP transport may abandon a
// request body when the server responds early; closing the writer unblocks the
// multipart goroutine instead of leaking it.
type formBody struct {
	*io.PipeReader
	writer *io.PipeWriter
}

func (b *formBody) Close() error {
	_ = b.writer.CloseWithError(io.ErrClosedPipe)
	return b.PipeReader.Close()
}

func (f *form) write(writer *multipart.Writer) error {
	for _, field := range f.fields {
		switch value := field.Value.(type) {
		case FileInput:
			if err := writeFile(writer, field.Name, value); err != nil {
				return err
			}
		case []FileInput:
			for _, file := range value {
				if err := writeFile(writer, field.Name, file); err != nil {
					return err
				}
			}
		default:
			text, err := scalarString(value)
			if err != nil {
				return fmt.Errorf("stirling: form field %q: %w", field.Name, err)
			}
			if err := writer.WriteField(field.Name, text); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeFile(writer *multipart.Writer, field string, in FileInput) error {
	if isNilFile(in) {
		return fmt.Errorf("stirling: form field %q has a nil file", field)
	}
	reader, err := in.Open()
	if err != nil {
		return fmt.Errorf("stirling: open %q: %w", in.Name(), err)
	}
	defer func() { _ = reader.Close() }()

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, escapeQuotes(field), escapeQuotes(in.Name())))
	header.Set("Content-Type", fileContentType(in.Name()))
	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, reader); err != nil {
		return fmt.Errorf("stirling: upload %q: %w", in.Name(), err)
	}
	return nil
}

func fileContentType(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if contentType := mime.TypeByExtension(ext); contentType != "" {
		return contentType
	}
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".eml":
		return "message/rfc822"
	}
	return "application/octet-stream"
}

func escapeQuotes(s string) string {
	return strings.NewReplacer("\\", "\\\\", `"`, "\\\"").Replace(s)
}

func scalarString(value any) (string, error) {
	switch v := value.(type) {
	case nil:
		return "", nil
	case string:
		return v, nil
	case bool:
		if v {
			return "true", nil
		}
		return "false", nil
	case int:
		return fmt.Sprintf("%d", v), nil
	case int8:
		return fmt.Sprintf("%d", v), nil
	case int16:
		return fmt.Sprintf("%d", v), nil
	case int32:
		return fmt.Sprintf("%d", v), nil
	case int64:
		return fmt.Sprintf("%d", v), nil
	case uint:
		return fmt.Sprintf("%d", v), nil
	case uint8:
		return fmt.Sprintf("%d", v), nil
	case uint16:
		return fmt.Sprintf("%d", v), nil
	case uint32:
		return fmt.Sprintf("%d", v), nil
	case uint64:
		return fmt.Sprintf("%d", v), nil
	case float32:
		return fmt.Sprintf("%v", v), nil
	case float64:
		return fmt.Sprintf("%v", v), nil
	case json.Number:
		return v.String(), nil
	case time.Time:
		return v.Format(time.RFC3339), nil
	case []string:
		return strings.Join(v, ","), nil
	default:
		return "", fmt.Errorf("unsupported value type %T", value)
	}
}
