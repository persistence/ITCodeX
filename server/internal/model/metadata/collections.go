package metadata

import "github.com/gogf/gf/v2/os/gtime"

type Collection struct {
	Id          int64       `json:"id"          orm:"id,primary"`
	Name        string      `json:"name"        orm:"name"`
	DisplayName string      `json:"displayName" orm:"display_name"`
	Type        string      `json:"type"        orm:"type"`
	Options     string      `json:"options"     orm:"options"`
	CreatedAt   *gtime.Time `json:"createdAt"   orm:"created_at"`
	UpdatedAt   *gtime.Time `json:"updatedAt"   orm:"updated_at"`
}

func (*Collection) TableName() string {
	return "_collections"
}
