package security

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"itcodex/server/internal/service/acl"
	"itcodex/server/internal/service/auth"
	"itcodex/server/internal/service/metadata"
	"itcodex/server/internal/service/resource"
)

type Runtime struct {
	Store     *Store
	Auth      *auth.Service
	ACL       *acl.Enforcer
	Resources *resource.Manager
}

func Bootstrap(ctx context.Context, db *metadata.Database) (*Runtime, error) {
	store, err := NewStore(db)
	if err != nil {
		return nil, err
	}
	bootstrapPassword := strings.TrimSpace(os.Getenv("ITCODEX_BOOTSTRAP_ADMIN_PASSWORD"))
	bootstrapUser := strings.TrimSpace(os.Getenv("ITCODEX_BOOTSTRAP_ADMIN_USERNAME"))
	if bootstrapUser == "" {
		bootstrapUser = "admin"
	}
	if err := store.EnsureBootstrapAdmin(ctx, bootstrapUser, bootstrapPassword); err != nil {
		return nil, fmt.Errorf("security: bootstrap administrator: %w", err)
	}

	policies, err := store.LoadPolicies(ctx)
	if err != nil {
		return nil, fmt.Errorf("security: load ACL policies: %w", err)
	}
	enforcer, err := acl.NewEnforcer(policies)
	if err != nil {
		return nil, fmt.Errorf("security: initialize ACL: %w", err)
	}

	secret := strings.TrimSpace(os.Getenv("ITCODEX_JWT_SECRET"))
	if secret == "" {
		secret = strings.TrimSpace(g.Cfg().MustGet(ctx, "auth.jwtSecret").String())
	}
	authService, err := auth.NewService(store, auth.Config{
		Secret:     []byte(secret),
		Issuer:     configString(ctx, "auth.issuer", "itcodex"),
		AccessTTL:  configDuration(ctx, "auth.accessTTL", 15*time.Minute),
		RefreshTTL: configDuration(ctx, "auth.refreshTTL", 7*24*time.Hour),
	})
	if err != nil {
		return nil, err
	}

	resources := resource.NewManager()
	resources.Use(ResourceMiddleware(enforcer))
	resources.RegisterMetadataCRUD(resource.DatabaseResolver{Database: db})
	resources.Register("*", resource.ActionUpload, func(*resource.ActionContext) (any, error) { return nil, nil })
	resources.Register("*", resource.ActionFileGet, func(*resource.ActionContext) (any, error) { return nil, nil })
	resources.Register("*", "*", func(actionContext *resource.ActionContext) (any, error) {
		run, ok := actionContext.Request.Data.(func(context.Context) error)
		if !ok {
			return nil, fmt.Errorf("security: invalid custom action handler")
		}
		return nil, run(actionContext.Context)
	})

	return &Runtime{
		Store:     store,
		Auth:      authService,
		ACL:       enforcer,
		Resources: resources,
	}, nil
}

func configString(ctx context.Context, key, fallback string) string {
	value := strings.TrimSpace(g.Cfg().MustGet(ctx, key).String())
	if value == "" {
		return fallback
	}
	return value
}

func configDuration(ctx context.Context, key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(g.Cfg().MustGet(ctx, key).String())
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return duration
}
