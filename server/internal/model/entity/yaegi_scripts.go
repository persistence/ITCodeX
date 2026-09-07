// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// YaegiScripts is the golang structure for table yaegi_scripts.
type YaegiScripts struct {
	Id             int64       `json:"id"             orm:"id"              description:""` //
	CollectionName string      `json:"collectionName" orm:"collection_name" description:""` //
	Name           string      `json:"name"           orm:"name"            description:""` //
	HookPoint      string      `json:"hookPoint"      orm:"hook_point"      description:""` //
	Content        string      `json:"content"        orm:"content"         description:""` //
	ApiPath        string      `json:"apiPath"        orm:"api_path"        description:""` //
	HttpMethod     string      `json:"httpMethod"     orm:"http_method"     description:""` //
	Enabled        int         `json:"enabled"        orm:"enabled"         description:""` //
	Priority       int         `json:"priority"       orm:"priority"        description:""` //
	Options        string      `json:"options"        orm:"options"         description:""` //
	CreatedAt      *gtime.Time `json:"createdAt"      orm:"created_at"      description:""` //
	UpdatedAt      *gtime.Time `json:"updatedAt"      orm:"updated_at"      description:""` //
}
