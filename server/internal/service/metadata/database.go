package metadata

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	_ "github.com/go-sql-driver/mysql"

	"itcodex/server/internal/dao"
	"itcodex/server/internal/model/entity"
	modelmd "itcodex/server/internal/model/metadata"
	"itcodex/server/pkg/utils"
	yaegictx "itcodex/server/pkg/yaegi/context"
)

const defaultMySQLDSN = "root:123456@tcp(127.0.0.1:3306)/itcodex?parseTime=true&loc=Local&charset=utf8mb4"

type DB interface {
	Exec(ctx context.Context, query string, args ...any) (sql.Result, error)
	Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) *sql.Row
	Close() error
}

type sqlDBWrapper struct {
	db *sql.DB
}

func (s *sqlDBWrapper) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.db.ExecContext(ctx, query, args...)
}

func (s *sqlDBWrapper) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, query, args...)
}

func (s *sqlDBWrapper) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return s.db.QueryRowContext(ctx, query, args...)
}

func (s *sqlDBWrapper) Close() error {
	return s.db.Close()
}

// QuoteIdent wraps a SQL identifier in MySQL backticks.
func QuoteIdent(s string) string {
	return "`" + strings.ReplaceAll(s, "`", "``") + "`"
}

// quoteIdent is an unexported alias for package-internal use.
func quoteIdent(s string) string {
	return QuoteIdent(s)
}

type YaegiManager interface {
	LoadScript(script *modelmd.YaegiScript) error
	DisableScript(id int64) error
	FindCustomAPI(method, path string) *modelmd.YaegiScript
	ExecuteBeforeCreate(ctx context.Context, coll *Collection, data map[string]any) (map[string]any, error)
	ExecuteAfterCreate(ctx context.Context, coll *Collection, record *Record) error
	ExecuteBeforeUpdate(ctx context.Context, coll *Collection, data map[string]any, filter Filter) (map[string]any, error)
	ExecuteAfterUpdate(ctx context.Context, coll *Collection, records []*Record) error
	ExecuteBeforeDelete(ctx context.Context, coll *Collection, filter Filter) error
	ExecuteAfterDelete(ctx context.Context, coll *Collection, affected int) error
	ExecuteAfterCommit(ctx context.Context, coll *Collection, record *Record) error
	ExecuteBeforeValidate(ctx context.Context, coll *Collection, data map[string]any) (map[string]any, error)
	ExecuteAfterValidate(ctx context.Context, coll *Collection, data map[string]any) error
	ExecuteCustomAPI(script *modelmd.YaegiScript, ctx *yaegictx.YaegiHTTPContext) error
	ValidateScript(content string) error
	ExecuteBeforeFind(ctx context.Context, coll *Collection, filter Filter) (Filter, error)
	ExecuteAfterFind(ctx context.Context, coll *Collection, records []*Record) error
}

type Database struct {
	mu          sync.RWMutex
	collections map[string]*Collection
	operators   map[string]OperatorFunc
	fieldTypes  map[string]FieldFactory
	db          DB
	sqlDB       *sql.DB
	options     DatabaseOptions
	yaegi       YaegiManager
	validator   *CELValidator
}

func NewDatabase(ctx context.Context, opts DatabaseOptions) (*Database, error) {
	utils.SetMachineID(1)

	d := &Database{
		collections: make(map[string]*Collection),
		operators:   make(map[string]OperatorFunc),
		fieldTypes:  make(map[string]FieldFactory),
		options:     opts,
		validator:   NewCELValidator(),
	}

	if opts.TablePrefix == "" {
		d.options.TablePrefix = "c_"
	}

	dsn := opts.DSN
	if dsn == "" {
		// Prefer DSN; empty DSN falls back to default MySQL DSN.
		dsn = defaultMySQLDSN
	}

	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)

	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return nil, err
	}

	d.sqlDB = sqlDB
	d.db = &sqlDBWrapper{db: sqlDB}

	for name, fn := range operators {
		d.operators[name] = fn
	}

	for t, factory := range defaultFieldFactories {
		d.fieldTypes[string(t)] = factory
	}

	return d, nil
}

