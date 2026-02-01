package http_test

import (
	"net/http"
	"testing"
	"time"
)

func TestAdminTeams(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createUser(t, env, "admin@example.com", "admin", "adminpass", "admin")

	rec := doRequest(t, env.router, http.MethodPost, "/api/admin/teams", map[string]string{"name": "Alpha"}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	accessUser, _, _ := registerAndLogin(t, env, "user2@example.com", "user2", "strong-password")
	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/teams", map[string]string{"name": "Alpha"}, authHeader(accessUser))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	adminAccess, _, _ := loginUser(t, env.router, "admin@example.com", "adminpass")
	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/teams", map[string]string{"name": "Alpha"}, authHeader(adminAccess))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/teams", map[string]string{"name": "Alpha"}, authHeader(adminAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/teams", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var teams []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	decodeJSON(t, rec, &teams)

	if len(teams) != 1 || teams[0].Name != "Alpha" {
		t.Fatalf("unexpected teams: %+v", teams)
	}
}

func TestRegistrationKeyTeamAssignment(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createUser(t, env, "admin@example.com", "admin", "adminpass", "admin")
	team := createTeam(t, env, "Alpha")

	adminAccess, _, _ := loginUser(t, env.router, "admin@example.com", "adminpass")
	rec := doRequest(t, env.router, http.MethodPost, "/api/admin/registration-keys", map[string]any{
		"count":   1,
		"team_id": team.ID,
	}, authHeader(adminAccess))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var created []registrationKeyResp
	decodeJSON(t, rec, &created)

	if len(created) != 1 || created[0].TeamID == nil || *created[0].TeamID != team.ID {
		t.Fatalf("expected team id in key, got %+v", created)
	}

	regBody := map[string]string{
		"email":            "user1@example.com",
		"username":         "user1",
		"password":         "strong-password",
		"registration_key": created[0].Code,
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/auth/register", regBody, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var regResp struct {
		ID int64 `json:"id"`
	}
	decodeJSON(t, rec, &regResp)

	rec = doRequest(t, env.router, http.MethodGet, "/api/users/"+itoa(regResp.ID), nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var userResp struct {
		TeamID   *int64 `json:"team_id"`
		TeamName string `json:"team_name"`
	}
	decodeJSON(t, rec, &userResp)

	if userResp.TeamID == nil || *userResp.TeamID != team.ID || userResp.TeamName != "Alpha" {
		t.Fatalf("expected team assignment, got %+v", userResp)
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/registration-keys", map[string]any{
		"count":   1,
		"team_id": 9999,
	}, authHeader(adminAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestTeamsDetailMembersSolved(t *testing.T) {
	env := setupTest(t, testCfg)
	team := createTeam(t, env, "Alpha")
	user1 := createUserWithTeam(t, env, "u1@example.com", "u1", "pass", "user", &team.ID)
	user2 := createUserWithTeam(t, env, "u2@example.com", "u2", "pass", "user", &team.ID)
	ch1 := createChallenge(t, env, "Ch1", 100, "flag{1}", true)
	ch2 := createChallenge(t, env, "Ch2", 50, "flag{2}", true)

	createSubmission(t, env, user1.ID, ch1.ID, true, time.Now().UTC())
	createSubmission(t, env, user2.ID, ch2.ID, true, time.Now().UTC())

	rec := doRequest(t, env.router, http.MethodGet, "/api/teams", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var list []struct {
		ID          int64  `json:"id"`
		Name        string `json:"name"`
		MemberCount int    `json:"member_count"`
		TotalScore  int    `json:"total_score"`
	}
	decodeJSON(t, rec, &list)

	if len(list) != 1 || list[0].ID != team.ID || list[0].MemberCount != 2 || list[0].TotalScore != 150 {
		t.Fatalf("unexpected team list: %+v", list)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/teams/"+itoa(team.ID), nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var detail struct {
		ID          int64  `json:"id"`
		Name        string `json:"name"`
		MemberCount int    `json:"member_count"`
		TotalScore  int    `json:"total_score"`
	}
	decodeJSON(t, rec, &detail)

	if detail.ID != team.ID || detail.MemberCount != 2 || detail.TotalScore != 150 {
		t.Fatalf("unexpected team detail: %+v", detail)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/teams/"+itoa(team.ID)+"/members", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var members []struct {
		ID int64 `json:"id"`
	}
	decodeJSON(t, rec, &members)

	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/teams/"+itoa(team.ID)+"/solved", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var solved []struct {
		ChallengeID int64 `json:"challenge_id"`
		SolveCount  int   `json:"solve_count"`
	}
	decodeJSON(t, rec, &solved)

	if len(solved) != 2 || solved[0].SolveCount < 1 {
		t.Fatalf("unexpected solved list: %+v", solved)
	}
}
