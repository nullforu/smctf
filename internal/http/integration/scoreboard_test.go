package http_test

import (
	"net/http"
	"smctf/internal/models"
	"smctf/internal/service"
	"testing"
	"time"
)

func TestScoreboard(t *testing.T) {
	env := setupTest(t, testCfg)
	user1 := createUser(t, env, "u1@example.com", "u1", "pass", "user")
	user2 := createUser(t, env, "u2@example.com", "u2", "pass", "user")
	challenge1 := createChallenge(t, env, "Ch1", 100, "flag{1}", true)
	challenge2 := createChallenge(t, env, "Ch2", 200, "flag{2}", true)

	createSubmission(t, env, user1.ID, challenge1.ID, true, time.Now().UTC())
	createSubmission(t, env, user2.ID, challenge1.ID, true, time.Now().UTC())
	createSubmission(t, env, user2.ID, challenge2.ID, true, time.Now().UTC())

	rec := doRequest(t, env.router, http.MethodGet, "/api/leaderboard", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var rows []models.LeaderboardEntry
	decodeJSON(t, rec, &rows)

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	if rows[0].UserID != user2.ID || rows[0].Score != 300 {
		t.Fatalf("unexpected first row: %+v", rows[0])
	}
}

func TestScoreboardTeams(t *testing.T) {
	env := setupTest(t, testCfg)
	teamA := createTeam(t, env, "Alpha")
	teamB := createTeam(t, env, "Beta")
	user1 := createUserWithTeam(t, env, "u1@example.com", "u1", "pass", "user", &teamA.ID)
	user2 := createUserWithTeam(t, env, "u2@example.com", "u2", "pass", "user", &teamB.ID)
	user3 := createUser(t, env, "u3@example.com", "u3", "pass", "user")
	challenge1 := createChallenge(t, env, "Ch1", 100, "flag{1}", true)
	challenge2 := createChallenge(t, env, "Ch2", 50, "flag{2}", true)

	createSubmission(t, env, user1.ID, challenge1.ID, true, time.Now().UTC())
	createSubmission(t, env, user2.ID, challenge2.ID, true, time.Now().UTC())
	createSubmission(t, env, user3.ID, challenge2.ID, true, time.Now().UTC())

	rec := doRequest(t, env.router, http.MethodGet, "/api/leaderboard/teams", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var rows []models.TeamLeaderboardEntry
	decodeJSON(t, rec, &rows)

	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}

	if rows[0].TeamName != "Alpha" || rows[0].Score != 100 {
		t.Fatalf("unexpected first row: %+v", rows[0])
	}

	if rows[2].TeamName != "not affiliated" || rows[2].Score != 50 {
		t.Fatalf("unexpected last row: %+v", rows[2])
	}
}

func TestScoreboardTeamTimeline(t *testing.T) {
	env := setupTest(t, testCfg)
	teamA := createTeam(t, env, "Alpha")
	user1 := createUserWithTeam(t, env, "u1@example.com", "u1", "pass", "user", &teamA.ID)
	user2 := createUser(t, env, "u2@example.com", "u2", "pass", "user")
	challenge1 := createChallenge(t, env, "Ch1", 100, "flag{1}", true)
	challenge2 := createChallenge(t, env, "Ch2", 200, "flag{2}", true)

	base := time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC)
	createSubmission(t, env, user1.ID, challenge1.ID, true, base.Add(3*time.Minute))
	createSubmission(t, env, user2.ID, challenge2.ID, true, base.Add(7*time.Minute))

	rec := doRequest(t, env.router, http.MethodGet, "/api/timeline/teams", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Submissions []struct {
			TeamID         *int64    `json:"team_id"`
			TeamName       string    `json:"team_name"`
			Timestamp      time.Time `json:"timestamp"`
			Points         int       `json:"points"`
			ChallengeCount int       `json:"challenge_count"`
		} `json:"submissions"`
	}
	decodeJSON(t, rec, &resp)

	if len(resp.Submissions) != 2 {
		t.Fatalf("expected 2 submissions, got %d", len(resp.Submissions))
	}

	if resp.Submissions[0].TeamName == "" || resp.Submissions[1].TeamName == "" {
		t.Fatalf("unexpected submissions: %+v", resp.Submissions)
	}
}

