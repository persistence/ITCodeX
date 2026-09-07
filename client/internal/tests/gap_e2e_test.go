package tests

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"itcodex/client/internal/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNestedCreate_HasMany(t *testing.T) {
	s := setupTest(t)
	users := uniqueName("e2e_users_n")
	posts := uniqueName("e2e_posts_n")

	s.createTestCollection(t, users,
		client.CreateFieldInput{Name: "name", Type: "string"},
		client.CreateFieldInput{Name: "posts", Type: "hasMany", Target: posts, ForeignKey: "user_id"},
	)
	s.createTestCollection(t, posts,
		client.CreateFieldInput{Name: "title", Type: "string"},
		client.CreateFieldInput{Name: "user_id", Type: "bigint"},
	)
	t.Cleanup(func() {
		_ = s.client.DropCollectionCascade(s.ctx, users, true)
		_ = s.client.DropCollectionCascade(s.ctx, posts, true)
	})

	user, err := s.client.Create(s.ctx, users, map[string]any{
		"name": "u1",
		"posts": []any{
			map[string]any{"title": "p1"},
			map[string]any{"title": "p2"},
		},
	})
	require.NoError(t, err)

	got, err := s.client.FindOne(s.ctx, users, idStr(user["id"]), &client.FindOneOptions{Appends: []string{"posts"}})
	require.NoError(t, err)
	switch v := got["posts"].(type) {
	case []any:
		assert.Len(t, v, 2)
	case []map[string]any:
		assert.Len(t, v, 2)
	default:
		t.Fatalf("posts type %T", got["posts"])
	}
}

