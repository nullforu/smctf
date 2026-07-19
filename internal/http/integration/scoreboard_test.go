package http_test

import (
	"context"
	"fmt"
	"net/http"
	"smctf/internal/models"
	"testing"
	"time"
)

func TestScoreboard(t *testing.T) {
	env := setupTest(t, testCfg)
	user1 := createUser(t, env, "u1@example.com", "u1", "pass", models.UserRole)
	user2 := createUser(t, env, "u2@example.com", "u2", "pass", models.UserRole)
	blocked := createUser(t, env, "blocked@example.com", models.BlockedRole, "pass", models.UserRole)
	blocked.Role = models.BlockedRole
	if err := env.userRepo.Update(context.Background(), blocked); err != nil {
		t.Fatalf("block user: %v", err)
	}
	challenge1 := createChallenge(t, env, "Ch1", 100, "flag{1}", true)
	challenge2 := createChallenge(t, env, "Ch2", 200, "flag{2}", true)

	createSubmission(t, env, user1.ID, challenge1.ID, true, time.Now().UTC())
	createSubmission(t, env, user2.ID, challenge1.ID, true, time.Now().UTC())
	createSubmission(t, env, user2.ID, challenge2.ID, true, time.Now().UTC())
	createSubmission(t, env, blocked.ID, challenge2.ID, true, time.Now().UTC())

	rec := doRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/leaderboard?division_id=%d", env.defaultDivisionID), nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp models.LeaderboardResponse
	decodeJSON(t, rec, &resp)

	if len(resp.Entries) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(resp.Entries))
	}

	if resp.Entries[0].UserID != user2.ID || resp.Entries[0].Score != 300 {
		t.Fatalf("unexpected first row: %+v", resp.Entries[0])
	}
}

func TestScoreboardTeams(t *testing.T) {
	env := setupTest(t, testCfg)
	teamA := createTeam(t, env, "Alpha")
	teamB := createTeam(t, env, "Beta")
	user1 := createUserWithTeam(t, env, "u1@example.com", "u1", "pass", models.UserRole, teamA.ID)
	user2 := createUserWithTeam(t, env, "u2@example.com", "u2", "pass", models.UserRole, teamB.ID)
	user3 := createUser(t, env, "u3@example.com", "u3", "pass", models.UserRole)
	blocked := createUserWithTeam(t, env, "blocked@example.com", models.BlockedRole, "pass", models.UserRole, teamB.ID)
	blocked.Role = models.BlockedRole
	if err := env.userRepo.Update(context.Background(), blocked); err != nil {
		t.Fatalf("block user: %v", err)
	}
	challenge1 := createChallenge(t, env, "Ch1", 100, "flag{1}", true)
	challenge2 := createChallenge(t, env, "Ch2", 50, "flag{2}", true)

	createSubmission(t, env, user1.ID, challenge1.ID, true, time.Now().UTC())
	createSubmission(t, env, user2.ID, challenge2.ID, true, time.Now().UTC())
	createSubmission(t, env, user3.ID, challenge2.ID, true, time.Now().UTC())
	createSubmission(t, env, blocked.ID, challenge2.ID, true, time.Now().UTC())

	rec := doRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/leaderboard/teams?division_id=%d", env.defaultDivisionID), nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp models.TeamLeaderboardResponse
	decodeJSON(t, rec, &resp)

	if len(resp.Entries) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(resp.Entries))
	}

	if resp.Entries[0].TeamName != "Alpha" || resp.Entries[0].Score != 100 {
		t.Fatalf("unexpected first row: %+v", resp.Entries[0])
	}

	if resp.Entries[2].TeamName != "team-u3" || resp.Entries[2].Score != 50 {
		t.Fatalf("unexpected last row: %+v", resp.Entries[2])
	}
}

