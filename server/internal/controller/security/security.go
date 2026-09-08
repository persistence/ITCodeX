package security

import (
	"context"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"

	v1 "itcodex/server/api/security/v1"
	"itcodex/server/internal/service/acl"
	authsvc "itcodex/server/internal/service/auth"
	securitysvc "itcodex/server/internal/service/security"
)

type ControllerV1 struct {
	store    *securitysvc.Store
	enforcer *acl.Enforcer
}

func NewV1(store *securitysvc.Store, enforcer *acl.Enforcer) *ControllerV1 {
	return &ControllerV1{store: store, enforcer: enforcer}
}

func (c *ControllerV1) UserList(ctx context.Context, _ *v1.UserListReq) (*v1.UserListRes, error) {
	if err := c.authorize(ctx, "auth.users", "list"); err != nil {
		return nil, err
	}
	list, err := c.store.ListUsers(ctx)
	if err != nil {
		return nil, internalError(err)
	}
	items := make([]v1.User, 0, len(list))
	for _, user := range list {
		items = append(items, userToAPI(user))
	}
	return &v1.UserListRes{List: items}, nil
}

func (c *ControllerV1) UserCreate(ctx context.Context, req *v1.UserCreateReq) (*v1.UserCreateRes, error) {
	if err := c.authorize(ctx, "auth.users", "create"); err != nil {
		return nil, err
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	user, err := c.store.CreateUser(ctx, req.Username, req.DisplayName, req.Password, enabled)
	if err != nil {
		return nil, internalError(err)
	}
	return &v1.UserCreateRes{User: userToAPI(user)}, nil
}

func (c *ControllerV1) RoleList(ctx context.Context, _ *v1.RoleListReq) (*v1.RoleListRes, error) {
	if err := c.authorize(ctx, "auth.roles", "list"); err != nil {
		return nil, err
	}
	list, err := c.store.ListRoles(ctx)
	if err != nil {
		return nil, internalError(err)
	}
	return &v1.RoleListRes{List: list}, nil
}

func (c *ControllerV1) RoleCreate(ctx context.Context, req *v1.RoleCreateReq) (*v1.RoleCreateRes, error) {
	if err := c.authorize(ctx, "auth.roles", "create"); err != nil {
		return nil, err
	}
	role, err := c.store.CreateRole(ctx, req.Name, req.DisplayName)
	if err != nil {
		return nil, internalError(err)
	}
	return &v1.RoleCreateRes{Role: role}, nil
}

func (c *ControllerV1) RoleAssign(ctx context.Context, req *v1.RoleAssignReq) (*v1.RoleAssignRes, error) {
	if err := c.authorize(ctx, "auth.roles", "assign"); err != nil {
		return nil, err
	}
	if err := c.store.AssignRole(ctx, req.UserID, req.RoleID); err != nil {
		return nil, internalError(err)
	}
	return &v1.RoleAssignRes{}, nil
}

func (c *ControllerV1) RoleUnassign(ctx context.Context, req *v1.RoleUnassignReq) (*v1.RoleUnassignRes, error) {
	if err := c.authorize(ctx, "auth.roles", "assign"); err != nil {
		return nil, err
	}
	if err := c.store.UnassignRole(ctx, req.UserID, req.RoleID); err != nil {
		return nil, internalError(err)
	}
	return &v1.RoleUnassignRes{}, nil
}

func (c *ControllerV1) PolicyList(ctx context.Context, _ *v1.PolicyListReq) (*v1.PolicyListRes, error) {
	if err := c.authorize(ctx, "acl.policies", "list"); err != nil {
		return nil, err
	}
	list, err := c.store.ListPolicies(ctx)
	if err != nil {
		return nil, internalError(err)
	}
	return &v1.PolicyListRes{List: list}, nil
}

func (c *ControllerV1) PolicyCreate(ctx context.Context, req *v1.PolicyCreateReq) (*v1.PolicyCreateRes, error) {
	if err := c.authorize(ctx, "acl.policies", "create"); err != nil {
		return nil, err
	}
	if !req.Enabled {
		req.Enabled = true
	}
	if _, err := acl.NewEnforcer([]acl.Policy{req.Policy}); err != nil {
		return nil, badRequest(err)
	}
	id, err := c.store.SavePolicy(ctx, securitysvc.StoredPolicy{Policy: req.Policy})
	if err != nil {
		return nil, internalError(err)
	}
	if err := c.reloadPolicies(ctx); err != nil {
		return nil, err
	}
	return &v1.PolicyCreateRes{ID: id}, nil
}

func (c *ControllerV1) PolicyUpdate(ctx context.Context, req *v1.PolicyUpdateReq) (*v1.PolicyUpdateRes, error) {
	if err := c.authorize(ctx, "acl.policies", "update"); err != nil {
		return nil, err
	}
	if _, err := acl.NewEnforcer([]acl.Policy{req.Policy}); err != nil {
		return nil, badRequest(err)
	}
	if _, err := c.store.SavePolicy(ctx, securitysvc.StoredPolicy{ID: req.ID, Policy: req.Policy}); err != nil {
		return nil, internalError(err)
	}
	if err := c.reloadPolicies(ctx); err != nil {
		return nil, err
	}
	return &v1.PolicyUpdateRes{}, nil
}

func (c *ControllerV1) PolicyDelete(ctx context.Context, req *v1.PolicyDeleteReq) (*v1.PolicyDeleteRes, error) {
	if err := c.authorize(ctx, "acl.policies", "destroy"); err != nil {
		return nil, err
	}
	if err := c.store.DeletePolicy(ctx, req.ID); err != nil {
		return nil, internalError(err)
	}
	if err := c.reloadPolicies(ctx); err != nil {
		return nil, err
	}
	return &v1.PolicyDeleteRes{}, nil
}

func (c *ControllerV1) authorize(ctx context.Context, resource, action string) error {
	identity := securitysvc.PrincipalFromContext(ctx)
	if _, err := c.enforcer.Authorize(ctx, identity, resource, action); err != nil {
		return gerror.NewCode(gcode.New(403, "禁止访问", nil), err.Error())
	}
	return nil
}

func (c *ControllerV1) reloadPolicies(ctx context.Context) error {
	policies, err := c.store.LoadPolicies(ctx)
	if err != nil {
		return internalError(err)
	}
	if err := c.enforcer.ReplacePolicies(policies); err != nil {
		return internalError(err)
	}
	return nil
}

func internalError(err error) error {
	return gerror.NewCode(gcode.CodeInternalError, err.Error())
}

func badRequest(err error) error {
	return gerror.NewCode(gcode.New(400, "请求参数无效", nil), err.Error())
}

func userToAPI(user *authsvc.User) v1.User {
	if user == nil {
		return v1.User{}
	}
	return v1.User{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Roles:       append([]string(nil), user.Roles...),
		Disabled:    user.Disabled,
	}
}
