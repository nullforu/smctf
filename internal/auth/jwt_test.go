package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"strings"
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

func TestParseTokenInvalidToken(t *testing.T) {
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

func TestParseTokenWrongSecret(t *testing.T) {
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

	wrongCfg := cfg
	wrongCfg.Secret = "wrong-secret"

	_, err = ParseToken(wrongCfg, token)
	if err == nil {
		t.Error("expected error with wrong secret, got nil")
	}
}

func TestParseTokenWrongIssuer(t *testing.T) {
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

	wrongCfg := cfg
	wrongCfg.Issuer = "wrong-issuer"

	_, err = ParseToken(wrongCfg, token)
	if err != jwt.ErrTokenInvalidIssuer {
		t.Errorf("expected ErrTokenInvalidIssuer, got %v", err)
	}
}

func TestParseTokenExpiredToken(t *testing.T) {
	cfg := config.JWTConfig{
		Secret:     "test-secret",
		Issuer:     "test-issuer",
		AccessTTL:  -time.Hour,
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

func TestParseTokenUnexpectedSigningMethod(t *testing.T) {
	cfg := config.JWTConfig{
		Secret: "test-secret",
		Issuer: "test-issuer",
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate rsa key: %v", err)
	}

	claims := Claims{
		UserID: 1,
		Role:   "admin",
		Type:   TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenStr, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	_, err = ParseToken(cfg, tokenStr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "unexpected signing method") {
		t.Fatalf("expected unexpected signing method error, got %v", err)
	}
}
