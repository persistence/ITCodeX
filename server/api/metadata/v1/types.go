package v1

type CollectionItem struct {
	Name        string         `json:"name"`
	DisplayName string         `json:"displayName"`
	TableName   string         `json:"tableName"`
	Type        string         `json:"type"`
	Description string         `json:"description,omitempty"`
	Categories  []string       `json:"categories,omitempty"`
	Options     map[string]any `json:"options"`
	FieldCount  int            `json:"fieldCount"`
	Fields      []FieldItem    `json:"fields,omitempty"`
}

type FieldItem struct {
	Name        string         `json:"name"`
	DisplayName string         `json:"displayName"`
	Type        string         `json:"type"`
	Required    bool           `json:"required"`
	Unique      bool           `json:"unique"`
	Indexed     bool           `json:"indexed"`
	IsSystem    bool           `json:"isSystem"`
	Options     map[string]any `json:"options"`
	Validation  any            `json:"validation,omitempty"`
}
