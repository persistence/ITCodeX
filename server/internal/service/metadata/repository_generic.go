package metadata

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cast"

	"itcodex/server/pkg/utils"
)

type txDB struct {
	tx *sql.Tx
}

func (t *txDB) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

func (t *txDB) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return t.tx.QueryContext(ctx, query, args...)
}

func (t *txDB) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return t.tx.QueryRowContext(ctx, query, args...)
}

func (t *txDB) Close() error {
	return nil
}

type GenericRepository struct {
	coll *Collection
	tx   *sql.Tx
	txdb DB
	unit *writeUnit
}

func NewGenericRepository(coll *Collection) *GenericRepository {
	return &GenericRepository{
		coll: coll,
	}
}

func (r *GenericRepository) Collection() *Collection {
	return r.coll
}

func (r *GenericRepository) execDB(ctx context.Context) DB {
	if u := WriteUnitFromContext(ctx); u != nil && u.tx != nil {
		return u.tx
	}
	if r.txdb != nil {
		return r.txdb
	}
	return r.coll.Db().DB()
}

func (r *GenericRepository) withTx(tx *sql.Tx) *GenericRepository {
	u := r.unit
	if u == nil {
		u = &writeUnit{tx: &txDB{tx: tx}}
	}
	return &GenericRepository{
		coll: r.coll,
		tx:   tx,
		txdb: &txDB{tx: tx},
		unit: u,
	}
}

func (r *GenericRepository) applyFieldSelection(opts *CommonOptions) []Field {
	allFields := r.coll.Fields()

	if len(opts.Fields) == 0 && len(opts.Except) == 0 {
		var result []Field
		for _, f := range allFields {
			if isVirtualField(f) {
				continue
			}
			result = append(result, f)
		}
		return result
	}

	exceptSet := make(map[string]bool)
	for _, e := range opts.Except {
		exceptSet[e] = true
	}

	// Ensure primary key and timestamp fields are always available when explicit fields are selected
	ensureSet := map[string]bool{DefaultPrimaryKey: true}
	for _, sysName := range []string{"created_at", "updated_at"} {
		if r.coll.HasField(sysName) {
			ensureSet[sysName] = true
		}
	}

	if len(opts.Fields) > 0 {
		fieldSet := make(map[string]bool)
		for _, f := range opts.Fields {
			fieldSet[f] = true
		}
		// also include the "ensure" fields
		for k := range ensureSet {
			fieldSet[k] = true
		}
		var result []Field
		for _, f := range allFields {
			if isVirtualField(f) {
				continue
			}
			if fieldSet[f.Name()] && !exceptSet[f.Name()] {
				result = append(result, f)
			}
		}
		return result
	}

	var result []Field
	for _, f := range allFields {
		if isVirtualField(f) {
			continue
		}
		if !exceptSet[f.Name()] {
			result = append(result, f)
		}
	}
	return result
}

