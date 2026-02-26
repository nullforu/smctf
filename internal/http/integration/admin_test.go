package http_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"smctf/internal/config"
	"smctf/internal/models"
	"smctf/internal/stack"
	"smctf/internal/utils"
)

func TestAdminCreateChallenge(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)

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
	_ = createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)

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
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	updatedModel, err := env.challengeRepo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	expectedHash := utils.HMACFlag(env.cfg.Security.FlagHMACSecret, "flag{rotated}")
	if updatedModel.FlagHash != expectedHash {
		t.Fatalf("expected flag hash to be updated")
	}

	nullCases := []struct {
		name string
		body map[string]any
	}{
		{"title null", map[string]any{"title": nil}},
		{"description null", map[string]any{"description": nil}},
		{"category null", map[string]any{"category": nil}},
		{"flag null", map[string]any{"flag": nil}},
	}
	for _, tc := range nullCases {
		rec = doRequest(t, env.router, http.MethodPut, "/api/admin/challenges/"+itoa(created.ID), tc.body, authHeader(adminAccess))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s status %d: %s", tc.name, rec.Code, rec.Body.String())
		}
	}

	rec = doRequest(t, env.router, http.MethodPut, "/api/admin/challenges/"+itoa(created.ID), map[string]any{
		"title": "   ",
	}, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPut, "/api/admin/challenges/"+itoa(created.ID), map[string]any{
		"stack_enabled":      true,
		"stack_target_ports": []map[string]any{{"container_port": 80, "protocol": "TCP"}},
		"stack_pod_spec":     nil,
	}, authHeader(adminAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPut, "/api/admin/challenges/"+itoa(created.ID), map[string]any{
		"stack_enabled":  false,
		"stack_pod_spec": "   ",
	}, authHeader(adminAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPut, "/api/admin/challenges/"+itoa(created.ID), map[string]any{
		"stack_enabled":      true,
		"stack_target_ports": []map[string]any{{"container_port": 70000, "protocol": "TCP"}},
		"stack_pod_spec":     "apiVersion: v1\nkind: Pod\nmetadata:\n  name: challenge\nspec:\n  containers:\n    - name: app\n      image: nginx\n      ports:\n        - containerPort: 80\n",
	}, authHeader(adminAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPut, "/api/admin/challenges/"+itoa(created.ID), map[string]any{
		"points":         10,
		"minimum_points": 20,
	}, authHeader(adminAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminGetChallengeDetail(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)
	adminAccess, _, _ := loginUser(t, env.router, "admin@example.com", "adminpass")

	podSpec := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: challenge\nspec:\n  containers:\n    - name: app\n      image: nginx\n      ports:\n        - containerPort: 80\n"
	challenge := createChallenge(t, env, "Stacked", 100, "flag{stack}", true)
	challenge.StackEnabled = true
	challenge.StackTargetPorts = models.StackPortSpecs{{ContainerPort: 80, Protocol: "TCP"}}
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
	_ = createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)

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
	_ = createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)
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

	if created[0].CreatedByUsername != models.AdminRole {
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

	if found.CreatedByUsername != models.AdminRole {
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

	if meResp.Role != models.BlockedRole || meResp.BlockedReason == nil {
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
	if userResp.Role != models.UserRole || userResp.BlockedReason != nil {
		t.Fatalf("expected unblocked user, got %+v", userResp)
	}

	access, _, _ := loginUser(t, env.router, regBody["email"], regBody["password"])

	rec = doRequest(t, env.router, http.MethodPut, "/api/me", map[string]string{"username": "newuser"}, authHeader(access))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected update ok, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminStackManagement(t *testing.T) {
	cfg := testCfg
	cfg.Stack = config.StackConfig{
		Enabled:      true,
		MaxPerUser:   3,
		CreateWindow: time.Minute,
		CreateMax:    1,
	}

	mock := stack.NewProvisionerMock()
	env := setupStackTest(t, cfg, mock.Client())

	_ = createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)
	adminAccess, _, _ := loginUser(t, env.router, "admin@example.com", "adminpass")
	userAccess, _, _ := registerAndLogin(t, env, "user@example.com", models.UserRole, "strong-pass")
	challenge := createStackChallenge(t, env, "StackChal")

	rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/stack", nil, authHeader(userAccess))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create stack status %d: %s", rec.Code, rec.Body.String())
	}

	var created struct {
		StackID string `json:"stack_id"`
	}
	decodeJSON(t, rec, &created)

	rec = doRequest(t, env.router, http.MethodGet, "/api/admin/stacks", nil, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("admin list stacks status %d: %s", rec.Code, rec.Body.String())
	}

	var listResp struct {
		Stacks []struct {
			StackID string `json:"stack_id"`
		} `json:"stacks"`
	}
	decodeJSON(t, rec, &listResp)
	if len(listResp.Stacks) != 1 || listResp.Stacks[0].StackID != created.StackID {
		t.Fatalf("unexpected admin stacks response: %+v", listResp.Stacks)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/admin/stacks/"+created.StackID, nil, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("admin get stack status %d: %s", rec.Code, rec.Body.String())
	}

	var detailResp struct {
		StackID string `json:"stack_id"`
	}
	decodeJSON(t, rec, &detailResp)
	if detailResp.StackID != created.StackID {
		t.Fatalf("unexpected admin stack detail: %+v", detailResp)
	}

	rec = doRequest(t, env.router, http.MethodDelete, "/api/admin/stacks/"+created.StackID, nil, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("admin delete stack status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminStackEndpointsAuth(t *testing.T) {
	env := setupTest(t, testCfg)

	rec := doRequest(t, env.router, http.MethodGet, "/api/admin/stacks", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("admin stacks unauth status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/admin/stacks/stack-missing", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("admin stack detail unauth status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodDelete, "/api/admin/stacks/stack-missing", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("admin stack delete unauth status %d: %s", rec.Code, rec.Body.String())
	}

	accessUser, _, _ := registerAndLogin(t, env, "user@example.com", models.UserRole, "strong-pass")

	rec = doRequest(t, env.router, http.MethodGet, "/api/admin/stacks", nil, authHeader(accessUser))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("admin stacks forbidden status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/admin/stacks/stack-missing", nil, authHeader(accessUser))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("admin stack detail forbidden status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodDelete, "/api/admin/stacks/stack-missing", nil, authHeader(accessUser))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("admin stack delete forbidden status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminReportAuth(t *testing.T) {
	env := setupTest(t, testCfg)

	rec := doRequest(t, env.router, http.MethodGet, "/api/admin/report", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("admin report unauth status %d: %s", rec.Code, rec.Body.String())
	}

	accessUser, _, _ := registerAndLogin(t, env, "user@example.com", models.UserRole, "strong-pass")
	rec = doRequest(t, env.router, http.MethodGet, "/api/admin/report", nil, authHeader(accessUser))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("admin report forbidden status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminReportSuccess(t *testing.T) {
	cfg := testCfg
	cfg.Stack = config.StackConfig{
		Enabled:      true,
		MaxPerUser:   3,
		CreateWindow: time.Minute,
		CreateMax:    1,
	}

	mock := stack.NewProvisionerMock()
	env := setupStackTest(t, cfg, mock.Client())

	_ = createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)
	adminAccess, _, _ := loginUser(t, env.router, "admin@example.com", "adminpass")
	userAccess, _, _ := registerAndLogin(t, env, "user@example.com", models.UserRole, "strong-pass")
	challenge := createStackChallenge(t, env, "StackChal")

	rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/stack", nil, authHeader(userAccess))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create stack status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{stack}"}, authHeader(userAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("submit flag status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/admin/report", nil, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("admin report status %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec, &resp)
	if _, ok := resp["challenges"]; !ok {
		t.Fatalf("expected challenges in report")
	}

	if _, ok := resp["users"]; !ok {
		t.Fatalf("expected users in report")
	}

	if _, ok := resp["stacks"]; !ok {
		t.Fatalf("expected stacks in report")
	}

	if _, ok := resp["leaderboard"]; !ok {
		t.Fatalf("expected leaderboard in report")
	}

	if _, ok := resp["timeline"]; !ok {
		t.Fatalf("expected timeline in report")
	}
}
