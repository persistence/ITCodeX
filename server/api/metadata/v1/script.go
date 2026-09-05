package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	modelmd "itcodex/server/internal/model/metadata"
)

type ScriptListReq struct {
	g.Meta     `path:"/scripts" method:"get" tags:"Meta" summary:"列出 Yaegi 脚本"`
	Collection string `json:"collection" p:"collection"`
	Hook       string `json:"hook" p:"hook"`
}
type ScriptListRes struct {
	List []*modelmd.YaegiScript `json:"list"`
}

type ScriptSaveReq struct {
	g.Meta `path:"/scripts" method:"post" tags:"Meta" summary:"创建或更新 Yaegi 脚本"`
	modelmd.YaegiScript
}
type ScriptSaveRes struct {
	*modelmd.YaegiScript
}

type ScriptDisableReq struct {
	g.Meta `path:"/scripts/{id}/disable" method:"post" tags:"Meta" summary:"禁用 Yaegi 脚本"`
	Id     int64 `json:"id" p:"id" v:"required"`
}
type ScriptDisableRes struct{}
