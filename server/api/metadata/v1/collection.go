package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	md "itcodex/server/internal/service/metadata"
)

type CollectionListReq struct {
	g.Meta   `path:"/collections" method:"get" tags:"Meta" summary:"列出 Collection"`
	Keyword  string `json:"keyword" p:"keyword"`
	Category string `json:"category" p:"category"`
	Page     int    `json:"page" p:"page"`
	PageSize int    `json:"pageSize" p:"pageSize"`
}
type CollectionListRes struct {
	List  []CollectionItem `json:"list"`
	Total int              `json:"total"`
}

type CollectionCreateReq struct {
	g.Meta `path:"/collections" method:"post" tags:"Meta" summary:"创建 Collection"`
	md.CreateCollectionInput
}
type CollectionCreateRes struct {
	*CollectionItem
}

type CollectionGetReq struct {
	g.Meta         `path:"/collections/{collectionName}" method:"get" tags:"Meta" summary:"获取 Collection"`
	CollectionName string `json:"collectionName" p:"collectionName" v:"required"`
}
type CollectionGetRes struct {
	*CollectionItem
}

type CollectionUpdateReq struct {
	g.Meta         `path:"/collections/{collectionName}" method:"put" tags:"Meta" summary:"更新 Collection"`
	CollectionName string `json:"collectionName" p:"collectionName" v:"required"`
	md.UpdateCollectionInput
}
type CollectionUpdateRes struct {
	*CollectionItem
}

type CollectionDeleteReq struct {
	g.Meta         `path:"/collections/{collectionName}" method:"delete" tags:"Meta" summary:"删除 Collection"`
	CollectionName string `json:"collectionName" p:"collectionName" v:"required"`
}
type CollectionDeleteRes struct{}

type CollectionSyncReq struct {
	g.Meta         `path:"/collections/{collectionName}/sync" method:"post" tags:"Meta" summary:"同步 Collection 表结构"`
	CollectionName string `json:"collectionName" p:"collectionName" v:"required"`
}
type CollectionSyncRes struct {
	*CollectionItem
}
