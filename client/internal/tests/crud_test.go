package tests

import (
	"fmt"
	"net/http"
	"testing"

	"itcodex/client/internal/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreate_SingleRecord(t *testing.T) {
	s := setupTest(t)

	collName := "test_crud_create"
	s.createTestCollection(t, collName,
		client.CreateFieldInput{Name: "title", Type: "string", Required: true},
		client.CreateFieldInput{Name: "views", Type: "integer"},
	)

	record := s.createTestRecord(t, collName, map[string]interface{}{
		"title": "测试标题",
		"views": 100,
	})

	assert.NotZero(t, record["id"])
	assert.NotZero(t, record["created_at"])
	assert.NotZero(t, record["updated_at"])
	assert.Equal(t, "测试标题", record["title"])
}

func TestGet_Exists(t *testing.T) {
	s := setupTest(t)

	collName := "test_crud_get"
	s.createTestCollection(t, collName,
		client.CreateFieldInput{Name: "title", Type: "string", Required: true},
	)

	created := s.createTestRecord(t, collName, map[string]interface{}{
		"title": "获取测试",
	})

	id := fmt.Sprintf("%v", created["id"])
	fetched, err := s.client.Get(s.ctx, collName, id, nil)
	require.NoError(t, err)
	assert.Equal(t, "获取测试", fetched["title"])
	assert.Equal(t, created["id"], fetched["id"])
}

func TestGet_NotFound(t *testing.T) {
	s := setupTest(t)

	collName := "test_crud_get_nf"
	s.createTestCollection(t, collName,
		client.CreateFieldInput{Name: "title", Type: "string"},
	)

	_, err := s.client.Get(s.ctx, collName, "999999", nil)
	require.Error(t, err)
	assert.True(t, s.isAPIError(err, http.StatusNotFound))
}

func TestList_DefaultPagination(t *testing.T) {
	s := setupTest(t)

	collName := "test_crud_list"
	s.createTestCollection(t, collName,
		client.CreateFieldInput{Name: "title", Type: "string", Required: true},
	)

	for i := 0; i < 5; i++ {
		s.createTestRecord(t, collName, map[string]interface{}{
			"title": fmt.Sprintf("标题 %d", i),
		})
	}

	result, err := s.client.List(s.ctx, collName, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, result.Total, int64(5))
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 20, result.PageSize)
	assert.LessOrEqual(t, len(result.List), 20)
}

func TestList_CustomPagination(t *testing.T) {
	s := setupTest(t)

	collName := "test_crud_list_page"
	s.createTestCollection(t, collName,
		client.CreateFieldInput{Name: "title", Type: "string", Required: true},
	)

	for i := 0; i < 10; i++ {
		s.createTestRecord(t, collName, map[string]interface{}{
			"title": fmt.Sprintf("分页标题 %d", i),
		})
	}

	opts := &client.FindOptions{
		Page:     2,
		PageSize: 3,
	}
	result, err := s.client.List(s.ctx, collName, opts)
	require.NoError(t, err)
	assert.Equal(t, 2, result.Page)
	assert.Equal(t, 3, result.PageSize)
	assert.LessOrEqual(t, len(result.List), 3)
}

func TestUpdate_Success(t *testing.T) {
	s := setupTest(t)

	collName := "test_crud_update"
	s.createTestCollection(t, collName,
		client.CreateFieldInput{Name: "title", Type: "string", Required: true},
		client.CreateFieldInput{Name: "views", Type: "number"},
	)

	created := s.createTestRecord(t, collName, map[string]interface{}{
		"title": "原标题",
		"views": 0,
	})

	id := fmt.Sprintf("%v", created["id"])
	updated, err := s.client.Update(s.ctx, collName, id, map[string]interface{}{
		"title": "更新后的标题",
		"views": 200,
	})
	require.NoError(t, err)
	assert.Equal(t, "更新后的标题", updated["title"])

	fetched, err := s.client.Get(s.ctx, collName, id, nil)
	require.NoError(t, err)
	assert.Equal(t, "更新后的标题", fetched["title"])
	assert.Equal(t, float64(200), fetched["views"])
}

func TestUpdate_NotFound(t *testing.T) {
	s := setupTest(t)

	collName := "test_crud_update_nf"
	s.createTestCollection(t, collName,
		client.CreateFieldInput{Name: "title", Type: "string"},
	)

	_, err := s.client.Update(s.ctx, collName, "999999", map[string]interface{}{
		"title": "不存在",
	})
	require.Error(t, err)
}

func TestDelete_Success(t *testing.T) {
	s := setupTest(t)

	collName := "test_crud_delete"
	s.createTestCollection(t, collName,
		client.CreateFieldInput{Name: "title", Type: "string", Required: true},
	)

	created := s.createTestRecord(t, collName, map[string]interface{}{
		"title": "要删除的",
	})

	id := fmt.Sprintf("%v", created["id"])
	affected, err := s.client.DeleteOne(s.ctx, collName, id)
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)

	_, err = s.client.Get(s.ctx, collName, id, nil)
	require.Error(t, err)
}

func TestDelete_NotFound(t *testing.T) {
	s := setupTest(t)

	collName := "test_crud_delete_nf"
	s.createTestCollection(t, collName,
		client.CreateFieldInput{Name: "title", Type: "string"},
	)

	affected, err := s.client.DeleteOne(s.ctx, collName, "999999")
	require.NoError(t, err)
	assert.Equal(t, int64(0), affected)
}

func TestBulkDelete_WithFilter(t *testing.T) {
	s := setupTest(t)

	collName := "test_crud_bulk_delete"
	s.createTestCollection(t, collName,
		client.CreateFieldInput{Name: "status", Type: "string"},
	)

	for i := 0; i < 5; i++ {
		status := "active"
		if i < 2 {
			status = "deleted"
		}
		s.createTestRecord(t, collName, map[string]interface{}{
			"status": status,
		})
	}

	filter := client.Filter{"status": "deleted"}
	affected, err := s.client.BulkDelete(s.ctx, collName, filter)
	require.NoError(t, err)
	assert.Equal(t, int64(2), affected)
}

func TestCount_Success(t *testing.T) {
	s := setupTest(t)

	collName := "test_crud_count"
	s.createTestCollection(t, collName,
		client.CreateFieldInput{Name: "name", Type: "string", Required: true},
	)

	for i := 0; i < 8; i++ {
		s.createTestRecord(t, collName, map[string]interface{}{
			"name": fmt.Sprintf("计数 %d", i),
		})
	}

	count, err := s.client.Count(s.ctx, collName, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(8))
}
