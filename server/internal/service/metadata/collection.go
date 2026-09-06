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
	opts            map[string]any
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

func (c *Collection) Options() map[string]any {
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
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE name = ?`, quoteIdent(prefix+"collections"))
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

	opts := make(map[string]any)
	opts["name"] = input.Name
	opts["displayName"] = input.DisplayName
	opts["required"] = input.IsRequired
	opts["unique"] = input.IsUnique
	opts["indexed"] = input.IsIndexed
	opts["isSystem"] = input.IsSystem
	opts["length"] = input.Length
	if input.Options != nil {
		for k, v := range input.Options {
			opts[k] = v
		}
	}
	attachFieldInputOptions(opts, input)
	attachRelationInput(opts, input)

	f, err := NewField(c, input.Type, opts)
	if err != nil {
		return err
	}

	virtual := isVirtualField(f)
	if c.db != nil && !c.isNew && !virtual && f.DDLColumn() != "" {
		ddl := c.alterAddColumnDDL(f)
		if _, err := c.db.db.Exec(ctx, ddl); err != nil {
			return NewSystemError(err)
		}
	}

	c.fields[input.Name] = f
	c.fieldOrder = append(c.fieldOrder, input.Name)

	if c.db != nil && !c.isNew {
		optionsJson, _ := json.Marshal(opts)
		query := fmt.Sprintf(`INSERT INTO %s (collection_name, name, type, display_name, is_required, is_unique, is_indexed, options, sort) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, quoteIdent(c.db.TablePrefix()+"fields"))
		_, err := c.db.db.Exec(ctx, query, c.name, input.Name, string(input.Type), input.DisplayName, input.IsRequired, input.IsUnique, input.IsIndexed, string(optionsJson), input.Sort)
		if err != nil {
			return NewSystemError(err)
		}
	}

	if FieldType(f.Type()) == FieldTypeBelongsToMany {
		ro := GetRelationOptions(f)
		if err := ensureThroughTableDDL(ctx, c.db, ro.Through, ro.ForeignKey, ro.OtherKey); err != nil {
			return NewSystemError(err)
		}
	}

	if !virtual && (input.IsIndexed || input.IsUnique) {
		if err := c.addIndexLocked(ctx, &Index{
			Fields: []string{input.Name},
			Unique: input.IsUnique,
		}); err != nil {
			return err
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
		_, err := c.db.db.Exec(ctx, fmt.Sprintf(`ALTER TABLE %s DROP COLUMN %s`, quoteIdent(c.tableName), quoteIdent(name)))
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
		query := fmt.Sprintf(`DELETE FROM %s WHERE collection_name = ? AND name = ?`, quoteIdent(c.db.TablePrefix()+"fields"))
		_, err := c.db.db.Exec(ctx, query, c.name, name)
		if err != nil {
			return NewSystemError(err)
		}
	}

	return nil
}

func (c *Collection) Indexes() []*Index {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]*Index, len(c.indexes))
	copy(out, c.indexes)
	return out
}

func (c *Collection) AddIndex(ctx context.Context, index *Index) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.addIndexLocked(ctx, index)
}

// addIndexLocked creates the index DDL, appends to memory, and persists metadata.
// Caller must hold c.mu.
func (c *Collection) addIndexLocked(ctx context.Context, index *Index) error {
	if index.Name == "" {
		index.Name = fmt.Sprintf("idx_%s_%s", c.name, strings.Join(index.Fields, "_"))
	}

	var b strings.Builder
	b.WriteString("CREATE ")
	if index.Unique {
		b.WriteString("UNIQUE ")
	}
	b.WriteString("INDEX ")
	b.WriteString(quoteIdent(index.Name))
	b.WriteString(" ON ")
	b.WriteString(quoteIdent(c.tableName))
	b.WriteString(" (")
	for i, f := range index.Fields {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(quoteIdent(f))
	}
	b.WriteString(")")

	if c.db != nil && !c.isNew {
		if _, err := c.db.db.Exec(ctx, b.String()); err != nil {
			return NewSystemError(err)
		}
		if err := c.persistIndex(ctx, index); err != nil {
			return err
		}
	}

	c.indexes = append(c.indexes, index)
	return nil
}

func (c *Collection) persistIndex(ctx context.Context, index *Index) error {
	if c.db == nil || c.isNew {
		return nil
	}
	fieldsJson, _ := json.Marshal(index.Fields)
	query := fmt.Sprintf(
		`INSERT INTO %s (collection_name, name, fields, %s) VALUES (?, ?, ?, ?)`,
		quoteIdent(c.db.TablePrefix()+"indexes"),
		quoteIdent("unique"),
	)
	_, err := c.db.db.Exec(ctx, query, c.name, index.Name, string(fieldsJson), index.Unique)
	if err != nil {
		return NewSystemError(err)
	}
	return nil
}

