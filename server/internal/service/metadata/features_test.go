package metadata

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	modelmd "itcodex/server/internal/model/metadata"
	"itcodex/server/pkg/utils"
)

func TestFilter_BuildWhereClause(t *testing.T) {
	a := assert.New(t)
	var params []interface{}

	t.Run("empty filter returns 1=1", func(t *testing.T) {
		params = nil
		sql, err := BuildWhereClause(Filter{}, &params)
		a.NoError(err)
		a.Equal("1=1", sql)
	})

	t.Run("simple eq", func(t *testing.T) {
		params = nil
		sql, err := BuildWhereClause(Filter{"title": "hello"}, &params)
		a.NoError(err)
		a.Contains(sql, "`title` = ?")
		a.Len(params, 1)
	})

	t.Run("$gt operator", func(t *testing.T) {
		params = nil
		sql, err := BuildWhereClause(Filter{"age": map[string]interface{}{"$gt": 18}}, &params)
		a.NoError(err)
		a.Contains(sql, "`age` > ?")
	})

	t.Run("$in operator", func(t *testing.T) {
		params = nil
		sql, err := BuildWhereClause(Filter{"id": map[string]interface{}{"$in": []interface{}{1, 2, 3}}}, &params)
		a.NoError(err)
		a.Contains(sql, "IN")
		a.Len(params, 3)
	})

	t.Run("$like operator", func(t *testing.T) {
		params = nil
		sql, err := BuildWhereClause(Filter{"title": map[string]interface{}{"$like": "%test%"}}, &params)
		a.NoError(err)
		a.Contains(sql, "LIKE")
	})

	t.Run("$and operator", func(t *testing.T) {
		params = nil
		sql, err := BuildWhereClause(Filter{"$and": []Filter{
			{"a": 1},
			{"b": 2},
		}}, &params)
		a.NoError(err)
		a.Contains(sql, "AND")
		a.Len(params, 2)
	})

	t.Run("$or operator", func(t *testing.T) {
		params = nil
		sql, err := BuildWhereClause(Filter{"$or": []Filter{
			{"a": 1},
			{"b": 2},
		}}, &params)
		a.NoError(err)
		a.Contains(sql, "OR")
	})

	t.Run("$isNull operator", func(t *testing.T) {
		params = nil
		sql, err := BuildWhereClause(Filter{"deletedAt": map[string]interface{}{"$isNull": true}}, &params)
		a.NoError(err)
		a.Contains(sql, "IS NULL")
	})
}

func TestCELValidator_RequiredField(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()

	coll := createBasicCollection(t, db, "cel_req")
	v := db.Validator()

	t.Run("missing required field fails", func(t *testing.T) {
		err := v.ValidateRecord(ctx, coll, map[string]interface{}{"age": 20}, nil, false)
		a.Error(err)
		var vErr *ValidationError
		a.ErrorAs(err, &vErr)
		a.NotEmpty(vErr.FieldErrors["title"])
	})

	t.Run("valid data passes", func(t *testing.T) {
		err := v.ValidateRecord(ctx, coll, map[string]interface{}{"title": "ok", "age": 20, "status": "draft"}, nil, false)
		a.NoError(err)
	})
}

func TestCELValidator_MultiFieldRule(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()

	coll := createBasicCollection(t, db, "cel_multi")
	coll.SetTableValidation(&TableValidationConfig{
		Rules: []TableCELRule{
			{
				Name:         "年龄非负",
				Expression:   `data.age == null || data.age >= 0`,
				ErrorMessage: "年龄必须大于等于0",
			},
		},
	})
	v := db.Validator()

	t.Run("negative age fails table rule", func(t *testing.T) {
		err := v.ValidateRecord(ctx, coll, map[string]interface{}{"title": "t", "age": -1}, nil, false)
		a.Error(err)
	})

	t.Run("positive age passes", func(t *testing.T) {
		err := v.ValidateRecord(ctx, coll, map[string]interface{}{"title": "t", "age": 25}, nil, false)
		a.NoError(err)
	})
}

