package acl

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"

	"itcodex/server/internal/service/metadata"
)

type policySnapshot struct {
	policies []Policy
}

type fixedRule struct {
	resource string
	action   string
	fn       FixedParamsFunc
}

// Enforcer evaluates an immutable policy snapshot. Replacing policies is
// atomic, while readers run without locks.
type Enforcer struct {
	snapshot  atomic.Pointer[policySnapshot]
	fixed     atomic.Pointer[[]fixedRule]
	forbidden ForbiddenFactory
}

func NewEnforcer(policies []Policy, opts ...Option) (*Enforcer, error) {
	e := &Enforcer{
		forbidden: func(resource, action string) error {
			return &ForbiddenError{Resource: resource, Action: action}
		},
	}
	for _, opt := range opts {
		opt(e)
	}
	if err := e.ReplacePolicies(policies); err != nil {
		return nil, err
	}
	return e, nil
}

// ReplacePolicies validates, clones, and atomically publishes a full snapshot.
func (e *Enforcer) ReplacePolicies(policies []Policy) error {
	cloned := make([]Policy, len(policies))
	for i := range policies {
		if err := validatePolicy(policies[i]); err != nil {
			return fmt.Errorf("policy[%d]: %w", i, err)
		}
		cloned[i] = clonePolicy(policies[i])
	}
	e.snapshot.Store(&policySnapshot{policies: cloned})
	return nil
}

func (e *Enforcer) Allow(policy Policy) error {
	current := e.snapshot.Load()
	var policies []Policy
	if current != nil {
		policies = append(policies, current.policies...)
	}
	policies = append(policies, policy)
	return e.ReplacePolicies(policies)
}

func (e *Enforcer) AddFixedParams(resource, action string, fn FixedParamsFunc) {
	if fn == nil {
		return
	}
	current := e.fixed.Load()
	var rules []fixedRule
	if current != nil {
		rules = append(rules, (*current)...)
	}
	rules = append(rules, fixedRule{resource: resource, action: action, fn: fn})
	e.fixed.Store(&rules)
}

// Can is the convenient boolean form. Evaluation errors are treated as denial.
func (e *Enforcer) Can(ctx context.Context, identity Identity, resource, action string) bool {
	decision, err := e.Evaluate(ctx, identity, resource, action, nil)
	return err == nil && decision.Allowed
}

// Authorize returns the effective constraints, or a customizable 403 error.
// At most one caller filter may be supplied.
func (e *Enforcer) Authorize(ctx context.Context, identity Identity, resource, action string, callerFilter ...metadata.Filter) (Decision, error) {
	if len(callerFilter) > 1 {
		return Decision{}, fmt.Errorf("Authorize accepts at most one caller filter")
	}
	var filter metadata.Filter
	if len(callerFilter) == 1 {
		filter = callerFilter[0]
	}
	decision, err := e.Evaluate(ctx, identity, resource, action, filter)
	if err != nil {
		return Decision{}, err
	}
	if !decision.Allowed {
		return decision, e.forbidden(resource, action)
	}
	return decision, nil
}

