package http_test

import (
	"context"
	"net/http"
	"testing"
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

	var challenges []map[string]any
	decodeJSON(t, rec, &challenges)

	if len(challenges) != 0 {
		t.Fatalf("expected 0 challenges, got %d", len(challenges))
	}
}

func TestAdminRegistrationKeys(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createUser(t, env, "admin@example.com", "admin", "adminpass", "admin")
	team := createTeam(t, env, "Alpha")

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

	if len(created[0].Code) != 6 || len(created[1].Code) != 6 {
		t.Fatalf("expected 6-digit codes, got %q and %q", created[0].Code, created[1].Code)
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

	if found.UsedByUsername == nil || *found.UsedByUsername != "user1" {
		t.Fatalf("expected used_by_username user1, got %v", found.UsedByUsername)
	}

	if found.UsedByIP == nil || *found.UsedByIP != "203.0.113.7" {
		t.Fatalf("expected used_by_ip 203.0.113.7, got %v", found.UsedByIP)
	}
}
