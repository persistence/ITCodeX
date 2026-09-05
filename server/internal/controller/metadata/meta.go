package metadata

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/os/gtime"

	v1 "itcodex/server/api/metadata/v1"
	modelmd "itcodex/server/internal/model/metadata"
	md "itcodex/server/internal/service/metadata"
)

type ControllerV1 struct {
	db *md.Database
}

func NewV1(db *md.Database) *ControllerV1 {
	return &ControllerV1{db: db}
}

func (c *ControllerV1) requireCollection(name string) (*md.Collection, error) {
	coll := c.db.Collection(name)
	if coll == nil {
		return nil, wrapSvcErr(md.NewNotFoundError("集合", "name", name))
	}
	return coll, nil
}

func (c *ControllerV1) CollectionList(_ context.Context, req *v1.CollectionListReq) (res *v1.CollectionListRes, err error) {
	colls := c.db.Collections()
	items := make([]v1.CollectionItem, 0, len(colls))
	for _, coll := range colls {
		if req.Keyword != "" && !strings.Contains(coll.Name(), req.Keyword) && !strings.Contains(coll.DisplayName(), req.Keyword) {
			continue
		}
		if req.Category != "" && !collectionHasCategory(coll, req.Category) {
			continue
		}
		items = append(items, collToItem(coll, false))
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	total := len(items)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return &v1.CollectionListRes{List: items[start:end], Total: total}, nil
}

func (c *ControllerV1) CollectionCreate(ctx context.Context, req *v1.CollectionCreateReq) (res *v1.CollectionCreateRes, err error) {
	coll, err := c.db.CreateCollection(ctx, req.CreateCollectionInput)
	if err != nil {
		return nil, wrapSvcErr(err)
	}
	item := collToItem(coll, true)
	return &v1.CollectionCreateRes{CollectionItem: &item}, nil
}

func (c *ControllerV1) CollectionGet(_ context.Context, req *v1.CollectionGetReq) (res *v1.CollectionGetRes, err error) {
	coll, err := c.requireCollection(req.CollectionName)
	if err != nil {
		return nil, err
	}
	item := collToItem(coll, true)
	return &v1.CollectionGetRes{CollectionItem: &item}, nil
}

func (c *ControllerV1) CollectionUpdate(ctx context.Context, req *v1.CollectionUpdateReq) (res *v1.CollectionUpdateRes, err error) {
	coll, err := c.db.UpdateCollection(ctx, req.CollectionName, req.UpdateCollectionInput)
	if err != nil {
		return nil, wrapSvcErr(err)
	}
	item := collToItem(coll, true)
	return &v1.CollectionUpdateRes{CollectionItem: &item}, nil
}

func (c *ControllerV1) CollectionDelete(ctx context.Context, req *v1.CollectionDeleteReq) (res *v1.CollectionDeleteRes, err error) {
	if err = c.db.DropCollection(ctx, req.CollectionName); err != nil {
		return nil, wrapSvcErr(err)
	}
	return &v1.CollectionDeleteRes{}, nil
}

func (c *ControllerV1) CollectionSync(ctx context.Context, req *v1.CollectionSyncReq) (res *v1.CollectionSyncRes, err error) {
	coll, err := c.requireCollection(req.CollectionName)
	if err != nil {
		return nil, err
	}
	if err = coll.Sync(ctx); err != nil {
		return nil, wrapSvcErr(err)
	}
	item := collToItem(coll, false)
	return &v1.CollectionSyncRes{CollectionItem: &item}, nil
}

func (c *ControllerV1) FieldList(_ context.Context, req *v1.FieldListReq) (res *v1.FieldListRes, err error) {
	coll, err := c.requireCollection(req.CollectionName)
	if err != nil {
		return nil, err
	}
	return &v1.FieldListRes{List: fieldsToItems(coll)}, nil
}

func (c *ControllerV1) FieldCreate(ctx context.Context, req *v1.FieldCreateReq) (res *v1.FieldCreateRes, err error) {
	coll, err := c.requireCollection(req.CollectionName)
	if err != nil {
		return nil, err
	}
	if err = coll.AddField(ctx, req.CreateFieldInput); err != nil {
		return nil, wrapSvcErr(err)
	}
	return &v1.FieldCreateRes{List: fieldsToItems(coll)}, nil
}

func (c *ControllerV1) FieldUpdate(ctx context.Context, req *v1.FieldUpdateReq) (res *v1.FieldUpdateRes, err error) {
	coll, err := c.requireCollection(req.CollectionName)
	if err != nil {
		return nil, err
	}
	if err = coll.UpdateField(ctx, req.FieldName, req.UpdateFieldInput); err != nil {
		return nil, wrapSvcErr(err)
	}
	return &v1.FieldUpdateRes{List: fieldsToItems(coll)}, nil
}

func (c *ControllerV1) FieldDelete(ctx context.Context, req *v1.FieldDeleteReq) (res *v1.FieldDeleteRes, err error) {
	coll, err := c.requireCollection(req.CollectionName)
	if err != nil {
		return nil, err
	}
	if err = coll.RemoveField(ctx, req.FieldName); err != nil {
		return nil, wrapSvcErr(err)
	}
	return &v1.FieldDeleteRes{}, nil
}

func (c *ControllerV1) ScriptList(ctx context.Context, req *v1.ScriptListReq) (res *v1.ScriptListRes, err error) {
	prefix := c.db.TablePrefix()
	query := fmt.Sprintf(
		`SELECT id, collection_name, name, hook_point, content, api_path, http_method, enabled, priority, options, created_at, updated_at FROM %s ORDER BY id DESC`,
		md.QuoteIdent(prefix+"yaegi_scripts"),
	)
	rows, err := c.db.SqlDB().QueryContext(ctx, query)
	if err != nil {
		return nil, wrapSvcErr(err)
	}
	defer rows.Close()

	var scripts []*modelmd.YaegiScript
	for rows.Next() {
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
		if err = rows.Scan(&id, &collectionName, &name, &hookPoint, &content, &apiPath, &httpMethod, &enabled, &priority, &options, &createdAt, &updatedAt); err != nil {
			return nil, wrapSvcErr(err)
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
		if req.Collection != "" && s.CollectionName != req.Collection {
			continue
		}
		if req.Hook != "" && s.HookPoint != req.Hook {
			continue
		}
		scripts = append(scripts, s)
	}
	return &v1.ScriptListRes{List: scripts}, nil
}

func (c *ControllerV1) ScriptSave(ctx context.Context, req *v1.ScriptSaveReq) (res *v1.ScriptSaveRes, err error) {
	script := req.YaegiScript
	now := gtime.Now()
	script.UpdatedAt = now
	if script.CreatedAt == nil {
		script.CreatedAt = now
	}
	if !script.Enabled {
		script.Enabled = true
	}

	prefix := c.db.TablePrefix()
	var result sql.Result
	if script.Id > 0 {
		query := fmt.Sprintf(`UPDATE %s SET collection_name=?, name=?, hook_point=?, content=?, api_path=?, http_method=?, enabled=?, priority=?, options=?, updated_at=? WHERE id=?`, md.QuoteIdent(prefix+"yaegi_scripts"))
		result, err = c.db.SqlDB().ExecContext(ctx, query, script.CollectionName, script.Name, script.HookPoint, script.Content, script.APIPath, script.HTTPMethod, script.Enabled, script.Priority, script.Options, now, script.Id)
	} else {
		query := fmt.Sprintf(`INSERT INTO %s (collection_name, name, hook_point, content, api_path, http_method, enabled, priority, options, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, md.QuoteIdent(prefix+"yaegi_scripts"))
		result, err = c.db.SqlDB().ExecContext(ctx, query, script.CollectionName, script.Name, script.HookPoint, script.Content, script.APIPath, script.HTTPMethod, script.Enabled, script.Priority, script.Options, now, now)
		if err == nil {
			script.Id, _ = result.LastInsertId()
		}
	}
	if err != nil {
		return nil, wrapSvcErr(err)
	}
	if yaegi := c.db.Yaegi(); yaegi != nil {
		if err = yaegi.LoadScript(&script); err != nil {
			return nil, gerror.Wrap(err, "加载脚本失败")
		}
	}
	return &v1.ScriptSaveRes{YaegiScript: &script}, nil
}

func (c *ControllerV1) ScriptDisable(ctx context.Context, req *v1.ScriptDisableReq) (res *v1.ScriptDisableRes, err error) {
	prefix := c.db.TablePrefix()
	query := fmt.Sprintf(`UPDATE %s SET enabled=0, updated_at=? WHERE id=?`, md.QuoteIdent(prefix+"yaegi_scripts"))
	if _, err = c.db.SqlDB().ExecContext(ctx, query, gtime.Now(), req.Id); err != nil {
		return nil, wrapSvcErr(err)
	}
	if yaegi := c.db.Yaegi(); yaegi != nil {
		if err = yaegi.DisableScript(req.Id); err != nil {
			return nil, wrapSvcErr(err)
		}
	}
	return &v1.ScriptDisableRes{}, nil
}
