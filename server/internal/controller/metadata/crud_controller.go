package metadata

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"

	md "itcodex/server/internal/service/metadata"
	resourcemgr "itcodex/server/internal/service/resource"
)

type CRUDController struct {
	db        *md.Database
	resources *resourcemgr.Manager
}

func NewCRUDController(db *md.Database, managers ...*resourcemgr.Manager) *CRUDController {
	var manager *resourcemgr.Manager
	if len(managers) > 0 {
		manager = managers[0]
	}
	if manager == nil {
		manager = resourcemgr.NewManager()
		manager.RegisterMetadataCRUD(resourcemgr.DatabaseResolver{Database: db})
	}
	return &CRUDController{db: db, resources: manager}
}

func (cc *CRUDController) dispatch(r *ghttp.Request, action string, params map[string]any, data, options any) (any, bool) {
	resourceName := r.GetRouter("collection").String()
	if resourceName == "" {
		resourceName = r.Get("collection").String()
	}
	result, err := cc.resources.Dispatch(r.Context(), &resourcemgr.ActionRequest{
		Resource: resourceName,
		Action:   action,
		Params:   params,
		Data:     data,
		Options:  options,
	})
	if err != nil {
		writeLogicError(r, err)
		return nil, false
	}
	return result, true
}

func (cc *CRUDController) repo(r *ghttp.Request) (md.Repository, bool) {
	name := r.GetRouter("collection").String()
	if name == "" {
		name = r.Get("collection").String()
	}
	if name == "" {
		writeFail(r, http.StatusBadRequest, 1, "collection 不能为空", nil)
		return nil, false
	}
	coll := cc.db.Collection(name)
	if coll == nil {
		writeFail(r, http.StatusNotFound, 404, "集合不存在: "+name, nil)
		return nil, false
	}
	return coll.Repository(), true
}

func parseID(raw string) any {
	if raw == "" {
		return nil
	}
	if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return v
	}
	return raw
}

func parseJSONBody(raw []byte) (any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var body any
	if err := dec.Decode(&body); err != nil {
		return nil, err
	}
	return normalizeJSONNumbers(body), nil
}

func normalizeJSONNumbers(v any) any {
	switch val := v.(type) {
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return i
		}
		if f, err := val.Float64(); err == nil {
			return f
		}
		return val.String()
	case map[string]any:
		for k, vv := range val {
			val[k] = normalizeJSONNumbers(vv)
		}
		return val
	case []any:
		for i, vv := range val {
			val[i] = normalizeJSONNumbers(vv)
		}
		return val
	default:
		return v
	}
}

func parseFilter(r *ghttp.Request) (md.Filter, bool) {
	filterStr := r.GetQuery("filter").String()
	if filterStr == "" {
		return nil, true
	}
	dec := json.NewDecoder(bytes.NewReader([]byte(filterStr)))
	dec.UseNumber()
	var raw any
	if err := dec.Decode(&raw); err != nil {
		writeFail(r, http.StatusBadRequest, 1, "filter 参数必须是有效的JSON", nil)
		return nil, false
	}
	raw = normalizeJSONNumbers(raw)
	m, ok := raw.(map[string]any)
	if !ok {
		writeFail(r, http.StatusBadRequest, 1, "filter 参数必须是 JSON 对象", nil)
		return nil, false
	}
	return md.Filter(m), true
}

func applyQuerySelection(r *ghttp.Request, opts *md.CommonOptions) {
	if fields := r.GetQuery("fields").String(); fields != "" {
		opts.Fields = splitAndTrim(fields)
	}
	if except := r.GetQuery("except").String(); except != "" {
		opts.Except = splitAndTrim(except)
	}
	if sortStr := r.GetQuery("sort").String(); sortStr != "" {
		var sort md.Sort
		for _, p := range strings.Split(sortStr, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				sort = append(sort, p)
			}
		}
		opts.Sort = sort
	}
	if appends := r.GetQuery("appends").String(); appends != "" {
		opts.Appends = splitAndTrim(appends)
	}
}

