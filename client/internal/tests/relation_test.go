package tests

import (
	"testing"

	"itcodex/client/internal/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBelongsTo_Appends(t *testing.T) {
	s := setupTest(t)
	authors := uniqueName("authors")
	posts := uniqueName("posts")

	s.createTestCollection(t, authors,
		client.CreateFieldInput{Name: "name", Type: "string", IsRequired: true},
	)
	s.createTestCollection(t, posts,
		client.CreateFieldInput{Name: "title", Type: "string", IsRequired: true},
		client.CreateFieldInput{Name: "author_id", Type: "belongsTo", Target: authors, ForeignKey: "author_id"},
	)

	author := s.createTestRecord(t, authors, map[string]any{"name": "Alice"})
	post := s.createTestRecord(t, posts, map[string]any{
		"title": "Hello", "author_id": author["id"],
	})

	got, err := s.client.FindOne(s.ctx, posts, idStr(post["id"]), &client.FindOneOptions{
		Appends: []string{"author_id"},
	})
	require.NoError(t, err)
	require.NotNil(t, got)

	list, err := s.client.ListAssociation(s.ctx, posts, idStr(post["id"]), "author_id")
	require.NoError(t, err)
	assert.NotEmpty(t, list)
}

func TestHasMany_SetAssociation(t *testing.T) {
	s := setupTest(t)
	users := uniqueName("users_hm")
	posts := uniqueName("posts_hm")

	s.createTestCollection(t, users,
		client.CreateFieldInput{Name: "name", Type: "string"},
		client.CreateFieldInput{Name: "posts", Type: "hasMany", Target: posts, ForeignKey: "user_id"},
	)
	s.createTestCollection(t, posts,
		client.CreateFieldInput{Name: "title", Type: "string"},
		client.CreateFieldInput{Name: "user_id", Type: "bigint"},
	)

	user := s.createTestRecord(t, users, map[string]any{"name": "u1"})
	p1 := s.createTestRecord(t, posts, map[string]any{"title": "p1"})
	p2 := s.createTestRecord(t, posts, map[string]any{"title": "p2"})

	err := s.client.SetAssociation(s.ctx, users, idStr(user["id"]), "posts", []map[string]any{
		{"id": idStr(p1["id"])},
		{"id": idStr(p2["id"])},
	})
	require.NoError(t, err)

	listed, err := s.client.List(s.ctx, posts, &client.FindOptions{
		Filter: client.Filter{"user_id": user["id"]},
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(listed.List), 2, "hasMany SetAssociation should link child rows")

	got, err := s.client.FindOne(s.ctx, users, idStr(user["id"]), &client.FindOneOptions{
		Appends: []string{"posts"},
	})
	require.NoError(t, err)
	switch v := got["posts"].(type) {
	case []any:
		assert.GreaterOrEqual(t, len(v), 2)
	case []map[string]any:
		assert.GreaterOrEqual(t, len(v), 2)
	default:
		t.Fatalf("unexpected posts append type %T", got["posts"])
	}
}

func TestBelongsToMany_Through(t *testing.T) {
	s := setupTest(t)
	posts := uniqueName("btm_posts")
	tags := uniqueName("btm_tags")
	through := uniqueName("btm_through")

	s.createTestCollection(t, tags,
		client.CreateFieldInput{Name: "name", Type: "string", IsRequired: true},
	)
	s.createTestCollection(t, posts,
		client.CreateFieldInput{Name: "title", Type: "string", IsRequired: true},
		client.CreateFieldInput{
			Name: "tags", Type: "belongsToMany", Target: tags,
			Through: through, ForeignKey: "post_id", OtherKey: "tag_id",
		},
	)

	post := s.createTestRecord(t, posts, map[string]any{"title": "p"})
	tag1 := s.createTestRecord(t, tags, map[string]any{"name": "t1"})
	tag2 := s.createTestRecord(t, tags, map[string]any{"name": "t2"})

	err := s.client.AddAssociation(s.ctx, posts, idStr(post["id"]), "tags", []map[string]any{
		{"id": idStr(tag1["id"])}, {"id": idStr(tag2["id"])},
	})
	require.NoError(t, err)

	list, err := s.client.ListAssociation(s.ctx, posts, idStr(post["id"]), "tags")
	require.NoError(t, err)
	require.Len(t, list, 2, "belongsToMany should return linked tags")

	err = s.client.RemoveAssociation(s.ctx, posts, idStr(post["id"]), "tags", map[string]any{"id": idStr(tag1["id"])})
	require.NoError(t, err)

	list, err = s.client.ListAssociation(s.ctx, posts, idStr(post["id"]), "tags")
	require.NoError(t, err)
	assert.Len(t, list, 1)
}
