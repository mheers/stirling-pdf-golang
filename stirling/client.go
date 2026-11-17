package stirling

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Version is the SDK version. Release builds may override it with
//
//	go build -ldflags "-X github.com/mheers/stirling-pdf-golang/stirling.Version=v1.2.3"
var Version = "0.1.0"

// DefaultBaseURL is the Stirling Cloud endpoint from the OpenAPI document.
// Point the client at your own server with [WithBaseURL] or STIRLING_URL.
const DefaultBaseURL = "https://api.stirling.com"

// RequestEditor mutates an outgoing HTTP request, for example to add tracing
// headers or credentials for a reverse proxy.
type RequestEditor func(*http.Request) error

// Option configures a [Client]. Options are applied in order.
type Option func(*Client) error

// Client talks to a Stirling PDF server.
//
// A Client is safe for concurrent use by multiple goroutines. Create one per
// server and reuse it.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	apiKey     string
	userAgent  string
	headers    http.Header
	retry      RetryPolicy
	editors    []RequestEditor
}

// New creates a client. With no options it targets the Stirling Cloud API at
// [DefaultBaseURL]; set an API key with [WithAPIKey] or STIRLING_API_KEY.
func New(opts ...Option) (*Client, error) {
	baseURL, err := url.Parse(DefaultBaseURL)
	if err != nil {
		return nil, err
	}
	c := &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Minute},
		userAgent:  "stirling-pdf-golang/" + Version,
		headers:    make(http.Header),
		retry: RetryPolicy{
			MaxRetries: 2,
			MinWait:    500 * time.Millisecond,
			MaxWait:    5 * time.Second,
		},
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// NewFromEnv creates a client from STIRLING_URL, STIRLING_API_KEY,
// STIRLING_TIMEOUT, and STIRLING_RETRIES. Explicit options are applied last and
// therefore take precedence over the environment.
func NewFromEnv(opts ...Option) (*Client, error) {
	envOpts := make([]Option, 0, 4)
	if value := os.Getenv("STIRLING_URL"); value != "" {
		envOpts = append(envOpts, WithBaseURL(value))
	}
	if value := os.Getenv("STIRLING_API_KEY"); value != "" {
		envOpts = append(envOpts, WithAPIKey(value))
	}
	if value := os.Getenv("STIRLING_TIMEOUT"); value != "" {
		timeout, err := time.ParseDuration(value)
		if err != nil {
			return nil, fmt.Errorf("stirling: STIRLING_TIMEOUT: %w", err)
		}
		envOpts = append(envOpts, WithTimeout(timeout))
	}
	if value := os.Getenv("STIRLING_RETRIES"); value != "" {
		retries, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("stirling: STIRLING_RETRIES: %w", err)
		}
		envOpts = append(envOpts, WithRetries(retries))
	}
	return New(append(envOpts, opts...)...)
}

// WithBaseURL sets the server root, for example "http://localhost:8080" or
// "https://example.com/pdf" when Stirling PDF runs behind a path prefix.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) error {
		parsed, err := url.Parse(strings.TrimSpace(baseURL))
		if err != nil {
			return fmt.Errorf("stirling: base URL: %w", err)
		}
		if parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("stirling: base URL %q must be absolute, for example http://localhost:8080", baseURL)
		}
		parsed.Path = strings.TrimSuffix(parsed.Path, "/")
		parsed.RawQuery = ""
		parsed.Fragment = ""
		c.baseURL = parsed
		return nil
	}
}

// WithAPIKey sets the X-API-KEY header sent with every request.
func WithAPIKey(key string) Option {
	return func(c *Client) error {
		c.apiKey = key
		return nil
	}
}

// WithHTTPClient replaces the underlying HTTP client. It controls transport,
// proxies, TLS, connection pooling, and timeouts.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) error {
		if client == nil {
			return errors.New("stirling: HTTP client must not be nil")
		}
		c.httpClient = client
		return nil
	}
}

// WithTimeout sets the overall request timeout. Use 0 for no timeout, which is
// sometimes useful for very large documents. The default is five minutes.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) error {
		clone := *c.httpClient
		clone.Timeout = timeout
		c.httpClient = &clone
		return nil
	}
}

// WithUserAgent overrides the User-Agent header.
func WithUserAgent(userAgent string) Option {
	return func(c *Client) error {
		c.userAgent = userAgent
		return nil
	}
}

// WithHeader adds a header to every request. It can be called repeatedly.
func WithHeader(name, value string) Option {
	return func(c *Client) error {
		c.headers.Set(name, value)
		return nil
	}
}

// WithRetries sets how many times a request is retried after a transport error
// or a 408, 429, 500, 502, 503, or 504 response. Requests containing one-shot
// readers are never retried.
func WithRetries(retries int) Option {
	return func(c *Client) error {
		if retries < 0 {
			retries = 0
		}
		c.retry.MaxRetries = retries
		return nil
	}
}

// WithRetryWait sets the exponential backoff bounds. Jitter is intentionally
// not applied so retries are predictable in tests and scripts.
func WithRetryWait(minWait, maxWait time.Duration) Option {
	return func(c *Client) error {
		if minWait < 0 || maxWait < 0 {
			return errors.New("stirling: retry waits must not be negative")
		}
		if maxWait < minWait {
			maxWait = minWait
		}
		c.retry.MinWait = minWait
		c.retry.MaxWait = maxWait
		return nil
	}
}

