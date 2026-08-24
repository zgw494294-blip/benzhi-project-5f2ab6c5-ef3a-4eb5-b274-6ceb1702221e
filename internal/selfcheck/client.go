package selfcheck

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
}

type responseEnvelope[T any] struct {
	Data  T `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func NewClient(baseURL string) *Client {
	return &Client{baseURL: baseURL, http: &http.Client{Timeout: 5 * time.Second}}
}

func request[T any](ctx context.Context, c *Client, method, path, key string, body any) (T, int, error) {
	var zero T
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return zero, 0, err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return zero, 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return zero, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return zero, resp.StatusCode, err
	}
	var envelope responseEnvelope[T]
	if err = json.Unmarshal(data, &envelope); err != nil {
		return zero, resp.StatusCode, fmt.Errorf("解析响应: %w", err)
	}
	if resp.StatusCode >= 400 {
		message := "HTTP 请求失败"
		if envelope.Error != nil {
			message = envelope.Error.Code + ": " + envelope.Error.Message
		}
		return zero, resp.StatusCode, fmt.Errorf("%s", message)
	}
	return envelope.Data, resp.StatusCode, nil
}

func (c *Client) WaitReady(ctx context.Context) error {
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		_, status, err := request[map[string]string](ctx, c, http.MethodGet, "/healthz", "", nil)
		if err == nil && status == http.StatusOK {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("等待 HTTP 服务: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}