func TestYaegi_BeforeCreateHook(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()

	ym := NewYaegiManager(db)
	db.SetYaegi(ym)

	coll := createBasicCollection(t, db, "yaegi_hook")
	err := coll.AddField(ctx, CreateFieldInput{Name: "auto_field", Type: "string", DisplayName: "auto"})
	a.NoError(err)
	repo := coll.Repository()

	const hookScript = `
package main

import (
	"context"
)

func BeforeCreate(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
	data["auto_field"] = "from_hook"
	return data, nil
}
`
	err = ym.LoadScript(&modelmd.YaegiScript{
		CollectionName: "yaegi_hook",
		Name:           "auto_field_hook",
		HookPoint:      string(HookPointBeforeCreate),
		Content:        hookScript,
		Enabled:        true,
	})
	a.NoError(err)

	record, err := repo.Create(ctx, &CreateOptions{
		Values: map[string]interface{}{"title": "hook_test", "age": 20},
	})
	a.NoError(err)
	a.NotNil(record)
	a.Equal("from_hook", record.Get("auto_field"))
}

func TestYaegi_WriteUnitOfWork(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()

	ym := NewYaegiManager(db)
	db.SetYaegi(ym)

	parent := createBasicCollection(t, db, "uow_parent")
	child := createBasicCollection(t, db, "uow_child")
	_ = child
	err := parent.AddField(ctx, CreateFieldInput{Name: "auto_field", Type: "string", DisplayName: "auto"})
	a.NoError(err)

	const afterFail = `
package main

import (
	"context"
	"fmt"

	"itcodex/metadata"
)

func AfterCreate(ctx context.Context, result map[string]interface{}) error {
	items := metadata.Collection(ctx, "uow_child")
	if items == nil {
		return fmt.Errorf("child collection missing")
	}
	_, err := items.Create(map[string]interface{}{"title": "child_row", "age": 1})
	if err != nil {
		return err
	}
	return fmt.Errorf("force rollback")
}
`
	err = ym.LoadScript(&modelmd.YaegiScript{
		CollectionName: "uow_parent",
		Name:           "uow_fail",
		HookPoint:      string(HookPointAfterCreate),
		Content:        afterFail,
		Enabled:        true,
	})
	a.NoError(err)

	_, err = parent.Repository().Create(ctx, &CreateOptions{
		Values: map[string]interface{}{"title": "parent_row", "age": 1},
	})
	a.Error(err)

	nParent, err := parent.Repository().Count(ctx, &CountOptions{})
	a.NoError(err)
	a.Equal(0, nParent)
	nChild, err := child.Repository().Count(ctx, &CountOptions{})
	a.NoError(err)
	a.Equal(0, nChild)
}

func TestYaegi_AfterCommitHook(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()

	ym := NewYaegiManager(db)
	db.SetYaegi(ym)

	coll := createBasicCollection(t, db, "uow_commit")
	err := coll.AddField(ctx, CreateFieldInput{Name: "auto_field", Type: "string", DisplayName: "auto"})
	a.NoError(err)

	const script = `
package main

import (
	"context"

	"itcodex/metadata"
)

func AfterCommit(ctx context.Context, result map[string]interface{}) error {
	repo := metadata.Collection(ctx, "uow_commit")
	_, err := repo.Update(result["id"], map[string]interface{}{"auto_field": "committed"})
	return err
}
`
	err = ym.LoadScript(&modelmd.YaegiScript{
		CollectionName: "uow_commit",
		Name:           "uow_commit_hook",
		HookPoint:      string(HookPointAfterCommit),
		Content:        script,
		Enabled:        true,
	})
	a.NoError(err)

	record, err := coll.Repository().Create(ctx, &CreateOptions{
		Values: map[string]interface{}{"title": "c1", "age": 2},
	})
	a.NoError(err)
	found, err := coll.Repository().FindOne(ctx, &FindOneOptions{FilterByTk: record.Id()})
	a.NoError(err)
	a.Equal("committed", found.Get("auto_field"))
}

func TestYaegi_CustomAPI(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)

	ym := NewYaegiManager(db)
	db.SetYaegi(ym)

	found := ym.FindCustomAPI("GET", "/global/hello")
	// No script loaded yet
	a.Nil(found)
}

func TestYaegi_ValidateScript(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ym := NewYaegiManager(db)

	t.Run("invalid syntax returns error", func(t *testing.T) {
		err := ym.ValidateScript(`package main; func invalid syntax {{{`)
		a.Error(err)
	})
}

func TestIDGenerator(t *testing.T) {
	a := assert.New(t)

	t.Run("snowflake ids are unique", func(t *testing.T) {
		seen := make(map[int64]bool)
		for i := 0; i < 1000; i++ {
			id := utils.NextID()
			a.False(seen[id])
			seen[id] = true
		}
	})
}
