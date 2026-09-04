package tests

import (
	"testing"

	"itcodex/client/internal/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFilterTestCollection(t *testing.T, s *TestSuite) string {
	collName := "test_filter_coll"
	s.createTestCollection(t, collName,
		client.CreateFieldInput{Name: "name", Type: "string", IsRequired: true},
		client.CreateFieldInput{Name: "age", Type: "integer"},
		client.CreateFieldInput{Name: "score", Type: "integer"},
		client.CreateFieldInput{Name: "status", Type: "string"},
	)

	testData := []map[string]interface{}{
		{"name": "Alice", "age": 25, "score": 85, "status": "active"},
		{"name": "Bob", "age": 30, "score": 92, "status": "active"},
		{"name": "Charlie", "age": 35, "score": 78, "status": "inactive"},
		{"name": "David", "age": 28, "score": 95, "status": "active"},
		{"name": "Eve", "age": 22, "score": 88, "status": "inactive"},
	}

	for _, d := range testData {
		s.createTestRecord(t, collName, d)
	}

	return collName
}

func TestFilter_Eq(t *testing.T) {
	s := setupTest(t)
	collName := setupFilterTestCollection(t, s)

	result, err := s.client.List(s.ctx, collName, &client.FindOptions{
		Filter: client.Filter{"name": "Alice"},
	})
	require.NoError(t, err)
	assert.Len(t, result.List, 1)
	assert.Equal(t, "Alice", result.List[0]["name"])
}

func TestFilter_Ne(t *testing.T) {
	s := setupTest(t)
	collName := setupFilterTestCollection(t, s)

	result, err := s.client.List(s.ctx, collName, &client.FindOptions{
		Filter: client.Filter{"status": map[string]interface{}{"$ne": "active"}},
	})
	require.NoError(t, err)
	for _, r := range result.List {
		assert.NotEqual(t, "active", r["status"])
	}
}

func TestFilter_Gt(t *testing.T) {
	s := setupTest(t)
	collName := setupFilterTestCollection(t, s)

	result, err := s.client.List(s.ctx, collName, &client.FindOptions{
		Filter: client.Filter{"age": map[string]interface{}{"$gt": 30}},
	})
	require.NoError(t, err)
	for _, r := range result.List {
		assert.Greater(t, asFloat64(r["age"]), float64(30))
	}
}

func TestFilter_Gte(t *testing.T) {
	s := setupTest(t)
	collName := setupFilterTestCollection(t, s)

	result, err := s.client.List(s.ctx, collName, &client.FindOptions{
		Filter: client.Filter{"age": map[string]interface{}{"$gte": 30}},
	})
	require.NoError(t, err)
	for _, r := range result.List {
		assert.GreaterOrEqual(t, asFloat64(r["age"]), float64(30))
	}
}

func TestFilter_Lt(t *testing.T) {
	s := setupTest(t)
	collName := setupFilterTestCollection(t, s)

	result, err := s.client.List(s.ctx, collName, &client.FindOptions{
		Filter: client.Filter{"age": map[string]interface{}{"$lt": 28}},
	})
	require.NoError(t, err)
	for _, r := range result.List {
		assert.Less(t, asFloat64(r["age"]), float64(28))
	}
}

func TestFilter_Lte(t *testing.T) {
	s := setupTest(t)
	collName := setupFilterTestCollection(t, s)

	result, err := s.client.List(s.ctx, collName, &client.FindOptions{
		Filter: client.Filter{"age": map[string]interface{}{"$lte": 28}},
	})
	require.NoError(t, err)
	for _, r := range result.List {
		assert.LessOrEqual(t, asFloat64(r["age"]), float64(28))
	}
}