func buildOrderBy(sort Sort) string {
	if len(sort) == 0 {
		return ""
	}
	var parts []string
	for _, s := range sort {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		desc := false
		if strings.HasPrefix(s, "-") {
			desc = true
			s = s[1:]
		}
		if desc {
			parts = append(parts, fmt.Sprintf("%s DESC", quoteIdent(s)))
		} else {
			parts = append(parts, fmt.Sprintf("%s ASC", quoteIdent(s)))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "ORDER BY " + strings.Join(parts, ", ")
}

func (r *GenericRepository) scanRows(rows *sql.Rows, fields []Field) ([]*Record, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, NewSystemError(err)
	}

	fieldMap := make(map[string]Field)
	for _, f := range fields {
		fieldMap[f.Name()] = f
	}

	var records []*Record
	for rows.Next() {
		values := make([]any, len(columns))
		scanArgs := make([]any, len(columns))
		for i := range values {
			scanArgs[i] = &values[i]
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, NewSystemError(err)
		}

		data := make(map[string]any)
		for i, col := range columns {
			val := values[i]
			if f, ok := fieldMap[col]; ok {
				converted, err := f.FromStoreValue(val)
				if err != nil {
					return nil, NewSystemError(err)
				}
				if t, ok := converted.(time.Time); ok {
					if f.DBType() == DataTypeDateTime || f.DBType() == DataTypeDate || f.DBType() == DataTypeTime || f.DBType() == DataTypeTimestamp {
						data[col] = utils.ToTime(t)
					} else {
						data[col] = converted
					}
				} else {
					data[col] = converted
				}
			} else {
				data[col] = val
			}
		}
		records = append(records, NewRecord(data))
	}

	if err := rows.Err(); err != nil {
		return nil, NewSystemError(err)
	}

	return records, nil
}

func (r *GenericRepository) filterFieldsByWhitelist(data map[string]any, whitelist []string, blacklist []string) map[string]any {
	if len(whitelist) == 0 && len(blacklist) == 0 {
		return data
	}

	blackSet := make(map[string]bool)
	for _, b := range blacklist {
		blackSet[b] = true
	}

	result := make(map[string]any)

	if len(whitelist) > 0 {
		whiteSet := make(map[string]bool)
		for _, w := range whitelist {
			whiteSet[w] = true
		}
		for k, v := range data {
			if whiteSet[k] && !blackSet[k] {
				result[k] = v
			}
		}
	} else {
		for k, v := range data {
			if !blackSet[k] {
				result[k] = v
			}
		}
	}

	return result
}

func (r *GenericRepository) applySystemFieldsForCreate(ctx context.Context, data map[string]any) (int64, time.Time) {
	now := time.Now()
	var id int64

	if v, ok := data[DefaultPrimaryKey]; ok {
		id = cast.ToInt64(v)
	} else {
		id = utils.NextID()
		data[DefaultPrimaryKey] = id
	}

	if _, ok := data["created_at"]; !ok {
		if r.coll.HasField("created_at") {
			data["created_at"] = now
		}
	}

	if _, ok := data["updated_at"]; !ok {
		if r.coll.HasField("updated_at") {
			data["updated_at"] = now
		}
	}

	if actorID, ok := ActorIDFromContext(ctx); ok {
		if _, exists := data["created_by"]; !exists && r.coll.HasField("created_by") {
			data["created_by"] = actorID
		}
		if _, exists := data["updated_by"]; !exists && r.coll.HasField("updated_by") {
			data["updated_by"] = actorID
		}
	}

	return id, now
}

func (r *GenericRepository) convertValuesToStore(data map[string]any) ([]string, []any, error) {
	var columns []string
	var args []any

	for k, v := range data {
		f := r.coll.GetField(k)
		if f == nil || isVirtualField(f) {
			continue
		}
		storeVal, err := f.ToStoreValue(v)
		if err != nil {
			return nil, nil, err
		}
		columns = append(columns, k)
		args = append(args, storeVal)
	}

	return columns, args, nil
}

func (r *GenericRepository) applyAutoFields(values map[string]any) error {
	for _, f := range r.coll.Fields() {
		name := f.Name()
		switch FieldType(f.Type()) {
		case FieldTypeSequence:
			if _, ok := values[name]; !ok || isEmptyValue(values[name]) {
				opts := f.Options()
				pattern, _ := opts["pattern"].(string)
				startsAt := cast.ToInt(opts["startsAt"])
				inc := cast.ToInt(opts["incrementBy"])
				values[name] = nextSequenceValue(r.coll.Name(), name, pattern, startsAt, inc)
			}
		case FieldTypeUUID, FieldTypeNanoID:
			if _, ok := values[name]; !ok || isEmptyValue(values[name]) {
				sv, err := f.ToStoreValue(nil)
				if err != nil {
					return err
				}
				values[name] = sv
			}
		case FieldTypeFormula:
			opts := f.Options()
			expr, _ := opts["expression"].(string)
			if expr == "" {
				continue
			}
			val, err := EvaluateFormula(r.coll, expr, values)
			if err != nil {
				return err
			}
			values[name] = val
		case FieldTypeSort:
			if _, ok := values[name]; !ok || isEmptyValue(values[name]) {
				// default to max+1 simplified: use timestamp seconds
				values[name] = int(time.Now().Unix() % 100000000)
			}
		}
	}
	return nil
}

func (r *GenericRepository) Create(ctx context.Context, opts *CreateOptions) (*Record, error) {
	if !r.inWriteUnit(ctx) {
		var record *Record
		err := r.runWrite(ctx, func(ctx context.Context, repo *GenericRepository) error {
			var e error
			record, e = repo.Create(ctx, opts)
			return e
		})
		return record, err
	}
	if opts == nil {
		opts = &CreateOptions{}
	}
	if opts.Values == nil {
		opts.Values = make(map[string]any)
	}

	values := make(map[string]any)
	for k, v := range opts.Values {
		values[k] = v
	}

	values = r.filterFieldsByWhitelist(values, opts.Whitelist, opts.Blacklist)
	id, _ := r.applySystemFieldsForCreate(ctx, values)

	cleaned, pending, err := r.processAssociationWrites(ctx, values, true)
	if err != nil {
		return nil, err
	}
	values = cleaned

	if err := r.applyAutoFields(values); err != nil {
		return nil, err
	}

	// Run validation
	if v := r.coll.Db().Validator(); v != nil {
		if err := v.ValidateRecord(ctx, r.coll, values, nil, false); err != nil {
			return nil, err
		}
	}

	if yaegi := r.coll.Db().Yaegi(); yaegi != nil {
		var err error
		values, err = yaegi.ExecuteBeforeCreate(ctx, r.coll, values)
		if err != nil {
			return nil, err
		}
	}

	columns, args, err := r.convertValuesToStore(values)
	if err != nil {
		return nil, err
	}

	if len(columns) == 0 {
		return nil, fmt.Errorf("no fields to insert")
	}

	placeholders := make([]string, len(columns))
	quotedColumns := make([]string, len(columns))
	for i, col := range columns {
		placeholders[i] = "?"
		quotedColumns[i] = quoteIdent(col)
	}

	query := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`,
		quoteIdent(r.coll.TableName()),
		strings.Join(quotedColumns, ", "),
		strings.Join(placeholders, ", "),
	)

	db := r.execDB(ctx)
	result, err := db.Exec(ctx, query, args...)
	if err != nil {
		return nil, NewSystemError(err)
	}

	insertId, err := result.LastInsertId()
	if err == nil && insertId > 0 {
		id = insertId
	}
	if v, ok := values[DefaultPrimaryKey]; ok {
		id = cast.ToInt64(v)
	}

	if err := r.applyPendingAssociations(ctx, id, pending); err != nil {
		return nil, err
	}

	record, err := r.FindOne(ctx, &FindOneOptions{
		CommonOptions: CommonOptions{
			Filter:  Filter{},
			Appends: opts.Appends,
		},
		FilterByTk: id,
	})
	if err != nil {
		return nil, err
	}

	if yaegi := r.coll.Db().Yaegi(); yaegi != nil {
		if err := yaegi.ExecuteAfterCreate(ctx, r.coll, record); err != nil {
			return nil, err
		}
	}

	created := record
	r.registerAfterCommit(ctx, func() {
		if yaegi := r.coll.Db().Yaegi(); yaegi != nil && created != nil {
			_ = yaegi.ExecuteAfterCommit(withAfterCommitRunning(context.Background()), r.coll, created)
		}
	})

	return record, nil
}

func (r *GenericRepository) CreateMany(ctx context.Context, opts *CreateManyOptions) ([]*Record, error) {
	if opts == nil {
		return nil, nil
	}

	var records []*Record
	for _, recData := range opts.Records {
		createOpts := &CreateOptions{
			CommonOptions: opts.CommonOptions,
			Values:        recData,
			Whitelist:     opts.Whitelist,
			Blacklist:     opts.Blacklist,
		}
		rec, err := r.Create(ctx, createOpts)
		if err != nil {
			return records, err
		}
		records = append(records, rec)
	}
	return records, nil
}

func (r *GenericRepository) Find(ctx context.Context, opts *FindOptions) ([]*Record, error) {
	if opts == nil {
		opts = &FindOptions{}
	}

	fields := r.applyFieldSelection(&opts.CommonOptions)
	if len(fields) == 0 {
		return nil, nil
	}

	fieldNames := make([]string, len(fields))
	for i, f := range fields {
		fieldNames[i] = quoteIdent(f.Name())
	}

	var params []any
	filter := opts.Filter
	if filter == nil {
		filter = make(Filter)
	}

	if opts.FilterByTk != nil {
		filter[DefaultPrimaryKey] = opts.FilterByTk
	}

	whereClause, err := BuildWhereClauseWithCollection(r.coll, filter, &params)
	if err != nil {
		return nil, err
	}

	orderBy := buildOrderBy(opts.Sort)

	limit := DefaultPageSize
	page := DefaultPage
	if opts.Limit > 0 {
		limit = opts.Limit
	} else if opts.PageSize > 0 {
		limit = opts.PageSize
	}
	if opts.Page > 0 {
		page = opts.Page
	}
	offset := (page - 1) * limit
	if opts.Offset > 0 {
		offset = opts.Offset
	}

	if limit > MaxPageSize {
		limit = MaxPageSize
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf(`SELECT %s FROM %s WHERE %s`,
		strings.Join(fieldNames, ", "),
		quoteIdent(r.coll.TableName()),
		whereClause,
	))
	if orderBy != "" {
		b.WriteString(" ")
		b.WriteString(orderBy)
	}
	b.WriteString(" LIMIT ? OFFSET ?")
	params = append(params, limit, offset)

	db := r.execDB(ctx)
	rows, err := db.Query(ctx, b.String(), params...)
	if err != nil {
		return nil, NewSystemError(err)
	}
	defer rows.Close()

	records, err := r.scanRows(rows, fields)
	if err != nil {
		return nil, err
	}

	if len(opts.Appends) > 0 {
		if err := r.loadAppends(ctx, records, opts.Appends); err != nil {
			return nil, err
		}
	}

	if yaegi := r.coll.Db().Yaegi(); yaegi != nil {
		_ = yaegi.ExecuteAfterFind(ctx, r.coll, records)
	}

	return records, nil
}

func (r *GenericRepository) FindOne(ctx context.Context, opts *FindOneOptions) (*Record, error) {
	if opts == nil {
		opts = &FindOneOptions{}
	}

	findOpts := &FindOptions{
		CommonOptions: opts.CommonOptions,
		Limit:         1,
		Page:          1,
	}

	filter := opts.Filter
	if filter == nil {
		filter = make(Filter)
	}

	if opts.FilterByTk != nil {
		filter[DefaultPrimaryKey] = opts.FilterByTk
	}
	findOpts.Filter = filter

	records, err := r.Find(ctx, findOpts)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, NewNotFoundError("记录", "", nil)
	}
	return records[0], nil
}

func (r *GenericRepository) FindAndCount(ctx context.Context, opts *FindOptions) ([]*Record, int, error) {
	records, err := r.Find(ctx, opts)
	if err != nil {
		return nil, 0, err
	}

	countOpts := &CountOptions{
		CommonOptions: opts.CommonOptions,
	}
	total, err := r.Count(ctx, countOpts)
	if err != nil {
		return records, 0, err
	}

	return records, total, nil
}

func (r *GenericRepository) Count(ctx context.Context, opts *CountOptions) (int, error) {
	if opts == nil {
		opts = &CountOptions{}
	}

	var params []any
	filter := opts.Filter
	if filter == nil {
		filter = make(Filter)
	}

	whereClause, err := BuildWhereClauseWithCollection(r.coll, filter, &params)
	if err != nil {
		return 0, err
	}

	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s`,
		quoteIdent(r.coll.TableName()),
		whereClause,
	)

	db := r.execDB(ctx)
	var count int
	row := db.QueryRow(ctx, query, params...)
	if err := row.Scan(&count); err != nil {
		return 0, NewSystemError(err)
	}

	return count, nil
}

