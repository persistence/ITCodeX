package metadata

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

func NewPasswordField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &PasswordField{
		BaseField: newBaseField(string(FieldTypePassword), DataTypeVarchar, opts),
	}, nil
}
