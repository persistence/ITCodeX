package metadata

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	md "itcodex/server/internal/logic/metadata"
)

type CRUDController struct {
	db *md.Database
}

func NewCRUDController(db *md.Database) *CRUDController {
	return &CRUDController{db: db}
}

func (cc *CRUDController) Handle(r *ghttp.Request) {
	action := r.Get("action").String()
	action = strings.TrimPrefix(action, "/")
	parts := strings.Split(action, "/")

	if len(parts) < 1 || parts[0] == "" {
		r.Response.WriteHeader(http.StatusNotFound)
		r.Response.WriteJson(g.Map{"code": 1, "message": "路由不存在"})
		return
	}

	collName := parts[0]
	var id string
	if len(parts) > 1 {
		id = parts[1]
	}

	coll := cc.db.Collection(collName)
	if coll == nil {
		r.Response.WriteHeader(http.StatusNotFound)
		r.Response.WriteJson(g.Map{"code": 1, "message": "集合不存在: " + collName})
		return
	}

	repo := coll.Repository()
	method := r.Method
	// Normalize numeric IDs (Snowflake int64) so that SQL comparison works correctly
	var idVal interface{}
	if id != "" {
		if v, err := strconv.ParseInt(id, 10, 64); err == nil {
			idVal = v
		} else {
			idVal = id
		}
	}

	switch method {
	case http.MethodGet:
		if id != "" {
			cc.handleGet(r, repo, idVal)
		} else {
			cc.handleList(r, repo)
		}
	case http.MethodPost:
		cc.handleCreate(r, repo)
	case http.MethodPut, http.MethodPatch:
		if id == "" {
			r.Response.WriteHeader(http.StatusBadRequest)
			r.Response.WriteJson(g.Map{"code": 1, "message": "更新操作需要ID"})
			return
		}
		cc.handleUpdate(r, repo, idVal)
	case http.MethodDelete:
		if id != "" {
			cc.handleDestroy(r, repo, idVal)
		} else {
			cc.handleBulkDestroy(r, repo)
		}
	default:
		r.Response.WriteHeader(http.StatusMethodNotAllowed)
		r.Response.WriteJson(g.Map{"code": 1, "message": "方法不允许"})
	}
}

func (cc *CRUDController) handleGet(r *ghttp.Request, repo md.Repository, idVal interface{}) {
	ctx := r.Context()
	opts := &md.FindOneOptions{}
	opts.FilterByTk = idVal
	if fields := r.GetQuery("fields").String(); fields != "" {
		opts.Fields = splitAndTrim(fields)
	}
	if except := r.GetQuery("except").String(); except != "" {
		opts.Except = splitAndTrim(except)
	}

	record, err := repo.FindOne(ctx, opts)
	if err != nil {
		handleCrudError(r, err)
		return
	}

	r.Response.WriteJson(g.Map{"code": 0, "data": record.Data()})
}

func (cc *CRUDController) handleList(r *ghttp.Request, repo md.Repository) {
	ctx := r.Context()
	opts := &md.FindOptions{}

	if filterStr := r.GetQuery("filter").String(); filterStr != "" {
		var f md.Filter
		if err := json.Unmarshal([]byte(filterStr), &f); err != nil {
			r.Response.WriteHeader(http.StatusBadRequest)
			r.Response.WriteJson(g.Map{"code": 1, "message": "filter 参数必须是有效的JSON"})
			return
		}
		opts.Filter = f
	}

	page := 1
	if p := r.GetQuery("page").String(); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	pageSize := 20
	if ps := r.GetQuery("pageSize").String(); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}
	opts.Page = page
	opts.PageSize = pageSize
	opts.Offset = (page - 1) * pageSize
	opts.Limit = pageSize

	if sortStr := r.GetQuery("sort").String(); sortStr != "" {
		parts := strings.Split(sortStr, ",")
		var sort md.Sort
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if strings.HasPrefix(p, "-") {
				sort = append(sort, "-"+strings.TrimPrefix(p, "-"))
			} else {
				sort = append(sort, p)
			}
		}
		opts.Sort = sort
	}

	if fields := r.GetQuery("fields").String(); fields != "" {
		opts.Fields = splitAndTrim(fields)
	}
	if except := r.GetQuery("except").String(); except != "" {
		opts.Except = splitAndTrim(except)
	}

	records, total, err := repo.FindAndCount(ctx, opts)
	if err != nil {
		handleCrudError(r, err)
		return
	}

	list := make([]map[string]interface{}, 0, len(records))
	for _, rec := range records {
		list = append(list, rec.Data())
	}

	totalPages := 0
	if total > 0 && pageSize > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}

	r.Response.WriteJson(g.Map{
		"code": 0,
		"data": g.Map{
			"list":       list,
			"total":      total,
			"page":       page,
			"pageSize":   pageSize,
			"totalPages": totalPages,
		},
	})
}

