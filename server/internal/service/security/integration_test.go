package security

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"itcodex/server/internal/service/acl"
	"itcodex/server/internal/service/auth"
	"itcodex/server/internal/service/metadata"
)

func TestRepositoryAuthorizerAppliesScopeAndFields(t *testing.T) {
	enforcer, err := acl.NewEnforcer([]acl.Policy{{
		SubjectType: acl.SubjectRole,
		Subject:     "member",
		Resource:    "posts",
		Action:      "list",
		RowFilter:   metadata.Filter{"created_by": "$currentUser.id"},
		ReadFields:  []string{"id", "title"},
		Enabled:     true,
	}})
	require.NoError(t, err)

	ctx := auth.WithIdentity(context.Background(), auth.Identity{
		UserID: "42",
		Roles:  []string{"member"},
	})
	access := &metadata.RepositoryAccess{
		Action: "list",
		Filter: metadata.Filter{"title": "hello"},
	}
	err = (RepositoryAuthorizer{Enforcer: enforcer}).AuthorizeRepository(ctx, "posts", access)
	require.NoError(t, err)
	require.True(t, access.StrictFields)
	require.ElementsMatch(t, []string{"id", "title"}, []string(access.Fields))
	require.Contains(t, access.Filter, "$and")
}

func TestRepositoryAuthorizerRejectsUnauthorizedField(t *testing.T) {
	enforcer, err := acl.NewEnforcer([]acl.Policy{{
		SubjectType: acl.SubjectRole,
		Subject:     "member",
		Resource:    "posts",
		Action:      "update",
		ReadFields:  []string{"id", "title"},
		WriteFields: []string{"title"},
		Enabled:     true,
	}})
	require.NoError(t, err)
	ctx := auth.WithIdentity(context.Background(), auth.Identity{
		UserID: "42",
		Roles:  []string{"member"},
	})
	access := &metadata.RepositoryAccess{
		Action: "update",
		Filter: metadata.Filter{"id": 1},
		Values: map[string]any{"secret": "no"},
	}
	err = (RepositoryAuthorizer{Enforcer: enforcer}).AuthorizeRepository(ctx, "posts", access)
	require.Error(t, err)
}
