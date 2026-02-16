package http_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestAdminCreateChallenge(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createUser(t, env, "admin@example.com", "admin", "adminpass", "admin")

	rec := doRequest(t, env.router, http.MethodPost, "/api/admin/challenges", map[string]string{"title": "Ch1"}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	accessUser, _, _ := registerAndLogin(t, env, "user2@example.com", "user2", "strong-password")

	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/challenges", map[string]any{
		"title":       "Ch1",
		"description": "desc",
		"category":    "Web",
		"points":      100,
		"flag":        "flag{1}",
		"is_active":   true,
	}, authHeader(accessUser))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	adminAccess, _, _ := loginUser(t, env.router, "admin@example.com", "adminpass")
	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/challenges", map[string]any{
		"title":       "Ch1",
		"description": "desc",
		"category":    "Web",
		"points":      100,
		"flag":        "flag{1}",
		"is_active":   true,
	}, authHeader(adminAccess))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/challenges", map[string]any{
		"title": "Ch2",
	}, authHeader(adminAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/challenges", map[string]any{
		"title":       "Ch3",
		"description": "desc",
		"category":    "Unknown",
		"points":      100,
		"flag":        "flag{1}",
		"is_active":   true,
	}, authHeader(adminAccess))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp errorResp
	decodeJSON(t, rec, &resp)

	assertFieldErrors(t, resp.Details, map[string]string{"category": "invalid"})
}

