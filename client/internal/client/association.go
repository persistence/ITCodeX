package client

import (
	"context"
	"fmt"
	"net/url"
)

// ListAssociation GET /api/c/:collection/:id/:association
func (c *Client) ListAssociation(ctx context.Context, collection, id, association string) ([]map[string]any, error) {
	resp, err := c.get(ctx, fmt.Sprintf("/api/c/%s/%s/%s",
		url.PathEscape(collection), url.PathEscape(id), url.PathEscape(association)), nil)
	if err != nil {
		return nil, err
	}
	m, _ := resp.(map[string]any)
	data, _ := m["data"].(map[string]any)
	if data == nil {
		return nil, fmt.Errorf("unexpected association list response")
	}
	raw, _ := data["list"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if rec, ok := item.(map[string]any); ok {
			out = append(out, rec)
		}
	}
	return out, nil
}

// AddAssociation POST
func (c *Client) AddAssociation(ctx context.Context, collection, id, association string, body any) error {
	_, err := c.post(ctx, fmt.Sprintf("/api/c/%s/%s/%s",
		url.PathEscape(collection), url.PathEscape(id), url.PathEscape(association)), body)
	return err
}

// SetAssociation PUT
func (c *Client) SetAssociation(ctx context.Context, collection, id, association string, body any) error {
	_, err := c.put(ctx, fmt.Sprintf("/api/c/%s/%s/%s",
		url.PathEscape(collection), url.PathEscape(id), url.PathEscape(association)), body)
	return err
}

// RemoveAssociation DELETE — also sends targetId as query for clients/proxies that drop DELETE bodies.
// Uses targetId (not id) so the query does not shadow the path source :id.
func (c *Client) RemoveAssociation(ctx context.Context, collection, id, association string, body any) error {
	params := map[string]string{}
	if m, ok := body.(map[string]any); ok {
		if v, ok := m["id"]; ok {
			params["targetId"] = fmt.Sprintf("%v", v)
		}
	}
	_, err := c.doRequest(ctx, "DELETE", fmt.Sprintf("/api/c/%s/%s/%s",
		url.PathEscape(collection), url.PathEscape(id), url.PathEscape(association)), body, params)
	return err
}
