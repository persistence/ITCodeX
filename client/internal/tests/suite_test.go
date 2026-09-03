package tests

import (
	"context"
	"fmt"
	"testing"

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
	c := client.NewClient("")
	ctx := context.Background()

	return &TestSuite{
		client: c,
		ctx:    ctx,
		t:      t,
	}
}

func (s *TestSuite) createTestCollection(t *testing.T, name string, fields ...client.CreateFieldInput) *client.Collection {
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

func (s *TestSuite) cleanupTestCollection(t *testing.T, name string) {
	err := s.client.DropCollection(s.ctx, name)
	assert.NoError(t, err)
}

func (s *TestSuite) createTestRecord(t *testing.T, collection string, data map[string]interface{}) map[string]interface{} {
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
