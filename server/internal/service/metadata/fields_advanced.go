package metadata

import (
	"context"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/os/gtime"
)

type DatetimeField struct {
	BaseField
}

func (f *DatetimeField) ToStoreValue(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	switch v := value.(type) {
	case *gtime.Time:
		return v.Time, nil
	case time.Time:
		return v, nil
	}
	return value, nil
}

func (f *DatetimeField) FromStoreValue(value interface{}) (interface{}, error) {
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

func NewDatetimeField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &DatetimeField{
		BaseField: newBaseField(string(FieldTypeDateTime), DataTypeDateTime, opts),
	}, nil
}

type DateField struct {
	BaseField
}

func (f *DateField) ToStoreValue(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	switch v := value.(type) {
	case *gtime.Time:
		return v.Time, nil
	case time.Time:
		return v, nil
	}
	return value, nil
}

func (f *DateField) FromStoreValue(value interface{}) (interface{}, error) {
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

func NewDateField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &DateField{
		BaseField: newBaseField(string(FieldTypeDate), DataTypeDate, opts),
	}, nil
}

type SelectField struct {
	BaseField
}

func (f *SelectField) ValidateValue(ctx context.Context, value interface{}) error {
	if err := f.BaseField.ValidateValue(ctx, value); err != nil {
		return err
	}
	if value == nil {
		return nil
	}
	opts, ok := f.options["options"].([]interface{})
	if !ok || len(opts) == 0 {
		return nil
	}
	strVal := fmt.Sprintf("%v", value)
	for _, opt := range opts {
		if fmt.Sprintf("%v", opt) == strVal {
			return nil
		}
		if m, ok := opt.(map[string]interface{}); ok {
			if v, ok := m["value"]; ok && fmt.Sprintf("%v", v) == strVal {
				return nil
			}
		}
	}
	return fmt.Errorf("字段 %s 的值 %v 不在可选范围内", f.name, value)
}

func NewSelectField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &SelectField{
		BaseField: newBaseField(string(FieldTypeSelect), DataTypeVarchar, opts),
	}, nil
}

type EmailField struct {
	BaseField
}

func (f *EmailField) ValidateValue(ctx context.Context, value interface{}) error {
	if err := f.BaseField.ValidateValue(ctx, value); err != nil {
		return err
	}
	if value == nil {
		return nil
	}
	str := fmt.Sprintf("%v", value)
	if str == "" {
		return nil
	}
	if len(str) < 3 || !contains(str, "@") || !contains(str, ".") {
		return fmt.Errorf("字段 %s 不是有效的邮箱地址", f.name)
	}
	return nil
}

func NewEmailField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &EmailField{
		BaseField: newBaseField(string(FieldTypeEmail), DataTypeVarchar, opts),
	}, nil
}

type URLField struct {
	BaseField
}

func (f *URLField) ValidateValue(ctx context.Context, value interface{}) error {
	if err := f.BaseField.ValidateValue(ctx, value); err != nil {
		return err
	}
	if value == nil {
		return nil
	}
	str := fmt.Sprintf("%v", value)
	if str == "" {
		return nil
	}
	if !startsWith(str, "http://") && !startsWith(str, "https://") {
		return fmt.Errorf("字段 %s 不是有效的URL地址", f.name)
	}
	return nil
}

func NewURLField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &URLField{
		BaseField: newBaseField(string(FieldTypeUrl), DataTypeVarchar, opts),
	}, nil
}

type PhoneField struct {
	BaseField
}

func (f *PhoneField) ValidateValue(ctx context.Context, value interface{}) error {
	if err := f.BaseField.ValidateValue(ctx, value); err != nil {
		return err
	}
	return nil
}

func NewPhoneField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &PhoneField{
		BaseField: newBaseField(string(FieldTypePhone), DataTypeVarchar, opts),
	}, nil
}

type JSONField struct {
	BaseField
}

func NewJSONField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &JSONField{
		BaseField: newBaseField(string(FieldTypeJSON), DataTypeJSON, opts),
	}, nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}