// WithRequestEditor registers a function that can modify every outgoing
// request. Editors run after the SDK sets its own headers.
func WithRequestEditor(editor RequestEditor) Option {
	return func(c *Client) error {
		c.editors = append(c.editors, editor)
		return nil
	}
}

// BaseURL returns the configured server root without a trailing slash.
func (c *Client) BaseURL() string { return c.baseURL.String() }

// HTTPClient returns the underlying HTTP client.
func (c *Client) HTTPClient() *http.Client { return c.httpClient }

// CloseIdleConnections closes idle connections held by the underlying HTTP
// client. It is useful before shutting down a long-running process.
func (c *Client) CloseIdleConnections() { c.httpClient.CloseIdleConnections() }

// Call performs a POST request to a Processing API path with multipart form
// parameters. It is the generic escape hatch behind the typed methods and is
// useful for endpoints that are not wrapped by the SDK yet.
func (c *Client) Call(ctx context.Context, path string, params ...Param) (*Result, error) {
	return c.CallMethod(ctx, http.MethodPost, path, params...)
}

// CallMethod is [Client.Call] with an explicit HTTP method. GET requests do not
// send a body.
func (c *Client) CallMethod(ctx context.Context, method, path string, params ...Param) (*Result, error) {
	if (method == http.MethodGet || method == http.MethodHead) && len(params) > 0 {
		return nil, fmt.Errorf("stirling: %s requests do not accept form parameters", method)
	}
	f := newForm()
	for _, param := range params {
		if err := f.add(param.Name, param.Value); err != nil {
			return nil, err
		}
	}
	resp, err := c.do(ctx, method, path, f)
	if err != nil {
		return nil, err
	}
	return newResult(resp), nil
}

// CallJSON performs a call and decodes a JSON object response. The Processing
// API describes analysis responses as free-form objects, so the result is a
// map. Use encoding/json on the raw body when a typed result is preferred.
func (c *Client) CallJSON(ctx context.Context, path string, params ...Param) (map[string]any, error) {
	res, err := c.Call(ctx, path, params...)
	if err != nil {
		return nil, err
	}
	body, err := res.Bytes()
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("stirling: %s: empty JSON response", path)
	}
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("stirling: %s: decode JSON: %w", path, err)
	}
	return result, nil
}

// CallText performs a call and returns the response body as a string.
func (c *Client) CallText(ctx context.Context, path string, params ...Param) (string, error) {
	res, err := c.Call(ctx, path, params...)
	if err != nil {
		return "", err
	}
	body, err := res.Bytes()
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (c *Client) do(ctx context.Context, method, path string, f *form) (*http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	target, err := c.resolve(path)
	if err != nil {
		return nil, err
	}

	attempts := c.retry.MaxRetries + 1
	if !f.reopenable() {
		attempts = 1
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		var body io.ReadCloser
		var contentType string
		if method != http.MethodGet && method != http.MethodHead {
			body, contentType, err = f.body()
			if err != nil {
				return nil, err
			}
		}
		req, err := http.NewRequestWithContext(ctx, method, target, body)
		if err != nil {
			if body != nil {
				_ = body.Close()
			}
			return nil, err
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		c.applyHeaders(req)
		for _, editor := range c.editors {
			if err := editor(req); err != nil {
				if body != nil {
					_ = body.Close()
				}
				return nil, fmt.Errorf("stirling: request editor: %w", err)
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("stirling: %s %s: %w", method, path, err)
			if !isRetryableTransportError(err) || attempt == attempts-1 {
				return nil, lastErr
			}
			if err := sleepContext(ctx, c.retry.delay(attempt, nil)); err != nil {
				return nil, err
			}
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}
		if isRetryableStatus(resp.StatusCode) && attempt < attempts-1 {
			wait := c.retry.delay(attempt, resp)
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 8<<10))
			_ = resp.Body.Close()
			if err := sleepContext(ctx, wait); err != nil {
				return nil, err
			}
			continue
		}
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		_ = resp.Body.Close()
		return nil, parseAPIError(method, target, resp.StatusCode, bodyBytes, resp.Header)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("stirling: %s %s: request failed", method, path)
	}
	return nil, lastErr
}

func (c *Client) applyHeaders(req *http.Request) {
	for name, values := range c.headers {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}
	if c.apiKey != "" {
		req.Header.Set("X-API-KEY", c.apiKey)
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "*/*")
	}
}

func (c *Client) resolve(apiPath string) (string, error) {
	relative, err := url.Parse(apiPath)
	if err != nil {
		return "", fmt.Errorf("stirling: invalid path %q: %w", apiPath, err)
	}
	joined, err := url.JoinPath(c.baseURL.String(), relative.Path)
	if err != nil {
		return "", fmt.Errorf("stirling: invalid path %q: %w", apiPath, err)
	}
	target, err := url.Parse(joined)
	if err != nil {
		return "", fmt.Errorf("stirling: invalid path %q: %w", apiPath, err)
	}
	target.RawQuery = relative.RawQuery
	return target.String(), nil
}

func isRetryableStatus(status int) bool {
	switch status {
	case http.StatusRequestTimeout, http.StatusTooEarly, http.StatusTooManyRequests,
		http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return false
}

func isRetryableTransportError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}
