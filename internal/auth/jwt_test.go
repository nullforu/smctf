package auth

import (
	"testing"
	"time"

	"smctf/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateAccessToken(t *testing.T) {
	cfg := config.JWTConfig{
		Secret:     "test-secret",
		Issuer:     "test-issuer",
		AccessTTL:  time.Hour,
		RefreshTTL: 24 * time.Hour,
	}

	token, err := GenerateAccessToken(cfg, 42, "admin")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Parse and verify
	claims, err := ParseToken(cfg, token)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}

	if claims.UserID != 42 {
		t.Errorf("expected UserID 42, got %d", claims.UserID)
	}

	if claims.Role != "admin" {
		t.Errorf("expected Role admin, got %s", claims.Role)
	}

	if claims.Type != TokenTypeAccess {
		t.Errorf("expected Type %s, got %s", TokenTypeAccess, claims.Type)
	}

	if claims.Issuer != cfg.Issuer {
		t.Errorf("expected Issuer %s, got %s", cfg.Issuer, claims.Issuer)
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	cfg := config.JWTConfig{
		Secret:     "test-secret",
		Issuer:     "test-issuer",
		AccessTTL:  time.Hour,
		RefreshTTL: 24 * time.Hour,
	}

	jti := "test-jti-123"
	token, err := GenerateRefreshToken(cfg, 42, "user", jti)
	if err != nil {
		t.Fatalf("GenerateRefreshToken failed: %v", err)
	}

	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Parse and verify
	claims, err := ParseToken(cfg, token)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}

	if claims.UserID != 42 {
		t.Errorf("expected UserID 42, got %d", claims.UserID)
	}

	if claims.Role != "user" {
		t.Errorf("expected Role user, got %s", claims.Role)
	}

	if claims.Type != TokenTypeRefresh {
		t.Errorf("expected Type %s, got %s", TokenTypeRefresh, claims.Type)
	}

	if claims.ID != jti {
		t.Errorf("expected JTI %s, got %s", jti, claims.ID)
	}

	if claims.Issuer != cfg.Issuer {
		t.Errorf("expected Issuer %s, got %s", cfg.Issuer, claims.Issuer)
	}
}

func TestParseToken_InvalidToken(t *testing.T) {
	cfg := config.JWTConfig{
		Secret:     "test-secret",
		Issuer:     "test-issuer",
		AccessTTL:  time.Hour,
		RefreshTTL: 24 * time.Hour,
	}

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"invalid format", "invalid.token.format"},
		{"malformed", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid.sig"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseToken(cfg, tt.token)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestParseToken_WrongSecret(t *testing.T) {
	cfg := config.JWTConfig{
		Secret:     "test-secret",
		Issuer:     "test-issuer",
		AccessTTL:  time.Hour,
		RefreshTTL: 24 * time.Hour,
	}

	token, err := GenerateAccessToken(cfg, 42, "admin")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	// Try to parse with different secret
	wrongCfg := cfg
	wrongCfg.Secret = "wrong-secret"

	_, err = ParseToken(wrongCfg, token)
	if err == nil {
		t.Error("expected error with wrong secret, got nil")
	}
}

func TestParseToken_WrongIssuer(t *testing.T) {
	cfg := config.JWTConfig{
		Secret:     "test-secret",
		Issuer:     "test-issuer",
		AccessTTL:  time.Hour,
		RefreshTTL: 24 * time.Hour,
	}

	token, err := GenerateAccessToken(cfg, 42, "admin")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	// Try to parse with different issuer
	wrongCfg := cfg
	wrongCfg.Issuer = "wrong-issuer"

	_, err = ParseToken(wrongCfg, token)
	if err != jwt.ErrTokenInvalidIssuer {
		t.Errorf("expected ErrTokenInvalidIssuer, got %v", err)
	}
}

func TestParseToken_ExpiredToken(t *testing.T) {
	cfg := config.JWTConfig{
		Secret:     "test-secret",
		Issuer:     "test-issuer",
		AccessTTL:  -time.Hour, // expired
		RefreshTTL: 24 * time.Hour,
	}

	token, err := GenerateAccessToken(cfg, 42, "admin")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	_, err = ParseToken(cfg, token)
	if err == nil {
		t.Error("expected error for expired token, got nil")
	}
}

func TestTokenTypes(t *testing.T) {
	if TokenTypeAccess != "access" {
		t.Errorf("expected TokenTypeAccess to be 'access', got %s", TokenTypeAccess)
	}

	if TokenTypeRefresh != "refresh" {
		t.Errorf("expected TokenTypeRefresh to be 'refresh', got %s", TokenTypeRefresh)
	}
}
