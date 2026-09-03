package metadata

import "github.com/gogf/gf/v2/os/gtime"

type YaegiScript struct {
	Id             int64       `json:"id"             orm:"id,primary"`
	CollectionName string      `json:"collectionName" orm:"collection_name"`
	Name           string      `json:"name"           orm:"name"`
	HookPoint      string      `json:"hookPoint"      orm:"hook_point"`
	Content        string      `json:"content"        orm:"content"`
	APIPath        string      `json:"apiPath"        orm:"api_path"`
	HTTPMethod     string      `json:"httpMethod"     orm:"http_method"`
	Enabled        bool        `json:"enabled"        orm:"enabled"`
	Priority       int         `json:"priority"       orm:"priority"`
	Options        string      `json:"options"        orm:"options"`
	CreatedAt      *gtime.Time `json:"createdAt"      orm:"created_at"`
	UpdatedAt      *gtime.Time `json:"updatedAt"      orm:"updated_at"`
}

func (*YaegiScript) TableName() string {
	return "_yaegi_scripts"
}
