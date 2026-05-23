package handlers

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"smctf/internal/config"

	"github.com/gin-gonic/gin"
)

func newCookieTestContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return ctx, rec
}

func testConfig(appEnv string, accessTTL, refreshTTL time.Duration) config.Config {
	return config.Config{
		AppEnv: appEnv,
		JWT: config.JWTConfig{
			AccessTTL:  accessTTL,
			RefreshTTL: refreshTTL,
		},
	}
}

func cookieHeaderByPrefix(values []string, prefix string) string {
	for _, v := range values {
		if strings.HasPrefix(v, prefix) {
			return v
		}
	}
	return ""
}

func TestSetAuthCookiesGeneratesCookiesAndHeaders(t *testing.T) {
	ctx, rec := newCookieTestContext(t)
	cfg := testConfig("local", time.Hour, 2*time.Hour)

	if err := setAuthCookies(ctx, cfg, "access-abc", "refresh-def"); err != nil {
		t.Fatalf("setAuthCookies: %v", err)
	}

	cookies := rec.Header().Values("Set-Cookie")
	if len(cookies) != 3 {
		t.Fatalf("expected 3 cookies, got %d", len(cookies))
	}

	access := cookieHeaderByPrefix(cookies, accessTokenCookieName+"=")
	if access == "" || !strings.Contains(access, "access-abc") || !strings.Contains(access, "HttpOnly") {
		t.Fatalf("unexpected access cookie: %q", access)
	}

	refresh := cookieHeaderByPrefix(cookies, refreshTokenCookieName+"=")
	if refresh == "" || !strings.Contains(refresh, "refresh-def") || !strings.Contains(refresh, "HttpOnly") {
		t.Fatalf("unexpected refresh cookie: %q", refresh)
	}

	csrf := cookieHeaderByPrefix(cookies, csrfTokenCookieName+"=")
	if csrf == "" || strings.Contains(csrf, "HttpOnly") {
		t.Fatalf("unexpected csrf cookie: %q", csrf)
	}

	csrfHeader := rec.Header().Get(csrfTokenHeaderName)
	if csrfHeader == "" {
		t.Fatal("expected csrf response header")
	}

	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("expected Cache-Control no-store, got %q", rec.Header().Get("Cache-Control"))
	}

	if rec.Header().Get("Pragma") != "no-cache" {
		t.Fatalf("expected Pragma no-cache, got %q", rec.Header().Get("Pragma"))
	}
}

func TestSetAuthCookiesReusesExistingCSRFToken(t *testing.T) {
	ctx, rec := newCookieTestContext(t)
	cfg := testConfig("local", 2*time.Hour, time.Hour)

	ctx.Request.AddCookie(&http.Cookie{Name: csrfTokenCookieName, Value: "persist-csrf"})
	if err := setAuthCookies(ctx, cfg, "access", "refresh"); err != nil {
		t.Fatalf("setAuthCookies: %v", err)
	}

	csrfHeader := rec.Header().Get(csrfTokenHeaderName)
	if csrfHeader != "persist-csrf" {
		t.Fatalf("expected existing csrf token, got %q", csrfHeader)
	}

	cookies := rec.Header().Values("Set-Cookie")
	csrf := cookieHeaderByPrefix(cookies, csrfTokenCookieName+"=")
	if !strings.Contains(csrf, "persist-csrf") {
		t.Fatalf("expected persisted csrf cookie, got %q", csrf)
	}
	if !strings.Contains(csrf, "Max-Age=7200") {
		t.Fatalf("expected csrf cookie max-age to use larger ttl, got %q", csrf)
	}
}

func TestClearAuthCookies(t *testing.T) {
	ctx, rec := newCookieTestContext(t)
	cfg := testConfig("local", time.Hour, time.Hour)

	clearAuthCookies(ctx, cfg)

	cookies := rec.Header().Values("Set-Cookie")
	if len(cookies) != 3 {
		t.Fatalf("expected 3 cookies, got %d", len(cookies))
	}

	for _, name := range []string{accessTokenCookieName, refreshTokenCookieName, csrfTokenCookieName} {
		v := cookieHeaderByPrefix(cookies, name+"=")
		if v == "" || !strings.Contains(v, "Max-Age=0") {
			t.Fatalf("expected clearing cookie for %s, got %q", name, v)
		}
	}
}

