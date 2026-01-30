package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

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
