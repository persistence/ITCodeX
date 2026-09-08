package metadata

import (
	"context"
	"time"

	"github.com/gogf/gf/v2/os/gtime"
)

type IDField struct {
	BaseField
}

func (f *IDField) ValidateValue(ctx context.Context, value any) error {
	return nil
}

func (f *IDField) DDLColumn() string {
	return quoteIdent(f.name) + " BIGINT NOT NULL PRIMARY KEY"
}

func NewIDField(coll *Collection, opts map[string]any) Field {
	if opts == nil {
		opts = make(map[string]any)
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

func (f *CreatedAtField) ToStoreValue(value any) (any, error) {
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

func (f *CreatedAtField) FromStoreValue(value any) (any, error) {
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

func (f *CreatedAtField) ValidateValue(ctx context.Context, value any) error {
	return nil
}

func NewCreatedAtField(coll *Collection, opts map[string]any) Field {
	if opts == nil {
		opts = make(map[string]any)
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

func (f *UpdatedAtField) ToStoreValue(value any) (any, error) {
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

func (f *UpdatedAtField) FromStoreValue(value any) (any, error) {
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

func (f *UpdatedAtField) ValidateValue(ctx context.Context, value any) error {
	return nil
}

func NewUpdatedAtField(coll *Collection, opts map[string]any) Field {
	if opts == nil {
		opts = make(map[string]any)
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

type ActorField struct {
	BaseField
}

func (f *ActorField) ValidateValue(ctx context.Context, value any) error {
	return nil
}

func newActorField(fieldType FieldType, name, displayName string, opts map[string]any) Field {
	if opts == nil {
		opts = make(map[string]any)
	}
	if _, ok := opts["name"]; !ok {
		opts["name"] = name
	}
	if _, ok := opts["displayName"]; !ok {
		opts["displayName"] = displayName
	}
	opts["isSystem"] = true
	opts["required"] = false
	return &ActorField{
		BaseField: newBaseField(string(fieldType), DataTypeBigInt, opts),
	}
}

func NewCreatedByField(coll *Collection, opts map[string]any) Field {
	return newActorField(FieldTypeCreatedBy, "created_by", "创建人", opts)
}

func NewUpdatedByField(coll *Collection, opts map[string]any) Field {
	return newActorField(FieldTypeUpdatedBy, "updated_by", "更新人", opts)
}

type PresetFieldFactory func(coll *Collection, opts map[string]any) Field

var PresetFieldsMap = map[string]PresetFieldFactory{
	"id":        NewIDField,
	"createdAt": NewCreatedAtField,
	"updatedAt": NewUpdatedAtField,
	"createdBy": NewCreatedByField,
	"updatedBy": NewUpdatedByField,
}
