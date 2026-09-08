package metadata

import (
	"context"
	"fmt"
	"reflect"
)

// YaegiDB exposes limited database operations to Yaegi scripts (no *sql.DB).
type YaegiDB struct {
	db  *Database
	ctx context.Context
}

func NewYaegiDB(db *Database) *YaegiDB {
	return &YaegiDB{db: db, ctx: context.Background()}
}

func (y *YaegiDB) hookCtx() context.Context {
	if y != nil && y.ctx != nil {
		return y.ctx
	}
	return context.Background()
}

func (y *YaegiDB) Collection(name string) *YaegiRepository {
	coll := y.db.Collection(name)
	if coll == nil {
		return nil
	}
	return &YaegiRepository{repo: coll.Repository(), ctx: y.hookCtx()}
}

func (y *YaegiDB) HasCollection(name string) bool {
	return y.db.HasCollection(name)
}

func (y *YaegiDB) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if ctx == nil {
		ctx = y.hookCtx()
	}
	if WriteUnitFromContext(ctx) != nil {
		return fn(ctx)
	}
	colls := y.db.Collections()
	if len(colls) == 0 {
		return fn(ctx)
	}
	return colls[0].Repository().Transaction(ctx, func(tx Repository) error {
		return fn(ctx)
	})
}

type YaegiRepository struct {
	repo Repository
	ctx  context.Context
}

func (r *YaegiRepository) repoCtx() context.Context {
	if r != nil && r.ctx != nil {
		return r.ctx
	}
	return context.Background()
}

func (r *YaegiRepository) Find(filter map[string]any) ([]map[string]any, error) {
	opts := &FindOptions{CommonOptions: CommonOptions{Filter: Filter(filter)}, PageSize: MaxPageSize}
	records, err := r.repo.Find(r.repoCtx(), opts)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(records))
	for _, rec := range records {
		out = append(out, rec.Data())
	}
	return out, nil
}

func (r *YaegiRepository) FindOne(filter map[string]any) (map[string]any, error) {
	opts := &FindOneOptions{CommonOptions: CommonOptions{Filter: Filter(filter)}}
	rec, err := r.repo.FindOne(r.repoCtx(), opts)
	if err != nil {
		return nil, err
	}
	return rec.Data(), nil
}

func (r *YaegiRepository) FindByID(id any) (map[string]any, error) {
	rec, err := r.repo.FindOne(r.repoCtx(), &FindOneOptions{FilterByTk: id})
	if err != nil {
		return nil, err
	}
	return rec.Data(), nil
}

func (r *YaegiRepository) Count(filter map[string]any) (int, error) {
	return r.repo.Count(r.repoCtx(), &CountOptions{CommonOptions: CommonOptions{Filter: Filter(filter)}})
}

func (r *YaegiRepository) FindAndCount(filter map[string]any) ([]map[string]any, int, error) {
	records, total, err := r.repo.FindAndCount(r.repoCtx(), &FindOptions{
		CommonOptions: CommonOptions{Filter: Filter(filter)},
		PageSize:      MaxPageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	out := make([]map[string]any, 0, len(records))
	for _, rec := range records {
		out = append(out, rec.Data())
	}
	return out, total, nil
}

func (r *YaegiRepository) Create(values map[string]any) (map[string]any, error) {
	rec, err := r.repo.Create(r.repoCtx(), &CreateOptions{Values: values})
	if err != nil {
		return nil, err
	}
	return rec.Data(), nil
}

func (r *YaegiRepository) CreateMany(records []map[string]any) ([]map[string]any, error) {
	created, err := r.repo.CreateMany(r.repoCtx(), &CreateManyOptions{Records: records})
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(created))
	for _, rec := range created {
		out = append(out, rec.Data())
	}
	return out, nil
}

func (r *YaegiRepository) Update(id any, values map[string]any) (map[string]any, error) {
	return r.UpdateByID(id, values)
}

func (r *YaegiRepository) UpdateByID(id any, values map[string]any) (map[string]any, error) {
	rec, _, err := r.repo.Update(r.repoCtx(), &UpdateOptions{FilterByTk: id, Values: values})
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, nil
	}
	return rec.Data(), nil
}

func (r *YaegiRepository) UpdateWhere(filter map[string]any, values map[string]any) (int, error) {
	_, n, err := r.repo.Update(r.repoCtx(), &UpdateOptions{
		CommonOptions: CommonOptions{Filter: Filter(filter)},
		Values:        values,
	})
	return n, err
}

func (r *YaegiRepository) Destroy(id any) error {
	return r.DeleteByID(id)
}

func (r *YaegiRepository) DeleteByID(id any) error {
	_, err := r.repo.Destroy(r.repoCtx(), &DestroyOptions{FilterByTk: id})
	return err
}

func (r *YaegiRepository) Delete(filter map[string]any) (int, error) {
	if len(filter) == 0 {
		return 0, fmt.Errorf("Delete 必须提供 filter")
	}
	return r.repo.Destroy(r.repoCtx(), &DestroyOptions{CommonOptions: CommonOptions{Filter: Filter(filter)}})
}

func (r *YaegiRepository) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if ctx == nil {
		ctx = r.repoCtx()
	}
	return r.repo.Transaction(ctx, func(tx Repository) error {
		return fn(ctx)
	})
}

func (m *DefaultYaegiManager) buildExports() map[string]map[string]reflect.Value {
	return map[string]map[string]reflect.Value{
		"itcodex/metadata/metadata": {
			"GetDB": reflect.ValueOf(func(ctx context.Context) *YaegiDB {
				return &YaegiDB{db: m.db, ctx: ctx}
			}),
			"Collection": reflect.ValueOf(func(ctx context.Context, name string) *YaegiRepository {
				return (&YaegiDB{db: m.db, ctx: ctx}).Collection(name)
			}),
		},
		"itcodex/validation/validation": {
			"ValidateCEL": reflect.ValueOf(func(data map[string]any, expression string) (bool, error) {
				if m.db == nil || m.db.Validator() == nil {
					return false, fmt.Errorf("CEL validator unavailable")
				}
				v := m.db.Validator()
				prog, err := v.compile(expression)
				if err != nil {
					return false, err
				}
				out, err := v.evalProgram(prog, map[string]any{"data": data, "oldData": map[string]any{}})
				if err != nil {
					return false, err
				}
				b, ok := out.Value().(bool)
				if !ok {
					return false, fmt.Errorf("CEL 结果不是 bool")
				}
				return b, nil
			}),
			"NewValidationError": reflect.ValueOf(func(msg string) error {
				e := NewValidationError()
				e.AddTableError(msg)
				return e
			}),
			"NewFieldValidationError": reflect.ValueOf(func(field, msg string) error {
				e := NewValidationError()
				e.AddFieldError(field, msg)
				return e
			}),
		},
	}
}
