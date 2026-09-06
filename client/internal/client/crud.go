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
		if len(opts.Appends) > 0 {
			params["appends"] = joinStrings(opts.Appends, ",")
		}
		if opts.Page > 0 {
			params["page"] = strconv.Itoa(opts.Page)
		}
		if opts.PageSize > 0 {
			params["pageSize"] = strconv.Itoa(opts.PageSize)
		}
	}

	resp, err := c.get(ctx, fmt.Sprintf("/api/c/%s", url.PathEscape(collection)), params)
	if err != nil {
		return nil, err
	}
	m, _ := resp.(map[string]any)
	return parseListResult(m["data"])
}

func (c *Client) FindOne(ctx context.Context, collection, id string, opts *FindOneOptions) (map[string]any, error) {
	params := make(map[string]string)
	if opts != nil {
		if len(opts.Fields) > 0 {
			params["fields"] = joinStrings(opts.Fields, ",")
		}
		if len(opts.Except) > 0 {
			params["except"] = joinStrings(opts.Except, ",")
		}
		if len(opts.Appends) > 0 {
			params["appends"] = joinStrings(opts.Appends, ",")
		}
	}

	resp, err := c.get(ctx, fmt.Sprintf("/api/c/%s/%s", url.PathEscape(collection), url.PathEscape(id)), params)
	if err != nil {
		return nil, err
	}
	m, _ := resp.(map[string]any)
	record, ok := m["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response data type: %T", m["data"])
	}
	return record, nil
}

func (c *Client) Create(ctx context.Context, collection string, data map[string]any) (map[string]any, error) {
	resp, err := c.post(ctx, fmt.Sprintf("/api/c/%s", url.PathEscape(collection)), data)
	if err != nil {
		return nil, err
	}
	m, _ := resp.(map[string]any)
	record, ok := m["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response data type: %T", m["data"])
	}
	return record, nil
}

func (c *Client) CreateMany(ctx context.Context, collection string, records []map[string]any) ([]map[string]any, error) {
	resp, err := c.post(ctx, fmt.Sprintf("/api/c/%s/batch", url.PathEscape(collection)), records)
	if err != nil {
		return nil, err
	}
	m, _ := resp.(map[string]any)
	data := m["data"]
	switch v := data.(type) {
	case []any:
		out := make([]map[string]any, 0, len(v))
		for _, item := range v {
			if rec, ok := item.(map[string]any); ok {
				out = append(out, rec)
			}
		}
		return out, nil
	case map[string]any:
		if list, ok := v["list"].([]any); ok {
			out := make([]map[string]any, 0, len(list))
			for _, item := range list {
				if rec, ok := item.(map[string]any); ok {
					out = append(out, rec)
				}
			}
			return out, nil
		}
	}
	return nil, fmt.Errorf("unexpected batch response: %T", data)
}

func (c *Client) Update(ctx context.Context, collection, id string, data map[string]any) (map[string]any, error) {
	resp, err := c.put(ctx, fmt.Sprintf("/api/c/%s/%s", url.PathEscape(collection), url.PathEscape(id)), data)
	if err != nil {
		return nil, err
	}
	m, _ := resp.(map[string]any)
	switch v := m["data"].(type) {
	case map[string]any:
		return v, nil
	default:
		// some update handlers return {affected:n}
		return map[string]any{"affected": asInt64(v)}, nil
	}
}

func (c *Client) DeleteOne(ctx context.Context, collection, id string) (int64, error) {
	resp, err := c.del(ctx, fmt.Sprintf("/api/c/%s/%s", url.PathEscape(collection), url.PathEscape(id)), nil)
	if err != nil {
		return 0, err
	}
	return extractAffected(resp), nil
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
	resp, err := c.del(ctx, fmt.Sprintf("/api/c/%s", url.PathEscape(collection)), params)
	if err != nil {
		return 0, err
	}
	return extractAffected(resp), nil
}

func (c *Client) Count(ctx context.Context, collection string, filter Filter) (int64, error) {
	params := make(map[string]string)
	if filter != nil {
		b, err := json.Marshal(filter)
		if err != nil {
			return 0, err
		}
		params["filter"] = string(b)
	}
	resp, err := c.get(ctx, fmt.Sprintf("/api/c/%s/count", url.PathEscape(collection)), params)
	if err != nil {
		return 0, err
	}
	m, _ := resp.(map[string]any)
	data, _ := m["data"].(map[string]any)
	if data == nil {
		return 0, fmt.Errorf("unexpected count response")
	}
	return asInt64(data["count"]), nil
}

func extractAffected(resp any) int64 {
	m, _ := resp.(map[string]any)
	if m == nil {
		return 0
	}
	if data, ok := m["data"].(map[string]any); ok {
		return asInt64(data["affected"])
	}
	return asInt64(m["affected"])
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

func parseListResult(data any) (*ListResult, error) {
	m, ok := data.(map[string]any)
	if !ok {
		if data == nil {
			return &ListResult{}, nil
		}
		return nil, fmt.Errorf("unexpected list response type: %T", data)
	}
	result := &ListResult{}
	if v, ok := m["total"]; ok {
		result.Total = asInt64(v)
	}
	if v, ok := m["page"]; ok {
		result.Page = int(asInt64(v))
	}
	if v, ok := m["pageSize"]; ok {
		result.PageSize = int(asInt64(v))
	}
	if v, ok := m["totalPages"]; ok {
		result.TotalPages = int(asInt64(v))
	}
	if v, ok := m["list"]; ok {
		if list, ok := v.([]any); ok {
			result.List = make([]map[string]any, 0, len(list))
			for _, item := range list {
				if rec, ok := item.(map[string]any); ok {
					result.List = append(result.List, rec)
				}
			}
		}
	}
	return result, nil
}
