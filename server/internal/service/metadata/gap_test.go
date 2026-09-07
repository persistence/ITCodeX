package metadata

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	modelmd "itcodex/server/internal/model/metadata"
)

func TestNestedAssociationCreate(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()

	_, err := db.CreateCollection(ctx, CreateCollectionInput{
		Name: "users_nested", DisplayName: "users",
		Fields: []CreateFieldInput{
			{Name: "name", Type: FieldTypeString},
			{Name: "posts", Type: FieldTypeHasMany, Target: "posts_nested", ForeignKey: "user_id"},
		},
	})
	require.NoError(t, err)

	_, err = db.CreateCollection(ctx, CreateCollectionInput{
		Name: "posts_nested", DisplayName: "posts",
		Fields: []CreateFieldInput{
			{Name: "title", Type: FieldTypeString},
			{Name: "user_id", Type: "bigint"},
		},
	})
	require.NoError(t, err)

	users := db.Collection("users_nested")
	user, err := users.Repository().Create(ctx, &CreateOptions{Values: map[string]any{
		"name": "u1",
		"posts": []any{
			map[string]any{"title": "p1"},
			map[string]any{"title": "p2"},
		},
	}})
	require.NoError(t, err)

	found, err := users.Repository().FindOne(ctx, &FindOneOptions{
		FilterByTk:    user.Id(),
		CommonOptions: CommonOptions{Appends: Appends{"posts"}},
	})
	require.NoError(t, err)
	list, ok := found.Get("posts").([]map[string]any)
	a.True(ok)
	a.Len(list, 2)
}

func TestBelongsToNestedCreate(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()

	_, err := db.CreateCollection(ctx, CreateCollectionInput{
		Name: "authors_nc", DisplayName: "authors",
		Fields: []CreateFieldInput{{Name: "name", Type: FieldTypeString, IsRequired: true}},
	})
	require.NoError(t, err)

	posts, err := db.CreateCollection(ctx, CreateCollectionInput{
		Name: "posts_nc", DisplayName: "posts",
		Fields: []CreateFieldInput{
			{Name: "title", Type: FieldTypeString, IsRequired: true},
			{Name: "author_id", Type: FieldTypeBelongsTo, Target: "authors_nc", ForeignKey: "author_id"},
		},
	})
	require.NoError(t, err)

	post, err := posts.Repository().Create(ctx, &CreateOptions{Values: map[string]any{
		"title":     "Hello",
		"author_id": map[string]any{"name": "Bob"},
	}})
	require.NoError(t, err)
	a.NotNil(post.Get("author_id"))

	found, err := posts.Repository().FindOne(ctx, &FindOneOptions{
		FilterByTk:    post.Id(),
		CommonOptions: CommonOptions{Appends: Appends{"author_id"}},
	})
	require.NoError(t, err)
	rel, ok := found.Get("author_id").(map[string]any)
	a.True(ok)
	a.Equal("Bob", rel["name"])
}

func TestNestedAssociationFilter(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()

	_, err := db.CreateCollection(ctx, CreateCollectionInput{
		Name: "users_nf", DisplayName: "users",
		Fields: []CreateFieldInput{
			{Name: "name", Type: FieldTypeString},
			{Name: "posts", Type: FieldTypeHasMany, Target: "posts_nf", ForeignKey: "user_id"},
		},
	})
	require.NoError(t, err)
	_, err = db.CreateCollection(ctx, CreateCollectionInput{
		Name: "posts_nf", DisplayName: "posts",
		Fields: []CreateFieldInput{
			{Name: "title", Type: FieldTypeString},
			{Name: "user_id", Type: "bigint"},
			{Name: "comments", Type: FieldTypeHasMany, Target: "comments_nf", ForeignKey: "post_id"},
		},
	})
	require.NoError(t, err)
	_, err = db.CreateCollection(ctx, CreateCollectionInput{
		Name: "comments_nf", DisplayName: "comments",
		Fields: []CreateFieldInput{
			{Name: "body", Type: FieldTypeString},
			{Name: "post_id", Type: "bigint"},
		},
	})
	require.NoError(t, err)

	users := db.Collection("users_nf")
	_, err = users.Repository().Create(ctx, &CreateOptions{Values: map[string]any{
		"name": "u1",
		"posts": []any{
			map[string]any{
				"title": "p1",
				"comments": []any{
					map[string]any{"body": "hello world"},
				},
			},
		},
	}})
	require.NoError(t, err)
	_, err = users.Repository().Create(ctx, &CreateOptions{Values: map[string]any{"name": "u2"}})
	require.NoError(t, err)

	list, err := users.Repository().Find(ctx, &FindOptions{
		CommonOptions: CommonOptions{Filter: Filter{"posts.comments.body": Filter{"$like": "%hello%"}}},
	})
	require.NoError(t, err)
	a.Len(list, 1)
	a.Equal("u1", list[0].Get("name"))
}

