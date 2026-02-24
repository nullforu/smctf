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
	"smctf/internal/repo"
	"smctf/internal/service"
	"smctf/internal/stack"
	"smctf/internal/storage"
	"smctf/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/uptrace/bun"
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
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	decodeJSON(t, rec, &loginResp)

	if loginResp.AccessToken == "" || loginResp.RefreshToken == "" {
		t.Fatalf("missing tokens")
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/auth/refresh", map[string]string{"refresh_token": "bad"})
	env.handler.Refresh(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("refresh invalid status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/auth/refresh", map[string]string{"refresh_token": loginResp.RefreshToken})
	env.handler.Refresh(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/auth/logout", map[string]string{"refresh_token": "bad"})
	env.handler.Logout(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("logout invalid status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/auth/logout", map[string]string{"refresh_token": loginResp.RefreshToken})
	env.handler.Logout(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("logout status %d: %s", rec.Code, rec.Body.String())
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

	setCachePayload(t, env, "leaderboard:users", []byte(`{"challenges":[],"entries":[]}`))
	setCachePayload(t, env, "leaderboard:teams", []byte(`{"challenges":[],"entries":[]}`))
	setCachePayload(t, env, "timeline:users", []byte(`{"submissions":[]}`))
	setCachePayload(t, env, "timeline:teams", []byte(`{"submissions":[]}`))

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

	waitForCacheClear(t, env, "leaderboard:users", "leaderboard:teams", "timeline:users", "timeline:teams")

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

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges", nil)

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

	ctx, rec = newJSONContext(t, http.MethodPut, "/api/admin/challenges/1", map[string]any{"stack_enabled": true, "stack_pod_spec": nil, "stack_target_port": 80})
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", challenge.ID)}}
	env.handler.UpdateChallenge(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for null stack_pod_spec with stack_enabled, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodPut, "/api/admin/challenges/1", "{")
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", challenge.ID)}}
	env.handler.UpdateChallenge(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodPut, "/api/admin/challenges/1", map[string]any{"stack_enabled": false, "stack_pod_spec": "   "})
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", challenge.ID)}}
	env.handler.UpdateChallenge(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for stack_pod_spec when stack disabled, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodPut, "/api/admin/challenges/1", map[string]any{
		"stack_enabled":     true,
		"stack_target_port": 70000,
		"stack_pod_spec":    "apiVersion: v1\nkind: Pod\nmetadata:\n  name: challenge\nspec:\n  containers:\n    - name: app\n      image: nginx\n      ports:\n        - containerPort: 80\n",
	})
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", challenge.ID)}}
	env.handler.UpdateChallenge(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for out-of-range stack_target_port, got %d", rec.Code)
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

func TestHandlerListChallengesNotStarted(t *testing.T) {
	env := setupHandlerTest(t)
	start := time.Now().Add(2 * time.Hour)
	setHandlerCTFWindow(t, env, &start, nil)

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges", nil)
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

func TestHandlerListChallengesNoPrereqWithAuth(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "nopreq@example.com", "nopreq", "pass", models.UserRole)
	access, err := auth.GenerateAccessToken(env.cfg.JWT, user.ID, user.Role)
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}

	challenge := createHandlerChallenge(t, env, "NoPrereq", 100, "FLAG{N}", true)

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges", nil)
	ctx.Request.Header.Set("Authorization", "Bearer "+access)
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

	prev := createHandlerChallenge(t, env, "Prev", 50, "FLAG{P}", true)
	locked := createHandlerChallenge(t, env, "Locked", 100, "FLAG{L}", true)
	locked.PreviousChallengeID = &prev.ID
	if err := env.challengeRepo.Update(context.Background(), locked); err != nil {
		t.Fatalf("update locked challenge: %v", err)
	}

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges", nil)
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

	foundLocked := false
	for _, item := range challenges {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if id, ok := row["id"].(float64); ok && int64(id) == locked.ID {
			foundLocked = true
			if row["is_locked"] != true {
				t.Fatalf("expected locked challenge to be marked is_locked")
			}

			if row["category"] != locked.Category {
				t.Fatalf("expected locked category %q, got %v", locked.Category, row["category"])
			}

			if row["initial_points"] == nil || row["minimum_points"] == nil || row["solve_count"] == nil {
				t.Fatalf("expected locked response to include points metadata")
			}

			if row["is_active"] != locked.IsActive {
				t.Fatalf("expected is_active %v, got %v", locked.IsActive, row["is_active"])
			}

			if prevID, ok := row["previous_challenge_id"].(float64); !ok || int64(prevID) != prev.ID {
				t.Fatalf("expected previous_challenge_id %d, got %v", prev.ID, row["previous_challenge_id"])
			}

			if row["previous_challenge_title"] != prev.Title {
				t.Fatalf("expected previous_challenge_title %q, got %v", prev.Title, row["previous_challenge_title"])
			}

			if row["previous_challenge_category"] != prev.Category {
				t.Fatalf(
					"expected previous_challenge_category %q, got %v",
					prev.Category,
					row["previous_challenge_category"],
				)
			}

			if _, ok := row["description"]; ok {
				t.Fatalf("expected description omitted for locked challenge")
			}
		}
	}

	if !foundLocked {
		t.Fatalf("expected locked challenge in list")
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/challenges", nil)
	ctx.Request.Header.Set("Authorization", "Bearer "+access)
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
			if row["is_locked"] != true {
				t.Fatalf("expected locked challenge for unsolved user")
			}
		}
	}

	createHandlerSubmission(t, env, user.ID, prev.ID, true, time.Now().UTC())

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/challenges", nil)
	ctx.Request.Header.Set("Authorization", "Bearer "+access)
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
	handler := New(env.cfg, env.authSvc, ctfSvc, env.appConfigSvc, env.userSvc, scoreSvc, env.teamSvc, nil, env.redis)

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

	setCachePayload(t, env, "leaderboard:users", []byte(`{"challenges":[],"entries":[]}`))
	setCachePayload(t, env, "leaderboard:teams", []byte(`{"challenges":[],"entries":[]}`))
	setCachePayload(t, env, "timeline:users", []byte(`{"submissions":[]}`))
	setCachePayload(t, env, "timeline:teams", []byte(`{"submissions":[]}`))

	ctx, rec := newJSONContext(t, http.MethodPut, "/api/admin/challenges/1", updateReq)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", challenge.ID)}}
	ctx.Set("userID", admin.ID)

	env.handler.UpdateChallenge(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("update challenge status %d: %s", rec.Code, rec.Body.String())
	}

	waitForCacheClear(t, env, "leaderboard:users", "leaderboard:teams", "timeline:users", "timeline:teams")

	setCachePayload(t, env, "leaderboard:users", []byte(`{"challenges":[],"entries":[]}`))
	setCachePayload(t, env, "leaderboard:teams", []byte(`{"challenges":[],"entries":[]}`))
	setCachePayload(t, env, "timeline:users", []byte(`{"submissions":[]}`))
	setCachePayload(t, env, "timeline:teams", []byte(`{"submissions":[]}`))

	ctx, rec = newJSONContext(t, http.MethodDelete, "/api/admin/challenges/1", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", challenge.ID)}}
	ctx.Set("userID", admin.ID)

	env.handler.DeleteChallenge(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete challenge status %d: %s", rec.Code, rec.Body.String())
	}

	waitForCacheClear(t, env, "leaderboard:users", "leaderboard:teams", "timeline:users", "timeline:teams")
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

	setCachePayload(t, env, "leaderboard:users", []byte(`{"challenges":[],"entries":[]}`))
	setCachePayload(t, env, "leaderboard:teams", []byte(`{"challenges":[],"entries":[]}`))
	setCachePayload(t, env, "timeline:users", []byte(`{"submissions":[]}`))
	setCachePayload(t, env, "timeline:teams", []byte(`{"submissions":[]}`))

	env.handler.CreateChallenge(ctx)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create challenge status %d: %s", rec.Code, rec.Body.String())
	}

	waitForCacheClear(t, env, "leaderboard:users", "leaderboard:teams", "timeline:users", "timeline:teams")
}

func createHandlerStackChallenge(t *testing.T, env handlerEnv, title string) *models.Challenge {
	t.Helper()
	podSpec := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: handler\nspec:\n  containers:\n    - name: app\n      image: nginx\n      ports:\n        - containerPort: 80\n"
	challenge := &models.Challenge{
		Title:           title,
		Description:     "desc",
		Category:        "Web",
		Points:          100,
		MinimumPoints:   100,
		FlagHash:        utils.HMACFlag(env.cfg.Security.FlagHMACSecret, "flag"),
		StackEnabled:    true,
		StackTargetPort: 80,
		StackPodSpec:    &podSpec,
		IsActive:        true,
		CreatedAt:       time.Now().UTC(),
	}

	if err := env.challengeRepo.Create(context.Background(), challenge); err != nil {
		t.Fatalf("create challenge: %v", err)
	}

	return challenge
}

func setupHandlerStackService(t *testing.T, env handlerEnv, client stack.API) (*service.StackService, *repo.StackRepo) {
	t.Helper()
	stackRepo := repo.NewStackRepo(env.db)
	stackCfg := config.StackConfig{
		Enabled:      true,
		MaxPerUser:   3,
		CreateWindow: time.Minute,
		CreateMax:    5,
	}

	stackSvc := service.NewStackService(stackCfg, stackRepo, env.challengeRepo, env.submissionRepo, client, env.redis)
	return stackSvc, stackRepo
}

func TestStackHandlersCRUD(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "u1@example.com", "u1", "pass", models.UserRole)
	challenge := createHandlerStackChallenge(t, env, "stack")

	var deleteCalls atomic.Int32
	mock := &stack.MockClient{
		CreateStackFn: func(ctx context.Context, targetPort int, podSpec string) (*stack.StackInfo, error) {
			return &stack.StackInfo{StackID: "stack-1", Status: "running", TargetPort: targetPort}, nil
		},
		GetStackStatusFn: func(ctx context.Context, stackID string) (*stack.StackStatus, error) {
			return &stack.StackStatus{StackID: stackID, Status: "running", TargetPort: 80}, nil
		},
		DeleteStackFn: func(ctx context.Context, stackID string) error {
			deleteCalls.Add(1)
			return nil
		},
	}

	stackSvc, _ := setupHandlerStackService(t, env, mock)
	env.handler.stacks = stackSvc

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/challenges/"+fmt.Sprint(challenge.ID)+"/stack", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(challenge.ID)}}
	ctx.Set("userID", user.ID)

	env.handler.CreateStack(ctx)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	var created stackResponse
	decodeJSON(t, rec, &created)
	if created.StackID == "" || created.TargetPort != 80 {
		t.Fatalf("unexpected response: %+v", created)
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/challenges/"+fmt.Sprint(challenge.ID)+"/stack", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(challenge.ID)}}
	ctx.Set("userID", user.ID)

	env.handler.GetStack(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodDelete, "/api/challenges/"+fmt.Sprint(challenge.ID)+"/stack", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(challenge.ID)}}
	ctx.Set("userID", user.ID)

	env.handler.DeleteStack(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if deleteCalls.Load() != 1 {
		t.Fatalf("expected delete call, got %d", deleteCalls.Load())
	}
}

func TestStackHandlersList(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "u2@example.com", "u2", "pass", models.UserRole)
	challenge1 := createHandlerStackChallenge(t, env, "stack-1")
	challenge2 := createHandlerStackChallenge(t, env, "stack-2")

	mock := &stack.MockClient{
		GetStackStatusFn: func(ctx context.Context, stackID string) (*stack.StackStatus, error) {
			return &stack.StackStatus{StackID: stackID, Status: "running", TargetPort: 80}, nil
		},
	}

	stackSvc, stackRepo := setupHandlerStackService(t, env, mock)
	env.handler.stacks = stackSvc

	stack1 := &models.Stack{UserID: user.ID, ChallengeID: challenge1.ID, StackID: "stack-1", Status: "running", TargetPort: 80, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	stack2 := &models.Stack{UserID: user.ID, ChallengeID: challenge2.ID, StackID: "stack-2", Status: "running", TargetPort: 80, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := stackRepo.Create(context.Background(), stack1); err != nil {
		t.Fatalf("create stack1: %v", err)
	}

	if err := stackRepo.Create(context.Background(), stack2); err != nil {
		t.Fatalf("create stack2: %v", err)
	}

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/stacks", nil)
	ctx.Set("userID", user.ID)
	env.handler.ListStacks(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp stacksListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.CTFState != string(service.CTFStateActive) {
		t.Fatalf("expected ctf_state active, got %s", resp.CTFState)
	}

	if len(resp.Stacks) != 2 {
		t.Fatalf("expected 2 stacks, got %d", len(resp.Stacks))
	}
}

func TestAdminStackHandlersList(t *testing.T) {
	env := setupHandlerTest(t)
	team := createHandlerTeam(t, env, "Alpha")
	user := createHandlerUserWithTeam(t, env, "admin@example.com", "uadmin", "pass", models.UserRole, team.ID)
	challenge := createHandlerStackChallenge(t, env, "admin-stack")

	mock := &stack.MockClient{
		GetStackStatusFn: func(ctx context.Context, stackID string) (*stack.StackStatus, error) {
			return &stack.StackStatus{StackID: stackID, Status: "running", TargetPort: 80}, nil
		},
	}

	stackSvc, stackRepo := setupHandlerStackService(t, env, mock)
	env.handler.stacks = stackSvc

	stackModel := &models.Stack{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		StackID:     "stack-admin-1",
		Status:      "running",
		TargetPort:  80,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := stackRepo.Create(context.Background(), stackModel); err != nil {
		t.Fatalf("create stack: %v", err)
	}

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/admin/stacks", nil)
	env.handler.AdminListStacks(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp adminStacksListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.Stacks) != 1 {
		t.Fatalf("expected 1 stack, got %d", len(resp.Stacks))
	}

	item := resp.Stacks[0]
	if item.StackID != "stack-admin-1" || item.Username != user.Username || item.TeamName != team.Name || item.ChallengeTitle != challenge.Title {
		t.Fatalf("unexpected admin stack response: %+v", item)
	}
}

func TestAdminStackHandlersListDisabled(t *testing.T) {
	env := setupHandlerTest(t)
	env.handler.stacks = nil

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/admin/stacks", nil)
	env.handler.AdminListStacks(ctx)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestAdminStackHandlersDelete(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "admin@example.com", "uadmin-del", "pass", models.UserRole)
	challenge := createHandlerStackChallenge(t, env, "admin-del")

	var deleteCalls atomic.Int32
	mock := &stack.MockClient{
		DeleteStackFn: func(ctx context.Context, stackID string) error {
			deleteCalls.Add(1)
			return nil
		},
	}

	stackSvc, stackRepo := setupHandlerStackService(t, env, mock)
	env.handler.stacks = stackSvc

	stackModel := &models.Stack{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		StackID:     "stack-admin-del",
		Status:      "running",
		TargetPort:  80,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := stackRepo.Create(context.Background(), stackModel); err != nil {
		t.Fatalf("create stack: %v", err)
	}

	ctx, rec := newJSONContext(t, http.MethodDelete, "/api/admin/stacks/stack-admin-del", nil)
	ctx.Params = gin.Params{{Key: "stack_id", Value: "stack-admin-del"}}
	env.handler.AdminDeleteStack(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if deleteCalls.Load() != 1 {
		t.Fatalf("expected delete call, got %d", deleteCalls.Load())
	}

	if _, err := stackRepo.GetByStackID(context.Background(), "stack-admin-del"); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("expected stack deleted, got %v", err)
	}
}

func TestAdminStackHandlersDeleteMissingStackID(t *testing.T) {
	env := setupHandlerTest(t)
	mock := &stack.MockClient{}
	stackSvc, _ := setupHandlerStackService(t, env, mock)
	env.handler.stacks = stackSvc

	ctx, rec := newJSONContext(t, http.MethodDelete, "/api/admin/stacks/", nil)
	env.handler.AdminDeleteStack(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	var resp errorResponse
	decodeJSON(t, rec, &resp)
	if resp.Error != service.ErrInvalidInput.Error() {
		t.Fatalf("expected invalid input, got %s", resp.Error)
	}
}

func TestAdminStackHandlersGet(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "u-admin-get@example.com", "uadmin-get", "pass", models.UserRole)
	challenge := createHandlerStackChallenge(t, env, "admin-get")

	mock := &stack.MockClient{
		GetStackStatusFn: func(ctx context.Context, stackID string) (*stack.StackStatus, error) {
			return &stack.StackStatus{StackID: stackID, Status: "running", TargetPort: 80}, nil
		},
	}

	stackSvc, stackRepo := setupHandlerStackService(t, env, mock)
	env.handler.stacks = stackSvc

	stackModel := &models.Stack{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		StackID:     "stack-admin-get",
		Status:      "running",
		TargetPort:  80,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := stackRepo.Create(context.Background(), stackModel); err != nil {
		t.Fatalf("create stack: %v", err)
	}

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/admin/stacks/stack-admin-get", nil)
	ctx.Params = gin.Params{{Key: "stack_id", Value: "stack-admin-get"}}
	env.handler.AdminGetStack(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp stackResponse
	decodeJSON(t, rec, &resp)
	if resp.StackID != "stack-admin-get" || resp.ChallengeID != challenge.ID {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestAdminStackHandlersGetMissingStackID(t *testing.T) {
	env := setupHandlerTest(t)
	mock := &stack.MockClient{}
	stackSvc, _ := setupHandlerStackService(t, env, mock)
	env.handler.stacks = stackSvc

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/admin/stacks/", nil)
	env.handler.AdminGetStack(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	var resp errorResponse
	decodeJSON(t, rec, &resp)
	if resp.Error != service.ErrInvalidInput.Error() {
		t.Fatalf("expected invalid input, got %s", resp.Error)
	}
}

func TestAdminStackHandlersGetNotFound(t *testing.T) {
	env := setupHandlerTest(t)
	mock := &stack.MockClient{}
	stackSvc, _ := setupHandlerStackService(t, env, mock)
	env.handler.stacks = stackSvc

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/admin/stacks/missing", nil)
	ctx.Params = gin.Params{{Key: "stack_id", Value: "missing"}}
	env.handler.AdminGetStack(ctx)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestAdminReport(t *testing.T) {
	env := setupHandlerTest(t)

	user := createHandlerUser(t, env, "report@example.com", "reporter", "pass", models.UserRole)
	challenge := createHandlerStackChallenge(t, env, "report-challenge")

	stackRepo := repo.NewStackRepo(env.db)
	stackModel := &models.Stack{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		StackID:     "stack-report",
		Status:      "running",
		TargetPort:  80,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := stackRepo.Create(context.Background(), stackModel); err != nil {
		t.Fatalf("create stack: %v", err)
	}

	mock := &stack.MockClient{}
	stackSvc := service.NewStackService(config.StackConfig{Enabled: true, MaxPerUser: 3, CreateWindow: time.Minute, CreateMax: 5}, stackRepo, env.challengeRepo, env.submissionRepo, mock, env.redis)
	env.handler.stacks = stackSvc

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

func TestStackHandlersNotStarted(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "u3@example.com", "u3", "pass", models.UserRole)
	challenge := createHandlerStackChallenge(t, env, "stack")

	start := time.Now().Add(2 * time.Hour)
	setHandlerCTFWindow(t, env, &start, nil)

	mock := &stack.MockClient{
		GetStackStatusFn: func(ctx context.Context, stackID string) (*stack.StackStatus, error) {
			return &stack.StackStatus{StackID: stackID, Status: "running", TargetPort: 80}, nil
		},
	}

	stackSvc, _ := setupHandlerStackService(t, env, mock)
	env.handler.stacks = stackSvc

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/challenges/"+fmt.Sprint(challenge.ID)+"/stack", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(challenge.ID)}}
	ctx.Set("userID", user.ID)
	env.handler.CreateStack(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("create stack status %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec, &resp)
	if resp["ctf_state"] != string(service.CTFStateNotStarted) {
		t.Fatalf("expected ctf_state not_started, got %v", resp["ctf_state"])
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/stacks", nil)
	ctx.Set("userID", user.ID)
	env.handler.ListStacks(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("list stacks status %d: %s", rec.Code, rec.Body.String())
	}

	resp = map[string]any{}
	decodeJSON(t, rec, &resp)
	if resp["ctf_state"] != string(service.CTFStateNotStarted) {
		t.Fatalf("expected ctf_state not_started, got %v", resp["ctf_state"])
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/challenges/"+fmt.Sprint(challenge.ID)+"/stack", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(challenge.ID)}}
	ctx.Set("userID", user.ID)
	env.handler.GetStack(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("get stack status %d: %s", rec.Code, rec.Body.String())
	}

	resp = map[string]any{}
	decodeJSON(t, rec, &resp)
	if resp["ctf_state"] != string(service.CTFStateNotStarted) {
		t.Fatalf("expected ctf_state not_started, got %v", resp["ctf_state"])
	}

	ctx, rec = newJSONContext(t, http.MethodDelete, "/api/challenges/"+fmt.Sprint(challenge.ID)+"/stack", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(challenge.ID)}}
	ctx.Set("userID", user.ID)
	env.handler.DeleteStack(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete stack status %d: %s", rec.Code, rec.Body.String())
	}

	resp = map[string]any{}
	decodeJSON(t, rec, &resp)
	if resp["ctf_state"] != string(service.CTFStateNotStarted) {
		t.Fatalf("expected ctf_state not_started, got %v", resp["ctf_state"])
	}
}

func TestAdminGetChallengeIncludesStackSpec(t *testing.T) {
	env := setupHandlerTest(t)
	challenge := createHandlerStackChallenge(t, env, "stack")

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

	if resp["stack_pod_spec"] == nil {
		t.Fatalf("expected stack_pod_spec in response")
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

func TestSubmitFlagDeletesStack(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "u3@example.com", "u3", "pass", models.UserRole)
	challenge := createHandlerStackChallenge(t, env, "stack")

	stackRepo := repo.NewStackRepo(env.db)
	stackModel := &models.Stack{UserID: user.ID, ChallengeID: challenge.ID, StackID: "stack-sub", Status: "running", TargetPort: 80, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := stackRepo.Create(context.Background(), stackModel); err != nil {
		t.Fatalf("create stack: %v", err)
	}

	deleted := false
	mock := &stack.MockClient{
		DeleteStackFn: func(ctx context.Context, stackID string) error {
			if stackID == "stack-sub" {
				deleted = true
			}
			return nil
		},
	}
	stackSvc := service.NewStackService(config.StackConfig{Enabled: true, MaxPerUser: 3, CreateWindow: time.Minute, CreateMax: 5}, stackRepo, env.challengeRepo, env.submissionRepo, mock, env.redis)
	env.handler.stacks = stackSvc

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/challenges/"+fmt.Sprint(challenge.ID)+"/submit", submitRequest{Flag: "flag"})
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(challenge.ID)}}
	ctx.Set("userID", user.ID)
	env.handler.SubmitFlag(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if !deleted {
		t.Fatalf("expected stack delete call")
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

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/leaderboard", nil)
	env.handler.Leaderboard(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("leaderboard status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/timeline", nil)
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

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/leaderboard/teams", nil)
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

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/timeline/teams", nil)
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

func TestHandlerTimelineUsesCache(t *testing.T) {
	env := setupHandlerTest(t)
	cacheKey := "timeline:users"
	payload := []byte(`{"submissions":[]}`)

	if err := env.redis.Set(context.Background(), cacheKey, payload, time.Minute).Err(); err != nil {
		t.Fatalf("set cache: %v", err)
	}

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/timeline", nil)
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
	cacheKey := "timeline:teams"
	payload := []byte(`{"submissions":[]}`)

	if err := env.redis.Set(context.Background(), cacheKey, payload, time.Minute).Err(); err != nil {
		t.Fatalf("set cache: %v", err)
	}

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/timeline/teams", nil)
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
	cacheKey := "leaderboard:users"
	payload := []byte(`{"challenges":[],"entries":[]}`)

	if err := env.redis.Set(context.Background(), cacheKey, payload, time.Minute).Err(); err != nil {
		t.Fatalf("set cache: %v", err)
	}

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/leaderboard", nil)
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
	cacheKey := "leaderboard:teams"
	payload := []byte(`{"challenges":[],"entries":[]}`)

	if err := env.redis.Set(context.Background(), cacheKey, payload, time.Minute).Err(); err != nil {
		t.Fatalf("set cache: %v", err)
	}

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/leaderboard/teams", nil)
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
	handler := New(handlerCfg, nil, nil, nil, nil, scoreSvc, nil, nil, handlerRedis)

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/leaderboard", nil)
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
	handler := New(handlerCfg, nil, ctfSvc, appConfigSvc, nil, scoreSvc, nil, nil, handlerRedis)

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges", nil)
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

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/admin/teams", map[string]string{"name": "Alpha"})
	env.handler.CreateTeam(ctx)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create team status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	decodeJSON(t, rec, &resp)
	if resp.ID == 0 || resp.Name != "Alpha" {
		t.Fatalf("unexpected team response: %+v", resp)
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/admin/teams", map[string]string{})
	env.handler.CreateTeam(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/admin/teams", map[string]string{"name": "Alpha"})
	env.handler.CreateTeam(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected duplicate 400, got %d", rec.Code)
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

	ctx, rec = newJSONContext(t, http.MethodPut, "/api/me", map[string]string{"username": "user2"})
	ctx.Set("userID", user.ID)

	setCachePayload(t, env, "leaderboard:users", []byte(`{"challenges":[],"entries":[]}`))
	setCachePayload(t, env, "leaderboard:teams", []byte(`{"challenges":[],"entries":[]}`))
	setCachePayload(t, env, "timeline:users", []byte(`{"submissions":[]}`))
	setCachePayload(t, env, "timeline:teams", []byte(`{"submissions":[]}`))

	env.handler.UpdateMe(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("update me status %d: %s", rec.Code, rec.Body.String())
	}

	waitForCacheClear(t, env, "leaderboard:users", "leaderboard:teams", "timeline:users", "timeline:teams")

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