func TestFilter_In(t *testing.T) {
	s := setupTest(t)
	collName := setupFilterTestCollection(t, s)

	result, err := s.client.List(s.ctx, collName, &client.FindOptions{
		Filter: client.Filter{"name": map[string]interface{}{"$in": []string{"Alice", "Bob"}}},
	})
	require.NoError(t, err)
	assert.Len(t, result.List, 2)
	names := make(map[string]bool)
	for _, r := range result.List {
		names[r["name"].(string)] = true
	}
	assert.True(t, names["Alice"])
	assert.True(t, names["Bob"])
}

func TestFilter_Like(t *testing.T) {
	s := setupTest(t)
	collName := setupFilterTestCollection(t, s)

	result, err := s.client.List(s.ctx, collName, &client.FindOptions{
		Filter: client.Filter{"name": map[string]interface{}{"$like": "A%"}},
	})
	require.NoError(t, err)
	for _, r := range result.List {
		assert.Contains(t, r["name"].(string), "A")
	}
}

func TestFilter_StartsWith(t *testing.T) {
	s := setupTest(t)
	collName := setupFilterTestCollection(t, s)

	result, err := s.client.List(s.ctx, collName, &client.FindOptions{
		Filter: client.Filter{"name": map[string]interface{}{"$startsWith": "A"}},
	})
	require.NoError(t, err)
	for _, r := range result.List {
		assert.True(t, len(r["name"].(string)) > 0 && r["name"].(string)[0] == 'A')
	}
}

func TestFilter_EndsWith(t *testing.T) {
	s := setupTest(t)
	collName := setupFilterTestCollection(t, s)

	result, err := s.client.List(s.ctx, collName, &client.FindOptions{
		Filter: client.Filter{"name": map[string]interface{}{"$endsWith": "e"}},
	})
	require.NoError(t, err)
	for _, r := range result.List {
		name := r["name"].(string)
		assert.True(t, len(name) > 0 && name[len(name)-1] == 'e')
	}
}

func TestSort_Ascending(t *testing.T) {
	s := setupTest(t)
	collName := setupFilterTestCollection(t, s)

	result, err := s.client.List(s.ctx, collName, &client.FindOptions{
		Sort: []string{"age"},
	})
	require.NoError(t, err)
	for i := 1; i < len(result.List); i++ {
		assert.LessOrEqual(t, asFloat64(result.List[i-1]["age"]), asFloat64(result.List[i]["age"]))
	}
}

func TestSort_Descending(t *testing.T) {
	s := setupTest(t)
	collName := setupFilterTestCollection(t, s)

	result, err := s.client.List(s.ctx, collName, &client.FindOptions{
		Sort: []string{"-age"},
	})
	require.NoError(t, err)
	for i := 1; i < len(result.List); i++ {
		assert.GreaterOrEqual(t, asFloat64(result.List[i-1]["age"]), asFloat64(result.List[i]["age"]))
	}
}

func TestFields_SelectFields(t *testing.T) {
	s := setupTest(t)
	collName := setupFilterTestCollection(t, s)

	result, err := s.client.List(s.ctx, collName, &client.FindOptions{
		Fields: []string{"name", "age"},
	})
	require.NoError(t, err)
	require.Greater(t, len(result.List), 0)
	record := result.List[0]
	assert.Contains(t, record, "name")
	assert.Contains(t, record, "age")
	assert.Contains(t, record, "id")
	assert.NotContains(t, record, "score")
}

func TestFields_ExceptFields(t *testing.T) {
	s := setupTest(t)
	collName := setupFilterTestCollection(t, s)

	result, err := s.client.List(s.ctx, collName, &client.FindOptions{
		Except: []string{"score", "status"},
	})
	require.NoError(t, err)
	require.Greater(t, len(result.List), 0)
	record := result.List[0]
	assert.Contains(t, record, "name")
	assert.Contains(t, record, "age")
	assert.NotContains(t, record, "score")
}

func TestFilter_Count(t *testing.T) {
	s := setupTest(t)
	collName := setupFilterTestCollection(t, s)

	count, err := s.client.Count(s.ctx, collName, client.Filter{"status": "active"})
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}
