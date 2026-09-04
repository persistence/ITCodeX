package metadata

import (
	"context"
	"fmt"
	"sync"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/types"
	"cel.dev/cel-go/common/types/ref"
)

type CELValidator struct {
	mu    sync.RWMutex
	cache map[string]*cel.Program
	env   *cel.Env
}

func NewCELValidator() *CELValidator {
	env, err := cel.NewEnv(
		cel.Variable("data", cel.DynType),
		cel.Variable("oldData", cel.DynType),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create CEL environment: %v", err))
	}
	return &CELValidator{
		cache: make(map[string]*cel.Program),
		env:   env,
	}
}

func (v *CELValidator) ValidateRecord(ctx context.Context, coll *Collection, data map[string]interface{}, oldData map[string]interface{}, isUpdate bool) error {
	validationErr := NewValidationError()

	for _, field := range coll.Fields() {
		fieldName := field.Name()
		value, exists := data[fieldName]

		if !isUpdate && field.IsRequired() && !field.IsSystem() {
			if !exists || isEmptyValue(value) {
				validationErr.AddFieldError(fieldName, "该字段为必填项")
				continue
			}
		}

		if exists && !isEmptyValue(value) {
			if err := field.ValidateValue(ctx, value); err != nil {
				validationErr.AddFieldError(fieldName, err.Error())
			}

			// unique check
			if field.IsUnique() && !field.IsSystem() {
				if err := checkUnique(ctx, coll, fieldName, value, oldData); err != nil {
					validationErr.AddFieldError(fieldName, err.Error())
				}
			}
		}

		if exists {
			fieldOpts := field.Options()
			if valRules, ok := fieldOpts["validationRules"].([]interface{}); ok {
				for _, r := range valRules {
					if ruleMap, ok := r.(map[string]interface{}); ok {
						expr, _ := ruleMap["expression"].(string)
						errMsg, _ := ruleMap["errorMessage"].(string)
						if expr != "" {
							prog, err := v.compile(expr)
							if err != nil {
								validationErr.AddFieldError(fieldName, fmt.Sprintf("CEL规则编译失败: %v", err))
								continue
							}
							vars := map[string]interface{}{
								"data":    data,
								"oldData": oldData,
							}
							result, err := v.evalProgram(prog, vars)
							if err != nil {
								validationErr.AddFieldError(fieldName, fmt.Sprintf("CEL规则执行失败: %v", err))
								continue
							}
							if boolVal, ok := result.Value().(bool); ok && !boolVal {
								if errMsg == "" {
									errMsg = "字段校验失败"
								}
								validationErr.AddFieldError(fieldName, errMsg)
							}
						}
					}
				}
			}
		}
	}

	tableCfg := coll.GetTableValidation()
	if tableCfg != nil && len(tableCfg.Rules) > 0 {
		for _, rule := range tableCfg.Rules {
			prog, err := v.compile(rule.Expression)
			if err != nil {
				validationErr.AddTableError(fmt.Sprintf("表级规则[%s]编译失败: %v", rule.Name, err))
				continue
			}
			vars := map[string]interface{}{
				"data":    data,
				"oldData": oldData,
			}
			result, err := v.evalProgram(prog, vars)
			if err != nil {
				validationErr.AddTableError(fmt.Sprintf("表级规则[%s]执行失败: %v", rule.Name, err))
				continue
			}
			if boolVal, ok := result.Value().(bool); ok && !boolVal {
				errMsg := rule.ErrorMessage
				if errMsg == "" {
					errMsg = fmt.Sprintf("表级校验规则[%s]验证失败", rule.Name)
				}
				validationErr.AddTableError(errMsg)
			}
		}
	}

	if validationErr.HasErrors() {
		return validationErr
	}

	return nil
}

func (v *CELValidator) compile(expr string) (*cel.Program, error) {
	v.mu.RLock()
	if prog, ok := v.cache[expr]; ok {
		v.mu.RUnlock()
		return prog, nil
	}
	v.mu.RUnlock()

	v.mu.Lock()
	defer v.mu.Unlock()

	if prog, ok := v.cache[expr]; ok {
		return prog, nil
	}

	ast, issues := v.env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}

	prog, err := v.env.Program(ast)
	if err != nil {
		return nil, err
	}

	v.cache[expr] = &prog
	return &prog, nil
}

func (v *CELValidator) evalProgram(prog *cel.Program, vars map[string]interface{}) (ref.Val, error) {
	celVars := make(map[string]interface{})
	for k, val := range vars {
		if m, ok := val.(map[string]interface{}); ok && m != nil {
			celVars[k] = types.NewStringInterfaceMap(types.DefaultTypeAdapter, m)
		} else {
			celVars[k] = val
		}
	}

	out, _, err := (*prog).Eval(celVars)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (v *CELValidator) ClearCache() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.cache = make(map[string]*cel.Program)
}

// checkUnique verifies that the given field value does not already exist in the collection.
// When updating, oldData is used to exclude the current record from the uniqueness check.
func checkUnique(ctx context.Context, coll *Collection, fieldName string, value interface{}, oldData map[string]interface{}) error {
	if coll.Db() == nil || coll.Db().db == nil {
		return nil
	}

	query := fmt.Sprintf(`SELECT COUNT(*) FROM "%s" WHERE "%s" = ?`, coll.TableName(), fieldName)
	args := []interface{}{value}

	// when updating, exclude self
	if oldData != nil {
		if oldID, ok := oldData[DefaultPrimaryKey]; ok && oldID != nil {
			query += fmt.Sprintf(` AND "%s" != ?`, DefaultPrimaryKey)
			args = append(args, oldID)
		}
	}

	var count int
	row := coll.Db().DB().QueryRow(ctx, query, args...)
	if err := row.Scan(&count); err != nil {
		return NewSystemError(err)
	}
	if count > 0 {
		return fmt.Errorf("该字段值已存在，必须唯一")
	}
	return nil
}
