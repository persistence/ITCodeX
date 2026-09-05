package tests

import (
	"testing"

	"itcodex/client/internal/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndex_CreateListDelete(t *testing.T) {
	s := setupTest(t)
	coll := uniqueName("idx_coll")
	s.createTestCollection(t, coll,
		client.CreateFieldInput{Name: "title", Type: "string", IsRequired: true},
		client.CreateFieldInput{Name: "code", Type: "string"},
	)

	idx, err := s.client.CreateIndex(s.ctx, coll, client.Index{
		Fields: []string{"title"},
		Unique: false,
	})
	require.NoError(t, err)
	require.NotNil(t, idx)

	list, err := s.client.ListIndexes(s.ctx, coll)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list), 1)

	err = s.client.DeleteIndex(s.ctx, coll, []string{"title"})
	require.NoError(t, err)
}
