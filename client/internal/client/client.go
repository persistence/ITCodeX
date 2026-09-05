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
			baseURL = "http://127.0.0.1:8000"
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

	var apiResp map[string]interface{}

	if len(respBody) > 0 {
		dec := json.NewDecoder(bytes.NewReader(respBody))
		dec.UseNumber()
		if err := dec.Decode(&apiResp); err != nil {
			if resp.StatusCode >= 400 {
				return nil, &APIError{
					Code:    resp.StatusCode,
					Message: string(respBody),
				}
			}
			return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(respBody))
		}
	}
	if apiResp == nil {
		apiResp = make(map[string]interface{})
	}

	apiResp = normalizeNumbers(apiResp).(map[string]interface{})

	if resp.StatusCode >= 400 {
		apiErr := &APIError{
			Code:    resp.StatusCode,
			Message: toString(apiResp["message"]),
			Data:    apiResp["errors"],
		}
		if apiErr.Message == "" {
			apiErr.Message = string(respBody)
		}
		return nil, apiErr
	}

	codeVal, _ := apiResp["code"]
	codeNum := asInt64(codeVal)
	if codeNum != 0 && codeNum != 200 && codeNum != 201 {
		return nil, &APIError{
			Code:    int(codeNum),
			Message: toString(apiResp["message"]),
			Data:    apiResp["errors"],
		}
	}

	return apiResp, nil
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func asInt64(v interface{}) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case int32:
		return int64(n)
	case float64:
		return int64(n)
	case float32:
		return int64(n)
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return i
		}
	}
	return 0
}

// normalizeNumbers converts json.Number values to int64 or float64 for ease of use.
func normalizeNumbers(v interface{}) interface{} {
	switch val := v.(type) {
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return i
		}
		if f, err := val.Float64(); err == nil {
			return f
		}
		return val.String()
	case map[string]interface{}:
		for k, vv := range val {
			val[k] = normalizeNumbers(vv)
		}
		return val
	case []interface{}:
		for i, vv := range val {
			val[i] = normalizeNumbers(vv)
		}
		return val
	default:
		return v
	}
}

func (c *Client) get(ctx context.Context, path string, params map[string]string) (interface{}, error) {
	return c.doRequest(ctx, http.MethodGet, path, nil, params)
}

func (c *Client) post(ctx context.Context, path string, body interface{}) (interface{}, error) {
	return c.doRequest(ctx, http.MethodPost, path, body, nil)
}

func (c *Client) put(ctx context.Context, path string, body interface{}) (interface{}, error) {
	return c.doRequest(ctx, http.MethodPut, path, body, nil)
}

func (c *Client) del(ctx context.Context, path string, params map[string]string) (interface{}, error) {
	return c.doRequest(ctx, http.MethodDelete, path, nil, params)
}