func (cc *CRUDController) handleCreate(r *ghttp.Request, repo md.Repository) {
	ctx := r.Context()
	var values map[string]interface{}
	if err := json.Unmarshal(r.GetBody(), &values); err != nil {
		r.Response.WriteHeader(http.StatusBadRequest)
		r.Response.WriteJson(g.Map{"code": 1, "message": "请求体解析失败: " + err.Error()})
		return
	}
	if values == nil {
		values = make(map[string]interface{})
	}

	opts := &md.CreateOptions{Values: values}
	record, err := repo.Create(ctx, opts)
	if err != nil {
		handleCrudError(r, err)
		return
	}

	r.Response.WriteHeader(http.StatusCreated)
	r.Response.WriteJson(g.Map{"code": 0, "data": record.Data()})
}

func (cc *CRUDController) handleUpdate(r *ghttp.Request, repo md.Repository, idVal interface{}) {
	ctx := r.Context()
	var values map[string]interface{}
	if err := json.Unmarshal(r.GetBody(), &values); err != nil {
		r.Response.WriteHeader(http.StatusBadRequest)
		r.Response.WriteJson(g.Map{"code": 1, "message": "请求体解析失败: " + err.Error()})
		return
	}
	if values == nil {
		values = make(map[string]interface{})
	}

	opts := &md.UpdateOptions{
		FilterByTk: idVal,
		Values:     values,
	}
	record, affected, err := repo.Update(ctx, opts)
	if err != nil {
		handleCrudError(r, err)
		return
	}

	r.Response.WriteJson(g.Map{
		"code":     0,
		"data":     record.Data(),
		"affected": affected,
	})
}

func (cc *CRUDController) handleDestroy(r *ghttp.Request, repo md.Repository, idVal interface{}) {
	ctx := r.Context()
	opts := &md.DestroyOptions{FilterByTk: idVal}
	affected, err := repo.Destroy(ctx, opts)
	if err != nil {
		handleCrudError(r, err)
		return
	}

	r.Response.WriteJson(g.Map{"code": 0, "data": nil, "affected": affected})
}

func (cc *CRUDController) handleBulkDestroy(r *ghttp.Request, repo md.Repository) {
	ctx := r.Context()
	opts := &md.DestroyOptions{}
	if filterStr := r.GetQuery("filter").String(); filterStr != "" {
		var f md.Filter
		if err := json.Unmarshal([]byte(filterStr), &f); err != nil {
			r.Response.WriteHeader(http.StatusBadRequest)
			r.Response.WriteJson(g.Map{"code": 1, "message": "filter 参数必须是有效的JSON"})
			return
		}
		opts.Filter = f
	}

	affected, err := repo.Destroy(ctx, opts)
	if err != nil {
		handleCrudError(r, err)
		return
	}

	r.Response.WriteJson(g.Map{"code": 0, "data": nil, "affected": affected})
}

func handleCrudError(r *ghttp.Request, err error) {
	switch e := err.(type) {
	case *md.ValidationError:
		r.Response.WriteHeader(http.StatusUnprocessableEntity)
		r.Response.WriteJson(g.Map{"code": 422, "message": "验证失败", "errors": e})
	case *md.NotFoundError:
		r.Response.WriteHeader(http.StatusNotFound)
		r.Response.WriteJson(g.Map{"code": 404, "message": err.Error()})
	default:
		r.Response.WriteHeader(http.StatusInternalServerError)
		r.Response.WriteJson(g.Map{"code": 1, "message": err.Error()})
	}
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
