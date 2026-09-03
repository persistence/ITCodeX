package metadata

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	_ "modernc.org/sqlite"

	modelmd "itcodex/server/internal/model/metadata"
	"itcodex/server/pkg/utils"
	yaegictx "itcodex/server/pkg/yaegi/context"
)

type DB interface {
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row
	Close() error
}

type sqliteDB struct {
	db *sql.DB
}

func (s *sqliteDB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return s.db.ExecContext(ctx, query, args...)
}

func (s *sqliteDB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, query, args...)
}

func (s *sqliteDB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return s.db.QueryRowContext(ctx, query, args...)
}

func (s *sqliteDB) Close() error {
	return s.db.Close()
}

type YaegiManager interface {
	LoadScript(script *modelmd.YaegiScript) error
	DisableScript(id int64) error
	FindCustomAPI(method, path string) *modelmd.YaegiScript
	ExecuteBeforeCreate(ctx context.Context, coll *Collection, data map[string]interface{}) (map[string]interface{}, error)
	ExecuteAfterCreate(ctx context.Context, coll *Collection, record *Record) error
	ExecuteBeforeUpdate(ctx context.Context, coll *Collection, data map[string]interface{}, filter Filter) (map[string]interface{}, error)
	ExecuteAfterUpdate(ctx context.Context, coll *Collection, records []*Record) error
	ExecuteBeforeDelete(ctx context.Context, coll *Collection, filter Filter) error
	ExecuteAfterDelete(ctx context.Context, coll *Collection, affected int) error
	ExecuteCustomAPI(script *modelmd.YaegiScript, ctx *yaegictx.YaegiHTTPContext) error
	ValidateScript(content string) error
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

	dsn := ""
	if opts.StoragePath == "" || opts.StoragePath == ":memory:" {
		dsn = "file::memory:?cache=shared&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	} else {
		dsn = fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", opts.StoragePath)
	}

	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return nil, err
	}

	d.sqlDB = sqlDB
	d.db = &sqliteDB{db: sqlDB}

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
	return nil
}

func (d *Database) createSystemTables(ctx context.Context) error {
	prefix := d.TablePrefix()

	ddlStatements := []string{
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS "%s_collections" (
				"id" INTEGER PRIMARY KEY AUTOINCREMENT,
				"name" VARCHAR(255) NOT NULL UNIQUE,
				"display_name" VARCHAR(255) NOT NULL DEFAULT '',
				"type" VARCHAR(50) NOT NULL DEFAULT 'general',
				"options" TEXT,
				"created_at" DATETIME,
				"updated_at" DATETIME
			)
		`, prefix),
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS "%s_fields" (
				"id" INTEGER PRIMARY KEY AUTOINCREMENT,
				"collection_name" VARCHAR(255) NOT NULL,
				"name" VARCHAR(255) NOT NULL,
				"type" VARCHAR(50) NOT NULL,
				"display_name" VARCHAR(255) NOT NULL DEFAULT '',
				"is_required" BOOLEAN NOT NULL DEFAULT 0,
				"is_unique" BOOLEAN NOT NULL DEFAULT 0,
				"is_indexed" BOOLEAN NOT NULL DEFAULT 0,
				"validation" TEXT,
				"options" TEXT,
				"sort" INTEGER NOT NULL DEFAULT 0,
				"created_at" DATETIME,
				UNIQUE("collection_name", "name")
			)
		`, prefix),
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS "%s_indexes" (
				"id" INTEGER PRIMARY KEY AUTOINCREMENT,
				"collection_name" VARCHAR(255) NOT NULL,
				"name" VARCHAR(255) NOT NULL,
				"fields" TEXT NOT NULL,
				"unique" BOOLEAN NOT NULL DEFAULT 0,
				"options" TEXT,
				"created_at" DATETIME
			)
		`, prefix),
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS "%s_yaegi_scripts" (
				"id" INTEGER PRIMARY KEY AUTOINCREMENT,
				"collection_name" VARCHAR(255),
				"name" VARCHAR(255) NOT NULL,
				"hook_point" VARCHAR(50) NOT NULL,
				"content" TEXT NOT NULL,
				"api_path" VARCHAR(255),
				"http_method" VARCHAR(20),
				"enabled" BOOLEAN NOT NULL DEFAULT 1,
				"priority" INTEGER NOT NULL DEFAULT 0,
				"options" TEXT,
				"created_at" DATETIME,
				"updated_at" DATETIME
			)
		`, prefix),
	}

	for _, ddl := range ddlStatements {
		if _, err := d.db.Exec(ctx, ddl); err != nil {
			return NewSystemError(err)
		}
	}

	return nil
}

