package token

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kaushlender/auth-service/internal/config"
)

func setupManager() *Manager {
	cfg := &config.JWTConfig{
		AccessSecret:    "test-access-secret-key-1234567890",
		RefreshSecret:   "test-refresh-secret-key-1234567890",
		AccessDuration:  15 * time.Minute,
		RefreshDuration: 7 * 24 * time.Hour,
		Issuer:          "test-auth-service",
	}
	return NewManager(cfg)
}

func TestGenerateAccessToken(t *testing.T) {
	m := setupManager()
	userID := uuid.New()
	email := "test@example.com"
	role := "user"

	token, err := m.GenerateAccessToken(userID, email, role)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestValidateAccessToken(t *testing.T) {
	m := setupManager()
	userID := uuid.New()
	email := "test@example.com"
	role := "user"

	token, err := m.GenerateAccessToken(userID, email, role)
	require.NoError(t, err)

	claims, err := m.ValidateAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, role, claims.Role)
}

func TestValidateAccessToken_Invalid(t *testing.T) {
	m := setupManager()

	_, err := m.ValidateAccessToken("invalid-token")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
}

func TestGenerateRefreshToken(t *testing.T) {
	m := setupManager()
	userID := uuid.New()

	token, err := m.GenerateRefreshToken(userID)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestValidateRefreshToken(t *testing.T) {
	m := setupManager()
	userID := uuid.New()

	token, err := m.GenerateRefreshToken(userID)
	require.NoError(t, err)

	claims, err := m.ValidateRefreshToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
}

func TestGenerateTokenPair(t *testing.T) {
	m := setupManager()
	userID := uuid.New()
	email := "test@example.com"
	role := "user"

	pair, err := m.GenerateTokenPair(userID, email, role)
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
}

func TestAccessTokenExpiry(t *testing.T) {
	cfg := &config.JWTConfig{
		AccessSecret:    "test-access-secret-key-1234567890",
		RefreshSecret:   "test-refresh-secret-key-1234567890",
		AccessDuration:  1 * time.Nanosecond, // Very short expiry
		RefreshDuration: 7 * 24 * time.Hour,
		Issuer:          "test-auth-service",
	}
	m := NewManager(cfg)

	userID := uuid.New()
	token, err := m.GenerateAccessToken(userID, "test@example.com", "user")
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	_, err = m.ValidateAccessToken(token)
	assert.Error(t, err)
	assert.Equal(t, ErrExpiredToken, err)
}

func TestTokenWithWrongSecret(t *testing.T) {
	cfg1 := &config.JWTConfig{
		AccessSecret:    "secret-1",
		RefreshSecret:   "refresh-secret-1",
		AccessDuration:  15 * time.Minute,
		RefreshDuration: 7 * 24 * time.Hour,
		Issuer:          "test",
	}
	cfg2 := &config.JWTConfig{
		AccessSecret:    "secret-2",
		RefreshSecret:   "refresh-secret-2",
		AccessDuration:  15 * time.Minute,
		RefreshDuration: 7 * 24 * time.Hour,
		Issuer:          "test",
	}

	m1 := NewManager(cfg1)
	m2 := NewManager(cfg2)

	userID := uuid.New()
	token, err := m1.GenerateAccessToken(userID, "test@example.com", "user")
	require.NoError(t, err)

	_, err = m2.ValidateAccessToken(token)
	assert.Error(t, err)
}