func (d *Database) Bootstrap(ctx context.Context) error {
	if err := d.createSystemTables(ctx); err != nil {
		return err
	}
	if err := d.loadCollections(ctx); err != nil {
		return err
	}
	if err := d.loadFields(ctx); err != nil {
		return err
	}
	if err := d.loadIndexes(ctx); err != nil {
		return err
	}
	if d.yaegi != nil {
		if err := d.loadEnabledScripts(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (d *Database) createSystemTables(ctx context.Context) error {
	prefix := d.TablePrefix()

	q := quoteIdent
	ddlStatements := []string{
		fmt.Sprintf(
			"CREATE TABLE IF NOT EXISTS %s ("+
				"%s BIGINT NOT NULL PRIMARY KEY AUTO_INCREMENT,"+
				"%s VARCHAR(255) NOT NULL UNIQUE,"+
				"%s VARCHAR(255) NOT NULL DEFAULT '',"+
				"%s VARCHAR(50) NOT NULL DEFAULT 'general',"+
				"%s JSON NULL,"+
				"%s DATETIME NULL,"+
				"%s DATETIME NULL"+
				") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
			q(prefix+"collections"), q("id"), q("name"), q("display_name"), q("type"), q("options"), q("created_at"), q("updated_at"),
		),
		fmt.Sprintf(
			"CREATE TABLE IF NOT EXISTS %s ("+
				"%s BIGINT NOT NULL PRIMARY KEY AUTO_INCREMENT,"+
				"%s VARCHAR(255) NOT NULL,"+
				"%s VARCHAR(255) NOT NULL,"+
				"%s VARCHAR(50) NOT NULL,"+
				"%s VARCHAR(255) NOT NULL DEFAULT '',"+
				"%s TINYINT(1) NOT NULL DEFAULT 0,"+
				"%s TINYINT(1) NOT NULL DEFAULT 0,"+
				"%s TINYINT(1) NOT NULL DEFAULT 0,"+
				"%s JSON NULL,"+
				"%s JSON NULL,"+
				"%s INT NOT NULL DEFAULT 0,"+
				"%s DATETIME NULL,"+
				"UNIQUE KEY %s (%s, %s)"+
				") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
			q(prefix+"fields"), q("id"), q("collection_name"), q("name"), q("type"), q("display_name"),
			q("is_required"), q("is_unique"), q("is_indexed"), q("validation"), q("options"), q("sort"), q("created_at"),
			q("uk_collection_field"), q("collection_name"), q("name"),
		),
		fmt.Sprintf(
			"CREATE TABLE IF NOT EXISTS %s ("+
				"%s BIGINT NOT NULL PRIMARY KEY AUTO_INCREMENT,"+
				"%s VARCHAR(255) NOT NULL,"+
				"%s VARCHAR(255) NOT NULL,"+
				"%s JSON NOT NULL,"+
				"%s TINYINT(1) NOT NULL DEFAULT 0,"+
				"%s JSON NULL,"+
				"%s DATETIME NULL"+
				") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
			q(prefix+"indexes"), q("id"), q("collection_name"), q("name"), q("fields"), q("unique"), q("options"), q("created_at"),
		),
		fmt.Sprintf(
			"CREATE TABLE IF NOT EXISTS %s ("+
				"%s BIGINT NOT NULL PRIMARY KEY AUTO_INCREMENT,"+
				"%s VARCHAR(255) NULL,"+
				"%s VARCHAR(255) NOT NULL,"+
				"%s VARCHAR(50) NOT NULL,"+
				"%s LONGTEXT NOT NULL,"+
				"%s VARCHAR(255) NULL,"+
				"%s VARCHAR(20) NULL,"+
				"%s TINYINT(1) NOT NULL DEFAULT 1,"+
				"%s INT NOT NULL DEFAULT 0,"+
				"%s JSON NULL,"+
				"%s DATETIME NULL,"+
				"%s DATETIME NULL"+
				") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
			q(prefix+"yaegi_scripts"), q("id"), q("collection_name"), q("name"), q("hook_point"), q("content"),
			q("api_path"), q("http_method"), q("enabled"), q("priority"), q("options"), q("created_at"), q("updated_at"),
		),
		fmt.Sprintf(
			"CREATE TABLE IF NOT EXISTS %s ("+
				"%s BIGINT NOT NULL PRIMARY KEY AUTO_INCREMENT,"+
				"%s VARCHAR(191) NOT NULL UNIQUE,"+
				"%s VARCHAR(255) NOT NULL DEFAULT '',"+
				"%s VARCHAR(512) NOT NULL,"+
				"%s TINYINT(1) NOT NULL DEFAULT 1,"+
				"%s DATETIME NULL,"+
				"%s DATETIME NULL"+
				") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
			q(prefix+"auth_users"), q("id"), q("username"), q("display_name"), q("password_hash"), q("enabled"), q("created_at"), q("updated_at"),
		),
		fmt.Sprintf(
			"CREATE TABLE IF NOT EXISTS %s ("+
				"%s BIGINT NOT NULL PRIMARY KEY AUTO_INCREMENT,"+
				"%s VARCHAR(191) NOT NULL UNIQUE,"+
				"%s VARCHAR(255) NOT NULL DEFAULT '',"+
				"%s DATETIME NULL,"+
				"%s DATETIME NULL"+
				") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
			q(prefix+"auth_roles"), q("id"), q("name"), q("display_name"), q("created_at"), q("updated_at"),
		),
		fmt.Sprintf(
			"CREATE TABLE IF NOT EXISTS %s ("+
				"%s BIGINT NOT NULL,"+
				"%s BIGINT NOT NULL,"+
				"%s DATETIME NULL,"+
				"PRIMARY KEY (%s, %s),"+
				"KEY %s (%s)"+
				") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
			q(prefix+"auth_user_roles"), q("user_id"), q("role_id"), q("created_at"),
			q("user_id"), q("role_id"), q("idx_auth_user_roles_role"), q("role_id"),
		),
		fmt.Sprintf(
			"CREATE TABLE IF NOT EXISTS %s ("+
				"%s VARCHAR(64) NOT NULL PRIMARY KEY,"+
				"%s BIGINT NOT NULL,"+
				"%s CHAR(64) NOT NULL,"+
				"%s DATETIME NOT NULL,"+
				"%s DATETIME NULL,"+
				"%s DATETIME NULL,"+
				"KEY %s (%s)"+
				") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
			q(prefix+"auth_sessions"), q("id"), q("user_id"), q("token_hash"), q("expires_at"),
			q("revoked_at"), q("created_at"), q("idx_auth_sessions_user"), q("user_id"),
		),
		fmt.Sprintf(
			"CREATE TABLE IF NOT EXISTS %s ("+
				"%s BIGINT NOT NULL PRIMARY KEY AUTO_INCREMENT,"+
				"%s VARCHAR(20) NOT NULL,"+
				"%s VARCHAR(191) NOT NULL DEFAULT '',"+
				"%s VARCHAR(191) NOT NULL,"+
				"%s VARCHAR(100) NOT NULL,"+
				"%s JSON NULL,"+
				"%s JSON NULL,"+
				"%s JSON NULL,"+
				"%s TINYINT(1) NOT NULL DEFAULT 1,"+
				"%s DATETIME NULL,"+
				"%s DATETIME NULL,"+
				"KEY %s (%s, %s)"+
				") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
			q(prefix+"acl_policies"), q("id"), q("subject_type"), q("subject"), q("resource"), q("action"),
			q("row_filter"), q("read_fields"), q("write_fields"), q("enabled"), q("created_at"), q("updated_at"),
			q("idx_acl_resource_action"), q("resource"), q("action"),
		),
	}

	for _, ddl := range ddlStatements {
		if _, err := d.db.Exec(ctx, ddl); err != nil {
			return NewSystemError(err)
		}
	}

	return nil
}

func (d *Database) loadCollections(ctx context.Context) error {
	var collections []*modelmd.Collection
	if d.useSystemDAO() {
		var rows []entity.Collections
		if err := dao.Collections.Ctx(ctx).Scan(&rows); err != nil {
			return NewSystemError(err)
		}
		for _, row := range rows {
			collections = append(collections, &modelmd.Collection{
				Id:          row.Id,
				Name:        row.Name,
				DisplayName: row.DisplayName,
				Type:        row.Type,
				Options:     row.Options,
				CreatedAt:   row.CreatedAt,
				UpdatedAt:   row.UpdatedAt,
			})
		}
	} else {
		prefix := d.TablePrefix()
		query := fmt.Sprintf(`SELECT id, name, display_name, type, options, created_at, updated_at FROM %s`, quoteIdent(prefix+"collections"))
		rows, err := d.db.Query(ctx, query)
		if err != nil {
			return NewSystemError(err)
		}
		defer rows.Close()

		for rows.Next() {
			var (
				id          int64
				name        string
				displayName string
				typ         string
				options     sql.NullString
				createdAt   sql.NullTime
				updatedAt   sql.NullTime
			)
			if err := rows.Scan(&id, &name, &displayName, &typ, &options, &createdAt, &updatedAt); err != nil {
				return NewSystemError(err)
			}
			m := &modelmd.Collection{
				Id:          id,
				Name:        name,
				DisplayName: displayName,
				Type:        typ,
			}
			if options.Valid {
				m.Options = options.String
			}
			collections = append(collections, m)
		}
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	for _, m := range collections {
		coll, err := d.buildCollectionFromModel(m)
		if err != nil {
			return err
		}
		d.collections[m.Name] = coll
	}

	return nil
}

func (d *Database) loadFields(ctx context.Context) error {
	type fieldModel struct {
		Id             int64
		CollectionName string
		Name           string
		Type           string
		DisplayName    string
		IsRequired     bool
		IsUnique       bool
		IsIndexed      bool
		Validation     sql.NullString
		Options        sql.NullString
		Sort           int
	}

	var fieldModels []*fieldModel
	if d.useSystemDAO() {
		var rows []entity.Fields
		if err := dao.Fields.Ctx(ctx).OrderAsc(dao.Fields.Columns().Sort).OrderAsc(dao.Fields.Columns().Id).Scan(&rows); err != nil {
			return NewSystemError(err)
		}
		for _, row := range rows {
			fm := &fieldModel{
				Id:             row.Id,
				CollectionName: row.CollectionName,
				Name:           row.Name,
				Type:           row.Type,
				DisplayName:    row.DisplayName,
				IsRequired:     row.IsRequired != 0,
				IsUnique:       row.IsUnique != 0,
				IsIndexed:      row.IsIndexed != 0,
				Sort:           row.Sort,
			}
			if row.Validation != "" {
				fm.Validation = sql.NullString{String: row.Validation, Valid: true}
			}
			if row.Options != "" {
				fm.Options = sql.NullString{String: row.Options, Valid: true}
			}
			fieldModels = append(fieldModels, fm)
		}
	} else {
		prefix := d.TablePrefix()
		query := fmt.Sprintf(`SELECT id, collection_name, name, type, display_name, is_required, is_unique, is_indexed, validation, options, sort FROM %s ORDER BY sort ASC, id ASC`, quoteIdent(prefix+"fields"))
		rows, err := d.db.Query(ctx, query)
		if err != nil {
			return NewSystemError(err)
		}
		defer rows.Close()

		for rows.Next() {
			var fm fieldModel
			if err := rows.Scan(&fm.Id, &fm.CollectionName, &fm.Name, &fm.Type, &fm.DisplayName, &fm.IsRequired, &fm.IsUnique, &fm.IsIndexed, &fm.Validation, &fm.Options, &fm.Sort); err != nil {
				return NewSystemError(err)
			}
			fieldModels = append(fieldModels, &fm)
		}
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	for _, fm := range fieldModels {
		coll, ok := d.collections[fm.CollectionName]
		if !ok {
			continue
		}

		opts := make(map[string]any)
		if fm.Options.Valid && fm.Options.String != "" {
			if err := json.Unmarshal([]byte(fm.Options.String), &opts); err != nil {
				continue
			}
		}
		opts["name"] = fm.Name
		opts["displayName"] = fm.DisplayName
		opts["required"] = fm.IsRequired
		opts["unique"] = fm.IsUnique
		opts["indexed"] = fm.IsIndexed
		opts["isSystem"] = isSystemFieldName(fm.Name)

		f, err := NewField(coll, FieldType(fm.Type), opts)
		if err != nil {
			continue
		}
		coll.addFieldInternal(f)
	}

	return nil
}

func (d *Database) loadIndexes(ctx context.Context) error {
	type indexRow struct {
		id             int64
		collectionName string
		name           string
		fieldsJson     string
		unique         bool
	}
	var list []indexRow
	if d.useSystemDAO() {
		var rows []entity.Indexes
		if err := dao.Indexes.Ctx(ctx).Scan(&rows); err != nil {
			return NewSystemError(err)
		}
		for _, row := range rows {
			list = append(list, indexRow{row.Id, row.CollectionName, row.Name, row.Fields, row.Unique != 0})
		}
	} else {
		prefix := d.TablePrefix()
		query := fmt.Sprintf(
			`SELECT id, collection_name, name, fields, %s FROM %s`,
			quoteIdent("unique"),
			quoteIdent(prefix+"indexes"),
		)
		rows, err := d.db.Query(ctx, query)
		if err != nil {
			return NewSystemError(err)
		}
		defer rows.Close()
		for rows.Next() {
			var item indexRow
			if err := rows.Scan(&item.id, &item.collectionName, &item.name, &item.fieldsJson, &item.unique); err != nil {
				return NewSystemError(err)
			}
			list = append(list, item)
		}
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	for _, item := range list {
		coll, ok := d.collections[item.collectionName]
		if !ok {
			continue
		}
		var fields []string
		if err := json.Unmarshal([]byte(item.fieldsJson), &fields); err != nil {
			continue
		}
		coll.indexes = append(coll.indexes, &Index{
			ID:     item.id,
			Name:   item.name,
			Fields: fields,
			Unique: item.unique,
		})
	}
	return nil
}

func (d *Database) loadEnabledScripts(ctx context.Context) error {
	if d.yaegi == nil {
		return nil
	}
	scripts, err := d.listScripts(ctx, "", "")
	if err != nil {
		return err
	}
	for _, script := range scripts {
		if script == nil || !script.Enabled {
			continue
		}
		if err := d.yaegi.LoadScript(script); err != nil {
			return err
		}
	}
	return nil
}

func isSystemFieldName(name string) bool {
	switch name {
	case "id", "created_at", "updated_at", "created_by", "updated_by":
		return true
	default:
		return false
	}
}

func (d *Database) Close(ctx context.Context) error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

func (d *Database) Collection(name string) *Collection {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.collections[name]
}

func (d *Database) Collections() []*Collection {
	d.mu.RLock()
	defer d.mu.RUnlock()
	result := make([]*Collection, 0, len(d.collections))
	for _, c := range d.collections {
		result = append(result, c)
	}
	return result
}

func (d *Database) HasCollection(name string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	_, ok := d.collections[name]
	return ok
}

func (d *Database) AllowTruncate() bool {
	return d.options.AllowTruncate
}

func applyCollectionInputMeta(coll *Collection, input CreateCollectionInput) {
	if input.Options.Extra != nil {
		coll.opts = input.Options.Extra
	}
	if coll.opts == nil {
		coll.opts = make(map[string]any)
	}
	if input.Description != "" {
		coll.opts["description"] = input.Description
	}
	if len(input.Categories) > 0 {
		coll.opts["categories"] = input.Categories
	}
	if input.Options.SimplePagination {
		coll.opts["simplePagination"] = true
	}
	if input.Options.TreeParentKey != "" {
		coll.opts["treeParentKey"] = input.Options.TreeParentKey
	}
	if input.Options.CalendarStartField != "" {
		coll.opts["calendarStartField"] = input.Options.CalendarStartField
	}
	if input.Options.CalendarEndField != "" {
		coll.opts["calendarEndField"] = input.Options.CalendarEndField
	}
	if input.Options.CommentForeignKey != "" {
		coll.opts["commentForeignKey"] = input.Options.CommentForeignKey
	}
}

// applySpecialCollectionDefaults sets type-specific options and injects default fields
// for tree / calendar / comment / file collections.
func applySpecialCollectionDefaults(coll *Collection, typ CollectionType) {
	if coll.opts == nil {
		coll.opts = make(map[string]any)
	}
	switch typ {
	case CollectionTypeTree:
		parentKey := "parent_id"
		if v, ok := coll.opts["treeParentKey"].(string); ok && v != "" {
			parentKey = v
		}
		coll.opts["treeParentKey"] = parentKey
		injectDefaultField(coll, parentKey, "父级", "bigint")
	case CollectionTypeCalendar:
		startField := "start"
		endField := "end"
		if v, ok := coll.opts["calendarStartField"].(string); ok && v != "" {
			startField = v
		}
		if v, ok := coll.opts["calendarEndField"].(string); ok && v != "" {
			endField = v
		}
		coll.opts["calendarStartField"] = startField
		coll.opts["calendarEndField"] = endField
		injectDefaultField(coll, startField, "开始时间", FieldTypeDateTime)
		injectDefaultField(coll, endField, "结束时间", FieldTypeDateTime)
	case CollectionTypeComment:
		fk := "target_id"
		if v, ok := coll.opts["commentForeignKey"].(string); ok && v != "" {
			fk = v
		}
		coll.opts["commentForeignKey"] = fk
		injectDefaultField(coll, fk, "关联目标", "bigint")
	case CollectionTypeFile:
		injectDefaultField(coll, "name", "文件名", FieldTypeString)
		injectDefaultField(coll, "url", "URL", FieldTypeUrl)
		injectDefaultField(coll, "mime", "MIME", FieldTypeString)
		injectDefaultField(coll, "size", "大小", FieldTypeInteger)
		injectDefaultField(coll, "path", "存储路径", FieldTypeString)
	}
}

func injectDefaultField(coll *Collection, name, displayName string, typ FieldType) {
	if _, exists := coll.fields[name]; exists {
		return
	}
	opts := map[string]any{
		"name":        name,
		"displayName": displayName,
		"required":    false,
		"isSystem":    true,
	}
	f, err := NewField(coll, typ, opts)
	if err != nil {
		return
	}
	coll.addFieldInternal(f)
}

func (d *Database) CreateCollection(ctx context.Context, input CreateCollectionInput) (*Collection, error) {
	if !collectionNameRegex.MatchString(input.Name) {
		return nil, fmt.Errorf("无效的集合名: %s", input.Name)
	}

	if input.Type == "" {
		input.Type = CollectionTypeGeneral
	}
	switch input.Type {
	case CollectionTypeGeneral, CollectionTypeTree, CollectionTypeCalendar, CollectionTypeComment, CollectionTypeFile:
	default:
		return nil, fmt.Errorf("不支持的集合类型: %s", input.Type)
	}

	if d.HasCollection(input.Name) {
		return nil, NewAlreadyExistsError("集合", "name", input.Name)
	}

	coll := newCollection(d, input.Name,
		WithDisplayName(input.DisplayName),
		WithType(input.Type),
	)
	applyCollectionInputMeta(coll, input)

	autoGenId := true
	if input.AutoGenId != nil {
		autoGenId = *input.AutoGenId
	}

	presetFields := input.PresetFields
	if len(presetFields) == 0 {
		presetFields = []string{"id", "createdAt", "createdBy", "updatedAt", "updatedBy"}
	}

	if autoGenId {
		idField := NewIDField(coll, nil)
		coll.addFieldInternal(idField)
	}

	for _, pf := range presetFields {
		if pf == "id" {
			continue
		}
		if factory, ok := PresetFieldsMap[pf]; ok {
			f := factory(coll, nil)
			coll.addFieldInternal(f)
		}
	}

	applySpecialCollectionDefaults(coll, input.Type)

	for _, fi := range input.Fields {
		opts := make(map[string]any)
		opts["name"] = fi.Name
		opts["displayName"] = fi.DisplayName
		opts["required"] = fi.IsRequired
		opts["unique"] = fi.IsUnique
		opts["indexed"] = fi.IsIndexed
		opts["isSystem"] = fi.IsSystem
		opts["length"] = fi.Length
		if fi.Options != nil {
			for k, v := range fi.Options {
				opts[k] = v
			}
		}
		attachFieldInputOptions(opts, fi)
		attachRelationInput(opts, fi)
		f, err := NewField(coll, fi.Type, opts)
		if err != nil {
			return nil, err
		}
		coll.addFieldInternal(f)
		if FieldType(f.Type()) == FieldTypeBelongsToMany {
			ro := GetRelationOptions(f)
			if err := ensureThroughTableDDL(ctx, d, ro.Through, ro.ForeignKey, ro.OtherKey); err != nil {
				return nil, NewSystemError(err)
			}
		}
	}

	pendingIndexes := make([]*Index, 0, len(input.Indexes))
	for _, idx := range input.Indexes {
		idxCopy := idx
		if idxCopy.Name == "" {
			idxCopy.Name = fmt.Sprintf("idx_%s_%s", coll.name, strings.Join(idxCopy.Fields, "_"))
		}
		pendingIndexes = append(pendingIndexes, &idxCopy)
		// Unique indexes participate in generateDDL UNIQUE constraints.
		if idxCopy.Unique {
			coll.indexes = append(coll.indexes, &idxCopy)
		}
	}

	if input.TableValidation != nil {
		coll.SetTableValidation(input.TableValidation)
	}

	ddl := coll.generateDDL()
	if _, err := d.db.Exec(ctx, ddl); err != nil {
		return nil, NewSystemError(err)
	}

	optionsJson, _ := json.Marshal(coll.opts)
	if err := d.insertCollectionRow(ctx, coll.name, coll.displayName, string(coll.type_), string(optionsJson)); err != nil {
		return nil, err
	}

	for _, f := range coll.Fields() {
		fOpts := f.Options()
		optsJson, _ := json.Marshal(fOpts)
		if err := d.insertFieldRow(ctx, coll.name, f.Name(), f.Type(), f.DisplayName(), f.IsRequired(), f.IsUnique(), f.IsIndexed(), string(optsJson), 0); err != nil {
			return nil, err
		}
	}

	coll.isNew = false

	// Reset indexes then create/persist all so memory matches c_indexes.
	coll.indexes = coll.indexes[:0]
	for _, idx := range pendingIndexes {
		if idx.Unique {
			coll.indexes = append(coll.indexes, idx)
			if err := coll.persistIndex(ctx, idx); err != nil {
				return nil, err
			}
			continue
		}
		if err := coll.AddIndex(ctx, idx); err != nil {
			return nil, err
		}
	}

	d.mu.Lock()
	d.collections[coll.name] = coll
	d.mu.Unlock()

	return coll, nil
}

func (d *Database) UpdateCollection(ctx context.Context, name string, input UpdateCollectionInput) (*Collection, error) {
	coll := d.Collection(name)
	if coll == nil {
		return nil, NewNotFoundError("集合", "name", name)
	}
	if err := coll.UpdateMeta(ctx, input); err != nil {
		return nil, err
	}
	return coll, nil
}

func (d *Database) DropCollection(ctx context.Context, name string, cascade ...bool) error {
	doCascade := false
	if len(cascade) > 0 {
		doCascade = cascade[0]
	}

	d.mu.Lock()
	coll, ok := d.collections[name]
	if !ok {
		d.mu.Unlock()
		return NewNotFoundError("集合", "name", name)
	}
	d.mu.Unlock()

	deps := d.dependentCollections(name)
	if len(deps) > 0 && !doCascade {
		return NewForbiddenError(fmt.Sprintf("集合 %s 被 %s 依赖，请使用 cascade=true 级联删除", name, strings.Join(deps, ", ")))
	}
	if doCascade {
		for _, dep := range deps {
			if err := d.DropCollection(ctx, dep, true); err != nil {
				if _, ok := err.(*NotFoundError); !ok {
					return err
				}
			}
		}
	}

	for _, f := range coll.Fields() {
		if FieldType(f.Type()) == FieldTypeBelongsToMany {
			ro := GetRelationOptions(f)
			if ro.Through != "" {
				_, _ = d.db.Exec(ctx, fmt.Sprintf(`DROP TABLE IF EXISTS %s`, quoteIdent(ro.Through)))
			}
		}
	}

	dropDDL := fmt.Sprintf(`DROP TABLE IF EXISTS %s`, quoteIdent(coll.tableName))
	if _, err := d.db.Exec(ctx, dropDDL); err != nil {
		return NewSystemError(err)
	}

	if err := d.deleteCollectionMeta(ctx, name); err != nil {
		return err
	}

	d.mu.Lock()
	delete(d.collections, name)
	d.mu.Unlock()

	return nil
}

func (d *Database) dependentCollections(name string) []string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	seen := map[string]bool{}
	var deps []string
	for _, coll := range d.collections {
		if coll == nil || coll.name == name {
			continue
		}
		for _, f := range coll.Fields() {
			if !isRelationField(f) {
				continue
			}
			ro := GetRelationOptions(f)
			if ro.Target == name && !seen[coll.name] {
				seen[coll.name] = true
				deps = append(deps, coll.name)
			}
		}
	}
	return deps
}

func (d *Database) RegisterFieldType(typeName string, factory FieldFactory) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.fieldTypes[typeName] = factory
	defaultFieldFactories[FieldType(typeName)] = factory
}

func (d *Database) RegisterOperator(op string, fn OperatorFunc) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.operators[op] = fn
	operators[op] = fn
}

func (d *Database) TablePrefix() string {
	return d.options.TablePrefix
}

func (d *Database) DB() DB {
	return d.db
}

func (d *Database) SqlDB() *sql.DB {
	return d.sqlDB
}

func (d *Database) Yaegi() YaegiManager {
	return d.yaegi
}

func (d *Database) SetYaegi(m YaegiManager) {
	d.yaegi = m
}

func (d *Database) Validator() *CELValidator {
	return d.validator
}

var _ = regexp.MatchString
var _ = strings.Builder{}
