package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

// UploadFile POST /api/c/:collection/upload (multipart field "file")
func (c *Client) UploadFile(ctx context.Context, collection, filename, contentType string, content []byte) (map[string]any, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(content); err != nil {
		return nil, err
	}
	_ = w.Close()

	reqURL := c.BaseURL + fmt.Sprintf("/api/c/%s/upload", url.PathEscape(collection))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var apiResp map[string]any
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("upload parse: %w body=%s", err, string(body))
	}
	apiResp = normalizeNumbers(apiResp).(map[string]any)
	if resp.StatusCode >= 400 {
		return nil, &APIError{Code: resp.StatusCode, Message: toString(apiResp["message"]), Data: apiResp["data"]}
	}
	if code := asInt64(apiResp["code"]); code != 0 && code != 200 && code != 201 {
		return nil, &APIError{Code: int(code), Message: toString(apiResp["message"]), Data: apiResp["data"]}
	}
	rec, ok := apiResp["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected upload data: %T", apiResp["data"])
	}
	_ = contentType
	return rec, nil
}

// DownloadFile GET /api/c/:collection/files/:id/content
func (c *Client) DownloadFile(ctx context.Context, collection, id string) ([]byte, string, error) {
	reqURL := c.BaseURL + fmt.Sprintf("/api/c/%s/files/%s/content", url.PathEscape(collection), url.PathEscape(id))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode >= 400 {
		return nil, "", &APIError{Code: resp.StatusCode, Message: string(body)}
	}
	return body, resp.Header.Get("Content-Type"), nil
}
