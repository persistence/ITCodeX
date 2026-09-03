package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(baseURL string) *Client {
	if baseURL == "" {
		if envURL := os.Getenv("TEST_SERVER_URL"); envURL != "" {
			baseURL = envURL
		} else {
			baseURL = "http://localhost:8000"
		}
	}

	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, params map[string]string) (interface{}, error) {
	reqURL, err := url.Parse(c.BaseURL + path)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if params != nil {
		q := reqURL.Query()
		for k, v := range params {
			q.Add(k, v)
		}
		reqURL.RawQuery = q.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data"`
		Errors  interface{} `json:"errors"`
	}

	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &apiResp); err != nil {
			if resp.StatusCode >= 400 {
				return nil, &APIError{
					Code:    resp.StatusCode,
					Message: string(respBody),
				}
			}
			return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(respBody))
		}
	}

	if resp.StatusCode >= 400 {
		apiErr := &APIError{
			Code:    resp.StatusCode,
			Message: apiResp.Message,
			Data:    apiResp.Errors,
		}
		if apiErr.Message == "" {
			apiErr.Message = string(respBody)
		}
		return nil, apiErr
	}

	if apiResp.Code != 0 && apiResp.Code != 200 && apiResp.Code != 201 {
		return nil, &APIError{
			Code:    apiResp.Code,
			Message: apiResp.Message,
			Data:    apiResp.Errors,
		}
	}

	return apiResp.Data, nil
}

func (c *Client) Get(ctx context.Context, path string, params map[string]string) (interface{}, error) {
	return c.doRequest(ctx, http.MethodGet, path, nil, params)
}

func (c *Client) Post(ctx context.Context, path string, body interface{}) (interface{}, error) {
	return c.doRequest(ctx, http.MethodPost, path, body, nil)
}

func (c *Client) Put(ctx context.Context, path string, body interface{}) (interface{}, error) {
	return c.doRequest(ctx, http.MethodPut, path, body, nil)
}

func (c *Client) Delete(ctx context.Context, path string, params map[string]string) (interface{}, error) {
	return c.doRequest(ctx, http.MethodDelete, path, nil, params)
}
