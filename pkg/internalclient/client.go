package internalclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkm-asia/pmb-backend/pkg/requestid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
)

// Header names
const (
	HeaderInternalToken = "X-Internal-Token"
	HeaderRequestID     = "X-Request-ID"
)

// Config konfigurasi internal HTTP client
type Config struct {
	BaseURL       string
	InternalToken string
	Timeout       time.Duration
	MaxRetries    int
	RetryDelay    time.Duration
}

// Error representasi error dari internal API call
type Error struct {
	StatusCode int                 `json:"status_code"`
	ErrorCode  string              `json:"error_code"`
	Message    string              `json:"message"`
	Errors     map[string][]string `json:"errors,omitempty"`
	RequestID  string              `json:"request_id,omitempty"`
}

func (e *Error) Error() string {
	if e.ErrorCode != "" {
		return fmt.Sprintf("internalclient: HTTP %d [%s] %s", e.StatusCode, e.ErrorCode, e.Message)
	}
	return fmt.Sprintf("internalclient: HTTP %d: %s", e.StatusCode, e.Message)
}

// Client interface untuk komunikasi HTTP internal antar service
type Client interface {
	Get(ctx context.Context, path string, query map[string]string, dest interface{}) error
	Post(ctx context.Context, path string, body interface{}, dest interface{}) error
	PostMultipart(ctx context.Context, path string, fieldName, fileName string, fileReader io.Reader, extraFields map[string]string, dest interface{}) error
	Put(ctx context.Context, path string, body interface{}, dest interface{}) error
	Patch(ctx context.Context, path string, body interface{}, dest interface{}) error
	Delete(ctx context.Context, path string, dest interface{}) error
}

type httpClient struct {
	cfg        Config
	client     *http.Client
	maxRetries int
	retryDelay time.Duration
}

// NewClient membuat instance baru Client inter-service
func NewClient(cfg Config) Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	retries := cfg.MaxRetries
	if retries <= 0 {
		retries = 2
	}
	delay := cfg.RetryDelay
	if delay <= 0 {
		delay = 50 * time.Millisecond
	}

	return &httpClient{
		cfg: cfg,
		client: &http.Client{
			Timeout: timeout,
		},
		maxRetries: retries,
		retryDelay: delay,
	}
}

func (c *httpClient) buildURL(path string, query map[string]string) (string, error) {
	baseURL := strings.TrimRight(c.cfg.BaseURL, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	fullURL := baseURL + path

	if len(query) > 0 {
		u, err := url.Parse(fullURL)
		if err != nil {
			return "", err
		}
		q := u.Query()
		for k, v := range query {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
		return u.String(), nil
	}

	return fullURL, nil
}

func (c *httpClient) doRequest(ctx context.Context, method, path string, query map[string]string, body interface{}, dest interface{}) error {
	reqURL, err := c.buildURL(path, query)
	if err != nil {
		return fmt.Errorf("gagal membangun URL request: %w", err)
	}

	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("gagal marshal request body: %w", err)
		}
		bodyBytes = b
	}

	retries := 0
	if method == http.MethodGet {
		retries = c.maxRetries
	}

	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.retryDelay * time.Duration(attempt)):
			}
		}

		var bodyReader io.Reader
		if len(bodyBytes) > 0 {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
		if err != nil {
			return fmt.Errorf("gagal membuat HTTP request: %w", err)
		}

		// Header standar internal
		req.Header.Set("Accept", "application/json")
		if c.cfg.InternalToken != "" {
			req.Header.Set(HeaderInternalToken, c.cfg.InternalToken)
		}

		// Propagasi Request ID jika ada di context
		reqID := requestid.FromContext(ctx)
		if reqID != "" {
			req.Header.Set(HeaderRequestID, reqID)
		}

		if len(bodyBytes) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		respBody, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("gagal membaca respons body: %w", readErr)
			continue
		}

		// Transient error check for retry (502, 503, 504)
		if method == http.MethodGet && (resp.StatusCode == http.StatusBadGateway || resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusGatewayTimeout) {
			lastErr = fmt.Errorf("server mengembalikan HTTP %d", resp.StatusCode)
			continue
		}

		// Handle error response (status code >= 400)
		if resp.StatusCode >= 400 {
			var errResp response.ErrorResponse
			if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.ErrorCode != "" {
				return &Error{
					StatusCode: resp.StatusCode,
					ErrorCode:  errResp.ErrorCode,
					Message:    errResp.Message,
					Errors:     errResp.Errors,
					RequestID:  errResp.RequestID,
				}
			}
			// Fallback jika bukan format response.ErrorResponse
			return &Error{
				StatusCode: resp.StatusCode,
				Message:    string(respBody),
			}
		}

		// Sukses: Parse ke dest
		if dest != nil && len(respBody) > 0 {
			// Cek apakah dest adalah pointer ke SuccessResponse
			if successResp, ok := dest.(*response.SuccessResponse); ok {
				return json.Unmarshal(respBody, successResp)
			}

			// Coba parse amplop JSON standar {"success":true, "data": ...}
			var env struct {
				Success bool            `json:"success"`
				Message string          `json:"message"`
				Data    json.RawMessage `json:"data"`
			}

			if err := json.Unmarshal(respBody, &env); err == nil && env.Data != nil && len(env.Data) > 0 {
				if err := json.Unmarshal(env.Data, dest); err == nil {
					return nil
				}
			}

			// Fallback unmarshal langsung ke dest
			return json.Unmarshal(respBody, dest)
		}

		return nil
	}

	return fmt.Errorf("request ke %s gagal setelah %d percobaan: %w", reqURL, retries+1, lastErr)
}

