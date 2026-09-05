package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gogf/gf/v2/os/gcfg"
	"github.com/spf13/cast"

	"itcodex/server/pkg/utils"
)

var (
	sequenceMu       sync.Mutex
	sequenceCounters = map[string]int{}
)

type FormulaField struct {
	BaseField
}

func (f *FormulaField) ToStoreValue(value interface{}) (interface{}, error) {
	return value, nil
}

func NewFormulaField(coll *Collection, opts map[string]interface{}) (Field, error) {
	bf := newBaseField(string(FieldTypeFormula), DataTypeVarchar, opts)
	return &FormulaField{BaseField: bf}, nil
}

// EvaluateFormula computes expression against data using CEL when available.
func EvaluateFormula(coll *Collection, expression string, data map[string]interface{}) (interface{}, error) {
	if expression == "" {
		return nil, nil
	}
	if coll == nil || coll.Db() == nil || coll.Db().Validator() == nil {
		return nil, fmt.Errorf("CEL validator unavailable")
	}
	v := coll.Db().Validator()
	prog, err := v.compile(expression)
	if err != nil {
		return nil, err
	}
	out, err := v.evalProgram(prog, map[string]interface{}{"data": data, "oldData": map[string]interface{}{}})
	if err != nil {
		return nil, err
	}
	return out.Value(), nil
}

type EncryptedField struct {
	BaseField
}

func encryptKeyFromConfig() string {
	if v, err := gcfg.Instance().Get(context.Background(), "metadata.encryptKey"); err == nil && v != nil {
		if s := v.String(); s != "" {
			return s
		}
	}
	return "itcodex-default-encrypt-key"
}

func (f *EncryptedField) ToStoreValue(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	s := cast.ToString(value)
	if s == "" {
		return "", nil
	}
	return utils.EncryptAESGCM(encryptKeyFromConfig(), s)
}

func (f *EncryptedField) FromStoreValue(value interface{}) (interface{}, error) {
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
		s = cast.ToString(value)
	}
	if s == "" {
		return "", nil
	}
	plain, err := utils.DecryptAESGCM(encryptKeyFromConfig(), s)
	if err != nil {
		return s, nil
	}
	return plain, nil
}

func NewEncryptedField(coll *Collection, opts map[string]interface{}) (Field, error) {
	bf := newBaseField(string(FieldTypeEncrypted), DataTypeText, opts)
	return &EncryptedField{BaseField: bf}, nil
}

type GeoJSONField struct {
	BaseField
}

func (f *GeoJSONField) ValidateValue(ctx context.Context, value interface{}) error {
	if err := f.BaseField.ValidateValue(ctx, value); err != nil {
		return err
	}
	if value == nil {
		return nil
	}
	var m map[string]interface{}
	switch v := value.(type) {
	case map[string]interface{}:
		m = v
	case string:
		if err := json.Unmarshal([]byte(v), &m); err != nil {
			return fmt.Errorf("字段 %s 需要 GeoJSON 对象", f.name)
		}
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("字段 %s 需要 GeoJSON 对象", f.name)
		}
		if err := json.Unmarshal(b, &m); err != nil {
			return fmt.Errorf("字段 %s 需要 GeoJSON 对象", f.name)
		}
	}
	if _, ok := m["type"]; !ok {
		return fmt.Errorf("字段 %s GeoJSON 缺少 type", f.name)
	}
	if _, ok := m["coordinates"]; !ok {
		if f.fieldType != string(FieldTypeCircle) {
			return fmt.Errorf("字段 %s GeoJSON 缺少 coordinates", f.name)
		}
	}
	return nil
}

func (f *GeoJSONField) ToStoreValue(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	b, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (f *GeoJSONField) FromStoreValue(value interface{}) (interface{}, error) {
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

func NewPointField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &GeoJSONField{BaseField: newBaseField(string(FieldTypePoint), DataTypeJSON, opts)}, nil
}
func NewLineStringField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &GeoJSONField{BaseField: newBaseField(string(FieldTypeLineString), DataTypeJSON, opts)}, nil
}
func NewPolygonField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &GeoJSONField{BaseField: newBaseField(string(FieldTypePolygon), DataTypeJSON, opts)}, nil
}
func NewCircleField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &GeoJSONField{BaseField: newBaseField(string(FieldTypeCircle), DataTypeJSON, opts)}, nil
}

type SortField struct {
	BaseField
}

func NewSortField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &SortField{BaseField: newBaseField(string(FieldTypeSort), DataTypeInteger, opts)}, nil
}

type SequenceField struct {
	BaseField
}

func NewSequenceField(coll *Collection, opts map[string]interface{}) (Field, error) {
	bf := newBaseField(string(FieldTypeSequence), DataTypeVarchar, opts)
	if bf.length <= 0 {
		bf.length = 64
	}
	return &SequenceField{BaseField: bf}, nil
}

func nextSequenceValue(collName, fieldName, pattern string, startsAt, incrementBy int) string {
	if startsAt <= 0 {
		startsAt = 1
	}
	if incrementBy <= 0 {
		incrementBy = 1
	}
	key := collName + "." + fieldName
	sequenceMu.Lock()
	sequenceCounters[key] += incrementBy
	n := sequenceCounters[key]
	if n < startsAt {
		n = startsAt
		sequenceCounters[key] = n
	}
	sequenceMu.Unlock()
	now := time.Now()
	out := pattern
	if out == "" {
		out = "{0000}"
	}
	out = strings.ReplaceAll(out, "{YYYY}", strconv.Itoa(now.Year()))
	out = strings.ReplaceAll(out, "{MM}", fmt.Sprintf("%02d", int(now.Month())))
	out = strings.ReplaceAll(out, "{DD}", fmt.Sprintf("%02d", now.Day()))
	re := regexp.MustCompile(`\{0+\}`)
	out = re.ReplaceAllStringFunc(out, func(m string) string {
		width := len(m) - 2
		if width < 1 {
			width = 4
		}
		return fmt.Sprintf("%0*d", width, n)
	})
	return out
}

type UUIDField struct {
	BaseField
}

func (f *UUIDField) ToStoreValue(value interface{}) (interface{}, error) {
	if value == nil || cast.ToString(value) == "" {
		return utils.UUID(), nil
	}
	return value, nil
}

func NewUUIDField(coll *Collection, opts map[string]interface{}) (Field, error) {
	bf := newBaseField(string(FieldTypeUUID), DataTypeUUID, opts)
	opts["autoGenerate"] = true
	return &UUIDField{BaseField: bf}, nil
}

type NanoIDField struct {
	BaseField
}

func (f *NanoIDField) ToStoreValue(value interface{}) (interface{}, error) {
	if value == nil || cast.ToString(value) == "" {
		return utils.NanoID(), nil
	}
	return value, nil
}

func NewNanoIDField(coll *Collection, opts map[string]interface{}) (Field, error) {
	bf := newBaseField(string(FieldTypeNanoID), DataTypeVarchar, opts)
	if bf.length <= 0 {
		bf.length = 21
	}
	opts["autoGenerate"] = true
	return &NanoIDField{BaseField: bf}, nil
}

type BelongsToManyArrayField struct {
	BaseField
}

func (f *BelongsToManyArrayField) ToStoreValue(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	b, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (f *BelongsToManyArrayField) FromStoreValue(value interface{}) (interface{}, error) {
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

func NewBelongsToManyArrayField(coll *Collection, opts map[string]interface{}) (Field, error) {
	return &BelongsToManyArrayField{
		BaseField: newBaseField(string(FieldTypeBelongsToManyArray), DataTypeJSON, opts),
	}, nil
}