// Evaluate computes matching grants. Matching row filters are ORed, the caller
// filter is then ANDed, and trusted callback filters are ANDed last.
func (e *Enforcer) Evaluate(ctx context.Context, identity Identity, resource, action string, callerFilter metadata.Filter) (Decision, error) {
	if hasSystemBypass(ctx) {
		return Decision{Allowed: true, Filter: cloneFilter(callerFilter)}, nil
	}

	snapshot := e.snapshot.Load()
	if snapshot == nil {
		return Decision{}, nil
	}

	var (
		matched           bool
		unrestrictedRows  bool
		rowFilters        []metadata.Filter
		fixedFilters      []metadata.Filter
		readFields        []string
		writeFields       []string
		readRestricted    bool
		writeRestricted   bool
		unrestrictedRead  bool
		unrestrictedWrite bool
	)
	for _, policy := range snapshot.policies {
		if !policy.Enabled ||
			!wildcardMatch(policy.Resource, resource) ||
			!wildcardMatch(policy.Action, action) ||
			!matchesSubject(policy, identity) {
			continue
		}

		matched = true
		resolved, err := resolveFilter(policy.RowFilter, identity)
		if err != nil {
			return Decision{}, err
		}
		if len(resolved) == 0 {
			unrestrictedRows = true
		} else {
			rowFilters = append(rowFilters, resolved)
		}
		if policy.ReadFields == nil {
			unrestrictedRead = true
		} else {
			readRestricted = true
			readFields = unionFields(readFields, policy.ReadFields)
		}
		if policy.WriteFields == nil {
			unrestrictedWrite = true
		} else {
			writeRestricted = true
			writeFields = unionFields(writeFields, policy.WriteFields)
		}

		if policy.AddFixedParams != nil {
			fixed, err := policy.AddFixedParams(ctx, identity)
			if err != nil {
				return Decision{}, fmt.Errorf("add fixed params: %w", err)
			}
			if len(fixed) > 0 {
				fixedFilters = append(fixedFilters, cloneFilter(fixed))
			}
		}
	}

	if !matched {
		return Decision{}, nil
	}

	var effective metadata.Filter
	if !unrestrictedRows && len(rowFilters) > 0 {
		if len(rowFilters) == 1 {
			effective = rowFilters[0]
		} else {
			effective = metadata.Filter{"$or": rowFilters}
		}
	}
	effective = andFilter(effective, cloneFilter(callerFilter))
	for _, fixed := range fixedFilters {
		effective = andFilter(effective, fixed)
	}
	if rules := e.fixed.Load(); rules != nil {
		for _, rule := range *rules {
			if !wildcardMatch(rule.resource, resource) || !wildcardMatch(rule.action, action) {
				continue
			}
			fixed, err := rule.fn(ctx, identity)
			if err != nil {
				return Decision{}, fmt.Errorf("add fixed params: %w", err)
			}
			effective = andFilter(effective, cloneFilter(fixed))
		}
	}

	return Decision{
		Allowed:         true,
		Filter:          effective,
		ReadFields:      readFields,
		WriteFields:     writeFields,
		ReadRestricted:  readRestricted && !unrestrictedRead,
		WriteRestricted: writeRestricted && !unrestrictedWrite,
	}, nil
}

func validatePolicy(policy Policy) error {
	switch policy.SubjectType {
	case SubjectRole, SubjectLoggedIn, SubjectPublic:
	default:
		return fmt.Errorf("unknown subject type %q", policy.SubjectType)
	}
	if policy.Resource == "" {
		return fmt.Errorf("resource is required")
	}
	if policy.Action == "" {
		return fmt.Errorf("action is required")
	}
	return validateVariables(policy.RowFilter)
}

func matchesSubject(policy Policy, identity Identity) bool {
	switch policy.SubjectType {
	case SubjectPublic:
		return true
	case SubjectLoggedIn:
		if identity == nil || !identity.IsLoggedIn() {
			return false
		}
		return policy.Subject == "" || wildcardMatch(policy.Subject, fmt.Sprint(identity.IdentityID()))
	case SubjectRole:
		if identity == nil {
			return false
		}
		for _, role := range identity.IdentityRoles() {
			if wildcardMatch(policy.Subject, role) {
				return true
			}
		}
	}
	return false
}

func unionFields(existing, added []string) []string {
	seen := make(map[string]struct{}, len(existing)+len(added))
	for _, field := range existing {
		seen[field] = struct{}{}
	}
	for _, field := range added {
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		existing = append(existing, field)
	}
	return existing
}

func andFilter(left, right metadata.Filter) metadata.Filter {
	if len(left) == 0 {
		return right
	}
	if len(right) == 0 {
		return left
	}
	return metadata.Filter{"$and": []metadata.Filter{left, right}}
}

func clonePolicy(policy Policy) Policy {
	policy.RowFilter = cloneFilter(policy.RowFilter)
	if policy.ReadFields != nil {
		policy.ReadFields = append([]string{}, policy.ReadFields...)
	}
	if policy.WriteFields != nil {
		policy.WriteFields = append([]string{}, policy.WriteFields...)
	}
	return policy
}

