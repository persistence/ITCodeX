// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// Fields is the golang structure for table fields.
type Fields struct {
	Id             int64       `json:"id"             orm:"id"              description:""` //
	CollectionName string      `json:"collectionName" orm:"collection_name" description:""` //
	Name           string      `json:"name"           orm:"name"            description:""` //
	Type           string      `json:"type"           orm:"type"            description:""` //
	DisplayName    string      `json:"displayName"    orm:"display_name"    description:""` //
	IsRequired     int         `json:"isRequired"     orm:"is_required"     description:""` //
	IsUnique       int         `json:"isUnique"       orm:"is_unique"       description:""` //
	IsIndexed      int         `json:"isIndexed"      orm:"is_indexed"      description:""` //
	Validation     string      `json:"validation"     orm:"validation"      description:""` //
	Options        string      `json:"options"        orm:"options"         description:""` //
	Sort           int         `json:"sort"           orm:"sort"            description:""` //
	CreatedAt      *gtime.Time `json:"createdAt"      orm:"created_at"      description:""` //
}
