package metadata

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/gogf/gf/v2/os/gtime"

	"itcodex/server/internal/consts"
	"itcodex/server/internal/dao"
	"itcodex/server/internal/model/do"
	"itcodex/server/internal/model/entity"
	modelmd "itcodex/server/internal/model/metadata"
)

// useSystemDAO 仅默认前缀 c_ 走 gf gen dao；测试用随机前缀仍走 database/sql。
func (d *Database) useSystemDAO() bool {
	return d != nil && d.TablePrefix() == consts.DefaultTablePrefix
}

func (d *Database) insertCollectionRow(ctx context.Context, name, displayName, typ, optionsJSON string) error {
	if d.useSystemDAO() {
		_, err := dao.Collections.Ctx(ctx).Data(do.Collections{
			Name:        name,
			DisplayName: displayName,
			Type:        typ,
			Options:     optionsJSON,
		}).Insert()
		if err != nil {
			return NewSystemError(err)
		}
		return nil
	}
	_, err := d.db.Exec(ctx, fmt.Sprintf(
		`INSERT INTO %s (name, display_name, type, options) VALUES (?, ?, ?, ?)`,
		quoteIdent(d.TablePrefix()+"collections"),
	), name, displayName, typ, optionsJSON)
	if err != nil {
		return NewSystemError(err)
	}
	return nil
}

func (d *Database) updateCollectionRow(ctx context.Context, name, displayName, optionsJSON string) error {
	if d.useSystemDAO() {
		_, err := dao.Collections.Ctx(ctx).Where(do.Collections{Name: name}).Data(do.Collections{
			DisplayName: displayName,
			Options:     optionsJSON,
		}).Update()
		if err != nil {
			return NewSystemError(err)
		}
		return nil
	}
	_, err := d.db.Exec(ctx, fmt.Sprintf(
		`UPDATE %s SET display_name = ?, options = ? WHERE name = ?`,
		quoteIdent(d.TablePrefix()+"collections"),
	), displayName, optionsJSON, name)
	if err != nil {
		return NewSystemError(err)
	}
	return nil
}

func (d *Database) deleteCollectionMeta(ctx context.Context, name string) error {
	if d.useSystemDAO() {
		if _, err := dao.Fields.Ctx(ctx).Where(do.Fields{CollectionName: name}).Delete(); err != nil {
			return NewSystemError(err)
		}
		if _, err := dao.Indexes.Ctx(ctx).Where(do.Indexes{CollectionName: name}).Delete(); err != nil {
			return NewSystemError(err)
		}
		if _, err := dao.YaegiScripts.Ctx(ctx).Where(do.YaegiScripts{CollectionName: name}).Delete(); err != nil {
			return NewSystemError(err)
		}
		if _, err := dao.Collections.Ctx(ctx).Where(do.Collections{Name: name}).Delete(); err != nil {
			return NewSystemError(err)
		}
		return nil
	}
	prefix := d.TablePrefix()
	if _, err := d.db.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE collection_name = ?`, quoteIdent(prefix+"fields")), name); err != nil {
		return NewSystemError(err)
	}
	if _, err := d.db.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE collection_name = ?`, quoteIdent(prefix+"indexes")), name); err != nil {
		return NewSystemError(err)
	}
	if _, err := d.db.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE collection_name = ?`, quoteIdent(prefix+"yaegi_scripts")), name); err != nil {
		return NewSystemError(err)
	}
	if _, err := d.db.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE name = ?`, quoteIdent(prefix+"collections")), name); err != nil {
		return NewSystemError(err)
	}
	return nil
}

