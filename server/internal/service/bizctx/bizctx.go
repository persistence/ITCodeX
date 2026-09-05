package bizctx

import (
	"context"
	"strconv"

	"github.com/gogf/gf/v2/net/ghttp"

	md "itcodex/server/internal/service/metadata"
)

func Ctx(r *ghttp.Request) {
	if actor := r.Header.Get("X-Actor-Id"); actor != "" {
		if id, err := strconv.ParseInt(actor, 10, 64); err == nil && id > 0 {
			r.SetCtx(context.WithValue(r.Context(), md.CtxActorID, id))
		}
	}
	r.Middleware.Next()
}
