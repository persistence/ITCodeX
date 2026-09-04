package client

import (
	"context"
	"encoding/json"
	"fmt"
)

func toJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func fromJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// extractData unmarshals the "data" field from a full response map into the given target struct.
func extractData(resp interface{}, target interface{}) error {
	m, ok := resp.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response type: %T", resp)
	}
	data := m["data"]
	// Marshal then unmarshal to convert map -> typed struct
	b, err := json.Marshal(data)
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
	if err := extractData(resp, &collections); err != nil {
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

func (c *Client) DropCollection(ctx context.Context, name string) error {
	_, err := c.del(ctx, fmt.Sprintf("/api/meta/collections/%s", name), nil)
	return err
}

func (c *Client) ListFields(ctx context.Context, collection string) ([]Field, error) {
	resp, err := c.get(ctx, fmt.Sprintf("/api/meta/collections/%s/fields", collection), nil)
	if err != nil {
		return nil, err
	}

	var fields []Field
	if err := extractData(resp, &fields); err != nil {
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
	if err := extractData(resp, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func (c *Client) RemoveField(ctx context.Context, collection, field string) ([]Field, error) {
	resp, err := c.del(ctx, fmt.Sprintf("/api/meta/collections/%s/fields/%s", collection, field), nil)
	if err != nil {
		return nil, err
	}

	var fields []Field
	if err := extractData(resp, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func (c *Client) ListScripts(ctx context.Context) ([]Script, error) {
	resp, err := c.get(ctx, "/api/meta/scripts", nil)
	if err != nil {
		return nil, err
	}

	var scripts []Script
	if err := extractData(resp, &scripts); err != nil {
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