func TestScoreboardTeamTimeline(t *testing.T) {
	env := setupTest(t, testCfg)
	teamA := createTeam(t, env, "Alpha")
	user1 := createUserWithTeam(t, env, "u1@example.com", "u1", "pass", models.UserRole, teamA.ID)
	user2 := createUser(t, env, "u2@example.com", "u2", "pass", models.UserRole)
	blocked := createUserWithTeam(t, env, "blocked@example.com", models.BlockedRole, "pass", models.UserRole, teamA.ID)
	blocked.Role = models.BlockedRole
	if err := env.userRepo.Update(context.Background(), blocked); err != nil {
		t.Fatalf("block user: %v", err)
	}
	challenge1 := createChallenge(t, env, "Ch1", 100, "flag{1}", true)
	challenge2 := createChallenge(t, env, "Ch2", 200, "flag{2}", true)

	base := time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC)
	createSubmission(t, env, user1.ID, challenge1.ID, true, base.Add(3*time.Minute))
	createSubmission(t, env, user2.ID, challenge2.ID, true, base.Add(7*time.Minute))
	createSubmission(t, env, blocked.ID, challenge1.ID, true, base.Add(5*time.Minute))

	rec := doRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/timeline/teams?division_id=%d", env.defaultDivisionID), nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Submissions []struct {
			TeamID         int64     `json:"team_id"`
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
	user1 := createUser(t, env, "u1@example.com", "u1", "pass", models.UserRole)
	user2 := createUser(t, env, "u2@example.com", "u2", "pass", models.UserRole)
	blocked := createUser(t, env, "blocked@example.com", models.BlockedRole, "pass", models.UserRole)
	blocked.Role = models.BlockedRole
	if err := env.userRepo.Update(context.Background(), blocked); err != nil {
		t.Fatalf("block user: %v", err)
	}
	challenge1 := createChallenge(t, env, "Ch1", 100, "flag{1}", true)
	challenge2 := createChallenge(t, env, "Ch2", 200, "flag{2}", true)

	base := time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC)
	createSubmission(t, env, user1.ID, challenge1.ID, true, base.Add(3*time.Minute))
	createSubmission(t, env, user2.ID, challenge2.ID, true, base.Add(7*time.Minute))
	createSubmission(t, env, user1.ID, challenge2.ID, true, base.Add(16*time.Minute))
	createSubmission(t, env, blocked.ID, challenge2.ID, true, base.Add(20*time.Minute))

	rec := doRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/timeline?division_id=%d", env.defaultDivisionID), nil, nil)
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

func TestScoreboardDynamicScoring(t *testing.T) {
	env := setupTest(t, testCfg)
	team := createTeam(t, env, fmt.Sprintf("Alpha-%d", time.Now().UnixNano()))
	userTeam := createUserWithTeam(t, env, "team@example.com", "team", "pass123", models.UserRole, team.ID)
	userSolo := createUser(t, env, "solo@example.com", "solo", "pass123", models.UserRole)
	blocked := createUser(t, env, "blocked@example.com", models.BlockedRole, "pass123", models.UserRole)
	blocked.Role = models.BlockedRole
	if err := env.userRepo.Update(context.Background(), blocked); err != nil {
		t.Fatalf("block user: %v", err)
	}

	challenge := createChallenge(t, env, "Dynamic", 500, "flag{dynamic}", true)
	challenge.MinimumPoints = 100
	if err := env.challengeRepo.Update(context.Background(), challenge); err != nil {
		t.Fatalf("update challenge: %v", err)
	}

	createSubmission(t, env, userTeam.ID, challenge.ID, true, time.Now().UTC())
	createSubmission(t, env, userSolo.ID, challenge.ID, true, time.Now().UTC())
	createSubmission(t, env, blocked.ID, challenge.ID, true, time.Now().UTC())

	rec := doRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/leaderboard?division_id=%d", env.defaultDivisionID), nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp models.LeaderboardResponse
	decodeJSON(t, rec, &resp)

	if len(resp.Entries) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(resp.Entries))
	}

	if resp.Entries[0].Score != 100 || resp.Entries[1].Score != 100 {
		t.Fatalf("expected dynamic scores 100, got %+v", resp.Entries)
	}
}

func TestScoreboardHidesChallengesBeforeStart(t *testing.T) {
	env := setupTest(t, testCfg)
	start := time.Now().Add(2 * time.Hour)
	setCTFWindow(t, env, &start, nil)

	user := createUser(t, env, "u1@example.com", "u1", "pass", models.UserRole)
	team := createTeam(t, env, "Alpha")
	teamUser := createUserWithTeam(t, env, "t1@example.com", "t1", "pass", models.UserRole, team.ID)
	challenge := createChallenge(t, env, "Ch1", 100, "flag{1}", true)

	createSubmission(t, env, user.ID, challenge.ID, true, time.Now().UTC())
	createSubmission(t, env, teamUser.ID, challenge.ID, true, time.Now().UTC())

	rec := doRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/leaderboard?division_id=%d", env.defaultDivisionID), nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("leaderboard status %d: %s", rec.Code, rec.Body.String())
	}

	var userResp models.LeaderboardResponse
	decodeJSON(t, rec, &userResp)
	if len(userResp.Challenges) != 0 {
		t.Fatalf("expected no leaderboard challenges before start, got %d", len(userResp.Challenges))
	}

	rec = doRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/leaderboard/teams?division_id=%d", env.defaultDivisionID), nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("team leaderboard status %d: %s", rec.Code, rec.Body.String())
	}

	var teamResp models.TeamLeaderboardResponse
	decodeJSON(t, rec, &teamResp)
	if len(teamResp.Challenges) != 0 {
		t.Fatalf("expected no team leaderboard challenges before start, got %d", len(teamResp.Challenges))
	}
}

