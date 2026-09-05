package metadata

import (
	"fmt"
)

type FieldFactory func(coll *Collection, opts map[string]interface{}) (Field, error)

var defaultFieldFactories = map[FieldType]FieldFactory{
	FieldTypeString:   wrapFactory(NewStringField),
	FieldTypeText:     wrapFactory(NewTextField),
	FieldTypeInteger:  wrapFactory(NewIntegerField),
	"bigint":          wrapFactory(NewBigintField),
	"float":           wrapFactory(NewFloatField),
	"double":          wrapFactory(NewDoubleField),
	FieldTypeBoolean:  wrapFactory(NewBooleanField),
	FieldTypePassword: wrapFactory(NewPasswordField),
	FieldTypeDateTime: wrapFactory(NewDatetimeField),
	FieldTypeDate:     wrapFactory(NewDateField),
	FieldTypeSelect:   wrapFactory(NewSelectField),
	FieldTypeEmail:    wrapFactory(NewEmailField),
	FieldTypeUrl:      wrapFactory(NewURLField),
	FieldTypePhone:    wrapFactory(NewPhoneField),
	FieldTypeJSON:     wrapFactory(NewJSONField),
}

func wrapFactory(f func(coll *Collection, opts map[string]interface{}) (Field, error)) FieldFactory {
	return f
}

func RegisterFieldFactory(fieldType FieldType, factory FieldFactory) {
	defaultFieldFactories[fieldType] = factory
}

func NewField(coll *Collection, fieldType FieldType, opts map[string]interface{}) (Field, error) {
	factory, ok := defaultFieldFactories[fieldType]
	if !ok {
		return nil, fmt.Errorf("不支持的字段类型: %s", fieldType)
	}
	return factory(coll, opts)
}

func GetRegisteredFieldTypes() []FieldType {
	types := make([]FieldType, 0, len(defaultFieldFactories))
	for t := range defaultFieldFactories {
		types = append(types, t)
	}
	return types
}
