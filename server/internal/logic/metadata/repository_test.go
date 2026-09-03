package metadata

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepository_Create(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()
	coll := createBasicCollection(t, db, "crud_create")
	repo := coll.Repository()

	t.Run("create single record succeeds", func(t *testing.T) {
		r, err := repo.Create(ctx, &CreateOptions{
			Values: map[string]interface{}{
				"title":  "第一条数据",
				"age":    25,
				"status": "published",
			},
		})
		a.NoError(err)
		a.NotNil(r)
		a.NotZero(r.Id())
		a.Equal("第一条数据", r.Get("title"))
	})

	t.Run("required field missing returns validation error", func(t *testing.T) {
		_, err := repo.Create(ctx, &CreateOptions{
			Values: map[string]interface{}{"age": 20},
		})
		a.Error(err)
		var vErr *ValidationError
		a.ErrorAs(err, &vErr)
	})

	t.Run("create many records", func(t *testing.T) {
		rs := seedData(t, repo, 5)
		a.Len(rs, 5)
	})
}

func TestRepository_FindAndCount(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()
	coll := createBasicCollection(t, db, "crud_find")
	repo := coll.Repository()
	seedData(t, repo, 10)

	t.Run("find all with pagination", func(t *testing.T) {
		list, total, err := repo.FindAndCount(ctx, &FindOptions{
			Page:     1,
			PageSize: 3,
		})
		a.NoError(err)
		a.Equal(10, total)
		a.Len(list, 3)
	})

	t.Run("find one by id", func(t *testing.T) {
		created, err := repo.Create(ctx, &CreateOptions{
			Values: map[string]interface{}{"title": "single", "age": 99, "status": "draft"},
		})
		a.NoError(err)
		r, err := repo.FindOne(ctx, &FindOneOptions{FilterByTk: created.Id()})
		a.NoError(err)
		a.Equal("single", r.Get("title"))
	})

	t.Run("find with eq filter", func(t *testing.T) {
		list, err := repo.Find(ctx, &FindOptions{
			CommonOptions: CommonOptions{Filter: Filter{"status": "published"}},
			PageSize:      100,
		})
		a.NoError(err)
		for _, r := range list {
			a.Equal("published", r.Get("status"))
		}
	})

	t.Run("find with $gt filter", func(t *testing.T) {
		list, err := repo.Find(ctx, &FindOptions{
			CommonOptions: CommonOptions{Filter: Filter{"age": map[string]interface{}{"$gt": 22}}},
			PageSize:      100,
		})
		a.NoError(err)
		for _, r := range list {
			a.GreaterOrEqual(forceInt(r.Get("age")), 23)
		}
	})

	t.Run("sort ascending", func(t *testing.T) {
		list, err := repo.Find(ctx, &FindOptions{
			CommonOptions: CommonOptions{Sort: Sort{"age"}},
			PageSize:      100,
		})
		a.NoError(err)
		for i := 1; i < len(list); i++ {
			a.LessOrEqual(forceInt(list[i-1].Get("age")), forceInt(list[i].Get("age")))
		}
	})

	t.Run("sort descending", func(t *testing.T) {
		list, err := repo.Find(ctx, &FindOptions{
			CommonOptions: CommonOptions{Sort: Sort{"-age"}},
			PageSize:      100,
		})
		a.NoError(err)
		for i := 1; i < len(list); i++ {
			a.GreaterOrEqual(forceInt(list[i-1].Get("age")), forceInt(list[i].Get("age")))
		}
	})

	t.Run("count", func(t *testing.T) {
		n, err := repo.Count(ctx, &CountOptions{
			CommonOptions: CommonOptions{Filter: Filter{"status": "draft"}},
		})
		a.NoError(err)
		a.GreaterOrEqual(n, 0)
	})
}

func TestRepository_Update(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()
	coll := createBasicCollection(t, db, "crud_update")
	repo := coll.Repository()

	created, err := repo.Create(ctx, &CreateOptions{
		Values: map[string]interface{}{"title": "原始标题", "age": 20, "status": "draft"},
	})
	a.NoError(err)

	t.Run("update single record by id", func(t *testing.T) {
		updated, affected, err := repo.Update(ctx, &UpdateOptions{
			FilterByTk: created.Id(),
			Values:     map[string]interface{}{"title": "更新后标题", "age": 30},
		})
		a.NoError(err)
		a.Equal(1, affected)
		a.NotNil(updated)
		a.Equal("更新后标题", updated.Get("title"))
	})
}

func TestRepository_Destroy(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()
	coll := createBasicCollection(t, db, "crud_del")
	repo := coll.Repository()

	created, err := repo.Create(ctx, &CreateOptions{Values: map[string]interface{}{"title": "待删除"}})
	a.NoError(err)

	t.Run("destroy by id", func(t *testing.T) {
		affected, err := repo.Destroy(ctx, &DestroyOptions{FilterByTk: created.Id()})
		a.NoError(err)
		a.Equal(1, affected)

		r, err := repo.FindOne(ctx, &FindOneOptions{FilterByTk: created.Id()})
		// FindOne returns NotFoundError when not found
		a.Error(err)
		var nfErr *NotFoundError
		a.ErrorAs(err, &nfErr)
		a.Nil(r)
	})

	t.Run("bulk destroy with filter", func(t *testing.T) {
		seedData(t, repo, 5)
		affected, err := repo.Destroy(ctx, &DestroyOptions{
			CommonOptions: CommonOptions{Filter: Filter{"status": "draft"}},
		})
		a.NoError(err)
		a.GreaterOrEqual(affected, 0)
	})

	t.Run("truncate", func(t *testing.T) {
		_, err := repo.Destroy(ctx, &DestroyOptions{Truncate: true})
		a.NoError(err)
		n, _ := repo.Count(ctx, &CountOptions{})
		a.Equal(0, n)
	})
}

func TestRepository_Transaction(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()
	coll := createBasicCollection(t, db, "crud_tx")
	repo := coll.Repository()

	t.Run("tx commit", func(t *testing.T) {
		before, _ := repo.Count(ctx, &CountOptions{})
		err := repo.Transaction(ctx, func(tx Repository) error {
			tx.Create(ctx, &CreateOptions{Values: map[string]interface{}{"title": "tx1"}})
			tx.Create(ctx, &CreateOptions{Values: map[string]interface{}{"title": "tx2"}})
			return nil
		})
		a.NoError(err)
		after, _ := repo.Count(ctx, &CountOptions{})
		a.Equal(before+2, after)
	})

	t.Run("tx rollback", func(t *testing.T) {
		before, _ := repo.Count(ctx, &CountOptions{})
		err := repo.Transaction(ctx, func(tx Repository) error {
			tx.Create(ctx, &CreateOptions{Values: map[string]interface{}{"title": "will_rollback"}})
			return assert.AnError
		})
		a.Error(err)
		after, _ := repo.Count(ctx, &CountOptions{})
		a.Equal(before, after)
	})
}

// forceInt coerces sqlite numeric values (int64/int/float64) to int for comparisons.
func forceInt(v interface{}) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	}
	return 0
}
