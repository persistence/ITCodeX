package auth

import (
	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/net/ghttp"
)

// HTTPMiddleware authenticates a Bearer access token and installs Identity in
// the request context. Apply it only to protected routes.
func (s *Service) HTTPMiddleware(r *ghttp.Request) {
	token, err := BearerToken(r.Header.Get("Authorization"))
	if err != nil {
		r.SetError(unauthorizedError())
		return
	}
	ctx, err := s.Authenticate(r.Context(), token)
	if err != nil {
		r.SetError(unauthorizedError())
		return
	}
	r.SetCtx(ctx)
	r.Middleware.Next()
}

func unauthorizedError() error {
	return gerror.NewCode(gcode.New(401, "未认证或令牌无效", nil), "未认证或令牌无效")
}
