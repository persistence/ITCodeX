package client

import "fmt"

type Collection struct {
	Name        string         `json:"name"`
	DisplayName string         `json:"displayName"`
	Type        string         `json:"type"`
	Description string         `json:"description,omitempty"`
	FieldCount  int            `json:"fieldCount,omitempty"`
	Options     map[string]any `json:"options,omitempty"`
	Fields      []Field        `json:"fields,omitempty"`
}

type CreateCollectionInput struct {
	Name         string             `json:"name"`
	DisplayName  string             `json:"displayName"`
	Type         string             `json:"type"`
	Description  string             `json:"description,omitempty"`
	Options      map[string]any     `json:"options,omitempty"`
	PresetFields []string           `json:"presetFields,omitempty"`
	Fields       []CreateFieldInput `json:"fields,omitempty"`
	Indexes      []Index            `json:"indexes,omitempty"`
}

type UpdateCollectionInput struct {
	DisplayName string         `json:"displayName,omitempty"`
	Description string         `json:"description,omitempty"`
	Options     map[string]any `json:"options,omitempty"`
}

type Field struct {
	Name         string         `json:"name"`
	DisplayName  string         `json:"displayName"`
	Type         string         `json:"type"`
	Required     bool           `json:"required"`
	Unique       bool           `json:"unique"`
	Indexed      bool           `json:"indexed"`
	IsSystem     bool           `json:"isSystem"`
	DefaultValue any            `json:"defaultValue,omitempty"`
	Options      map[string]any `json:"options,omitempty"`
}

type CreateFieldInput struct {
	Name         string         `json:"name"`
	DisplayName  string         `json:"displayName"`
	Type         string         `json:"type"`
	IsRequired   bool           `json:"isRequired,omitempty"`
	IsUnique     bool           `json:"isUnique,omitempty"`
	IsIndexed    bool           `json:"isIndexed,omitempty"`
	DefaultValue any            `json:"defaultValue,omitempty"`
	Options      map[string]any `json:"options,omitempty"`
	// Relation / advanced
	Target       string `json:"target,omitempty"`
	ForeignKey   string `json:"foreignKey,omitempty"`
	SourceKey    string `json:"sourceKey,omitempty"`
	Through      string `json:"through,omitempty"`
	OtherKey     string `json:"otherKey,omitempty"`
	TargetKey    string `json:"targetKey,omitempty"`
	Expression   string `json:"expression,omitempty"`
	Pattern      string `json:"pattern,omitempty"`
	AutoGenerate bool   `json:"autoGenerate,omitempty"`
	StartsAt     int    `json:"startsAt,omitempty"`
	IncrementBy  int    `json:"incrementBy,omitempty"`
}

type UpdateFieldInput struct {
	DisplayName string         `json:"displayName,omitempty"`
	IsRequired  *bool          `json:"isRequired,omitempty"`
	IsUnique    *bool          `json:"isUnique,omitempty"`
	IsIndexed   *bool          `json:"isIndexed,omitempty"`
	Options     map[string]any `json:"options,omitempty"`
}

type Index struct {
	Name   string   `json:"name,omitempty"`
	Fields []string `json:"fields"`
	Unique bool     `json:"unique,omitempty"`
}

type FindOptions struct {
	Filter   Filter
	Sort     []string
	Fields   []string
	Except   []string
	Appends  []string
	Page     int
	PageSize int
}

type FindOneOptions struct {
	Fields  []string
	Except  []string
	Appends []string
}

type ListResult struct {
	List       []map[string]any `json:"list"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"pageSize"`
	TotalPages int              `json:"totalPages"`
}

type Filter map[string]any

type Script struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Content        string `json:"content"`
	HookPoint      string `json:"hookPoint"`
	CollectionName string `json:"collectionName,omitempty"`
	APIPath        string `json:"apiPath,omitempty"`
	HTTPMethod     string `json:"httpMethod,omitempty"`
	Enabled        bool   `json:"enabled"`
	Priority       int    `json:"priority,omitempty"`
}

type CreateScriptInput struct {
	ID             int64  `json:"id,omitempty"`
	Name           string `json:"name"`
	Content        string `json:"content"`
	HookPoint      string `json:"hookPoint"`
	CollectionName string `json:"collectionName,omitempty"`
	APIPath        string `json:"apiPath,omitempty"`
	HTTPMethod     string `json:"httpMethod,omitempty"`
	Priority       int    `json:"priority,omitempty"`
	Enabled        bool   `json:"enabled,omitempty"`
}

type ValidateScriptResult struct {
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type APIError struct {
	Code    int
	Message string
	Data    any
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.Code, e.Message)
}

type ValidationError struct {
	FieldErrors map[string][]string `json:"fieldErrors"`
	TableErrors []string            `json:"tableErrors"`
}
