package metadata

import (
	"context"
	"fmt"
	"strings"
)

// RelationOptions extracted from field options.
type RelationOptions struct {
	Target     string
	ForeignKey string
	SourceKey  string
	Through    string
	OtherKey   string
	TargetKey  string
}

func relationOptsFromMap(opts map[string]interface{}, fieldName string, typ FieldType) RelationOptions {
	ro := RelationOptions{
		SourceKey: DefaultPrimaryKey,
		TargetKey: DefaultPrimaryKey,
	}
	if opts == nil {
		opts = map[string]interface{}{}
	}
	if v, ok := opts["target"].(string); ok {
		ro.Target = v
	}
	if v, ok := opts["foreignKey"].(string); ok {
		ro.ForeignKey = v
	}
	if v, ok := opts["sourceKey"].(string); ok && v != "" {
		ro.SourceKey = v
	}
	if v, ok := opts["through"].(string); ok {
		ro.Through = v
	}
	if v, ok := opts["otherKey"].(string); ok {
		ro.OtherKey = v
	}
	if v, ok := opts["targetKey"].(string); ok && v != "" {
		ro.TargetKey = v
	}
	switch typ {
	case FieldTypeBelongsTo:
		if ro.ForeignKey == "" {
			ro.ForeignKey = fieldName
		}
	case FieldTypeHasMany, FieldTypeHasOne:
		if ro.ForeignKey == "" && ro.Target != "" {
			ro.ForeignKey = strings.ToLower(fieldName) + "_id"
		}
	case FieldTypeBelongsToMany:
		if ro.ForeignKey == "" {
			ro.ForeignKey = "source_id"
		}
		if ro.OtherKey == "" {
			ro.OtherKey = "target_id"
		}
		if ro.Through == "" && ro.Target != "" {
			ro.Through = fieldName + "_" + ro.Target
		}
	}
	return ro
}

func attachRelationInput(opts map[string]interface{}, input CreateFieldInput) {
	if input.Target != "" {
		opts["target"] = input.Target
	}
	if input.ForeignKey != "" {
		opts["foreignKey"] = input.ForeignKey
	}
	if input.SourceKey != "" {
		opts["sourceKey"] = input.SourceKey
	}
	if input.Through != "" {
		opts["through"] = input.Through
	}
	if input.OtherKey != "" {
		opts["otherKey"] = input.OtherKey
	}
	if input.TargetKey != "" {
		opts["targetKey"] = input.TargetKey
	}
	if input.Expression != "" {
		opts["expression"] = input.Expression
	}
	if input.Pattern != "" {
		opts["pattern"] = input.Pattern
	}
	if input.ScopeKey != "" {
		opts["scopeKey"] = input.ScopeKey
	}
	if input.AutoGenerate {
		opts["autoGenerate"] = true
	}
	if input.StartsAt != 0 {
		opts["startsAt"] = input.StartsAt
	}
	if input.IncrementBy != 0 {
		opts["incrementBy"] = input.IncrementBy
	}
	if input.Algorithm != "" {
		opts["algorithm"] = input.Algorithm
	}
	if input.TargetCollection != "" {
		opts["targetCollection"] = input.TargetCollection
	}
}

func isVirtualFieldType(t string) bool {
	switch FieldType(t) {
	case FieldTypeHasMany, FieldTypeHasOne, FieldTypeBelongsToMany:
		return true
	default:
		return false
	}
}

func isVirtualField(f Field) bool {
	if f == nil {
		return false
	}
	if isVirtualFieldType(f.Type()) {
		return true
	}
	if opts := f.Options(); opts != nil {
		if v, ok := opts["virtual"].(bool); ok && v {
			return true
		}
	}
	return false
}

func isRelationField(f Field) bool {
	if f == nil {
		return false
	}
	switch FieldType(f.Type()) {
	case FieldTypeBelongsTo, FieldTypeHasMany, FieldTypeHasOne, FieldTypeBelongsToMany, FieldTypeBelongsToManyArray:
		return true
	}
	return false
}

func GetRelationOptions(f Field) RelationOptions {
	name := ""
	typ := FieldType("")
	opts := map[string]interface{}{}
	if f != nil {
		name = f.Name()
		typ = FieldType(f.Type())
		if f.Options() != nil {
			opts = f.Options()
		}
	}
	return relationOptsFromMap(opts, name, typ)
}

type BelongsToField struct {
	BaseField
}

