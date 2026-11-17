// Package testutil contains helpers shared by the integration and end-to-end
// tests. It is not part of the public SDK.
package testutil

import (
	"bytes"
	"fmt"
	"strings"
)

// PDF builds a small, valid PDF with one page per text entry. The text is
// drawn with Helvetica so that analysis and text-extraction endpoints can find
// it.
func PDF(pages ...string) []byte {
	if len(pages) == 0 {
		pages = []string{""}
	}
	var objects []string
	pageIDs := make([]int, len(pages))
	for i := range pages {
		pageIDs[i] = 4 + 2*i
	}
	objects = append(objects, "<< /Type /Catalog /Pages 2 0 R >>")
	kids := make([]string, len(pageIDs))
	for i, id := range pageIDs {
		kids[i] = fmt.Sprintf("%d 0 R", id)
	}
	objects = append(objects, fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(pages)))
	objects = append(objects, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
	for i, text := range pages {
		contentID := pageIDs[i] + 1
		objects = append(objects, fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 3 0 R >> >> /Contents %d 0 R >>", contentID))
		stream := fmt.Sprintf("BT /F1 24 Tf 72 720 Td (%s) Tj ET", escapePDFText(text))
		objects = append(objects, fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream))
	}

	var buffer bytes.Buffer
	buffer.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, object := range objects {
		offsets[i+1] = buffer.Len()
		fmt.Fprintf(&buffer, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}
	xref := buffer.Len()
	fmt.Fprintf(&buffer, "xref\n0 %d\n", len(objects)+1)
	buffer.WriteString("0000000000 65535 f \n")
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&buffer, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&buffer, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return buffer.Bytes()
}

func escapePDFText(text string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`)
	return replacer.Replace(text)
}
