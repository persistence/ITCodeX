package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	modelmd "itcodex/server/internal/model/metadata"
)

var collectionNameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

type Collection struct {
	mu              sync.RWMutex
	name            string
	displayName     string
	type_           CollectionType
	fields          map[string]Field
	fieldOrder      []string
	opts            map[string]interface{}
	tableName       string
	db              *Database
	repo            Repository
	indexes         []*Index
	tableValidation *TableValidationConfig
	isNew           bool
}

func (c *Collection) Name() string {
	return c.name
}

func (c *Collection) DisplayName() string {
	return c.displayName
}

func (c *Collection) Type() CollectionType {
	return c.type_
}

func (c *Collection) HasField(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.fields[name]
	return ok
}

func (c *Collection) GetField(name string) Field {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fields[name]
}

func (c *Collection) Fields() []Field {
	c.mu.RLock()
	defer c.mu.RUnlock()
	fields := make([]Field, 0, len(c.fieldOrder))
	for _, name := range c.fieldOrder {
		if f, ok := c.fields[name]; ok {
			fields = append(fields, f)
		}
	}
	return fields
}

func (c *Collection) FieldNames() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]string, len(c.fieldOrder))
	copy(result, c.fieldOrder)
	return result
}

func (c *Collection) TableName() string {
	return c.tableName
}

func (c *Collection) Options() map[string]interface{} {
	return c.opts
}

func (c *Collection) GetTableValidation() *TableValidationConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tableValidation
}

func (c *Collection) SetTableValidation(tv *TableValidationConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tableValidation = tv
}

func (c *Collection) Db() *Database {
	return c.db
}

func (c *Collection) Repository() Repository {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.repo == nil {
		c.repo = NewGenericRepository(c)
	}
	return c.repo
}