func (r *GenericRepository) Update(ctx context.Context, opts *UpdateOptions) (*Record, int, error) {
	if !r.inWriteUnit(ctx) {
		var (
			record   *Record
			affected int
		)
		err := r.runWrite(ctx, func(ctx context.Context, repo *GenericRepository) error {
			var e error
			record, affected, e = repo.Update(ctx, opts)
			return e
		})
		return record, affected, err
	}
	if opts == nil {
		return nil, 0, fmt.Errorf("update options required")
	}
	if opts.Values == nil {
		opts.Values = make(map[string]any)
	}

	values := make(map[string]any)
	for k, v := range opts.Values {
		values[k] = v
	}

	values = r.filterFieldsByWhitelist(values, opts.Whitelist, opts.Blacklist)

	now := time.Now()
	if r.coll.HasField("updated_at") {
		values["updated_at"] = now
	}
	if actorID, ok := ActorIDFromContext(ctx); ok && r.coll.HasField("updated_by") {
		values["updated_by"] = actorID
	}

	delete(values, DefaultPrimaryKey)
	delete(values, "created_at")

	cleaned, pending, err := r.processAssociationWrites(ctx, values, false)
	if err != nil {
		return nil, 0, err
	}
	values = cleaned

	if err := r.applyAutoFields(values); err != nil {
		return nil, 0, err
	}

	filter := opts.Filter
	if filter == nil {
		filter = make(Filter)
	}

	single := false
	var singleId any
	if opts.FilterByTk != nil {
		single = true
		singleId = opts.FilterByTk
		filter[DefaultPrimaryKey] = singleId
	}

	// Fetch existing records for update validation (oldData)
	var oldData map[string]any
	if single {
		old, err := r.FindOne(ctx, &FindOneOptions{FilterByTk: singleId})
		if err == nil && old != nil {
			oldData = old.Data()
		}
	}

	// Run validation for update
	if v := r.coll.Db().Validator(); v != nil {
		if err := v.ValidateRecord(ctx, r.coll, values, oldData, true); err != nil {
			return nil, 0, err
		}
	}

	if yaegi := r.coll.Db().Yaegi(); yaegi != nil {
		var err error
		values, err = yaegi.ExecuteBeforeUpdate(ctx, r.coll, values, filter)
		if err != nil {
			return nil, 0, err
		}
	}

	var setClauses []string
	var params []any

	for k, v := range values {
		f := r.coll.GetField(k)
		if f == nil || isVirtualField(f) {
			continue
		}
		storeVal, err := f.ToStoreValue(v)
		if err != nil {
			return nil, 0, err
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", quoteIdent(k)))
		params = append(params, storeVal)
	}

	var affected int64
	if len(setClauses) == 0 {
		if len(pending) == 0 {
			return nil, 0, fmt.Errorf("no fields to update")
		}
	} else {
		whereClause, werr := BuildWhereClauseWithCollection(r.coll, filter, &params)
		if werr != nil {
			return nil, 0, werr
		}

		query := fmt.Sprintf(`UPDATE %s SET %s WHERE %s`,
			quoteIdent(r.coll.TableName()),
			strings.Join(setClauses, ", "),
			whereClause,
		)

		db := r.execDB(ctx)
		result, execErr := db.Exec(ctx, query, params...)
		if execErr != nil {
			return nil, 0, NewSystemError(execErr)
		}

		affected, _ = result.RowsAffected()
	}

	if single {
		if err := r.applyPendingAssociations(ctx, singleId, pending); err != nil {
			return nil, 0, err
		}
	}

	var updatedRecord *Record
	if single {
		var findErr error
		updatedRecord, findErr = r.FindOne(ctx, &FindOneOptions{
			CommonOptions: CommonOptions{Appends: opts.Appends},
			FilterByTk:    singleId,
		})
		if findErr != nil {
			return nil, int(affected), findErr
		}
		if affected == 0 {
			affected = 1
		}
	}

	if yaegi := r.coll.Db().Yaegi(); yaegi != nil {
		var records []*Record
		if updatedRecord != nil {
			records = []*Record{updatedRecord}
		}
		if err := yaegi.ExecuteAfterUpdate(ctx, r.coll, records); err != nil {
			return nil, int(affected), err
		}
	}

	committed := updatedRecord
	r.registerAfterCommit(ctx, func() {
		if yaegi := r.coll.Db().Yaegi(); yaegi != nil && committed != nil {
			_ = yaegi.ExecuteAfterCommit(withAfterCommitRunning(context.Background()), r.coll, committed)
		}
	})

	return updatedRecord, int(affected), nil
}

func (r *GenericRepository) Destroy(ctx context.Context, opts *DestroyOptions) (int, error) {
	if !r.inWriteUnit(ctx) {
		var affected int
		err := r.runWrite(ctx, func(ctx context.Context, repo *GenericRepository) error {
			var e error
			affected, e = repo.Destroy(ctx, opts)
			return e
		})
		return affected, err
	}
	if opts == nil {
		return 0, fmt.Errorf("destroy options required")
	}

	var filter Filter
	var params []any

	emptyFilter := (opts.Filter == nil || len(opts.Filter) == 0) && opts.FilterByTk == nil
	if opts.Truncate || (emptyFilter && r.coll.Db() != nil && r.coll.Db().AllowTruncate()) {
		if !opts.Truncate && (r.coll.Db() == nil || !r.coll.Db().AllowTruncate()) {
			return 0, NewForbiddenError("禁止无条件清空表，请提供 filter 或开启 allowTruncate")
		}
		query := fmt.Sprintf(`DELETE FROM %s`, quoteIdent(r.coll.TableName()))
		db := r.execDB(ctx)
		result, err := db.Exec(ctx, query)
		if err != nil {
			return 0, NewSystemError(err)
		}
		affected, _ := result.RowsAffected()
		return int(affected), nil
	}

	if emptyFilter {
		return 0, NewForbiddenError("禁止无条件清空表，请提供 filter 或开启 allowTruncate")
	}

	filter = opts.Filter
	if filter == nil {
		filter = make(Filter)
	}

	if opts.FilterByTk != nil {
		filter[DefaultPrimaryKey] = opts.FilterByTk
	}

	if yaegi := r.coll.Db().Yaegi(); yaegi != nil {
		if err := yaegi.ExecuteBeforeDelete(ctx, r.coll, filter); err != nil {
			return 0, err
		}
	}

	whereClause, err := BuildWhereClauseWithCollection(r.coll, filter, &params)
	if err != nil {
		return 0, err
	}

	query := fmt.Sprintf(`DELETE FROM %s WHERE %s`,
		quoteIdent(r.coll.TableName()),
		whereClause,
	)

	db := r.execDB(ctx)
	result, err := db.Exec(ctx, query, params...)
	if err != nil {
		return 0, NewSystemError(err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		affected = 0
	}

	if yaegi := r.coll.Db().Yaegi(); yaegi != nil {
		if err := yaegi.ExecuteAfterDelete(ctx, r.coll, int(affected)); err != nil {
			return int(affected), err
		}
	}

	return int(affected), nil
}

func (r *GenericRepository) Transaction(ctx context.Context, fn func(tx Repository) error) error {
	if r.inWriteUnit(ctx) {
		u := WriteUnitFromContext(ctx)
		repo := r
		if u != nil {
			repo = r.withUnit(u)
		}
		return fn(repo)
	}
	return r.runWrite(ctx, func(ctx context.Context, repo *GenericRepository) error {
		return fn(repo)
	})
}

var _ Repository = (*GenericRepository)(nil)