func (c *httpClient) Get(ctx context.Context, path string, query map[string]string, dest interface{}) error {
	return c.doRequest(ctx, http.MethodGet, path, query, nil, dest)
}

func (c *httpClient) Post(ctx context.Context, path string, body interface{}, dest interface{}) error {
	return c.doRequest(ctx, http.MethodPost, path, nil, body, dest)
}

func (c *httpClient) PostMultipart(ctx context.Context, path string, fieldName, fileName string, fileReader io.Reader, extraFields map[string]string, dest interface{}) error {
	reqURL, err := c.buildURL(path, nil)
	if err != nil {
		return fmt.Errorf("gagal membangun URL request: %w", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for k, v := range extraFields {
		if err := writer.WriteField(k, v); err != nil {
			return fmt.Errorf("gagal menulis field multipart %s: %w", k, err)
		}
	}

	if fileName == "" {
		fileName = "upload.bin"
	}
	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		return fmt.Errorf("gagal membuat form file %s: %w", fieldName, err)
	}

	if _, err := io.Copy(part, fileReader); err != nil {
		return fmt.Errorf("gagal menyalin isi file ke multipart part: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("gagal menutup multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, body)
	if err != nil {
		return fmt.Errorf("gagal membuat HTTP request multipart: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.cfg.InternalToken != "" {
		req.Header.Set(HeaderInternalToken, c.cfg.InternalToken)
	}
	reqID := requestid.FromContext(ctx)
	if reqID != "" {
		req.Header.Set(HeaderRequestID, reqID)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("request multipart ke %s gagal: %w", reqURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("gagal membaca respons body multipart: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp response.ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.ErrorCode != "" {
			return &Error{
				StatusCode: resp.StatusCode,
				ErrorCode:  errResp.ErrorCode,
				Message:    errResp.Message,
				Errors:     errResp.Errors,
				RequestID:  errResp.RequestID,
			}
		}
		return &Error{
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
		}
	}

	if dest != nil && len(respBody) > 0 {
		if successResp, ok := dest.(*response.SuccessResponse); ok {
			return json.Unmarshal(respBody, successResp)
		}

		var env struct {
			Success bool            `json:"success"`
			Message string          `json:"message"`
			Data    json.RawMessage `json:"data"`
		}

		if err := json.Unmarshal(respBody, &env); err == nil && env.Data != nil && len(env.Data) > 0 {
			if err := json.Unmarshal(env.Data, dest); err == nil {
				return nil
			}
		}

		return json.Unmarshal(respBody, dest)
	}

	return nil
}

func (c *httpClient) Put(ctx context.Context, path string, body interface{}, dest interface{}) error {
	return c.doRequest(ctx, http.MethodPut, path, nil, body, dest)
}

func (c *httpClient) Patch(ctx context.Context, path string, body interface{}, dest interface{}) error {
	return c.doRequest(ctx, http.MethodPatch, path, nil, body, dest)
}

func (c *httpClient) Delete(ctx context.Context, path string, dest interface{}) error {
	return c.doRequest(ctx, http.MethodDelete, path, nil, nil, dest)
}
