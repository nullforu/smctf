package vm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrNotFound    = errors.New("vm not found")
	ErrInvalid     = errors.New("vm request invalid")
	ErrUnavailable = errors.New("vm orchestrator unavailable")
	ErrUnexpected  = errors.New("vm orchestrator error")
)

type StatusError struct {
	StatusCode int
	Message    string
}

func (e *StatusError) Error() string {
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}

	return fmt.Sprintf("vm orchestrator returned status %d", e.StatusCode)
}

func (e *StatusError) Unwrap() error {
	switch e.StatusCode {
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusBadRequest:
		return ErrInvalid
	case http.StatusServiceUnavailable, http.StatusBadGateway, http.StatusGatewayTimeout:
		return ErrUnavailable
	default:
		return ErrUnexpected
	}
}

type API interface {
	CreateSandbox(ctx context.Context, id string, specYAML string) (*Sandbox, error)
	GetSandbox(ctx context.Context, id string) (*Sandbox, error)
	DeleteSandbox(ctx context.Context, id string) error
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) CreateSandbox(ctx context.Context, id string, specYAML string) (*Sandbox, error) {
	_, req, err := RenderManifestWithID(specYAML, id)
	if err != nil {
		return nil, err
	}

	var resp sandboxObjectResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/sandboxes", req, &resp); err != nil {
		return nil, err
	}

	if resp.Sandbox == nil {
		return nil, ErrUnexpected
	}

	return resp.Sandbox, nil
}

func (c *Client) GetSandbox(ctx context.Context, id string) (*Sandbox, error) {
	var resp sandboxObjectResponse
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/sandboxes/"+url.PathEscape(id), nil, &resp); err != nil {
		return nil, err
	}

	if resp.Sandbox == nil {
		return nil, ErrUnexpected
	}

	return resp.Sandbox, nil
}

func (c *Client) DeleteSandbox(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/sandboxes/"+url.PathEscape(id), nil, nil)
}

func (c *Client) doJSON(ctx context.Context, method, path string, body any, out any) error {
	reader, err := encodeBody(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("vm client request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	if err := handleStatus(resp); err != nil {
		return err
	}

	if out == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("vm client decode: %w", err)
	}

	return nil
}

func encodeBody(body any) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("vm client marshal: %w", err)
	}

	return bytes.NewReader(payload), nil
}

func handleStatus(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	message := strings.TrimSpace(resp.Status)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err == nil && len(body) > 0 {
		message = parseErrorMessage(body, message)
	}

	return &StatusError{StatusCode: resp.StatusCode, Message: message}
}

func parseErrorMessage(body []byte, fallback string) string {
	var resp ErrorResponse
	if err := json.Unmarshal(body, &resp); err == nil && strings.TrimSpace(resp.Error) != "" {
		return strings.TrimSpace(resp.Error)
	}

	if text := strings.TrimSpace(string(body)); text != "" {
		return text
	}

	return fallback
}