func (d *Database) insertFieldRow(ctx context.Context, collectionName, name, typ, displayName string, required, unique, indexed bool, optionsJSON string, sort int) error {
	if d.useSystemDAO() {
		_, err := dao.Fields.Ctx(ctx).Data(do.Fields{
			CollectionName: collectionName,
			Name:           name,
			Type:           typ,
			DisplayName:    displayName,
			IsRequired:     boolToTiny(required),
			IsUnique:       boolToTiny(unique),
			IsIndexed:      boolToTiny(indexed),
			Options:        optionsJSON,
			Sort:           sort,
		}).Insert()
		if err != nil {
			return NewSystemError(err)
		}
		return nil
	}
	_, err := d.db.Exec(ctx, fmt.Sprintf(
		`INSERT INTO %s (collection_name, name, type, display_name, is_required, is_unique, is_indexed, options, sort) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		quoteIdent(d.TablePrefix()+"fields"),
	), collectionName, name, typ, displayName, required, unique, indexed, optionsJSON, sort)
	if err != nil {
		return NewSystemError(err)
	}
	return nil
}

func (d *Database) updateFieldRow(ctx context.Context, collectionName, name, displayName string, required, unique, indexed bool, optionsJSON string) error {
	if d.useSystemDAO() {
		_, err := dao.Fields.Ctx(ctx).Where(do.Fields{
			CollectionName: collectionName,
			Name:           name,
		}).Data(do.Fields{
			DisplayName: displayName,
			IsRequired:  boolToTiny(required),
			IsUnique:    boolToTiny(unique),
			IsIndexed:   boolToTiny(indexed),
			Options:     optionsJSON,
		}).Update()
		if err != nil {
			return NewSystemError(err)
		}
		return nil
	}
	_, err := d.db.Exec(ctx, fmt.Sprintf(
		`UPDATE %s SET display_name = ?, is_required = ?, is_unique = ?, is_indexed = ?, options = ? WHERE collection_name = ? AND name = ?`,
		quoteIdent(d.TablePrefix()+"fields"),
	), displayName, required, unique, indexed, optionsJSON, collectionName, name)
	if err != nil {
		return NewSystemError(err)
	}
	return nil
}

func (d *Database) deleteFieldRow(ctx context.Context, collectionName, name string) error {
	if d.useSystemDAO() {
		_, err := dao.Fields.Ctx(ctx).Where(do.Fields{CollectionName: collectionName, Name: name}).Delete()
		if err != nil {
			return NewSystemError(err)
		}
		return nil
	}
	_, err := d.db.Exec(ctx, fmt.Sprintf(
		`DELETE FROM %s WHERE collection_name = ? AND name = ?`,
		quoteIdent(d.TablePrefix()+"fields"),
	), collectionName, name)
	if err != nil {
		return NewSystemError(err)
	}
	return nil
}

func (d *Database) insertIndexRow(ctx context.Context, collectionName, name, fieldsJSON string, unique bool) error {
	if d.useSystemDAO() {
		_, err := dao.Indexes.Ctx(ctx).Data(do.Indexes{
			CollectionName: collectionName,
			Name:           name,
			Fields:         fieldsJSON,
			Unique:         boolToTiny(unique),
		}).Insert()
		if err != nil {
			return NewSystemError(err)
		}
		return nil
	}
	_, err := d.db.Exec(ctx, fmt.Sprintf(
		`INSERT INTO %s (collection_name, name, fields, %s) VALUES (?, ?, ?, ?)`,
		quoteIdent(d.TablePrefix()+"indexes"),
		quoteIdent("unique"),
	), collectionName, name, fieldsJSON, unique)
	if err != nil {
		return NewSystemError(err)
	}
	return nil
}

func (d *Database) deleteIndexRow(ctx context.Context, collectionName, name string) error {
	if d.useSystemDAO() {
		_, err := dao.Indexes.Ctx(ctx).Where(do.Indexes{CollectionName: collectionName, Name: name}).Delete()
		if err != nil {
			return NewSystemError(err)
		}
		return nil
	}
	_, err := d.db.Exec(ctx, fmt.Sprintf(
		`DELETE FROM %s WHERE collection_name = ? AND name = ?`,
		quoteIdent(d.TablePrefix()+"indexes"),
	), collectionName, name)
	if err != nil {
		return NewSystemError(err)
	}
	return nil
}

func (d *Database) ListScripts(ctx context.Context, collection, hook string) ([]*modelmd.YaegiScript, error) {
	return d.listScripts(ctx, collection, hook)
}

func (d *Database) SaveScript(ctx context.Context, script *modelmd.YaegiScript) error {
	return d.saveScriptRow(ctx, script)
}

func (d *Database) SetScriptEnabled(ctx context.Context, id int64, enabled bool) error {
	return d.setScriptEnabled(ctx, id, enabled)
}

func (d *Database) DeleteScript(ctx context.Context, id int64) error {
	return d.deleteScriptRow(ctx, id)
}

func (d *Database) ToggleScript(ctx context.Context, id int64) (*modelmd.YaegiScript, error) {
	script, err := d.getScript(ctx, id)
	if err != nil {
		return nil, err
	}
	script.Enabled = !script.Enabled
	if err := d.setScriptEnabled(ctx, id, script.Enabled); err != nil {
		return nil, err
	}
	return script, nil
}

func (d *Database) listScripts(ctx context.Context, collection, hook string) ([]*modelmd.YaegiScript, error) {
	if d.useSystemDAO() {
		m := dao.YaegiScripts.Ctx(ctx)
		if collection != "" {
			m = m.Where(do.YaegiScripts{CollectionName: collection})
		}
		if hook != "" {
			m = m.Where(do.YaegiScripts{HookPoint: hook})
		}
		var rows []entity.YaegiScripts
		if err := m.OrderDesc(dao.YaegiScripts.Columns().Id).Scan(&rows); err != nil {
			return nil, NewSystemError(err)
		}
		out := make([]*modelmd.YaegiScript, 0, len(rows))
		for i := range rows {
			out = append(out, entityScriptToModel(&rows[i]))
		}
		return out, nil
	}
	query := fmt.Sprintf(
		`SELECT id, collection_name, name, hook_point, content, api_path, http_method, enabled, priority, options, created_at, updated_at FROM %s ORDER BY id DESC`,
		quoteIdent(d.TablePrefix()+"yaegi_scripts"),
	)
	rows, err := d.db.Query(ctx, query)
	if err != nil {
		return nil, NewSystemError(err)
	}
	defer rows.Close()
	var scripts []*modelmd.YaegiScript
	for rows.Next() {
		s, err := scanScriptRow(rows)
		if err != nil {
			return nil, err
		}
		if collection != "" && s.CollectionName != collection {
			continue
		}
		if hook != "" && s.HookPoint != hook {
			continue
		}
		scripts = append(scripts, s)
	}
	return scripts, nil
}

func (d *Database) getScript(ctx context.Context, id int64) (*modelmd.YaegiScript, error) {
	if d.useSystemDAO() {
		var row entity.YaegiScripts
		err := dao.YaegiScripts.Ctx(ctx).Where(do.YaegiScripts{Id: id}).Scan(&row)
		if err != nil {
			return nil, NewSystemError(err)
		}
		if row.Id == 0 {
			return nil, NewNotFoundError("脚本", "id", id)
		}
		return entityScriptToModel(&row), nil
	}
	query := fmt.Sprintf(
		`SELECT id, collection_name, name, hook_point, content, api_path, http_method, enabled, priority, options, created_at, updated_at FROM %s WHERE id=?`,
		quoteIdent(d.TablePrefix()+"yaegi_scripts"),
	)
	row := d.db.QueryRow(ctx, query, id)
	s, err := scanScriptRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewNotFoundError("脚本", "id", id)
		}
		return nil, NewSystemError(err)
	}
	return s, nil
}

func (d *Database) saveScriptRow(ctx context.Context, script *modelmd.YaegiScript) error {
	opts := script.Options
	if opts == "" {
		opts = "{}"
	}
	if d.useSystemDAO() {
		data := do.YaegiScripts{
			CollectionName: script.CollectionName,
			Name:           script.Name,
			HookPoint:      script.HookPoint,
			Content:        script.Content,
			ApiPath:        script.APIPath,
			HttpMethod:     script.HTTPMethod,
			Enabled:        boolToTiny(script.Enabled),
			Priority:       script.Priority,
			Options:        opts,
		}
		if script.Id > 0 {
			_, err := dao.YaegiScripts.Ctx(ctx).Where(do.YaegiScripts{Id: script.Id}).Data(data).Update()
			if err != nil {
				return NewSystemError(err)
			}
			return nil
		}
		res, err := dao.YaegiScripts.Ctx(ctx).Data(data).Insert()
		if err != nil {
			return NewSystemError(err)
		}
		id, _ := res.LastInsertId()
		script.Id = id
		return nil
	}
	if script.Id > 0 {
		_, err := d.db.Exec(ctx, fmt.Sprintf(
			`UPDATE %s SET collection_name=?, name=?, hook_point=?, content=?, api_path=?, http_method=?, enabled=?, priority=?, options=? WHERE id=?`,
			quoteIdent(d.TablePrefix()+"yaegi_scripts"),
		), script.CollectionName, script.Name, script.HookPoint, script.Content, script.APIPath, script.HTTPMethod, script.Enabled, script.Priority, opts, script.Id)
		if err != nil {
			return NewSystemError(err)
		}
		return nil
	}
	res, err := d.db.Exec(ctx, fmt.Sprintf(
		`INSERT INTO %s (collection_name, name, hook_point, content, api_path, http_method, enabled, priority, options) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		quoteIdent(d.TablePrefix()+"yaegi_scripts"),
	), script.CollectionName, script.Name, script.HookPoint, script.Content, script.APIPath, script.HTTPMethod, script.Enabled, script.Priority, opts)
	if err != nil {
		return NewSystemError(err)
	}
	script.Id, _ = res.LastInsertId()
	return nil
}

