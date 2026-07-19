package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"smctf/internal/auth"
	"smctf/internal/config"
	"smctf/internal/db"
	"smctf/internal/models"
	"smctf/internal/realtime"
	"smctf/internal/repo"
	"smctf/internal/service"
	"smctf/internal/storage"
	"smctf/internal/utils"
	"smctf/internal/vm"

	"github.com/gin-gonic/gin"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

func newJSONContext(t *testing.T, method, path string, body any) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)

	var reader *bytes.Reader

	if body != nil {
		switch v := body.(type) {
		case string:
			reader = bytes.NewReader([]byte(v))
		default:
			data, err := json.Marshal(v)
			if err != nil {
				t.Fatalf("marshal body: %v", err)
			}
			reader = bytes.NewReader(data)
		}
	} else {
		reader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	ctx.Request = req

	return ctx, rec
}

func ptrString(value string) *string {
	return &value
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, dest any) {
	t.Helper()

	if err := json.Unmarshal(rec.Body.Bytes(), dest); err != nil {
		t.Fatalf("decode json: %v", err)
	}
}

func setCachePayload(t *testing.T, env handlerEnv, key string, payload []byte) {
	t.Helper()

	if err := env.redis.Set(context.Background(), key, payload, time.Minute).Err(); err != nil {
		t.Fatalf("set cache: %v", err)
	}
}

func cacheKeyForDivision(env handlerEnv, base string) string {
	return cacheKeyWithDivision(base, &env.defaultDivisionID)
}

