package metadata

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDatabase_Bootstrap(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)

	coll := createBasicCollection(t, db, "boot_test")
	a.NotNil(coll)
	a.Equal("boot_test", coll.Name())
}

func TestDatabase_CreateCollection(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()

	t.Run("create collection with preset fields", func(t *testing.T) {
		coll, err := db.CreateCollection(ctx, CreateCollectionInput{
			Name:         "users",
			DisplayName:  "用户",
			PresetFields: []string{"id", "createdAt", "updatedAt"},
			Fields: []CreateFieldInput{
				{Name: "username", Type: "string", IsRequired: true, IsUnique: true},
				{Name: "email", Type: "email", IsRequired: true},
			},
		})
		a.NoError(err)
		a.NotNil(coll)
		a.True(coll.HasField("id"))
		a.True(coll.HasField("username"))
		a.True(coll.HasField("email"))
	})

	t.Run("duplicate name returns error", func(t *testing.T) {
		createBasicCollection(t, db, "dup_test")
		_, err := db.CreateCollection(ctx, CreateCollectionInput{Name: "dup_test"})
		a.Error(err)
	})

	t.Run("invalid name returns error", func(t *testing.T) {
		_, err := db.CreateCollection(ctx, CreateCollectionInput{Name: "123_bad"})
		a.Error(err)
	})
}

func TestCollection_AddRemoveField(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()
	coll := createBasicCollection(t, db, "fieldmgmt")

	t.Run("add field succeeds", func(t *testing.T) {
		err := coll.AddField(ctx, CreateFieldInput{Name: "email", Type: "email"})
		a.NoError(err)
		a.True(coll.HasField("email"))
	})

	t.Run("add duplicate field fails", func(t *testing.T) {
		err := coll.AddField(ctx, CreateFieldInput{Name: "title", Type: "string"})
		a.Error(err)
	})

	t.Run("protect system fields from removal", func(t *testing.T) {
		err := coll.RemoveField(ctx, "id")
		a.Error(err)
	})
}

func TestCollection_Drop(t *testing.T) {
	a := assert.New(t)
	db := newTestDB(t)
	ctx := context.Background()
	createBasicCollection(t, db, "todrop")

	err := db.DropCollection(ctx, "todrop")
	a.NoError(err)
	a.False(db.HasCollection("todrop"))
}
