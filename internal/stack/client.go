package stack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var (
	ErrNotFound    = errors.New("stack not found")
	ErrInvalid     = errors.New("stack request invalid")
	ErrUnavailable = errors.New("stack provisioner unavailable")
	ErrUnexpected  = errors.New("stack provisioner error")
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

type API interface {
	CreateStack(ctx context.Context, targetPort int, podSpec string) (*StackInfo, error)
	GetStackStatus(ctx context.Context, stackID string) (*StackStatus, error)
	DeleteStack(ctx context.Context, stackID string) error
}

type CreateRequest struct {
	TargetPort int    `json:"target_port"`
	PodSpec    string `json:"pod_spec"`
}

type StackInfo struct {
	StackID              string    `json:"stack_id"`
	PodID                string    `json:"pod_id"`
	Namespace            string    `json:"namespace"`
	NodeID               string    `json:"node_id"`
	NodePublicIP         string    `json:"node_public_ip"`
	PodSpec              string    `json:"pod_spec"`
	TargetPort           int       `json:"target_port"`
	NodePort             int       `json:"node_port"`
	ServiceName          string    `json:"service_name"`
	Status               string    `json:"status"`
	TTLExpiresAt         time.Time `json:"ttl_expires_at"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	RequestedCPUMilli    int       `json:"requested_cpu_milli"`
	RequestedMemoryBytes int       `json:"requested_memory_bytes"`
}

type StackStatus struct {
	StackID      string    `json:"stack_id"`
	Status       string    `json:"status"`
	TTL          time.Time `json:"ttl"`
	NodePort     int       `json:"node_port"`
	TargetPort   int       `json:"target_port"`
	NodePublicIP string    `json:"node_public_ip"`
}

func NewClient(baseURL, apiKey string, timeout time.Duration) *Client {
	baseURL = strings.TrimRight(baseURL, "/")

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) CreateStack(ctx context.Context, targetPort int, podSpec string) (*StackInfo, error) {
	reqBody := CreateRequest{TargetPort: targetPort, PodSpec: podSpec}
	var resp StackInfo
	if err := c.doJSON(ctx, http.MethodPost, "/stacks", reqBody, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *Client) GetStack(ctx context.Context, stackID string) (*StackInfo, error) {
	var resp StackInfo
	if err := c.doJSON(ctx, http.MethodGet, stackPath(stackID), nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *Client) GetStackStatus(ctx context.Context, stackID string) (*StackStatus, error) {
	var resp StackStatus
	if err := c.doJSON(ctx, http.MethodGet, stackStatusPath(stackID), nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *Client) DeleteStack(ctx context.Context, stackID string) error {
	return c.doJSON(ctx, http.MethodDelete, stackPath(stackID), nil, nil)
}

func (c *Client) doJSON(ctx context.Context, method, path string, body any, out any) error {
	reader, err := encodeBody(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("stack client request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.apiKey != "" {
		req.Header.Set("X-API-KEY", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("stack client request: %w", err)
	}
	defer resp.Body.Close()

	if err := handleStatus(resp.StatusCode); err != nil {
		return err
	}

	if out == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("stack client decode: %w", err)
	}

	return nil
}

func encodeBody(body any) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("stack client marshal: %w", err)
	}

	return bytes.NewReader(payload), nil
}

func handleStatus(status int) error {
	if status >= 200 && status < 300 {
		return nil
	}

	return mapStatus(status)
}

func mapStatus(status int) error {
	switch status {
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusBadRequest:
		return ErrInvalid
	case http.StatusServiceUnavailable:
		return ErrUnavailable
	default:
		return ErrUnexpected
	}
}

func stackPath(stackID string) string {
	return fmt.Sprintf("/stacks/%s", stackID)
}

func stackStatusPath(stackID string) string {
	return fmt.Sprintf("/stacks/%s/status", stackID)
}