func waitForCacheClear(t *testing.T, env handlerEnv, keys ...string) {
	t.Helper()

	deadline := time.Now().Add(200 * time.Millisecond)
	for {
		exists, err := env.redis.Exists(context.Background(), keys...).Result()
		if err != nil {
			t.Fatalf("cache exists: %v", err)
		}

		if exists == 0 {
			return
		}

		if time.Now().After(deadline) {
			t.Fatalf("expected cache cleared: %v", keys)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// Helper Tests

func TestParseIDParam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Params = gin.Params{{Key: "id", Value: "123"}}
	if got, ok := parseIDParam(ctx, "id"); !ok || got != 123 {
		t.Fatalf("expected 123 ok, got %d ok %v", got, ok)
	}

	ctx, _ = gin.CreateTestContext(httptest.NewRecorder())
	ctx.Params = gin.Params{{Key: "id", Value: "0"}}
	if _, ok := parseIDParam(ctx, "id"); ok {
		t.Fatalf("expected invalid id")
	}
}

// App Config Tests

func TestNormalizeETag(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "quoted", in: "\"abc\"", want: "abc"},
		{name: "weak", in: "W/\"abc\"", want: "abc"},
		{name: "spaced", in: "  \"abc\"  ", want: "abc"},
		{name: "unquoted", in: "abc", want: "abc"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeETag(tc.in); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestETagMatches(t *testing.T) {
	cases := []struct {
		name        string
		ifNoneMatch string
		etag        string
		want        bool
	}{
		{name: "exact", ifNoneMatch: "\"abc\"", etag: "\"abc\"", want: true},
		{name: "weak", ifNoneMatch: "W/\"abc\"", etag: "\"abc\"", want: true},
		{name: "multiple", ifNoneMatch: "\"def\", \"abc\"", etag: "\"abc\"", want: true},
		{name: "star", ifNoneMatch: "*", etag: "\"abc\"", want: true},
		{name: "mismatch", ifNoneMatch: "\"def\"", etag: "\"abc\"", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := etagMatches(tc.ifNoneMatch, tc.etag); got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestHandlerGetConfigETag(t *testing.T) {
	env := setupHandlerTest(t)

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/config", nil)
	env.handler.GetConfig(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("config status %d: %s", rec.Code, rec.Body.String())
	}

	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("expected etag header")
	}
}

func TestHandlerAdminConfigUpdate(t *testing.T) {
	env := setupHandlerTest(t)

	body := map[string]string{"title": "My CTF", "description": "Hello"}
	ctx, rec := newJSONContext(t, http.MethodPut, "/api/admin/config", body)
	env.handler.AdminUpdateConfig(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin config status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	decodeJSON(t, rec, &resp)
	if resp.Title != "My CTF" || resp.Description != "Hello" {
		t.Fatalf("unexpected config: %+v", resp)
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/config", nil)
	env.handler.GetConfig(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("config status %d: %s", rec.Code, rec.Body.String())
	}
	decodeJSON(t, rec, &resp)
	if resp.Title != "My CTF" || resp.Description != "Hello" {
		t.Fatalf("unexpected public config: %+v", resp)
	}
}

func TestHandlerAdminConfigValidation(t *testing.T) {
	env := setupHandlerTest(t)

	body := map[string]any{"title": nil}
	ctx, rec := newJSONContext(t, http.MethodPut, "/api/admin/config", body)
	env.handler.AdminUpdateConfig(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandlerAdminConfigBindError(t *testing.T) {
	env := setupHandlerTest(t)

	ctx, rec := newJSONContext(t, http.MethodPut, "/api/admin/config", "{")
	env.handler.AdminUpdateConfig(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandlerAdminConfigFieldMatrix(t *testing.T) {
	env := setupHandlerTest(t)

	cases := []struct {
		name       string
		body       map[string]any
		wantStatus int
	}{
		{"header_title whitespace allowed", map[string]any{"header_title": "   "}, http.StatusOK},
		{"header_description whitespace allowed", map[string]any{"header_description": "   "}, http.StatusOK},
		{"description null rejected", map[string]any{"description": nil}, http.StatusBadRequest},
		{"header_title null rejected", map[string]any{"header_title": nil}, http.StatusBadRequest},
		{"header_description null rejected", map[string]any{"header_description": nil}, http.StatusBadRequest},
		{"ctf_start_at whitespace rejected", map[string]any{"ctf_start_at": "   "}, http.StatusBadRequest},
		{"ctf_end_at whitespace rejected", map[string]any{"ctf_end_at": "   "}, http.StatusBadRequest},
		{"title too long rejected", map[string]any{"title": strings.Repeat("a", 201)}, http.StatusBadRequest},
		{"description too long rejected", map[string]any{"description": strings.Repeat("b", 2001)}, http.StatusBadRequest},
		{"header_title too long rejected", map[string]any{"header_title": strings.Repeat("c", 81)}, http.StatusBadRequest},
		{"header_description too long rejected", map[string]any{"header_description": strings.Repeat("d", 201)}, http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, rec := newJSONContext(t, http.MethodPut, "/api/admin/config", tc.body)
			env.handler.AdminUpdateConfig(ctx)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestHandlerAdminConfigUpdateCTFWindow(t *testing.T) {
	env := setupHandlerTest(t)

	body := map[string]string{
		"ctf_start_at": "2026-02-10T10:00:00Z",
		"ctf_end_at":   "2026-02-10T18:00:00Z",
	}
	ctx, rec := newJSONContext(t, http.MethodPut, "/api/admin/config", body)
	env.handler.AdminUpdateConfig(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin config status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		CTFStartAt string `json:"ctf_start_at"`
		CTFEndAt   string `json:"ctf_end_at"`
	}
	decodeJSON(t, rec, &resp)
	if resp.CTFStartAt != body["ctf_start_at"] || resp.CTFEndAt != body["ctf_end_at"] {
		t.Fatalf("unexpected ctf window: %+v", resp)
	}
}

func TestHandlerAdminConfigInvalidCTFWindow(t *testing.T) {
	env := setupHandlerTest(t)

	body := map[string]string{"ctf_start_at": "nope"}
	ctx, rec := newJSONContext(t, http.MethodPut, "/api/admin/config", body)
	env.handler.AdminUpdateConfig(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandlerAdminConfigCTFWindowClear(t *testing.T) {
	env := setupHandlerTest(t)

	body := map[string]any{
		"ctf_start_at": "2026-02-10T10:00:00Z",
		"ctf_end_at":   "2026-02-10T18:00:00Z",
	}
	ctx, rec := newJSONContext(t, http.MethodPut, "/api/admin/config", body)
	env.handler.AdminUpdateConfig(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin config status %d: %s", rec.Code, rec.Body.String())
	}

	body = map[string]any{
		"ctf_start_at": nil,
		"ctf_end_at":   nil,
	}
	ctx, rec = newJSONContext(t, http.MethodPut, "/api/admin/config", body)
	env.handler.AdminUpdateConfig(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin config status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		CTFStartAt string `json:"ctf_start_at"`
		CTFEndAt   string `json:"ctf_end_at"`
	}
	decodeJSON(t, rec, &resp)
	if resp.CTFStartAt != "" || resp.CTFEndAt != "" {
		t.Fatalf("expected cleared ctf window, got %+v", resp)
	}
}

// Auth Handler Tests

func TestHandlerRegisterLoginRefreshLogout(t *testing.T) {
	env := setupHandlerTest(t)
	admin := createHandlerUser(t, env, "admin@example.com", models.AdminRole, "pass", models.AdminRole)
	key := createHandlerRegistrationKey(t, env, "ABCDEFGHJKLMNPQ2", admin.ID)

	regBody := map[string]string{
		"email":            "user@example.com",
		"username":         "user1",
		"password":         "pass1",
		"registration_key": key.Code,
	}

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/auth/register", regBody)
	env.handler.Register(ctx)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status %d: %s", rec.Code, rec.Body.String())
	}

	loginBody := map[string]string{
		"email":    "user@example.com",
		"password": "wrong",
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/auth/login", loginBody)
	env.handler.Login(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("login invalid status %d: %s", rec.Code, rec.Body.String())
	}

	loginBody["password"] = "pass1"

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/auth/login", loginBody)
	env.handler.Login(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status %d: %s", rec.Code, rec.Body.String())
	}

	var loginResp struct {
		User struct {
			TeamID        int64      `json:"team_id"`
			TeamName      string     `json:"team_name"`
			DivisionID    int64      `json:"division_id"`
			DivisionName  string     `json:"division_name"`
			VMCount       int        `json:"vm_count"`
			VMLimit       int        `json:"vm_limit"`
			BlockedReason *string    `json:"blocked_reason"`
			BlockedAt     *time.Time `json:"blocked_at"`
		} `json:"user"`
	}
	decodeJSON(t, rec, &loginResp)

	if loginResp.User.TeamID != key.TeamID {
		t.Fatalf("expected team_id %d, got %d", key.TeamID, loginResp.User.TeamID)
	}
	if loginResp.User.TeamName != "reg-"+key.Code {
		t.Fatalf("expected team_name %q, got %q", "reg-"+key.Code, loginResp.User.TeamName)
	}
	if loginResp.User.DivisionID == 0 || loginResp.User.DivisionName == "" {
		t.Fatalf("missing division fields in login response")
	}
	if loginResp.User.VMCount != 0 {
		t.Fatalf("expected vm_count 0, got %d", loginResp.User.VMCount)
	}
	if loginResp.User.VMLimit != env.cfg.VM.MaxPer {
		t.Fatalf("expected vm_limit %d, got %d", env.cfg.VM.MaxPer, loginResp.User.VMLimit)
	}
	if loginResp.User.BlockedReason != nil || loginResp.User.BlockedAt != nil {
		t.Fatalf("expected blocked fields to be null")
	}

	refreshToken := ""
	csrfToken := ""
	for _, c := range rec.Header().Values("Set-Cookie") {
		if after, ok := strings.CutPrefix(c, "refresh_token="); ok {
			refreshToken = strings.SplitN(after, ";", 2)[0]
		}
		if after, ok := strings.CutPrefix(c, "csrf_token="); ok {
			csrfToken = strings.SplitN(after, ";", 2)[0]
		}
	}
	if refreshToken == "" || csrfToken == "" {
		t.Fatalf("missing refresh/csrf cookies on login")
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/auth/refresh", nil)
	ctx.Request.AddCookie(&http.Cookie{Name: "refresh_token", Value: "bad"})
	ctx.Request.AddCookie(&http.Cookie{Name: "csrf_token", Value: "csrf"})
	ctx.Request.Header.Set("X-CSRF-Token", "csrf")
	env.handler.Refresh(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("refresh invalid status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/auth/refresh", nil)
	ctx.Request.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})
	ctx.Request.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfToken})
	ctx.Request.Header.Set("X-CSRF-Token", csrfToken)
	env.handler.Refresh(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/auth/logout", nil)
	ctx.Request.AddCookie(&http.Cookie{Name: "refresh_token", Value: "bad"})
	ctx.Request.AddCookie(&http.Cookie{Name: "csrf_token", Value: "csrf"})
	ctx.Request.Header.Set("X-CSRF-Token", "csrf")
	env.handler.Logout(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("logout invalid status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/auth/logout", nil)
	ctx.Request.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})
	ctx.Request.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfToken})
	ctx.Request.Header.Set("X-CSRF-Token", csrfToken)
	env.handler.Logout(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("logout status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerUserVMSummaryWithoutVMService(t *testing.T) {
	handler := New(handlerCfg, nil, nil, nil, nil, nil, nil, nil, nil, handlerRedis, nil)

	count, limit := handler.userVMSummary(context.Background(), 123)
	if count != 0 || limit != 0 {
		t.Fatalf("expected zero summary, got %d/%d", count, limit)
	}
}

func TestHandlerBindErrorDetails(t *testing.T) {
	env := setupHandlerTest(t)
	ctx, rec := newJSONContext(t, http.MethodPost, "/api/auth/register", "{")

	env.handler.Register(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bind invalid json status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/auth/login", map[string]any{"email": 123, "password": true})
	env.handler.Login(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bind type status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerAdminMoveUserTeam(t *testing.T) {
	env := setupHandlerTest(t)
	teamA := createHandlerTeam(t, env, "Alpha")
	teamB := createHandlerTeam(t, env, "Beta")
	user := createHandlerUserWithTeam(t, env, "user@example.com", "user1", "pass", models.UserRole, teamA.ID)

	setCachePayload(t, env, cacheKeyForDivision(env, "leaderboard:users"), []byte(`{"challenges":[],"entries":[]}`))
	setCachePayload(t, env, cacheKeyForDivision(env, "leaderboard:teams"), []byte(`{"challenges":[],"entries":[]}`))
	setCachePayload(t, env, cacheKeyForDivision(env, "timeline:users"), []byte(`{"submissions":[]}`))
	setCachePayload(t, env, cacheKeyForDivision(env, "timeline:teams"), []byte(`{"submissions":[]}`))

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/admin/users/1/team", map[string]any{"team_id": teamB.ID})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.FormatInt(user.ID, 10)}}

	env.handler.AdminMoveUserTeam(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("move team status %d: %s", rec.Code, rec.Body.String())
	}

	var resp adminUserResponse
	decodeJSON(t, rec, &resp)
	if resp.TeamID != teamB.ID {
		t.Fatalf("expected team_id %d, got %d", teamB.ID, resp.TeamID)
	}

	waitForCacheClear(t, env,
		cacheKeyForDivision(env, "leaderboard:users"),
		cacheKeyForDivision(env, "leaderboard:teams"),
		cacheKeyForDivision(env, "timeline:users"),
		cacheKeyForDivision(env, "timeline:teams"),
	)

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/admin/users/1/team", map[string]any{"team_id": -1})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.FormatInt(user.ID, 10)}}
	env.handler.AdminMoveUserTeam(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/admin/users/1/team", map[string]any{"team_id": 9999})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.FormatInt(user.ID, 10)}}
	env.handler.AdminMoveUserTeam(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/admin/users/1/team", map[string]any{"team_id": teamB.ID})
	ctx.Params = gin.Params{{Key: "id", Value: "0"}}
	env.handler.AdminMoveUserTeam(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandlerAdminBlockUser(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "user@example.com", "user1", "pass", models.UserRole)

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/admin/users/1/block", map[string]any{"reason": "policy"})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.FormatInt(user.ID, 10)}}

	env.handler.AdminBlockUser(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("block status %d: %s", rec.Code, rec.Body.String())
	}

	var resp adminUserResponse
	decodeJSON(t, rec, &resp)
	if resp.Role != models.BlockedRole || resp.BlockedReason == nil {
		t.Fatalf("expected blocked user, got %+v", resp)
	}

	admin := createHandlerUser(t, env, "admin@example.com", models.AdminRole, "pass", models.AdminRole)
	ctx, rec = newJSONContext(t, http.MethodPost, "/api/admin/users/1/block", map[string]any{"reason": "policy"})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.FormatInt(admin.ID, 10)}}
	env.handler.AdminBlockUser(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/admin/users/1/block", map[string]any{"reason": " "})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.FormatInt(user.ID, 10)}}
	env.handler.AdminBlockUser(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/admin/users/1/block", map[string]any{"reason": "policy"})
	ctx.Params = gin.Params{{Key: "id", Value: "0"}}
	env.handler.AdminBlockUser(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandlerAdminUnblockUser(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "user@example.com", "user1", "pass", models.UserRole)

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/admin/users/1/block", map[string]any{"reason": "policy"})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.FormatInt(user.ID, 10)}}
	env.handler.AdminBlockUser(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("block status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/admin/users/1/unblock", nil)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.FormatInt(user.ID, 10)}}
	env.handler.AdminUnblockUser(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("unblock status %d: %s", rec.Code, rec.Body.String())
	}

	var resp adminUserResponse
	decodeJSON(t, rec, &resp)
	if resp.Role != models.UserRole || resp.BlockedReason != nil || resp.BlockedAt != nil {
		t.Fatalf("expected unblocked user, got %+v", resp)
	}

	admin := createHandlerUser(t, env, "admin@example.com", models.AdminRole, "pass", models.AdminRole)
	ctx, rec = newJSONContext(t, http.MethodPost, "/api/admin/users/1/unblock", nil)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.FormatInt(admin.ID, 10)}}
	env.handler.AdminUnblockUser(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/admin/users/1/unblock", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "0"}}
	env.handler.AdminUnblockUser(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

// Challenge Handler Tests

func TestHandlerChallengesAndSubmit(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "user@example.com", "user1", "pass", models.UserRole)
	challenge := createHandlerChallenge(t, env, "Challenge", 100, "FLAG{1}", true)
	other := createHandlerChallenge(t, env, "Other", 50, "FLAG{2}", true)

	divisionID := int64(1)
	ctx, rec := newJSONContext(t, http.MethodGet, fmt.Sprintf("/api/challenges?division_id=%d", divisionID), nil)

	env.handler.ListChallenges(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("list challenges status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/challenges/bad/submit", map[string]string{"flag": "FLAG{1}"})
	ctx.Params = gin.Params{{Key: "id", Value: "bad"}}
	ctx.Set("userID", user.ID)

	env.handler.SubmitFlag(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("submit invalid id status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/challenges/1/submit", map[string]string{"flag": "FLAG{1}"})
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", challenge.ID)}}
	ctx.Set("userID", user.ID)

	env.handler.SubmitFlag(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("submit correct status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/challenges/1/submit", map[string]string{"flag": "FLAG{1}"})
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", challenge.ID)}}
	ctx.Set("userID", user.ID)

	env.handler.SubmitFlag(ctx)
	if rec.Code != http.StatusConflict {
		t.Fatalf("submit already status %d: %s", rec.Code, rec.Body.String())
	}

	team := createHandlerTeam(t, env, "Alpha")
	teamUser1 := createHandlerUserWithTeam(t, env, "t1@example.com", "t1", "pass", models.UserRole, team.ID)
	teamUser2 := createHandlerUserWithTeam(t, env, "t2@example.com", "t2", "pass", models.UserRole, team.ID)
	teamChallenge := createHandlerChallenge(t, env, "Team", 120, "FLAG{TEAM}", true)

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/challenges/3/submit", map[string]string{"flag": "FLAG{TEAM}"})
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", teamChallenge.ID)}}
	ctx.Set("userID", teamUser1.ID)

	env.handler.SubmitFlag(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("submit team correct status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/challenges/3/submit", map[string]string{"flag": "FLAG{TEAM}"})
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", teamChallenge.ID)}}
	ctx.Set("userID", teamUser2.ID)

	env.handler.SubmitFlag(ctx)
	if rec.Code != http.StatusConflict {
		t.Fatalf("submit team already status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/challenges/2/submit", map[string]string{"flag": "WRONG"})
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", other.ID)}}
	ctx.Set("userID", user.ID)

	env.handler.SubmitFlag(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("submit wrong status %d: %s", rec.Code, rec.Body.String())
	}

	updateReq := map[string]any{
		"title":       "Updated",
		"description": "New",
		"category":    "Crypto",
		"points":      200,
		"is_active":   false,
	}

	ctx, rec = newJSONContext(t, http.MethodPut, "/api/admin/challenges/1", updateReq)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", challenge.ID)}}

	env.handler.UpdateChallenge(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("update challenge status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPut, "/api/admin/challenges/bad", updateReq)
	ctx.Params = gin.Params{{Key: "id", Value: "bad"}}

	env.handler.UpdateChallenge(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("update challenge invalid id status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPut, "/api/admin/challenges/1", map[string]any{"flag": "new"})
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", challenge.ID)}}

	env.handler.UpdateChallenge(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("update challenge flag status %d: %s", rec.Code, rec.Body.String())
	}

	nullCases := []struct {
		name string
		body map[string]any
	}{
		{"title null", map[string]any{"title": nil}},
		{"description null", map[string]any{"description": nil}},
		{"category null", map[string]any{"category": nil}},
		{"flag null", map[string]any{"flag": nil}},
	}
	for _, tc := range nullCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, rec = newJSONContext(t, http.MethodPut, "/api/admin/challenges/1", tc.body)
			ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", challenge.ID)}}
			env.handler.UpdateChallenge(ctx)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", rec.Code)
			}
		})
	}

	ctx, rec = newJSONContext(t, http.MethodPut, "/api/admin/challenges/1", map[string]any{"title": "   "})
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", challenge.ID)}}
	env.handler.UpdateChallenge(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected whitespace title to be allowed, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodPut, "/api/admin/challenges/1", map[string]any{"vm_enabled": true, "vm_spec": nil})
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", challenge.ID)}}
	env.handler.UpdateChallenge(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for null vm_spec with vm_enabled, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodPut, "/api/admin/challenges/1", "{")
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", challenge.ID)}}
	env.handler.UpdateChallenge(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodPut, "/api/admin/challenges/1", map[string]any{"vm_enabled": false, "vm_spec": "   "})
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", challenge.ID)}}
	env.handler.UpdateChallenge(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for vm_spec when vm disabled, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodPut, "/api/admin/challenges/1", map[string]any{
		"vm_enabled": true,
		"vm_spec":    "",
	})
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", challenge.ID)}}
	env.handler.UpdateChallenge(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for out-of-range vm_spec, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodPut, "/api/admin/challenges/1", map[string]any{
		"points":         10,
		"minimum_points": 20,
	})
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", challenge.ID)}}
	env.handler.UpdateChallenge(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for minimum_points > points, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodDelete, "/api/admin/challenges/1", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", challenge.ID)}}

	env.handler.DeleteChallenge(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete challenge status %d: %s", rec.Code, rec.Body.String())
	}

	_ = challenge
	_ = other
}

func TestHandlerChallengesRequiresDivisionID(t *testing.T) {
	env := setupHandlerTest(t)

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges", nil)
	env.handler.ListChallenges(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandlerListChallengesNotStarted(t *testing.T) {
	env := setupHandlerTest(t)
	start := time.Now().Add(2 * time.Hour)
	setHandlerCTFWindow(t, env, &start, nil)

	ctx, rec := newJSONContext(t, http.MethodGet, fmt.Sprintf("/api/challenges?division_id=%d", env.defaultDivisionID), nil)
	env.handler.ListChallenges(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("list challenges status %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec, &resp)
	if resp["ctf_state"] != string(service.CTFStateNotStarted) {
		t.Fatalf("expected ctf_state not_started, got %v", resp["ctf_state"])
	}

	if _, ok := resp["challenges"]; ok {
		t.Fatalf("expected challenges to be omitted before start")
	}
}

func TestHandlerListChallengesNotStartedAdminBypass(t *testing.T) {
	env := setupHandlerTest(t)
	admin := createHandlerUser(t, env, "admin@example.com", models.AdminRole, "pass", models.AdminRole)
	access, err := auth.GenerateAccessToken(env.cfg.JWT, admin.ID, admin.Role)
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}

	start := time.Now().Add(2 * time.Hour)
	setHandlerCTFWindow(t, env, &start, nil)
	challenge := createHandlerChallenge(t, env, "AdminPreview", 100, "FLAG{A}", true)

	ctx, rec := newJSONContext(t, http.MethodGet, fmt.Sprintf("/api/challenges?division_id=%d", env.defaultDivisionID), nil)
	ctx.Request.AddCookie(&http.Cookie{Name: "access_token", Value: access})
	env.handler.ListChallenges(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("list challenges status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		CTFState   string           `json:"ctf_state"`
		Challenges []map[string]any `json:"challenges"`
	}
	decodeJSON(t, rec, &resp)
	if resp.CTFState != string(service.CTFStateNotStarted) {
		t.Fatalf("expected ctf_state not_started, got %s", resp.CTFState)
	}
	if len(resp.Challenges) != 1 || int64(resp.Challenges[0]["id"].(float64)) != challenge.ID {
		t.Fatalf("expected challenge list for admin before start, got %+v", resp.Challenges)
	}
}

func TestHandlerListChallengesNoPrereqWithAuth(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "nopreq@example.com", "nopreq", "pass", models.UserRole)
	access, err := auth.GenerateAccessToken(env.cfg.JWT, user.ID, user.Role)
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}

	challenge := createHandlerChallenge(t, env, "NoPrereq", 100, "FLAG{N}", true)

	ctx, rec := newJSONContext(t, http.MethodGet, fmt.Sprintf("/api/challenges?division_id=%d", env.defaultDivisionID), nil)
	ctx.Request.AddCookie(&http.Cookie{Name: "access_token", Value: access})
	env.handler.ListChallenges(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec, &resp)
	challenges, ok := resp["challenges"].([]any)
	if !ok || len(challenges) == 0 {
		t.Fatalf("expected challenges in response")
	}

	found := false
	for _, item := range challenges {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}

		if id, ok := row["id"].(float64); ok && int64(id) == challenge.ID {
			found = true
			if row["is_locked"] != false {
				t.Fatalf("expected is_locked false for no prereq")
			}

			if _, ok := row["description"]; !ok {
				t.Fatalf("expected description for unlocked challenge")
			}
		}
	}

	if !found {
		t.Fatalf("expected challenge in list")
	}
}

func TestHandlerListChallengesLocked(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "locked@example.com", "locked", "pass", models.UserRole)
	access, err := auth.GenerateAccessToken(env.cfg.JWT, user.ID, user.Role)
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}
	admin := createHandlerUser(t, env, "admin@example.com", models.AdminRole, "pass", models.AdminRole)
	adminAccess, err := auth.GenerateAccessToken(env.cfg.JWT, admin.ID, admin.Role)
	if err != nil {
		t.Fatalf("generate admin access token: %v", err)
	}

	prev := createHandlerChallenge(t, env, "Prev", 50, "FLAG{P}", true)
	locked := createHandlerChallenge(t, env, "Locked", 100, "FLAG{L}", true)
	locked.PreviousChallengeID = &prev.ID
	if err := env.challengeRepo.Update(context.Background(), locked); err != nil {
		t.Fatalf("update locked challenge: %v", err)
	}

	ctx, rec := newJSONContext(t, http.MethodGet, fmt.Sprintf("/api/challenges?division_id=%d", env.defaultDivisionID), nil)
	env.handler.ListChallenges(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec, &resp)
	challenges, ok := resp["challenges"].([]any)
	if !ok || len(challenges) == 0 {
		t.Fatalf("expected challenges in response")
	}

	for _, item := range challenges {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if id, ok := row["id"].(float64); ok && int64(id) == locked.ID {
			t.Fatalf("expected locked challenge hidden for anonymous user")
		}
	}

	ctx, rec = newJSONContext(t, http.MethodGet, fmt.Sprintf("/api/challenges?division_id=%d", env.defaultDivisionID), nil)
	ctx.Request.AddCookie(&http.Cookie{Name: "access_token", Value: access})
	env.handler.ListChallenges(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("list auth status %d: %s", rec.Code, rec.Body.String())
	}

	resp = map[string]any{}
	decodeJSON(t, rec, &resp)
	challenges, ok = resp["challenges"].([]any)
	if !ok || len(challenges) == 0 {
		t.Fatalf("expected challenges in response")
	}

	for _, item := range challenges {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}

		if id, ok := row["id"].(float64); ok && int64(id) == locked.ID {
			t.Fatalf("expected locked challenge hidden for unsolved user")
		}
	}

	ctx, rec = newJSONContext(t, http.MethodGet, fmt.Sprintf("/api/challenges?division_id=%d", env.defaultDivisionID), nil)
	ctx.Request.AddCookie(&http.Cookie{Name: "access_token", Value: adminAccess})
	env.handler.ListChallenges(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin list status %d: %s", rec.Code, rec.Body.String())
	}

	resp = map[string]any{}
	decodeJSON(t, rec, &resp)
	challenges, ok = resp["challenges"].([]any)
	if !ok || len(challenges) == 0 {
		t.Fatalf("expected admin challenges in response")
	}

	foundLocked := false
	for _, item := range challenges {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if id, ok := row["id"].(float64); ok && int64(id) == locked.ID {
			foundLocked = true
			if row["is_locked"] != true {
				t.Fatalf("expected locked challenge to be marked is_locked for admin")
			}
		}
	}
	if !foundLocked {
		t.Fatalf("expected locked challenge in admin list")
	}

	createHandlerSubmission(t, env, user.ID, prev.ID, true, time.Now().UTC())

	ctx, rec = newJSONContext(t, http.MethodGet, fmt.Sprintf("/api/challenges?division_id=%d", env.defaultDivisionID), nil)
	ctx.Request.AddCookie(&http.Cookie{Name: "access_token", Value: access})
	env.handler.ListChallenges(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("list unlocked status %d: %s", rec.Code, rec.Body.String())
	}

	resp = map[string]any{}
	decodeJSON(t, rec, &resp)
	challenges, ok = resp["challenges"].([]any)
	if !ok || len(challenges) == 0 {
		t.Fatalf("expected challenges in response")
	}

	for _, item := range challenges {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}

		if id, ok := row["id"].(float64); ok && int64(id) == locked.ID {
			if row["is_locked"] != false {
				t.Fatalf("expected locked challenge to be unlocked after solve")
			}

			if _, ok := row["description"]; !ok {
				t.Fatalf("expected description for unlocked challenge")
			}
		}
	}
}

func TestHandlerSubmitFlagEnded(t *testing.T) {
	env := setupHandlerTest(t)
	end := time.Now().Add(-2 * time.Hour)
	setHandlerCTFWindow(t, env, nil, &end)

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/challenges/1/submit", map[string]string{"flag": "FLAG{1}"})
	env.handler.SubmitFlag(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("submit status %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec, &resp)
	if resp["ctf_state"] != string(service.CTFStateEnded) {
		t.Fatalf("expected ctf_state ended, got %v", resp["ctf_state"])
	}
}

func TestHandlerRequestChallengeFileUpload(t *testing.T) {
	env := setupHandlerTest(t)
	challenge := createHandlerChallenge(t, env, "ZipTest", 100, "FLAG{zip}", true)

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/admin/challenges/1/file/upload", map[string]string{"filename": "bundle.zip"})
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", challenge.ID)}}

	env.handler.RequestChallengeFileUpload(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("upload status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerRequestChallengeFileUploadBindError(t *testing.T) {
	env := setupHandlerTest(t)
	challenge := createHandlerChallenge(t, env, "ZipTest", 100, "FLAG{zip}", true)

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/admin/challenges/1/file/upload", "")
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", challenge.ID)}}

	env.handler.RequestChallengeFileUpload(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bind status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerRequestChallengeFileUploadInvalidID(t *testing.T) {
	env := setupHandlerTest(t)
	createHandlerChallenge(t, env, "ZipTest", 100, "FLAG{zip}", true)

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/admin/challenges/bad/file/upload", map[string]string{"filename": "bundle.zip"})
	ctx.Params = gin.Params{{Key: "id", Value: "bad"}}

	env.handler.RequestChallengeFileUpload(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid id status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerRequestChallengeFileUploadStorageUnavailable(t *testing.T) {
	env := setupHandlerTest(t)
	challenge := createHandlerChallenge(t, env, "ZipTest", 100, "FLAG{zip}", true)

	ctfSvc := service.NewCTFService(env.cfg, env.challengeRepo, env.submissionRepo, env.redis, nil)
	scoreRepo := repo.NewScoreboardRepo(env.db)
	scoreSvc := service.NewScoreboardService(scoreRepo)
	handler := New(env.cfg, env.authSvc, ctfSvc, env.appConfigSvc, env.userSvc, scoreSvc, env.divisionSvc, env.teamSvc, nil, env.redis, nil)

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/admin/challenges/1/file/upload", map[string]string{"filename": "bundle.zip"})
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", challenge.ID)}}

	handler.RequestChallengeFileUpload(ctx)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("storage unavailable status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerChallengeCacheInvalidation(t *testing.T) {
	env := setupHandlerTest(t)
	admin := createHandlerUser(t, env, "admin@example.com", models.AdminRole, "pass", models.AdminRole)
	challenge := createHandlerChallenge(t, env, "Ch1", 100, "FLAG{1}", true)

	updateReq := map[string]any{
		"title":       "Updated",
		"description": "New",
		"category":    "Crypto",
		"points":      200,
		"is_active":   false,
	}

	setCachePayload(t, env, cacheKeyForDivision(env, "leaderboard:users"), []byte(`{"challenges":[],"entries":[]}`))
	setCachePayload(t, env, cacheKeyForDivision(env, "leaderboard:teams"), []byte(`{"challenges":[],"entries":[]}`))
	setCachePayload(t, env, cacheKeyForDivision(env, "timeline:users"), []byte(`{"submissions":[]}`))
	setCachePayload(t, env, cacheKeyForDivision(env, "timeline:teams"), []byte(`{"submissions":[]}`))

	ctx, rec := newJSONContext(t, http.MethodPut, "/api/admin/challenges/1", updateReq)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", challenge.ID)}}
	ctx.Set("userID", admin.ID)

	env.handler.UpdateChallenge(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("update challenge status %d: %s", rec.Code, rec.Body.String())
	}

	waitForCacheClear(t, env,
		cacheKeyForDivision(env, "leaderboard:users"),
		cacheKeyForDivision(env, "leaderboard:teams"),
		cacheKeyForDivision(env, "timeline:users"),
		cacheKeyForDivision(env, "timeline:teams"),
	)

	setCachePayload(t, env, cacheKeyForDivision(env, "leaderboard:users"), []byte(`{"challenges":[],"entries":[]}`))
	setCachePayload(t, env, cacheKeyForDivision(env, "leaderboard:teams"), []byte(`{"challenges":[],"entries":[]}`))
	setCachePayload(t, env, cacheKeyForDivision(env, "timeline:users"), []byte(`{"submissions":[]}`))
	setCachePayload(t, env, cacheKeyForDivision(env, "timeline:teams"), []byte(`{"submissions":[]}`))

	ctx, rec = newJSONContext(t, http.MethodDelete, "/api/admin/challenges/1", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", challenge.ID)}}
	ctx.Set("userID", admin.ID)

	env.handler.DeleteChallenge(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete challenge status %d: %s", rec.Code, rec.Body.String())
	}

	waitForCacheClear(t, env,
		cacheKeyForDivision(env, "leaderboard:users"),
		cacheKeyForDivision(env, "leaderboard:teams"),
		cacheKeyForDivision(env, "timeline:users"),
		cacheKeyForDivision(env, "timeline:teams"),
	)
}

func TestHandlerCreateChallengeAndBindErrors(t *testing.T) {
	env := setupHandlerTest(t)

	admin := createHandlerUser(t, env, "admin@example.com", models.AdminRole, "pass", models.AdminRole)

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/admin/challenges", "")
	env.handler.CreateChallenge(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("create challenge bind status %d: %s", rec.Code, rec.Body.String())
	}

	body := map[string]any{
		"title":       "New Challenge",
		"description": "desc",
		"category":    "Misc",
		"points":      100,
		"flag":        "FLAG{X}",
		"is_active":   true,
	}
	ctx, rec = newJSONContext(t, http.MethodPost, "/api/admin/challenges", body)
	ctx.Set("userID", admin.ID)

	setCachePayload(t, env, cacheKeyForDivision(env, "leaderboard:users"), []byte(`{"challenges":[],"entries":[]}`))
	setCachePayload(t, env, cacheKeyForDivision(env, "leaderboard:teams"), []byte(`{"challenges":[],"entries":[]}`))
	setCachePayload(t, env, cacheKeyForDivision(env, "timeline:users"), []byte(`{"submissions":[]}`))
	setCachePayload(t, env, cacheKeyForDivision(env, "timeline:teams"), []byte(`{"submissions":[]}`))

	env.handler.CreateChallenge(ctx)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create challenge status %d: %s", rec.Code, rec.Body.String())
	}

	waitForCacheClear(t, env,
		cacheKeyForDivision(env, "leaderboard:users"),
		cacheKeyForDivision(env, "leaderboard:teams"),
		cacheKeyForDivision(env, "timeline:users"),
		cacheKeyForDivision(env, "timeline:teams"),
	)
}

func createHandlerVMChallenge(t *testing.T, env handlerEnv, title string) *models.Challenge {
	t.Helper()
	sandboxSpec := "apiVersion: v1\nkind: Sandbox\nmetadata:\n  name: handler\nspec:\n  containers:\n    - name: app\n      image: nginx\n      ports:\n        - containerPort: 80\n"
	challenge := &models.Challenge{
		Title:         title,
		Description:   "desc",
		Category:      "Web",
		Points:        100,
		MinimumPoints: 100,
		VMEnabled:     true,
		VMSpec:        &sandboxSpec,
		IsActive:      true,
		CreatedAt:     time.Now().UTC(),
	}
	hash, err := utils.HashFlag("flag", bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash flag: %v", err)
	}
	challenge.FlagHash = hash

	if err := env.challengeRepo.Create(context.Background(), challenge); err != nil {
		t.Fatalf("create challenge: %v", err)
	}

	return challenge
}

func setupHandlerVMService(t *testing.T, env handlerEnv, client vm.API) (*service.VMService, *repo.VMRepo) {
	t.Helper()
	vmRepo := repo.NewVMRepo(env.db)
	vmCfg := config.VMConfig{
		Enabled:      true,
		MaxPer:       3,
		CreateWindow: time.Minute,
		CreateMax:    5,
	}

	vmSvc := service.NewVMService(vmCfg, vmRepo, env.challengeRepo, env.submissionRepo, client, env.redis)
	return vmSvc, vmRepo
}

func setupHandlerVMServiceWithScope(t *testing.T, env handlerEnv, client vm.API, scope string) (*service.VMService, *repo.VMRepo) {
	t.Helper()
	vmRepo := repo.NewVMRepo(env.db)
	vmCfg := config.VMConfig{
		Enabled:      true,
		MaxScope:     scope,
		MaxPer:       3,
		CreateWindow: time.Minute,
		CreateMax:    5,
	}

	vmSvc := service.NewVMService(vmCfg, vmRepo, env.challengeRepo, env.submissionRepo, client, env.redis)
	return vmSvc, vmRepo
}

func TestVMHandlersCRUD(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "u1@example.com", "u1", "pass", models.UserRole)
	challenge := createHandlerVMChallenge(t, env, "vm")

	var deleteCalls atomic.Int32
	mock := &vm.MockClient{
		CreateSandboxFn: func(ctx context.Context, id string, specYAML string) (*vm.Sandbox, error) {
			return &vm.Sandbox{
				ID: id,
				Status: vm.SandboxStatus{
					Phase:         "Running",
					ExternalIP:    "127.0.0.1",
					AssignedPorts: []vm.PortMapping{{ContainerPort: 80, Protocol: "TCP", HostPort: 31001}},
				},
			}, nil
		},
		GetSandboxFn: func(ctx context.Context, vmID string) (*vm.Sandbox, error) {
			return &vm.Sandbox{
				ID: vmID,
				Status: vm.SandboxStatus{
					Phase:         "Running",
					ExternalIP:    "127.0.0.1",
					AssignedPorts: []vm.PortMapping{{ContainerPort: 80, Protocol: "TCP", HostPort: 31001}},
				},
			}, nil
		},
		DeleteSandboxFn: func(ctx context.Context, vmID string) error {
			deleteCalls.Add(1)
			return nil
		},
	}

	vmSvc, _ := setupHandlerVMService(t, env, mock)
	env.handler.vms = vmSvc

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/challenges/"+fmt.Sprint(challenge.ID)+"/vm", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(challenge.ID)}}
	ctx.Set("userID", user.ID)

	env.handler.CreateVM(ctx)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	var created vmResponse
	decodeJSON(t, rec, &created)
	if created.VMID == "" || len(created.Ports) != 1 || created.Ports[0].ContainerPort != 80 {
		t.Fatalf("unexpected response: %+v", created)
	}

	if created.CreatedByUsername == "" || created.ChallengeTitle == "" {
		t.Fatalf("expected created_by and challenge_title, got %+v", created)
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/challenges/"+fmt.Sprint(challenge.ID)+"/vm", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(challenge.ID)}}
	ctx.Set("userID", user.ID)

	env.handler.GetVM(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodDelete, "/api/challenges/"+fmt.Sprint(challenge.ID)+"/vm", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(challenge.ID)}}
	ctx.Set("userID", user.ID)

	env.handler.DeleteVM(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if deleteCalls.Load() != 1 {
		t.Fatalf("expected delete call, got %d", deleteCalls.Load())
	}
}

func TestVMHandlersList(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "u2@example.com", "u2", "pass", models.UserRole)
	challenge1 := createHandlerVMChallenge(t, env, "vm-1")
	challenge2 := createHandlerVMChallenge(t, env, "vm-2")

	mock := &vm.MockClient{
		GetSandboxFn: func(ctx context.Context, vmID string) (*vm.Sandbox, error) {
			return &vm.Sandbox{ID: vmID, Status: vm.SandboxStatus{Phase: "running", AssignedPorts: []vm.PortMapping{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}}}}, nil
		},
	}

	vmSvc, vmRepo := setupHandlerVMService(t, env, mock)
	env.handler.vms = vmSvc

	vm1 := &models.VM{UserID: user.ID, ChallengeID: challenge1.ID, VMID: "vm-1", Status: "running", Ports: vm.PortMappings{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	vm2 := &models.VM{UserID: user.ID, ChallengeID: challenge2.ID, VMID: "vm-2", Status: "running", Ports: vm.PortMappings{{ContainerPort: 80, Protocol: "TCP", NodePort: 31002}}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := vmRepo.Create(context.Background(), vm1); err != nil {
		t.Fatalf("create vm1: %v", err)
	}

	if err := vmRepo.Create(context.Background(), vm2); err != nil {
		t.Fatalf("create vm2: %v", err)
	}

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/vms", nil)
	ctx.Set("userID", user.ID)
	env.handler.ListVMs(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp vmsListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.CTFState != string(service.CTFStateActive) {
		t.Fatalf("expected ctf_state active, got %s", resp.CTFState)
	}

	if len(resp.VMs) != 2 {
		t.Fatalf("expected 2 vms, got %d", len(resp.VMs))
	}
}

func TestVMHandlersListTeamScope(t *testing.T) {
	env := setupHandlerTest(t)
	team := createHandlerTeam(t, env, "TeamList")
	user := createHandlerUserWithTeam(t, env, "t1@example.com", "t1", "pass", models.UserRole, team.ID)
	user2 := createHandlerUserWithTeam(t, env, "t2@example.com", "t2", "pass", models.UserRole, team.ID)
	challenge1 := createHandlerVMChallenge(t, env, "team-vm-1")
	challenge2 := createHandlerVMChallenge(t, env, "team-vm-2")

	mock := &vm.MockClient{
		GetSandboxFn: func(ctx context.Context, vmID string) (*vm.Sandbox, error) {
			return &vm.Sandbox{ID: vmID, Status: vm.SandboxStatus{Phase: "running", AssignedPorts: []vm.PortMapping{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}}}}, nil
		},
	}

	vmSvc, vmRepo := setupHandlerVMServiceWithScope(t, env, mock, "team")
	env.handler.vms = vmSvc

	now := time.Now().UTC()
	vm1 := &models.VM{UserID: user.ID, ChallengeID: challenge1.ID, VMID: "team-vm-1", Status: "running", Ports: vm.PortMappings{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}}, CreatedAt: now, UpdatedAt: now}
	vm2 := &models.VM{UserID: user2.ID, ChallengeID: challenge2.ID, VMID: "team-vm-2", Status: "running", Ports: vm.PortMappings{{ContainerPort: 80, Protocol: "TCP", NodePort: 31002}}, CreatedAt: now, UpdatedAt: now}
	if err := vmRepo.Create(context.Background(), vm1); err != nil {
		t.Fatalf("create vm1: %v", err)
	}

	if err := vmRepo.Create(context.Background(), vm2); err != nil {
		t.Fatalf("create vm2: %v", err)
	}

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/vms", nil)
	ctx.Set("userID", user.ID)
	env.handler.ListVMs(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp vmsListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.VMs) != 2 {
		t.Fatalf("expected 2 team vms, got %d", len(resp.VMs))
	}

	if resp.VMs[0].CreatedByUserID == 0 || resp.VMs[0].CreatedByUsername == "" {
		t.Fatalf("expected created_by fields set, got %+v", resp.VMs[0])
	}

	if resp.VMs[0].ChallengeTitle == "" {
		t.Fatalf("expected challenge_title set, got %+v", resp.VMs[0])
	}
}

func TestVMHandlersGetTeamScope(t *testing.T) {
	env := setupHandlerTest(t)
	team := createHandlerTeam(t, env, "TeamGet")
	user := createHandlerUserWithTeam(t, env, "g1@example.com", "g1", "pass", models.UserRole, team.ID)
	user2 := createHandlerUserWithTeam(t, env, "g2@example.com", "g2", "pass", models.UserRole, team.ID)
	challenge := createHandlerVMChallenge(t, env, "team-get")

	mock := &vm.MockClient{
		GetSandboxFn: func(ctx context.Context, vmID string) (*vm.Sandbox, error) {
			return &vm.Sandbox{ID: vmID, Status: vm.SandboxStatus{Phase: "running", AssignedPorts: []vm.PortMapping{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}}}}, nil
		},
	}

	vmSvc, vmRepo := setupHandlerVMServiceWithScope(t, env, mock, "team")
	env.handler.vms = vmSvc

	now := time.Now().UTC()
	vmModel := &models.VM{UserID: user2.ID, ChallengeID: challenge.ID, VMID: "team-get", Status: "running", Ports: vm.PortMappings{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}}, CreatedAt: now, UpdatedAt: now}
	if err := vmRepo.Create(context.Background(), vmModel); err != nil {
		t.Fatalf("create vm: %v", err)
	}

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges/"+fmt.Sprint(challenge.ID)+"/vm", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(challenge.ID)}}
	ctx.Set("userID", user.ID)
	env.handler.GetVM(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("get vm status %d: %s", rec.Code, rec.Body.String())
	}

	var resp vmResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.VMID != "team-get" {
		t.Fatalf("expected team vm, got %+v", resp)
	}

	if resp.CreatedByUserID != user2.ID || resp.CreatedByUsername == "" {
		t.Fatalf("expected created_by fields, got %+v", resp)
	}

	if resp.ChallengeTitle != challenge.Title {
		t.Fatalf("expected challenge_title %q, got %q", challenge.Title, resp.ChallengeTitle)
	}
}

func TestAdminVMHandlersList(t *testing.T) {
	env := setupHandlerTest(t)
	team := createHandlerTeam(t, env, "Alpha")
	user := createHandlerUserWithTeam(t, env, "admin@example.com", "uadmin", "pass", models.UserRole, team.ID)
	challenge := createHandlerVMChallenge(t, env, "admin-vm")

	mock := &vm.MockClient{
		GetSandboxFn: func(ctx context.Context, vmID string) (*vm.Sandbox, error) {
			return &vm.Sandbox{ID: vmID, Status: vm.SandboxStatus{Phase: "running", AssignedPorts: []vm.PortMapping{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}}}}, nil
		},
	}

	vmSvc, vmRepo := setupHandlerVMService(t, env, mock)
	env.handler.vms = vmSvc

	vmModel := &models.VM{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		VMID:        "vm-admin-1",
		Status:      "running",
		Ports:       vm.PortMappings{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := vmRepo.Create(context.Background(), vmModel); err != nil {
		t.Fatalf("create vm: %v", err)
	}

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/admin/vms", nil)
	env.handler.AdminListVMs(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp adminVMsListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.VMs) != 1 {
		t.Fatalf("expected 1 vm, got %d", len(resp.VMs))
	}

	item := resp.VMs[0]
	if item.VMID != "vm-admin-1" || item.Username != user.Username || item.TeamName != team.Name || item.ChallengeTitle != challenge.Title {
		t.Fatalf("unexpected admin vm response: %+v", item)
	}
}

func TestAdminVMHandlersListDisabled(t *testing.T) {
	env := setupHandlerTest(t)
	env.handler.vms = nil

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/admin/vms", nil)
	env.handler.AdminListVMs(ctx)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestAdminVMHandlersDelete(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "admin@example.com", "uadmin-del", "pass", models.UserRole)
	challenge := createHandlerVMChallenge(t, env, "admin-del")

	var deleteCalls atomic.Int32
	mock := &vm.MockClient{
		DeleteSandboxFn: func(ctx context.Context, vmID string) error {
			deleteCalls.Add(1)
			return nil
		},
	}

	vmSvc, vmRepo := setupHandlerVMService(t, env, mock)
	env.handler.vms = vmSvc

	vmModel := &models.VM{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		VMID:        "vm-admin-del",
		Status:      "running",
		Ports:       vm.PortMappings{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := vmRepo.Create(context.Background(), vmModel); err != nil {
		t.Fatalf("create vm: %v", err)
	}

	ctx, rec := newJSONContext(t, http.MethodDelete, "/api/admin/vms/vm-admin-del", nil)
	ctx.Params = gin.Params{{Key: "vm_id", Value: "vm-admin-del"}}
	env.handler.AdminDeleteVM(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if deleteCalls.Load() != 1 {
		t.Fatalf("expected delete call, got %d", deleteCalls.Load())
	}

	if _, err := vmRepo.GetByVMID(context.Background(), "vm-admin-del"); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("expected vm deleted, got %v", err)
	}
}

func TestAdminVMHandlersDeleteMissingVMID(t *testing.T) {
	env := setupHandlerTest(t)
	mock := &vm.MockClient{}
	vmSvc, _ := setupHandlerVMService(t, env, mock)
	env.handler.vms = vmSvc

	ctx, rec := newJSONContext(t, http.MethodDelete, "/api/admin/vms/", nil)
	env.handler.AdminDeleteVM(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	var resp errorResponse
	decodeJSON(t, rec, &resp)
	if resp.Error != service.ErrInvalidInput.Error() {
		t.Fatalf("expected invalid input, got %s", resp.Error)
	}
}

func TestAdminVMHandlersGet(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "u-admin-get@example.com", "uadmin-get", "pass", models.UserRole)
	challenge := createHandlerVMChallenge(t, env, "admin-get")

	mock := &vm.MockClient{
		GetSandboxFn: func(ctx context.Context, vmID string) (*vm.Sandbox, error) {
			return &vm.Sandbox{ID: vmID, Status: vm.SandboxStatus{Phase: "running", AssignedPorts: []vm.PortMapping{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}}}}, nil
		},
	}

	vmSvc, vmRepo := setupHandlerVMService(t, env, mock)
	env.handler.vms = vmSvc

	vmModel := &models.VM{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		VMID:        "vm-admin-get",
		Status:      "running",
		Ports:       vm.PortMappings{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := vmRepo.Create(context.Background(), vmModel); err != nil {
		t.Fatalf("create vm: %v", err)
	}

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/admin/vms/vm-admin-get", nil)
	ctx.Params = gin.Params{{Key: "vm_id", Value: "vm-admin-get"}}
	env.handler.AdminGetVM(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp vmResponse
	decodeJSON(t, rec, &resp)
	if resp.VMID != "vm-admin-get" || resp.ChallengeID != challenge.ID {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestAdminVMHandlersGetMissingVMID(t *testing.T) {
	env := setupHandlerTest(t)
	mock := &vm.MockClient{}
	vmSvc, _ := setupHandlerVMService(t, env, mock)
	env.handler.vms = vmSvc

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/admin/vms/", nil)
	env.handler.AdminGetVM(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	var resp errorResponse
	decodeJSON(t, rec, &resp)
	if resp.Error != service.ErrInvalidInput.Error() {
		t.Fatalf("expected invalid input, got %s", resp.Error)
	}
}

func TestAdminVMHandlersGetNotFound(t *testing.T) {
	env := setupHandlerTest(t)
	mock := &vm.MockClient{}
	vmSvc, _ := setupHandlerVMService(t, env, mock)
	env.handler.vms = vmSvc

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/admin/vms/missing", nil)
	ctx.Params = gin.Params{{Key: "vm_id", Value: "missing"}}
	env.handler.AdminGetVM(ctx)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestAdminReport(t *testing.T) {
	env := setupHandlerTest(t)

	user := createHandlerUser(t, env, "report@example.com", "reporter", "pass", models.UserRole)
	challenge := createHandlerVMChallenge(t, env, "report-challenge")

	vmRepo := repo.NewVMRepo(env.db)
	vmModel := &models.VM{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		VMID:        "vm-report",
		Status:      "running",
		Ports:       vm.PortMappings{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := vmRepo.Create(context.Background(), vmModel); err != nil {
		t.Fatalf("create vm: %v", err)
	}

	mock := &vm.MockClient{}
	vmSvc := service.NewVMService(config.VMConfig{Enabled: true, MaxPer: 3, CreateWindow: time.Minute, CreateMax: 5}, vmRepo, env.challengeRepo, env.submissionRepo, mock, env.redis)
	env.handler.vms = vmSvc

	createHandlerSubmission(t, env, user.ID, challenge.ID, true, time.Now().UTC())

	if _, err := env.appConfigRepo.Upsert(context.Background(), "title", "Report CTF"); err != nil {
		t.Fatalf("upsert app config: %v", err)
	}

	key := createHandlerRegistrationKey(t, env, "ABCDEFGHJKLMNPQ2", user.ID)
	use := &models.RegistrationKeyUse{
		RegistrationKeyID: key.ID,
		UsedBy:            user.ID,
		UsedByIP:          "127.0.0.1",
		UsedAt:            time.Now().UTC(),
	}
	if _, err := env.db.NewInsert().Model(use).Exec(context.Background()); err != nil {
		t.Fatalf("create registration key use: %v", err)
	}

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/admin/report", nil)
	env.handler.AdminReport(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]any
	decodeJSON(t, rec, &resp)

	challenges, ok := resp["challenges"].([]any)
	if !ok || len(challenges) == 0 {
		t.Fatalf("expected challenges in report")
	}

	challengeMap, ok := challenges[0].(map[string]any)
	if !ok {
		t.Fatalf("expected challenge object")
	}

	if _, exists := challengeMap["flag_hash"]; exists {
		t.Fatalf("expected flag_hash to be omitted")
	}

	users, ok := resp["users"].([]any)
	if !ok || len(users) == 0 {
		t.Fatalf("expected users in report")
	}

	userMap, ok := users[0].(map[string]any)
	if !ok {
		t.Fatalf("expected user object")
	}

	if _, exists := userMap["password_hash"]; exists {
		t.Fatalf("expected password_hash to be omitted")
	}

	submissions, ok := resp["submissions"].([]any)
	if !ok || len(submissions) == 0 {
		t.Fatalf("expected submissions in report")
	}

	submissionMap, ok := submissions[0].(map[string]any)
	if !ok {
		t.Fatalf("expected submission object")
	}

	if _, exists := submissionMap["provided"]; exists {
		t.Fatalf("expected provided to be omitted")
	}
}

func TestVMHandlersNotStarted(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "u3@example.com", "u3", "pass", models.UserRole)
	challenge := createHandlerVMChallenge(t, env, "vm")

	start := time.Now().Add(2 * time.Hour)
	setHandlerCTFWindow(t, env, &start, nil)

	mock := &vm.MockClient{
		GetSandboxFn: func(ctx context.Context, vmID string) (*vm.Sandbox, error) {
			return &vm.Sandbox{ID: vmID, Status: vm.SandboxStatus{Phase: "running", AssignedPorts: []vm.PortMapping{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}}}}, nil
		},
	}

	vmSvc, _ := setupHandlerVMService(t, env, mock)
	env.handler.vms = vmSvc

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/challenges/"+fmt.Sprint(challenge.ID)+"/vm", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(challenge.ID)}}
	ctx.Set("userID", user.ID)
	env.handler.CreateVM(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("create vm status %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec, &resp)
	if resp["ctf_state"] != string(service.CTFStateNotStarted) {
		t.Fatalf("expected ctf_state not_started, got %v", resp["ctf_state"])
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/vms", nil)
	ctx.Set("userID", user.ID)
	env.handler.ListVMs(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("list vms status %d: %s", rec.Code, rec.Body.String())
	}

	resp = map[string]any{}
	decodeJSON(t, rec, &resp)
	if resp["ctf_state"] != string(service.CTFStateNotStarted) {
		t.Fatalf("expected ctf_state not_started, got %v", resp["ctf_state"])
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/challenges/"+fmt.Sprint(challenge.ID)+"/vm", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(challenge.ID)}}
	ctx.Set("userID", user.ID)
	env.handler.GetVM(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("get vm status %d: %s", rec.Code, rec.Body.String())
	}

	resp = map[string]any{}
	decodeJSON(t, rec, &resp)
	if resp["ctf_state"] != string(service.CTFStateNotStarted) {
		t.Fatalf("expected ctf_state not_started, got %v", resp["ctf_state"])
	}

	ctx, rec = newJSONContext(t, http.MethodDelete, "/api/challenges/"+fmt.Sprint(challenge.ID)+"/vm", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(challenge.ID)}}
	ctx.Set("userID", user.ID)
	env.handler.DeleteVM(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete vm status %d: %s", rec.Code, rec.Body.String())
	}

	resp = map[string]any{}
	decodeJSON(t, rec, &resp)
	if resp["ctf_state"] != string(service.CTFStateNotStarted) {
		t.Fatalf("expected ctf_state not_started, got %v", resp["ctf_state"])
	}
}

func TestVMHandlersNotStartedAdminBypassAndEndedBlocked(t *testing.T) {
	env := setupHandlerTest(t)
	admin := createHandlerUser(t, env, "admin@example.com", models.AdminRole, "pass", models.AdminRole)
	access, err := auth.GenerateAccessToken(env.cfg.JWT, admin.ID, admin.Role)
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}
	challenge := createHandlerVMChallenge(t, env, "vm-admin")

	start := time.Now().Add(2 * time.Hour)
	setHandlerCTFWindow(t, env, &start, nil)

	mock := &vm.MockClient{
		CreateSandboxFn: func(ctx context.Context, vmID string, specYAML string) (*vm.Sandbox, error) {
			return &vm.Sandbox{ID: vmID, Status: vm.SandboxStatus{Phase: "pending"}}, nil
		},
		GetSandboxFn: func(ctx context.Context, vmID string) (*vm.Sandbox, error) {
			return &vm.Sandbox{ID: vmID, Status: vm.SandboxStatus{Phase: "running", AssignedPorts: []vm.PortMapping{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}}}}, nil
		},
		DeleteSandboxFn: func(ctx context.Context, vmID string) error { return nil },
	}
	vmSvc, _ := setupHandlerVMService(t, env, mock)
	env.handler.vms = vmSvc

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/challenges/"+fmt.Sprint(challenge.ID)+"/vm", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(challenge.ID)}}
	ctx.Set("userID", admin.ID)
	ctx.Request.AddCookie(&http.Cookie{Name: "access_token", Value: access})
	env.handler.CreateVM(ctx)
	if rec.Code != http.StatusCreated {
		t.Fatalf("admin create vm before start status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/vms", nil)
	ctx.Set("userID", admin.ID)
	ctx.Request.AddCookie(&http.Cookie{Name: "access_token", Value: access})
	env.handler.ListVMs(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin list vms before start status %d: %s", rec.Code, rec.Body.String())
	}

	var listResp map[string]any
	decodeJSON(t, rec, &listResp)
	if _, ok := listResp["vms"]; !ok {
		t.Fatalf("expected vm list for admin before start")
	}

	end := time.Now().Add(-2 * time.Hour)
	setHandlerCTFWindow(t, env, nil, &end)

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/challenges/"+fmt.Sprint(challenge.ID)+"/vm", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(challenge.ID)}}
	ctx.Set("userID", admin.ID)
	ctx.Request.AddCookie(&http.Cookie{Name: "access_token", Value: access})
	env.handler.CreateVM(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin create vm after end status %d: %s", rec.Code, rec.Body.String())
	}

	var endedResp map[string]any
	decodeJSON(t, rec, &endedResp)
	if endedResp["ctf_state"] != string(service.CTFStateEnded) {
		t.Fatalf("expected ctf_state ended, got %v", endedResp["ctf_state"])
	}
}

func TestAdminGetChallengeIncludesVMSpec(t *testing.T) {
	env := setupHandlerTest(t)
	challenge := createHandlerVMChallenge(t, env, "vm")

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/admin/challenges/"+fmt.Sprint(challenge.ID), nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(challenge.ID)}}
	env.handler.AdminGetChallenge(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp["vm_spec"] == nil {
		t.Fatalf("expected vm_spec in response")
	}
}

func TestAdminGetChallengeInvalidID(t *testing.T) {
	env := setupHandlerTest(t)

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/admin/challenges/bad", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "bad"}}
	env.handler.AdminGetChallenge(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestSubmitFlagDeletesVM(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "u3@example.com", "u3", "pass", models.UserRole)
	challenge := createHandlerVMChallenge(t, env, "vm")

	vmRepo := repo.NewVMRepo(env.db)
	vmModel := &models.VM{UserID: user.ID, ChallengeID: challenge.ID, VMID: "vm-sub", Status: "running", Ports: vm.PortMappings{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := vmRepo.Create(context.Background(), vmModel); err != nil {
		t.Fatalf("create vm: %v", err)
	}

	deleted := false
	mock := &vm.MockClient{
		DeleteSandboxFn: func(ctx context.Context, vmID string) error {
			if vmID == "vm-sub" {
				deleted = true
			}
			return nil
		},
	}
	vmSvc := service.NewVMService(config.VMConfig{Enabled: true, MaxPer: 3, CreateWindow: time.Minute, CreateMax: 5}, vmRepo, env.challengeRepo, env.submissionRepo, mock, env.redis)
	env.handler.vms = vmSvc

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/challenges/"+fmt.Sprint(challenge.ID)+"/submit", submitRequest{Flag: "flag"})
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(challenge.ID)}}
	ctx.Set("userID", user.ID)
	env.handler.SubmitFlag(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if !deleted {
		t.Fatalf("expected vm delete call")
	}
}

func TestHandlerDownloadNotStarted(t *testing.T) {
	env := setupHandlerTest(t)
	challenge := createHandlerChallenge(t, env, "Download", 100, "FLAG{D}", true)
	start := time.Now().Add(2 * time.Hour)
	setHandlerCTFWindow(t, env, &start, nil)

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/challenges/"+fmt.Sprint(challenge.ID)+"/file/download", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(challenge.ID)}}
	env.handler.RequestChallengeFileDownload(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("download status %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec, &resp)
	if resp["ctf_state"] != string(service.CTFStateNotStarted) {
		t.Fatalf("expected ctf_state not_started, got %v", resp["ctf_state"])
	}
}

func TestHandlerDownloadNotStartedAdminBypass(t *testing.T) {
	env := setupHandlerTest(t)
	admin := createHandlerUser(t, env, "admin@example.com", models.AdminRole, "pass", models.AdminRole)
	access, err := auth.GenerateAccessToken(env.cfg.JWT, admin.ID, admin.Role)
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}
	challenge := createHandlerChallenge(t, env, "Download", 100, "FLAG{D}", true)
	fileKey := "challenge-files/admin-before-start.zip"
	fileName := "admin-before-start.zip"
	fileUploadedAt := time.Now().UTC()
	challenge.FileKey = &fileKey
	challenge.FileName = &fileName
	challenge.FileUploadedAt = &fileUploadedAt
	if err := env.challengeRepo.Update(context.Background(), challenge); err != nil {
		t.Fatalf("update challenge file metadata: %v", err)
	}
	start := time.Now().Add(2 * time.Hour)
	setHandlerCTFWindow(t, env, &start, nil)

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/challenges/"+fmt.Sprint(challenge.ID)+"/file/download", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(challenge.ID)}}
	ctx.Set("userID", admin.ID)
	ctx.Request.AddCookie(&http.Cookie{Name: "access_token", Value: access})
	env.handler.RequestChallengeFileDownload(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin download before start status %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec, &resp)
	if _, ok := resp["url"]; !ok {
		t.Fatalf("expected presigned download response for admin before start")
	}
}

func TestHandlerDownloadInactiveAccessControl(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "inactive-user@example.com", "inactive-user", "pass", models.UserRole)
	admin := createHandlerUser(t, env, "inactive-admin@example.com", "inactive-admin", "pass", models.AdminRole)
	adminAccess, err := auth.GenerateAccessToken(env.cfg.JWT, admin.ID, admin.Role)
	if err != nil {
		t.Fatalf("generate admin access token: %v", err)
	}

	challenge := createHandlerChallenge(t, env, "InactiveDownload", 100, "FLAG{INACTIVE-DOWNLOAD}", true)
	challenge.IsActive = false
	fileKey := "challenge-files/inactive.zip"
	fileName := "inactive.zip"
	fileUploadedAt := time.Now().UTC()
	challenge.FileKey = &fileKey
	challenge.FileName = &fileName
	challenge.FileUploadedAt = &fileUploadedAt
	if err := env.challengeRepo.Update(context.Background(), challenge); err != nil {
		t.Fatalf("update inactive challenge: %v", err)
	}

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/challenges/"+fmt.Sprint(challenge.ID)+"/file/download", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(challenge.ID)}}
	ctx.Set("userID", user.ID)
	env.handler.RequestChallengeFileDownload(ctx)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for user inactive download, got %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/challenges/"+fmt.Sprint(challenge.ID)+"/file/download", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(challenge.ID)}}
	ctx.Set("userID", admin.ID)
	ctx.Request.AddCookie(&http.Cookie{Name: "access_token", Value: adminAccess})
	env.handler.RequestChallengeFileDownload(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected admin inactive download success, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec, &resp)
	if _, ok := resp["url"]; !ok {
		t.Fatalf("expected presigned download url for admin")
	}
}

func TestVMHandlersInactiveAccessControl(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "inactive-vm-user@example.com", "inactive-vm-user", "pass", models.UserRole)
	admin := createHandlerUser(t, env, "inactive-vm-admin@example.com", "inactive-vm-admin", "pass", models.AdminRole)
	adminAccess, err := auth.GenerateAccessToken(env.cfg.JWT, admin.ID, admin.Role)
	if err != nil {
		t.Fatalf("generate admin access token: %v", err)
	}
	challenge := createHandlerVMChallenge(t, env, "inactive-vm")
	challenge.IsActive = false
	if err := env.challengeRepo.Update(context.Background(), challenge); err != nil {
		t.Fatalf("update inactive challenge: %v", err)
	}

	mock := &vm.MockClient{
		CreateSandboxFn: func(ctx context.Context, id string, specYAML string) (*vm.Sandbox, error) {
			return &vm.Sandbox{ID: id, Status: vm.SandboxStatus{Phase: "pending"}}, nil
		},
		GetSandboxFn: func(ctx context.Context, vmID string) (*vm.Sandbox, error) {
			return &vm.Sandbox{ID: vmID, Status: vm.SandboxStatus{Phase: "running"}}, nil
		},
		DeleteSandboxFn: func(ctx context.Context, vmID string) error { return nil },
	}
	vmSvc, _ := setupHandlerVMService(t, env, mock)
	env.handler.vms = vmSvc

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/challenges/"+fmt.Sprint(challenge.ID)+"/vm", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(challenge.ID)}}
	ctx.Set("userID", user.ID)
	env.handler.CreateVM(ctx)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for user inactive vm create, got %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/challenges/"+fmt.Sprint(challenge.ID)+"/vm", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(challenge.ID)}}
	ctx.Set("userID", admin.ID)
	ctx.Request.AddCookie(&http.Cookie{Name: "access_token", Value: adminAccess})
	env.handler.CreateVM(ctx)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected admin inactive vm create success, got %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/vms", nil)
	ctx.Set("userID", user.ID)
	env.handler.ListVMs(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected user vm list ok, got %d: %s", rec.Code, rec.Body.String())
	}

	var userResp vmsListResponse
	decodeJSON(t, rec, &userResp)
	if len(userResp.VMs) != 0 {
		t.Fatalf("expected inactive vm hidden from user list, got %d", len(userResp.VMs))
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/vms", nil)
	ctx.Set("userID", admin.ID)
	ctx.Request.AddCookie(&http.Cookie{Name: "access_token", Value: adminAccess})
	env.handler.ListVMs(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected admin vm list ok, got %d: %s", rec.Code, rec.Body.String())
	}

	var adminResp vmsListResponse
	decodeJSON(t, rec, &adminResp)
	if len(adminResp.VMs) != 1 {
		t.Fatalf("expected admin vm list to include inactive vm, got %d", len(adminResp.VMs))
	}
}

// Registration Key Handler Tests

func TestHandlerRegistrationKeys(t *testing.T) {
	env := setupHandlerTest(t)
	admin := createHandlerUser(t, env, "admin@example.com", models.AdminRole, "pass", models.AdminRole)
	team := createHandlerTeam(t, env, "Alpha")

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/admin/registration-keys", map[string]int{"count": 1})
	ctx.Set("userID", admin.ID)

	env.handler.CreateRegistrationKeys(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("create keys missing team status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/admin/registration-keys", map[string]int{"count": 0, "team_id": int(team.ID)})
	ctx.Set("userID", admin.ID)

	env.handler.CreateRegistrationKeys(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("create keys invalid status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/admin/registration-keys", map[string]int{"count": 2, "team_id": int(team.ID), "max_uses": 3})
	ctx.Set("userID", admin.ID)

	env.handler.CreateRegistrationKeys(ctx)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create keys status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/admin/registration-keys", nil)
	ctx.Set("userID", admin.ID)

	env.handler.ListRegistrationKeys(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("list keys status %d: %s", rec.Code, rec.Body.String())
	}

	key := createHandlerRegistrationKeyWithTeam(t, env, "ABCDEFGHJKLMNPQ7", admin.ID, team.ID)

	keyID := strconv.FormatInt(key.ID, 10)
	ctx, rec = newJSONContext(t, http.MethodGet, "/api/admin/registration-keys/export?ids="+keyID, nil)
	ctx.Request.URL.RawQuery = "ids=" + keyID
	ctx.Set("userID", admin.ID)

	env.handler.ExportRegistrationKeys(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("export keys status %d: %s", rec.Code, rec.Body.String())
	}

	importBody := map[string]any{
		"version": 1,
		"registration_keys": []map[string]any{
			{
				"id":         1234,
				"code":       "ABCDEFGHJKLMNPQ8",
				"team_name":  team.Name,
				"max_uses":   2,
				"created_at": time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
	ctx, rec = newJSONContext(t, http.MethodPost, "/api/admin/registration-keys/import", importBody)
	ctx.Set("userID", admin.ID)

	env.handler.ImportRegistrationKeys(ctx)
	if rec.Code != http.StatusCreated {
		t.Fatalf("import keys status %d: %s", rec.Code, rec.Body.String())
	}
}

// Scoreboard Handler Tests

func TestHandlerLeaderboardTimelineSolved(t *testing.T) {
	env := setupHandlerTest(t)
	user1 := createHandlerUser(t, env, "user1@example.com", "user1", "pass", models.UserRole)
	user2 := createHandlerUser(t, env, "user2@example.com", "user2", "pass", models.UserRole)
	ch1 := createHandlerChallenge(t, env, "Ch1", 100, "FLAG{1}", true)
	ch2 := createHandlerChallenge(t, env, "Ch2", 50, "FLAG{2}", true)

	createHandlerSubmission(t, env, user1.ID, ch1.ID, true, time.Now().Add(-2*time.Minute))
	createHandlerSubmission(t, env, user2.ID, ch2.ID, true, time.Now().Add(-1*time.Minute))

	ctx, rec := newJSONContext(t, http.MethodGet, fmt.Sprintf("/api/leaderboard?division_id=%d", env.defaultDivisionID), nil)
	env.handler.Leaderboard(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("leaderboard status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodGet, fmt.Sprintf("/api/timeline?division_id=%d", env.defaultDivisionID), nil)
	env.handler.Timeline(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("timeline status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/users/solved", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "bad"}}
	env.handler.GetUserSolved(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("get user solved invalid status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/users/1/solved", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", user1.ID)}}
	env.handler.GetUserSolved(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("get user solved status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/users/1/solved", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", user1.ID)}}
	env.handler.GetUserSolved(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("me solved status %d: %s", rec.Code, rec.Body.String())
	}

	team := createHandlerTeam(t, env, "Alpha")
	teamUser1 := createHandlerUserWithTeam(t, env, "t1@example.com", "t1", "pass", models.UserRole, team.ID)
	teamUser2 := createHandlerUserWithTeam(t, env, "t2@example.com", "t2", "pass", models.UserRole, team.ID)
	teamChallenge := createHandlerChallenge(t, env, "TeamSolved", 120, "FLAG{TEAM}", true)

	createHandlerSubmission(t, env, teamUser1.ID, teamChallenge.ID, true, time.Now().Add(-time.Minute))

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/users/1/solved", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", teamUser2.ID)}}
	env.handler.GetUserSolved(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("me solved status %d: %s", rec.Code, rec.Body.String())
	}

	var personal []struct {
		ChallengeID int64 `json:"challenge_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &personal); err != nil {
		t.Fatalf("decode me solved: %v", err)
	}

	if len(personal) != 0 {
		t.Fatalf("expected personal solved empty, got %+v", personal)
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/teams/1/solved", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", team.ID)}}
	env.handler.ListTeamSolved(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("me solved team status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerTeamScoreboard(t *testing.T) {
	env := setupHandlerTest(t)
	teamA := createHandlerTeam(t, env, "Alpha")
	teamB := createHandlerTeam(t, env, "Beta")
	teamC := createHandlerTeam(t, env, "Gamma")
	user1 := createHandlerUserWithTeam(t, env, "u1@example.com", "u1", "pass", models.UserRole, teamA.ID)
	user2 := createHandlerUserWithTeam(t, env, "u2@example.com", "u2", "pass", models.UserRole, teamB.ID)
	user3 := createHandlerUserWithTeam(t, env, "u3@example.com", "u3", "pass", models.UserRole, teamC.ID)

	ch1 := createHandlerChallenge(t, env, "Ch1", 100, "FLAG{1}", true)
	ch2 := createHandlerChallenge(t, env, "Ch2", 50, "FLAG{2}", true)

	createHandlerSubmission(t, env, user1.ID, ch1.ID, true, time.Now().Add(-3*time.Minute))
	createHandlerSubmission(t, env, user2.ID, ch2.ID, true, time.Now().Add(-2*time.Minute))
	createHandlerSubmission(t, env, user3.ID, ch2.ID, true, time.Now().Add(-1*time.Minute))

	ctx, rec := newJSONContext(t, http.MethodGet, fmt.Sprintf("/api/leaderboard/teams?division_id=%d", env.defaultDivisionID), nil)
	env.handler.TeamLeaderboard(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("team leaderboard status %d: %s", rec.Code, rec.Body.String())
	}

	var leaderboard struct {
		Challenges []struct {
			ID int64 `json:"id"`
		} `json:"challenges"`
		Entries []struct {
			TeamName string `json:"team_name"`
			Score    int    `json:"score"`
			Solves   []struct {
				ChallengeID int64 `json:"challenge_id"`
			} `json:"solves"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &leaderboard); err != nil {
		t.Fatalf("decode leaderboard: %v", err)
	}

	if len(leaderboard.Entries) != 3 || leaderboard.Entries[0].TeamName != "Alpha" || leaderboard.Entries[2].TeamName != "Gamma" {
		t.Fatalf("unexpected leaderboard: %+v", leaderboard)
	}

	if len(leaderboard.Challenges) != 2 {
		t.Fatalf("expected 2 challenges, got %d", len(leaderboard.Challenges))
	}

	ctx, rec = newJSONContext(t, http.MethodGet, fmt.Sprintf("/api/timeline/teams?division_id=%d", env.defaultDivisionID), nil)
	env.handler.TeamTimeline(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("team timeline status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Submissions []struct {
			TeamName       string `json:"team_name"`
			Points         int    `json:"points"`
			ChallengeCount int    `json:"challenge_count"`
		} `json:"submissions"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode timeline: %v", err)
	}

	if len(resp.Submissions) == 0 || resp.Submissions[0].TeamName == "" {
		t.Fatalf("unexpected timeline response: %+v", resp)
	}
}

func TestHandlerScoreboardHidesChallengesBeforeStart(t *testing.T) {
	env := setupHandlerTest(t)
	start := time.Now().Add(2 * time.Hour)
	setHandlerCTFWindow(t, env, &start, nil)

	team := createHandlerTeam(t, env, "Alpha")
	user := createHandlerUser(t, env, "u1@example.com", "u1", "pass", models.UserRole)
	teamUser := createHandlerUserWithTeam(t, env, "t1@example.com", "t1", "pass", models.UserRole, team.ID)
	ch1 := createHandlerChallenge(t, env, "Ch1", 100, "FLAG{1}", true)

	createHandlerSubmission(t, env, user.ID, ch1.ID, true, time.Now().Add(-2*time.Minute))
	createHandlerSubmission(t, env, teamUser.ID, ch1.ID, true, time.Now().Add(-time.Minute))

	ctx, rec := newJSONContext(t, http.MethodGet, fmt.Sprintf("/api/leaderboard?division_id=%d", env.defaultDivisionID), nil)
	env.handler.Leaderboard(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("leaderboard status %d: %s", rec.Code, rec.Body.String())
	}

	var leaderboard struct {
		Challenges []struct {
			ID int64 `json:"id"`
		} `json:"challenges"`
	}
	decodeJSON(t, rec, &leaderboard)
	if len(leaderboard.Challenges) != 0 {
		t.Fatalf("expected no leaderboard challenges before start, got %d", len(leaderboard.Challenges))
	}

	ctx, rec = newJSONContext(t, http.MethodGet, fmt.Sprintf("/api/leaderboard/teams?division_id=%d", env.defaultDivisionID), nil)
	env.handler.TeamLeaderboard(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("team leaderboard status %d: %s", rec.Code, rec.Body.String())
	}

	var teamLeaderboard struct {
		Challenges []struct {
			ID int64 `json:"id"`
		} `json:"challenges"`
	}
	decodeJSON(t, rec, &teamLeaderboard)
	if len(teamLeaderboard.Challenges) != 0 {
		t.Fatalf("expected no team leaderboard challenges before start, got %d", len(teamLeaderboard.Challenges))
	}
}

func TestHandlerTimelineUsesCache(t *testing.T) {
	env := setupHandlerTest(t)
	cacheKey := cacheKeyForDivision(env, "timeline:users")
	payload := []byte(`{"submissions":[]}`)

	if err := env.redis.Set(context.Background(), cacheKey, payload, time.Minute).Err(); err != nil {
		t.Fatalf("set cache: %v", err)
	}

	ctx, rec := newJSONContext(t, http.MethodGet, fmt.Sprintf("/api/timeline?division_id=%d", env.defaultDivisionID), nil)
	env.handler.Timeline(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("timeline cache status %d: %s", rec.Code, rec.Body.String())
	}

	if !bytes.Equal(rec.Body.Bytes(), payload) {
		t.Fatalf("expected cached response")
	}
}

func TestHandlerTeamTimelineUsesCache(t *testing.T) {
	env := setupHandlerTest(t)
	cacheKey := cacheKeyForDivision(env, "timeline:teams")
	payload := []byte(`{"submissions":[]}`)

	if err := env.redis.Set(context.Background(), cacheKey, payload, time.Minute).Err(); err != nil {
		t.Fatalf("set cache: %v", err)
	}

	ctx, rec := newJSONContext(t, http.MethodGet, fmt.Sprintf("/api/timeline/teams?division_id=%d", env.defaultDivisionID), nil)
	env.handler.TeamTimeline(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("team timeline cache status %d: %s", rec.Code, rec.Body.String())
	}

	if !bytes.Equal(rec.Body.Bytes(), payload) {
		t.Fatalf("expected cached response")
	}
}

func TestHandlerLeaderboardUsesCache(t *testing.T) {
	env := setupHandlerTest(t)
	cacheKey := cacheKeyForDivision(env, "leaderboard:users")
	payload := []byte(`{"challenges":[],"entries":[]}`)

	if err := env.redis.Set(context.Background(), cacheKey, payload, time.Minute).Err(); err != nil {
		t.Fatalf("set cache: %v", err)
	}

	ctx, rec := newJSONContext(t, http.MethodGet, fmt.Sprintf("/api/leaderboard?division_id=%d", env.defaultDivisionID), nil)
	env.handler.Leaderboard(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("leaderboard cache status %d: %s", rec.Code, rec.Body.String())
	}

	if !bytes.Equal(rec.Body.Bytes(), payload) {
		t.Fatalf("expected cached response")
	}
}

func TestHandlerTeamLeaderboardUsesCache(t *testing.T) {
	env := setupHandlerTest(t)
	cacheKey := cacheKeyForDivision(env, "leaderboard:teams")
	payload := []byte(`{"challenges":[],"entries":[]}`)

	if err := env.redis.Set(context.Background(), cacheKey, payload, time.Minute).Err(); err != nil {
		t.Fatalf("set cache: %v", err)
	}

	ctx, rec := newJSONContext(t, http.MethodGet, fmt.Sprintf("/api/leaderboard/teams?division_id=%d", env.defaultDivisionID), nil)
	env.handler.TeamLeaderboard(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("leaderboard teams cache status %d: %s", rec.Code, rec.Body.String())
	}

	if !bytes.Equal(rec.Body.Bytes(), payload) {
		t.Fatalf("expected cached response")
	}
}

func TestHandlerLeaderboardError(t *testing.T) {
	closedDB := newClosedHandlerDB(t)
	scoreRepo := repo.NewScoreboardRepo(closedDB)
	scoreSvc := service.NewScoreboardService(scoreRepo)
	appConfigRepo := repo.NewAppConfigRepo(closedDB)
	appConfigSvc := service.NewAppConfigService(appConfigRepo, handlerRedis, handlerCfg.Cache.AppConfigTTL)
	handler := New(handlerCfg, nil, nil, appConfigSvc, nil, scoreSvc, nil, nil, nil, handlerRedis, nil)

	divisionID := int64(1)
	ctx, rec := newJSONContext(t, http.MethodGet, fmt.Sprintf("/api/leaderboard?division_id=%d", divisionID), nil)
	handler.Leaderboard(ctx)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("leaderboard status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerListChallengesError(t *testing.T) {
	closedDB := newClosedHandlerDB(t)
	challengeRepo := repo.NewChallengeRepo(closedDB)
	submissionRepo := repo.NewSubmissionRepo(closedDB)
	fileStore := storage.NewMemoryChallengeFileStore(10 * time.Minute)
	ctfSvc := service.NewCTFService(handlerCfg, challengeRepo, submissionRepo, handlerRedis, fileStore)
	scoreRepo := repo.NewScoreboardRepo(closedDB)
	scoreSvc := service.NewScoreboardService(scoreRepo)
	appConfigRepo := repo.NewAppConfigRepo(closedDB)
	appConfigSvc := service.NewAppConfigService(appConfigRepo, handlerRedis, handlerCfg.Cache.AppConfigTTL)
	handler := New(handlerCfg, nil, ctfSvc, appConfigSvc, nil, scoreSvc, nil, nil, nil, handlerRedis, nil)

	divisionID := int64(1)
	ctx, rec := newJSONContext(t, http.MethodGet, fmt.Sprintf("/api/challenges?division_id=%d", divisionID), nil)
	handler.ListChallenges(ctx)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("list challenges status %d: %s", rec.Code, rec.Body.String())
	}
}

func newClosedHandlerDB(t *testing.T) *bun.DB {
	t.Helper()
	conn, err := db.New(handlerCfg.DB, "test")
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	_ = conn.Close()
	return conn
}

// Team Handler Tests

func TestHandlerCreateTeam(t *testing.T) {
	env := setupHandlerTest(t)

	longName := "Alpha Team With Long Name"
	ctx, rec := newJSONContext(t, http.MethodPost, "/api/admin/teams", map[string]any{"name": longName, "division_id": env.defaultDivisionID})
	env.handler.CreateTeam(ctx)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create team status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	decodeJSON(t, rec, &resp)
	if resp.ID == 0 || resp.Name != longName {
		t.Fatalf("unexpected team response: %+v", resp)
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/admin/teams", map[string]any{})
	env.handler.CreateTeam(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/admin/teams", map[string]any{"name": longName, "division_id": env.defaultDivisionID})
	env.handler.CreateTeam(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected duplicate 400, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/admin/teams", map[string]any{"name": "Beta", "division_id": env.defaultDivisionID + 999})
	env.handler.CreateTeam(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid division 400, got %d", rec.Code)
	}
}

func TestHandlerTeams(t *testing.T) {
	env := setupHandlerTest(t)
	team := createHandlerTeam(t, env, "Alpha")
	user := createHandlerUserWithTeam(t, env, "u1@example.com", "u1", "pass", models.UserRole, team.ID)

	challenge := createHandlerChallenge(t, env, "Ch1", 100, "FLAG{1}", true)
	createHandlerSubmission(t, env, user.ID, challenge.ID, true, time.Now().Add(-time.Minute))

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/teams", nil)
	env.handler.ListTeams(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("list teams status %d: %s", rec.Code, rec.Body.String())
	}

	var teams []struct {
		ID          int64 `json:"id"`
		MemberCount int   `json:"member_count"`
		TotalScore  int   `json:"total_score"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &teams); err != nil {
		t.Fatalf("decode teams: %v", err)
	}

	if len(teams) != 1 || teams[0].ID != team.ID || teams[0].MemberCount != 1 || teams[0].TotalScore != 100 {
		t.Fatalf("unexpected teams: %+v", teams)
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/teams/1", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "0"}}
	env.handler.GetTeam(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("get team invalid status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/teams/1", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	env.handler.GetTeam(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("get team status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/teams/1/members", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	env.handler.ListTeamMembers(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("members status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/teams/1/solved", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	env.handler.ListTeamSolved(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("solved status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerListUsersDivisionFilter(t *testing.T) {
	env := setupHandlerTest(t)

	teamA := createHandlerTeam(t, env, "Alpha")
	_ = createHandlerUserWithTeam(t, env, "a@example.com", "a", "pass", models.UserRole, teamA.ID)

	divB := createHandlerDivision(t, env, "B")
	teamB := createHandlerTeamInDivision(t, env, "Beta", divB.ID)
	userB := createHandlerUserWithTeam(t, env, "b@example.com", "b", "pass", models.UserRole, teamB.ID)

	ctx, rec := newJSONContext(t, http.MethodGet, fmt.Sprintf("/api/users?division_id=%d", divB.ID), nil)
	env.handler.ListUsers(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("list users status %d: %s", rec.Code, rec.Body.String())
	}

	var users []struct {
		ID         int64 `json:"id"`
		DivisionID int64 `json:"division_id"`
	}
	decodeJSON(t, rec, &users)

	if len(users) != 1 || users[0].ID != userB.ID || users[0].DivisionID != divB.ID {
		t.Fatalf("unexpected users list: %+v", users)
	}
}

// User Handler Tests

func TestHandlerMeUpdateUsers(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "user@example.com", "user1", "pass", models.UserRole)

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/me", nil)
	ctx.Set("userID", user.ID)

	env.handler.Me(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("me status %d: %s", rec.Code, rec.Body.String())
	}

	var meResp struct {
		ID      int64 `json:"id"`
		VMCount int   `json:"vm_count"`
		VMLimit int   `json:"vm_limit"`
	}
	decodeJSON(t, rec, &meResp)
	if meResp.ID != user.ID {
		t.Fatalf("unexpected me response id: %d", meResp.ID)
	}

	if meResp.VMCount != 0 {
		t.Fatalf("expected me vm_count 0, got %d", meResp.VMCount)
	}

	if meResp.VMLimit != env.cfg.VM.MaxPer {
		t.Fatalf("expected me vm_limit %d, got %d", env.cfg.VM.MaxPer, meResp.VMLimit)
	}

	ctx, rec = newJSONContext(t, http.MethodPut, "/api/me", map[string]string{"username": "user2"})
	ctx.Set("userID", user.ID)

	setCachePayload(t, env, cacheKeyForDivision(env, "leaderboard:users"), []byte(`{"challenges":[],"entries":[]}`))
	setCachePayload(t, env, cacheKeyForDivision(env, "leaderboard:teams"), []byte(`{"challenges":[],"entries":[]}`))
	setCachePayload(t, env, cacheKeyForDivision(env, "timeline:users"), []byte(`{"submissions":[]}`))
	setCachePayload(t, env, cacheKeyForDivision(env, "timeline:teams"), []byte(`{"submissions":[]}`))

	env.handler.UpdateMe(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("update me status %d: %s", rec.Code, rec.Body.String())
	}

	var updateResp struct {
		ID      int64 `json:"id"`
		VMCount int   `json:"vm_count"`
		VMLimit int   `json:"vm_limit"`
	}
	decodeJSON(t, rec, &updateResp)
	if updateResp.ID != user.ID {
		t.Fatalf("unexpected update response id: %d", updateResp.ID)
	}

	if updateResp.VMCount != 0 {
		t.Fatalf("expected update vm_count 0, got %d", updateResp.VMCount)
	}

	if updateResp.VMLimit != env.cfg.VM.MaxPer {
		t.Fatalf("expected update vm_limit %d, got %d", env.cfg.VM.MaxPer, updateResp.VMLimit)
	}

	waitForCacheClear(t, env,
		cacheKeyForDivision(env, "leaderboard:users"),
		cacheKeyForDivision(env, "leaderboard:teams"),
		cacheKeyForDivision(env, "timeline:users"),
		cacheKeyForDivision(env, "timeline:teams"),
	)

	ctx, rec = newJSONContext(t, http.MethodPut, "/api/me", "")
	ctx.Set("userID", user.ID)

	env.handler.UpdateMe(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("update me bind status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/users", nil)
	env.handler.ListUsers(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("list users status %d: %s", rec.Code, rec.Body.String())
	}

	var users []struct {
		ID            int64   `json:"id"`
		BlockedReason *string `json:"blocked_reason"`
	}
	decodeJSON(t, rec, &users)
	if len(users) == 0 {
		t.Fatalf("expected users list")
	}

	user.BlockedReason = ptrString("policy")
	now := time.Now().UTC()
	user.BlockedAt = &now
	if err := env.userRepo.Update(context.Background(), user); err != nil {
		t.Fatalf("update user: %v", err)
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/users/1", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", user.ID)}}

	env.handler.GetUser(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("get user status %d: %s", rec.Code, rec.Body.String())
	}

	var detail struct {
		ID            int64   `json:"id"`
		BlockedReason *string `json:"blocked_reason"`
	}
	decodeJSON(t, rec, &detail)
	if detail.ID != user.ID || detail.BlockedReason == nil {
		t.Fatalf("expected blocked reason, got %+v", detail)
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/users/0", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "0"}}

	env.handler.GetUser(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("get user invalid status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/users/1", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", user.ID)}}

	env.handler.GetUser(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("get user status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerNotifyScoreboardChangedPublishesEvent(t *testing.T) {
	env := setupHandlerTest(t)

	setCachePayload(t, env, cacheKeyForDivision(env, "leaderboard:users"), []byte(`{"challenges":[],"entries":[]}`))
	setCachePayload(t, env, cacheKeyForDivision(env, "leaderboard:teams"), []byte(`{"challenges":[],"entries":[]}`))
	setCachePayload(t, env, cacheKeyForDivision(env, "timeline:users"), []byte(`{"submissions":[]}`))
	setCachePayload(t, env, cacheKeyForDivision(env, "timeline:teams"), []byte(`{"submissions":[]}`))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sub := env.redis.Subscribe(ctx, "scoreboard.events")
	defer sub.Close()

	env.handler.notifyScoreboardChanged(ctx, "test_reason", env.defaultDivisionID)

	waitForCacheClear(t, env,
		cacheKeyForDivision(env, "leaderboard:users"),
		cacheKeyForDivision(env, "leaderboard:teams"),
		cacheKeyForDivision(env, "timeline:users"),
		cacheKeyForDivision(env, "timeline:teams"),
	)

	msg, err := sub.ReceiveMessage(ctx)
	if err != nil {
		t.Fatalf("receive event: %v", err)
	}

	var got realtime.ScoreboardEvent
	if err := json.Unmarshal([]byte(msg.Payload), &got); err != nil {
		t.Fatalf("decode event: %v", err)
	}
	if got.Reason != "test_reason" || got.Scope != "division" || len(got.DivisionIDs) != 1 || got.DivisionIDs[0] != env.defaultDivisionID {
		t.Fatalf("unexpected event: %+v", got)
	}
}

func TestHandlerNotifyScoreboardChangedPublishesMultipleDivisions(t *testing.T) {
	env := setupHandlerTest(t)

	other := models.Division{Name: "Other", CreatedAt: time.Now().UTC()}
	if err := env.divisionRepo.Create(context.Background(), &other); err != nil {
		t.Fatalf("create division: %v", err)
	}

	cacheA := cacheKeyWithDivision("leaderboard:users", &env.defaultDivisionID)
	cacheB := cacheKeyWithDivision("leaderboard:users", &other.ID)
	setCachePayload(t, env, cacheA, []byte(`{"challenges":[],"entries":[]}`))
	setCachePayload(t, env, cacheB, []byte(`{"challenges":[],"entries":[]}`))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sub := env.redis.Subscribe(ctx, "scoreboard.events")
	defer sub.Close()

	env.handler.notifyScoreboardChanged(ctx, "test_multi", env.defaultDivisionID, other.ID, env.defaultDivisionID)

	waitForCacheClear(t, env, cacheA, cacheB)

	msg, err := sub.ReceiveMessage(ctx)
	if err != nil {
		t.Fatalf("receive event: %v", err)
	}

	var got realtime.ScoreboardEvent
	if err := json.Unmarshal([]byte(msg.Payload), &got); err != nil {
		t.Fatalf("decode event: %v", err)
	}

	if got.Scope != "division" || got.Reason != "test_multi" {
		t.Fatalf("unexpected event: %+v", got)
	}

	if len(got.DivisionIDs) != 2 {
		t.Fatalf("expected 2 divisions, got %v", got.DivisionIDs)
	}

	seen := map[int64]struct{}{}
	for _, id := range got.DivisionIDs {
		seen[id] = struct{}{}
	}

	if _, ok := seen[env.defaultDivisionID]; !ok {
		t.Fatalf("missing default division id")
	}

	if _, ok := seen[other.ID]; !ok {
		t.Fatalf("missing other division id")
	}
}

func TestParseIDParamOrError(t *testing.T) {
	ctx, rec := newJSONContext(t, http.MethodGet, "/api/users/bad", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "bad"}}

	if _, ok := parseIDParamOrError(ctx, "id"); ok {
		t.Fatalf("expected invalid id to fail")
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestOptionalUserID(t *testing.T) {
	cfg := config.Config{
		JWT: config.JWTConfig{
			Secret:     "test-secret",
			Issuer:     "smctf-test",
			AccessTTL:  time.Minute,
			RefreshTTL: time.Hour,
		},
	}
	handler := New(cfg, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	token, err := auth.GenerateAccessToken(cfg.JWT, 99, models.UserRole)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	ctx, _ := newJSONContext(t, http.MethodGet, "/api/me", nil)
	ctx.Request.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	userID := handler.optionalUserID(ctx)
	if userID != 99 {
		t.Fatalf("expected userID 99, got %d", userID)
	}
}

func TestOptionalUserIDInvalidHeaders(t *testing.T) {
	cfg := config.Config{
		JWT: config.JWTConfig{
			Secret:     "test-secret",
			Issuer:     "smctf-test",
			AccessTTL:  time.Minute,
			RefreshTTL: time.Hour,
		},
	}
	handler := New(cfg, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	ctx, _ := newJSONContext(t, http.MethodGet, "/api/me", nil)
	if got := handler.optionalUserID(ctx); got != 0 {
		t.Fatalf("expected 0 for missing header, got %d", got)
	}

	ctx, _ = newJSONContext(t, http.MethodGet, "/api/me", nil)
	ctx.Request.AddCookie(&http.Cookie{Name: "access_token", Value: "not-a-jwt"})
	if got := handler.optionalUserID(ctx); got != 0 {
		t.Fatalf("expected 0 for invalid scheme, got %d", got)
	}

	ctx, _ = newJSONContext(t, http.MethodGet, "/api/me", nil)
	ctx.Request.AddCookie(&http.Cookie{Name: "access_token", Value: "not-a-token"})
	if got := handler.optionalUserID(ctx); got != 0 {
		t.Fatalf("expected 0 for malformed token, got %d", got)
	}
}

func TestUniqueDivisionIDs(t *testing.T) {
	if got := uniqueDivisionIDs(nil); got != nil {
		t.Fatalf("expected nil for empty input, got %v", got)
	}

	got := uniqueDivisionIDs([]int64{-1, 0, 3, 3, 2, 0, 2, 1})
	want := []int64{3, 2, 1}
	if len(got) != len(want) {
		t.Fatalf("unexpected length: %v", got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected result: %v", got)
		}
	}
}

func TestParseOptionalPositiveIDQuery(t *testing.T) {
	ctx, _ := newJSONContext(t, http.MethodGet, "/api/challenges", nil)
	id, err := parseOptionalPositiveIDQuery(ctx, "division_id")
	if err != nil || id != nil {
		t.Fatalf("expected nil id and nil error, got id=%v err=%v", id, err)
	}

	ctx, _ = newJSONContext(t, http.MethodGet, "/api/challenges?division_id=12", nil)
	id, err = parseOptionalPositiveIDQuery(ctx, "division_id")
	if err != nil || id == nil || *id != 12 {
		t.Fatalf("expected id 12, got id=%v err=%v", id, err)
	}

	ctx, _ = newJSONContext(t, http.MethodGet, "/api/challenges?division_id=0", nil)
	if _, err = parseOptionalPositiveIDQuery(ctx, "division_id"); err == nil {
		t.Fatalf("expected validation error for zero")
	}
}

func TestOptionalStringUnmarshalJSON(t *testing.T) {
	var opt optionalString
	if err := opt.UnmarshalJSON([]byte(`"value"`)); err != nil {
		t.Fatalf("unmarshal string: %v", err)
	}

	if !opt.Set || opt.Value == nil || *opt.Value != "value" {
		t.Fatalf("unexpected optionalString: %+v", opt)
	}

	var nullOpt optionalString
	if err := nullOpt.UnmarshalJSON([]byte(`null`)); err != nil {
		t.Fatalf("unmarshal null: %v", err)
	}

	if !nullOpt.Set || nullOpt.Value != nil {
		t.Fatalf("expected nil value, got %+v", nullOpt)
	}
}

func TestOptionalInt64UnmarshalJSON(t *testing.T) {
	var opt optionalInt64
	if err := opt.UnmarshalJSON([]byte(`123`)); err != nil {
		t.Fatalf("unmarshal int64: %v", err)
	}

	if !opt.Set || opt.Value == nil || *opt.Value != 123 {
		t.Fatalf("unexpected optionalInt64: %+v", opt)
	}

	var nullOpt optionalInt64
	if err := nullOpt.UnmarshalJSON([]byte(`null`)); err != nil {
		t.Fatalf("unmarshal null: %v", err)
	}

	if !nullOpt.Set || nullOpt.Value != nil {
		t.Fatalf("expected nil value, got %+v", nullOpt)
	}
}

func TestTimePtrUTC(t *testing.T) {
	if timePtrUTC(nil) != nil {
		t.Fatalf("expected nil time")
	}

	loc := time.FixedZone("TEST", 3*60*60)
	value := time.Date(2025, 1, 1, 12, 0, 0, 0, loc)
	utc := timePtrUTC(&value)
	if utc == nil || utc.Location() != time.UTC {
		t.Fatalf("expected UTC time, got %v", utc)
	}
}

func TestHandlerCreateDivisionWithDiscordConfig(t *testing.T) {
	env := setupHandlerTest(t)

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/admin/divisions", map[string]any{
		"name":                        "Uni",
		"discord_role_id":             "123456789012345678",
		"discord_announce_channel_id": "987654321098765432",
	})
	env.handler.CreateDivision(ctx)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create division status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		ID                       int64   `json:"id"`
		Name                     string  `json:"name"`
		DiscordRoleID            *string `json:"discord_role_id"`
		DiscordAnnounceChannelID *string `json:"discord_announce_channel_id"`
	}
	decodeJSON(t, rec, &resp)
	if resp.ID == 0 || resp.Name != "Uni" {
		t.Fatalf("unexpected division: %+v", resp)
	}
	if resp.DiscordRoleID == nil || *resp.DiscordRoleID != "123456789012345678" {
		t.Fatalf("role id = %+v", resp.DiscordRoleID)
	}
	if resp.DiscordAnnounceChannelID == nil || *resp.DiscordAnnounceChannelID != "987654321098765432" {
		t.Fatalf("channel id = %+v", resp.DiscordAnnounceChannelID)
	}

	// Invalid snowflake -> 400.
	ctx, rec = newJSONContext(t, http.MethodPost, "/api/admin/divisions", map[string]any{"name": "Bad", "discord_role_id": "abc"})
	env.handler.CreateDivision(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid snowflake, got %d", rec.Code)
	}

	// Missing name -> 400.
	ctx, rec = newJSONContext(t, http.MethodPost, "/api/admin/divisions", map[string]any{})
	env.handler.CreateDivision(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing name, got %d", rec.Code)
	}
}

func TestHandlerUpdateDivision(t *testing.T) {
	env := setupHandlerTest(t)

	created, err := env.divisionSvc.CreateDivision(context.Background(), "Orig", nil, nil)
	if err != nil {
		t.Fatalf("seed division: %v", err)
	}

	ctx, rec := newJSONContext(t, http.MethodPut, "/api/admin/divisions/x", map[string]any{
		"name":            "Renamed",
		"discord_role_id": "555555555555555555",
	})
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", created.ID)}}
	env.handler.UpdateDivision(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Name          string  `json:"name"`
		DiscordRoleID *string `json:"discord_role_id"`
	}
	decodeJSON(t, rec, &resp)
	if resp.Name != "Renamed" || resp.DiscordRoleID == nil || *resp.DiscordRoleID != "555555555555555555" {
		t.Fatalf("unexpected update response: %+v", resp)
	}

	// Non-numeric id -> 400.
	ctx, rec = newJSONContext(t, http.MethodPut, "/api/admin/divisions/abc", map[string]any{"name": "X"})
	ctx.Params = gin.Params{{Key: "id", Value: "abc"}}
	env.handler.UpdateDivision(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad id, got %d", rec.Code)
	}

	// Unknown id -> 404.
	ctx, rec = newJSONContext(t, http.MethodPut, "/api/admin/divisions/999999", map[string]any{"name": "X"})
	ctx.Params = gin.Params{{Key: "id", Value: "999999"}}
	env.handler.UpdateDivision(ctx)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown id, got %d", rec.Code)
	}
}
