package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

func (c *Client) List(ctx context.Context, collection string, opts *FindOptions) (*ListResult, error) {
	params := make(map[string]string)

	if opts != nil {
		if opts.Filter != nil {
			filterBytes, err := json.Marshal(opts.Filter)
			if err != nil {
				return nil, err
			}
			params["filter"] = string(filterBytes)
		}

		if len(opts.Sort) > 0 {
			params["sort"] = joinStrings(opts.Sort, ",")
		}

		if len(opts.Fields) > 0 {
			params["fields"] = joinStrings(opts.Fields, ",")
		}

		if len(opts.Except) > 0 {
			params["except"] = joinStrings(opts.Except, ",")
		}

		if opts.Page > 0 {
			params["page"] = strconv.Itoa(opts.Page)
		}

		if opts.PageSize > 0 {
			params["pageSize"] = strconv.Itoa(opts.PageSize)
		}
	}

	data, err := c.Get(ctx, fmt.Sprintf("/api/c/%s", url.PathEscape(collection)), params)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := toJSON(data)
	if err != nil {
		return nil, err
	}

	var result ListResult
	if err := fromJSON(jsonBytes, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) Get(ctx context.Context, collection, id string, opts *FindOneOptions) (map[string]interface{}, error) {
	params := make(map[string]string)

	if opts != nil {
		if len(opts.Fields) > 0 {
			params["fields"] = joinStrings(opts.Fields, ",")
		}

		if len(opts.Except) > 0 {
			params["except"] = joinStrings(opts.Except, ",")
		}
	}

	data, err := c.Get(ctx, fmt.Sprintf("/api/c/%s/%s", url.PathEscape(collection), url.PathEscape(id)), params)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := toJSON(data)
	if err != nil {
		return nil, err
	}

	var record map[string]interface{}
	if err := fromJSON(jsonBytes, &record); err != nil {
		return nil, err
	}

	return record, nil
}

func (c *Client) Create(ctx context.Context, collection string, data map[string]interface{}) (map[string]interface{}, error) {
	respData, err := c.Post(ctx, fmt.Sprintf("/api/c/%s", url.PathEscape(collection)), data)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := toJSON(respData)
	if err != nil {
		return nil, err
	}

	var record map[string]interface{}
	if err := fromJSON(jsonBytes, &record); err != nil {
		return nil, err
	}

	return record, nil
}

func (c *Client) Update(ctx context.Context, collection, id string, data map[string]interface{}) (map[string]interface{}, error) {
	respData, err := c.Put(ctx, fmt.Sprintf("/api/c/%s/%s", url.PathEscape(collection), url.PathEscape(id)), data)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := toJSON(respData)
	if err != nil {
		return nil, err
	}

	var record map[string]interface{}
	if err := fromJSON(jsonBytes, &record); err != nil {
		return nil, err
	}

	return record, nil
}

func (c *Client) DeleteOne(ctx context.Context, collection, id string) (int64, error) {
	respData, err := c.doRequest(ctx, "DELETE", fmt.Sprintf("/api/c/%s/%s", url.PathEscape(collection), url.PathEscape(id)), nil, nil)
	if err != nil {
		return 0, err
	}

	return extractAffected(respData), nil
}

func (c *Client) BulkDelete(ctx context.Context, collection string, filter Filter) (int64, error) {
	params := make(map[string]string)

	if filter != nil {
		filterBytes, err := json.Marshal(filter)
		if err != nil {
			return 0, err
		}
		params["filter"] = string(filterBytes)
	}

	respData, err := c.doRequest(ctx, "DELETE", fmt.Sprintf("/api/c/%s", url.PathEscape(collection)), nil, params)
	if err != nil {
		return 0, err
	}

	return extractAffected(respData), nil
}

func (c *Client) Count(ctx context.Context, collection string, filter Filter) (int64, error) {
	result, err := c.List(ctx, collection, &FindOptions{
		Filter:   filter,
		Page:     1,
		PageSize: 1,
	})
	if err != nil {
		return 0, err
	}

	return result.Total, nil
}

func joinStrings(items []string, sep string) string {
	if len(items) == 0 {
		return ""
	}
	result := items[0]
	for i := 1; i < len(items); i++ {
		result += sep + items[i]
	}
	return result
}

func extractAffected(data interface{}) int64 {
	if m, ok := data.(map[string]interface{}); ok {
		if affected, ok := m["affected"].(float64); ok {
			return int64(affected)
		}
	}
	return 0
}