func TestDestroyCascadeOnDelete(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()

	users, err := db.CreateCollection(ctx, CreateCollectionInput{
		Name: "users_cd", DisplayName: "users",
		Fields: []CreateFieldInput{
			{Name: "name", Type: FieldTypeString},
			{Name: "posts", Type: FieldTypeHasMany, Target: "posts_cd", ForeignKey: "user_id", OnDelete: "CASCADE"},
		},
	})
	require.NoError(t, err)
	posts, err := db.CreateCollection(ctx, CreateCollectionInput{
		Name: "posts_cd", DisplayName: "posts",
		Fields: []CreateFieldInput{
			{Name: "title", Type: FieldTypeString},
			{Name: "user_id", Type: "bigint"},
		},
	})
	require.NoError(t, err)

	user, err := users.Repository().Create(ctx, &CreateOptions{Values: map[string]any{
		"name":  "u1",
		"posts": []any{map[string]any{"title": "p1"}},
	}})
	require.NoError(t, err)

	_, err = users.Repository().Destroy(ctx, &DestroyOptions{FilterByTk: user.Id()})
	require.NoError(t, err)
	n, err := posts.Repository().Count(ctx, &CountOptions{})
	require.NoError(t, err)
	a.Equal(0, n)
}

func TestDropCollectionCascade(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()

	_, err := db.CreateCollection(ctx, CreateCollectionInput{
		Name: "authors_dc", DisplayName: "authors",
		Fields: []CreateFieldInput{{Name: "name", Type: FieldTypeString}},
	})
	require.NoError(t, err)
	_, err = db.CreateCollection(ctx, CreateCollectionInput{
		Name: "posts_dc", DisplayName: "posts",
		Fields: []CreateFieldInput{
			{Name: "title", Type: FieldTypeString},
			{Name: "author_id", Type: FieldTypeBelongsTo, Target: "authors_dc", ForeignKey: "author_id"},
		},
	})
	require.NoError(t, err)

	err = db.DropCollection(ctx, "authors_dc")
	a.Error(err)

	err = db.DropCollection(ctx, "authors_dc", true)
	a.NoError(err)
	a.Nil(db.Collection("authors_dc"))
	a.Nil(db.Collection("posts_dc"))
}

func TestBeforeValidateHook(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()
	ym := NewYaegiManager(db)
	db.SetYaegi(ym)

	coll := createBasicCollection(t, db, "validate_hook")
	err := ym.LoadScript(&modelmd.YaegiScript{
		CollectionName: "validate_hook",
		Name:           "bv",
		HookPoint:      string(HookPointBeforeValidate),
		Enabled:        true,
		Content: `package main
import "context"
func BeforeValidate(ctx context.Context, data map[string]any) (map[string]any, error) {
	data["title"] = "normalized"
	return data, nil
}`,
	})
	require.NoError(t, err)

	rec, err := coll.Repository().Create(ctx, &CreateOptions{Values: map[string]any{"title": "raw", "age": 1}})
	require.NoError(t, err)
	a.Equal("normalized", rec.Get("title"))
}

func TestScriptToggle(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()

	script := &modelmd.YaegiScript{
		Name:      "t1",
		HookPoint: string(HookPointBeforeCreate),
		Content:   "package main\n",
		Enabled:   true,
	}
	require.NoError(t, db.SaveScript(ctx, script))
	a.True(script.Id > 0)

	toggled, err := db.ToggleScript(ctx, script.Id)
	require.NoError(t, err)
	a.False(toggled.Enabled)
}

func TestFileObjectStorage(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()
	dir := t.TempDir()
	db.options.StoragePath = dir

	_, err := db.CreateCollection(ctx, CreateCollectionInput{
		Name: "files_store", DisplayName: "files", Type: CollectionTypeFile,
	})
	require.NoError(t, err)

	rec, err := db.SaveFileObject(ctx, "files_store", "hello.txt", "text/plain", bytes.NewBufferString("hi"))
	require.NoError(t, err)
	a.Equal("hello.txt", rec["name"])
	a.Equal(int64(2), rec["size"])

	abs, mimeType, name, err := db.OpenFileObject(ctx, "files_store", rec["id"])
	require.NoError(t, err)
	a.Equal("text/plain", mimeType)
	a.Equal("hello.txt", name)
	b, err := os.ReadFile(abs)
	require.NoError(t, err)
	a.Equal("hi", string(b))
}
