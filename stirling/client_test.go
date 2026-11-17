package stirling

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewDefaults(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if got := c.BaseURL(); got != "https://api.stirling.com" {
		t.Fatalf("BaseURL() = %q", got)
	}
	if c.HTTPClient() == nil {
		t.Fatal("HTTPClient() is nil")
	}
}

func TestWithBaseURL(t *testing.T) {
	c, err := New(WithBaseURL("http://localhost:8080/"))
	if err != nil {
		t.Fatal(err)
	}
	if got := c.BaseURL(); got != "http://localhost:8080" {
		t.Fatalf("BaseURL() = %q", got)
	}

	c, err = New(WithBaseURL("http://localhost:8080/pdf/"))
	if err != nil {
		t.Fatal(err)
	}
	if got := c.BaseURL(); got != "http://localhost:8080/pdf" {
		t.Fatalf("BaseURL() = %q", got)
	}

	if _, err := New(WithBaseURL("://bad")); err == nil {
		t.Fatal("expected an error for an invalid base URL")
	}
}

func TestNewFromEnv(t *testing.T) {
	t.Setenv("STIRLING_URL", "http://stirling.test:9000/base")
	t.Setenv("STIRLING_API_KEY", "secret")
	c, err := NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if c.BaseURL() != "http://stirling.test:9000/base" {
		t.Fatalf("BaseURL() = %q", c.BaseURL())
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-API-KEY"); got != "secret" {
			t.Errorf("X-API-KEY = %q", got)
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()
	c, _ = NewFromEnv(WithBaseURL(srv.URL))
	if _, err := c.Call(context.Background(), "/api/v1/test"); err != nil {
		t.Fatal(err)
	}
}

func TestRequestHeadersAndSubpath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pdf/api/v1/general/merge-pdfs" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("X-API-KEY"); got != "key-1" {
			t.Errorf("X-API-KEY = %q", got)
		}
		if got := r.Header.Get("User-Agent"); !strings.HasPrefix(got, "stirling-pdf-golang/") {
			t.Errorf("User-Agent = %q", got)
		}
		if got := r.Header.Get("X-Custom"); got != "yes" {
			t.Errorf("X-Custom = %q", got)
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c, err := New(
		WithBaseURL(srv.URL+"/pdf"),
		WithAPIKey("key-1"),
		WithHeader("X-Custom", "yes"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Call(context.Background(), "/api/v1/general/merge-pdfs"); err != nil {
		t.Fatal(err)
	}
}

func TestCallReturnsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"status":422,"error":"Unprocessable Entity","message":"nope"}`))
	}))
	defer srv.Close()

	c, _ := New(WithBaseURL(srv.URL), WithRetries(0))
	_, err := c.Call(context.Background(), "/x")
	if err == nil {
		t.Fatal("expected an error")
	}
	if StatusCode(err) != 422 {
		t.Fatalf("StatusCode = %d", StatusCode(err))
	}
	if !strings.Contains(err.Error(), "nope") {
		t.Fatalf("error = %v", err)
	}
}

func TestCallRetriesServerErrors(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("third time"))
	}))
	defer srv.Close()

	c, _ := New(WithBaseURL(srv.URL), WithRetries(2), WithRetryWait(0, time.Millisecond))
	res, err := c.Call(context.Background(), "/x")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := res.Bytes()
	if string(data) != "third time" {
		t.Fatalf("body = %q", data)
	}
	if got := attempts.Load(); got != 3 {
		t.Fatalf("attempts = %d", got)
	}
}

func TestCallHonorsRetryAfter(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c, _ := New(WithBaseURL(srv.URL), WithRetries(1), WithRetryWait(0, time.Millisecond))
	start := time.Now()
	res, err := c.Call(context.Background(), "/x")
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Close()
	if elapsed := time.Since(start); elapsed < 900*time.Millisecond {
		t.Fatalf("retry did not honor Retry-After (elapsed %v)", elapsed)
	}
}

func TestCallDoesNotRetryWithOneShotInput(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c, _ := New(WithBaseURL(srv.URL), WithRetries(3), WithRetryWait(0, time.Millisecond))
	_, err := c.Call(context.Background(), "/x", Param{Name: "fileInput", Value: FromReader("x.pdf", strings.NewReader("x"))})
	if err == nil {
		t.Fatal("expected an error")
	}
	if got := attempts.Load(); got != 1 {
		t.Fatalf("attempts = %d, want 1", got)
	}
}

func TestCallRetriesTransportError(t *testing.T) {
	var attempts atomic.Int32
	c, _ := New(
		WithBaseURL("http://127.0.0.1:1"),
		WithRetries(1),
		WithRetryWait(0, time.Millisecond),
	)
	c.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if attempts.Add(1) == 1 {
			return nil, errors.New("temporary dial error")
		}
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{},
			Body:       io.NopCloser(bytes.NewBufferString("ok")),
			Request:    r,
		}, nil
	})
	res, err := c.Call(context.Background(), "/x")
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Close()
	if got := attempts.Load(); got != 2 {
		t.Fatalf("attempts = %d", got)
	}
}

func TestCallRequestEditor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Edited"); got != "1" {
			t.Errorf("X-Edited = %q", got)
		}
	}))
	defer srv.Close()

	c, _ := New(
		WithBaseURL(srv.URL),
		WithRequestEditor(func(r *http.Request) error {
			r.Header.Set("X-Edited", "1")
			return nil
		}),
	)
	if _, err := c.Call(context.Background(), "/x"); err != nil {
		t.Fatal(err)
	}
}

func TestCallJSONAndText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"page_count":3}`))
		case "/text":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("hello"))
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	c, _ := New(WithBaseURL(srv.URL))
	obj, err := c.CallJSON(context.Background(), "/json")
	if err != nil {
		t.Fatal(err)
	}
	if obj["page_count"].(float64) != 3 {
		t.Fatalf("json = %#v", obj)
	}

	text, err := c.CallText(context.Background(), "/text")
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello" {
		t.Fatalf("text = %q", text)
	}
}

func TestCallJSONRawAndEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c, _ := New(WithBaseURL(srv.URL))
	res, err := c.Call(context.Background(), "/empty")
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsEmpty() {
		t.Fatal("expected an empty result")
	}
	if _, err := c.CallJSON(context.Background(), "/empty"); err == nil {
		t.Fatal("expected an error decoding an empty JSON response")
	}
}

func TestCallContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	c, _ := New(WithBaseURL(srv.URL))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := c.Call(ctx, "/slow"); err == nil {
		t.Fatal("expected a context error")
	}
}

func TestCallUnsupportedParam(t *testing.T) {
	c, _ := New(WithBaseURL("http://127.0.0.1:1"))
	if _, err := c.Call(context.Background(), "/x", Param{Name: "bad", Value: make(chan int)}); err == nil {
		t.Fatal("expected an encoding error")
	}
}

func TestWithTimeout(t *testing.T) {
	c, _ := New(WithTimeout(1234 * time.Millisecond))
	if got := c.HTTPClient().Timeout; got != 1234*time.Millisecond {
		t.Fatalf("timeout = %v", got)
	}
}

func TestWithRetriesNegative(t *testing.T) {
	c, err := New(WithRetries(-3))
	if err != nil {
		t.Fatal(err)
	}
	if c.retry.MaxRetries != 0 {
		t.Fatalf("max retries = %d", c.retry.MaxRetries)
	}
}

func TestCallSetsAccept(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != "*/*" {
			t.Errorf("Accept = %q", got)
		}
	}))
	defer srv.Close()
	c, _ := New(WithBaseURL(srv.URL))
	if _, err := c.Call(context.Background(), "/x"); err != nil {
		t.Fatal(err)
	}
}

func TestCallQueryPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/general/job/123" || r.URL.RawQuery != "async=true" {
			t.Errorf("url = %s", r.URL.String())
		}
	}))
	defer srv.Close()
	c, _ := New(WithBaseURL(srv.URL))
	if _, err := c.Call(context.Background(), "/api/v1/general/job/123?async=true"); err != nil {
		t.Fatal(err)
	}
}

func TestCallMethodRejectsGETParams(t *testing.T) {
	c, _ := New(WithBaseURL("http://127.0.0.1:1"))
	if _, err := c.CallMethod(context.Background(), http.MethodGet, "/x", Param{Name: "a", Value: "b"}); err == nil {
		t.Fatal("expected an error")
	}
}

func TestCallJSONInvalid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()
	c, _ := New(WithBaseURL(srv.URL))
	if _, err := c.CallJSON(context.Background(), "/x"); err == nil {
		t.Fatal("expected a JSON error")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
