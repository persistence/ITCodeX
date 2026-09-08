package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPasswordHash(t *testing.T) {
	first, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	second, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("password hashes must use random salts")
	}
	if !VerifyPassword(first, "correct horse battery staple") {
		t.Fatal("valid password was rejected")
	}
	if VerifyPassword(first, "wrong password") {
		t.Fatal("invalid password was accepted")
	}
}

func TestServiceLoginRefreshLogoutAndMe(t *testing.T) {
	passwordHash, err := HashPassword("long-enough-password")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	store := NewMemoryStoreWithClock(clock, &User{
		ID:           "user-1",
		Username:     "alice",
		DisplayName:  "Alice",
		PasswordHash: passwordHash,
		Roles:        []string{"admin"},
	})
	service, err := NewService(store, Config{
		Secret:     []byte("0123456789abcdef0123456789abcdef"),
		Issuer:     "itcodex-test",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 24 * time.Hour,
		Clock:      clock,
	})
	if err != nil {
		t.Fatal(err)
	}

	pair, err := service.Login(context.Background(), "Alice", "long-enough-password")
	if err != nil {
		t.Fatal(err)
	}
	identityCtx, err := service.Authenticate(context.Background(), pair.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	identity, ok := IdentityFromContext(identityCtx)
	if !ok || identity.UserID != "user-1" || len(identity.Roles) != 1 {
		t.Fatalf("unexpected identity: %#v", identity)
	}
	me, err := service.Me(identityCtx)
	if err != nil {
		t.Fatal(err)
	}
	if me.PasswordHash != "" || me.Username != "alice" {
		t.Fatalf("unexpected public user: %#v", me)
	}

	rotated, err := service.Refresh(context.Background(), pair.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if rotated.RefreshToken == pair.RefreshToken {
		t.Fatal("refresh token was not rotated")
	}
	if _, err = service.Refresh(context.Background(), pair.RefreshToken); !errors.Is(err, ErrTokenRevoked) {
		t.Fatalf("reused refresh token: got %v, want %v", err, ErrTokenRevoked)
	}
	if err = service.Logout(context.Background(), rotated.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Refresh(context.Background(), rotated.RefreshToken); !errors.Is(err, ErrTokenRevoked) {
		t.Fatalf("revoked refresh token: got %v, want %v", err, ErrTokenRevoked)
	}
}

func TestAuthenticateRejectsTamperedToken(t *testing.T) {
	passwordHash, err := HashPassword("long-enough-password")
	if err != nil {
		t.Fatal(err)
	}
	store := NewMemoryStore(&User{
		ID:           "user-1",
		Username:     "alice",
		PasswordHash: passwordHash,
	})
	service, err := NewService(store, Config{
		Secret: []byte("0123456789abcdef0123456789abcdef"),
	})
	if err != nil {
		t.Fatal(err)
	}
	pair, err := service.Login(context.Background(), "alice", "long-enough-password")
	if err != nil {
		t.Fatal(err)
	}
	tampered := pair.AccessToken[:len(pair.AccessToken)-1] + "A"
	if _, err = service.Authenticate(context.Background(), tampered); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("tampered access token: got %v, want %v", err, ErrUnauthenticated)
	}
}
