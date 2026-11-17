package stirling

import (
	"io"
	"mime"
	"mime/multipart"
	"strings"
	"testing"
)

func parseMultipart(t *testing.T, body io.Reader, contentType string) (*multipart.Reader, map[string][]string, map[string][]filePart) {
	t.Helper()
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		t.Fatalf("ParseMediaType: %v", err)
	}
	if mediaType != "multipart/form-data" {
		t.Fatalf("content type = %q", mediaType)
	}
	mr := multipart.NewReader(body, params["boundary"])
	values := map[string][]string{}
	files := map[string][]filePart{}
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(part)
		if err != nil {
			t.Fatal(err)
		}
		if part.FileName() != "" {
			files[part.FormName()] = append(files[part.FormName()], filePart{
				name:        part.FileName(),
				contentType: part.Header.Get("Content-Type"),
				data:        string(data),
			})
			continue
		}
		values[part.FormName()] = append(values[part.FormName()], string(data))
	}
	return mr, values, files
}

type filePart struct {
	name        string
	contentType string
	data        string
}

func TestFormScalars(t *testing.T) {
	f := newForm()
	for _, p := range []Param{
		{Name: "text", Value: "hello"},
		{Name: "yes", Value: true},
		{Name: "no", Value: false},
		{Name: "count", Value: 7},
		{Name: "big", Value: int64(9000)},
		{Name: "ratio", Value: 1.5},
		{Name: "hf", Value: float32(2.5)},
	} {
		if err := f.add(p.Name, p.Value); err != nil {
			t.Fatal(err)
		}
	}
	body, ct, err := f.body()
	if err != nil {
		t.Fatal(err)
	}
	_, values, files := parseMultipart(t, body, ct)
	if len(files) != 0 {
		t.Fatalf("unexpected files: %v", files)
	}
	want := map[string]string{
		"text":  "hello",
		"yes":   "true",
		"no":    "false",
		"count": "7",
		"big":   "9000",
		"ratio": "1.5",
		"hf":    "2.5",
	}
	for k, v := range want {
		if len(values[k]) != 1 || values[k][0] != v {
			t.Errorf("%s = %v, want %q", k, values[k], v)
		}
	}
}

func TestFormSkipsNils(t *testing.T) {
	f := newForm()
	if err := f.add("a", nil); err != nil {
		t.Fatal(err)
	}
	if err := f.add("b", (*string)(nil)); err != nil {
		t.Fatal(err)
	}
	body, ct, err := f.body()
	if err != nil {
		t.Fatal(err)
	}
	_, values, _ := parseMultipart(t, body, ct)
	if len(values) != 0 {
		t.Fatalf("values = %v", values)
	}
}

func TestFormSlicesAndMaps(t *testing.T) {
	f := newForm()
	_ = f.add("languages", []string{"eng", "deu"})
	_ = f.add("counts", []int{2, 3})
	_ = f.add("custom", map[string]string{"b": "2", "a": "1"})
	_ = f.add("redactions", []map[string]any{
		{"x": 1.5, "page": 2, "color": "#ff0000"},
		{"x": 3, "page": 4, "color": ""},
	})
	body, ct, err := f.body()
	if err != nil {
		t.Fatal(err)
	}
	_, values, _ := parseMultipart(t, body, ct)
	if got := values["languages"]; len(got) != 2 || got[0] != "eng" || got[1] != "deu" {
		t.Errorf("languages = %v", got)
	}
	if got := values["counts"]; len(got) != 2 || got[0] != "2" || got[1] != "3" {
		t.Errorf("counts = %v", got)
	}
	if got := values["custom[a]"]; len(got) != 1 || got[0] != "1" {
		t.Errorf("custom[a] = %v", got)
	}
	if got := values["redactions[0].x"]; len(got) != 1 || got[0] != "1.5" {
		t.Errorf("redactions[0].x = %v", got)
	}
	if got := values["redactions[0].page"]; len(got) != 1 || got[0] != "2" {
		t.Errorf("redactions[0].page = %v", got)
	}
	if got := values["redactions[1].color"]; len(got) != 1 || got[0] != "" {
		t.Errorf("redactions[1].color = %v", got)
	}
}

