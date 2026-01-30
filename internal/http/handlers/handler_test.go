package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func newJSONContext(t *testing.T, method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
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

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, dest interface{}) {
	t.Helper()

	if err := json.Unmarshal(rec.Body.Bytes(), dest); err != nil {
		t.Fatalf("decode json: %v", err)
	}
}

func TestHandlerRegisterLoginRefreshLogout(t *testing.T) {
	env := setupHandlerTest(t)
	admin := createHandlerUser(t, env, "admin@example.com", "admin", "pass", "admin")
	key := createHandlerRegistrationKey(t, env, "123456", admin.ID)

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

func TestHandlerMeUpdateUsers(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "user@example.com", "user1", "pass", "user")

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/me", nil)
	ctx.Set("userID", user.ID)

	env.handler.Me(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("me status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPut, "/api/me", map[string]string{"username": "user2"})
	ctx.Set("userID", user.ID)

	env.handler.UpdateMe(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("update me status %d: %s", rec.Code, rec.Body.String())
	}

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

func TestHandlerChallengesAndSubmit(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "user@example.com", "user1", "pass", "user")
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
	teamUser1 := createHandlerUserWithTeam(t, env, "t1@example.com", "t1", "pass", "user", &team.ID)
	teamUser2 := createHandlerUserWithTeam(t, env, "t2@example.com", "t2", "pass", "user", &team.ID)
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

	updateReq := map[string]interface{}{
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

	ctx, rec = newJSONContext(t, http.MethodPut, "/api/admin/challenges/1", map[string]interface{}{"flag": "new"})
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", challenge.ID)}}

	env.handler.UpdateChallenge(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("update challenge flag status %d: %s", rec.Code, rec.Body.String())
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

func TestHandlerRegistrationKeys(t *testing.T) {
	env := setupHandlerTest(t)
	admin := createHandlerUser(t, env, "admin@example.com", "admin", "pass", "admin")

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/admin/registration-keys", map[string]int{"count": 0})
	ctx.Set("userID", admin.ID)

	env.handler.CreateRegistrationKeys(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("create keys invalid status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/admin/registration-keys", map[string]int{"count": 2})
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

func TestHandlerLeaderboardTimelineSolved(t *testing.T) {
	env := setupHandlerTest(t)
	user1 := createHandlerUser(t, env, "user1@example.com", "user1", "pass", "user")
	user2 := createHandlerUser(t, env, "user2@example.com", "user2", "pass", "user")
	ch1 := createHandlerChallenge(t, env, "Ch1", 100, "FLAG{1}", true)
	ch2 := createHandlerChallenge(t, env, "Ch2", 50, "FLAG{2}", true)

	createHandlerSubmission(t, env, user1.ID, ch1.ID, true, time.Now().Add(-2*time.Minute))
	createHandlerSubmission(t, env, user2.ID, ch2.ID, true, time.Now().Add(-1*time.Minute))

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/leaderboard", nil)
	env.handler.Leaderboard(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("leaderboard status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/timeline?window=bad", nil)
	env.handler.Timeline(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("timeline invalid status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/timeline?window=5", nil)
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

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/me/solved", nil)
	ctx.Set("userID", user1.ID)
	env.handler.MeSolved(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("me solved status %d: %s", rec.Code, rec.Body.String())
	}

	team := createHandlerTeam(t, env, "Alpha")
	teamUser1 := createHandlerUserWithTeam(t, env, "t1@example.com", "t1", "pass", "user", &team.ID)
	teamUser2 := createHandlerUserWithTeam(t, env, "t2@example.com", "t2", "pass", "user", &team.ID)
	teamChallenge := createHandlerChallenge(t, env, "TeamSolved", 120, "FLAG{TEAM}", true)

	createHandlerSubmission(t, env, teamUser1.ID, teamChallenge.ID, true, time.Now().Add(-time.Minute))

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/me/solved", nil)
	ctx.Set("userID", teamUser2.ID)
	env.handler.MeSolved(ctx)
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

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/me/solved/team", nil)
	ctx.Set("userID", teamUser2.ID)
	env.handler.MeSolvedTeam(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("me solved team status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerCreateChallengeAndBindErrors(t *testing.T) {
	env := setupHandlerTest(t)

	admin := createHandlerUser(t, env, "admin@example.com", "admin", "pass", "admin")

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/admin/challenges", "")
	env.handler.CreateChallenge(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("create challenge bind status %d: %s", rec.Code, rec.Body.String())
	}

	body := map[string]interface{}{
		"title":       "New Challenge",
		"description": "desc",
		"category":    "Misc",
		"points":      100,
		"flag":        "FLAG{X}",
		"is_active":   true,
	}
	ctx, rec = newJSONContext(t, http.MethodPost, "/api/admin/challenges", body)
	ctx.Set("userID", admin.ID)

	env.handler.CreateChallenge(ctx)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create challenge status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerTimelineUsesCache(t *testing.T) {
	env := setupHandlerTest(t)
	cacheKey := "timeline:0"
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

func TestHandlerBindErrorDetails(t *testing.T) {
	env := setupHandlerTest(t)
	ctx, rec := newJSONContext(t, http.MethodPost, "/api/auth/register", "{")

	env.handler.Register(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bind invalid json status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/auth/login", map[string]interface{}{"email": 123, "password": true})
	env.handler.Login(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bind type status %d: %s", rec.Code, rec.Body.String())
	}
}
