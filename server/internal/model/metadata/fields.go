package metadata

import "github.com/gogf/gf/v2/os/gtime"

type Field struct {
	Id             int64       `json:"id"             orm:"id,primary"`
	CollectionName string      `json:"collectionName" orm:"collection_name"`
	Name           string      `json:"name"           orm:"name"`
	Type           string      `json:"type"           orm:"type"`
	DisplayName    string      `json:"displayName"    orm:"display_name"`
	IsRequired     bool        `json:"isRequired"     orm:"is_required"`
	IsUnique       bool        `json:"isUnique"       orm:"is_unique"`
	IsIndexed      bool        `json:"isIndexed"      orm:"is_indexed"`
	Validation     string      `json:"validation"     orm:"validation"`
	Options        string      `json:"options"        orm:"options"`
	Sort           int         `json:"sort"           orm:"sort"`
	CreatedAt      *gtime.Time `json:"createdAt"      orm:"created_at"`
}

func (*Field) TableName() string {
	return "_fields"
}
