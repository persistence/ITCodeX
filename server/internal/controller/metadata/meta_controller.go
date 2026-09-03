package metadata

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gtime"

	md "itcodex/server/internal/logic/metadata"
	modelmd "itcodex/server/internal/model/metadata"
)

type MetaController struct {
	db *md.Database
}

func NewMetaController(db *md.Database) *MetaController {
	return &MetaController{db: db}
}

type collectionResponse struct {
	Name        string                 `json:"name"`
	DisplayName string                 `json:"displayName"`
	Type        string                 `json:"type"`
	Options     map[string]interface{} `json:"options"`
}

type fieldResponse struct {
	Name        string                 `json:"name"`
	DisplayName string                 `json:"displayName"`
	Type        string                 `json:"type"`
	Required    bool                   `json:"required"`
	Unique      bool                   `json:"unique"`
	Indexed     bool                   `json:"indexed"`
	IsSystem    bool                   `json:"isSystem"`
	Options     map[string]interface{} `json:"options"`
}

func collToResponse(c *md.Collection) collectionResponse {
	return collectionResponse{
		Name:        c.Name(),
		DisplayName: c.DisplayName(),
		Type:        string(c.Type()),
		Options:     c.Options(),
	}
}

func fieldToResponse(f md.Field) fieldResponse {
	return fieldResponse{
		Name:        f.Name(),
		DisplayName: f.DisplayName(),
		Type:        string(f.Type()),
		Required:    f.IsRequired(),
		Unique:      f.IsUnique(),
		Indexed:     f.IsIndexed(),
		IsSystem:    f.IsSystem(),
		Options:     f.Options(),
	}
}

func handleError(r *ghttp.Request, err error) {
	switch e := err.(type) {
	case *md.ValidationError:
		r.Response.WriteHeader(http.StatusUnprocessableEntity)
		r.Response.WriteJson(g.Map{"code": 422, "message": "验证失败", "errors": e})
	case *md.NotFoundError:
		r.Response.WriteHeader(http.StatusNotFound)
		r.Response.WriteJson(g.Map{"code": 1, "message": err.Error()})
	case *md.AlreadyExistsError:
		r.Response.WriteHeader(http.StatusConflict)
		r.Response.WriteJson(g.Map{"code": 1, "message": err.Error()})
	case *md.ForbiddenError:
		r.Response.WriteHeader(http.StatusForbidden)
		r.Response.WriteJson(g.Map{"code": 1, "message": err.Error()})
	default:
		r.Response.WriteHeader(http.StatusInternalServerError)
		r.Response.WriteJson(g.Map{"code": 1, "message": err.Error()})
	}
}

func (mc *MetaController) Collections(r *ghttp.Request) {
	colls := mc.db.Collections()
	resp := make([]collectionResponse, 0, len(colls))
	for _, c := range colls {
		resp = append(resp, collToResponse(c))
	}
	r.Response.WriteJson(g.Map{"code": 0, "data": resp})
}

func (mc *MetaController) CreateCollection(r *ghttp.Request) {
	var input md.CreateCollectionInput
	if err := r.Parse(&input); err != nil {
		r.Response.WriteHeader(http.StatusBadRequest)
		r.Response.WriteJson(g.Map{"code": 1, "message": err.Error()})
		return
	}

	coll, err := mc.db.CreateCollection(r.Context(), input)
	if err != nil {
		handleError(r, err)
		return
	}

	r.Response.WriteJson(g.Map{"code": 0, "data": collToResponse(coll)})
}

func (mc *MetaController) GetCollection(r *ghttp.Request) {
	name := r.Get("collectionName").String()
	if name == "" {
		r.Response.WriteHeader(http.StatusBadRequest)
		r.Response.WriteJson(g.Map{"code": 1, "message": "collectionName 不能为空"})
		return
	}

	coll := mc.db.Collection(name)
	if coll == nil {
		r.Response.WriteHeader(http.StatusNotFound)
		r.Response.WriteJson(g.Map{"code": 1, "message": fmt.Sprintf("集合 %s 不存在", name)})
		return
	}

	r.Response.WriteJson(g.Map{"code": 0, "data": collToResponse(coll)})
}

