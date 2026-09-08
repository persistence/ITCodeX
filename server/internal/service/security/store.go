package security

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"itcodex/server/internal/service/acl"
	"itcodex/server/internal/service/auth"
	"itcodex/server/internal/service/metadata"
)

type Store struct {
	db     *sql.DB
	prefix string
}

type StoredPolicy struct {
	ID int64 `json:"id"`
	acl.Policy
}

type Role struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

func NewStore(db *metadata.Database) (*Store, error) {
	if db == nil || db.SqlDB() == nil {
		return nil, fmt.Errorf("security: metadata database is required")
	}
	return &Store{db: db.SqlDB(), prefix: db.TablePrefix()}, nil
}

func (s *Store) table(name string) string {
	return metadata.QuoteIdent(s.prefix + name)
}

func (s *Store) UserByUsername(ctx context.Context, username string) (*auth.User, error) {
	return s.user(ctx, "username", username)
}

func (s *Store) UserByID(ctx context.Context, id string) (*auth.User, error) {
	return s.user(ctx, "id", id)
}

func (s *Store) user(ctx context.Context, column, value string) (*auth.User, error) {
	query := fmt.Sprintf(
		`SELECT id, username, display_name, password_hash, enabled FROM %s WHERE %s = ?`,
		s.table("auth_users"), metadata.QuoteIdent(column),
	)
	var (
		id      int64
		user    auth.User
		enabled bool
	)
	if err := s.db.QueryRowContext(ctx, query, value).Scan(
		&id, &user.Username, &user.DisplayName, &user.PasswordHash, &enabled,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, auth.ErrUserNotFound
		}
		return nil, err
	}
	user.ID = strconv.FormatInt(id, 10)
	user.Disabled = !enabled
	roles, err := s.rolesForUser(ctx, id)
	if err != nil {
		return nil, err
	}
	user.Roles = roles
	return &user, nil
}

func (s *Store) rolesForUser(ctx context.Context, userID int64) ([]string, error) {
	query := fmt.Sprintf(
		`SELECT r.name FROM %s ur JOIN %s r ON r.id = ur.role_id WHERE ur.user_id = ? ORDER BY r.name`,
		s.table("auth_user_roles"), s.table("auth_roles"),
	)
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (s *Store) CreateRefreshSession(ctx context.Context, session auth.RefreshSession) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(
		`INSERT INTO %s (id, user_id, token_hash, expires_at, created_at) VALUES (?, ?, ?, ?, NOW())`,
		s.table("auth_sessions"),
	), session.ID, session.UserID, session.TokenHash, session.ExpiresAt)
	return err
}

func (s *Store) RotateRefreshSession(ctx context.Context, previousID, previousTokenHash string, replacement auth.RefreshSession) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, fmt.Sprintf(
		`UPDATE %s SET revoked_at = NOW() WHERE id = ? AND token_hash = ? AND revoked_at IS NULL AND expires_at > NOW()`,
		s.table("auth_sessions"),
	), previousID, previousTokenHash)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return auth.ErrTokenRevoked
	}
	if _, err = tx.ExecContext(ctx, fmt.Sprintf(
		`INSERT INTO %s (id, user_id, token_hash, expires_at, created_at) VALUES (?, ?, ?, ?, NOW())`,
		s.table("auth_sessions"),
	), replacement.ID, replacement.UserID, replacement.TokenHash, replacement.ExpiresAt); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) RevokeRefreshSession(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, fmt.Sprintf(
		`UPDATE %s SET revoked_at = NOW() WHERE id = ? AND revoked_at IS NULL`,
		s.table("auth_sessions"),
	), id)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return auth.ErrTokenRevoked
	}
	return nil
}

func (s *Store) CreateUser(ctx context.Context, username, displayName, password string, enabled bool) (*auth.User, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	result, err := s.db.ExecContext(ctx, fmt.Sprintf(
		`INSERT INTO %s (username, display_name, password_hash, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, NOW(), NOW())`,
		s.table("auth_users"),
	), username, displayName, hash, enabled)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return s.UserByID(ctx, strconv.FormatInt(id, 10))
}

func (s *Store) ListUsers(ctx context.Context) ([]*auth.User, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(
		`SELECT id FROM %s ORDER BY id`, s.table("auth_users"),
	))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	users := make([]*auth.User, 0, len(ids))
	for _, id := range ids {
		user, err := s.UserByID(ctx, strconv.FormatInt(id, 10))
		if err != nil {
			return nil, err
		}
		user.PasswordHash = ""
		users = append(users, user)
	}
	return users, rows.Err()
}

func (s *Store) CreateRole(ctx context.Context, name, displayName string) (*Role, error) {
	result, err := s.db.ExecContext(ctx, fmt.Sprintf(
		`INSERT INTO %s (name, display_name, created_at, updated_at) VALUES (?, ?, NOW(), NOW())`,
		s.table("auth_roles"),
	), name, displayName)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return &Role{ID: id, Name: name, DisplayName: displayName}, nil
}

