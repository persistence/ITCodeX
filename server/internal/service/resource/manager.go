package resource

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

var (
	ErrInvalidRequest = errors.New("resource: invalid action request")
	ErrActionNotFound = errors.New("resource: action handler not found")
)

// ActionRequest describes one operation on a resource.
type ActionRequest struct {
	Resource string
	Action   string
	Params   map[string]any
	Data     any
	Options  any
}

// ActionContext is passed through middleware and into the selected handler.
type ActionContext struct {
	context.Context
	Manager *Manager
	Request *ActionRequest
	State   map[string]any
}

func (c *ActionContext) Param(name string) any {
	if c == nil || c.Request == nil {
		return nil
	}
	return c.Request.Params[name]
}

// ActionHandler handles a resource action.
type ActionHandler func(*ActionContext) (any, error)

// Middleware wraps an ActionHandler. Middleware added first runs first.
type Middleware func(ActionHandler) ActionHandler

type registration struct {
	resourcePattern string
	actionPattern   string
	handler         ActionHandler
	specificity     int
	sequence        uint64
}

// Manager is a concurrent-safe resource action registry and dispatcher.
type Manager struct {
	mu          sync.RWMutex
	handlers    []registration
	middlewares []Middleware
	sequence    uint64
}

func NewManager() *Manager {
	return &Manager{}
}

// Register adds or replaces a handler and returns an idempotent unregister
// function. Patterns may contain "*" and "?" wildcards.
func (m *Manager) Register(resourcePattern, actionPattern string, handler ActionHandler) func() {
	if m == nil {
		return func() {}
	}
	resourcePattern = normalizePattern(resourcePattern)
	actionPattern = normalizePattern(actionPattern)
	if handler == nil {
		return func() {}
	}

	m.mu.Lock()
	m.sequence++
	sequence := m.sequence
	replaced := false
	for i := range m.handlers {
		if m.handlers[i].resourcePattern == resourcePattern && m.handlers[i].actionPattern == actionPattern {
			m.handlers[i] = newRegistration(resourcePattern, actionPattern, handler, sequence)
			replaced = true
			break
		}
	}
	if !replaced {
		m.handlers = append(m.handlers, newRegistration(resourcePattern, actionPattern, handler, sequence))
	}
	m.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			m.unregisterSequence(resourcePattern, actionPattern, sequence)
		})
	}
}

// Unregister removes the handler registered for an exact pattern pair.
func (m *Manager) Unregister(resourcePattern, actionPattern string) bool {
	if m == nil {
		return false
	}
	resourcePattern = normalizePattern(resourcePattern)
	actionPattern = normalizePattern(actionPattern)

	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.handlers {
		if m.handlers[i].resourcePattern == resourcePattern && m.handlers[i].actionPattern == actionPattern {
			m.handlers = append(m.handlers[:i], m.handlers[i+1:]...)
			return true
		}
	}
	return false
}

func (m *Manager) unregisterSequence(resourcePattern, actionPattern string, sequence uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.handlers {
		entry := m.handlers[i]
		if entry.resourcePattern == resourcePattern && entry.actionPattern == actionPattern && entry.sequence == sequence {
			m.handlers = append(m.handlers[:i], m.handlers[i+1:]...)
			return
		}
	}
}

// Use appends middleware to the dispatch chain.
func (m *Manager) Use(middlewares ...Middleware) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, middleware := range middlewares {
		if middleware != nil {
			m.middlewares = append(m.middlewares, middleware)
		}
	}
}

// Dispatch selects the most specific matching handler and executes it.
func (m *Manager) Dispatch(ctx context.Context, request *ActionRequest) (any, error) {
	if m == nil || request == nil || request.Resource == "" || request.Action == "" {
		return nil, ErrInvalidRequest
	}
	if ctx == nil {
		ctx = context.Background()
	}

	handler, middlewares := m.lookup(request.Resource, request.Action)
	if handler == nil {
		return nil, fmt.Errorf("%w: %s/%s", ErrActionNotFound, request.Resource, request.Action)
	}
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler(&ActionContext{
		Context: ctx,
		Manager: m,
		Request: request,
		State:   make(map[string]any),
	})
}

func (m *Manager) lookup(resourceName, actionName string) (ActionHandler, []Middleware) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var selected *registration
	for i := range m.handlers {
		entry := &m.handlers[i]
		if !wildcardMatch(entry.resourcePattern, resourceName) || !wildcardMatch(entry.actionPattern, actionName) {
			continue
		}
		if selected == nil || entry.specificity > selected.specificity ||
			(entry.specificity == selected.specificity && entry.sequence > selected.sequence) {
			selected = entry
		}
	}
	if selected == nil {
		return nil, nil
	}
	middlewares := append([]Middleware(nil), m.middlewares...)
	return selected.handler, middlewares
}

func newRegistration(resourcePattern, actionPattern string, handler ActionHandler, sequence uint64) registration {
	return registration{
		resourcePattern: resourcePattern,
		actionPattern:   actionPattern,
		handler:         handler,
		specificity:     patternSpecificity(resourcePattern) + patternSpecificity(actionPattern),
		sequence:        sequence,
	}
}

func normalizePattern(pattern string) string {
	if pattern == "" {
		return "*"
	}
	return pattern
}

func patternSpecificity(pattern string) int {
	score := 0
	for _, r := range pattern {
		if r != '*' && r != '?' {
			score++
		}
	}
	return score
}

func wildcardMatch(pattern, value string) bool {
	if pattern == "*" {
		return true
	}
	if !strings.ContainsAny(pattern, "*?") {
		return pattern == value
	}

	var expression strings.Builder
	expression.WriteByte('^')
	for _, r := range pattern {
		switch r {
		case '*':
			expression.WriteString(".*")
		case '?':
			expression.WriteByte('.')
		default:
			expression.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	expression.WriteByte('$')
	matched, err := regexp.MatchString(expression.String(), value)
	return err == nil && matched
}
