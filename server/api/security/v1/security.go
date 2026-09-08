package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	"itcodex/server/internal/service/acl"
	securitysvc "itcodex/server/internal/service/security"
)

type User struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"displayName"`
	Roles       []string `json:"roles"`
	Disabled    bool     `json:"disabled"`
}

type UserListReq struct {
	g.Meta `path:"/users" method:"get" tags:"Security" summary:"用户列表"`
}
type UserListRes struct {
	List []User `json:"list"`
}

type UserCreateReq struct {
	g.Meta      `path:"/users" method:"post" tags:"Security" summary:"创建用户"`
	Username    string `json:"username" v:"required|length:1,128"`
	DisplayName string `json:"displayName" v:"length:0,255"`
	Password    string `json:"password" v:"required|length:8,256"`
	Enabled     *bool  `json:"enabled"`
}
type UserCreateRes struct {
	User User `json:"user"`
}

type RoleListReq struct {
	g.Meta `path:"/roles" method:"get" tags:"Security" summary:"角色列表"`
}
type RoleListRes struct {
	List []securitysvc.Role `json:"list"`
}

type RoleCreateReq struct {
	g.Meta      `path:"/roles" method:"post" tags:"Security" summary:"创建角色"`
	Name        string `json:"name" v:"required|regex:^[a-zA-Z][a-zA-Z0-9_-]*$"`
	DisplayName string `json:"displayName" v:"length:0,255"`
}
type RoleCreateRes struct {
	Role *securitysvc.Role `json:"role"`
}

type RoleAssignReq struct {
	g.Meta `path:"/users/{userId}/roles/{roleId}" method:"post" tags:"Security" summary:"为用户分配角色"`
	UserID int64 `json:"userId" p:"userId" v:"required|min:1"`
	RoleID int64 `json:"roleId" p:"roleId" v:"required|min:1"`
}
type RoleAssignRes struct{}

type RoleUnassignReq struct {
	g.Meta `path:"/users/{userId}/roles/{roleId}" method:"delete" tags:"Security" summary:"移除用户角色"`
	UserID int64 `json:"userId" p:"userId" v:"required|min:1"`
	RoleID int64 `json:"roleId" p:"roleId" v:"required|min:1"`
}
type RoleUnassignRes struct{}

type PolicyListReq struct {
	g.Meta `path:"/policies" method:"get" tags:"Security" summary:"ACL 策略列表"`
}
type PolicyListRes struct {
	List []securitysvc.StoredPolicy `json:"list"`
}

type PolicyCreateReq struct {
	g.Meta `path:"/policies" method:"post" tags:"Security" summary:"创建 ACL 策略"`
	acl.Policy
}
type PolicyCreateRes struct {
	ID int64 `json:"id"`
}

type PolicyUpdateReq struct {
	g.Meta `path:"/policies/{id}" method:"put" tags:"Security" summary:"更新 ACL 策略"`
	ID     int64 `json:"id" p:"id" v:"required|min:1"`
	acl.Policy
}
type PolicyUpdateRes struct{}

type PolicyDeleteReq struct {
	g.Meta `path:"/policies/{id}" method:"delete" tags:"Security" summary:"删除 ACL 策略"`
	ID     int64 `json:"id" p:"id" v:"required|min:1"`
}
type PolicyDeleteRes struct{}
