package metadata

import (
	"context"
	"encoding/json"
	"fmt"
)

// RadioField is a single-select stored as varchar.
type RadioField struct {
	SelectField
}

func NewRadioField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &RadioField{
		SelectField: SelectField{BaseField: newBaseField(string(FieldTypeRadio), DataTypeVarchar, opts)},
	}, nil
}

// MultiSelectField stores selected values as JSON array.
type MultiSelectField struct {
	BaseField
}

func (f *MultiSelectField) ToStoreValue(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	b, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (f *MultiSelectField) FromStoreValue(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	var s string
	switch v := value.(type) {
	case []byte:
		s = string(v)
	case string:
		s = v
	default:
		return value, nil
	}
	var out interface{}
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return s, nil
	}
	return out, nil
}

func (f *MultiSelectField) ValidateValue(ctx context.Context, value interface{}) error {
	if err := f.BaseField.ValidateValue(ctx, value); err != nil {
		return err
	}
	return nil
}

func NewMultiSelectField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &MultiSelectField{
		BaseField: newBaseField(string(FieldTypeMultiSelect), DataTypeJSON, opts),
	}, nil
}

func NewCheckboxGroupField(coll *Collection, opts map[string]interface{}) (Field, error) {
	f, err := NewMultiSelectField(coll, opts)
	if err != nil {
		return nil, err
	}
	ms := f.(*MultiSelectField)
	ms.fieldType = string(FieldTypeCheckboxGroup)
	return ms, nil
}

type MarkdownField struct {
	BaseField
}

func NewMarkdownField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &MarkdownField{
		BaseField: newBaseField(string(FieldTypeMarkdown), DataTypeLongText, opts),
	}, nil
}

type RichTextField struct {
	BaseField
}

func NewRichTextField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &RichTextField{
		BaseField: newBaseField(string(FieldTypeRichText), DataTypeLongText, opts),
	}, nil
}

type AttachmentURLField struct {
	BaseField
}

func NewAttachmentURLField(coll *Collection, opts map[string]interface{}) (Field, error) {
	bf := newBaseField(string(FieldTypeAttachmentUrl), DataTypeVarchar, opts)
	if bf.length <= 0 {
		bf.length = 2048
	}
	return &AttachmentURLField{BaseField: bf}, nil
}

type TimeField struct {
	BaseField
}

func NewTimeField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &TimeField{
		BaseField: newBaseField(string(FieldTypeTime), DataTypeTime, opts),
	}, nil
}

type UnixTimestampField struct {
	BaseField
}

func NewUnixTimestampField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &UnixTimestampField{
		BaseField: newBaseField(string(FieldTypeUnixTimestamp), DataTypeBigInt, opts),
	}, nil
}

type PercentField struct {
	BaseField
}

func (f *PercentField) ValidateValue(ctx context.Context, value interface{}) error {
	if err := f.BaseField.ValidateValue(ctx, value); err != nil {
		return err
	}
	if value == nil {
		return nil
	}
	return nil
}

func NewPercentField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &PercentField{
		BaseField: newBaseField(string(FieldTypePercent), DataTypeFloat, opts),
	}, nil
}

type ColorField struct {
	BaseField
}

func (f *ColorField) ValidateValue(ctx context.Context, value interface{}) error {
	if err := f.BaseField.ValidateValue(ctx, value); err != nil {
		return err
	}
	if value == nil {
		return nil
	}
	s := fmt.Sprintf("%v", value)
	if s == "" {
		return nil
	}
	if len(s) > 0 && s[0] != '#' {
		return fmt.Errorf("字段 %s 颜色值应以 # 开头", f.name)
	}
	return nil
}

func NewColorField(coll *Collection, opts map[string]interface{}) (Field, error) {
	bf := newBaseField(string(FieldTypeColor), DataTypeVarchar, opts)
	if bf.length <= 0 {
		bf.length = 32
	}
	return &ColorField{BaseField: bf}, nil
}
