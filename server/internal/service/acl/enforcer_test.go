package acl

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itcodex/server/internal/service/metadata"
)

func mustEnforcer(t *testing.T, policies []Policy, opts ...Option) *Enforcer {
	t.Helper()
	enforcer, err := NewEnforcer(policies, opts...)
	require.NoError(t, err)
	return enforcer
}

func TestDefaultDenyAndEnabledPolicies(t *testing.T) {
	ctx := context.Background()
	identity := Principal{ID: "u1", Roles: []string{"editor"}, LoggedIn: true}
	enforcer := mustEnforcer(t, []Policy{
		{SubjectType: SubjectRole, Subject: "editor", Resource: "posts", Action: "read", Enabled: false},
	})

	assert.False(t, enforcer.Can(ctx, identity, "posts", "read"))
	decision, err := enforcer.Authorize(ctx, identity, "posts", "read")
	assert.False(t, decision.Allowed)
	var forbidden *ForbiddenError
	require.ErrorAs(t, err, &forbidden)
	assert.Equal(t, 403, forbidden.Code())
}

func TestSubjectsAndResourceActionWildcards(t *testing.T) {
	ctx := context.Background()
	enforcer := mustEnforcer(t, []Policy{
		{SubjectType: SubjectPublic, Resource: "public/*", Action: "read", Enabled: true},
		{SubjectType: SubjectLoggedIn, Resource: "profile", Action: "*", Enabled: true},
		{SubjectType: SubjectRole, Subject: "admin-*", Resource: "*", Action: "*", Enabled: true},
	})

	assert.True(t, enforcer.Can(ctx, nil, "public/news/today", "read"))
	assert.False(t, enforcer.Can(ctx, nil, "profile", "read"))
	assert.True(t, enforcer.Can(ctx, Principal{ID: 7, LoggedIn: true}, "profile", "update"))
	assert.True(t, enforcer.Can(ctx, Principal{Roles: []string{"admin-global"}}, "secrets", "delete"))
	assert.False(t, enforcer.Can(ctx, Principal{Roles: []string{"viewer"}}, "secrets", "delete"))
}

func TestRoleFiltersORCallerANDFieldsUnionAndVariables(t *testing.T) {
	identity := Principal{ID: "user-7", Roles: []string{"editor", "owner"}, LoggedIn: true}
	enforcer := mustEnforcer(t, []Policy{
		{
			SubjectType: SubjectRole,
			Subject:     "editor",
			Resource:    "posts",
			Action:      "read",
			RowFilter:   metadata.Filter{"status": "published"},
			ReadFields:  []string{"id", "title"},
			WriteFields: []string{"title"},
			Enabled:     true,
		},
		{
			SubjectType: SubjectRole,
			Subject:     "owner",
			Resource:    "posts",
			Action:      "read",
			RowFilter:   metadata.Filter{"ownerId": "$currentUser.id", "ownerRole": metadata.Filter{"$in": "$currentUser.roles"}},
			ReadFields:  []string{"title", "internalNotes"},
			WriteFields: []string{"body", "title"},
			Enabled:     true,
		},
	})

	caller := metadata.Filter{"deletedAt": metadata.Filter{"$isNull": true}}
	decision, err := enforcer.Authorize(context.Background(), identity, "posts", "read", caller)
	require.NoError(t, err)
	assert.Equal(t, []string{"id", "title", "internalNotes"}, decision.ReadFields)
	assert.Equal(t, []string{"title", "body"}, decision.WriteFields)

	andParts := decision.Filter["$and"].([]metadata.Filter)
	require.Len(t, andParts, 2)
	orParts := andParts[0]["$or"].([]metadata.Filter)
	require.Len(t, orParts, 2)
	assert.Equal(t, "user-7", orParts[1]["ownerId"])
	assert.Equal(t, []any{"editor", "owner"}, orParts[1]["ownerRole"].(metadata.Filter)["$in"])
	assert.Equal(t, caller, andParts[1])

	var params []any
	where, err := metadata.BuildWhereClause(decision.Filter, &params)
	require.NoError(t, err)
	assert.Contains(t, where, " OR ")
	assert.Contains(t, where, " AND ")
	assert.Len(t, params, 4)

	// Inputs and snapshots are not aliased.
	orParts[0]["status"] = "changed"
	next, err := enforcer.Authorize(context.Background(), identity, "posts", "read", caller)
	require.NoError(t, err)
	nextOR := next.Filter["$and"].([]metadata.Filter)[0]["$or"].([]metadata.Filter)
	assert.Equal(t, "published", nextOR[0]["status"])
}

