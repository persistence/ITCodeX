package metadata

import (
	"context"
	"reflect"
)

// YaegiDB exposes limited database operations to Yaegi scripts (no *sql.DB).
type YaegiDB struct {
	db *Database
}

func NewYaegiDB(db *Database) *YaegiDB {
	return &YaegiDB{db: db}
}

func (y *YaegiDB) Collection(name string) *YaegiRepository {
	coll := y.db.Collection(name)
	if coll == nil {
		return nil
	}
	return &YaegiRepository{repo: coll.Repository()}
}

func (y *YaegiDB) HasCollection(name string) bool {
	return y.db.HasCollection(name)
}

type YaegiRepository struct {
	repo Repository
}

func (r *YaegiRepository) Find(filter map[string]interface{}) ([]map[string]interface{}, error) {
	opts := &FindOptions{CommonOptions: CommonOptions{Filter: Filter(filter)}, PageSize: MaxPageSize}
	records, err := r.repo.Find(context.Background(), opts)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(records))
	for _, rec := range records {
		out = append(out, rec.Data())
	}
	return out, nil
}

func (r *YaegiRepository) FindOne(filter map[string]interface{}) (map[string]interface{}, error) {
	opts := &FindOneOptions{CommonOptions: CommonOptions{Filter: Filter(filter)}}
	rec, err := r.repo.FindOne(context.Background(), opts)
	if err != nil {
		return nil, err
	}
	return rec.Data(), nil
}

func (r *YaegiRepository) Create(values map[string]interface{}) (map[string]interface{}, error) {
	rec, err := r.repo.Create(context.Background(), &CreateOptions{Values: values})
	if err != nil {
		return nil, err
	}
	return rec.Data(), nil
}

func (r *YaegiRepository) Update(id interface{}, values map[string]interface{}) (map[string]interface{}, error) {
	rec, _, err := r.repo.Update(context.Background(), &UpdateOptions{FilterByTk: id, Values: values})
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, nil
	}
	return rec.Data(), nil
}

func (r *YaegiRepository) Destroy(id interface{}) error {
	_, err := r.repo.Destroy(context.Background(), &DestroyOptions{FilterByTk: id})
	return err
}

func (m *DefaultYaegiManager) buildExports() map[string]map[string]reflect.Value {
	y := NewYaegiDB(m.db)
	exports := map[string]map[string]reflect.Value{
		"itcodex/metadata": {
			"GetDB":      reflect.ValueOf(func() *YaegiDB { return y }),
			"NewYaegiDB": reflect.ValueOf(NewYaegiDB),
		},
	}
	return exports
}
