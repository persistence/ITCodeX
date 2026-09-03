package tests

import (
	"net/http"
	"testing"

	"itcodex/client/internal/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCollections_Empty(t *testing.T) {
	s := setupTest(t)

	colls, err := s.client.ListCollections(s.ctx)
	require.NoError(t, err)
	assert.NotNil(t, colls)
}

func TestCreateCollection_Success(t *testing.T) {
	s := setupTest(t)

	collName := "test_create_simple"
	coll := s.createTestCollection(t, collName)

	assert.Equal(t, collName, coll.Name)
	assert.Equal(t, "general", coll.Type)

	fetched, err := s.client.GetCollection(s.ctx, collName)
	require.NoError(t, err)
	assert.Equal(t, collName, fetched.Name)
}

func TestCreateCollection_WithFields(t *testing.T) {
	s := setupTest(t)

	collName := "test_create_with_fields"
	fields := []client.CreateFieldInput{
		{
			Name:        "title",
			DisplayName: "标题",
			Type:        "string",
			Required:    true,
		},
		{
			Name:        "content",
			DisplayName: "内容",
			Type:        "text",
		},
		{
			Name:        "views",
			DisplayName: "浏览量",
			Type:        "integer",
		},
	}

	coll := s.createTestCollection(t, collName, fields...)
	assert.Equal(t, collName, coll.Name)

	fieldList, err := s.client.ListFields(s.ctx, collName)
	require.NoError(t, err)

	fieldMap := make(map[string]client.Field)
	for _, f := range fieldList {
		fieldMap[f.Name] = f
	}

	assert.Contains(t, fieldMap, "id")
	assert.Contains(t, fieldMap, "created_at")
	assert.Contains(t, fieldMap, "updated_at")
	assert.Contains(t, fieldMap, "title")
	assert.Contains(t, fieldMap, "content")
	assert.Contains(t, fieldMap, "views")

	assert.True(t, fieldMap["title"].Required)
	assert.Equal(t, "string", fieldMap["title"].Type)
}

func TestCreateCollection_DuplicateName(t *testing.T) {
	s := setupTest(t)

	collName := "test_duplicate"
	_ = s.createTestCollection(t, collName)

	_, err := s.client.CreateCollection(s.ctx, client.CreateCollectionInput{
		Name:        collName,
		DisplayName: "重复集合",
		Type:        "general",
	})
	require.Error(t, err)
	assert.True(t, s.isAPIError(err, http.StatusConflict))
}

func TestGetCollection_Exists(t *testing.T) {
	s := setupTest(t)

	collName := "test_get_exists"
	_ = s.createTestCollection(t, collName)

	coll, err := s.client.GetCollection(s.ctx, collName)
	require.NoError(t, err)
	assert.Equal(t, collName, coll.Name)
}

func TestGetCollection_NotFound(t *testing.T) {
	s := setupTest(t)

	_, err := s.client.GetCollection(s.ctx, "nonexistent_collection_12345")
	require.Error(t, err)
	assert.True(t, s.isAPIError(err, http.StatusNotFound))
}

func TestDropCollection_Success(t *testing.T) {
	s := setupTest(t)

	collName := "test_drop_success"
	_ = s.createTestCollection(t, collName)

	err := s.client.DropCollection(s.ctx, collName)
	require.NoError(t, err)

	_, err = s.client.GetCollection(s.ctx, collName)
	require.Error(t, err)
	assert.True(t, s.isAPIError(err, http.StatusNotFound))
}

func TestDropCollection_NotFound(t *testing.T) {
	s := setupTest(t)

	err := s.client.DropCollection(s.ctx, "nonexistent_collection_drop")
	require.Error(t, err)
}

func TestAddField_Success(t *testing.T) {
	s := setupTest(t)

	collName := "test_add_field"
	_ = s.createTestCollection(t, collName)

	newField := client.CreateFieldInput{
		Name:        "new_field",
		DisplayName: "新字段",
		Type:        "string",
	}

	fields, err := s.client.AddField(s.ctx, collName, newField)
	require.NoError(t, err)

	fieldMap := make(map[string]client.Field)
	for _, f := range fields {
		fieldMap[f.Name] = f
	}
	assert.Contains(t, fieldMap, "new_field")
}

func TestAddField_DuplicateName(t *testing.T) {
	s := setupTest(t)

	collName := "test_add_field_dup"
	_ = s.createTestCollection(t, collName, client.CreateFieldInput{
		Name: "dup_field",
		Type: "string",
	})

	_, err := s.client.AddField(s.ctx, collName, client.CreateFieldInput{
		Name: "dup_field",
		Type: "string",
	})
	require.Error(t, err)
}

func TestListFields_Success(t *testing.T) {
	s := setupTest(t)

	collName := "test_list_fields"
	_ = s.createTestCollection(t, collName,
		client.CreateFieldInput{Name: "f1", Type: "string"},
		client.CreateFieldInput{Name: "f2", Type: "number"},
	)

	fields, err := s.client.ListFields(s.ctx, collName)
	require.NoError(t, err)

	names := make([]string, 0)
	for _, f := range fields {
		names = append(names, f.Name)
	}
	assert.Contains(t, names, "f1")
	assert.Contains(t, names, "f2")
	assert.Contains(t, names, "id")
}

func TestRemoveField_Success(t *testing.T) {
	s := setupTest(t)

	collName := "test_remove_field"
	_ = s.createTestCollection(t, collName, client.CreateFieldInput{
		Name: "to_remove",
		Type: "string",
	})

	fields, err := s.client.RemoveField(s.ctx, collName, "to_remove")
	require.NoError(t, err)

	names := make([]string, 0)
	for _, f := range fields {
		names = append(names, f.Name)
	}
	assert.NotContains(t, names, "to_remove")
}