func (d *Database) setScriptEnabled(ctx context.Context, id int64, enabled bool) error {
	if d.useSystemDAO() {
		_, err := dao.YaegiScripts.Ctx(ctx).Where(do.YaegiScripts{Id: id}).Data(do.YaegiScripts{
			Enabled: boolToTiny(enabled),
		}).Update()
		if err != nil {
			return NewSystemError(err)
		}
		return nil
	}
	_, err := d.db.Exec(ctx, fmt.Sprintf(
		`UPDATE %s SET enabled=? WHERE id=?`,
		quoteIdent(d.TablePrefix()+"yaegi_scripts"),
	), enabled, id)
	if err != nil {
		return NewSystemError(err)
	}
	return nil
}

func (d *Database) deleteScriptRow(ctx context.Context, id int64) error {
	if d.useSystemDAO() {
		_, err := dao.YaegiScripts.Ctx(ctx).Where(do.YaegiScripts{Id: id}).Delete()
		if err != nil {
			return NewSystemError(err)
		}
		return nil
	}
	_, err := d.db.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id=?`, quoteIdent(d.TablePrefix()+"yaegi_scripts")), id)
	if err != nil {
		return NewSystemError(err)
	}
	return nil
}

func boolToTiny(v bool) int {
	if v {
		return 1
	}
	return 0
}

func entityScriptToModel(row *entity.YaegiScripts) *modelmd.YaegiScript {
	return &modelmd.YaegiScript{
		Id:             row.Id,
		CollectionName: row.CollectionName,
		Name:           row.Name,
		HookPoint:      row.HookPoint,
		Content:        row.Content,
		APIPath:        row.ApiPath,
		HTTPMethod:     row.HttpMethod,
		Enabled:        row.Enabled != 0,
		Priority:       row.Priority,
		Options:        row.Options,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

type scriptScanner interface {
	Scan(dest ...any) error
}

func scanScriptRow(sc scriptScanner) (*modelmd.YaegiScript, error) {
	var (
		id             int64
		collectionName sql.NullString
		name           string
		hookPoint      string
		content        string
		apiPath        sql.NullString
		httpMethod     sql.NullString
		enabled        bool
		priority       int
		options        sql.NullString
		createdAt      sql.NullTime
		updatedAt      sql.NullTime
	)
	if err := sc.Scan(&id, &collectionName, &name, &hookPoint, &content, &apiPath, &httpMethod, &enabled, &priority, &options, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	s := &modelmd.YaegiScript{
		Id:        id,
		Name:      name,
		HookPoint: hookPoint,
		Content:   content,
		Enabled:   enabled,
		Priority:  priority,
	}
	if collectionName.Valid {
		s.CollectionName = collectionName.String
	}
	if apiPath.Valid {
		s.APIPath = apiPath.String
	}
	if httpMethod.Valid {
		s.HTTPMethod = httpMethod.String
	}
	if options.Valid {
		s.Options = options.String
	}
	if createdAt.Valid {
		s.CreatedAt = gtime.New(createdAt.Time)
	}
	if updatedAt.Valid {
		s.UpdatedAt = gtime.New(updatedAt.Time)
	}
	return s, nil
}

func optionsJSONString(v any) string {
	if v == nil {
		return "{}"
	}
	if s, ok := v.(string); ok {
		if s == "" {
			return "{}"
		}
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