func TestNestedCreate_BelongsTo(t *testing.T) {
	s := setupTest(t)
	authors := uniqueName("e2e_authors_n")
	posts := uniqueName("e2e_posts_bt")

	s.createTestCollection(t, authors,
		client.CreateFieldInput{Name: "name", Type: "string", IsRequired: true},
	)
	s.createTestCollection(t, posts,
		client.CreateFieldInput{Name: "title", Type: "string", IsRequired: true},
		client.CreateFieldInput{Name: "author_id", Type: "belongsTo", Target: authors, ForeignKey: "author_id"},
	)
	t.Cleanup(func() {
		_ = s.client.DropCollectionCascade(s.ctx, posts, true)
		_ = s.client.DropCollectionCascade(s.ctx, authors, true)
	})

	post, err := s.client.Create(s.ctx, posts, map[string]any{
		"title":     "Hello",
		"author_id": map[string]any{"name": "Bob"},
	})
	require.NoError(t, err)
	require.NotNil(t, post["author_id"])

	got, err := s.client.FindOne(s.ctx, posts, idStr(post["id"]), &client.FindOneOptions{Appends: []string{"author_id"}})
	require.NoError(t, err)
	rel, ok := got["author_id"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Bob", rel["name"])
}

func TestFilter_NestedAssociationPath(t *testing.T) {
	s := setupTest(t)
	users := uniqueName("e2e_users_f")
	posts := uniqueName("e2e_posts_f")
	comments := uniqueName("e2e_comments_f")

	s.createTestCollection(t, users,
		client.CreateFieldInput{Name: "name", Type: "string"},
		client.CreateFieldInput{Name: "posts", Type: "hasMany", Target: posts, ForeignKey: "user_id"},
	)
	s.createTestCollection(t, posts,
		client.CreateFieldInput{Name: "title", Type: "string"},
		client.CreateFieldInput{Name: "user_id", Type: "bigint"},
		client.CreateFieldInput{Name: "comments", Type: "hasMany", Target: comments, ForeignKey: "post_id"},
	)
	s.createTestCollection(t, comments,
		client.CreateFieldInput{Name: "body", Type: "string"},
		client.CreateFieldInput{Name: "post_id", Type: "bigint"},
	)
	t.Cleanup(func() {
		_ = s.client.DropCollectionCascade(s.ctx, users, true)
		_ = s.client.DropCollectionCascade(s.ctx, posts, true)
		_ = s.client.DropCollectionCascade(s.ctx, comments, true)
	})

	_, err := s.client.Create(s.ctx, users, map[string]any{
		"name": "u1",
		"posts": []any{
			map[string]any{
				"title": "p1",
				"comments": []any{
					map[string]any{"body": "hello world"},
				},
			},
		},
	})
	require.NoError(t, err)
	_, err = s.client.Create(s.ctx, users, map[string]any{"name": "u2"})
	require.NoError(t, err)

	list, err := s.client.List(s.ctx, users, &client.FindOptions{
		Filter: client.Filter{"posts.comments.body": client.Filter{"$like": "%hello%"}},
	})
	require.NoError(t, err)
	require.Len(t, list.List, 1)
	assert.Equal(t, "u1", list.List[0]["name"])
}

func TestDestroy_OnDeleteCascade(t *testing.T) {
	s := setupTest(t)
	users := uniqueName("e2e_users_cd")
	posts := uniqueName("e2e_posts_cd")

	s.createTestCollection(t, users,
		client.CreateFieldInput{Name: "name", Type: "string"},
		client.CreateFieldInput{Name: "posts", Type: "hasMany", Target: posts, ForeignKey: "user_id", OnDelete: "CASCADE"},
	)
	s.createTestCollection(t, posts,
		client.CreateFieldInput{Name: "title", Type: "string"},
		client.CreateFieldInput{Name: "user_id", Type: "bigint"},
	)
	t.Cleanup(func() {
		_ = s.client.DropCollectionCascade(s.ctx, users, true)
		_ = s.client.DropCollectionCascade(s.ctx, posts, true)
	})

	user, err := s.client.Create(s.ctx, users, map[string]any{
		"name":  "u1",
		"posts": []any{map[string]any{"title": "p1"}},
	})
	require.NoError(t, err)

	_, err = s.client.DeleteOne(s.ctx, users, idStr(user["id"]))
	require.NoError(t, err)

	n, err := s.client.Count(s.ctx, posts, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)
}

func TestDropCollection_Cascade(t *testing.T) {
	s := setupTest(t)
	authors := uniqueName("e2e_authors_dc")
	posts := uniqueName("e2e_posts_dc")

	s.createTestCollection(t, authors,
		client.CreateFieldInput{Name: "name", Type: "string"},
	)
	s.createTestCollection(t, posts,
		client.CreateFieldInput{Name: "title", Type: "string"},
		client.CreateFieldInput{Name: "author_id", Type: "belongsTo", Target: authors, ForeignKey: "author_id"},
	)

	err := s.client.DropCollection(s.ctx, authors)
	require.Error(t, err)
	assert.True(t, s.isAPIError(err, http.StatusForbidden) || s.isAPIError(err, 403))

	err = s.client.DropCollectionCascade(s.ctx, authors, true)
	require.NoError(t, err)

	_, err = s.client.GetCollection(s.ctx, authors)
	require.Error(t, err)
	_, err = s.client.GetCollection(s.ctx, posts)
	require.Error(t, err)
}

func TestScript_Toggle(t *testing.T) {
	s := setupTest(t)
	coll := uniqueName("e2e_script_tg")
	s.createTestCollection(t, coll,
		client.CreateFieldInput{Name: "title", Type: "string"},
		client.CreateFieldInput{Name: "flag", Type: "string"},
	)

	script, err := s.client.CreateScript(s.ctx, client.CreateScriptInput{
		Name:           "tg",
		CollectionName: coll,
		HookPoint:      "beforeCreate",
		Enabled:        true,
		Content: `package main
import "context"
func BeforeCreate(ctx context.Context, data map[string]any) (map[string]any, error) {
	data["flag"] = "on"
	return data, nil
}`,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.client.DeleteScript(s.ctx, script.ID) })

	rec := s.createTestRecord(t, coll, map[string]any{"title": "a"})
	assert.Equal(t, "on", rec["flag"])

	toggled, err := s.client.ToggleScript(s.ctx, script.ID)
	require.NoError(t, err)
	assert.False(t, toggled.Enabled)

	rec2 := s.createTestRecord(t, coll, map[string]any{"title": "b"})
	assert.Nil(t, rec2["flag"])
}

func TestUpdate_Whitelist(t *testing.T) {
	s := setupTest(t)
	coll := uniqueName("e2e_wl")
	s.createTestCollection(t, coll,
		client.CreateFieldInput{Name: "title", Type: "string"},
		client.CreateFieldInput{Name: "views", Type: "integer"},
	)

	rec := s.createTestRecord(t, coll, map[string]any{"title": "t1", "views": 1})
	updated, err := s.client.UpdateWithOptions(s.ctx, coll, idStr(rec["id"]), map[string]any{
		"title": "t2",
		"views": 99,
	}, &client.UpdateOptions{Whitelist: []string{"title"}})
	require.NoError(t, err)
	assert.Equal(t, "t2", updated["title"])

	got, err := s.client.FindOne(s.ctx, coll, idStr(rec["id"]), nil)
	require.NoError(t, err)
	assert.Equal(t, "t2", got["title"])
	assert.Equal(t, int64(1), got["views"])
}

func TestCollection_FileUpload(t *testing.T) {
	s := setupTest(t)
	coll := uniqueName("e2e_files")
	s.createSpecialCollection(t, coll, "file")

	rec, err := s.client.UploadFile(s.ctx, coll, "hello.txt", "text/plain", []byte("hello-e2e"))
	require.NoError(t, err)
	assert.Equal(t, "hello.txt", rec["name"])
	assert.Equal(t, int64(9), rec["size"])

	body, ctype, err := s.client.DownloadFile(s.ctx, coll, idStr(rec["id"]))
	require.NoError(t, err)
	assert.Equal(t, "hello-e2e", string(body))
	assert.Contains(t, ctype, "text/plain")
}

func TestCollection_CalendarRange(t *testing.T) {
	s := setupTest(t)
	coll := uniqueName("e2e_cal")
	s.createSpecialCollection(t, coll, "calendar",
		client.CreateFieldInput{Name: "title", Type: "string"},
	)

	_, err := s.client.Create(s.ctx, coll, map[string]any{
		"title": "a",
		"start": "2026-09-01 10:00:00",
		"end":   "2026-09-01 11:00:00",
	})
	require.NoError(t, err)
	_, err = s.client.Create(s.ctx, coll, map[string]any{
		"title": "b",
		"start": "2026-10-01 10:00:00",
		"end":   "2026-10-01 11:00:00",
	})
	require.NoError(t, err)

	list, err := s.client.List(s.ctx, coll, &client.FindOptions{
		Start: "2026-09-01 00:00:00",
		End:   "2026-09-30 23:59:59",
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.List), 1)
	for _, row := range list.List {
		assert.Equal(t, "a", row["title"])
	}
}

func TestScript_BeforeValidate(t *testing.T) {
	s := setupTest(t)
	coll := uniqueName("e2e_bv")
	s.createTestCollection(t, coll,
		client.CreateFieldInput{Name: "title", Type: "string", IsRequired: true},
	)

	script, err := s.client.CreateScript(s.ctx, client.CreateScriptInput{
		Name:           "bv",
		CollectionName: coll,
		HookPoint:      "beforeValidate",
		Enabled:        true,
		Content: `package main
import "context"
func BeforeValidate(ctx context.Context, data map[string]any) (map[string]any, error) {
	data["title"] = "normalized"
	return data, nil
}`,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.client.DeleteScript(s.ctx, script.ID) })

	rec := s.createTestRecord(t, coll, map[string]any{"title": "raw"})
	assert.Equal(t, "normalized", rec["title"])
}

func TestCreateMany_Batch(t *testing.T) {
	s := setupTest(t)
	coll := uniqueName("e2e_batch")
	s.createTestCollection(t, coll,
		client.CreateFieldInput{Name: "title", Type: "string", IsRequired: true},
	)
	list, err := s.client.CreateMany(s.ctx, coll, []map[string]any{
		{"title": fmt.Sprintf("b1-%d", time.Now().UnixNano())},
		{"title": fmt.Sprintf("b2-%d", time.Now().UnixNano())},
	})
	require.NoError(t, err)
	assert.Len(t, list, 2)
}
