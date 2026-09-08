package middleware

import (
	"testing"

	"github.com/stretchr/testify/require"

	modelmd "itcodex/server/internal/model/metadata"
)

func TestCustomActionIdentity(t *testing.T) {
	resourceName, actionName, configured := customActionIdentity(&modelmd.YaegiScript{
		CollectionName: "posts",
		Name:           "发布",
		Options:        `{"resourceName":"posts","actionName":"publish"}`,
	})
	require.True(t, configured)
	require.Equal(t, "posts", resourceName)
	require.Equal(t, "publish", actionName)

	resourceName, actionName, configured = customActionIdentity(&modelmd.YaegiScript{
		CollectionName: "posts",
		Name:           "legacy",
	})
	require.False(t, configured)
	require.Equal(t, "posts", resourceName)
	require.Equal(t, "custom:legacy", actionName)
}