func TestAdminUpdateChallenge(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createUser(t, env, "admin@example.com", "admin", "adminpass", "admin")

	adminAccess, _, _ := loginUser(t, env.router, "admin@example.com", "adminpass")

	rec := doRequest(t, env.router, http.MethodPost, "/api/admin/challenges", map[string]any{
		"title":       "Ch1",
		"description": "desc",
		"category":    "Web",
		"points":      100,
		"flag":        "flag{1}",
		"is_active":   true,
	}, authHeader(adminAccess))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var created struct {
		ID int64 `json:"id"`
	}
	decodeJSON(t, rec, &created)

	rec = doRequest(t, env.router, http.MethodPut, "/api/admin/challenges/"+itoa(created.ID), map[string]any{
		"title":     "Ch1 Updated",
		"points":    150,
		"is_active": false,
	}, authHeader(adminAccess))

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var updated struct {
		ID          int64  `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Category    string `json:"category"`
		Points      int    `json:"points"`
		IsActive    bool   `json:"is_active"`
	}
	decodeJSON(t, rec, &updated)

	if updated.Title != "Ch1 Updated" || updated.Description != "desc" || updated.Category != "Web" || updated.Points != 150 || updated.IsActive != false {
		t.Fatalf("unexpected updated challenge: %+v", updated)
	}

	rec = doRequest(t, env.router, http.MethodPut, "/api/admin/challenges/"+itoa(created.ID), map[string]any{
		"category": "",
	}, authHeader(adminAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var errResp errorResp
	decodeJSON(t, rec, &errResp)

	assertFieldErrors(t, errResp.Details, map[string]string{"category": "required"})

	rec = doRequest(t, env.router, http.MethodPut, "/api/admin/challenges/"+itoa(created.ID), map[string]any{
		"category": "Unknown",
	}, authHeader(adminAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	decodeJSON(t, rec, &errResp)

	assertFieldErrors(t, errResp.Details, map[string]string{"category": "invalid"})

	rec = doRequest(t, env.router, http.MethodPut, "/api/admin/challenges/"+itoa(created.ID), map[string]any{
		"flag": "flag{rotated}",
	}, authHeader(adminAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	decodeJSON(t, rec, &errResp)

	assertFieldErrors(t, errResp.Details, map[string]string{"flag": "immutable"})
}

func TestAdminGetChallengeDetail(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createUser(t, env, "admin@example.com", "admin", "adminpass", "admin")
	adminAccess, _, _ := loginUser(t, env.router, "admin@example.com", "adminpass")

	podSpec := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: challenge\nspec:\n  containers:\n    - name: app\n      image: nginx\n      ports:\n        - containerPort: 80\n"
	challenge := createChallenge(t, env, "Stacked", 100, "flag{stack}", true)
	challenge.StackEnabled = true
	challenge.StackTargetPort = 80
	challenge.StackPodSpec = &podSpec
	if err := env.challengeRepo.Update(context.Background(), challenge); err != nil {
		t.Fatalf("update challenge: %v", err)
	}

	rec := doRequest(t, env.router, http.MethodGet, "/api/admin/challenges/"+itoa(challenge.ID), nil, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec, &resp)
	if resp["stack_pod_spec"] == nil {
		t.Fatalf("expected stack_pod_spec")
	}
}

func TestAdminDeleteChallenge(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createUser(t, env, "admin@example.com", "admin", "adminpass", "admin")

	adminAccess, _, _ := loginUser(t, env.router, "admin@example.com", "adminpass")
	rec := doRequest(t, env.router, http.MethodPost, "/api/admin/challenges", map[string]any{
		"title":       "Ch1",
		"description": "desc",
		"category":    "Web",
		"points":      100,
		"flag":        "flag{1}",
		"is_active":   true,
	}, authHeader(adminAccess))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var created struct {
		ID int64 `json:"id"`
	}
	decodeJSON(t, rec, &created)

	rec = doRequest(t, env.router, http.MethodDelete, "/api/admin/challenges/"+itoa(created.ID), nil, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var challenges struct {
		CTFState   string           `json:"ctf_state"`
		Challenges []map[string]any `json:"challenges"`
	}
	decodeJSON(t, rec, &challenges)

	if len(challenges.Challenges) != 0 {
		t.Fatalf("expected 0 challenges, got %d", len(challenges.Challenges))
	}
}

func TestAdminRegistrationKeys(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createUser(t, env, "admin@example.com", "admin", "adminpass", "admin")
	team := createTeam(t, env, fmt.Sprintf("Alpha-%d", time.Now().UnixNano()))

	rec := doRequest(t, env.router, http.MethodPost, "/api/admin/registration-keys", map[string]int{"count": 1, "team_id": int(team.ID)}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	accessUser, _, _ := registerAndLogin(t, env, "user2@example.com", "user2", "strong-password")
	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/registration-keys", map[string]int{"count": 1, "team_id": int(team.ID)}, authHeader(accessUser))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	adminAccess, _, _ := loginUser(t, env.router, "admin@example.com", "adminpass")
	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/registration-keys", map[string]int{"count": 0, "team_id": int(team.ID)}, authHeader(adminAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var errResp errorResp
	decodeJSON(t, rec, &errResp)
	assertFieldErrors(t, errResp.Details, map[string]string{"count": "must be >= 1"})

	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/registration-keys", map[string]int{"count": 2, "team_id": int(team.ID)}, authHeader(adminAccess))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var created []registrationKeyResp
	decodeJSON(t, rec, &created)

	if len(created) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(created))
	}

	if len(created[0].Code) != 16 || len(created[1].Code) != 16 {
		t.Fatalf("expected 16-char codes, got %q and %q", created[0].Code, created[1].Code)
	}

	if created[0].CreatedByUsername != "admin" {
		t.Fatalf("expected created_by_username admin, got %q", created[0].CreatedByUsername)
	}

	regBody := map[string]string{
		"email":            "user1@example.com",
		"username":         "user1",
		"password":         "strong-password",
		"registration_key": created[0].Code,
	}
	regHeaders := map[string]string{"X-Forwarded-For": "203.0.113.7"}

	rec = doRequest(t, env.router, http.MethodPost, "/api/auth/register", regBody, regHeaders)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/admin/registration-keys", nil, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var listed []registrationKeyResp
	decodeJSON(t, rec, &listed)

	var found *registrationKeyResp
	for i := range listed {
		if listed[i].Code == created[0].Code {
			found = &listed[i]
			break
		}
	}

	if found == nil {
		t.Fatalf("expected key %s in list", created[0].Code)
	}

	if found.CreatedByUsername != "admin" {
		t.Fatalf("expected created_by_username admin, got %q", found.CreatedByUsername)
	}

	if found.UsedCount != 1 || len(found.Uses) != 1 {
		t.Fatalf("expected uses list, got %+v", found)
	}

	if found.Uses[0].UsedByUsername != "user1" {
		t.Fatalf("expected used_by_username user1, got %v", found.Uses[0].UsedByUsername)
	}

	if found.Uses[0].UsedByIP != "203.0.113.7" {
		t.Fatalf("expected used_by_ip 203.0.113.7, got %v", found.Uses[0].UsedByIP)
	}
}

func TestAdminMoveUserTeam(t *testing.T) {
	env := setupTest(t, testCfg)
	admin := ensureAdminUser(t, env)
	adminAccess, _, _ := loginUser(t, env.router, admin.Email, "adminpass")

	teamA := createTeam(t, env, "Alpha")
	teamB := createTeam(t, env, "Beta")
	key := createRegistrationKeyWithTeam(t, env, admin.ID, teamA.ID)

	rec := doRequest(t, env.router, http.MethodPost, "/api/auth/register", map[string]string{
		"email":            "user@example.com",
		"username":         "user1",
		"password":         "strong-password",
		"registration_key": key.Code,
	}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status %d: %s", rec.Code, rec.Body.String())
	}

	var regResp struct {
		ID int64 `json:"id"`
	}
	decodeJSON(t, rec, &regResp)

	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/users/"+itoa(regResp.ID)+"/team", map[string]int64{"team_id": teamB.ID}, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var adminResp struct {
		TeamID int64 `json:"team_id"`
	}
	decodeJSON(t, rec, &adminResp)

	if adminResp.TeamID != teamB.ID {
		t.Fatalf("expected team_id %d, got %d", teamB.ID, adminResp.TeamID)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/users/"+itoa(regResp.ID), nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var userResp struct {
		TeamID int64 `json:"team_id"`
	}
	decodeJSON(t, rec, &userResp)

	if userResp.TeamID != teamB.ID {
		t.Fatalf("expected user team_id %d, got %d", teamB.ID, userResp.TeamID)
	}
}

func TestAdminBlockUser(t *testing.T) {
	env := setupTest(t, testCfg)
	admin := ensureAdminUser(t, env)
	adminAccess, _, _ := loginUser(t, env.router, admin.Email, "adminpass")

	key := createRegistrationKey(t, env, admin.ID)
	regBody := map[string]string{
		"email":            "user@example.com",
		"username":         "user1",
		"password":         "strong-password",
		"registration_key": key.Code,
	}

	rec := doRequest(t, env.router, http.MethodPost, "/api/auth/register", regBody, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status %d: %s", rec.Code, rec.Body.String())
	}

	var regResp struct {
		ID int64 `json:"id"`
	}
	decodeJSON(t, rec, &regResp)

	access, _, _ := loginUser(t, env.router, regBody["email"], regBody["password"])

	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/users/"+itoa(regResp.ID)+"/block", map[string]string{
		"reason": "policy violation",
	}, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("block status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/me", nil, authHeader(access))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected ok, got %d: %s", rec.Code, rec.Body.String())
	}

	var meResp struct {
		Role          string  `json:"role"`
		BlockedReason *string `json:"blocked_reason"`
	}
	decodeJSON(t, rec, &meResp)

	if meResp.Role != "blocked" || meResp.BlockedReason == nil {
		t.Fatalf("expected blocked info, got %+v", meResp)
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/auth/login", map[string]string{
		"email":    regBody["email"],
		"password": regBody["password"],
	}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected ok, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminUnblockUser(t *testing.T) {
	env := setupTest(t, testCfg)
	admin := ensureAdminUser(t, env)
	adminAccess, _, _ := loginUser(t, env.router, admin.Email, "adminpass")

	key := createRegistrationKey(t, env, admin.ID)
	regBody := map[string]string{
		"email":            "user@example.com",
		"username":         "user1",
		"password":         "strong-password",
		"registration_key": key.Code,
	}

	rec := doRequest(t, env.router, http.MethodPost, "/api/auth/register", regBody, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status %d: %s", rec.Code, rec.Body.String())
	}

	var regResp struct {
		ID int64 `json:"id"`
	}
	decodeJSON(t, rec, &regResp)

	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/users/"+itoa(regResp.ID)+"/block", map[string]string{
		"reason": "policy violation",
	}, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("block status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/users/"+itoa(regResp.ID)+"/unblock", nil, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("unblock status %d: %s", rec.Code, rec.Body.String())
	}

	var userResp struct {
		Role          string  `json:"role"`
		BlockedReason *string `json:"blocked_reason"`
	}
	decodeJSON(t, rec, &userResp)
	if userResp.Role != "user" || userResp.BlockedReason != nil {
		t.Fatalf("expected unblocked user, got %+v", userResp)
	}

	access, _, _ := loginUser(t, env.router, regBody["email"], regBody["password"])

	rec = doRequest(t, env.router, http.MethodPut, "/api/me", map[string]string{"username": "newuser"}, authHeader(access))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected update ok, got %d: %s", rec.Code, rec.Body.String())
	}
}
