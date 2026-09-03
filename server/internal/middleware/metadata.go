package middleware

import (
	"net/http"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	md "itcodex/server/internal/logic/metadata"
	yaegictx "itcodex/server/pkg/yaegi/context"
)

func MetadataContext(db *md.Database) func(r *ghttp.Request) {
	return func(r *ghttp.Request) {
		r.SetCtxVar("metadataDB", db)
		r.Middleware.Next()
	}
}

func CustomAPIRouter(db *md.Database) func(r *ghttp.Request) {
	return func(r *ghttp.Request) {
		action := r.Get("action").String()
		method := r.Method
		fullPath := r.URL.Path

		var apiPath string
		if strings.HasPrefix(fullPath, "/api/custom/global/") {
			apiPath = strings.TrimPrefix(fullPath, "/api/custom/global")
		} else if strings.HasPrefix(fullPath, "/api/custom/c/") {
			parts := strings.SplitN(strings.TrimPrefix(fullPath, "/api/custom/c/"), "/", 2)
			if len(parts) == 2 {
				collName := parts[0]
				apiPath = "/c/" + collName + "/" + parts[1]
			} else if len(parts) == 1 {
				apiPath = "/c/" + parts[0]
			}
		} else {
			apiPath = action
		}

		if !strings.HasPrefix(apiPath, "/") {
			apiPath = "/" + apiPath
		}

		if db.Yaegi() == nil {
			r.Middleware.Next()
			return
		}

		script := db.Yaegi().FindCustomAPI(method, apiPath)
		if script == nil {
			r.Middleware.Next()
			return
		}

		params := make(map[string]string)

		yctx := yaegictx.NewYaegiHTTPContext(r.Response.Writer, r.Request, params)

		if err := db.Yaegi().ExecuteCustomAPI(script, yctx); err != nil {
			if yctx.Response.Status == 0 {
				yctx.Response.JSON(http.StatusInternalServerError, g.Map{
					"code":    1,
					"message": err.Error(),
				})
			}
			return
		}

		if yctx.Response.Status == 0 {
			r.Response.WriteHeader(http.StatusOK)
		}
	}
}
