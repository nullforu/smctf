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

func TestHandlerGroupScoreboard(t *testing.T) {
	env := setupHandlerTest(t)
	groupA := createHandlerGroup(t, env, "Alpha")
	groupB := createHandlerGroup(t, env, "Beta")
	user1 := createHandlerUser(t, env, "u1@example.com", "u1", "pass", "user")
	user2 := createHandlerUser(t, env, "u2@example.com", "u2", "pass", "user")
	user3 := createHandlerUser(t, env, "u3@example.com", "u3", "pass", "user")

	user1.GroupID = &groupA.ID
	user2.GroupID = &groupB.ID
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

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/leaderboard/groups", nil)
	env.handler.GroupLeaderboard(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("group leaderboard status %d: %s", rec.Code, rec.Body.String())
	}

	var leaderboard []struct {
		GroupName string `json:"group_name"`
		Score     int    `json:"score"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &leaderboard); err != nil {
		t.Fatalf("decode leaderboard: %v", err)
	}

	if len(leaderboard) != 3 || leaderboard[0].GroupName != "Alpha" || leaderboard[2].GroupName != "not affiliated" {
		t.Fatalf("unexpected leaderboard: %+v", leaderboard)
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/timeline/groups", nil)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/timeline/groups?window=60", nil)
	env.handler.GroupTimeline(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("group timeline status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Submissions []struct {
			GroupName      string `json:"group_name"`
			Points         int    `json:"points"`
			ChallengeCount int    `json:"challenge_count"`
		} `json:"submissions"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode timeline: %v", err)
	}

	if len(resp.Submissions) == 0 || resp.Submissions[0].GroupName == "" {
		t.Fatalf("unexpected timeline response: %+v", resp)
	}
}

func TestHandlerGroups(t *testing.T) {
	env := setupHandlerTest(t)
	group := createHandlerGroup(t, env, "Alpha")
	user := createHandlerUser(t, env, "u1@example.com", "u1", "pass", "user")

	user.GroupID = &group.ID
	if err := env.userRepo.Update(context.Background(), user); err != nil {
		t.Fatalf("update user: %v", err)
	}

	challenge := createHandlerChallenge(t, env, "Ch1", 100, "FLAG{1}", true)
	createHandlerSubmission(t, env, user.ID, challenge.ID, true, time.Now().Add(-time.Minute))

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/groups", nil)
	env.handler.ListGroups(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("list groups status %d: %s", rec.Code, rec.Body.String())
	}

	var groups []struct {
		ID          int64 `json:"id"`
		MemberCount int   `json:"member_count"`
		TotalScore  int   `json:"total_score"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &groups); err != nil {
		t.Fatalf("decode groups: %v", err)
	}

	if len(groups) != 1 || groups[0].ID != group.ID || groups[0].MemberCount != 1 || groups[0].TotalScore != 100 {
		t.Fatalf("unexpected groups: %+v", groups)
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/groups/1", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "0"}}
	env.handler.GetGroup(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("get group invalid status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/groups/1", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	env.handler.GetGroup(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("get group status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/groups/1/members", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	env.handler.ListGroupMembers(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("members status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/groups/1/solved", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	env.handler.ListGroupSolved(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("solved status %d: %s", rec.Code, rec.Body.String())
	}
}
