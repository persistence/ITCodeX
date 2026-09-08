package security

import (
	"context"
	"strings"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/net/ghttp"

	"itcodex/server/internal/service/acl"
	"itcodex/server/internal/service/auth"
	"itcodex/server/internal/service/metadata"
	"itcodex/server/internal/service/resource"
)

type principal struct {
	id    string
	roles []string
}

func (p principal) IdentityID() any         { return p.id }
func (p principal) IdentityRoles() []string { return append([]string(nil), p.roles...) }
func (p principal) IsLoggedIn() bool        { return p.id != "" }

func PrincipalFromContext(ctx context.Context) acl.Identity {
	identity, ok := auth.IdentityFromContext(ctx)
	if !ok {
		return nil
	}
	return principal{id: identity.UserID, roles: identity.Roles}
}

type RepositoryAuthorizer struct {
	Enforcer *acl.Enforcer
}

func (a RepositoryAuthorizer) AuthorizeRepository(ctx context.Context, collection string, access *metadata.RepositoryAccess) error {
	if a.Enforcer == nil {
		return metadata.NewForbiddenError("权限服务不可用")
	}
	callerFilter := access.Filter
	decision, err := a.Enforcer.Authorize(
		ctx,
		PrincipalFromContext(ctx),
		collection,
		access.Action,
		access.Filter,
	)
	if err != nil {
		return metadata.NewForbiddenError(err.Error())
	}

	access.Filter = callerFilter
	if err := validateReadAccess(access, decision.ReadFields, decision.ReadRestricted); err != nil {
		return err
	}
	if err := validateWriteAccess(access.Values, decision.WriteFields, decision.WriteRestricted); err != nil {
		return err
	}
	access.Filter = decision.Filter
	return nil
}

func ResourceMiddleware(enforcer *acl.Enforcer) resource.Middleware {
	authorizer := RepositoryAuthorizer{Enforcer: enforcer}
	return func(next resource.ActionHandler) resource.ActionHandler {
		return func(actionContext *resource.ActionContext) (any, error) {
			_, err := enforcer.Authorize(
				actionContext.Context,
				PrincipalFromContext(actionContext.Context),
				actionContext.Request.Resource,
				actionContext.Request.Action,
			)
			if err != nil {
				return nil, metadata.NewForbiddenError(err.Error())
			}
			actionContext.Context = acl.WithEnforcer(actionContext.Context, enforcer)
			actionContext.Context = metadata.WithRepositoryAuthorizer(actionContext.Context, authorizer)
			actionContext.Context = metadata.WithRepositoryAction(actionContext.Context, actionContext.Request.Action)
			return next(actionContext)
		}
	}
}

func WithRepositoryAuthorization(ctx context.Context, enforcer *acl.Enforcer) context.Context {
	ctx = acl.WithEnforcer(ctx, enforcer)
	return metadata.WithRepositoryAuthorizer(ctx, RepositoryAuthorizer{Enforcer: enforcer})
}

func MetaHTTPMiddleware(enforcer *acl.Enforcer) func(*ghttp.Request) {
	return func(r *ghttp.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/meta/security/") || r.URL.Path == "/api/meta/security" {
			r.Middleware.Next()
			return
		}
		resourceName := "meta.collections"
		switch {
		case strings.Contains(r.URL.Path, "/fields"):
			resourceName = "meta.fields"
		case strings.Contains(r.URL.Path, "/indexes"):
			resourceName = "meta.indexes"
		case strings.Contains(r.URL.Path, "/scripts"):
			resourceName = "meta.scripts"
		}
		action := strings.ToLower(r.Method)
		switch action {
		case "get":
			action = "list"
		case "post":
			action = "create"
		case "put", "patch":
			action = "update"
		case "delete":
			action = "destroy"
		}
		if _, err := enforcer.Authorize(r.Context(), PrincipalFromContext(r.Context()), resourceName, action); err != nil {
			r.SetError(gerror.NewCode(gcode.New(403, "禁止访问", nil), err.Error()))
			return
		}
		r.SetCtx(WithRepositoryAuthorization(r.Context(), enforcer))
		r.Middleware.Next()
	}
}

func validateReadAccess(access *metadata.RepositoryAccess, allowed []string, restricted bool) error {
	if !restricted {
		return nil
	}
	if len(allowed) == 0 {
		return metadata.NewForbiddenError("该操作未授权任何可读字段")
	}
	allowedSet := fieldSet(allowed)
	if _, ok := allowedSet["*"]; ok {
		return nil
	}
	for _, field := range access.Fields {
		if !fieldAllowed(field, allowedSet) {
			return metadata.NewForbiddenError("无权读取字段: " + field)
		}
	}
	for _, field := range access.Sort {
		field = strings.TrimPrefix(field, "-")
		if !fieldAllowed(field, allowedSet) {
			return metadata.NewForbiddenError("无权按字段排序: " + field)
		}
	}
	for _, field := range access.Appends {
		if !fieldAllowed(field, allowedSet) {
			return metadata.NewForbiddenError("无权读取关联字段: " + field)
		}
	}
	if err := validateFilterFields(access.Filter, allowedSet); err != nil {
		return err
	}
	if len(access.Fields) == 0 {
		access.Fields = append(metadata.Fields(nil), allowed...)
	}
	access.StrictFields = true
	return nil
}

func validateWriteAccess(values map[string]any, allowed []string, restricted bool) error {
	if len(values) == 0 || !restricted {
		return nil
	}
	allowedSet := fieldSet(allowed)
	if _, ok := allowedSet["*"]; ok {
		return nil
	}
	for field := range values {
		if !fieldAllowed(field, allowedSet) {
			return metadata.NewForbiddenError("无权写入字段: " + field)
		}
	}
	return nil
}

func validateFilterFields(filter metadata.Filter, allowed map[string]struct{}) error {
	for key, value := range filter {
		if !strings.HasPrefix(key, "$") && !fieldAllowed(key, allowed) {
			return metadata.NewForbiddenError("无权按字段过滤: " + key)
		}
		switch nested := value.(type) {
		case metadata.Filter:
			if strings.HasPrefix(key, "$") {
				if err := validateFilterFields(nested, allowed); err != nil {
					return err
				}
			}
		case map[string]any:
			if strings.HasPrefix(key, "$") {
				if err := validateFilterFields(metadata.Filter(nested), allowed); err != nil {
					return err
				}
			}
		case []metadata.Filter:
			for _, item := range nested {
				if err := validateFilterFields(item, allowed); err != nil {
					return err
				}
			}
		case []any:
			for _, item := range nested {
				if itemFilter, ok := item.(map[string]any); ok {
					if err := validateFilterFields(metadata.Filter(itemFilter), allowed); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func fieldSet(fields []string) map[string]struct{} {
	result := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		result[field] = struct{}{}
	}
	return result
}

func fieldAllowed(field string, allowed map[string]struct{}) bool {
	if _, ok := allowed["*"]; ok {
		return true
	}
	if _, ok := allowed[field]; ok {
		return true
	}
	root, _, _ := strings.Cut(field, ".")
	_, ok := allowed[root]
	return ok
}

var _ metadata.RepositoryAuthorizer = RepositoryAuthorizer{}
