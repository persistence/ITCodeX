package metadata

import "github.com/gogf/gf/v2/os/gtime"

type Index struct {
	Id             int64       `json:"id"             orm:"id,primary"`
	CollectionName string      `json:"collectionName" orm:"collection_name"`
	Name           string      `json:"name"           orm:"name"`
	Fields         string      `json:"fields"         orm:"fields"`
	Unique         bool        `json:"unique"         orm:"unique"`
	Options        string      `json:"options"        orm:"options"`
	CreatedAt      *gtime.Time `json:"createdAt"      orm:"created_at"`
}

func (*Index) TableName() string {
	return "_indexes"
}
