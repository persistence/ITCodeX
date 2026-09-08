package auth

import (
	"context"
	"strings"
	"sync"
	"time"
)

// MemoryStore is a concurrency-safe Store intended for tests and local
// development. Production integrations should persist refresh sessions.
type MemoryStore struct {
	mu              sync.RWMutex
	usersByID       map[string]*User
	userIDByName    map[string]string
	refreshSessions map[string]RefreshSession
	consumed        map[string]struct{}
	clock           func() time.Time
}

func NewMemoryStore(users ...*User) *MemoryStore {
	return NewMemoryStoreWithClock(time.Now, users...)
}

func NewMemoryStoreWithClock(clock func() time.Time, users ...*User) *MemoryStore {
	if clock == nil {
		clock = time.Now
	}
	store := &MemoryStore{
		usersByID:       make(map[string]*User),
		userIDByName:    make(map[string]string),
		refreshSessions: make(map[string]RefreshSession),
		consumed:        make(map[string]struct{}),
		clock:           clock,
	}
	for _, user := range users {
		_ = store.AddUser(user)
	}
	return store
}

func (s *MemoryStore) AddUser(user *User) error {
	if user == nil || strings.TrimSpace(user.ID) == "" || strings.TrimSpace(user.Username) == "" {
		return ErrUserNotFound
	}
	name := normalizeUsername(user.Username)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.usersByID[user.ID]; exists {
		return ErrUserExists
	}
	if _, exists := s.userIDByName[name]; exists {
		return ErrUserExists
	}
	s.usersByID[user.ID] = cloneUser(user)
	s.userIDByName[name] = user.ID
	return nil
}

func (s *MemoryStore) UserByUsername(_ context.Context, username string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.userIDByName[normalizeUsername(username)]
	if !ok {
		return nil, ErrUserNotFound
	}
	return cloneUser(s.usersByID[id]), nil
}

func (s *MemoryStore) UserByID(_ context.Context, id string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.usersByID[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return cloneUser(user), nil
}

func (s *MemoryStore) CreateRefreshSession(_ context.Context, session RefreshSession) error {
	if session.ID == "" || session.UserID == "" || session.ExpiresAt.IsZero() {
		return ErrInvalidToken
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.refreshSessions[session.ID]; exists {
		return ErrTokenRevoked
	}
	if _, consumed := s.consumed[session.ID]; consumed {
		return ErrTokenRevoked
	}
	s.refreshSessions[session.ID] = session
	return nil
}

func (s *MemoryStore) RotateRefreshSession(
	_ context.Context,
	previousID string,
	previousTokenHash string,
	replacement RefreshSession,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous, ok := s.refreshSessions[previousID]
	if !ok || previous.TokenHash != previousTokenHash || !previous.ExpiresAt.After(s.clock()) {
		delete(s.refreshSessions, previousID)
		s.consumed[previousID] = struct{}{}
		return ErrTokenRevoked
	}
	if replacement.ID == "" ||
		replacement.UserID != previous.UserID ||
		!replacement.ExpiresAt.After(s.clock()) {
		return ErrInvalidToken
	}
	if _, exists := s.refreshSessions[replacement.ID]; exists {
		return ErrTokenRevoked
	}
	if _, consumed := s.consumed[replacement.ID]; consumed {
		return ErrTokenRevoked
	}
	delete(s.refreshSessions, previousID)
	s.consumed[previousID] = struct{}{}
	s.refreshSessions[replacement.ID] = replacement
	return nil
}

func (s *MemoryStore) RevokeRefreshSession(_ context.Context, id string) error {
	if id == "" {
		return ErrInvalidToken
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.refreshSessions, id)
	s.consumed[id] = struct{}{}
	return nil
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}