func cloneFilter(filter metadata.Filter) metadata.Filter {
	if filter == nil {
		return nil
	}
	return metadata.Filter(cloneMap(map[string]any(filter)))
}

func cloneMap(input map[string]any) map[string]any {
	result := make(map[string]any, len(input))
	for key, value := range input {
		result[key] = cloneValue(value)
	}
	return result
}

func cloneValue(value any) any {
	switch typed := value.(type) {
	case metadata.Filter:
		return cloneFilter(typed)
	case map[string]any:
		return cloneMap(typed)
	case []metadata.Filter:
		result := make([]metadata.Filter, len(typed))
		for i := range typed {
			result[i] = cloneFilter(typed[i])
		}
		return result
	case []string:
		return append([]string(nil), typed...)
	case []any:
		result := make([]any, len(typed))
		for i := range typed {
			result[i] = cloneValue(typed[i])
		}
		return result
	default:
		return value
	}
}

func validateVariables(filter metadata.Filter) error {
	_, err := walkVariables(filter, nil, false)
	return err
}

func resolveFilter(filter metadata.Filter, identity Identity) (metadata.Filter, error) {
	value, err := walkVariables(filter, identity, true)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, nil
	}
	return value.(metadata.Filter), nil
}

func walkVariables(value any, identity Identity, resolve bool) (any, error) {
	switch typed := value.(type) {
	case metadata.Filter:
		result := make(metadata.Filter, len(typed))
		for key, item := range typed {
			resolved, err := walkVariables(item, identity, resolve)
			if err != nil {
				return nil, err
			}
			result[key] = resolved
		}
		return result, nil
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			resolved, err := walkVariables(item, identity, resolve)
			if err != nil {
				return nil, err
			}
			result[key] = resolved
		}
		return result, nil
	case []metadata.Filter:
		result := make([]metadata.Filter, len(typed))
		for i := range typed {
			resolved, err := walkVariables(typed[i], identity, resolve)
			if err != nil {
				return nil, err
			}
			result[i] = resolved.(metadata.Filter)
		}
		return result, nil
	case []any:
		result := make([]any, len(typed))
		for i := range typed {
			resolved, err := walkVariables(typed[i], identity, resolve)
			if err != nil {
				return nil, err
			}
			result[i] = resolved
		}
		return result, nil
	case string:
		if !strings.HasPrefix(typed, "$") {
			return typed, nil
		}
		switch typed {
		case "$currentUser.id":
			if resolve {
				if identity == nil {
					return nil, nil
				}
				return identity.IdentityID(), nil
			}
		case "$currentUser.roles":
			if resolve {
				if identity == nil {
					return []any(nil), nil
				}
				roles := identity.IdentityRoles()
				result := make([]any, len(roles))
				for i := range roles {
					result[i] = roles[i]
				}
				return result, nil
			}
		default:
			return nil, fmt.Errorf("unsupported ACL variable %q", typed)
		}
		return typed, nil
	default:
		return cloneValue(value), nil
	}
}

var wildcardCache atomic.Pointer[map[string]*regexp.Regexp]

func wildcardMatch(pattern, value string) bool {
	cache := wildcardCache.Load()
	if cache != nil {
		if compiled := (*cache)[pattern]; compiled != nil {
			return compiled.MatchString(value)
		}
	}

	var expression strings.Builder
	expression.WriteByte('^')
	for _, char := range pattern {
		switch char {
		case '*':
			expression.WriteString(".*")
		case '?':
			expression.WriteByte('.')
		default:
			expression.WriteString(regexp.QuoteMeta(string(char)))
		}
	}
	expression.WriteByte('$')
	compiled := regexp.MustCompile(expression.String())

	// Duplicate compilation is harmless. Copy-on-write keeps published maps immutable.
	next := make(map[string]*regexp.Regexp)
	if cache != nil {
		for key, item := range *cache {
			next[key] = item
		}
	}
	next[pattern] = compiled
	wildcardCache.Store(&next)
	return compiled.MatchString(value)
}
