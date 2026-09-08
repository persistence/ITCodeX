package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	modelmd "itcodex/server/internal/model/metadata"
	authsvc "itcodex/server/internal/service/auth"
	md "itcodex/server/internal/service/metadata"
	resourcemgr "itcodex/server/internal/service/resource"
	yaegictx "itcodex/server/pkg/yaegi/context"
)

func MetadataContext(db *md.Database) func(r *ghttp.Request) {
	return func(r *ghttp.Request) {
		r.SetCtxVar("metadataDB", db)
		r.Middleware.Next()
	}
}

func CustomAPIRouter(db *md.Database, managers ...*resourcemgr.Manager) func(r *ghttp.Request) {
	var manager *resourcemgr.Manager
	if len(managers) > 0 {
		manager = managers[0]
	}
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
		// Simple path param: last non-empty segment as id when path has trailing id-like segment
		segs := strings.Split(strings.Trim(apiPath, "/"), "/")
		if len(segs) > 0 {
			last := segs[len(segs)-1]
			if last != "" && last != "action" {
				params["id"] = last
			}
		}

		execute := func(ctx context.Context) error {
			request := r.Request.WithContext(ctx)
			yctx := yaegictx.NewYaegiHTTPContext(r.Response.Writer, request, params)
			if err := db.Yaegi().ExecuteCustomAPI(script, yctx); err != nil {
				if yctx.Response.Status == 0 {
					yctx.Response.JSON(http.StatusInternalServerError, g.Map{
						"code":    1,
						"message": err.Error(),
					})
				}
				return err
			}
			if yctx.Response.Status == 0 {
				r.Response.WriteHeader(http.StatusOK)
			}
			return nil
		}

		var err error
		if manager != nil {
			resourceName, actionName, configured := customActionIdentity(script)
			if !configured && !isAdminRequest(r.Context()) {
				r.Response.WriteHeader(http.StatusForbidden)
				r.Response.WriteJson(g.Map{
					"code":    http.StatusForbidden,
					"message": "旧版自定义 API 未声明 resourceName/actionName，仅管理员可访问",
					"data":    nil,
				})
				return
			}
			_, err = manager.Dispatch(r.Context(), &resourcemgr.ActionRequest{
				Resource: resourceName,
				Action:   actionName,
				Data:     execute,
			})
		} else {
			err = execute(r.Context())
		}
		if err != nil && r.Response.BufferLength() == 0 {
			status := http.StatusInternalServerError
			if _, ok := err.(*md.ForbiddenError); ok {
				status = http.StatusForbidden
			}
			r.Response.WriteHeader(status)
			r.Response.WriteJson(g.Map{
				"code":    status,
				"message": err.Error(),
				"data":    nil,
			})
		}
	}
}

func customActionIdentity(script *modelmd.YaegiScript) (resourceName, actionName string, configured bool) {
	if script == nil {
		return "global", "custom:unknown", false
	}
	var options struct {
		ResourceName string `json:"resourceName"`
		ActionName   string `json:"actionName"`
	}
	if script.Options != "" {
		_ = json.Unmarshal([]byte(script.Options), &options)
	}
	if options.ResourceName != "" && options.ActionName != "" {
		return options.ResourceName, options.ActionName, true
	}
	resourceName = script.CollectionName
	if resourceName == "" {
		resourceName = "global"
	}
	return resourceName, "custom:" + script.Name, false
}

func isAdminRequest(ctx context.Context) bool {
	identity, ok := authsvc.IdentityFromContext(ctx)
	if !ok {
		return false
	}
	for _, role := range identity.Roles {
		if role == "admin" {
			return true
		}
	}
	return false
}
