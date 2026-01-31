package http_test

import (
	"net/http"
	"testing"
	"time"
)

func TestListUsers(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createUser(t, env, "user1@example.com", "user1", "pass1", "user")
	_ = createUser(t, env, "user2@example.com", "user2", "pass2", "user")
	_ = createUser(t, env, "admin@example.com", "admin", "pass3", "admin")

	rec := doRequest(t, env.router, http.MethodGet, "/api/users", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp []struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	decodeJSON(t, rec, &resp)

	if len(resp) != 3 {
		t.Fatalf("expected 3 users, got %d", len(resp))
	}

	if resp[0].Username != "user1" || resp[1].Username != "user2" || resp[2].Username != "admin" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestGetUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		env := setupTest(t, testCfg)
		user := createUser(t, env, "user1@example.com", "user1", "pass1", "user")

		rec := doRequest(t, env.router, http.MethodGet, "/api/users/"+itoa(user.ID), nil, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
			Role     string `json:"role"`
		}
		decodeJSON(t, rec, &resp)

		if resp.ID != user.ID || resp.Username != "user1" || resp.Role != "user" {
			t.Fatalf("unexpected response: %+v", resp)
		}
	})

	t.Run("not found", func(t *testing.T) {
		env := setupTest(t, testCfg)

		rec := doRequest(t, env.router, http.MethodGet, "/api/users/999", nil, nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		env := setupTest(t, testCfg)

		rec := doRequest(t, env.router, http.MethodGet, "/api/users/invalid", nil, nil)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestGetUserSolved(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		env := setupTest(t, testCfg)
		user := createUser(t, env, "user1@example.com", "user1", "pass1", "user")
		challenge := createChallenge(t, env, "Warmup", 100, "flag{ok}", true)
		createSubmission(t, env, user.ID, challenge.ID, true, time.Now().UTC())

		rec := doRequest(t, env.router, http.MethodGet, "/api/users/"+itoa(user.ID)+"/solved", nil, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp []struct {
			ChallengeID int64  `json:"challenge_id"`
			Title       string `json:"title"`
			Points      int    `json:"points"`
			SolvedAt    string `json:"solved_at"`
		}
		decodeJSON(t, rec, &resp)

		if len(resp) != 1 {
			t.Fatalf("expected 1 solved challenge, got %d", len(resp))
		}

		if resp[0].ChallengeID != challenge.ID || resp[0].Title != "Warmup" || resp[0].Points != 100 {
			t.Fatalf("unexpected response: %+v", resp)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		env := setupTest(t, testCfg)
		user := createUser(t, env, "user1@example.com", "user1", "pass1", "user")

		rec := doRequest(t, env.router, http.MethodGet, "/api/users/"+itoa(user.ID)+"/solved", nil, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp []any
		decodeJSON(t, rec, &resp)

		if len(resp) != 0 {
			t.Fatalf("expected empty list, got %d", len(resp))
		}
	})

	t.Run("not found", func(t *testing.T) {
		env := setupTest(t, testCfg)

		rec := doRequest(t, env.router, http.MethodGet, "/api/users/999/solved", nil, nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
	})
}
