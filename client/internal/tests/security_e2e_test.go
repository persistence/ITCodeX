package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"itcodex/client/internal/client"
)

func TestSecurityAnonymousCRUDDenied(t *testing.T) {
	baseURL := os.Getenv("TEST_SERVER_URL")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8000"
	}
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		strings.TrimRight(baseURL, "/")+"/api/c/security_probe",
		nil,
	)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("metadata server unreachable: %v", err)
	}
	defer resp.Body.Close()
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.EqualValues(t, http.StatusForbidden, body["code"])
}

func TestSecurityLoginAndMe(t *testing.T) {
	username := os.Getenv("TEST_USERNAME")
	password := os.Getenv("TEST_PASSWORD")
	if username == "" || password == "" {
		t.Skip("TEST_USERNAME/TEST_PASSWORD not configured")
	}
	c := client.NewClient("")
	pair, err := c.Login(context.Background(), username, password)
	if err != nil {
		t.Skipf("metadata login unavailable: %v", err)
	}
	require.NotEmpty(t, pair.AccessToken)
	require.NotEmpty(t, pair.RefreshToken)
	user, err := c.Me(context.Background())
	require.NoError(t, err)
	require.Equal(t, username, user.Username)
}
