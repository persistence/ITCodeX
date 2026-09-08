package acl

import "context"

type enforcerContextKey struct{}
type systemBypassContextKey struct{}
type systemBypassToken struct{}

// WithEnforcer makes an Enforcer available to downstream service code.
func WithEnforcer(ctx context.Context, enforcer *Enforcer) context.Context {
	return context.WithValue(ctx, enforcerContextKey{}, enforcer)
}

func EnforcerFromContext(ctx context.Context) (*Enforcer, bool) {
	enforcer, ok := ctx.Value(enforcerContextKey{}).(*Enforcer)
	return enforcer, ok && enforcer != nil
}

// WithSystemBypass marks trusted internal work as unrestricted. The token and
// its construction stay private so callers cannot forge context values.
func WithSystemBypass(ctx context.Context) context.Context {
	return context.WithValue(ctx, systemBypassContextKey{}, systemBypassToken{})
}

func hasSystemBypass(ctx context.Context) bool {
	_, ok := ctx.Value(systemBypassContextKey{}).(systemBypassToken)
	return ok
}
