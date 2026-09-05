package metadata

import (
	"fmt"

	"itcodex/server/pkg/utils"
)

type StringField struct {
	BaseField
}

func NewStringField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &StringField{
		BaseField: newBaseField(string(FieldTypeString), DataTypeVarchar, opts),
	}, nil
}

type TextField struct {
	BaseField
}

func NewTextField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &TextField{
		BaseField: newBaseField(string(FieldTypeText), DataTypeText, opts),
	}, nil
}

type IntegerField struct {
	BaseField
}

func NewIntegerField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &IntegerField{
		BaseField: newBaseField(string(FieldTypeInteger), DataTypeInteger, opts),
	}, nil
}

type BigintField struct {
	BaseField
}

func NewBigintField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &BigintField{
		BaseField: newBaseField("bigint", DataTypeBigInt, opts),
	}, nil
}

type FloatField struct {
	BaseField
}

func NewFloatField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &FloatField{
		BaseField: newBaseField("float", DataTypeFloat, opts),
	}, nil
}

type DoubleField struct {
	BaseField
}

func NewDoubleField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &DoubleField{
		BaseField: newBaseField("double", DataTypeDouble, opts),
	}, nil
}

type BooleanField struct {
	BaseField
}

func NewBooleanField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &BooleanField{
		BaseField: newBaseField(string(FieldTypeBoolean), DataTypeBoolean, opts),
	}, nil
}

type PasswordField struct {
	BaseField
}

func (f *PasswordField) ToStoreValue(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	s := castToString(value)
	if s == "" {
		return "", nil
	}
	// Already hashed (sha256 hex length 64)
	if len(s) == 64 && isHexString(s) {
		return s, nil
	}
	return utils.HashPassword(s), nil
}

func (f *PasswordField) FromStoreValue(value interface{}) (interface{}, error) {
	return f.BaseField.FromStoreValue(value)
}

func NewPasswordField(coll *Collection, opts map[string]interface{}) (Field, error) {
	bf := newBaseField(string(FieldTypePassword), DataTypeVarchar, opts)
	if bf.length <= 0 {
		bf.length = 128
	}
	return &PasswordField{BaseField: bf}, nil
}

func castToString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
