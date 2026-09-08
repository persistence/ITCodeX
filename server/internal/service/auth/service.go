package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	tokenTypeAccess  = "access"
	tokenTypeRefresh = "refresh"
)

type Config struct {
	Secret     []byte
	Issuer     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	Clock      func() time.Time
}

type Service struct {
	store      Store
	secret     []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
	clock      func() time.Time
}

func NewService(store Store, config Config) (*Service, error) {
	if store == nil {
		return nil, fmt.Errorf("auth: store is required")
	}
	if len(config.Secret) < 32 {
		return nil, fmt.Errorf("auth: JWT secret must contain at least 32 bytes")
	}
	if config.Issuer == "" {
		config.Issuer = "itcodex"
	}
	if config.AccessTTL == 0 {
		config.AccessTTL = 15 * time.Minute
	}
	if config.RefreshTTL == 0 {
		config.RefreshTTL = 7 * 24 * time.Hour
	}
	if config.AccessTTL < time.Second || config.RefreshTTL <= config.AccessTTL {
		return nil, fmt.Errorf("auth: invalid token lifetime configuration")
	}
	if config.Clock == nil {
		config.Clock = time.Now
	}
	return &Service{
		store:      store,
		secret:     append([]byte(nil), config.Secret...),
		issuer:     config.Issuer,
		accessTTL:  config.AccessTTL,
		refreshTTL: config.RefreshTTL,
		clock:      config.Clock,
	}, nil
}

func (s *Service) Login(ctx context.Context, username, password string) (*TokenPair, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, ErrInvalidCredentials
	}
	user, err := s.store.UserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// Do equivalent password work to reduce username-enumeration timing leaks.
			_ = pbkdf2SHA256([]byte(password), []byte("itcodex-auth-dummy"), passwordIterations, passwordKeySize)
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("auth: find user: %w", err)
	}
	if !VerifyPassword(user.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}
	if user.Disabled {
		return nil, ErrUserDisabled
	}

	pair, session, err := s.issueTokenPair(user)
	if err != nil {
		return nil, err
	}
	if err = s.store.CreateRefreshSession(ctx, session); err != nil {
		return nil, fmt.Errorf("auth: save refresh session: %w", err)
	}
	return pair, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	now := s.clock().UTC()
	claims, err := parseJWT(s.secret, s.issuer, tokenTypeRefresh, refreshToken, now)
	if err != nil {
		return nil, ErrInvalidToken
	}
	user, err := s.store.UserByID(ctx, claims.Subject)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("auth: find user: %w", err)
	}
	if user.Disabled {
		return nil, ErrUserDisabled
	}

	pair, replacement, err := s.issueTokenPair(user)
	if err != nil {
		return nil, err
	}
	if err = s.store.RotateRefreshSession(ctx, claims.ID, tokenHash(refreshToken), replacement); err != nil {
		if errors.Is(err, ErrTokenRevoked) {
			return nil, ErrTokenRevoked
		}
		return nil, fmt.Errorf("auth: rotate refresh session: %w", err)
	}
	return pair, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	claims, err := parseJWT(
		s.secret,
		s.issuer,
		tokenTypeRefresh,
		refreshToken,
		s.clock().UTC(),
	)
	if err != nil {
		return ErrInvalidToken
	}
	if err = s.store.RevokeRefreshSession(ctx, claims.ID); err != nil {
		return fmt.Errorf("auth: revoke refresh session: %w", err)
	}
	return nil
}

func (s *Service) Me(ctx context.Context) (*User, error) {
	identity, ok := IdentityFromContext(ctx)
	if !ok {
		return nil, ErrUnauthenticated
	}
	user, err := s.store.UserByID(ctx, identity.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrUnauthenticated
		}
		return nil, fmt.Errorf("auth: find user: %w", err)
	}
	if user.Disabled {
		return nil, ErrUserDisabled
	}
	return publicUser(user), nil
}

// Authenticate validates an access token and returns a context carrying Identity.
func (s *Service) Authenticate(ctx context.Context, accessToken string) (context.Context, error) {
	claims, err := parseJWT(
		s.secret,
		s.issuer,
		tokenTypeAccess,
		accessToken,
		s.clock().UTC(),
	)
	if err != nil {
		return ctx, ErrUnauthenticated
	}
	user, err := s.store.UserByID(ctx, claims.Subject)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return ctx, ErrUnauthenticated
		}
		return ctx, fmt.Errorf("auth: find user: %w", err)
	}
	if user.Disabled {
		return ctx, ErrUserDisabled
	}
	identity := Identity{
		UserID:      user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Roles:       append([]string(nil), user.Roles...),
		TokenID:     claims.ID,
	}
	return WithIdentity(ctx, identity), nil
}

func BearerToken(authorization string) (string, error) {
	const prefix = "Bearer "
	if len(authorization) <= len(prefix) ||
		!strings.EqualFold(authorization[:len(prefix)], prefix) {
		return "", ErrUnauthenticated
	}
	token := strings.TrimSpace(authorization[len(prefix):])
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return "", ErrUnauthenticated
	}
	return token, nil
}

func (s *Service) issueTokenPair(user *User) (*TokenPair, RefreshSession, error) {
	now := s.clock().UTC()
	accessID, err := randomTokenID()
	if err != nil {
		return nil, RefreshSession{}, err
	}
	refreshID, err := randomTokenID()
	if err != nil {
		return nil, RefreshSession{}, err
	}
	accessExpiry := now.Add(s.accessTTL)
	refreshExpiry := now.Add(s.refreshTTL)
	base := tokenClaims{
		Issuer:   s.issuer,
		Subject:  user.ID,
		Username: user.Username,
		Roles:    append([]string(nil), user.Roles...),
		IssuedAt: now.Unix(),
	}
	accessClaims := base
	accessClaims.TokenType = tokenTypeAccess
	accessClaims.ID = accessID
	accessClaims.ExpiresAt = accessExpiry.Unix()
	accessToken, err := signJWT(s.secret, accessClaims)
	if err != nil {
		return nil, RefreshSession{}, err
	}
	refreshClaims := base
	refreshClaims.Roles = nil
	refreshClaims.TokenType = tokenTypeRefresh
	refreshClaims.ID = refreshID
	refreshClaims.ExpiresAt = refreshExpiry.Unix()
	refreshToken, err := signJWT(s.secret, refreshClaims)
	if err != nil {
		return nil, RefreshSession{}, err
	}
	return &TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		TokenType:        "Bearer",
		ExpiresIn:        int64(s.accessTTL / time.Second),
		RefreshExpiresIn: int64(s.refreshTTL / time.Second),
	}, RefreshSession{
		ID:        refreshID,
		UserID:    user.ID,
		TokenHash: tokenHash(refreshToken),
		ExpiresAt: refreshExpiry,
	}, nil
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomTokenID() (string, error) {
	random := make([]byte, 24)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("auth: generate token ID: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(random), nil
}

func publicUser(user *User) *User {
	copy := cloneUser(user)
	copy.PasswordHash = ""
	return copy
}

func cloneUser(user *User) *User {
	if user == nil {
		return nil
	}
	copy := *user
	copy.Roles = append([]string(nil), user.Roles...)
	return &copy
}
