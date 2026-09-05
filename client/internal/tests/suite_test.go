package tests

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"itcodex/client/internal/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestSuite struct {
	client *client.Client
	ctx    context.Context
	t      *testing.T
}

func setupTest(t *testing.T) *TestSuite {
	t.Helper()
	c := client.NewClient("")
	ctx := context.Background()

	// Skip entire suite when server is unreachable
	if _, err := c.ListCollections(ctx); err != nil {
		t.Skipf("metadata server unreachable at %s: %v", c.BaseURL, err)
	}

	return &TestSuite{
		client: c,
		ctx:    ctx,
		t:      t,
	}
}

func uniqueName(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano()%1_000_000_000)
}

func (s *TestSuite) createTestCollection(t *testing.T, name string, fields ...client.CreateFieldInput) *client.Collection {
	t.Helper()
	if fields == nil {
		fields = []client.CreateFieldInput{}
	}

	input := client.CreateCollectionInput{
		Name:        name,
		DisplayName: fmt.Sprintf("测试集合 %s", name),
		Type:        "general",
		Fields:      fields,
	}

	coll, err := s.client.CreateCollection(s.ctx, input)
	require.NoError(t, err)
	require.NotNil(t, coll)

	t.Cleanup(func() {
		_ = s.client.DropCollection(s.ctx, name)
	})

	return coll
}

func (s *TestSuite) createSpecialCollection(t *testing.T, name, typ string, fields ...client.CreateFieldInput) *client.Collection {
	t.Helper()
	input := client.CreateCollectionInput{
		Name:        name,
		DisplayName: name,
		Type:        typ,
		Fields:      fields,
	}
	coll, err := s.client.CreateCollection(s.ctx, input)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = s.client.DropCollection(s.ctx, name)
	})
	return coll
}

func (s *TestSuite) cleanupTestCollection(t *testing.T, name string) {
	err := s.client.DropCollection(s.ctx, name)
	assert.NoError(t, err)
}

func (s *TestSuite) createTestRecord(t *testing.T, collection string, data map[string]interface{}) map[string]interface{} {
	t.Helper()
	record, err := s.client.Create(s.ctx, collection, data)
	require.NoError(t, err)
	require.NotNil(t, record)
	require.NotZero(t, record["id"])
	return record
}

func (s *TestSuite) isAPIError(err error, statusCode int) bool {
	if err == nil {
		return false
	}
	apiErr, ok := err.(*client.APIError)
	if !ok {
		return false
	}
	return apiErr.Code == statusCode
}

func asFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case int32:
		return float64(n)
	}
	return 0
}

func idStr(v interface{}) string {
	return fmt.Sprintf("%v", v)
}

var _ = http.StatusOK