func (cc *CRUDController) List(r *ghttp.Request) {
	_, ok := cc.repo(r)
	if !ok {
		return
	}
	opts := &md.FindOptions{}
	filter, ok := parseFilter(r)
	if !ok {
		return
	}
	opts.Filter = filter
	applyQuerySelection(r, &opts.CommonOptions)
	applySpecialCollectionQuery(r, cc.db, &opts.CommonOptions)

	page := 1
	if p, err := strconv.Atoi(r.GetQuery("page").String()); err == nil && p > 0 {
		page = p
	}
	pageSize := 20
	if ps, err := strconv.Atoi(r.GetQuery("pageSize").String()); err == nil && ps > 0 {
		pageSize = ps
	}
	if limit, err := strconv.Atoi(r.GetQuery("limit").String()); err == nil && limit > 0 {
		pageSize = limit
		opts.Limit = limit
	}
	if offset, err := strconv.Atoi(r.GetQuery("offset").String()); err == nil && offset >= 0 {
		opts.Offset = offset
	}
	opts.Page = page
	opts.PageSize = pageSize

	result, ok := cc.dispatch(r, resourcemgr.ActionList, nil, nil, opts)
	if !ok {
		return
	}
	listResult, ok := result.(resourcemgr.CollectionListResult)
	if !ok {
		writeFail(r, http.StatusInternalServerError, 1, "资源列表结果类型错误", nil)
		return
	}
	totalPages := 0
	if listResult.Total > 0 && pageSize > 0 {
		totalPages = (listResult.Total + pageSize - 1) / pageSize
	}
	writeOK(r, map[string]any{
		"list": listResult.List, "total": listResult.Total, "page": page, "pageSize": pageSize, "totalPages": totalPages,
	})
}

func (cc *CRUDController) Count(r *ghttp.Request) {
	_, ok := cc.repo(r)
	if !ok {
		return
	}
	opts := &md.CountOptions{}
	filter, ok := parseFilter(r)
	if !ok {
		return
	}
	opts.Filter = filter
	result, ok := cc.dispatch(r, resourcemgr.ActionCount, nil, nil, opts)
	if !ok {
		return
	}
	count, ok := result.(int)
	if !ok {
		writeFail(r, http.StatusInternalServerError, 1, "资源计数结果类型错误", nil)
		return
	}
	writeOK(r, map[string]any{"count": count})
}

func (cc *CRUDController) Get(r *ghttp.Request) {
	_, ok := cc.repo(r)
	if !ok {
		return
	}
	opts := &md.FindOneOptions{FilterByTk: parseID(r.GetRouter("id").String())}
	applyQuerySelection(r, &opts.CommonOptions)
	result, ok := cc.dispatch(r, resourcemgr.ActionGet, map[string]any{
		"id": parseID(r.GetRouter("id").String()),
	}, nil, opts)
	if !ok {
		return
	}
	writeOK(r, result)
}

func (cc *CRUDController) Create(r *ghttp.Request) {
	_, ok := cc.repo(r)
	if !ok {
		return
	}
	var values map[string]any
	if err := json.Unmarshal(r.GetBody(), &values); err != nil {
		writeFail(r, http.StatusBadRequest, 1, "请求体解析失败: "+err.Error(), nil)
		return
	}
	result, ok := cc.dispatch(r, resourcemgr.ActionCreate, nil, values, &md.CreateOptions{})
	if !ok {
		return
	}
	writeCreated(r, result)
}

func (cc *CRUDController) CreateMany(r *ghttp.Request) {
	_, ok := cc.repo(r)
	if !ok {
		return
	}
	var records []map[string]any
	if err := json.Unmarshal(r.GetBody(), &records); err != nil {
		writeFail(r, http.StatusBadRequest, 1, "请求体必须是对象数组", nil)
		return
	}
	result, ok := cc.dispatch(r, resourcemgr.ActionCreateMany, nil, records, &md.CreateManyOptions{})
	if !ok {
		return
	}
	writeCreated(r, result)
}

func (cc *CRUDController) Update(r *ghttp.Request) {
	_, ok := cc.repo(r)
	if !ok {
		return
	}
	var values map[string]any
	if err := json.Unmarshal(r.GetBody(), &values); err != nil {
		writeFail(r, http.StatusBadRequest, 1, "请求体解析失败: "+err.Error(), nil)
		return
	}
	opts := &md.UpdateOptions{FilterByTk: parseID(r.GetRouter("id").String()), Values: values}
	if wl := r.GetQuery("whitelist").String(); wl != "" {
		opts.Whitelist = splitAndTrim(wl)
	}
	if bl := r.GetQuery("blacklist").String(); bl != "" {
		opts.Blacklist = splitAndTrim(bl)
	}
	result, ok := cc.dispatch(r, resourcemgr.ActionUpdate, map[string]any{
		"id": opts.FilterByTk,
	}, values, opts)
	if !ok {
		return
	}
	mutation, ok := result.(resourcemgr.MutationResult)
	if !ok {
		writeFail(r, http.StatusInternalServerError, 1, "资源更新结果类型错误", nil)
		return
	}
	writeOK(r, map[string]any{"record": mutation.Record, "affected": mutation.Affected})
}

