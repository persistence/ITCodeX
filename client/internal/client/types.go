package client

import "fmt"

type Collection struct {
	Name        string                 `json:"name"`
	DisplayName string                 `json:"displayName"`
	Type        string                 `json:"type"`
	Description string                 `json:"description,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

type CreateCollectionInput struct {
	Name        string                 `json:"name"`
	DisplayName string                 `json:"displayName"`
	Type        string                 `json:"type"`
	Description string                 `json:"description,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
	Fields      []CreateFieldInput     `json:"fields,omitempty"`
}

type Field struct {
	Name         string                 `json:"name"`
	DisplayName  string                 `json:"displayName"`
	Type         string                 `json:"type"`
	Required     bool                   `json:"required"`
	Unique       bool                   `json:"unique"`
	Indexed      bool                   `json:"indexed"`
	IsSystem     bool                   `json:"isSystem"`
	DefaultValue interface{}            `json:"defaultValue,omitempty"`
	Options      map[string]interface{} `json:"options,omitempty"`
}

type CreateFieldInput struct {
	Name         string                 `json:"name"`
	DisplayName  string                 `json:"displayName"`
	Type         string                 `json:"type"`
	Required     bool                   `json:"required,omitempty"`
	Unique       bool                   `json:"unique,omitempty"`
	Indexed      bool                   `json:"indexed,omitempty"`
	DefaultValue interface{}            `json:"defaultValue,omitempty"`
	Options      map[string]interface{} `json:"options,omitempty"`
}

type FindOptions struct {
	Filter   Filter
	Sort     []string
	Fields   []string
	Except   []string
	Page     int
	PageSize int
}

type FindOneOptions struct {
	Fields []string
	Except []string
}

type ListResult struct {
	List       []map[string]interface{} `json:"list"`
	Total      int64                    `json:"total"`
	Page       int                      `json:"page"`
	PageSize   int                      `json:"pageSize"`
	TotalPages int                      `json:"totalPages"`
}

type Filter map[string]interface{}

type Script struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Content   string `json:"content"`
	HookPoint string `json:"hookPoint"`
	Enabled   bool   `json:"enabled"`
}

type CreateScriptInput struct {
	Name           string `json:"name"`
	Content        string `json:"content"`
	HookPoint      string `json:"hookPoint"`
	CollectionName string `json:"collectionName,omitempty"`
	APIPath        string `json:"apiPath,omitempty"`
	HTTPMethod     string `json:"httpMethod,omitempty"`
	Priority       int    `json:"priority,omitempty"`
}

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type APIError struct {
	Code    int
	Message string
	Data    interface{}
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.Code, e.Message)
}

type ValidationError struct {
	FieldErrors map[string][]string `json:"fieldErrors"`
	TableErrors []string            `json:"tableErrors"`
}
