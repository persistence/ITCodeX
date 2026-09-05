package metadata

import (
	"fmt"
	"strings"
)

type ValidationError struct {
	FieldErrors map[string][]string `json:"fieldErrors"`
	TableErrors []string            `json:"tableErrors"`
}

func (e *ValidationError) Error() string {
	var parts []string
	if len(e.FieldErrors) > 0 {
		var fieldParts []string
		for field, errs := range e.FieldErrors {
			fieldParts = append(fieldParts, fmt.Sprintf("%s: %s", field, strings.Join(errs, "; ")))
		}
		parts = append(parts, fmt.Sprintf("字段错误: %s", strings.Join(fieldParts, ", ")))
	}
	if len(e.TableErrors) > 0 {
		parts = append(parts, fmt.Sprintf("表级错误: %s", strings.Join(e.TableErrors, "; ")))
	}
	return strings.Join(parts, "; ")
}

func (e *ValidationError) HasErrors() bool {
	return len(e.FieldErrors) > 0 || len(e.TableErrors) > 0
}

func (e *ValidationError) AddFieldError(field, msg string) {
	if e.FieldErrors == nil {
		e.FieldErrors = make(map[string][]string)
	}
	e.FieldErrors[field] = append(e.FieldErrors[field], msg)
}

func (e *ValidationError) AddTableError(msg string) {
	e.TableErrors = append(e.TableErrors, msg)
}

func NewValidationError() *ValidationError {
	return &ValidationError{
		FieldErrors: make(map[string][]string),
		TableErrors: make([]string, 0),
	}
}

type NotFoundError struct {
	Resource string `json:"resource"`
	Key      string `json:"key"`
	Value    interface{} `json:"value"`
}

func (e *NotFoundError) Error() string {
	if e.Key != "" {
		return fmt.Sprintf("%s 不存在: %s=%v", e.Resource, e.Key, e.Value)
	}
	return fmt.Sprintf("%s 不存在", e.Resource)
}

func NewNotFoundError(resource string, key string, value interface{}) *NotFoundError {
	return &NotFoundError{
		Resource: resource,
		Key:      key,
		Value:    value,
	}
}

type AlreadyExistsError struct {
	Resource string `json:"resource"`
	Key      string `json:"key"`
	Value    interface{} `json:"value"`
}

func (e *AlreadyExistsError) Error() string {
	return fmt.Sprintf("%s 已存在: %s=%v", e.Resource, e.Key, e.Value)
}

func NewAlreadyExistsError(resource string, key string, value interface{}) *AlreadyExistsError {
	return &AlreadyExistsError{
		Resource: resource,
		Key:      key,
		Value:    value,
	}
}

type ForbiddenError struct {
	Message string `json:"message"`
}

func (e *ForbiddenError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "禁止访问"
}

func NewForbiddenError(message string) *ForbiddenError {
	return &ForbiddenError{Message: message}
}

type SystemError struct {
	Err error
}

func (e *SystemError) Error() string {
	return fmt.Sprintf("系统错误: %v", e.Err)
}

func (e *SystemError) Unwrap() error {
	return e.Err
}

func NewSystemError(err error) *SystemError {
	return &SystemError{Err: err}
}

type CompileError struct {
	RuleName string `json:"ruleName"`
	Message  string `json:"message"`
}

func (e *CompileError) Error() string {
	return fmt.Sprintf("规则 %s 编译错误: %s", e.RuleName, e.Message)
}

func NewCompileError(ruleName, message string) *CompileError {
	return &CompileError{
		RuleName: ruleName,
		Message:  message,
	}
}