func (cc *CRUDController) UpdateMany(r *ghttp.Request) {
	_, ok := cc.repo(r)
	if !ok {
		return
	}
	filter, ok := parseFilter(r)
	if !ok {
		return
	}
	if len(filter) == 0 {
		writeFail(r, http.StatusBadRequest, 1, "批量更新必须提供 filter", nil)
		return
	}
	var values map[string]any
	if err := json.Unmarshal(r.GetBody(), &values); err != nil {
		writeFail(r, http.StatusBadRequest, 1, "请求体解析失败: "+err.Error(), nil)
		return
	}
	opts := &md.UpdateOptions{Values: values}
	opts.Filter = filter
	if wl := r.GetQuery("whitelist").String(); wl != "" {
		opts.Whitelist = splitAndTrim(wl)
	}
	if bl := r.GetQuery("blacklist").String(); bl != "" {
		opts.Blacklist = splitAndTrim(bl)
	}
	result, ok := cc.dispatch(r, resourcemgr.ActionUpdateMany, nil, values, opts)
	if !ok {
		return
	}
	mutation, ok := result.(resourcemgr.MutationResult)
	if !ok {
		writeFail(r, http.StatusInternalServerError, 1, "资源批量更新结果类型错误", nil)
		return
	}
	writeOK(r, map[string]any{"affected": mutation.Affected})
}

func (cc *CRUDController) Destroy(r *ghttp.Request) {
	_, ok := cc.repo(r)
	if !ok {
		return
	}
	id := parseID(r.GetRouter("id").String())
	result, ok := cc.dispatch(r, resourcemgr.ActionDestroy, map[string]any{"id": id}, nil, &md.DestroyOptions{FilterByTk: id})
	if !ok {
		return
	}
	mutation, ok := result.(resourcemgr.MutationResult)
	if !ok {
		writeFail(r, http.StatusInternalServerError, 1, "资源删除结果类型错误", nil)
		return
	}
	writeOK(r, map[string]any{"affected": mutation.Affected})
}

func (cc *CRUDController) DestroyMany(r *ghttp.Request) {
	_, ok := cc.repo(r)
	if !ok {
		return
	}
	filter, ok := parseFilter(r)
	if !ok {
		return
	}
	opts := &md.DestroyOptions{}
	opts.Filter = filter
	if len(filter) == 0 {
		if !cc.db.AllowTruncate() {
			writeFail(r, http.StatusForbidden, 403, "禁止无条件清空表，请提供 filter 或开启 allowTruncate", nil)
			return
		}
		opts.Truncate = true
	}
	result, ok := cc.dispatch(r, resourcemgr.ActionDestroyMany, nil, nil, opts)
	if !ok {
		return
	}
	mutation, ok := result.(resourcemgr.MutationResult)
	if !ok {
		writeFail(r, http.StatusInternalServerError, 1, "资源批量删除结果类型错误", nil)
		return
	}
	writeOK(r, map[string]any{"affected": mutation.Affected})
}

func (cc *CRUDController) AssociationList(r *ghttp.Request) {
	_, ok := cc.repo(r)
	if !ok {
		return
	}
	// Use router params only — query ?id= must not shadow path :id
	result, ok := cc.dispatch(r, resourcemgr.ActionAssociationList, map[string]any{
		"id":          parseID(r.GetRouter("id").String()),
		"association": r.GetRouter("association").String(),
	}, nil, nil)
	if !ok {
		return
	}
	writeOK(r, map[string]any{"list": result})
}

func (cc *CRUDController) AssociationAdd(r *ghttp.Request) {
	_, ok := cc.repo(r)
	if !ok {
		return
	}
	body, err := parseJSONBody(r.GetBody())
	if err != nil {
		writeFail(r, http.StatusBadRequest, 1, "invalid json body", nil)
		return
	}
	_, ok = cc.dispatch(r, resourcemgr.ActionAssociationAdd, map[string]any{
		"id":          parseID(r.GetRouter("id").String()),
		"association": r.GetRouter("association").String(),
	}, body, nil)
	if !ok {
		return
	}
	writeOK(r, map[string]any{"ok": true})
}

