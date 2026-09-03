package tests

import (
	"net/http"
	"testing"

	"itcodex/client/internal/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidation_Required(t *testing.T) {
	s := setupTest(t)

	collName := "test_val_required"
	s.createTestCollection(t, collName,
		client.CreateFieldInput{Name: "title", Type: "string", Required: true},
	)

	_, err := s.client.Create(s.ctx, collName, map[string]interface{}{
		"other": "value",
	})
	require.Error(t, err)
	assert.True(t, s.isAPIError(err, http.StatusUnprocessableEntity))
}

func TestValidation_Unique(t *testing.T) {
	s := setupTest(t)

	collName := "test_val_unique"
	s.createTestCollection(t, collName,
		client.CreateFieldInput{Name: "code", Type: "string", Required: true, Unique: true},
	)

	_, err := s.client.Create(s.ctx, collName, map[string]interface{}{
		"code": "UNIQUE001",
	})
	require.NoError(t, err)

	_, err = s.client.Create(s.ctx, collName, map[string]interface{}{
		"code": "UNIQUE001",
	})
	require.Error(t, err)
}

func TestValidation_Email(t *testing.T) {
	s := setupTest(t)

	collName := "test_val_email"
	s.createTestCollection(t, collName,
		client.CreateFieldInput{Name: "email", Type: "email", Required: true},
	)

	_, err := s.client.Create(s.ctx, collName, map[string]interface{}{
		"email": "not-an-email",
	})
	require.Error(t, err)
}

func TestCreate_WithAllFields(t *testing.T) {
	s := setupTest(t)

	collName := "test_val_all"
	s.createTestCollection(t, collName,
		client.CreateFieldInput{Name: "name", Type: "string", Required: true},
		client.CreateFieldInput{Name: "email", Type: "email"},
		client.CreateFieldInput{Name: "age", Type: "integer"},
		client.CreateFieldInput{Name: "active", Type: "boolean"},
	)

	record, err := s.client.Create(s.ctx, collName, map[string]interface{}{
		"name":   "Test User",
		"email":  "t***@example.com",
		"age":    25,
		"active": true,
	})
	require.NoError(t, err)
	assert.Equal(t, "Test User", record["name"])
	assert.Equal(t, "t***@example.com", record["email"])
}
