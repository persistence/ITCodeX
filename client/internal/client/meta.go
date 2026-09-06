package client

import (
	"context"
	"encoding/json"
	"fmt"
)

func toJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}

func fromJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// extractData unmarshals the "data" field from a full response map into the given target.
func extractData(resp any, target any) error {
	m, ok := resp.(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected response type: %T", resp)
	}
	data := m["data"]
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}

func extractList(resp any, target any) error {
	m, ok := resp.(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected response type: %T", resp)
	}
	data, ok := m["data"].(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected data type: %T", m["data"])
	}
	b, err := json.Marshal(data["list"])
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}

func (c *Client) ListCollections(ctx context.Context) ([]Collection, error) {
	resp, err := c.get(ctx, "/api/meta/collections", nil)
	if err != nil {
		return nil, err
	}
	var collections []Collection
	if err := extractList(resp, &collections); err != nil {
		return nil, err
	}
	return collections, nil
}

func (c *Client) CreateCollection(ctx context.Context, input CreateCollectionInput) (*Collection, error) {
	resp, err := c.post(ctx, "/api/meta/collections", input)
	if err != nil {
		return nil, err
	}
	var collection Collection
	if err := extractData(resp, &collection); err != nil {
		return nil, err
	}
	return &collection, nil
}

func (c *Client) GetCollection(ctx context.Context, name string) (*Collection, error) {
	resp, err := c.get(ctx, fmt.Sprintf("/api/meta/collections/%s", name), nil)
	if err != nil {
		return nil, err
	}
	var collection Collection
	if err := extractData(resp, &collection); err != nil {
		return nil, err
	}
	return &collection, nil
}

func (c *Client) UpdateCollection(ctx context.Context, name string, input UpdateCollectionInput) (*Collection, error) {
	resp, err := c.put(ctx, fmt.Sprintf("/api/meta/collections/%s", name), input)
	if err != nil {
		return nil, err
	}
	var collection Collection
	if err := extractData(resp, &collection); err != nil {
		return nil, err
	}
	return &collection, nil
}

func (c *Client) DropCollection(ctx context.Context, name string) error {
	_, err := c.del(ctx, fmt.Sprintf("/api/meta/collections/%s", name), nil)
	return err
}

func (c *Client) SyncCollection(ctx context.Context, name string) (*Collection, error) {
	resp, err := c.post(ctx, fmt.Sprintf("/api/meta/collections/%s/sync", name), nil)
	if err != nil {
		return nil, err
	}
	var collection Collection
	if err := extractData(resp, &collection); err != nil {
		return nil, err
	}
	return &collection, nil
}

func (c *Client) ListFields(ctx context.Context, collection string) ([]Field, error) {
	resp, err := c.get(ctx, fmt.Sprintf("/api/meta/collections/%s/fields", collection), nil)
	if err != nil {
		return nil, err
	}
	var fields []Field
	if err := extractList(resp, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func (c *Client) AddField(ctx context.Context, collection string, input CreateFieldInput) ([]Field, error) {
	resp, err := c.post(ctx, fmt.Sprintf("/api/meta/collections/%s/fields", collection), input)
	if err != nil {
		return nil, err
	}
	var fields []Field
	if err := extractList(resp, &fields); err != nil {
		// some handlers may return single field or wrapped list
		if err2 := extractData(resp, &fields); err2 == nil {
			return fields, nil
		}
		return nil, err
	}
	return fields, nil
}

func (c *Client) UpdateField(ctx context.Context, collection, field string, input UpdateFieldInput) ([]Field, error) {
	resp, err := c.put(ctx, fmt.Sprintf("/api/meta/collections/%s/fields/%s", collection, field), input)
	if err != nil {
		return nil, err
	}
	var fields []Field
	if err := extractList(resp, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func (c *Client) RemoveField(ctx context.Context, collection, field string) ([]Field, error) {
	_, err := c.del(ctx, fmt.Sprintf("/api/meta/collections/%s/fields/%s", collection, field), nil)
	if err != nil {
		return nil, err
	}
	return c.ListFields(ctx, collection)
}

func (c *Client) ListIndexes(ctx context.Context, collection string) ([]Index, error) {
	resp, err := c.get(ctx, fmt.Sprintf("/api/meta/collections/%s/indexes", collection), nil)
	if err != nil {
		return nil, err
	}
	var indexes []Index
	if err := extractList(resp, &indexes); err != nil {
		return nil, err
	}
	return indexes, nil
}

func (c *Client) CreateIndex(ctx context.Context, collection string, index Index) (*Index, error) {
	resp, err := c.post(ctx, fmt.Sprintf("/api/meta/collections/%s/indexes", collection), index)
	if err != nil {
		return nil, err
	}
	var out struct {
		Index *Index `json:"index"`
	}
	if err := extractData(resp, &out); err != nil {
		var idx Index
		if err2 := extractData(resp, &idx); err2 == nil {
			return &idx, nil
		}
		return nil, err
	}
	if out.Index != nil {
		return out.Index, nil
	}
	return &index, nil
}

func (c *Client) DeleteIndex(ctx context.Context, collection string, fields []string) error {
	_, err := c.doRequest(ctx, "DELETE", fmt.Sprintf("/api/meta/collections/%s/indexes", collection), map[string]any{
		"fields": fields,
	}, nil)
	return err
}

func (c *Client) ListScripts(ctx context.Context) ([]Script, error) {
	resp, err := c.get(ctx, "/api/meta/scripts", nil)
	if err != nil {
		return nil, err
	}
	var scripts []Script
	if err := extractList(resp, &scripts); err != nil {
		return nil, err
	}
	return scripts, nil
}

func (c *Client) CreateScript(ctx context.Context, input CreateScriptInput) (*Script, error) {
	resp, err := c.post(ctx, "/api/meta/scripts", input)
	if err != nil {
		return nil, err
	}
	var script Script
	if err := extractData(resp, &script); err != nil {
		return nil, err
	}
	return &script, nil
}

func (c *Client) DisableScript(ctx context.Context, id int64) error {
	_, err := c.post(ctx, fmt.Sprintf("/api/meta/scripts/%d/disable", id), nil)
	return err
}

func (c *Client) DeleteScript(ctx context.Context, id int64) error {
	_, err := c.del(ctx, fmt.Sprintf("/api/meta/scripts/%d", id), nil)
	return err
}

func (c *Client) ValidateScript(ctx context.Context, content string) (*ValidateScriptResult, error) {
	resp, err := c.post(ctx, "/api/meta/scripts/validate", map[string]string{"content": content})
	if err != nil {
		return nil, err
	}
	var result ValidateScriptResult
	if err := extractData(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
