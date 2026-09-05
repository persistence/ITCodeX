package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cast"
)

// loadAppends fills association data onto records for requested append names.
func (r *GenericRepository) loadAppends(ctx context.Context, records []*Record, appends Appends) error {
	if len(records) == 0 || len(appends) == 0 {
		return nil
	}
	for _, name := range appends {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		// tree children special
		if name == "children" && r.coll.Type() == CollectionTypeTree {
			if err := r.loadTreeChildren(ctx, records); err != nil {
				return err
			}
			continue
		}
		f := r.coll.GetField(name)
		if f == nil {
			continue
		}
		switch FieldType(f.Type()) {
		case FieldTypeBelongsTo:
			if err := r.loadBelongsTo(ctx, records, f); err != nil {
				return err
			}
		case FieldTypeHasOne:
			if err := r.loadHasOne(ctx, records, f); err != nil {
				return err
			}
		case FieldTypeHasMany:
			if err := r.loadHasMany(ctx, records, f); err != nil {
				return err
			}
		case FieldTypeBelongsToMany:
			if err := r.loadBelongsToMany(ctx, records, f); err != nil {
				return err
			}
		case FieldTypeBelongsToManyArray:
			if err := r.loadBelongsToManyArray(ctx, records, f); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *GenericRepository) loadBelongsTo(ctx context.Context, records []*Record, f Field) error {
	ro := GetRelationOptions(f)
	target := r.coll.Db().Collection(ro.Target)
	if target == nil {
		return nil
	}
	fkCol := ro.ForeignKey
	if fkCol == "" {
		fkCol = f.Name()
	}
	ids := make([]interface{}, 0, len(records))
	idSet := map[string]bool{}
	for _, rec := range records {
		id := rec.Get(fkCol)
		if id == nil {
			continue
		}
		key := cast.ToString(id)
		if idSet[key] {
			continue
		}
		idSet[key] = true
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}
	related, err := target.Repository().Find(ctx, &FindOptions{
		CommonOptions: CommonOptions{Filter: Filter{ro.TargetKey: Filter{"$in": ids}}},
		PageSize:      MaxPageSize,
	})
	if err != nil {
		return err
	}
	byID := map[string]*Record{}
	for _, rel := range related {
		byID[cast.ToString(rel.Get(ro.TargetKey))] = rel
	}
	for _, rec := range records {
		id := cast.ToString(rec.Get(fkCol))
		if rel, ok := byID[id]; ok {
			rec.Set(f.Name(), rel.Data())
		} else {
			rec.Set(f.Name(), nil)
		}
	}
	return nil
}

func (r *GenericRepository) loadHasMany(ctx context.Context, records []*Record, f Field) error {
	ro := GetRelationOptions(f)
	target := r.coll.Db().Collection(ro.Target)
	if target == nil {
		return nil
	}
	ids := make([]interface{}, 0, len(records))
	for _, rec := range records {
		ids = append(ids, rec.Get(ro.SourceKey))
	}
	related, err := target.Repository().Find(ctx, &FindOptions{
		CommonOptions: CommonOptions{Filter: Filter{ro.ForeignKey: Filter{"$in": ids}}},
		PageSize:      MaxPageSize,
	})
	if err != nil {
		return err
	}
	grouped := map[string][]map[string]interface{}{}
	for _, rel := range related {
		key := cast.ToString(rel.Get(ro.ForeignKey))
		grouped[key] = append(grouped[key], rel.Data())
	}
	for _, rec := range records {
		key := cast.ToString(rec.Get(ro.SourceKey))
		if list, ok := grouped[key]; ok {
			rec.Set(f.Name(), list)
		} else {
			rec.Set(f.Name(), []map[string]interface{}{})
		}
	}
	return nil
}

func (r *GenericRepository) loadHasOne(ctx context.Context, records []*Record, f Field) error {
	if err := r.loadHasMany(ctx, records, f); err != nil {
		return err
	}
	for _, rec := range records {
		v := rec.Get(f.Name())
		if list, ok := v.([]map[string]interface{}); ok {
			if len(list) > 0 {
				rec.Set(f.Name(), list[0])
			} else {
				rec.Set(f.Name(), nil)
			}
		}
	}
	return nil
}

func (r *GenericRepository) loadBelongsToMany(ctx context.Context, records []*Record, f Field) error {
	ro := GetRelationOptions(f)
	target := r.coll.Db().Collection(ro.Target)
	if target == nil || ro.Through == "" {
		return nil
	}
	db := r.execDB()
	for _, rec := range records {
		srcID := rec.Get(ro.SourceKey)
		q := fmt.Sprintf(`SELECT %s FROM %s WHERE %s = ?`, quoteIdent(ro.OtherKey), quoteIdent(ro.Through), quoteIdent(ro.ForeignKey))
		rows, err := db.Query(ctx, q, srcID)
		if err != nil {
			return NewSystemError(err)
		}
		var targetIDs []interface{}
		for rows.Next() {
			var tid interface{}
			if err := rows.Scan(&tid); err != nil {
				rows.Close()
				return NewSystemError(err)
			}
			targetIDs = append(targetIDs, tid)
		}
		rows.Close()
		if len(targetIDs) == 0 {
			rec.Set(f.Name(), []map[string]interface{}{})
			continue
		}
		related, err := target.Repository().Find(ctx, &FindOptions{
			CommonOptions: CommonOptions{Filter: Filter{ro.TargetKey: Filter{"$in": targetIDs}}},
			PageSize:      MaxPageSize,
		})
		if err != nil {
			return err
		}
		list := make([]map[string]interface{}, 0, len(related))
		for _, rel := range related {
			list = append(list, rel.Data())
		}
		rec.Set(f.Name(), list)
	}
	return nil
}

func (r *GenericRepository) loadBelongsToManyArray(ctx context.Context, records []*Record, f Field) error {
	ro := GetRelationOptions(f)
	targetName := ro.Target
	if targetName == "" {
		if opts := f.Options(); opts != nil {
			targetName, _ = opts["target"].(string)
		}
	}
	target := r.coll.Db().Collection(targetName)
	if target == nil {
		return nil
	}
	for _, rec := range records {
		raw := rec.Get(f.Name())
		ids := toInterfaceSlice(raw)
		if len(ids) == 0 {
			rec.Set(f.Name()+"_items", []map[string]interface{}{})
			continue
		}
		related, err := target.Repository().Find(ctx, &FindOptions{
			CommonOptions: CommonOptions{Filter: Filter{DefaultPrimaryKey: Filter{"$in": ids}}},
			PageSize:      MaxPageSize,
		})
		if err != nil {
			return err
		}
		list := make([]map[string]interface{}, 0, len(related))
		for _, rel := range related {
			list = append(list, rel.Data())
		}
		rec.Set(f.Name()+"_items", list)
	}
	return nil
}

func (r *GenericRepository) loadTreeChildren(ctx context.Context, records []*Record) error {
	parentKey := "parent_id"
	if opts := r.coll.Options(); opts != nil {
		if v, ok := opts["treeParentKey"].(string); ok && v != "" {
			parentKey = v
		}
	}
	ids := make([]interface{}, 0, len(records))
	for _, rec := range records {
		ids = append(ids, rec.Get(DefaultPrimaryKey))
	}
	children, err := r.Find(ctx, &FindOptions{
		CommonOptions: CommonOptions{Filter: Filter{parentKey: Filter{"$in": ids}}},
		PageSize:      MaxPageSize,
	})
	if err != nil {
		return err
	}
	grouped := map[string][]map[string]interface{}{}
	for _, ch := range children {
		key := cast.ToString(ch.Get(parentKey))
		grouped[key] = append(grouped[key], ch.Data())
	}
	for _, rec := range records {
		key := cast.ToString(rec.Get(DefaultPrimaryKey))
		if list, ok := grouped[key]; ok {
			rec.Set("children", list)
		} else {
			rec.Set("children", []map[string]interface{}{})
		}
	}
	return nil
}

func toInterfaceSlice(raw interface{}) []interface{} {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []interface{}:
		return v
	case []int64:
		out := make([]interface{}, len(v))
		for i, x := range v {
			out[i] = x
		}
		return out
	case string:
		var out []interface{}
		_ = json.Unmarshal([]byte(v), &out)
		return out
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		var out []interface{}
		_ = json.Unmarshal(b, &out)
		return out
	}
}

// extractAssociationValue normalizes nested association payloads into FK or ID lists.
func extractAssociationIDs(value interface{}) []interface{} {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case map[string]interface{}:
		if id, ok := v["id"]; ok {
			return []interface{}{normalizeAssocID(id)}
		}
		return nil
	case []interface{}:
		var ids []interface{}
		for _, item := range v {
			ids = append(ids, extractAssociationIDs(item)...)
		}
		return ids
	case []map[string]interface{}:
		var ids []interface{}
		for _, item := range v {
			if id, ok := item["id"]; ok {
				ids = append(ids, normalizeAssocID(id))
			}
		}
		return ids
	default:
		return []interface{}{normalizeAssocID(v)}
	}
}

