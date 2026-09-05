package metadata

import (
	"fmt"
	"strings"
)

func BuildWhereClause(filter Filter, params *[]interface{}) (string, error) {
	return BuildWhereClauseWithCollection(nil, filter, params)
}

func BuildWhereClauseWithCollection(coll *Collection, filter Filter, params *[]interface{}) (string, error) {
	if len(filter) == 0 {
		return "1=1", nil
	}

	conditions := make([]string, 0, len(filter))

	for key, value := range filter {
		if strings.HasPrefix(key, "$") {
			cond, err := buildLogicalCondition(coll, key, value, params)
			if err != nil {
				return "", err
			}
			conditions = append(conditions, cond)
		} else {
			cond, err := buildFieldCondition(columnName(key), value, params, coll)
			if err != nil {
				return "", err
			}
			conditions = append(conditions, cond)
		}
	}

	if len(conditions) == 0 {
		return "1=1", nil
	}
	if len(conditions) == 1 {
		return conditions[0], nil
	}
	return "(" + strings.Join(conditions, " AND ") + ")", nil
}

func columnName(key string) string { return key }

func buildLogicalCondition(coll *Collection, op string, value interface{}, params *[]interface{}) (string, error) {
	switch op {
	case "$and":
		return buildAndOrCondition(coll, "AND", value, params)
	case "$or":
		return buildAndOrCondition(coll, "OR", value, params)
	case "$not":
		return buildNotCondition(coll, value, params)
	default:
		return "", fmt.Errorf("unknown logical operator: %s", op)
	}
}

func buildAndOrCondition(coll *Collection, logicOp string, value interface{}, params *[]interface{}) (string, error) {
	filters, err := toFilterSlice(value)
	if err != nil {
		return "", err
	}
	if len(filters) == 0 {
		return "1=1", nil
	}
	parts := make([]string, 0, len(filters))
	for _, f := range filters {
		cond, err := BuildWhereClauseWithCollection(coll, f, params)
		if err != nil {
			return "", err
		}
		parts = append(parts, cond)
	}
	if len(parts) == 1 {
		return parts[0], nil
	}
	return "(" + strings.Join(parts, " "+logicOp+" ") + ")", nil
}

func buildNotCondition(coll *Collection, value interface{}, params *[]interface{}) (string, error) {
	filter, ok := value.(Filter)
	if !ok {
		m, ok := value.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("$not requires a Filter object")
		}
		filter = Filter(m)
	}
	cond, err := BuildWhereClauseWithCollection(coll, filter, params)
	if err != nil {
		return "", err
	}
	return "NOT (" + cond + ")", nil
}

func buildFieldCondition(columnName string, value interface{}, params *[]interface{}, coll *Collection) (string, error) {
	// Association filter: posts.title -> EXISTS subquery (one level)
	if coll != nil && strings.Contains(columnName, ".") {
		parts := strings.SplitN(columnName, ".", 2)
		assocName, targetCol := parts[0], parts[1]
		f := coll.GetField(assocName)
		if f != nil && isRelationField(f) {
			return buildAssociationFilter(coll, f, targetCol, value, params)
		}
	}

	switch v := value.(type) {
	case Filter:
		return buildOperatorCondition(columnName, v, params)
	case map[string]interface{}:
		return buildOperatorCondition(columnName, Filter(v), params)
	default:
		opFn, _ := GetOperator("$eq")
		return opFn(columnName, value, params)
	}
}

func buildAssociationFilter(coll *Collection, f Field, targetCol string, value interface{}, params *[]interface{}) (string, error) {
	ro := GetRelationOptions(f)
	target := coll.Db().Collection(ro.Target)
	if target == nil {
		return "1=0", nil
	}
	var inner string
	var err error
	switch v := value.(type) {
	case Filter:
		inner, err = buildOperatorCondition(targetCol, v, params)
	case map[string]interface{}:
		inner, err = buildOperatorCondition(targetCol, Filter(v), params)
	default:
		opFn, _ := GetOperator("$eq")
		inner, err = opFn(targetCol, value, params)
	}
	if err != nil {
		return "", err
	}
	srcTable := quoteIdent(coll.TableName())
	tgtTable := quoteIdent(target.TableName())
	switch FieldType(f.Type()) {
	case FieldTypeBelongsTo:
		fk := ro.ForeignKey
		if fk == "" {
			fk = f.Name()
		}
		return fmt.Sprintf("EXISTS (SELECT 1 FROM %s WHERE %s.%s = %s.%s AND %s)",
			tgtTable, tgtTable, quoteIdent(ro.TargetKey), srcTable, quoteIdent(fk), inner), nil
	case FieldTypeHasMany, FieldTypeHasOne:
		return fmt.Sprintf("EXISTS (SELECT 1 FROM %s WHERE %s.%s = %s.%s AND %s)",
			tgtTable, tgtTable, quoteIdent(ro.ForeignKey), srcTable, quoteIdent(ro.SourceKey), inner), nil
	case FieldTypeBelongsToMany:
		through := quoteIdent(ro.Through)
		return fmt.Sprintf("EXISTS (SELECT 1 FROM %s JOIN %s ON %s.%s = %s.%s WHERE %s.%s = %s.%s AND %s)",
			through, tgtTable, through, quoteIdent(ro.OtherKey), tgtTable, quoteIdent(ro.TargetKey),
			through, quoteIdent(ro.ForeignKey), srcTable, quoteIdent(ro.SourceKey), inner), nil
	default:
		return "1=1", nil
	}
}

func buildOperatorCondition(columnName string, opMap Filter, params *[]interface{}) (string, error) {
	if len(opMap) == 0 {
		return "1=1", nil
	}

	conditions := make([]string, 0, len(opMap))
	for opName, opValue := range opMap {
		if !strings.HasPrefix(opName, "$") {
			return "", fmt.Errorf("operator must start with $: %s", opName)
		}
		if opName == "$and" || opName == "$or" || opName == "$not" {
			cond, err := buildLogicalCondition(nil, opName, opValue, params)
			if err != nil {
				return "", err
			}
			conditions = append(conditions, cond)
			continue
		}
		opFn, ok := GetOperator(opName)
		if !ok {
			return "", fmt.Errorf("unknown operator: %s", opName)
		}
		cond, err := opFn(columnName, opValue, params)
		if err != nil {
			return "", err
		}
		conditions = append(conditions, cond)
	}

	if len(conditions) == 0 {
		return "1=1", nil
	}
	if len(conditions) == 1 {
		return conditions[0], nil
	}
	return "(" + strings.Join(conditions, " AND ") + ")", nil
}

func toFilterSlice(value interface{}) ([]Filter, error) {
	switch v := value.(type) {
	case []Filter:
		return v, nil
	case []interface{}:
		result := make([]Filter, 0, len(v))
		for i, item := range v {
			switch f := item.(type) {
			case Filter:
				result = append(result, f)
			case map[string]interface{}:
				result = append(result, Filter(f))
			default:
				return nil, fmt.Errorf("element %d in logical operator array must be a Filter object", i)
			}
		}
		return result, nil
	default:
		return nil, fmt.Errorf("logical operator $and/$or requires an array of Filter objects")
	}
}
