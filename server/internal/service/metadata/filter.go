package metadata

import (
	"fmt"
	"strings"
)

func BuildWhereClause(filter Filter, params *[]interface{}) (string, error) {
	if len(filter) == 0 {
		return "1=1", nil
	}

	conditions := make([]string, 0, len(filter))

	for key, value := range filter {
		if strings.HasPrefix(key, "$") {
			cond, err := buildLogicalCondition(key, value, params)
			if err != nil {
				return "", err
			}
			conditions = append(conditions, cond)
		} else {
			cond, err := buildFieldCondition(key, value, params)
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

func buildLogicalCondition(op string, value interface{}, params *[]interface{}) (string, error) {
	switch op {
	case "$and":
		return buildAndOrCondition("AND", value, params)
	case "$or":
		return buildAndOrCondition("OR", value, params)
	case "$not":
		return buildNotCondition(value, params)
	default:
		return "", fmt.Errorf("unknown logical operator: %s", op)
	}
}

func buildAndOrCondition(logicOp string, value interface{}, params *[]interface{}) (string, error) {
	filters, err := toFilterSlice(value)
	if err != nil {
		return "", err
	}
	if len(filters) == 0 {
		return "1=1", nil
	}
	parts := make([]string, 0, len(filters))
	for _, f := range filters {
		cond, err := BuildWhereClause(f, params)
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

func buildNotCondition(value interface{}, params *[]interface{}) (string, error) {
	filter, ok := value.(Filter)
	if !ok {
		m, ok := value.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("$not requires a Filter object")
		}
		filter = Filter(m)
	}
	cond, err := BuildWhereClause(filter, params)
	if err != nil {
		return "", err
	}
	return "NOT (" + cond + ")", nil
}

func buildFieldCondition(columnName string, value interface{}, params *[]interface{}) (string, error) {
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
			cond, err := buildLogicalCondition(opName, opValue, params)
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
