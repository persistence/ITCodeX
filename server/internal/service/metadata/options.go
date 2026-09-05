package metadata

import (
	"context"
)

type ctxKey string

// CtxActorID 请求上下文中的操作者 ID，写入 created_by / updated_by；无则留空。
const CtxActorID ctxKey = "metadata.actorId"

func ActorIDFromContext(ctx context.Context) (int64, bool) {
	if ctx == nil {
		return 0, false
	}
	switch v := ctx.Value(CtxActorID).(type) {
	case int64:
		return v, v > 0
	case int:
		return int64(v), v > 0
	default:
		return 0, false
	}
}

type DatabaseOptions struct {
	TablePrefix   string `json:"tablePrefix,omitempty"`
	Logging       bool   `json:"logging,omitempty"`
	DSN           string `json:"dsn,omitempty"`
	ScriptsPath   string `json:"scriptsPath,omitempty"`
	AllowTruncate bool   `json:"allowTruncate,omitempty"`
}

type CollectionOptions struct {
	SimplePagination   bool                   `json:"simplePagination,omitempty"`
	TreeParentKey      string                 `json:"treeParentKey,omitempty"`
	CalendarStartField string                 `json:"calendarStartField,omitempty"`
	CalendarEndField   string                 `json:"calendarEndField,omitempty"`
	CommentForeignKey  string                 `json:"commentForeignKey,omitempty"`
	Extra              map[string]interface{} `json:"extra,omitempty"`
}

type SortableConfig struct {
	Name     string `json:"name"`
	ScopeKey string `json:"scopeKey,omitempty"`
}

type Index struct {
	ID      int64                  `json:"id,omitempty"`
	Name    string                 `json:"name,omitempty"`
	Fields  []string               `json:"fields"`
	Unique  bool                   `json:"unique,omitempty"`
	Options map[string]interface{} `json:"options,omitempty"`
}

type CreateCollectionInput struct {
	Name            string                 `json:"name"`
	DisplayName     string                 `json:"displayName"`
	Type            CollectionType         `json:"type,omitempty"`
	Description     string                 `json:"description,omitempty"`
	Categories      []string               `json:"categories,omitempty"`
	Options         CollectionOptions      `json:"options,omitempty"`
	IsSystem        bool                   `json:"isSystem,omitempty"`
	AutoGenId       *bool                  `json:"autoGenId,omitempty"`
	FilterTargetKey string                 `json:"filterTargetKey,omitempty"`
	Sortable        *SortableConfig        `json:"sortable,omitempty"`
	PresetFields    []string               `json:"presetFields,omitempty"`
	Fields          []CreateFieldInput     `json:"fields,omitempty"`
	Indexes         []Index                `json:"indexes,omitempty"`
	TableValidation *TableValidationConfig `json:"tableValidation,omitempty"`
}

type CreateFieldInput struct {
	Name             string                 `json:"name"`
	DisplayName      string                 `json:"displayName"`
	Type             FieldType              `json:"type"`
	DataType         DataType               `json:"dataType,omitempty"`
	InterfaceType    string                 `json:"interfaceType,omitempty"`
	IsSystem         bool                   `json:"isSystem,omitempty"`
	IsRequired       bool                   `json:"isRequired,omitempty"`
	IsUnique         bool                   `json:"isUnique,omitempty"`
	IsIndexed        bool                   `json:"isIndexed,omitempty"`
	DefaultValue     interface{}            `json:"defaultValue,omitempty"`
	Description      string                 `json:"description,omitempty"`
	Sort             int                    `json:"sort,omitempty"`
	Options          map[string]interface{} `json:"options,omitempty"`
	Validation       *FieldValidationConfig `json:"validation,omitempty"`
	ValidationRules  []CELRule              `json:"validationRules,omitempty"`
	Target           string                 `json:"target,omitempty"`
	ForeignKey       string                 `json:"foreignKey,omitempty"`
	SourceKey        string                 `json:"sourceKey,omitempty"`
	Through          string                 `json:"through,omitempty"`
	OtherKey         string                 `json:"otherKey,omitempty"`
	TargetKey        string                 `json:"targetKey,omitempty"`
	AutoGenerate     bool                   `json:"autoGenerate,omitempty"`
	Length           int                    `json:"length,omitempty"`
	ScopeKey         string                 `json:"scopeKey,omitempty"`
	Expression       string                 `json:"expression,omitempty"`
	Pattern          string                 `json:"pattern,omitempty"`
	StartsAt         int                    `json:"startsAt,omitempty"`
	IncrementBy      int                    `json:"incrementBy,omitempty"`
	Algorithm        string                 `json:"algorithm,omitempty"`
	TargetCollection string                 `json:"targetCollection,omitempty"`
}