func TestCurrentTokenReaders(t *testing.T) {
	t.Run("refresh missing", func(t *testing.T) {
		ctx, _ := newCookieTestContext(t)
		if v, ok := currentRefreshToken(ctx); ok || v != "" {
			t.Fatalf("expected missing refresh token")
		}
	})

	t.Run("refresh exists", func(t *testing.T) {
		ctx, _ := newCookieTestContext(t)
		ctx.Request.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: "ref"})
		if v, ok := currentRefreshToken(ctx); !ok || v != "ref" {
			t.Fatalf("expected refresh token ref, got %q ok=%v", v, ok)
		}
	})

	t.Run("csrf missing", func(t *testing.T) {
		ctx, _ := newCookieTestContext(t)
		if v, ok := currentCSRFToken(ctx); ok || v != "" {
			t.Fatalf("expected missing csrf token")
		}
	})

	t.Run("csrf exists", func(t *testing.T) {
		ctx, _ := newCookieTestContext(t)
		ctx.Request.AddCookie(&http.Cookie{Name: csrfTokenCookieName, Value: "csrf"})
		if v, ok := currentCSRFToken(ctx); !ok || v != "csrf" {
			t.Fatalf("expected csrf token csrf, got %q ok=%v", v, ok)
		}
	})
}

func TestSetCookieProductionSecureFlag(t *testing.T) {
	ctx, rec := newCookieTestContext(t)
	cfg := testConfig("production", time.Hour, time.Hour)

	setCookie(ctx, cfg, "x", "y", 10, true)

	cookie := cookieHeaderByPrefix(rec.Header().Values("Set-Cookie"), "x=")
	if !strings.Contains(cookie, "Secure") {
		t.Fatalf("expected secure cookie in production, got %q", cookie)
	}

	if !strings.Contains(cookie, "HttpOnly") {
		t.Fatalf("expected httponly cookie, got %q", cookie)
	}
}

func TestSetCookieLocalHTTPSUsesSameSiteNoneAndSecure(t *testing.T) {
	ctx, rec := newCookieTestContext(t)
	cfg := testConfig("local", time.Hour, time.Hour)
	ctx.Request.TLS = &tls.ConnectionState{}

	setCookie(ctx, cfg, "x", "y", 10, true)

	cookie := cookieHeaderByPrefix(rec.Header().Values("Set-Cookie"), "x=")
	if !strings.Contains(cookie, "Secure") {
		t.Fatalf("expected secure cookie for local https, got %q", cookie)
	}

	if !strings.Contains(cookie, "SameSite=None") {
		t.Fatalf("expected SameSite=None for local https, got %q", cookie)
	}
}

func TestSetCookieLocalHTTPUsesSameSiteLaxWithoutSecure(t *testing.T) {
	ctx, rec := newCookieTestContext(t)
	cfg := testConfig("local", time.Hour, time.Hour)

	setCookie(ctx, cfg, "x", "y", 10, true)

	cookie := cookieHeaderByPrefix(rec.Header().Values("Set-Cookie"), "x=")
	if strings.Contains(cookie, "Secure") {
		t.Fatalf("expected non-secure cookie for local http, got %q", cookie)
	}
	if !strings.Contains(cookie, "SameSite=Lax") {
		t.Fatalf("expected SameSite=Lax for local http, got %q", cookie)
	}
}

func TestSetCookieLocalForwardedHTTPSUsesSameSiteNoneAndSecure(t *testing.T) {
	ctx, rec := newCookieTestContext(t)
	cfg := testConfig("local", time.Hour, time.Hour)
	ctx.Request.Header.Set("X-Forwarded-Proto", "https")

	setCookie(ctx, cfg, "x", "y", 10, true)

	cookie := cookieHeaderByPrefix(rec.Header().Values("Set-Cookie"), "x=")
	if !strings.Contains(cookie, "Secure") {
		t.Fatalf("expected secure cookie for forwarded https, got %q", cookie)
	}
	if !strings.Contains(cookie, "SameSite=None") {
		t.Fatalf("expected SameSite=None for forwarded https, got %q", cookie)
	}
}

func TestRandomTokenHexLength(t *testing.T) {
	tok, err := randomTokenHex(32)
	if err != nil {
		t.Fatalf("randomTokenHex: %v", err)
	}
	if len(tok) != 64 {
		t.Fatalf("expected hex length 64, got %d", len(tok))
	}
}

func TestSetAuthCookiesReturnsErrorWhenRandomFails(t *testing.T) {
	ctx, _ := newCookieTestContext(t)
	cfg := testConfig("local", time.Hour, 2*time.Hour)

	prev := randRead
	randRead = func([]byte) (int, error) { return 0, errors.New("rng down") }
	t.Cleanup(func() { randRead = prev })

	err := setAuthCookies(ctx, cfg, "access", "refresh")
	if !errors.Is(err, errCSRFTokenGeneration) {
		t.Fatalf("expected errCSRFTokenGeneration, got %v", err)
	}
}
