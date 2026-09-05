package metadata

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"

	md "itcodex/server/internal/service/metadata"
)

type CRUDController struct {
	db *md.Database
}

func NewCRUDController(db *md.Database) *CRUDController {
	return &CRUDController{db: db}
}

func (cc *CRUDController) repo(r *ghttp.Request) (md.Repository, bool) {
	name := r.Get("collection").String()
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

func parseID(raw string) interface{} {
	if raw == "" {
		return nil
	}
	if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return v
	}
	return raw
}

func parseFilter(r *ghttp.Request) (md.Filter, bool) {
	filterStr := r.GetQuery("filter").String()
	if filterStr == "" {
		return nil, true
	}
	var f md.Filter
	if err := json.Unmarshal([]byte(filterStr), &f); err != nil {
		writeFail(r, http.StatusBadRequest, 1, "filter 参数必须是有效的JSON", nil)
		return nil, false
	}
	return f, true
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
}

func (cc *CRUDController) List(r *ghttp.Request) {
	repo, ok := cc.repo(r)
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

	records, total, err := repo.FindAndCount(r.Context(), opts)
	if err != nil {
		writeLogicError(r, err)
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
	writeOK(r, map[string]interface{}{
		"list": list, "total": total, "page": page, "pageSize": pageSize, "totalPages": totalPages,
	})
}

func (cc *CRUDController) Count(r *ghttp.Request) {
	repo, ok := cc.repo(r)
	if !ok {
		return
	}
	opts := &md.CountOptions{}
	filter, ok := parseFilter(r)
	if !ok {
		return
	}
	opts.Filter = filter
	n, err := repo.Count(r.Context(), opts)
	if err != nil {
		writeLogicError(r, err)
		return
	}
	writeOK(r, map[string]interface{}{"count": n})
}

func (cc *CRUDController) Get(r *ghttp.Request) {
	repo, ok := cc.repo(r)
	if !ok {
		return
	}
	opts := &md.FindOneOptions{FilterByTk: parseID(r.Get("id").String())}
	applyQuerySelection(r, &opts.CommonOptions)
	record, err := repo.FindOne(r.Context(), opts)
	if err != nil {
		writeLogicError(r, err)
		return
	}
	writeOK(r, record.Data())
}

func (cc *CRUDController) Create(r *ghttp.Request) {
	repo, ok := cc.repo(r)
	if !ok {
		return
	}
	var values map[string]interface{}
	if err := json.Unmarshal(r.GetBody(), &values); err != nil {
		writeFail(r, http.StatusBadRequest, 1, "请求体解析失败: "+err.Error(), nil)
		return
	}
	record, err := repo.Create(r.Context(), &md.CreateOptions{Values: values})
	if err != nil {
		writeLogicError(r, err)
		return
	}
	writeCreated(r, record.Data())
}

func (cc *CRUDController) CreateMany(r *ghttp.Request) {
	repo, ok := cc.repo(r)
	if !ok {
		return
	}
	var records []map[string]interface{}
	if err := json.Unmarshal(r.GetBody(), &records); err != nil {
		writeFail(r, http.StatusBadRequest, 1, "请求体必须是对象数组", nil)
		return
	}
	created, err := repo.CreateMany(r.Context(), &md.CreateManyOptions{Records: records})
	if err != nil {
		writeLogicError(r, err)
		return
	}
	list := make([]map[string]interface{}, 0, len(created))
	for _, rec := range created {
		list = append(list, rec.Data())
	}
	writeCreated(r, list)
}

func (cc *CRUDController) Update(r *ghttp.Request) {
	repo, ok := cc.repo(r)
	if !ok {
		return
	}
	var values map[string]interface{}
	if err := json.Unmarshal(r.GetBody(), &values); err != nil {
		writeFail(r, http.StatusBadRequest, 1, "请求体解析失败: "+err.Error(), nil)
		return
	}
	opts := &md.UpdateOptions{FilterByTk: parseID(r.Get("id").String()), Values: values}
	if wl := r.GetQuery("whitelist").String(); wl != "" {
		opts.Whitelist = splitAndTrim(wl)
	}
	if bl := r.GetQuery("blacklist").String(); bl != "" {
		opts.Blacklist = splitAndTrim(bl)
	}
	record, affected, err := repo.Update(r.Context(), opts)
	if err != nil {
		writeLogicError(r, err)
		return
	}
	data := interface{}(nil)
	if record != nil {
		data = record.Data()
	}
	writeOK(r, map[string]interface{}{"record": data, "affected": affected})
}

func (cc *CRUDController) UpdateMany(r *ghttp.Request) {
	repo, ok := cc.repo(r)
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
	var values map[string]interface{}
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
	_, affected, err := repo.Update(r.Context(), opts)
	if err != nil {
		writeLogicError(r, err)
		return
	}
	writeOK(r, map[string]interface{}{"affected": affected})
}

func (cc *CRUDController) Destroy(r *ghttp.Request) {
	repo, ok := cc.repo(r)
	if !ok {
		return
	}
	affected, err := repo.Destroy(r.Context(), &md.DestroyOptions{FilterByTk: parseID(r.Get("id").String())})
	if err != nil {
		writeLogicError(r, err)
		return
	}
	writeOK(r, map[string]interface{}{"affected": affected})
}

func (cc *CRUDController) DestroyMany(r *ghttp.Request) {
	repo, ok := cc.repo(r)
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
	affected, err := repo.Destroy(r.Context(), opts)
	if err != nil {
		writeLogicError(r, err)
		return
	}
	writeOK(r, map[string]interface{}{"affected": affected})
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
