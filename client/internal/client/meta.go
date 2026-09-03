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

func (c *Client) ListCollections(ctx context.Context) ([]Collection, error) {
	data, err := c.Get(ctx, "/api/meta/collections", nil)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := toJSON(data)
	if err != nil {
		return nil, err
	}

	var collections []Collection
	if err := fromJSON(jsonBytes, &collections); err != nil {
		return nil, err
	}

	return collections, nil
}

func (c *Client) CreateCollection(ctx context.Context, input CreateCollectionInput) (*Collection, error) {
	data, err := c.Post(ctx, "/api/meta/collections", input)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := toJSON(data)
	if err != nil {
		return nil, err
	}

	var collection Collection
	if err := fromJSON(jsonBytes, &collection); err != nil {
		return nil, err
	}

	return &collection, nil
}

func (c *Client) GetCollection(ctx context.Context, name string) (*Collection, error) {
	data, err := c.Get(ctx, fmt.Sprintf("/api/meta/collections/%s", name), nil)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := toJSON(data)
	if err != nil {
		return nil, err
	}

	var collection Collection
	if err := fromJSON(jsonBytes, &collection); err != nil {
		return nil, err
	}

	return &collection, nil
}

func (c *Client) DropCollection(ctx context.Context, name string) error {
	_, err := c.Delete(ctx, fmt.Sprintf("/api/meta/collections/%s", name), nil)
	return err
}

func (c *Client) ListFields(ctx context.Context, collection string) ([]Field, error) {
	data, err := c.Get(ctx, fmt.Sprintf("/api/meta/collections/%s/fields", collection), nil)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := toJSON(data)
	if err != nil {
		return nil, err
	}

	var fields []Field
	if err := fromJSON(jsonBytes, &fields); err != nil {
		return nil, err
	}

	return fields, nil
}

func (c *Client) AddField(ctx context.Context, collection string, input CreateFieldInput) ([]Field, error) {
	data, err := c.Post(ctx, fmt.Sprintf("/api/meta/collections/%s/fields", collection), input)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := toJSON(data)
	if err != nil {
		return nil, err
	}

	var fields []Field
	if err := fromJSON(jsonBytes, &fields); err != nil {
		return nil, err
	}

	return fields, nil
}

func (c *Client) RemoveField(ctx context.Context, collection, field string) ([]Field, error) {
	data, err := c.Delete(ctx, fmt.Sprintf("/api/meta/collections/%s/fields/%s", collection, field), nil)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := toJSON(data)
	if err != nil {
		return nil, err
	}

	var fields []Field
	if err := fromJSON(jsonBytes, &fields); err != nil {
		return nil, err
	}

	return fields, nil
}

func (c *Client) ListScripts(ctx context.Context) ([]Script, error) {
	data, err := c.Get(ctx, "/api/meta/scripts", nil)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := toJSON(data)
	if err != nil {
		return nil, err
	}

	var scripts []Script
	if err := fromJSON(jsonBytes, &scripts); err != nil {
		return nil, err
	}

	return scripts, nil
}

func (c *Client) CreateScript(ctx context.Context, input CreateScriptInput) (*Script, error) {
	data, err := c.Post(ctx, "/api/meta/scripts", input)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := toJSON(data)
	if err != nil {
		return nil, err
	}

	var script Script
	if err := fromJSON(jsonBytes, &script); err != nil {
		return nil, err
	}

	return &script, nil
}

func (c *Client) DisableScript(ctx context.Context, id int64) error {
	_, err := c.Post(ctx, fmt.Sprintf("/api/meta/scripts/%d/disable", id), nil)
	return err
}