func (mc *MetaController) DropCollection(r *ghttp.Request) {
	name := r.Get("collectionName").String()
	if name == "" {
		r.Response.WriteHeader(http.StatusBadRequest)
		r.Response.WriteJson(g.Map{"code": 1, "message": "collectionName 不能为空"})
		return
	}

	if err := mc.db.DropCollection(r.Context(), name); err != nil {
		handleError(r, err)
		return
	}

	r.Response.WriteJson(g.Map{"code": 0, "data": nil})
}

func (mc *MetaController) Fields(r *ghttp.Request) {
	name := r.Get("collectionName").String()
	if name == "" {
		r.Response.WriteHeader(http.StatusBadRequest)
		r.Response.WriteJson(g.Map{"code": 1, "message": "collectionName 不能为空"})
		return
	}

	coll := mc.db.Collection(name)
	if coll == nil {
		r.Response.WriteHeader(http.StatusNotFound)
		r.Response.WriteJson(g.Map{"code": 1, "message": fmt.Sprintf("集合 %s 不存在", name)})
		return
	}

	fields := coll.Fields()
	resp := make([]fieldResponse, 0, len(fields))
	for _, f := range fields {
		resp = append(resp, fieldToResponse(f))
	}

	r.Response.WriteJson(g.Map{"code": 0, "data": resp})
}

func (mc *MetaController) AddField(r *ghttp.Request) {
	name := r.Get("collectionName").String()
	if name == "" {
		r.Response.WriteHeader(http.StatusBadRequest)
		r.Response.WriteJson(g.Map{"code": 1, "message": "collectionName 不能为空"})
		return
	}

	coll := mc.db.Collection(name)
	if coll == nil {
		r.Response.WriteHeader(http.StatusNotFound)
		r.Response.WriteJson(g.Map{"code": 1, "message": fmt.Sprintf("集合 %s 不存在", name)})
		return
	}

	var input md.CreateFieldInput
	if err := r.Parse(&input); err != nil {
		r.Response.WriteHeader(http.StatusBadRequest)
		r.Response.WriteJson(g.Map{"code": 1, "message": err.Error()})
		return
	}

	if err := coll.AddField(r.Context(), input); err != nil {
		handleError(r, err)
		return
	}

	fields := coll.Fields()
	resp := make([]fieldResponse, 0, len(fields))
	for _, f := range fields {
		resp = append(resp, fieldToResponse(f))
	}

	r.Response.WriteJson(g.Map{"code": 0, "data": resp})
}

func (mc *MetaController) RemoveField(r *ghttp.Request) {
	name := r.Get("collectionName").String()
	fieldName := r.Get("fieldName").String()
	if name == "" || fieldName == "" {
		r.Response.WriteHeader(http.StatusBadRequest)
		r.Response.WriteJson(g.Map{"code": 1, "message": "collectionName 和 fieldName 不能为空"})
		return
	}

	coll := mc.db.Collection(name)
	if coll == nil {
		r.Response.WriteHeader(http.StatusNotFound)
		r.Response.WriteJson(g.Map{"code": 1, "message": fmt.Sprintf("集合 %s 不存在", name)})
		return
	}

	if err := coll.RemoveField(r.Context(), fieldName); err != nil {
		handleError(r, err)
		return
	}

	fields := coll.Fields()
	resp := make([]fieldResponse, 0, len(fields))
	for _, f := range fields {
		resp = append(resp, fieldToResponse(f))
	}

	r.Response.WriteJson(g.Map{"code": 0, "data": resp})
}

