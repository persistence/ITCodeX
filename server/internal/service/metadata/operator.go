package metadata

import (
	"fmt"
	"strings"

	"github.com/spf13/cast"
)

type OperatorFunc func(columnName string, value interface{}, params *[]interface{}) (string, error)

var operators = make(map[string]OperatorFunc)

func RegisterOperator(name string, fn OperatorFunc) {
	operators[name] = fn
}

func GetOperator(name string) (OperatorFunc, bool) {
	fn, ok := operators[name]
	return fn, ok
}

func quoteColumn(name string) string {
	return quoteIdent(name)
}

func init() {
	RegisterOperator("$eq", func(columnName string, value interface{}, params *[]interface{}) (string, error) {
		col := quoteColumn(columnName)
		if value == nil {
			return fmt.Sprintf("%s IS NULL", col), nil
		}
		*params = append(*params, value)
		return fmt.Sprintf("%s = ?", col), nil
	})

	RegisterOperator("$ne", func(columnName string, value interface{}, params *[]interface{}) (string, error) {
		col := quoteColumn(columnName)
		if value == nil {
			return fmt.Sprintf("%s IS NOT NULL", col), nil
		}
		*params = append(*params, value)
		return fmt.Sprintf("%s != ?", col), nil
	})

	RegisterOperator("$gt", func(columnName string, value interface{}, params *[]interface{}) (string, error) {
		col := quoteColumn(columnName)
		*params = append(*params, value)
		return fmt.Sprintf("%s > ?", col), nil
	})

	RegisterOperator("$gte", func(columnName string, value interface{}, params *[]interface{}) (string, error) {
		col := quoteColumn(columnName)
		*params = append(*params, value)
		return fmt.Sprintf("%s >= ?", col), nil
	})

	RegisterOperator("$lt", func(columnName string, value interface{}, params *[]interface{}) (string, error) {
		col := quoteColumn(columnName)
		*params = append(*params, value)
		return fmt.Sprintf("%s < ?", col), nil
	})

	RegisterOperator("$lte", func(columnName string, value interface{}, params *[]interface{}) (string, error) {
		col := quoteColumn(columnName)
		*params = append(*params, value)
		return fmt.Sprintf("%s <= ?", col), nil
	})

	RegisterOperator("$in", func(columnName string, value interface{}, params *[]interface{}) (string, error) {
		col := quoteColumn(columnName)
		var slice []interface{}
		switch v := value.(type) {
		case []interface{}:
			slice = v
		default:
			slice = cast.ToSlice(value)
		}
		if len(slice) == 0 {
			return "1=0", nil
		}
		placeholders := make([]string, len(slice))
		for i, item := range slice {
			placeholders[i] = "?"
			*params = append(*params, item)
		}
		return fmt.Sprintf("%s IN (%s)", col, strings.Join(placeholders, ", ")), nil
	})

	RegisterOperator("$notIn", func(columnName string, value interface{}, params *[]interface{}) (string, error) {
		col := quoteColumn(columnName)
		var slice []interface{}
		switch v := value.(type) {
		case []interface{}:
			slice = v
		default:
			slice = cast.ToSlice(value)
		}
		if len(slice) == 0 {
			return "1=1", nil
		}
		placeholders := make([]string, len(slice))
		for i, item := range slice {
			placeholders[i] = "?"
			*params = append(*params, item)
		}
		return fmt.Sprintf("%s NOT IN (%s)", col, strings.Join(placeholders, ", ")), nil
	})

	RegisterOperator("$like", func(columnName string, value interface{}, params *[]interface{}) (string, error) {
		col := quoteColumn(columnName)
		*params = append(*params, value)
		return fmt.Sprintf("%s LIKE ?", col), nil
	})

	RegisterOperator("$notLike", func(columnName string, value interface{}, params *[]interface{}) (string, error) {
		col := quoteColumn(columnName)
		*params = append(*params, value)
		return fmt.Sprintf("%s NOT LIKE ?", col), nil
	})

	RegisterOperator("$isNull", func(columnName string, value interface{}, params *[]interface{}) (string, error) {
		col := quoteColumn(columnName)
		return fmt.Sprintf("%s IS NULL", col), nil
	})

	RegisterOperator("$notNull", func(columnName string, value interface{}, params *[]interface{}) (string, error) {
		col := quoteColumn(columnName)
		return fmt.Sprintf("%s IS NOT NULL", col), nil
	})

	RegisterOperator("$between", func(columnName string, value interface{}, params *[]interface{}) (string, error) {
		col := quoteColumn(columnName)
		var slice []interface{}
		switch v := value.(type) {
		case []interface{}:
			slice = v
		default:
			slice = cast.ToSlice(value)
		}
		if len(slice) != 2 {
			return "", fmt.Errorf("$between requires an array of 2 values")
		}
		*params = append(*params, slice[0], slice[1])
		return fmt.Sprintf("%s BETWEEN ? AND ?", col), nil
	})

	RegisterOperator("$notBetween", func(columnName string, value interface{}, params *[]interface{}) (string, error) {
		col := quoteColumn(columnName)
		var slice []interface{}
		switch v := value.(type) {
		case []interface{}:
			slice = v
		default:
			slice = cast.ToSlice(value)
		}
		if len(slice) != 2 {
			return "", fmt.Errorf("$notBetween requires an array of 2 values")
		}
		*params = append(*params, slice[0], slice[1])
		return fmt.Sprintf("%s NOT BETWEEN ? AND ?", col), nil
	})

	RegisterOperator("$startsWith", func(columnName string, value interface{}, params *[]interface{}) (string, error) {
		col := quoteColumn(columnName)
		str := cast.ToString(value)
		*params = append(*params, str+"%")
		return fmt.Sprintf("%s LIKE ?", col), nil
	})

	RegisterOperator("$endsWith", func(columnName string, value interface{}, params *[]interface{}) (string, error) {
		col := quoteColumn(columnName)
		str := cast.ToString(value)
		*params = append(*params, "%"+str)
		return fmt.Sprintf("%s LIKE ?", col), nil
	})
}
