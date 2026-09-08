package v1

import "github.com/gogf/gf/v2/frame/g"

type LoginReq struct {
	g.Meta   `path:"/login" method:"post" tags:"Auth" summary:"登录"`
	Username string `json:"username" v:"required|length:1,128"`
	Password string `json:"password" v:"required|length:8,256"`
}

type LoginRes struct {
	TokenPair
}

type RefreshReq struct {
	g.Meta       `path:"/refresh" method:"post" tags:"Auth" summary:"轮换访问令牌"`
	RefreshToken string `json:"refreshToken" v:"required"`
}

type RefreshRes struct {
	TokenPair
}

type LogoutReq struct {
	g.Meta       `path:"/logout" method:"post" tags:"Auth" summary:"退出登录"`
	RefreshToken string `json:"refreshToken" v:"required"`
}

type LogoutRes struct{}

type MeReq struct {
	g.Meta `path:"/me" method:"get" tags:"Auth" summary:"获取当前用户"`
}

type MeRes struct {
	User UserInfo `json:"user"`
}

type TokenPair struct {
	AccessToken      string `json:"accessToken"`
	RefreshToken     string `json:"refreshToken"`
	TokenType        string `json:"tokenType"`
	ExpiresIn        int64  `json:"expiresIn"`
	RefreshExpiresIn int64  `json:"refreshExpiresIn"`
}

type UserInfo struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"displayName"`
	Roles       []string `json:"roles"`
}