func (cc *CRUDController) AssociationSet(r *ghttp.Request) {
	_, ok := cc.repo(r)
	if !ok {
		return
	}
	body, err := parseJSONBody(r.GetBody())
	if err != nil {
		writeFail(r, http.StatusBadRequest, 1, "invalid json body", nil)
		return
	}
	_, ok = cc.dispatch(r, resourcemgr.ActionAssociationSet, map[string]any{
		"id":          parseID(r.GetRouter("id").String()),
		"association": r.GetRouter("association").String(),
	}, body, nil)
	if !ok {
		return
	}
	writeOK(r, map[string]any{"ok": true})
}

func (cc *CRUDController) AssociationRemove(r *ghttp.Request) {
	_, ok := cc.repo(r)
	if !ok {
		return
	}
	body, err := parseJSONBody(r.GetBody())
	if err != nil {
		writeFail(r, http.StatusBadRequest, 1, "invalid json body", nil)
		return
	}
	// DELETE body may be stripped; accept ?targetId= (not ?id=, which shadows path :id)
	if body == nil {
		if qid := r.GetQuery("targetId").String(); qid != "" {
			body = map[string]any{"id": parseID(qid)}
		} else if qid := r.GetQuery("fk").String(); qid != "" {
			body = map[string]any{"id": parseID(qid)}
		}
	}
	_, ok = cc.dispatch(r, resourcemgr.ActionAssociationRemove, map[string]any{
		"id":          parseID(r.GetRouter("id").String()),
		"association": r.GetRouter("association").String(),
	}, body, nil)
	if !ok {
		return
	}
	writeOK(r, map[string]any{"ok": true})
}

func applySpecialCollectionQuery(r *ghttp.Request, db *md.Database, opts *md.CommonOptions) {
	name := r.GetRouter("collection").String()
	if name == "" {
		name = r.Get("collection").String()
	}
	coll := db.Collection(name)
	if coll == nil {
		return
	}
	if opts.Filter == nil {
		opts.Filter = md.Filter{}
	}
	switch coll.Type() {
	case md.CollectionTypeCalendar:
		startField, endField := "start", "end"
		if v, ok := coll.Options()["calendarStartField"].(string); ok && v != "" {
			startField = v
		}
		if v, ok := coll.Options()["calendarEndField"].(string); ok && v != "" {
			endField = v
		}
		if start := r.GetQuery("start").String(); start != "" {
			opts.Filter[startField] = md.Filter{"$gte": start}
		}
		if end := r.GetQuery("end").String(); end != "" {
			opts.Filter[endField] = md.Filter{"$lte": end}
		}
	case md.CollectionTypeComment:
		fk := "target_id"
		if v, ok := coll.Options()["commentForeignKey"].(string); ok && v != "" {
			fk = v
		}
		if targetID := r.GetQuery("targetId").String(); targetID != "" {
			opts.Filter[fk] = parseID(targetID)
		}
	}
}

func (cc *CRUDController) Upload(r *ghttp.Request) {
	name := r.GetRouter("collection").String()
	if _, ok := cc.dispatch(r, resourcemgr.ActionUpload, nil, nil, nil); !ok {
		return
	}
	file := r.GetUploadFile("file")
	if file == nil {
		writeFail(r, http.StatusBadRequest, 1, "缺少 file 字段", nil)
		return
	}
	src, err := file.Open()
	if err != nil {
		writeFail(r, http.StatusBadRequest, 1, "无法读取上传文件", nil)
		return
	}
	defer src.Close()
	rec, err := cc.db.SaveFileObject(r.Context(), name, file.Filename, file.Header.Get("Content-Type"), src)
	if err != nil {
		writeLogicError(r, err)
		return
	}
	writeCreated(r, rec)
}

func (cc *CRUDController) FileContent(r *ghttp.Request) {
	if _, ok := cc.dispatch(r, resourcemgr.ActionFileGet, map[string]any{
		"id": parseID(r.GetRouter("id").String()),
	}, nil, nil); !ok {
		return
	}
	abs, mimeType, downloadName, err := cc.db.OpenFileObject(r.Context(), r.GetRouter("collection").String(), parseID(r.GetRouter("id").String()))
	if err != nil {
		writeLogicError(r, err)
		return
	}
	if mimeType != "" {
		r.Response.Header().Set("Content-Type", mimeType)
	}
	if downloadName != "" {
		r.Response.Header().Set("Content-Disposition", `inline; filename="`+downloadName+`"`)
	}
	r.Response.ServeFile(abs)
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
