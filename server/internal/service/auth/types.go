package auth

import (
	"context"
	"errors"
	"time"
)

var (
	ErrUserNotFound       = errors.New("auth: user not found")
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
	ErrUnauthenticated    = errors.New("auth: unauthenticated")
	ErrInvalidToken       = errors.New("auth: invalid token")
	ErrTokenRevoked       = errors.New("auth: refresh token revoked or already used")
	ErrUserDisabled       = errors.New("auth: user disabled")
	ErrUserExists         = errors.New("auth: user already exists")
)

type User struct {
	ID           string
	Username     string
	DisplayName  string
	PasswordHash string
	Roles        []string
	Disabled     bool
}

type RefreshSession struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
}

// Store abstracts user lookup and refresh-token persistence. RotateRefreshSession
// must atomically consume previousID and save replacement.
type Store interface {
	UserByUsername(ctx context.Context, username string) (*User, error)
	UserByID(ctx context.Context, id string) (*User, error)
	CreateRefreshSession(ctx context.Context, session RefreshSession) error
	RotateRefreshSession(ctx context.Context, previousID, previousTokenHash string, replacement RefreshSession) error
	RevokeRefreshSession(ctx context.Context, id string) error
}

type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	TokenType        string
	ExpiresIn        int64
	RefreshExpiresIn int64
}

type Identity struct {
	UserID      string
	Username    string
	DisplayName string
	Roles       []string
	TokenID     string
}

type identityContextKey struct{}

func WithIdentity(ctx context.Context, identity Identity) context.Context {
	identity.Roles = append([]string(nil), identity.Roles...)
	return context.WithValue(ctx, identityContextKey{}, identity)
}

func IdentityFromContext(ctx context.Context) (Identity, bool) {
	if ctx == nil {
		return Identity{}, false
	}
	identity, ok := ctx.Value(identityContextKey{}).(Identity)
	if !ok || identity.UserID == "" {
		return Identity{}, false
	}
	identity.Roles = append([]string(nil), identity.Roles...)
	return identity, true
}
