package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"smctf/internal/config"

	"github.com/gin-gonic/gin"
)

const (
	accessTokenCookieName  = "access_token"
	refreshTokenCookieName = "refresh_token"
	csrfTokenCookieName    = "csrf_token"
	csrfTokenHeaderName    = "X-CSRF-Token"
)

var errCSRFTokenGeneration = errors.New("failed to generate csrf token")

var randRead = rand.Read

func setAuthCookies(ctx *gin.Context, cfg config.Config, accessToken, refreshToken string) error {
	setCookie(ctx, cfg, accessTokenCookieName, accessToken, int(cfg.JWT.AccessTTL.Seconds()), true)
	setCookie(ctx, cfg, refreshTokenCookieName, refreshToken, int(cfg.JWT.RefreshTTL.Seconds()), true)

	csrfToken, ok := currentCSRFToken(ctx)
	if !ok {
		var err error
		csrfToken, err = randomTokenHex(32)
		if err != nil {
			return err
		}
	}

	maxAge := max(int(cfg.JWT.RefreshTTL.Seconds()), int(cfg.JWT.AccessTTL.Seconds()))
	setCookie(ctx, cfg, csrfTokenCookieName, csrfToken, maxAge, false)
	ctx.Header(csrfTokenHeaderName, csrfToken)
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Pragma", "no-cache")

	return nil
}

func clearAuthCookies(ctx *gin.Context, cfg config.Config) {
	setCookie(ctx, cfg, accessTokenCookieName, "", -1, true)
	setCookie(ctx, cfg, refreshTokenCookieName, "", -1, true)
	setCookie(ctx, cfg, csrfTokenCookieName, "", -1, false)
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Pragma", "no-cache")
}

func currentRefreshToken(ctx *gin.Context) (string, bool) {
	v, err := ctx.Cookie(refreshTokenCookieName)
	if err != nil || v == "" {
		return "", false
	}

	return v, true
}

func currentCSRFToken(ctx *gin.Context) (string, bool) {
	v, err := ctx.Cookie(csrfTokenCookieName)
	if err != nil || v == "" {
		return "", false
	}

	return v, true
}

func setCookie(ctx *gin.Context, cfg config.Config, name, value string, maxAge int, httpOnly bool) {
	secure := cfg.AppEnv == "production"
	sameSite := http.SameSiteLaxMode
	if cfg.AppEnv == "local" {
		if isHTTPSRequest(ctx.Request) {
			sameSite = http.SameSiteNoneMode
			secure = true
		} else {
			sameSite = http.SameSiteLaxMode
			secure = false
		}
	}

	ctx.SetSameSite(sameSite)
	ctx.SetCookie(name, value, maxAge, "/", cfg.CookieDomain, secure, httpOnly)
}

func isHTTPSRequest(req *http.Request) bool {
	if req == nil {
		return false
	}

	if req.TLS != nil {
		return true
	}

	v := strings.ToLower(strings.TrimSpace(req.Header.Get("X-Forwarded-Proto")))
	if v == "" {
		return false
	}

	for p := range strings.SplitSeq(v, ",") {
		if strings.TrimSpace(p) == "https" {
			return true
		}
	}

	return false
}

func randomTokenHex(bytesLen int) (string, error) {
	b := make([]byte, bytesLen)
	if _, err := randRead(b); err != nil {
		return "", errCSRFTokenGeneration
	}

	return hex.EncodeToString(b), nil
}
