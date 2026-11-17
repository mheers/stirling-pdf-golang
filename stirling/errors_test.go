package stirling

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestAPIErrorFromJSON(t *testing.T) {
	body := []byte(`{
		"timestamp": "2026-11-17T10:30:00Z",
		"path": "/api/v1/misc/compress-pdf",
		"status": 422,
		"message": "File is not a PDF",
		"error": "Unprocessable Entity"
	}`)

	err := parseAPIError(http.MethodPost, "https://example.test/api/v1/misc/compress-pdf", 422, body, http.Header{"X-Request-Id": []string{"req-1"}})

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error %T is not *APIError", err)
	}
	if apiErr.StatusCode != 422 {
		t.Errorf("StatusCode = %d", apiErr.StatusCode)
	}
	if apiErr.Message != "File is not a PDF" {
		t.Errorf("Message = %q", apiErr.Message)
	}
	if apiErr.Code != "Unprocessable Entity" {
		t.Errorf("Code = %q", apiErr.Code)
	}
	if apiErr.Path != "/api/v1/misc/compress-pdf" {
		t.Errorf("Path = %q", apiErr.Path)
	}
	if apiErr.RequestID != "req-1" {
		t.Errorf("RequestID = %q", apiErr.RequestID)
	}
	msg := err.Error()
	for _, want := range []string{"422", "Unprocessable Entity", "File is not a PDF"} {
		if !strings.Contains(msg, want) {
			t.Errorf("Error() = %q, missing %q", msg, want)
		}
	}
	if StatusCode(err) != 422 {
		t.Errorf("StatusCode(err) = %d", StatusCode(err))
	}
}

func TestAPIErrorFromPlainText(t *testing.T) {
	err := parseAPIError(http.MethodPost, "https://example.test/x", 502, []byte("bad gateway"), nil)
	if !strings.Contains(err.Error(), "bad gateway") {
		t.Fatalf("Error() = %q", err.Error())
	}
}

func TestAPIErrorFromProblemJSON(t *testing.T) {
	body := []byte(`{
		"detail": "An unexpected error occurred. Please try again later.",
		"instance": "/api/v1/security/redact",
		"status": 500,
		"title": "Internal Server Error",
		"type": "/errors/unexpected"
	}`)
	err := parseAPIError(http.MethodPost, "https://example.test/api/v1/security/redact", 500, body, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error %T is not *APIError", err)
	}
	if apiErr.Code != "Internal Server Error" {
		t.Errorf("Code = %q", apiErr.Code)
	}
	if apiErr.Message != "An unexpected error occurred. Please try again later." {
		t.Errorf("Message = %q", apiErr.Message)
	}
	if got := err.Error(); !strings.Contains(got, "Internal Server Error") || !strings.Contains(got, "unexpected error") {
		t.Errorf("Error() = %q", got)
	}
}

func TestTruncateKeepsValidUTF8(t *testing.T) {
	body := strings.Repeat("é", 400)
	got := truncate(body, 241)
	if !utf8.ValidString(got) {
		t.Fatalf("truncate produced invalid UTF-8: %q", got)
	}
}

func TestAPIErrorTruncatesLargeBodies(t *testing.T) {
	body := strings.Repeat("x", 4096)
	err := parseAPIError(http.MethodPost, "https://example.test/x", 500, []byte(body), nil)
	if len(err.Error()) > 1000 {
		t.Fatalf("Error() is %d bytes, want a truncated message", len(err.Error()))
	}
}

func TestStatusCodeHelpers(t *testing.T) {
	if got := StatusCode(nil); got != 0 {
		t.Fatalf("StatusCode(nil) = %d", got)
	}
	if got := StatusCode(fmt.Errorf("plain")); got != 0 {
		t.Fatalf("StatusCode(plain) = %d", got)
	}
	wrapped := fmt.Errorf("wrapped: %w", &APIError{StatusCode: 404})
	if got := StatusCode(wrapped); got != 404 {
		t.Fatalf("StatusCode(wrapped) = %d", got)
	}
	if !IsNotFound(wrapped) {
		t.Fatal("IsNotFound() = false")
	}
	if IsNotFound(&APIError{StatusCode: 500}) {
		t.Fatal("IsNotFound(500) = true")
	}
}

func TestAPIErrorEmptyMessage(t *testing.T) {
	err := parseAPIError(http.MethodGet, "https://example.test/x", 503, nil, nil)
	if !strings.Contains(err.Error(), "503") {
		t.Fatalf("Error() = %q", err.Error())
	}
}
