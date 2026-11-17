package stirling

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"
)

// APIError describes a non-2xx response from Stirling PDF.
//
// The Processing API returns JSON error bodies with the fields parsed here.
// Plain-text and HTML error bodies from proxies are preserved in Body and
// included (truncated) in the error message.
type APIError struct {
	// StatusCode is the HTTP status code, for example 422.
	StatusCode int

	// Status is the HTTP status text, for example "Unprocessable Entity".
	Status string

	// Method is the HTTP method of the failed request.
	Method string

	// URL is the full request URL.
	URL string

	// Code is the API's short error name: "error" in the Processing API body
	// or "title" in an RFC 7807 problem response.
	Code string

	// Message is the human-readable API message: "message" in the Processing
	// API body or "detail" in an RFC 7807 problem response.
	Message string

	// Timestamp and Path mirror the remaining fields of the API error body.
	Timestamp string
	Path      string

	// RequestID is the value of the X-Request-Id response header, when set.
	RequestID string

	// Header contains the response headers.
	Header http.Header

	// Body is the raw response body. It is capped at 64 KiB.
	Body []byte
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e == nil {
		return "<nil>"
	}
	var b strings.Builder
	b.WriteString("stirling: ")
	if e.Method != "" {
		b.WriteString(e.Method)
		b.WriteByte(' ')
	}
	if e.Path != "" {
		b.WriteString(e.Path)
	} else if e.URL != "" {
		b.WriteString(e.URL)
	}
	if e.Method != "" || e.Path != "" || e.URL != "" {
		b.WriteString(": ")
	}
	status := e.Status
	if status == "" {
		status = http.StatusText(e.StatusCode)
	}
	fmt.Fprintf(&b, "HTTP %d", e.StatusCode)
	if status != "" {
		b.WriteString(" ")
		b.WriteString(status)
	}
	if e.Message != "" {
		b.WriteString(": ")
		b.WriteString(e.Message)
	} else if e.Code != "" {
		b.WriteString(": ")
		b.WriteString(e.Code)
	} else if len(e.Body) > 0 {
		b.WriteString(": ")
		b.WriteString(truncate(strings.TrimSpace(string(e.Body)), 240))
	}
	if e.RequestID != "" {
		b.WriteString(" (request id ")
		b.WriteString(e.RequestID)
		b.WriteByte(')')
	}
	return b.String()
}

// StatusCode returns the HTTP status code carried by err, or 0 when err is not
// an [*APIError]. It unwraps wrapped errors.
func StatusCode(err error) int {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode
	}
	return 0
}

// IsNotFound reports whether err is an [*APIError] with status 404.
func IsNotFound(err error) bool { return StatusCode(err) == http.StatusNotFound }

func parseAPIError(method, url string, statusCode int, body []byte, header http.Header) error {
	if len(body) > 64<<10 {
		body = body[:64<<10]
	}
	apiErr := &APIError{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Method:     method,
		URL:        url,
		Header:     header,
		Body:       body,
	}
	if header != nil {
		apiErr.RequestID = header.Get("X-Request-Id")
	}

	var payload struct {
		Status    int    `json:"status"`
		Error     string `json:"error"`
		Title     string `json:"title"`
		Message   string `json:"message"`
		Detail    string `json:"detail"`
		Timestamp string `json:"timestamp"`
		Path      string `json:"path"`
	}
	if err := json.Unmarshal(body, &payload); err == nil {
		apiErr.Code = firstNonEmpty(payload.Error, payload.Title)
		apiErr.Message = firstNonEmpty(payload.Message, payload.Detail)
		apiErr.Timestamp = payload.Timestamp
		apiErr.Path = payload.Path
	}
	return apiErr
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func truncate(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	cut := n
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "…"
}
