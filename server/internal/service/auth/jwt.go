package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const maxTokenLength = 8192

type tokenClaims struct {
	Issuer    string   `json:"iss"`
	Subject   string   `json:"sub"`
	Username  string   `json:"username"`
	Roles     []string `json:"roles,omitempty"`
	TokenType string   `json:"token_type"`
	ID        string   `json:"jti"`
	IssuedAt  int64    `json:"iat"`
	ExpiresAt int64    `json:"exp"`
}

type jwtHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

func signJWT(secret []byte, claims tokenClaims) (string, error) {
	headerJSON, err := json.Marshal(jwtHeader{Algorithm: "HS256", Type: "JWT"})
	if err != nil {
		return "", fmt.Errorf("auth: encode JWT header: %w", err)
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("auth: encode JWT claims: %w", err)
	}
	unsigned := base64.RawURLEncoding.EncodeToString(headerJSON) + "." +
		base64.RawURLEncoding.EncodeToString(claimsJSON)
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(unsigned))
	signature := mac.Sum(nil)
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func parseJWT(secret []byte, issuer, expectedType, token string, now time.Time) (tokenClaims, error) {
	if token == "" || len(token) > maxTokenLength {
		return tokenClaims{}, ErrInvalidToken
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return tokenClaims{}, ErrInvalidToken
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return tokenClaims{}, ErrInvalidToken
	}
	var header jwtHeader
	if err = json.Unmarshal(headerBytes, &header); err != nil ||
		header.Algorithm != "HS256" || header.Type != "JWT" {
		return tokenClaims{}, ErrInvalidToken
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(signature) != sha256.Size {
		return tokenClaims{}, ErrInvalidToken
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	if subtle.ConstantTimeCompare(mac.Sum(nil), signature) != 1 {
		return tokenClaims{}, ErrInvalidToken
	}

	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return tokenClaims{}, ErrInvalidToken
	}
	var claims tokenClaims
	if err = json.Unmarshal(claimsBytes, &claims); err != nil {
		return tokenClaims{}, ErrInvalidToken
	}
	nowUnix := now.Unix()
	if claims.Issuer != issuer ||
		claims.Subject == "" ||
		claims.ID == "" ||
		claims.TokenType != expectedType ||
		claims.IssuedAt <= 0 ||
		claims.ExpiresAt <= claims.IssuedAt ||
		claims.ExpiresAt <= nowUnix ||
		claims.IssuedAt > now.Add(time.Minute).Unix() {
		return tokenClaims{}, ErrInvalidToken
	}
	return claims, nil
}
