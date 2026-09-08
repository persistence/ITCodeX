package acl

import (
	"context"
	"fmt"
	"net/http"

	"itcodex/server/internal/service/metadata"
)

// Identity is the deliberately small boundary between authentication and ACL.
// Authentication implementations can adapt their user/session without ACL
// importing the auth package.
type Identity interface {
	IdentityID() any
	IdentityRoles() []string
	IsLoggedIn() bool
}

// Principal is a convenient Identity implementation for simple integrations and tests.
type Principal struct {
	ID       any      `json:"id"`
	Roles    []string `json:"roles"`
	LoggedIn bool     `json:"loggedIn"`
}

func (p Principal) IdentityID() any         { return p.ID }
func (p Principal) IdentityRoles() []string { return append([]string(nil), p.Roles...) }
func (p Principal) IsLoggedIn() bool        { return p.LoggedIn }

type SubjectType string

const (
	SubjectRole     SubjectType = "role"
	SubjectLoggedIn SubjectType = "loggedIn"
	SubjectPublic   SubjectType = "public"
)

// FixedParamsFunc supplies trusted constraints from code. Its result is always
// ANDed after policy and caller filters.
type FixedParamsFunc func(context.Context, Identity) (metadata.Filter, error)

type Policy struct {
	SubjectType    SubjectType     `json:"subject_type"`
	Subject        string          `json:"subject"`
	Resource       string          `json:"resource"`
	Action         string          `json:"action"`
	RowFilter      metadata.Filter `json:"row_filter,omitempty"`
	ReadFields     []string        `json:"read_fields,omitempty"`
	WriteFields    []string        `json:"write_fields,omitempty"`
	Enabled        bool            `json:"enabled"`
	AddFixedParams FixedParamsFunc `json:"-"`
}

// Decision contains all constraints for an allowed operation. Empty field
// lists mean that ACL does not impose a field restriction.
type Decision struct {
	Allowed         bool            `json:"allowed"`
	Filter          metadata.Filter `json:"filter,omitempty"`
	ReadFields      []string        `json:"read_fields,omitempty"`
	WriteFields     []string        `json:"write_fields,omitempty"`
	ReadRestricted  bool            `json:"read_restricted,omitempty"`
	WriteRestricted bool            `json:"write_restricted,omitempty"`
}

// ForbiddenError is the default domain error returned by Authorize.
type ForbiddenError struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
	Message  string `json:"message"`
}

func (e *ForbiddenError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("无权对资源 %s 执行 %s 操作", e.Resource, e.Action)
}

func (e *ForbiddenError) Code() int       { return http.StatusForbidden }
func (e *ForbiddenError) HTTPStatus() int { return http.StatusForbidden }

type ForbiddenFactory func(resource, action string) error

type Option func(*Enforcer)

// WithForbiddenFactory customizes the 403 domain error returned on denial.
func WithForbiddenFactory(factory ForbiddenFactory) Option {
	return func(e *Enforcer) {
		if factory != nil {
			e.forbidden = factory
		}
	}
}