func NewBelongsToField(coll *Collection, opts map[string]interface{}) (Field, error) {
	name, _, _, _, _, _, _, _ := parseBaseFieldOptions(opts)
	ro := relationOptsFromMap(opts, name, FieldTypeBelongsTo)
	if ro.Target == "" {
		return nil, fmt.Errorf("belongsTo 字段 %s 需要 target", name)
	}
	opts["target"] = ro.Target
	opts["foreignKey"] = ro.ForeignKey
	opts["sourceKey"] = ro.SourceKey
	bf := newBaseField(string(FieldTypeBelongsTo), DataTypeBigInt, opts)
	// Physical column is the foreign key name
	bf.name = ro.ForeignKey
	if opts["name"] != nil {
		// Keep logical name as field name from input; store FK name separately
		if n, ok := opts["name"].(string); ok && n != "" {
			bf.name = n
		}
	}
	// Column name for DDL: use foreignKey when different? For simplicity FK column = field name.
	if ro.ForeignKey != "" && ro.ForeignKey != bf.name {
		opts["columnName"] = ro.ForeignKey
	}
	return &BelongsToField{BaseField: bf}, nil
}

type HasManyField struct {
	BaseField
}

func NewHasManyField(coll *Collection, opts map[string]interface{}) (Field, error) {
	name, _, _, _, _, _, _, _ := parseBaseFieldOptions(opts)
	ro := relationOptsFromMap(opts, name, FieldTypeHasMany)
	if ro.Target == "" {
		return nil, fmt.Errorf("hasMany 字段 %s 需要 target", name)
	}
	opts["target"] = ro.Target
	opts["foreignKey"] = ro.ForeignKey
	opts["sourceKey"] = ro.SourceKey
	opts["virtual"] = true
	bf := newBaseField(string(FieldTypeHasMany), DataTypeJSON, opts)
	return &HasManyField{BaseField: bf}, nil
}

func (f *HasManyField) DDLColumn() string {
	return ""
}

type HasOneField struct {
	BaseField
}

func NewHasOneField(coll *Collection, opts map[string]interface{}) (Field, error) {
	name, _, _, _, _, _, _, _ := parseBaseFieldOptions(opts)
	ro := relationOptsFromMap(opts, name, FieldTypeHasOne)
	if ro.Target == "" {
		return nil, fmt.Errorf("hasOne 字段 %s 需要 target", name)
	}
	opts["target"] = ro.Target
	opts["foreignKey"] = ro.ForeignKey
	opts["sourceKey"] = ro.SourceKey
	opts["virtual"] = true
	bf := newBaseField(string(FieldTypeHasOne), DataTypeJSON, opts)
	return &HasOneField{BaseField: bf}, nil
}

func (f *HasOneField) DDLColumn() string {
	return ""
}

type BelongsToManyField struct {
	BaseField
}

func NewBelongsToManyField(coll *Collection, opts map[string]interface{}) (Field, error) {
	name, _, _, _, _, _, _, _ := parseBaseFieldOptions(opts)
	ro := relationOptsFromMap(opts, name, FieldTypeBelongsToMany)
	if ro.Target == "" {
		return nil, fmt.Errorf("belongsToMany 字段 %s 需要 target", name)
	}
	opts["target"] = ro.Target
	opts["through"] = ro.Through
	opts["foreignKey"] = ro.ForeignKey
	opts["otherKey"] = ro.OtherKey
	opts["sourceKey"] = ro.SourceKey
	opts["targetKey"] = ro.TargetKey
	opts["virtual"] = true
	bf := newBaseField(string(FieldTypeBelongsToMany), DataTypeJSON, opts)
	return &BelongsToManyField{BaseField: bf}, nil
}

func (f *BelongsToManyField) DDLColumn() string {
	return ""
}

func ensureThroughTableDDL(ctx context.Context, db *Database, through, fk, otherKey string) error {
	if db == nil || through == "" {
		return nil
	}
	ddl := fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s (%s BIGINT NOT NULL, %s BIGINT NOT NULL, PRIMARY KEY (%s, %s), KEY %s (%s), KEY %s (%s)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		quoteIdent(through),
		quoteIdent(fk), quoteIdent(otherKey),
		quoteIdent(fk), quoteIdent(otherKey),
		quoteIdent("idx_"+through+"_"+fk), quoteIdent(fk),
		quoteIdent("idx_"+through+"_"+otherKey), quoteIdent(otherKey),
	)
	_, err := db.DB().Exec(ctx, ddl)
	return err
}
