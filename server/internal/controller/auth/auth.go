package auth

import (
	"context"
	"errors"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"

	v1 "itcodex/server/api/auth/v1"
	authsvc "itcodex/server/internal/service/auth"
)

type ControllerV1 struct {
	service *authsvc.Service
}

func NewV1(service *authsvc.Service) *ControllerV1 {
	return &ControllerV1{service: service}
}

func (c *ControllerV1) Login(
	ctx context.Context,
	req *v1.LoginReq,
) (res *v1.LoginRes, err error) {
	pair, err := c.service.Login(ctx, req.Username, req.Password)
	if err != nil {
		return nil, wrapServiceError(err)
	}
	return &v1.LoginRes{TokenPair: tokenPairToAPI(pair)}, nil
}

func (c *ControllerV1) Refresh(
	ctx context.Context,
	req *v1.RefreshReq,
) (res *v1.RefreshRes, err error) {
	pair, err := c.service.Refresh(ctx, req.RefreshToken)
	if err != nil {
		return nil, wrapServiceError(err)
	}
	return &v1.RefreshRes{TokenPair: tokenPairToAPI(pair)}, nil
}

func (c *ControllerV1) Logout(
	ctx context.Context,
	req *v1.LogoutReq,
) (res *v1.LogoutRes, err error) {
	if err = c.service.Logout(ctx, req.RefreshToken); err != nil {
		return nil, wrapServiceError(err)
	}
	return &v1.LogoutRes{}, nil
}

func (c *ControllerV1) Me(
	ctx context.Context,
	_ *v1.MeReq,
) (res *v1.MeRes, err error) {
	user, err := c.service.Me(ctx)
	if err != nil {
		return nil, wrapServiceError(err)
	}
	return &v1.MeRes{User: v1.UserInfo{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Roles:       append([]string(nil), user.Roles...),
	}}, nil
}

func tokenPairToAPI(pair *authsvc.TokenPair) v1.TokenPair {
	return v1.TokenPair{
		AccessToken:      pair.AccessToken,
		RefreshToken:     pair.RefreshToken,
		TokenType:        pair.TokenType,
		ExpiresIn:        pair.ExpiresIn,
		RefreshExpiresIn: pair.RefreshExpiresIn,
	}
}

func wrapServiceError(err error) error {
	switch {
	case errors.Is(err, authsvc.ErrInvalidCredentials),
		errors.Is(err, authsvc.ErrUnauthenticated),
		errors.Is(err, authsvc.ErrInvalidToken),
		errors.Is(err, authsvc.ErrTokenRevoked):
		return gerror.NewCode(gcode.New(401, "未认证或令牌无效", nil), "未认证或令牌无效")
	case errors.Is(err, authsvc.ErrUserDisabled):
		return gerror.NewCode(gcode.New(403, "用户已禁用", nil), "用户已禁用")
	default:
		return gerror.NewCode(gcode.CodeInternalError, err.Error())
	}
}