func normalizeAssocID(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return i
		}
		return t.String()
	case string:
		if t == "" {
			return t
		}
		if i, err := strconv.ParseInt(t, 10, 64); err == nil {
			return i
		}
		return t
	default:
		s := cast.ToString(v)
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i
		}
		return v
	}
}

// processAssociationWrites handles nested relation values on create/update.
// Returns cleaned values (relation virtual keys removed; belongsTo FK set) and deferred association ops.
func (r *GenericRepository) processAssociationWrites(ctx context.Context, values map[string]interface{}, isCreate bool) (map[string]interface{}, map[string]interface{}, error) {
	assocPending := map[string]interface{}{}
	cleaned := make(map[string]interface{}, len(values))
	for k, v := range values {
		cleaned[k] = v
	}

	for _, f := range r.coll.Fields() {
		if !isRelationField(f) {
			continue
		}
		name := f.Name()
		val, exists := cleaned[name]
		if !exists {
			continue
		}
		ro := GetRelationOptions(f)
		switch FieldType(f.Type()) {
		case FieldTypeBelongsTo:
			delete(cleaned, name)
			if val == nil {
				cleaned[name] = nil
				continue
			}
			ids := extractAssociationIDs(val)
			if len(ids) > 0 {
				cleaned[name] = ids[0]
			} else {
				cleaned[name] = val
			}
		case FieldTypeHasMany, FieldTypeHasOne, FieldTypeBelongsToMany:
			delete(cleaned, name)
			assocPending[name] = val
		case FieldTypeBelongsToManyArray:
			// keep as JSON array of ids
			ids := extractAssociationIDs(val)
			if ids != nil {
				cleaned[name] = ids
			}
		}
		_ = ro
		_ = isCreate
		_ = ctx
	}
	return cleaned, assocPending, nil
}

