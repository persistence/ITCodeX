// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// Indexes is the golang structure for table indexes.
type Indexes struct {
	Id             int64       `json:"id"             orm:"id"              description:""` //
	CollectionName string      `json:"collectionName" orm:"collection_name" description:""` //
	Name           string      `json:"name"           orm:"name"            description:""` //
	Fields         string      `json:"fields"         orm:"fields"          description:""` //
	Unique         int         `json:"unique"         orm:"unique"          description:""` //
	Options        string      `json:"options"        orm:"options"         description:""` //
	CreatedAt      *gtime.Time `json:"createdAt"      orm:"created_at"      description:""` //
}