func (s *Store) ListRoles(ctx context.Context) ([]Role, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(
		`SELECT id, name, display_name FROM %s ORDER BY id`, s.table("auth_roles"),
	))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var roles []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.Name, &role.DisplayName); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (s *Store) AssignRole(ctx context.Context, userID, roleID int64) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(
		`INSERT IGNORE INTO %s (user_id, role_id, created_at) VALUES (?, ?, NOW())`,
		s.table("auth_user_roles"),
	), userID, roleID)
	return err
}

func (s *Store) UnassignRole(ctx context.Context, userID, roleID int64) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(
		`DELETE FROM %s WHERE user_id=? AND role_id=?`,
		s.table("auth_user_roles"),
	), userID, roleID)
	return err
}

func (s *Store) LoadPolicies(ctx context.Context) ([]acl.Policy, error) {
	stored, err := s.ListPolicies(ctx)
	if err != nil {
		return nil, err
	}
	policies := make([]acl.Policy, 0, len(stored))
	for _, item := range stored {
		policies = append(policies, item.Policy)
	}
	return policies, nil
}

func (s *Store) ListPolicies(ctx context.Context) ([]StoredPolicy, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(
		`SELECT id, subject_type, subject, resource, action, row_filter, read_fields, write_fields, enabled FROM %s ORDER BY id`,
		s.table("acl_policies"),
	))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var policies []StoredPolicy
	for rows.Next() {
		var (
			item                     StoredPolicy
			subjectType              string
			rowFilter, reads, writes sql.NullString
		)
		if err := rows.Scan(
			&item.ID, &subjectType, &item.Subject, &item.Resource, &item.Action,
			&rowFilter, &reads, &writes, &item.Enabled,
		); err != nil {
			return nil, err
		}
		item.SubjectType = acl.SubjectType(subjectType)
		if err := decodeJSON(rowFilter, &item.RowFilter); err != nil {
			return nil, err
		}
		if err := decodeJSON(reads, &item.ReadFields); err != nil {
			return nil, err
		}
		if err := decodeJSON(writes, &item.WriteFields); err != nil {
			return nil, err
		}
		policies = append(policies, item)
	}
	return policies, rows.Err()
}

func (s *Store) SavePolicy(ctx context.Context, item StoredPolicy) (int64, error) {
	rowFilter, _ := json.Marshal(item.RowFilter)
	reads, _ := json.Marshal(item.ReadFields)
	writes, _ := json.Marshal(item.WriteFields)
	if item.ID > 0 {
		_, err := s.db.ExecContext(ctx, fmt.Sprintf(
			`UPDATE %s SET subject_type=?, subject=?, resource=?, action=?, row_filter=?, read_fields=?, write_fields=?, enabled=?, updated_at=NOW() WHERE id=?`,
			s.table("acl_policies"),
		), item.SubjectType, item.Subject, item.Resource, item.Action, rowFilter, reads, writes, item.Enabled, item.ID)
		return item.ID, err
	}
	result, err := s.db.ExecContext(ctx, fmt.Sprintf(
		`INSERT INTO %s (subject_type, subject, resource, action, row_filter, read_fields, write_fields, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`,
		s.table("acl_policies"),
	), item.SubjectType, item.Subject, item.Resource, item.Action, rowFilter, reads, writes, item.Enabled)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) DeletePolicy(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(
		`DELETE FROM %s WHERE id=?`, s.table("acl_policies"),
	), id)
	return err
}

func (s *Store) EnsureBootstrapAdmin(ctx context.Context, username, password string) error {
	if username == "" || password == "" {
		return nil
	}
	var count int
	if err := s.db.QueryRowContext(ctx, fmt.Sprintf(
		`SELECT COUNT(*) FROM %s`, s.table("auth_users"),
	)).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	userResult, err := tx.ExecContext(ctx, fmt.Sprintf(
		`INSERT INTO %s (username, display_name, password_hash, enabled, created_at, updated_at) VALUES (?, ?, ?, 1, NOW(), NOW())`,
		s.table("auth_users"),
	), username, "Administrator", hash)
	if err != nil {
		return err
	}
	userID, _ := userResult.LastInsertId()
	roleResult, err := tx.ExecContext(ctx, fmt.Sprintf(
		`INSERT INTO %s (name, display_name, created_at, updated_at) VALUES ('admin', '管理员', NOW(), NOW())`,
		s.table("auth_roles"),
	))
	if err != nil {
		return err
	}
	roleID, _ := roleResult.LastInsertId()
	if _, err = tx.ExecContext(ctx, fmt.Sprintf(
		`INSERT INTO %s (user_id, role_id, created_at) VALUES (?, ?, NOW())`,
		s.table("auth_user_roles"),
	), userID, roleID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, fmt.Sprintf(
		`INSERT INTO %s (subject_type, subject, resource, action, enabled, created_at, updated_at) VALUES ('role', 'admin', '*', '*', 1, NOW(), NOW())`,
		s.table("acl_policies"),
	)); err != nil {
		return err
	}
	return tx.Commit()
}

func decodeJSON(value sql.NullString, destination any) error {
	if !value.Valid || value.String == "" || value.String == "null" {
		return nil
	}
	return json.Unmarshal([]byte(value.String), destination)
}

var _ auth.Store = (*Store)(nil)