func (r *GenericRepository) applyPendingAssociations(ctx context.Context, recordID interface{}, pending map[string]interface{}) error {
	for name, val := range pending {
		f := r.coll.GetField(name)
		if f == nil {
			continue
		}
		if err := r.SetAssociation(ctx, recordID, name, val); err != nil {
			return err
		}
	}
	return nil
}

// ListAssociation returns related records for association field.
func (r *GenericRepository) ListAssociation(ctx context.Context, sourceID interface{}, association string) ([]map[string]interface{}, error) {
	f := r.coll.GetField(association)
	if f == nil {
		return nil, NewNotFoundError("关联字段", "name", association)
	}
	rec, err := r.FindOne(ctx, &FindOneOptions{
		CommonOptions: CommonOptions{Appends: Appends{association}},
		FilterByTk:    sourceID,
	})
	if err != nil {
		return nil, err
	}
	v := rec.Get(association)
	switch t := v.(type) {
	case []map[string]interface{}:
		return t, nil
	case map[string]interface{}:
		if t == nil {
			return []map[string]interface{}{}, nil
		}
		return []map[string]interface{}{t}, nil
	case nil:
		return []map[string]interface{}{}, nil
	default:
		return []map[string]interface{}{}, nil
	}
}

// AddAssociation adds related ids (POST).
func (r *GenericRepository) AddAssociation(ctx context.Context, sourceID interface{}, association string, body interface{}) error {
	f := r.coll.GetField(association)
	if f == nil {
		return NewNotFoundError("关联字段", "name", association)
	}
	ids := extractAssociationIDs(body)
	ro := GetRelationOptions(f)
	switch FieldType(f.Type()) {
	case FieldTypeBelongsTo:
		if len(ids) == 0 {
			return fmt.Errorf("需要关联 id")
		}
		_, _, err := r.Update(ctx, &UpdateOptions{
			CommonOptions: CommonOptions{},
			FilterByTk:    sourceID,
			Values:        map[string]interface{}{f.Name(): ids[0]},
		})
		return err
	case FieldTypeHasMany, FieldTypeHasOne:
		target := r.coll.Db().Collection(ro.Target)
		if target == nil {
			return fmt.Errorf("目标集合不存在: %s", ro.Target)
		}
		db := r.execDB()
		for _, id := range ids {
			id = normalizeAssocID(id)
			_, err := db.Exec(ctx, fmt.Sprintf(`UPDATE %s SET %s = ? WHERE %s = ?`,
				quoteIdent(target.TableName()), quoteIdent(ro.ForeignKey), quoteIdent(DefaultPrimaryKey)),
				normalizeAssocID(sourceID), id)
			if err != nil {
				return NewSystemError(err)
			}
		}
		return nil
	case FieldTypeBelongsToMany:
		if ro.Through == "" {
			return fmt.Errorf("缺少 through 表")
		}
		db := r.execDB()
		src := normalizeAssocID(sourceID)
		for _, id := range ids {
			_, err := db.Exec(ctx, fmt.Sprintf(`INSERT IGNORE INTO %s (%s, %s) VALUES (?, ?)`,
				quoteIdent(ro.Through), quoteIdent(ro.ForeignKey), quoteIdent(ro.OtherKey)), src, normalizeAssocID(id))
			if err != nil {
				return NewSystemError(err)
			}
		}
		return nil
	default:
		return fmt.Errorf("不支持的关联类型: %s", f.Type())
	}
}