func (mc *MetaController) Scripts(r *ghttp.Request) {
	ctx := r.Context()
	prefix := mc.db.TablePrefix()
	query := fmt.Sprintf(`SELECT id, collection_name, name, hook_point, content, api_path, http_method, enabled, priority, options, created_at, updated_at FROM "%s_yaegi_scripts" ORDER BY id DESC`, prefix)
	rows, err := mc.db.SqlDB().QueryContext(ctx, query)
	if err != nil {
		r.Response.WriteHeader(http.StatusInternalServerError)
		r.Response.WriteJson(g.Map{"code": 1, "message": err.Error()})
		return
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
		if err := rows.Scan(&id, &collectionName, &name, &hookPoint, &content, &apiPath, &httpMethod, &enabled, &priority, &options, &createdAt, &updatedAt); err != nil {
			r.Response.WriteHeader(http.StatusInternalServerError)
			r.Response.WriteJson(g.Map{"code": 1, "message": err.Error()})
			return
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
		scripts = append(scripts, s)
	}

	r.Response.WriteJson(g.Map{"code": 0, "data": scripts})
}

func (mc *MetaController) LoadScript(r *ghttp.Request) {
	ctx := r.Context()
	var script modelmd.YaegiScript
	if err := r.Parse(&script); err != nil {
		r.Response.WriteHeader(http.StatusBadRequest)
		r.Response.WriteJson(g.Map{"code": 1, "message": err.Error()})
		return
	}

	now := gtime.Now()
	script.CreatedAt = now
	script.UpdatedAt = now
	if script.Priority == 0 {
		script.Priority = 0
	}
	if !script.Enabled {
		script.Enabled = true
	}

	prefix := mc.db.TablePrefix()
	var result sql.Result
	var err error
	if script.Id > 0 {
		query := fmt.Sprintf(`UPDATE "%s_yaegi_scripts" SET collection_name=?, name=?, hook_point=?, content=?, api_path=?, http_method=?, enabled=?, priority=?, options=?, updated_at=? WHERE id=?`, prefix)
		result, err = mc.db.SqlDB().ExecContext(ctx, query, script.CollectionName, script.Name, script.HookPoint, script.Content, script.APIPath, script.HTTPMethod, script.Enabled, script.Priority, script.Options, now, script.Id)
	} else {
		query := fmt.Sprintf(`INSERT INTO "%s_yaegi_scripts" (collection_name, name, hook_point, content, api_path, http_method, enabled, priority, options, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, prefix)
		result, err = mc.db.SqlDB().ExecContext(ctx, query, script.CollectionName, script.Name, script.HookPoint, script.Content, script.APIPath, script.HTTPMethod, script.Enabled, script.Priority, script.Options, now, now)
		if err == nil {
			script.Id, _ = result.LastInsertId()
		}
	}

	if err != nil {
		r.Response.WriteHeader(http.StatusInternalServerError)
		r.Response.WriteJson(g.Map{"code": 1, "message": err.Error()})
		return
	}

	if err := mc.db.Yaegi().LoadScript(&script); err != nil {
		r.Response.WriteHeader(http.StatusBadRequest)
		r.Response.WriteJson(g.Map{"code": 1, "message": err.Error()})
		return
	}

	r.Response.WriteJson(g.Map{"code": 0, "data": script})
}

func (mc *MetaController) DisableScript(r *ghttp.Request) {
	ctx := r.Context()
	idStr := r.Get("id").String()
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		r.Response.WriteHeader(http.StatusBadRequest)
		r.Response.WriteJson(g.Map{"code": 1, "message": "无效的脚本ID"})
		return
	}

	prefix := mc.db.TablePrefix()
	now := gtime.Now()
	query := fmt.Sprintf(`UPDATE "%s_yaegi_scripts" SET enabled=0, updated_at=? WHERE id=?`, prefix)
	if _, err := mc.db.SqlDB().ExecContext(ctx, query, now, id); err != nil {
		r.Response.WriteHeader(http.StatusInternalServerError)
		r.Response.WriteJson(g.Map{"code": 1, "message": err.Error()})
		return
	}

	if err := mc.db.Yaegi().DisableScript(id); err != nil {
		r.Response.WriteHeader(http.StatusInternalServerError)
		r.Response.WriteJson(g.Map{"code": 1, "message": err.Error()})
		return
	}

	r.Response.WriteJson(g.Map{"code": 0, "data": nil})
}

var _ context.Context
