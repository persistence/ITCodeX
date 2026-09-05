package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	md "itcodex/server/internal/service/metadata"
)

type IndexListReq struct {
	g.Meta         `path:"/collections/{collectionName}/indexes" method:"get" tags:"Meta" summary:"列出索引"`
	CollectionName string `json:"collectionName" p:"collectionName" v:"required"`
}

type IndexListRes struct {
	List []md.Index `json:"list"`
}

type IndexCreateReq struct {
	g.Meta         `path:"/collections/{collectionName}/indexes" method:"post" tags:"Meta" summary:"创建索引"`
	CollectionName string `json:"collectionName" p:"collectionName" v:"required"`
	md.Index
}

type IndexCreateRes struct {
	Index *md.Index `json:"index"`
}

type IndexDeleteReq struct {
	g.Meta         `path:"/collections/{collectionName}/indexes" method:"delete" tags:"Meta" summary:"删除索引"`
	CollectionName string   `json:"collectionName" p:"collectionName" v:"required"`
	Fields         []string `json:"fields" v:"required"`
}

type IndexDeleteRes struct{}