func (c *Collection) RemoveIndex(ctx context.Context, fields []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, idx := range c.indexes {
		if stringSliceEqual(idx.Fields, fields) {
			if c.db != nil && !c.isNew {
				_, err := c.db.db.Exec(ctx, fmt.Sprintf(`DROP INDEX %s ON %s`, quoteIdent(idx.Name), quoteIdent(c.tableName)))
				if err != nil {
					return NewSystemError(err)
				}
				del := fmt.Sprintf(`DELETE FROM %s WHERE collection_name = ? AND name = ?`, quoteIdent(c.db.TablePrefix()+"indexes"))
				if _, err := c.db.db.Exec(ctx, del, c.name, idx.Name); err != nil {
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
	b.WriteString("CREATE TABLE IF NOT EXISTS ")
	b.WriteString(quoteIdent(c.tableName))
	b.WriteString(" (")

	first := true
	for _, f := range c.Fields() {
		if isVirtualField(f) || f.DDLColumn() == "" {
			continue
		}
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
		b.WriteString(", UNIQUE (")
		for i, field := range idx.Fields {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(quoteIdent(field))
		}
		b.WriteString(")")
	}

	b.WriteString(") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4")
	return b.String()
}

func (c *Collection) UpdateMeta(ctx context.Context, input UpdateCollectionInput) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if input.DisplayName != "" {
		c.displayName = input.DisplayName
	}
	if c.opts == nil {
		c.opts = make(map[string]any)
	}
	if input.Description != "" {
		c.opts["description"] = input.Description
	}
	if input.Categories != nil {
		c.opts["categories"] = input.Categories
	}
	if input.Options.Extra != nil {
		for k, v := range input.Options.Extra {
			c.opts[k] = v
		}
	}
	if input.Options.SimplePagination {
		c.opts["simplePagination"] = true
	}

	if c.db != nil && !c.isNew {
		optionsJson, _ := json.Marshal(c.opts)
		query := fmt.Sprintf(`UPDATE %s SET display_name = ?, options = ? WHERE name = ?`, quoteIdent(c.db.TablePrefix()+"collections"))
		if _, err := c.db.db.Exec(ctx, query, c.displayName, string(optionsJson), c.name); err != nil {
			return NewSystemError(err)
		}
	}
	return nil
}

func (c *Collection) UpdateField(ctx context.Context, name string, input UpdateFieldInput) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	f, exists := c.fields[name]
	if !exists {
		return NewNotFoundError("字段", "name", name)
	}

	opts := f.Options()
	if opts == nil {
		opts = make(map[string]any)
	}
	if input.DisplayName != "" {
		opts["displayName"] = input.DisplayName
	}
	if input.IsRequired != nil {
		opts["required"] = *input.IsRequired
	}
	if input.IsUnique != nil {
		opts["unique"] = *input.IsUnique
	}
	if input.IsIndexed != nil {
		opts["indexed"] = *input.IsIndexed
	}
	if input.Options != nil {
		for k, v := range input.Options {
			opts[k] = v
		}
	}
	if input.Validation != nil {
		opts["validation"] = input.Validation
	}
	f.SetOptions(opts)

	replaced, err := NewField(c, FieldType(f.Type()), opts)
	if err != nil {
		return err
	}
	c.fields[name] = replaced

	if c.db != nil && !c.isNew {
		optionsJson, _ := json.Marshal(opts)
		displayName := replaced.DisplayName()
		query := fmt.Sprintf(`UPDATE %s SET display_name = ?, is_required = ?, is_unique = ?, is_indexed = ?, options = ? WHERE collection_name = ? AND name = ?`, quoteIdent(c.db.TablePrefix()+"fields"))
		if _, err := c.db.db.Exec(ctx, query, displayName, replaced.IsRequired(), replaced.IsUnique(), replaced.IsIndexed(), string(optionsJson), c.name, name); err != nil {
			return NewSystemError(err)
		}
	}
	return nil
}

func (c *Collection) Sync(ctx context.Context) error {
	if c.db == nil {
		return nil
	}
	if _, err := c.db.db.Exec(ctx, c.generateDDL()); err != nil {
		return NewSystemError(err)
	}

	existing, err := c.existingColumnNames(ctx)
	if err != nil {
		return err
	}
	for _, f := range c.Fields() {
		if isVirtualField(f) || f.DDLColumn() == "" {
			continue
		}
		if existing[f.Name()] {
			continue
		}
		if _, err := c.db.db.Exec(ctx, c.alterAddColumnDDL(f)); err != nil {
			return NewSystemError(err)
		}
	}
	return nil
}

func (c *Collection) existingColumnNames(ctx context.Context) (map[string]bool, error) {
	query := `SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?`
	rows, err := c.db.db.Query(ctx, query, c.tableName)
	if err != nil {
		return nil, NewSystemError(err)
	}
	defer rows.Close()

	out := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, NewSystemError(err)
		}
		out[name] = true
	}
	return out, nil
}

func (c *Collection) alterAddColumnDDL(f Field) string {
	return fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s`, quoteIdent(c.tableName), f.DDLColumn())
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

func newCollection(db *Database, name string, opts ...CollectionOption) *Collection {
	c := &Collection{
		name:       name,
		type_:      CollectionTypeGeneral,
		fields:     make(map[string]Field),
		fieldOrder: make([]string, 0),
		opts:       make(map[string]any),
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

func WithOptions(opts map[string]any) CollectionOption {
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
		opts:        make(map[string]any),
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
