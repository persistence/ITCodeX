package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"

	"itcodex/server/internal/service/auth"
	"itcodex/server/internal/service/metadata"
)

func OptionalAuth(service *auth.Service) func(*ghttp.Request) {
	return func(r *ghttp.Request) {
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if header == "" {
			r.Middleware.Next()
			return
		}
		token, err := auth.BearerToken(header)
		if err != nil {
			writeUnauthorized(r)
			return
		}
		ctx, err := service.Authenticate(r.Context(), token)
		if err != nil {
			writeUnauthorized(r)
			return
		}
		if identity, ok := auth.IdentityFromContext(ctx); ok {
			if actorID, parseErr := strconv.ParseInt(identity.UserID, 10, 64); parseErr == nil && actorID > 0 {
				ctx = context.WithValue(ctx, metadata.CtxActorID, actorID)
			}
		}
		r.SetCtx(ctx)
		r.Middleware.Next()
	}
}

func writeUnauthorized(r *ghttp.Request) {
	r.Response.WriteHeader(http.StatusUnauthorized)
	r.Response.WriteJson(map[string]any{
		"code":    http.StatusUnauthorized,
		"message": "未认证或令牌无效",
		"data":    nil,
	})
}