func TestScoreboardExcludesInactiveChallenges(t *testing.T) {
	env := setupTest(t, testCfg)
	activeTeam := createTeam(t, env, "ActiveTeam")
	inactiveTeam := createTeam(t, env, "InactiveTeam")
	user := createUserWithTeam(t, env, "u1@example.com", "u1", "pass", models.UserRole, activeTeam.ID)
	teamUser := createUserWithTeam(t, env, "t1@example.com", "t1", "pass", models.UserRole, inactiveTeam.ID)
	active := createChallenge(t, env, "Active", 100, "flag{active}", true)
	inactive := createChallenge(t, env, "Inactive", 200, "flag{inactive}", false)

	createSubmission(t, env, user.ID, active.ID, true, time.Now().UTC())
	createSubmission(t, env, user.ID, inactive.ID, true, time.Now().UTC())
	createSubmission(t, env, teamUser.ID, inactive.ID, true, time.Now().UTC())

	rec := doRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/leaderboard?division_id=%d", env.defaultDivisionID), nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("leaderboard status %d: %s", rec.Code, rec.Body.String())
	}

	var userResp models.LeaderboardResponse
	decodeJSON(t, rec, &userResp)
	if len(userResp.Challenges) != 1 || userResp.Challenges[0].ID != active.ID {
		t.Fatalf("expected only active challenge in user leaderboard, got %+v", userResp.Challenges)
	}
	if len(userResp.Entries) == 0 || userResp.Entries[0].Score != 100 {
		t.Fatalf("expected inactive challenge to be excluded from score, got %+v", userResp.Entries)
	}

	rec = doRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/leaderboard/teams?division_id=%d", env.defaultDivisionID), nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("team leaderboard status %d: %s", rec.Code, rec.Body.String())
	}

	var teamResp models.TeamLeaderboardResponse
	decodeJSON(t, rec, &teamResp)
	if len(teamResp.Challenges) != 1 || teamResp.Challenges[0].ID != active.ID {
		t.Fatalf("expected only active challenge in team leaderboard, got %+v", teamResp.Challenges)
	}
	foundActiveTeam := false
	for _, entry := range teamResp.Entries {
		if entry.TeamID == activeTeam.ID {
			foundActiveTeam = true
			if entry.Score != 100 {
				t.Fatalf("expected active team score 100, got %+v", entry)
			}
		}
		if entry.TeamID == inactiveTeam.ID && entry.Score != 0 {
			t.Fatalf("expected inactive challenge excluded from team score, got %+v", entry)
		}
	}
	if !foundActiveTeam {
		t.Fatalf("expected active team in team leaderboard")
	}

	rec = doRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/timeline?division_id=%d", env.defaultDivisionID), nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("timeline status %d: %s", rec.Code, rec.Body.String())
	}

	var timelineResp struct {
		Submissions []struct {
			ChallengeCount int `json:"challenge_count"`
			Points         int `json:"points"`
		} `json:"submissions"`
	}
	decodeJSON(t, rec, &timelineResp)
	if len(timelineResp.Submissions) != 1 || timelineResp.Submissions[0].Points != 100 {
		t.Fatalf("expected inactive challenge excluded from user timeline, got %+v", timelineResp.Submissions)
	}

	rec = doRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/timeline/teams?division_id=%d", env.defaultDivisionID), nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("team timeline status %d: %s", rec.Code, rec.Body.String())
	}

	var teamTimelineResp struct {
		Submissions []struct {
			TeamID         int64 `json:"team_id"`
			Points         int   `json:"points"`
			ChallengeCount int   `json:"challenge_count"`
		} `json:"submissions"`
	}
	decodeJSON(t, rec, &teamTimelineResp)
	if len(teamTimelineResp.Submissions) != 1 || teamTimelineResp.Submissions[0].TeamID != activeTeam.ID || teamTimelineResp.Submissions[0].Points != 100 {
		t.Fatalf("expected only active team timeline submission, got %+v", teamTimelineResp.Submissions)
	}
}
