package metadata

import (
	v1 "itcodex/server/api/metadata/v1"
	md "itcodex/server/internal/service/metadata"
)

func collToItem(c *md.Collection, withFields bool) v1.CollectionItem {
	item := v1.CollectionItem{
		Name:        c.Name(),
		DisplayName: c.DisplayName(),
		TableName:   c.TableName(),
		Type:        string(c.Type()),
		Description: collectionDescription(c),
		Categories:  collectionCategories(c),
		Options:     collectionPublicOptions(c),
		FieldCount:  len(c.Fields()),
	}
	if withFields {
		item.Fields = fieldsToItems(c)
	}
	return item
}

func fieldsToItems(c *md.Collection) []v1.FieldItem {
	fields := c.Fields()
	out := make([]v1.FieldItem, 0, len(fields))
	for _, f := range fields {
		out = append(out, fieldToItem(f))
	}
	return out
}

func fieldToItem(f md.Field) v1.FieldItem {
	opts := f.Options()
	item := v1.FieldItem{
		Name:        f.Name(),
		DisplayName: f.DisplayName(),
		Type:        string(f.Type()),
		Required:    f.IsRequired(),
		Unique:      f.IsUnique(),
		Indexed:     f.IsIndexed(),
		IsSystem:    f.IsSystem(),
		Options:     opts,
	}
	if opts != nil {
		item.Validation = opts["validation"]
	}
	return item
}

func collectionDescription(c *md.Collection) string {
	if s, ok := c.Options()["description"].(string); ok {
		return s
	}
	return ""
}

func collectionCategories(c *md.Collection) []string {
	raw, ok := c.Options()["categories"]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func collectionPublicOptions(c *md.Collection) map[string]any {
	opts := c.Options()
	if opts == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(opts))
	for k, v := range opts {
		if k == "description" || k == "categories" {
			continue
		}
		out[k] = v
	}
	return out
}

func collectionHasCategory(c *md.Collection, category string) bool {
	for _, item := range collectionCategories(c) {
		if item == category {
			return true
		}
	}
	return false
}
