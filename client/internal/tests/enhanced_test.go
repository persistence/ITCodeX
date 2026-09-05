package tests

import (
	"testing"

	"itcodex/client/internal/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestField_SequenceUUIDEncrypted(t *testing.T) {
	s := setupTest(t)
	coll := uniqueName("adv_fields")
	s.createTestCollection(t, coll,
		client.CreateFieldInput{Name: "code", Type: "sequence", Pattern: "ORD-{YYYY}-{0000}", StartsAt: 1},
		client.CreateFieldInput{Name: "uid", Type: "uuid"},
		client.CreateFieldInput{Name: "secret", Type: "encrypted"},
	)

	rec := s.createTestRecord(t, coll, map[string]interface{}{
		"secret": "hello-secret",
	})
	assert.NotEmpty(t, rec["code"])
	assert.NotEmpty(t, rec["uid"])
	assert.Equal(t, "hello-secret", rec["secret"])
}

func TestField_PointGeo(t *testing.T) {
	s := setupTest(t)
	coll := uniqueName("geo_fields")
	s.createTestCollection(t, coll,
		client.CreateFieldInput{Name: "loc", Type: "point"},
	)

	rec := s.createTestRecord(t, coll, map[string]interface{}{
		"loc": map[string]interface{}{
			"type":        "Point",
			"coordinates": []interface{}{116.4, 39.9},
		},
	})
	assert.NotNil(t, rec["loc"])
}

func TestCollection_TreeChildren(t *testing.T) {
	s := setupTest(t)
	coll := uniqueName("tree_nodes")
	c := s.createSpecialCollection(t, coll, "tree",
		client.CreateFieldInput{Name: "title", Type: "string"},
	)
	require.NotNil(t, c)

	fields, err := s.client.ListFields(s.ctx, coll)
	require.NoError(t, err)
	hasParent := false
	for _, f := range fields {
		if f.Name == "parent_id" {
			hasParent = true
			break
		}
	}
	assert.True(t, hasParent, "tree collection should have parent_id")

	root := s.createTestRecord(t, coll, map[string]interface{}{"title": "root"})
	_ = s.createTestRecord(t, coll, map[string]interface{}{
		"title": "child", "parent_id": root["id"],
	})

	got, err := s.client.FindOne(s.ctx, coll, idStr(root["id"]), &client.FindOneOptions{
		Appends: []string{"children"},
	})
	require.NoError(t, err)
	children := got["children"]
	assert.NotNil(t, children)
}

func TestCollection_FileDefaults(t *testing.T) {
	s := setupTest(t)
	coll := uniqueName("files")
	s.createSpecialCollection(t, coll, "file")

	fields, err := s.client.ListFields(s.ctx, coll)
	require.NoError(t, err)
	names := map[string]bool{}
	for _, f := range fields {
		names[f.Name] = true
	}
	for _, n := range []string{"name", "url", "mime", "size"} {
		assert.True(t, names[n], "file collection missing field %s", n)
	}
}
