package client

import (
	"context"
	"fmt"
)

type TokenPair struct {
	AccessToken      string `json:"accessToken"`
	RefreshToken     string `json:"refreshToken"`
	TokenType        string `json:"tokenType"`
	ExpiresIn        int64  `json:"expiresIn"`
	RefreshExpiresIn int64  `json:"refreshExpiresIn"`
}

type CurrentUser struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"displayName"`
	Roles       []string `json:"roles"`
}

func (c *Client) Login(ctx context.Context, username, password string) (*TokenPair, error) {
	raw, err := c.post(ctx, "/api/auth/login", map[string]any{
		"username": username,
		"password": password,
	})
	if err != nil {
		return nil, err
	}
	response, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid login response")
	}
	data, ok := response["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid login response data")
	}
	pair := &TokenPair{
		AccessToken:      toString(data["accessToken"]),
		RefreshToken:     toString(data["refreshToken"]),
		TokenType:        toString(data["tokenType"]),
		ExpiresIn:        asInt64(data["expiresIn"]),
		RefreshExpiresIn: asInt64(data["refreshExpiresIn"]),
	}
	c.AccessToken = pair.AccessToken
	return pair, nil
}

func (c *Client) SetAccessToken(token string) {
	c.AccessToken = token
}

func (c *Client) Me(ctx context.Context) (*CurrentUser, error) {
	raw, err := c.get(ctx, "/api/auth/me", nil)
	if err != nil {
		return nil, err
	}
	response, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid me response")
	}
	data, ok := response["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid me response data")
	}
	userData, ok := data["user"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid user response")
	}
	user := &CurrentUser{
		ID:          toString(userData["id"]),
		Username:    toString(userData["username"]),
		DisplayName: toString(userData["displayName"]),
	}
	if roles, ok := userData["roles"].([]any); ok {
		for _, role := range roles {
			user.Roles = append(user.Roles, toString(role))
		}
	}
	return user, nil
}