// SetAssociation replaces association (PUT).
func (r *GenericRepository) SetAssociation(ctx context.Context, sourceID interface{}, association string, body interface{}) error {
	f := r.coll.GetField(association)
	if f == nil {
		return NewNotFoundError("关联字段", "name", association)
	}
	if body == nil {
		return r.RemoveAssociation(ctx, sourceID, association, nil)
	}
	ro := GetRelationOptions(f)
	switch FieldType(f.Type()) {
	case FieldTypeBelongsTo:
		return r.AddAssociation(ctx, sourceID, association, body)
	case FieldTypeHasMany, FieldTypeHasOne:
		target := r.coll.Db().Collection(ro.Target)
		if target == nil {
			return fmt.Errorf("目标集合不存在")
		}
		db := r.execDB()
		src := normalizeAssocID(sourceID)
		// clear existing
		_, err := db.Exec(ctx, fmt.Sprintf(`UPDATE %s SET %s = NULL WHERE %s = ?`,
			quoteIdent(target.TableName()), quoteIdent(ro.ForeignKey), quoteIdent(ro.ForeignKey)), src)
		if err != nil {
			return NewSystemError(err)
		}
		return r.AddAssociation(ctx, sourceID, association, body)
	case FieldTypeBelongsToMany:
		db := r.execDB()
		_, err := db.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE %s = ?`, quoteIdent(ro.Through), quoteIdent(ro.ForeignKey)), normalizeAssocID(sourceID))
		if err != nil {
			return NewSystemError(err)
		}
		return r.AddAssociation(ctx, sourceID, association, body)
	default:
		return fmt.Errorf("不支持的关联类型: %s", f.Type())
	}
}

// RemoveAssociation removes association (DELETE).
func (r *GenericRepository) RemoveAssociation(ctx context.Context, sourceID interface{}, association string, body interface{}) error {
	f := r.coll.GetField(association)
	if f == nil {
		return NewNotFoundError("关联字段", "name", association)
	}
	ro := GetRelationOptions(f)
	ids := extractAssociationIDs(body)
	switch FieldType(f.Type()) {
	case FieldTypeBelongsTo:
		_, _, err := r.Update(ctx, &UpdateOptions{
			FilterByTk: sourceID,
			Values:     map[string]interface{}{f.Name(): nil},
		})
		return err
	case FieldTypeHasMany, FieldTypeHasOne:
		target := r.coll.Db().Collection(ro.Target)
		if target == nil {
			return fmt.Errorf("目标集合不存在")
		}
		if len(ids) == 0 {
			existing, err := target.Repository().Find(ctx, &FindOptions{
				CommonOptions: CommonOptions{Filter: Filter{ro.ForeignKey: sourceID}},
				PageSize:      MaxPageSize,
			})
			if err != nil {
				return err
			}
			for _, rec := range existing {
				ids = append(ids, rec.Get(DefaultPrimaryKey))
			}
		}
		for _, id := range ids {
			_, _, err := target.Repository().Update(ctx, &UpdateOptions{
				FilterByTk: id,
				Values:     map[string]interface{}{ro.ForeignKey: nil},
			})
			if err != nil {
				return err
			}
		}
		return nil
	case FieldTypeBelongsToMany:
		db := r.execDB()
		src := normalizeAssocID(sourceID)
		if len(ids) == 0 {
			_, err := db.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE %s = ?`, quoteIdent(ro.Through), quoteIdent(ro.ForeignKey)), src)
			if err != nil {
				return NewSystemError(err)
			}
			return nil
		}
		for _, id := range ids {
			_, err := db.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE %s = ? AND %s = ?`,
				quoteIdent(ro.Through), quoteIdent(ro.ForeignKey), quoteIdent(ro.OtherKey)), src, normalizeAssocID(id))
			if err != nil {
				return NewSystemError(err)
			}
		}
		return nil
	default:
		return fmt.Errorf("不支持的关联类型: %s", f.Type())
	}
}