func TestScoreboardTimeline(t *testing.T) {
	env := setupTest(t, testCfg)
	user1 := createUser(t, env, "u1@example.com", "u1", "pass", "user")
	user2 := createUser(t, env, "u2@example.com", "u2", "pass", "user")
	challenge1 := createChallenge(t, env, "Ch1", 100, "flag{1}", true)
	challenge2 := createChallenge(t, env, "Ch2", 200, "flag{2}", true)

	base := time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC)
	createSubmission(t, env, user1.ID, challenge1.ID, true, base.Add(3*time.Minute))
	createSubmission(t, env, user2.ID, challenge2.ID, true, base.Add(7*time.Minute))
	createSubmission(t, env, user1.ID, challenge2.ID, true, base.Add(16*time.Minute))

	rec := doRequest(t, env.router, http.MethodGet, "/api/timeline", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Submissions []struct {
			Timestamp      time.Time `json:"timestamp"`
			UserID         int64     `json:"user_id"`
			Username       string    `json:"username"`
			Points         int       `json:"points"`
			ChallengeCount int       `json:"challenge_count"`
		} `json:"submissions"`
	}
	decodeJSON(t, rec, &resp)

	if len(resp.Submissions) != 3 {
		t.Fatalf("expected 3 submissions, got %d", len(resp.Submissions))
	}

	if resp.Submissions[0].UserID != 1 || resp.Submissions[0].Points != 100 || resp.Submissions[0].ChallengeCount != 1 {
		t.Fatalf("unexpected first submission: %+v", resp.Submissions[0])
	}

	if resp.Submissions[1].UserID != 2 || resp.Submissions[1].Points != 200 || resp.Submissions[1].ChallengeCount != 1 {
		t.Fatalf("unexpected second submission: %+v", resp.Submissions[1])
	}

	if resp.Submissions[2].UserID != 1 || resp.Submissions[2].Points != 200 || resp.Submissions[2].ChallengeCount != 1 {
		t.Fatalf("unexpected third submission: %+v", resp.Submissions[2])
	}
}

func TestScoreboardTimelineWindow(t *testing.T) {
	env := setupTest(t, testCfg)
	user1 := createUser(t, env, "u1@example.com", "u1", "pass", "user")
	user2 := createUser(t, env, "u2@example.com", "u2", "pass", "user")
	challenge1 := createChallenge(t, env, "Ch1", 100, "flag{1}", true)
	challenge2 := createChallenge(t, env, "Ch2", 200, "flag{2}", true)

	now := time.Now().UTC()
	createSubmission(t, env, user1.ID, challenge1.ID, true, now.Add(-2*time.Hour))

	recent := now.Add(-20 * time.Minute)
	createSubmission(t, env, user2.ID, challenge2.ID, true, recent)

	rec := doRequest(t, env.router, http.MethodGet, "/api/timeline?window=60", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Submissions []struct {
			Timestamp      time.Time `json:"timestamp"`
			UserID         int64     `json:"user_id"`
			Username       string    `json:"username"`
			Points         int       `json:"points"`
			ChallengeCount int       `json:"challenge_count"`
		} `json:"submissions"`
	}
	decodeJSON(t, rec, &resp)

	if len(resp.Submissions) != 1 {
		t.Fatalf("expected 1 submission, got %d", len(resp.Submissions))
	}

	if resp.Submissions[0].UserID != user2.ID {
		t.Fatalf("unexpected user: %d", resp.Submissions[0].UserID)
	}

	windowStart := now.Add(-60 * time.Minute)
	if resp.Submissions[0].Timestamp.Before(windowStart) {
		t.Fatalf("submission outside window: %s", resp.Submissions[0].Timestamp)
	}
}

func TestScoreboardTimelineInvalidWindow(t *testing.T) {
	env := setupTest(t, testCfg)
	rec := doRequest(t, env.router, http.MethodGet, "/api/timeline?window=0", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp errorResp
	decodeJSON(t, rec, &resp)

	if resp.Error != service.ErrInvalidInput.Error() {
		t.Fatalf("unexpected error: %s", resp.Error)
	}
}
