package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	md "itcodex/server/internal/service/metadata"
)

type FieldListReq struct {
	g.Meta         `path:"/collections/{collectionName}/fields" method:"get" tags:"Meta" summary:"列出字段"`
	CollectionName string `json:"collectionName" p:"collectionName" v:"required"`
}
type FieldListRes struct {
	List []FieldItem `json:"list"`
}

type FieldCreateReq struct {
	g.Meta         `path:"/collections/{collectionName}/fields" method:"post" tags:"Meta" summary:"添加字段"`
	CollectionName string `json:"collectionName" p:"collectionName" v:"required"`
	md.CreateFieldInput
}
type FieldCreateRes struct {
	List []FieldItem `json:"list"`
}

type FieldUpdateReq struct {
	g.Meta         `path:"/collections/{collectionName}/fields/{fieldName}" method:"put" tags:"Meta" summary:"更新字段"`
	CollectionName string `json:"collectionName" p:"collectionName" v:"required"`
	FieldName      string `json:"fieldName" p:"fieldName" v:"required"`
	md.UpdateFieldInput
}
type FieldUpdateRes struct {
	List []FieldItem `json:"list"`
}

type FieldDeleteReq struct {
	g.Meta         `path:"/collections/{collectionName}/fields/{fieldName}" method:"delete" tags:"Meta" summary:"删除字段"`
	CollectionName string `json:"collectionName" p:"collectionName" v:"required"`
	FieldName      string `json:"fieldName" p:"fieldName" v:"required"`
}
type FieldDeleteRes struct{}