func TestFormFiles(t *testing.T) {
	f := newForm()
	if err := f.add("fileInput", FromBytes("doc.pdf", []byte("%PDF-1.7"))); err != nil {
		t.Fatal(err)
	}
	if err := f.add("attachments", []FileInput{
		FromBytes("a.txt", []byte("a")),
		FromBytes("b.csv", []byte("b")),
	}); err != nil {
		t.Fatal(err)
	}
	body, ct, err := f.body()
	if err != nil {
		t.Fatal(err)
	}
	_, _, files := parseMultipart(t, body, ct)
	pdfs := files["fileInput"]
	if len(pdfs) != 1 || pdfs[0].name != "doc.pdf" || pdfs[0].data != "%PDF-1.7" {
		t.Fatalf("fileInput = %+v", pdfs)
	}
	if pdfs[0].contentType != "application/pdf" {
		t.Errorf("pdf content type = %q", pdfs[0].contentType)
	}
	atts := files["attachments"]
	if len(atts) != 2 || atts[0].name != "a.txt" || atts[1].name != "b.csv" {
		t.Fatalf("attachments = %+v", atts)
	}
}

func TestFormUnsupportedType(t *testing.T) {
	err := newForm().add("bad", struct{ X int }{1})
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("error = %v", err)
	}
}

func TestFormReopenable(t *testing.T) {
	reopenable := newForm()
	_ = reopenable.add("fileInput", FromBytes("a.pdf", nil))
	if !reopenable.reopenable() {
		t.Fatal("bytes input should be reopenable")
	}

	oneShot := newForm()
	_ = oneShot.add("fileInput", FromReader("a.pdf", strings.NewReader("x")))
	if oneShot.reopenable() {
		t.Fatal("reader input should not be reopenable")
	}

	byID := newForm()
	_ = byID.add("fileInput", FromFileID("abc"))
	if !byID.reopenable() {
		t.Fatal("file ID input should be reopenable")
	}
}

func TestFormBodyTwice(t *testing.T) {
	f := newForm()
	_ = f.add("fileInput", FromBytes("a.pdf", []byte("data")))
	for i := 0; i < 2; i++ {
		body, ct, err := f.body()
		if err != nil {
			t.Fatal(err)
		}
		_, values, files := parseMultipart(t, body, ct)
		if len(values) != 0 || len(files["fileInput"]) != 1 {
			t.Fatalf("attempt %d: values=%v files=%v", i, values, files)
		}
	}
}

func TestFormFileID(t *testing.T) {
	f := newForm()
	if err := f.add("fileInput", FromFileID("abc")); err != nil {
		t.Fatal(err)
	}
	body, ct, err := f.body()
	if err != nil {
		t.Fatal(err)
	}
	_, values, files := parseMultipart(t, body, ct)
	if got := values["fileId"]; len(got) != 1 || got[0] != "abc" {
		t.Fatalf("fileId = %v", got)
	}
	if len(files) != 0 {
		t.Fatalf("unexpected files: %v", files)
	}

	if err := newForm().add("stampImage", FromFileID("abc")); err == nil {
		t.Fatal("expected an error for a file ID in a non-primary file field")
	}
	if err := newForm().add("attachments", []FileInput{FromFileID("abc")}); err == nil {
		t.Fatal("expected an error for a file ID in a file list")
	}
}

func TestFormSanitizesUploadHeaders(t *testing.T) {
	f := newForm()
	name := "evil\r\nX-Injected: yes\".pdf"
	if err := f.add("fileInput", FromBytes(name, []byte("data"))); err != nil {
		t.Fatal(err)
	}
	body, ct, err := f.body()
	if err != nil {
		t.Fatal(err)
	}
	_, values, files := parseMultipart(t, body, ct)
	parts := files["fileInput"]
	if len(parts) != 1 {
		t.Fatalf("files = %v", files)
	}
	if parts[0].name != `evil  X-Injected: yes\".pdf` {
		t.Fatalf("filename = %q", parts[0].name)
	}
	if len(values) != 0 {
		t.Fatalf("injected fields: %v", values)
	}
	if len(files) != 1 {
		t.Fatalf("injected parts: %v", files)
	}
}
