package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandlerGroups(t *testing.T) {
	env := setupHandlerTest(t)
	admin := createHandlerUser(t, env, "admin@example.com", "admin", "pass", "admin")

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/admin/groups", map[string]string{"name": "  Alpha School  "})
	env.handler.CreateGroup(ctx)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create group status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/admin/groups", map[string]string{"name": "Alpha School"})
	env.handler.CreateGroup(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("duplicate group status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/admin/groups", map[string]string{"name": "   "})
	env.handler.CreateGroup(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty group status %d: %s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/admin/groups", nil)
	env.handler.ListGroups(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("list groups status %d: %s", rec.Code, rec.Body.String())
	}

	var rows []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode groups: %v", err)
	}

	if len(rows) != 1 || rows[0].Name != "Alpha School" {
		t.Fatalf("unexpected groups: %+v", rows)
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/admin/registration-keys", map[string]interface{}{
		"count":    1,
		"group_id": rows[0].ID,
	})
	ctx.Set("userID", admin.ID)
	env.handler.CreateRegistrationKeys(ctx)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create keys status %d: %s", rec.Code, rec.Body.String())
	}

	var created []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode keys: %v", err)
	}

	if len(created) != 1 || created[0]["group_name"] != "Alpha School" {
		t.Fatalf("expected group name in key response, got %+v", created)
	}
}

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

	if len(leaderboard) != 3 || leaderboard[0].GroupName != "Alpha" || leaderboard[2].GroupName != "무소속" {
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
