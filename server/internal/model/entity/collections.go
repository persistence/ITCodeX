// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// Collections is the golang structure for table collections.
type Collections struct {
	Id          int64       `json:"id"          orm:"id"           description:""` //
	Name        string      `json:"name"        orm:"name"         description:""` //
	DisplayName string      `json:"displayName" orm:"display_name" description:""` //
	Type        string      `json:"type"        orm:"type"         description:""` //
	Options     string      `json:"options"     orm:"options"      description:""` //
	CreatedAt   *gtime.Time `json:"createdAt"   orm:"created_at"   description:""` //
	UpdatedAt   *gtime.Time `json:"updatedAt"   orm:"updated_at"   description:""` //
}