func TestUnrestrictedGrantDominatesPolicyRowFilters(t *testing.T) {
	enforcer := mustEnforcer(t, []Policy{
		{SubjectType: SubjectRole, Subject: "reader", Resource: "posts", Action: "read", RowFilter: metadata.Filter{"state": "live"}, Enabled: true},
		{SubjectType: SubjectRole, Subject: "reader", Resource: "posts", Action: "read", Enabled: true},
	})
	caller := metadata.Filter{"tenantId": 9}

	decision, err := enforcer.Authorize(context.Background(), Principal{Roles: []string{"reader"}}, "posts", "read", caller)
	require.NoError(t, err)
	assert.Equal(t, caller, decision.Filter)
}

func TestFixedParamsAreAppliedLast(t *testing.T) {
	var called bool
	enforcer := mustEnforcer(t, []Policy{{
		SubjectType: SubjectPublic,
		Resource:    "posts",
		Action:      "read",
		RowFilter:   metadata.Filter{"visible": true},
		Enabled:     true,
		AddFixedParams: func(_ context.Context, identity Identity) (metadata.Filter, error) {
			called = true
			assert.Nil(t, identity)
			return metadata.Filter{"tenantId": 42}, nil
		},
	}})

	decision, err := enforcer.Authorize(context.Background(), nil, "posts", "read", metadata.Filter{"language": "zh"})
	require.NoError(t, err)
	assert.True(t, called)

	outer := decision.Filter["$and"].([]metadata.Filter)
	require.Len(t, outer, 2)
	assert.Equal(t, metadata.Filter{"tenantId": 42}, outer[1])
	inner := outer[0]["$and"].([]metadata.Filter)
	assert.Equal(t, metadata.Filter{"visible": true}, inner[0])
	assert.Equal(t, metadata.Filter{"language": "zh"}, inner[1])
}

func TestVariableValidation(t *testing.T) {
	_, err := NewEnforcer([]Policy{{
		SubjectType: SubjectPublic,
		Resource:    "posts",
		Action:      "read",
		RowFilter:   metadata.Filter{"owner": "$currentUser.email"},
		Enabled:     true,
	}})
	assert.ErrorContains(t, err, `unsupported ACL variable "$currentUser.email"`)

	valid := []Policy{{
		SubjectType: SubjectPublic,
		Resource:    "posts",
		Action:      "read",
		RowFilter: metadata.Filter{"$or": []metadata.Filter{
			{"owner": "$currentUser.id"},
			{"role": metadata.Filter{"$in": "$currentUser.roles"}},
		}},
		Enabled: true,
	}}
	enforcer, err := NewEnforcer(valid)
	require.NoError(t, err)
	err = enforcer.ReplacePolicies([]Policy{{
		SubjectType: SubjectPublic,
		Resource:    "posts",
		Action:      "read",
		RowFilter:   metadata.Filter{"owner": "$unknown"},
		Enabled:     true,
	}})
	require.Error(t, err)
	assert.True(t, enforcer.Can(context.Background(), nil, "posts", "read"))
}

func TestFixedParamsErrorDeniesCanAndPropagatesAuthorize(t *testing.T) {
	sentinel := errors.New("tenant unavailable")
	enforcer := mustEnforcer(t, []Policy{{
		SubjectType: SubjectPublic,
		Resource:    "*",
		Action:      "*",
		Enabled:     true,
		AddFixedParams: func(context.Context, Identity) (metadata.Filter, error) {
			return nil, sentinel
		},
	}})

	assert.False(t, enforcer.Can(context.Background(), nil, "posts", "read"))
	_, err := enforcer.Authorize(context.Background(), nil, "posts", "read")
	assert.ErrorIs(t, err, sentinel)
}