func (c *Collection) ExistsInDb() (bool, error) {
	if c.db == nil || c.db.db == nil {
		return false, nil
	}
	prefix := c.db.TablePrefix()
	query := fmt.Sprintf(`SELECT COUNT(*) FROM "%s_collections" WHERE name = ?`, prefix)
	var count int
	row := c.db.db.QueryRow(context.Background(), query, c.name)
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (c *Collection) AddField(ctx context.Context, input CreateFieldInput) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !collectionNameRegex.MatchString(input.Name) {
		return fmt.Errorf("无效的字段名: %s", input.Name)
	}

	if _, exists := c.fields[input.Name]; exists {
		return NewAlreadyExistsError("字段", "name", input.Name)
	}

	opts := make(map[string]interface{})
	opts["name"] = input.Name
	opts["displayName"] = input.DisplayName
	opts["required"] = input.IsRequired
	opts["unique"] = input.IsUnique
	opts["indexed"] = input.IsUnique
	opts["isSystem"] = input.IsSystem
	opts["length"] = input.Length
	if input.Options != nil {
		for k, v := range input.Options {
			opts[k] = v
		}
	}

	f, err := NewField(c, input.Type, opts)
	if err != nil {
		return err
	}

	if c.db != nil && !c.isNew {
		ddl := c.alterAddColumnDDL(f)
		if _, err := c.db.db.Exec(ctx, ddl); err != nil {
			return NewSystemError(err)
		}
	}

	c.fields[input.Name] = f
	c.fieldOrder = append(c.fieldOrder, input.Name)

	if c.db != nil && !c.isNew {
		optionsJson, _ := json.Marshal(opts)
		query := fmt.Sprintf(`INSERT INTO "%s_fields" (collection_name, name, type, display_name, is_required, is_unique, is_indexed, options, sort) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, c.db.TablePrefix())
		_, err := c.db.db.Exec(ctx, query, c.name, input.Name, string(input.Type), input.DisplayName, input.IsRequired, input.IsUnique, input.IsUnique, string(optionsJson), input.Sort)
		if err != nil {
			return NewSystemError(err)
		}
	}

	return nil
}

func (c *Collection) RemoveField(ctx context.Context, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	f, exists := c.fields[name]
	if !exists {
		return NewNotFoundError("字段", "name", name)
	}

	if f.IsSystem() {
		return NewForbiddenError("系统字段不可删除")
	}

	if c.db != nil && !c.isNew {
		_, err := c.db.db.Exec(ctx, fmt.Sprintf(`ALTER TABLE "%s" DROP COLUMN "%s"`, c.tableName, name))
		if err != nil {
			return NewSystemError(err)
		}
	}

	delete(c.fields, name)
	for i, fn := range c.fieldOrder {
		if fn == name {
			c.fieldOrder = append(c.fieldOrder[:i], c.fieldOrder[i+1:]...)
			break
		}
	}

	if c.db != nil && !c.isNew {
		query := fmt.Sprintf(`DELETE FROM "%s_fields" WHERE collection_name = ? AND name = ?`, c.db.TablePrefix())
		_, err := c.db.db.Exec(ctx, query, c.name, name)
		if err != nil {
			return NewSystemError(err)
		}
	}

	return nil
}

func (c *Collection) AddIndex(ctx context.Context, index *Index) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if index.Name == "" {
		index.Name = fmt.Sprintf("idx_%s_%s", c.name, strings.Join(index.Fields, "_"))
	}

	var b strings.Builder
	b.WriteString("CREATE ")
	if index.Unique {
		b.WriteString("UNIQUE ")
	}
	b.WriteString("INDEX ")
	b.WriteString(`"`)
	b.WriteString(index.Name)
	b.WriteString(`" ON "`)
	b.WriteString(c.tableName)
	b.WriteString(`" (`)
	for i, f := range index.Fields {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(`"`)
		b.WriteString(f)
		b.WriteString(`"`)
	}
	b.WriteString(")")

	if c.db != nil && !c.isNew {
		if _, err := c.db.db.Exec(ctx, b.String()); err != nil {
			return NewSystemError(err)
		}
	}

	c.indexes = append(c.indexes, index)
	return nil
}

func (c *Collection) RemoveIndex(ctx context.Context, fields []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, idx := range c.indexes {
		if stringSliceEqual(idx.Fields, fields) {
			if c.db != nil && !c.isNew {
				_, err := c.db.db.Exec(ctx, fmt.Sprintf(`DROP INDEX "%s"`, idx.Name))
				if err != nil {
					return NewSystemError(err)
				}
			}
			c.indexes = append(c.indexes[:i], c.indexes[i+1:]...)
			return nil
		}
	}
	return NewNotFoundError("索引", "fields", fields)
}

func (c *Collection) generateDDL() string {
	var b strings.Builder
	b.WriteString(`CREATE TABLE IF NOT EXISTS "`)
	b.WriteString(c.tableName)
	b.WriteString(`" (`)

	first := true
	for _, f := range c.Fields() {
		if !first {
			b.WriteString(", ")
		}
		first = false
		b.WriteString(f.DDLColumn())
	}

	for _, idx := range c.indexes {
		if !idx.Unique {
			continue
		}
		b.WriteString(", ")
		b.WriteString(`UNIQUE ("`)
		b.WriteString(strings.Join(idx.Fields, `", "`))
		b.WriteString(`")`)
	}

	b.WriteString(")")
	return b.String()
}

func (c *Collection) alterAddColumnDDL(f Field) string {
	return fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN %s`, c.tableName, f.DDLColumn())
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (c *Collection) addFieldInternal(f Field) {
	c.mu.Lock()
	defer c.mu.Unlock()
	name := f.Name()
	if _, exists := c.fields[name]; !exists {
		c.fields[name] = f
		c.fieldOrder = append(c.fieldOrder, name)
	}
}

type unimplementedRepository struct {
	coll *Collection
}

func (r *unimplementedRepository) Find(ctx context.Context, opts *FindOptions) ([]*Record, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *unimplementedRepository) FindOne(ctx context.Context, opts *FindOneOptions) (*Record, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *unimplementedRepository) FindAndCount(ctx context.Context, opts *FindOptions) ([]*Record, int, error) {
	return nil, 0, fmt.Errorf("not implemented")
}

func (r *unimplementedRepository) Count(ctx context.Context, opts *CountOptions) (int, error) {
	return 0, fmt.Errorf("not implemented")
}

func (r *unimplementedRepository) Create(ctx context.Context, opts *CreateOptions) (*Record, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *unimplementedRepository) CreateMany(ctx context.Context, opts *CreateManyOptions) ([]*Record, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *unimplementedRepository) Update(ctx context.Context, opts *UpdateOptions) (*Record, int, error) {
	return nil, 0, fmt.Errorf("not implemented")
}

func (r *unimplementedRepository) Destroy(ctx context.Context, opts *DestroyOptions) (int, error) {
	return 0, fmt.Errorf("not implemented")
}

func (r *unimplementedRepository) Transaction(ctx context.Context, fn func(tx Repository) error) error {
	return fmt.Errorf("not implemented")
}

func (r *unimplementedRepository) Collection() *Collection {
	return r.coll
}

func newCollection(db *Database, name string, opts ...CollectionOption) *Collection {
	c := &Collection{
		name:       name,
		type_:      CollectionTypeGeneral,
		fields:     make(map[string]Field),
		fieldOrder: make([]string, 0),
		opts:       make(map[string]interface{}),
		db:         db,
		isNew:      true,
	}
	if db != nil {
		c.tableName = db.TablePrefix() + name
	} else {
		c.tableName = name
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type CollectionOption func(c *Collection)

func WithDisplayName(displayName string) CollectionOption {
	return func(c *Collection) {
		c.displayName = displayName
	}
}

func WithType(t CollectionType) CollectionOption {
	return func(c *Collection) {
		c.type_ = t
	}
}

func WithOptions(opts map[string]interface{}) CollectionOption {
	return func(c *Collection) {
		c.opts = opts
	}
}

func (d *Database) buildCollectionFromModel(m *modelmd.Collection) (*Collection, error) {
	c := &Collection{
		name:        m.Name,
		displayName: m.DisplayName,
		type_:       CollectionType(m.Type),
		fields:      make(map[string]Field),
		fieldOrder:  make([]string, 0),
		opts:        make(map[string]interface{}),
		tableName:   d.TablePrefix() + m.Name,
		db:          d,
		isNew:       false,
	}

	if m.Options != "" {
		if err := json.Unmarshal([]byte(m.Options), &c.opts); err != nil {
			return nil, err
		}
	}

	return c, nil
}
