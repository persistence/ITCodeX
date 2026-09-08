package metadata

import "context"

type accessContextKey string

const repositoryAuthorizerKey accessContextKey = "metadata.repositoryAuthorizer"
const repositoryActionKey accessContextKey = "metadata.repositoryAction"

type RepositoryAccess struct {
	Action       string
	Filter       Filter
	Fields       Fields
	Except       Fields
	Sort         Sort
	Appends      Appends
	Values       map[string]any
	StrictFields bool
}

type RepositoryAuthorizer interface {
	AuthorizeRepository(ctx context.Context, collection string, access *RepositoryAccess) error
}

func WithRepositoryAuthorizer(ctx context.Context, authorizer RepositoryAuthorizer) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, repositoryAuthorizerKey, authorizer)
}

func repositoryAuthorizerFromContext(ctx context.Context) RepositoryAuthorizer {
	if ctx == nil {
		return nil
	}
	authorizer, _ := ctx.Value(repositoryAuthorizerKey).(RepositoryAuthorizer)
	return authorizer
}

func WithRepositoryAction(ctx context.Context, action string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, repositoryActionKey, action)
}

func repositoryActionFromContext(ctx context.Context, fallback string) string {
	if ctx != nil {
		if action, ok := ctx.Value(repositoryActionKey).(string); ok && action != "" {
			return action
		}
	}
	return fallback
}

func authorizeRepository(ctx context.Context, collection string, access *RepositoryAccess) error {
	authorizer := repositoryAuthorizerFromContext(ctx)
	if authorizer == nil {
		return nil
	}
	return authorizer.AuthorizeRepository(ctx, collection, access)
}
