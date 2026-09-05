package metadata

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itcodex/server/pkg/utils"
)

func TestFilter_EmptyAndIncludes(t *testing.T) {
	a := assert.New(t)
	var params []interface{}

	sql, err := BuildWhereClause(Filter{"title": map[string]interface{}{"$empty": true}}, &params)
	a.NoError(err)
	a.Contains(sql, "IS NULL")

	params = nil
	sql, err = BuildWhereClause(Filter{"title": map[string]interface{}{"$includes": "ab"}}, &params)
	a.NoError(err)
	a.Contains(sql, "LIKE")
	a.Equal("%ab%", params[0])
}

func TestPasswordHash(t *testing.T) {
	a := assert.New(t)
	f, err := NewPasswordField(nil, map[string]interface{}{"name": "pwd"})
	a.NoError(err)
	stored, err := f.ToStoreValue("secret")
	a.NoError(err)
	a.Equal(utils.HashPassword("secret"), stored)
}

func TestRelationBelongsToAppends(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()

	authors, err := db.CreateCollection(ctx, CreateCollectionInput{
		Name: "authors_rel", DisplayName: "authors",
		Fields: []CreateFieldInput{{Name: "name", Type: FieldTypeString, IsRequired: true}},
	})
	require.NoError(t, err)

	posts, err := db.CreateCollection(ctx, CreateCollectionInput{
		Name: "posts_rel", DisplayName: "posts",
		Fields: []CreateFieldInput{
			{Name: "title", Type: FieldTypeString, IsRequired: true},
			{Name: "author_id", Type: FieldTypeBelongsTo, Target: "authors_rel", ForeignKey: "author_id"},
		},
	})
	require.NoError(t, err)

	author, err := authors.Repository().Create(ctx, &CreateOptions{Values: map[string]interface{}{"name": "Alice"}})
	require.NoError(t, err)

	post, err := posts.Repository().Create(ctx, &CreateOptions{Values: map[string]interface{}{
		"title": "Hello", "author_id": author.Id(),
	}})
	require.NoError(t, err)

	found, err := posts.Repository().FindOne(ctx, &FindOneOptions{
		FilterByTk:    post.Id(),
		CommonOptions: CommonOptions{Appends: Appends{"author_id"}},
	})
	require.NoError(t, err)
	rel := found.Get("author_id")
	// after appends, author_id may be object when field name equals FK
	if m, ok := rel.(map[string]interface{}); ok {
		a.Equal("Alice", m["name"])
	}

	list, err := posts.Repository().ListAssociation(ctx, post.Id(), "author_id")
	a.NoError(err)
	a.NotEmpty(list)
}

func TestHasManyAssociationSet(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()

	_, err := db.CreateCollection(ctx, CreateCollectionInput{
		Name: "users_hm", DisplayName: "users",
		Fields: []CreateFieldInput{
			{Name: "name", Type: FieldTypeString},
			{Name: "posts", Type: FieldTypeHasMany, Target: "posts_hm", ForeignKey: "user_id"},
		},
	})
	require.NoError(t, err)

	posts, err := db.CreateCollection(ctx, CreateCollectionInput{
		Name: "posts_hm", DisplayName: "posts",
		Fields: []CreateFieldInput{
			{Name: "title", Type: FieldTypeString},
			{Name: "user_id", Type: "bigint"},
		},
	})
	require.NoError(t, err)
	_ = posts

	users := db.Collection("users_hm")
	user, err := users.Repository().Create(ctx, &CreateOptions{Values: map[string]interface{}{"name": "u1"}})
	require.NoError(t, err)

	p1, err := posts.Repository().Create(ctx, &CreateOptions{Values: map[string]interface{}{"title": "p1"}})
	require.NoError(t, err)
	p2, err := posts.Repository().Create(ctx, &CreateOptions{Values: map[string]interface{}{"title": "p2"}})
	require.NoError(t, err)

	err = users.Repository().SetAssociation(ctx, user.Id(), "posts", []interface{}{
		map[string]interface{}{"id": p1.Id()},
		map[string]interface{}{"id": p2.Id()},
	})
	a.NoError(err)

	found, err := users.Repository().FindOne(ctx, &FindOneOptions{
		FilterByTk:    user.Id(),
		CommonOptions: CommonOptions{Appends: Appends{"posts"}},
	})
	require.NoError(t, err)
	list, ok := found.Get("posts").([]map[string]interface{})
	a.True(ok)
	a.Len(list, 2)
}

func TestPhase5Fields_SequenceUUIDEncrypted(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()

	coll, err := db.CreateCollection(ctx, CreateCollectionInput{
		Name: "adv_fields", DisplayName: "adv",
		Fields: []CreateFieldInput{
			{Name: "code", Type: FieldTypeSequence, Pattern: "ORD-{YYYY}-{0000}", StartsAt: 1},
			{Name: "uid", Type: FieldTypeUUID},
			{Name: "secret", Type: FieldTypeEncrypted},
			{Name: "loc", Type: FieldTypePoint},
		},
	})
	require.NoError(t, err)

	rec, err := coll.Repository().Create(ctx, &CreateOptions{Values: map[string]interface{}{
		"secret": "hello",
		"loc":    map[string]interface{}{"type": "Point", "coordinates": []interface{}{1.0, 2.0}},
	}})
	require.NoError(t, err)
	a.NotEmpty(rec.Get("code"))
	a.NotEmpty(rec.Get("uid"))
	a.Equal("hello", rec.Get("secret"))
}

func TestTreeChildrenAppend(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()

	coll, err := db.CreateCollection(ctx, CreateCollectionInput{
		Name: "tree_nodes", DisplayName: "tree", Type: CollectionTypeTree,
		Fields: []CreateFieldInput{{Name: "title", Type: FieldTypeString}},
	})
	require.NoError(t, err)
	a.True(coll.HasField("parent_id"))

	root, err := coll.Repository().Create(ctx, &CreateOptions{Values: map[string]interface{}{"title": "root"}})
	require.NoError(t, err)
	_, err = coll.Repository().Create(ctx, &CreateOptions{Values: map[string]interface{}{
		"title": "child", "parent_id": root.Id(),
	}})
	require.NoError(t, err)

	found, err := coll.Repository().FindOne(ctx, &FindOneOptions{
		FilterByTk:    root.Id(),
		CommonOptions: CommonOptions{Appends: Appends{"children"}},
	})
	require.NoError(t, err)
	children, ok := found.Get("children").([]map[string]interface{})
	a.True(ok)
	a.Len(children, 1)
}

func TestIndexPersist(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()

	coll := createBasicCollection(t, db, "idx_coll")
	err := coll.AddIndex(ctx, &Index{Fields: []string{"title"}, Unique: false})
	a.NoError(err)
	a.NotEmpty(coll.Indexes())
}

func TestFieldFactories_Registered(t *testing.T) {
	a := assert.New(t)
	types := GetRegisteredFieldTypes()
	need := []FieldType{
		FieldTypeBelongsTo, FieldTypeHasMany, FieldTypeFormula, FieldTypeEncrypted,
		FieldTypeMultiSelect, FieldTypeMarkdown, FieldTypeSequence,
	}
	set := map[FieldType]bool{}
	for _, t := range types {
		set[t] = true
	}
	for _, n := range need {
		a.True(set[n], "missing factory %s", n)
	}
}
