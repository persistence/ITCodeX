package metadata

import (
	"fmt"
)

type FieldFactory func(coll *Collection, opts map[string]interface{}) (Field, error)

var defaultFieldFactories = map[FieldType]FieldFactory{
	FieldTypeString:             wrapFactory(NewStringField),
	FieldTypeText:               wrapFactory(NewTextField),
	FieldTypeInteger:            wrapFactory(NewIntegerField),
	"bigint":                    wrapFactory(NewBigintField),
	"float":                     wrapFactory(NewFloatField),
	"double":                    wrapFactory(NewDoubleField),
	FieldTypeBoolean:            wrapFactory(NewBooleanField),
	FieldTypePassword:           wrapFactory(NewPasswordField),
	FieldTypeDateTime:           wrapFactory(NewDatetimeField),
	FieldTypeDate:               wrapFactory(NewDateField),
	FieldTypeSelect:             wrapFactory(NewSelectField),
	FieldTypeEmail:              wrapFactory(NewEmailField),
	FieldTypeUrl:                wrapFactory(NewURLField),
	FieldTypePhone:              wrapFactory(NewPhoneField),
	FieldTypeJSON:               wrapFactory(NewJSONField),
	FieldTypeRadio:              wrapFactory(NewRadioField),
	FieldTypeMultiSelect:        wrapFactory(NewMultiSelectField),
	FieldTypeCheckboxGroup:      wrapFactory(NewCheckboxGroupField),
	FieldTypeMarkdown:           wrapFactory(NewMarkdownField),
	FieldTypeRichText:           wrapFactory(NewRichTextField),
	FieldTypeAttachmentUrl:      wrapFactory(NewAttachmentURLField),
	FieldTypeTime:               wrapFactory(NewTimeField),
	FieldTypeUnixTimestamp:      wrapFactory(NewUnixTimestampField),
	FieldTypePercent:            wrapFactory(NewPercentField),
	FieldTypeColor:              wrapFactory(NewColorField),
	FieldTypeBelongsTo:          wrapFactory(NewBelongsToField),
	FieldTypeHasMany:            wrapFactory(NewHasManyField),
	FieldTypeHasOne:             wrapFactory(NewHasOneField),
	FieldTypeBelongsToMany:      wrapFactory(NewBelongsToManyField),
	FieldTypeBelongsToManyArray: wrapFactory(NewBelongsToManyArrayField),
	FieldTypeFormula:            wrapFactory(NewFormulaField),
	FieldTypeEncrypted:          wrapFactory(NewEncryptedField),
	FieldTypePoint:              wrapFactory(NewPointField),
	FieldTypeLineString:         wrapFactory(NewLineStringField),
	FieldTypePolygon:            wrapFactory(NewPolygonField),
	FieldTypeCircle:             wrapFactory(NewCircleField),
	FieldTypeSort:               wrapFactory(NewSortField),
	FieldTypeSequence:           wrapFactory(NewSequenceField),
	FieldTypeUUID:               wrapFactory(NewUUIDField),
	FieldTypeNanoID:             wrapFactory(NewNanoIDField),
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
