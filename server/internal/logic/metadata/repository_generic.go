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

func (t *txDB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

func (t *txDB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return t.tx.QueryContext(ctx, query, args...)
}

func (t *txDB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return t.tx.QueryRowContext(ctx, query, args...)
}

func (t *txDB) Close() error {
	return nil
}

type GenericRepository struct {
	coll *Collection
	tx   *sql.Tx
	txdb DB
}

func NewGenericRepository(coll *Collection) *GenericRepository {
	return &GenericRepository{
		coll: coll,
	}
}

func (r *GenericRepository) Collection() *Collection {
	return r.coll
}

func (r *GenericRepository) execDB() DB {
	if r.txdb != nil {
		return r.txdb
	}
	return r.coll.Db().DB()
}

func (r *GenericRepository) withTx(tx *sql.Tx) *GenericRepository {
	return &GenericRepository{
		coll: r.coll,
		tx:   tx,
		txdb: &txDB{tx: tx},
	}
}

func (r *GenericRepository) applyFieldSelection(opts *CommonOptions) []Field {
	allFields := r.coll.Fields()

	if len(opts.Fields) == 0 && len(opts.Except) == 0 {
		return allFields
	}

	exceptSet := make(map[string]bool)
	for _, e := range opts.Except {
		exceptSet[e] = true
	}

	if len(opts.Fields) > 0 {
		fieldSet := make(map[string]bool)
		for _, f := range opts.Fields {
			fieldSet[f] = true
		}
		var result []Field
		for _, f := range allFields {
			if fieldSet[f.Name()] && !exceptSet[f.Name()] {
				result = append(result, f)
			}
		}
		return result
	}

	var result []Field
	for _, f := range allFields {
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
			parts = append(parts, fmt.Sprintf(`"%s" DESC`, s))
		} else {
			parts = append(parts, fmt.Sprintf(`"%s" ASC`, s))
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
		values := make([]interface{}, len(columns))
		scanArgs := make([]interface{}, len(columns))
		for i := range values {
			scanArgs[i] = &values[i]
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, NewSystemError(err)
		}

		data := make(map[string]interface{})
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

func (r *GenericRepository) filterFieldsByWhitelist(data map[string]interface{}, whitelist []string, blacklist []string) map[string]interface{} {
	if len(whitelist) == 0 && len(blacklist) == 0 {
		return data
	}

	blackSet := make(map[string]bool)
	for _, b := range blacklist {
		blackSet[b] = true
	}

	result := make(map[string]interface{})

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

func (r *GenericRepository) applySystemFieldsForCreate(data map[string]interface{}) (int64, time.Time) {
	now := time.Now()
	var id int64

	if _, ok := data[DefaultPrimaryKey]; !ok {
		id = utils.NextID()
		data[DefaultPrimaryKey] = id
	} else {
		id = cast.ToInt64(data[DefaultPrimaryKey])
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

	return id, now
}

func (r *GenericRepository) convertValuesToStore(data map[string]interface{}) ([]string, []interface{}, error) {
	var columns []string
	var args []interface{}

	for k, v := range data {
		f := r.coll.GetField(k)
		if f == nil {
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

func (r *GenericRepository) Create(ctx context.Context, opts *CreateOptions) (*Record, error) {
	if opts == nil {
		opts = &CreateOptions{}
	}
	if opts.Values == nil {
		opts.Values = make(map[string]interface{})
	}

	values := make(map[string]interface{})
	for k, v := range opts.Values {
		values[k] = v
	}

	values = r.filterFieldsByWhitelist(values, opts.Whitelist, opts.Blacklist)
	id, _ := r.applySystemFieldsForCreate(values)

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
		quotedColumns[i] = fmt.Sprintf(`"%s"`, col)
	}

	query := fmt.Sprintf(`INSERT INTO "%s" (%s) VALUES (%s)`,
		r.coll.TableName(),
		strings.Join(quotedColumns, ", "),
		strings.Join(placeholders, ", "),
	)

	db := r.execDB()
	result, err := db.Exec(ctx, query, args...)
	if err != nil {
		return nil, NewSystemError(err)
	}

	insertId, err := result.LastInsertId()
	if err == nil && insertId > 0 {
		id = insertId
	}

	record, err := r.FindOne(ctx, &FindOneOptions{
		CommonOptions: CommonOptions{
			Filter: Filter{},
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
		fieldNames[i] = fmt.Sprintf(`"%s"`, f.Name())
	}

	var params []interface{}
	filter := opts.Filter
	if filter == nil {
		filter = make(Filter)
	}

	if opts.FilterByTk != nil {
		filter[DefaultPrimaryKey] = opts.FilterByTk
	}

	whereClause, err := BuildWhereClause(filter, &params)
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
	b.WriteString(fmt.Sprintf(`SELECT %s FROM "%s" WHERE %s`,
		strings.Join(fieldNames, ", "),
		r.coll.TableName(),
		whereClause,
	))
	if orderBy != "" {
		b.WriteString(" ")
		b.WriteString(orderBy)
	}
	b.WriteString(fmt.Sprintf(" LIMIT ? OFFSET ?"))
	params = append(params, limit, offset)

	db := r.execDB()
	rows, err := db.Query(ctx, b.String(), params...)
	if err != nil {
		return nil, NewSystemError(err)
	}
	defer rows.Close()

	records, err := r.scanRows(rows, fields)
	if err != nil {
		return nil, err
	}

	if yaegi := r.coll.Db().Yaegi(); yaegi != nil {
		_ = yaegi
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

	var params []interface{}
	filter := opts.Filter
	if filter == nil {
		filter = make(Filter)
	}

	whereClause, err := BuildWhereClause(filter, &params)
	if err != nil {
		return 0, err
	}

	query := fmt.Sprintf(`SELECT COUNT(*) FROM "%s" WHERE %s`,
		r.coll.TableName(),
		whereClause,
	)

	db := r.execDB()
	var count int
	row := db.QueryRow(ctx, query, params...)
	if err := row.Scan(&count); err != nil {
		return 0, NewSystemError(err)
	}

	return count, nil
}

func (r *GenericRepository) Update(ctx context.Context, opts *UpdateOptions) (*Record, int, error) {
	if opts == nil {
		return nil, 0, fmt.Errorf("update options required")
	}
	if opts.Values == nil {
		opts.Values = make(map[string]interface{})
	}

	values := make(map[string]interface{})
	for k, v := range opts.Values {
		values[k] = v
	}

	values = r.filterFieldsByWhitelist(values, opts.Whitelist, opts.Blacklist)

	now := time.Now()
	if r.coll.HasField("updated_at") {
		values["updated_at"] = now
	}

	delete(values, DefaultPrimaryKey)
	delete(values, "created_at")

	filter := opts.Filter
	if filter == nil {
		filter = make(Filter)
	}

	single := false
	var singleId interface{}
	if opts.FilterByTk != nil {
		single = true
		singleId = opts.FilterByTk
		filter[DefaultPrimaryKey] = singleId
	}

	// Fetch existing records for update validation (oldData)
	var oldData map[string]interface{}
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
	var params []interface{}

	for k, v := range values {
		f := r.coll.GetField(k)
		if f == nil {
			continue
		}
		storeVal, err := f.ToStoreValue(v)
		if err != nil {
			return nil, 0, err
		}
		setClauses = append(setClauses, fmt.Sprintf(`"%s" = ?`, k))
		params = append(params, storeVal)
	}

	if len(setClauses) == 0 {
		return nil, 0, fmt.Errorf("no fields to update")
	}

	whereClause, err := BuildWhereClause(filter, &params)
	if err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`UPDATE "%s" SET %s WHERE %s`,
		r.coll.TableName(),
		strings.Join(setClauses, ", "),
		whereClause,
	)

	db := r.execDB()
	result, err := db.Exec(ctx, query, params...)
	if err != nil {
		return nil, 0, NewSystemError(err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		affected = 0
	}

	var updatedRecord *Record
	if single {
		updatedRecord, err = r.FindOne(ctx, &FindOneOptions{
			FilterByTk: singleId,
		})
		if err != nil {
			return nil, int(affected), err
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

	return updatedRecord, int(affected), nil
}

func (r *GenericRepository) Destroy(ctx context.Context, opts *DestroyOptions) (int, error) {
	if opts == nil {
		return 0, fmt.Errorf("destroy options required")
	}

	var filter Filter
	var params []interface{}

	if opts.Truncate {
		query := fmt.Sprintf(`DELETE FROM "%s"`, r.coll.TableName())
		db := r.execDB()
		result, err := db.Exec(ctx, query)
		if err != nil {
			return 0, NewSystemError(err)
		}
		affected, _ := result.RowsAffected()
		return int(affected), nil
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

	whereClause, err := BuildWhereClause(filter, &params)
	if err != nil {
		return 0, err
	}

	query := fmt.Sprintf(`DELETE FROM "%s" WHERE %s`,
		r.coll.TableName(),
		whereClause,
	)

	db := r.execDB()
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
	sqlDB := r.coll.Db().SqlDB()
	if sqlDB == nil {
		return fmt.Errorf("no sql db available")
	}

	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return NewSystemError(err)
	}

	txRepo := r.withTx(tx)

	if err := fn(txRepo); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return NewSystemError(fmt.Errorf("transaction rollback failed: %v (original error: %w)", rbErr, err))
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return NewSystemError(err)
	}

	return nil
}

var _ Repository = (*GenericRepository)(nil)
