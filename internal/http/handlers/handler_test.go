package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"smctf/internal/models"
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

// Auth Handler Tests

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

// Challenge Handler Tests

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

// Registration Key Handler Tests

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

// Scoreboard Helper Tests

func TestTeamSubmissions(t *testing.T) {
	base := time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC)

	raw := []models.UserTimelineRow{
		{SubmittedAt: base.Add(2 * time.Minute), UserID: 1, Username: "user1", Points: 100},
		{SubmittedAt: base.Add(5 * time.Minute), UserID: 1, Username: "user1", Points: 200},
		{SubmittedAt: base.Add(15 * time.Minute), UserID: 1, Username: "user1", Points: 50},
		{SubmittedAt: base.Add(3 * time.Minute), UserID: 2, Username: "user2", Points: 150},
	}

	result := aggregateUserTimeline(raw)

	if len(result) != 3 {
		t.Fatalf("expected 3 teams, got %d", len(result))
	}

	if result[0].UserID != 1 || result[0].Points != 300 || result[0].ChallengeCount != 2 {
		t.Fatalf("unexpected first team: %+v", result[0])
	}

	if result[1].UserID != 2 || result[1].Points != 150 || result[1].ChallengeCount != 1 {
		t.Fatalf("unexpected second team: %+v", result[1])
	}

	if result[2].UserID != 1 || result[2].Points != 50 || result[2].ChallengeCount != 1 {
		t.Fatalf("unexpected third team: %+v", result[2])
	}
}

func TestTeamTeamSubmissions(t *testing.T) {
	base := time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC)
	teamID := int64(10)

	raw := []models.TeamTimelineRow{
		{SubmittedAt: base.Add(2 * time.Minute), TeamID: &teamID, TeamName: "Alpha", Points: 100},
		{SubmittedAt: base.Add(7 * time.Minute), TeamID: &teamID, TeamName: "Alpha", Points: 50},
		{SubmittedAt: base.Add(12 * time.Minute), TeamID: nil, TeamName: "not affiliated", Points: 30},
	}

	result := aggregateTeamTimeline(raw)

	if len(result) != 2 {
		t.Fatalf("expected 2 teams, got %d", len(result))
	}

	if result[0].TeamName != "Alpha" || result[0].Points != 150 || result[0].ChallengeCount != 2 {
		t.Fatalf("unexpected first team: %+v", result[0])
	}

	if result[1].TeamName != "not affiliated" || result[1].Points != 30 || result[1].ChallengeCount != 1 {
		t.Fatalf("unexpected second team: %+v", result[1])
	}
}

// Scoreboard Handler Tests

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

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/users/1/solved", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", user1.ID)}}
	env.handler.GetUserSolved(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("me solved status %d: %s", rec.Code, rec.Body.String())
	}

	team := createHandlerTeam(t, env, "Alpha")
	teamUser1 := createHandlerUserWithTeam(t, env, "t1@example.com", "t1", "pass", "user", &team.ID)
	teamUser2 := createHandlerUserWithTeam(t, env, "t2@example.com", "t2", "pass", "user", &team.ID)
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
	user1 := createHandlerUser(t, env, "u1@example.com", "u1", "pass", "user")
	user2 := createHandlerUser(t, env, "u2@example.com", "u2", "pass", "user")
	user3 := createHandlerUser(t, env, "u3@example.com", "u3", "pass", "user")

	user1.TeamID = &teamA.ID
	user2.TeamID = &teamB.ID
	if err := env.userRepo.Update(context.Background(), user1); err != nil {
		t.Fatalf("update user1: %v", err)
	}
	if err := env.userRepo.Update(context.Background(), user2); err != nil {
		t.Fatalf("update user2: %v", err)
	}

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

	var leaderboard []struct {
		TeamName string `json:"team_name"`
		Score    int    `json:"score"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &leaderboard); err != nil {
		t.Fatalf("decode leaderboard: %v", err)
	}

	if len(leaderboard) != 3 || leaderboard[0].TeamName != "Alpha" || leaderboard[2].TeamName != "not affiliated" {
		t.Fatalf("unexpected leaderboard: %+v", leaderboard)
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/timeline/teams", nil)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/timeline/teams?window=60", nil)
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

func TestHandlerTeamTimelineUsesCache(t *testing.T) {
	env := setupHandlerTest(t)
	cacheKey := "timeline:teams:0"
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

// Team Handler Tests

func TestHandlerTeams(t *testing.T) {
	env := setupHandlerTest(t)
	team := createHandlerTeam(t, env, "Alpha")
	user := createHandlerUser(t, env, "u1@example.com", "u1", "pass", "user")

	user.TeamID = &team.ID
	if err := env.userRepo.Update(context.Background(), user); err != nil {
		t.Fatalf("update user: %v", err)
	}

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
