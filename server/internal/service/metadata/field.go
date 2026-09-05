package metadata

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cast"
)

type Field interface {
	Name() string
	Type() string
	DBType() DataType
	IsRequired() bool
	IsUnique() bool
	IsIndexed() bool
	IsSystem() bool
	DisplayName() string
	Options() map[string]interface{}
	SetOptions(opts map[string]interface{})
	ValidateValue(ctx context.Context, value interface{}) error
	ToStoreValue(value interface{}) (interface{}, error)
	FromStoreValue(value interface{}) (interface{}, error)
	DDLColumn() string
	RefCollectionName() string
}

type BaseField struct {
	name        string
	fieldType   string
	dbType      DataType
	required    bool
	unique      bool
	indexed     bool
	system      bool
	displayName string
	options     map[string]interface{}
	length      int
	refCollName string
}

func (f *BaseField) Name() string                    { return f.name }
func (f *BaseField) Type() string                    { return f.fieldType }
func (f *BaseField) DBType() DataType                { return f.dbType }
func (f *BaseField) IsRequired() bool                { return f.required }
func (f *BaseField) IsUnique() bool                  { return f.unique }
func (f *BaseField) IsIndexed() bool                 { return f.indexed }
func (f *BaseField) IsSystem() bool                  { return f.system }
func (f *BaseField) DisplayName() string             { return f.displayName }
func (f *BaseField) Options() map[string]interface{} { return f.options }
func (f *BaseField) RefCollectionName() string       { return f.refCollName }

func (f *BaseField) SetOptions(opts map[string]interface{}) {
	f.options = opts
}

func (f *BaseField) ValidateValue(ctx context.Context, value interface{}) error {
	if f.required && isEmptyValue(value) {
		return fmt.Errorf("字段 %s 不能为空", f.name)
	}
	if isEmptyValue(value) {
		return nil
	}
	return applyBuiltinValidation(f.name, value, validationFromOptions(f.options))
}

func validationFromOptions(opts map[string]interface{}) *FieldValidationConfig {
	if opts == nil {
		return nil
	}
	raw, ok := opts["validation"]
	if !ok || raw == nil {
		return nil
	}
	if cfg, ok := raw.(*FieldValidationConfig); ok {
		return cfg
	}
	m, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}
	cfg := &FieldValidationConfig{}
	if v, ok := m["pattern"].(string); ok {
		cfg.Pattern = v
	}
	if v, ok := m["format"].(string); ok {
		cfg.Format = v
	}
	if v, exists := m["minLength"]; exists {
		n := cast.ToInt(v)
		cfg.MinLength = &n
	}
	if v, exists := m["maxLength"]; exists {
		n := cast.ToInt(v)
		cfg.MaxLength = &n
	}
	if v, exists := m["length"]; exists {
		n := cast.ToInt(v)
		cfg.Length = &n
	}
	if v, exists := m["min"]; exists {
		n := cast.ToFloat64(v)
		cfg.Min = &n
	}
	if v, exists := m["max"]; exists {
		n := cast.ToFloat64(v)
		cfg.Max = &n
	}
	return cfg
}

func applyBuiltinValidation(fieldName string, value interface{}, cfg *FieldValidationConfig) error {
	if cfg == nil {
		return nil
	}
	if s, ok := value.(string); ok {
		if cfg.MinLength != nil && len(s) < *cfg.MinLength {
			return fmt.Errorf("字段 %s 长度不能小于 %d", fieldName, *cfg.MinLength)
		}
		if cfg.MaxLength != nil && len(s) > *cfg.MaxLength {
			return fmt.Errorf("字段 %s 长度不能大于 %d", fieldName, *cfg.MaxLength)
		}
		if cfg.Length != nil && len(s) != *cfg.Length {
			return fmt.Errorf("字段 %s 长度必须为 %d", fieldName, *cfg.Length)
		}
		if cfg.Pattern != "" {
			ok, err := regexp.MatchString(cfg.Pattern, s)
			if err != nil || !ok {
				return fmt.Errorf("字段 %s 格式不正确", fieldName)
			}
		}
	}
	if cfg.Min != nil || cfg.Max != nil {
		n := cast.ToFloat64(value)
		if cfg.Min != nil && n < *cfg.Min {
			return fmt.Errorf("字段 %s 不能小于 %v", fieldName, *cfg.Min)
		}
		if cfg.Max != nil && n > *cfg.Max {
			return fmt.Errorf("字段 %s 不能大于 %v", fieldName, *cfg.Max)
		}
	}
	return nil
}