type FieldValidationConfig struct {
	Required     bool      `json:"required,omitempty"`
	Nullable     bool      `json:"nullable,omitempty"`
	Unique       bool      `json:"unique,omitempty"`
	MinLength    *int      `json:"minLength,omitempty"`
	MaxLength    *int      `json:"maxLength,omitempty"`
	Length       *int      `json:"length,omitempty"`
	Pattern      string    `json:"pattern,omitempty"`
	Min          *float64  `json:"min,omitempty"`
	Max          *float64  `json:"max,omitempty"`
	ExclusiveMin *float64  `json:"exclusiveMin,omitempty"`
	ExclusiveMax *float64  `json:"exclusiveMax,omitempty"`
	MultipleOf   *float64  `json:"multipleOf,omitempty"`
	Integer      bool      `json:"integer,omitempty"`
	Format       string    `json:"format,omitempty"`
	Rules        []CELRule `json:"rules,omitempty"`
}

type CELRule struct {
	Name         string `json:"name"`
	Expression   string `json:"expression"`
	ErrorMessage string `json:"errorMessage"`
}

type TableValidationConfig struct {
	UniqueConstraints []UniqueConstraint `json:"uniqueConstraints,omitempty"`
	Rules             []TableCELRule     `json:"rules,omitempty"`
}

type UniqueConstraint struct {
	Name   string   `json:"name"`
	Fields []string `json:"fields"`
}

type TableCELRule struct {
	Name         string   `json:"name"`
	Expression   string   `json:"expression"`
	ErrorMessage string   `json:"errorMessage"`
	Dependencies []string `json:"dependencies,omitempty"`
}

type Filter map[string]interface{}
type Sort []string
type Fields []string
type Appends []string

type CommonOptions struct {
	Ctx     context.Context
	Filter  Filter  `json:"filter,omitempty"`
	Fields  Fields  `json:"fields,omitempty"`
	Except  Fields  `json:"except,omitempty"`
	Appends Appends `json:"appends,omitempty"`
	Sort    Sort    `json:"sort,omitempty"`
}

type UpdateCollectionInput struct {
	DisplayName string                 `json:"displayName,omitempty"`
	Description string                 `json:"description,omitempty"`
	Categories  []string               `json:"categories,omitempty"`
	Options     CollectionOptions      `json:"options,omitempty"`
}

type UpdateFieldInput struct {
	DisplayName string                 `json:"displayName,omitempty"`
	IsRequired  *bool                  `json:"isRequired,omitempty"`
	IsUnique    *bool                  `json:"isUnique,omitempty"`
	IsIndexed   *bool                  `json:"isIndexed,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
	Validation  *FieldValidationConfig `json:"validation,omitempty"`
	Sort        *int                   `json:"sort,omitempty"`
}

type FindOptions struct {
	CommonOptions
	FilterByTk interface{} `json:"filterByTk,omitempty"`
	Limit      int         `json:"limit,omitempty"`
	Offset     int         `json:"offset,omitempty"`
	Page       int         `json:"page,omitempty"`
	PageSize   int         `json:"pageSize,omitempty"`
}

type FindOneOptions struct {
	CommonOptions
	FilterByTk interface{} `json:"filterByTk,omitempty"`
}

type CountOptions struct {
	CommonOptions
}

type CreateOptions struct {
	CommonOptions
	Values    map[string]interface{} `json:"values"`
	Whitelist []string               `json:"whitelist,omitempty"`
	Blacklist []string               `json:"blacklist,omitempty"`
}

type CreateManyOptions struct {
	CommonOptions
	Records   []map[string]interface{} `json:"records"`
	Whitelist []string                 `json:"whitelist,omitempty"`
	Blacklist []string                 `json:"blacklist,omitempty"`
}

type UpdateOptions struct {
	CommonOptions
	FilterByTk interface{}            `json:"filterByTk,omitempty"`
	Values     map[string]interface{} `json:"values"`
	Whitelist  []string               `json:"whitelist,omitempty"`
	Blacklist  []string               `json:"blacklist,omitempty"`
}

type DestroyOptions struct {
	CommonOptions
	FilterByTk interface{} `json:"filterByTk,omitempty"`
	Truncate   bool        `json:"truncate,omitempty"`
}

type FindAndCountResult struct {
	List       []map[string]interface{} `json:"list"`
	Total      int                      `json:"total"`
	Page       int                      `json:"page,omitempty"`
	PageSize   int                      `json:"pageSize,omitempty"`
	TotalPages int                      `json:"totalPages,omitempty"`
}