func TestContextEnforcerSystemBypassAndCustomForbidden(t *testing.T) {
	custom := errors.New("custom forbidden")
	enforcer := mustEnforcer(t, nil, WithForbiddenFactory(func(resource, action string) error {
		assert.Equal(t, "posts", resource)
		assert.Equal(t, "delete", action)
		return custom
	}))

	ctx := WithEnforcer(context.Background(), enforcer)
	fromContext, ok := EnforcerFromContext(ctx)
	assert.True(t, ok)
	assert.Same(t, enforcer, fromContext)

	_, err := enforcer.Authorize(ctx, nil, "posts", "delete")
	assert.ErrorIs(t, err, custom)

	bypass := WithSystemBypass(ctx)
	caller := metadata.Filter{"tenantId": 3}
	decision, err := enforcer.Authorize(bypass, nil, "posts", "delete", caller)
	require.NoError(t, err)
	assert.True(t, decision.Allowed)
	assert.Equal(t, caller, decision.Filter)
}

func TestReplacePoliciesUsesConcurrentImmutableSnapshots(t *testing.T) {
	allowed := []Policy{{SubjectType: SubjectPublic, Resource: "*", Action: "read", Enabled: true}}
	denied := []Policy{{SubjectType: SubjectPublic, Resource: "*", Action: "read", Enabled: false}}
	enforcer := mustEnforcer(t, denied)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				_ = enforcer.Can(context.Background(), nil, "posts", "read")
			}
		}()
	}
	for i := 0; i < 100; i++ {
		require.NoError(t, enforcer.ReplacePolicies(allowed))
		require.NoError(t, enforcer.ReplacePolicies(denied))
	}
	wg.Wait()

	require.NoError(t, enforcer.ReplacePolicies(allowed))
	allowed[0].Enabled = false
	assert.True(t, enforcer.Can(context.Background(), nil, "posts", "read"))
}

func TestAddFixedParamsAlwaysAndsConstraint(t *testing.T) {
	enforcer := mustEnforcer(t, []Policy{{
		SubjectType: SubjectPublic,
		Resource:    "posts",
		Action:      "list",
		Enabled:     true,
	}})
	enforcer.AddFixedParams("posts", "list", func(context.Context, Identity) (metadata.Filter, error) {
		return metadata.Filter{"deleted": false}, nil
	})
	decision, err := enforcer.Authorize(
		context.Background(),
		nil,
		"posts",
		"list",
		metadata.Filter{"status": "published"},
	)
	require.NoError(t, err)
	assert.Contains(t, decision.Filter, "$and")
}

func TestFieldRestrictionTracksNilVersusEmpty(t *testing.T) {
	enforcer := mustEnforcer(t, []Policy{
		{
			SubjectType: SubjectRole,
			Subject:     "restricted",
			Resource:    "posts",
			Action:      "list",
			ReadFields:  []string{},
			Enabled:     true,
		},
		{
			SubjectType: SubjectRole,
			Subject:     "unrestricted",
			Resource:    "posts",
			Action:      "list",
			ReadFields:  nil,
			Enabled:     true,
		},
	})
	restricted, err := enforcer.Authorize(context.Background(), Principal{
		ID: 1, Roles: []string{"restricted"}, LoggedIn: true,
	}, "posts", "list")
	require.NoError(t, err)
	assert.True(t, restricted.ReadRestricted)

	unrestricted, err := enforcer.Authorize(context.Background(), Principal{
		ID: 2, Roles: []string{"restricted", "unrestricted"}, LoggedIn: true,
	}, "posts", "list")
	require.NoError(t, err)
	assert.False(t, unrestricted.ReadRestricted)
}
