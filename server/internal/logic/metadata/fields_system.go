package metadata

import (
	"context"
	"time"

	"github.com/gogf/gf/v2/os/gtime"
)

type IDField struct {
	BaseField
}

func (f *IDField) ValidateValue(ctx context.Context, value interface{}) error {
	return nil
}

func (f *IDField) DDLColumn() string {
	return f.name + " INTEGER PRIMARY KEY AUTOINCREMENT"
}

func NewIDField(coll *Collection, opts map[string]interface{}) Field {
	if opts == nil {
		opts = make(map[string]interface{})
	}
	if _, ok := opts["name"]; !ok {
		opts["name"] = DefaultPrimaryKey
	}
	if _, ok := opts["displayName"]; !ok {
		opts["displayName"] = "ID"
	}
	opts["isSystem"] = true
	opts["required"] = false
	return &IDField{
		BaseField: newBaseField("id", DataTypeBigInt, opts),
	}
}

type CreatedAtField struct {
	BaseField
}

func (f *CreatedAtField) ToStoreValue(value interface{}) (interface{}, error) {
	if value == nil {
		return time.Now(), nil
	}
	switch v := value.(type) {
	case *gtime.Time:
		return v.Time, nil
	case time.Time:
		return v, nil
	}
	return value, nil
}

func (f *CreatedAtField) FromStoreValue(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	switch v := value.(type) {
	case time.Time:
		return gtime.New(v), nil
	case *gtime.Time:
		return v, nil
	}
	return value, nil
}

func (f *CreatedAtField) ValidateValue(ctx context.Context, value interface{}) error {
	return nil
}

func NewCreatedAtField(coll *Collection, opts map[string]interface{}) Field {
	if opts == nil {
		opts = make(map[string]interface{})
	}
	if _, ok := opts["name"]; !ok {
		opts["name"] = "created_at"
	}
	if _, ok := opts["displayName"]; !ok {
		opts["displayName"] = "创建时间"
	}
	opts["isSystem"] = true
	opts["required"] = false
	return &CreatedAtField{
		BaseField: newBaseField(string(FieldTypeCreatedAt), DataTypeDateTime, opts),
	}
}

type UpdatedAtField struct {
	BaseField
}

func (f *UpdatedAtField) ToStoreValue(value interface{}) (interface{}, error) {
	if value == nil {
		return time.Now(), nil
	}
	switch v := value.(type) {
	case *gtime.Time:
		return v.Time, nil
	case time.Time:
		return v, nil
	}
	return value, nil
}

func (f *UpdatedAtField) FromStoreValue(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	switch v := value.(type) {
	case time.Time:
		return gtime.New(v), nil
	case *gtime.Time:
		return v, nil
	}
	return value, nil
}

func (f *UpdatedAtField) ValidateValue(ctx context.Context, value interface{}) error {
	return nil
}

func NewUpdatedAtField(coll *Collection, opts map[string]interface{}) Field {
	if opts == nil {
		opts = make(map[string]interface{})
	}
	if _, ok := opts["name"]; !ok {
		opts["name"] = "updated_at"
	}
	if _, ok := opts["displayName"]; !ok {
		opts["displayName"] = "更新时间"
	}
	opts["isSystem"] = true
	opts["required"] = false
	return &UpdatedAtField{
		BaseField: newBaseField(string(FieldTypeUpdatedAt), DataTypeDateTime, opts),
	}
}

type PresetFieldFactory func(coll *Collection, opts map[string]interface{}) Field

var PresetFieldsMap = map[string]PresetFieldFactory{
	"id":        NewIDField,
	"createdAt": NewCreatedAtField,
	"updatedAt": NewUpdatedAtField,
}