func attachFieldInputOptions(opts map[string]interface{}, input CreateFieldInput) {
	if input.Validation != nil {
		opts["validation"] = input.Validation
	}
	if len(input.ValidationRules) > 0 {
		opts["validationRules"] = input.ValidationRules
	}
	if input.Description != "" {
		opts["description"] = input.Description
	}
	if input.DefaultValue != nil {
		opts["defaultValue"] = input.DefaultValue
	}
}

func (f *BaseField) ToStoreValue(value interface{}) (interface{}, error) {
	return value, nil
}

func (f *BaseField) FromStoreValue(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	switch f.dbType {
	case DataTypeVarchar, DataTypeText, DataTypeLongText, DataTypeUUID, DataTypeJSON, DataTypeJSONB:
		switch v := value.(type) {
		case []byte:
			return string(v), nil
		}
	case DataTypeBoolean:
		switch v := value.(type) {
		case int64:
			return v != 0, nil
		case int:
			return v != 0, nil
		case []byte:
			if len(v) == 0 {
				return false, nil
			}
			return v[0] != 0 && string(v) != "0", nil
		}
	}
	return value, nil
}

func (f *BaseField) DDLColumn() string {
	var b strings.Builder
	b.WriteString(quoteIdent(f.name))
	b.WriteString(" ")
	switch f.dbType {
	case DataTypeVarchar:
		l := f.length
		if l <= 0 {
			l = 255
		}
		b.WriteString(fmt.Sprintf("VARCHAR(%d)", l))
	case DataTypeText:
		b.WriteString("TEXT")
	case DataTypeLongText:
		b.WriteString("LONGTEXT")
	case DataTypeInteger:
		b.WriteString("INT")
	case DataTypeBigInt:
		b.WriteString("BIGINT")
	case DataTypeFloat:
		b.WriteString("FLOAT")
	case DataTypeDouble:
		b.WriteString("DOUBLE")
	case DataTypeDecimal:
		b.WriteString("DECIMAL")
	case DataTypeBoolean:
		b.WriteString("TINYINT(1)")
	case DataTypeDate:
		b.WriteString("DATE")
	case DataTypeTime:
		b.WriteString("TIME")
	case DataTypeDateTime:
		b.WriteString("DATETIME")
	case DataTypeTimestamp:
		b.WriteString("TIMESTAMP")
	case DataTypeJSON:
		b.WriteString("JSON")
	case DataTypeJSONB:
		b.WriteString("JSON")
	case DataTypeUUID:
		b.WriteString("CHAR(36)")
	case DataTypeBlob:
		b.WriteString("BLOB")
	default:
		b.WriteString(strings.ToUpper(string(f.dbType)))
	}
	return b.String()
}

func isEmptyValue(v interface{}) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case string:
		return val == ""
	case int:
		return val == 0
	case int8:
		return val == 0
	case int16:
		return val == 0
	case int32:
		return val == 0
	case int64:
		return val == 0
	case uint:
		return val == 0
	case uint8:
		return val == 0
	case uint16:
		return val == 0
	case uint32:
		return val == 0
	case uint64:
		return val == 0
	case float32:
		return val == 0
	case float64:
		return val == 0
	case bool:
		return false
	default:
		return false
	}
}

func parseBaseFieldOptions(opts map[string]interface{}) (name, displayName string, required, unique, indexed, system bool, length int, refCollName string) {
	if opts == nil {
		return
	}
	if v, ok := opts["name"].(string); ok {
		name = v
	}
	if v, ok := opts["displayName"].(string); ok {
		displayName = v
	}
	if v, ok := opts["required"].(bool); ok {
		required = v
	}
	if v, ok := opts["unique"].(bool); ok {
		unique = v
	}
	if v, ok := opts["indexed"].(bool); ok {
		indexed = v
	}
	if v, ok := opts["isSystem"].(bool); ok {
		system = v
	}
	if v, ok := opts["length"].(int); ok {
		length = v
	}
	if v, ok := opts["refCollectionName"].(string); ok {
		refCollName = v
	}
	return
}

func newBaseField(fieldType string, dbType DataType, opts map[string]interface{}) BaseField {
	name, displayName, required, unique, indexed, system, length, refCollName := parseBaseFieldOptions(opts)
	return BaseField{
		name:        name,
		fieldType:   fieldType,
		dbType:      dbType,
		required:    required,
		unique:      unique,
		indexed:     indexed,
		system:      system,
		displayName: displayName,
		options:     opts,
		length:      length,
		refCollName: refCollName,
	}
}
