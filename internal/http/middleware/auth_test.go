package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"smctf/internal/auth"
	"smctf/internal/config"

	"github.com/gin-gonic/gin"
)

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.JWTConfig{
		Secret:     "secret",
		Issuer:     "issuer",
		AccessTTL:  time.Hour,
		RefreshTTL: time.Hour,
	}

	if UserID(&gin.Context{}) != 0 {
		t.Fatalf("expected 0, got %d", UserID(&gin.Context{}))
	}

	if Role(&gin.Context{}) != "" {
		t.Fatalf("expected empty role, got %s", Role(&gin.Context{}))
	}

	router := gin.New()
	router.GET("/protected", Auth(cfg), func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"user_id": UserID(ctx),
			"role":    Role(ctx),
		})
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)

	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Token abc")

	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token")

	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	refresh, err := auth.GenerateRefreshToken(cfg, 42, "user", "jti-1")
	if err != nil {
		t.Fatalf("refresh token: %v", err)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+refresh)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	access, err := auth.GenerateAccessToken(cfg, 42, "admin")
	if err != nil {
		t.Fatalf("access token: %v", err)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+access)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequireRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.JWTConfig{
		Secret:     "secret",
		Issuer:     "issuer",
		AccessTTL:  time.Hour,
		RefreshTTL: time.Hour,
	}

	router := gin.New()
	router.GET("/admin", Auth(cfg), RequireRole("admin"), func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	userToken, err := auth.GenerateAccessToken(cfg, 1, "user")
	if err != nil {
		t.Fatalf("user token: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}

	adminToken, err := auth.GenerateAccessToken(cfg, 1, "admin")
	if err != nil {
		t.Fatalf("admin token: %v", err)
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
