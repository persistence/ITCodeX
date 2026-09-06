package tests

import (
	"testing"

	"itcodex/client/internal/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScript_ValidateAndLifecycle(t *testing.T) {
	s := setupTest(t)
	coll := uniqueName("script_coll")
	s.createTestCollection(t, coll,
		client.CreateFieldInput{Name: "title", Type: "string"},
		client.CreateFieldInput{Name: "auto_field", Type: "string"},
	)

	const content = `
package main

import "context"

func BeforeCreate(ctx context.Context, data map[string]any) (map[string]any, error) {
	data["auto_field"] = "from_hook"
	return data, nil
}
`
	vr, err := s.client.ValidateScript(s.ctx, content)
	require.NoError(t, err)
	assert.True(t, vr.Valid, vr.Error)

	script, err := s.client.CreateScript(s.ctx, client.CreateScriptInput{
		Name:           "before_create_hook",
		Content:        content,
		HookPoint:      "beforeCreate",
		CollectionName: coll,
		Enabled:        true,
	})
	require.NoError(t, err)
	require.NotZero(t, script.ID)
	t.Cleanup(func() {
		_ = s.client.DeleteScript(s.ctx, script.ID)
	})

	rec := s.createTestRecord(t, coll, map[string]any{"title": "t"})
	assert.Equal(t, "from_hook", rec["auto_field"])

	err = s.client.DisableScript(s.ctx, script.ID)
	require.NoError(t, err)

	err = s.client.DeleteScript(s.ctx, script.ID)
	require.NoError(t, err)
}
