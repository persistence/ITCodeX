package metadata

import (
	"context"
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

func (r *YaegiRepository) Create(values map[string]any) (map[string]any, error) {
	rec, err := r.repo.Create(r.repoCtx(), &CreateOptions{Values: values})
	if err != nil {
		return nil, err
	}
	return rec.Data(), nil
}

func (r *YaegiRepository) Update(id any, values map[string]any) (map[string]any, error) {
	rec, _, err := r.repo.Update(r.repoCtx(), &UpdateOptions{FilterByTk: id, Values: values})
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, nil
	}
	return rec.Data(), nil
}

func (r *YaegiRepository) Destroy(id any) error {
	_, err := r.repo.Destroy(r.repoCtx(), &DestroyOptions{FilterByTk: id})
	return err
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
			"NewYaegiDB": reflect.ValueOf(NewYaegiDB),
		},
	}
}