func (d *Database) loadCollections(ctx context.Context) error {
	prefix := d.TablePrefix()
	query := fmt.Sprintf(`SELECT id, name, display_name, type, options, created_at, updated_at FROM "%s_collections"`, prefix)
	rows, err := d.db.Query(ctx, query)
	if err != nil {
		return NewSystemError(err)
	}
	defer rows.Close()

	var collections []*modelmd.Collection
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
	prefix := d.TablePrefix()
	query := fmt.Sprintf(`SELECT id, collection_name, name, type, display_name, is_required, is_unique, is_indexed, validation, options, sort FROM "%s_fields" ORDER BY sort ASC, id ASC`, prefix)
	rows, err := d.db.Query(ctx, query)
	if err != nil {
		return NewSystemError(err)
	}
	defer rows.Close()

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
	for rows.Next() {
		var fm fieldModel
		if err := rows.Scan(&fm.Id, &fm.CollectionName, &fm.Name, &fm.Type, &fm.DisplayName, &fm.IsRequired, &fm.IsUnique, &fm.IsIndexed, &fm.Validation, &fm.Options, &fm.Sort); err != nil {
			return NewSystemError(err)
		}
		fieldModels = append(fieldModels, &fm)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	for _, fm := range fieldModels {
		coll, ok := d.collections[fm.CollectionName]
		if !ok {
			continue
		}

		opts := make(map[string]interface{})
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

func (d *Database) CreateCollection(ctx context.Context, input CreateCollectionInput) (*Collection, error) {
	if !collectionNameRegex.MatchString(input.Name) {
		return nil, fmt.Errorf("无效的集合名: %s", input.Name)
	}

	if d.HasCollection(input.Name) {
		return nil, NewAlreadyExistsError("集合", "name", input.Name)
	}

	coll := newCollection(d, input.Name,
		WithDisplayName(input.DisplayName),
		WithType(input.Type),
	)
	if input.Options.Extra != nil {
		coll.opts = input.Options.Extra
	}

	autoGenId := true
	if input.AutoGenId != nil {
		autoGenId = *input.AutoGenId
	}

	presetFields := input.PresetFields
	if len(presetFields) == 0 {
		presetFields = []string{"id", "createdAt", "updatedAt"}
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

	for _, fi := range input.Fields {
		opts := make(map[string]interface{})
		opts["name"] = fi.Name
		opts["displayName"] = fi.DisplayName
		opts["required"] = fi.IsRequired
		opts["unique"] = fi.IsUnique
		opts["indexed"] = fi.IsUnique
		opts["isSystem"] = fi.IsSystem
		opts["length"] = fi.Length
		if fi.Options != nil {
			for k, v := range fi.Options {
				opts[k] = v
			}
		}
		f, err := NewField(coll, fi.Type, opts)
		if err != nil {
			return nil, err
		}
		coll.addFieldInternal(f)
	}

	for _, idx := range input.Indexes {
		idxCopy := idx
		coll.indexes = append(coll.indexes, &idxCopy)
	}

	if input.TableValidation != nil {
		coll.SetTableValidation(input.TableValidation)
	}

	ddl := coll.generateDDL()
	if _, err := d.db.Exec(ctx, ddl); err != nil {
		return nil, NewSystemError(err)
	}

	optionsJson, _ := json.Marshal(coll.opts)
	prefix := d.TablePrefix()
	_, err := d.db.Exec(ctx, fmt.Sprintf(`INSERT INTO "%s_collections" (name, display_name, type, options) VALUES (?, ?, ?, ?)`, prefix),
		coll.name, coll.displayName, string(coll.type_), string(optionsJson))
	if err != nil {
		return nil, NewSystemError(err)
	}

	for _, f := range coll.Fields() {
		fOpts := f.Options()
		optsJson, _ := json.Marshal(fOpts)
		_, err := d.db.Exec(ctx, fmt.Sprintf(`INSERT INTO "%s_fields" (collection_name, name, type, display_name, is_required, is_unique, is_indexed, options) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, prefix),
			coll.name, f.Name(), f.Type(), f.DisplayName(), f.IsRequired(), f.IsUnique(), f.IsUnique(), string(optsJson))
		if err != nil {
			return nil, NewSystemError(err)
		}
	}

	coll.isNew = false

	d.mu.Lock()
	d.collections[coll.name] = coll
	d.mu.Unlock()

	return coll, nil
}

func (d *Database) DropCollection(ctx context.Context, name string) error {
	d.mu.Lock()
	coll, ok := d.collections[name]
	if !ok {
		d.mu.Unlock()
		return NewNotFoundError("集合", "name", name)
	}
	d.mu.Unlock()

	dropDDL := fmt.Sprintf(`DROP TABLE IF EXISTS "%s"`, coll.tableName)
	if _, err := d.db.Exec(ctx, dropDDL); err != nil {
		return NewSystemError(err)
	}

	prefix := d.TablePrefix()
	_, err := d.db.Exec(ctx, fmt.Sprintf(`DELETE FROM "%s_fields" WHERE collection_name = ?`, prefix), name)
	if err != nil {
		return NewSystemError(err)
	}
	_, err = d.db.Exec(ctx, fmt.Sprintf(`DELETE FROM "%s_collections" WHERE name = ?`, prefix), name)
	if err != nil {
		return NewSystemError(err)
	}

	d.mu.Lock()
	delete(d.collections, name)
	d.mu.Unlock()

	return nil
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
