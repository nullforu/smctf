package http_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"smctf/internal/config"
	"smctf/internal/models"
	"smctf/internal/utils"
	"smctf/internal/vm"
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

	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/challenges", map[string]any{
		"title":       "Ch4",
		"description": "desc",
		"category":    "Web",
		"points":      100,
		"flag":        strings.Repeat("a", 73),
		"is_active":   true,
	}, authHeader(adminAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	decodeJSON(t, rec, &resp)
	assertFieldErrors(t, resp.Details, map[string]string{"flag": "max bytes is 72"})
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

	ok, err := utils.CheckFlag(updatedModel.FlagHash, "flag{rotated}")
	if err != nil || !ok {
		t.Fatalf("expected flag hash to be updated")
	}

	rec = doRequest(t, env.router, http.MethodPut, "/api/admin/challenges/"+itoa(created.ID), map[string]any{
		"flag": strings.Repeat("a", 73),
	}, authHeader(adminAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	decodeJSON(t, rec, &errResp)
	assertFieldErrors(t, errResp.Details, map[string]string{"flag": "max bytes is 72"})

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
		"vm_enabled": true,
		"vm_spec":    nil,
	}, authHeader(adminAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPut, "/api/admin/challenges/"+itoa(created.ID), map[string]any{
		"vm_enabled": false,
		"vm_spec":    "   ",
	}, authHeader(adminAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPut, "/api/admin/challenges/"+itoa(created.ID), map[string]any{
		"vm_enabled": true,
		"vm_spec":    "",
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

	sandboxSpec := "apiVersion: v1\nkind: Sandbox\nmetadata:\n  name: challenge\nspec:\n  containers:\n    - name: app\n      image: nginx\n      ports:\n        - containerPort: 80\n"
	rec = doRequest(t, env.router, http.MethodPut, "/api/admin/challenges/"+itoa(created.ID), map[string]any{
		"vm_enabled": true,
		"vm_spec":    sandboxSpec,
	}, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	model, err := env.challengeRepo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if !model.VMEnabled {
		t.Fatalf("expected vm_enabled to be true")
	}

	if model.VMSpec == nil || strings.TrimSpace(*model.VMSpec) != strings.TrimSpace(sandboxSpec) {
		t.Fatalf("expected vm_spec to be persisted")
	}

	rec = doRequest(t, env.router, http.MethodPut, "/api/admin/challenges/"+itoa(created.ID), map[string]any{
		"vm_enabled": false,
	}, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	model, err = env.challengeRepo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID after disable: %v", err)
	}

	if model.VMEnabled {
		t.Fatalf("expected vm_enabled to be false")
	}

	if model.VMSpec != nil {
		t.Fatalf("expected vm_spec to be cleared when vm is disabled")
	}
}

func TestAdminGetChallengeDetail(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)
	adminAccess, _, _ := loginUser(t, env.router, "admin@example.com", "adminpass")

	sandboxSpec := "apiVersion: v1\nkind: Sandbox\nmetadata:\n  name: challenge\nspec:\n  containers:\n    - name: app\n      image: nginx\n      ports:\n        - containerPort: 80\n"
	challenge := createChallenge(t, env, "VMed", 100, "flag{vm}", true)
	challenge.VMEnabled = true
	challenge.VMSpec = &sandboxSpec
	if err := env.challengeRepo.Update(context.Background(), challenge); err != nil {
		t.Fatalf("update challenge: %v", err)
	}

	rec := doRequest(t, env.router, http.MethodGet, "/api/admin/challenges/"+itoa(challenge.ID), nil, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec, &resp)
	if resp["vm_spec"] == nil {
		t.Fatalf("expected vm_spec")
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

	rec = doRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/challenges?division_id=%d", env.defaultDivisionID), nil, nil)
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

func TestAdminExportChallenges(t *testing.T) {
	env := setupTest(t, testCfg)
	admin := ensureAdminUser(t, env)
	adminAccess, _, _ := loginUser(t, env.router, admin.Email, "adminpass")

	first := createChallenge(t, env, "First", 100, "flag{first}", true)
	second := createChallenge(t, env, "Second", 200, "flag{second}", false)
	fileKey := "challenge-bundle.zip"
	fileName := "challenge.zip"
	fileUploadedAt := time.Now().UTC().Add(-time.Minute)
	second.PreviousChallengeID = &first.ID
	second.FileKey = &fileKey
	second.FileName = &fileName
	second.FileUploadedAt = &fileUploadedAt
	if err := env.challengeRepo.Update(context.Background(), second); err != nil {
		t.Fatalf("update second: %v", err)
	}

	rec := doRequest(t, env.router, http.MethodGet, "/api/admin/challenges/export", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	accessUser, _, _ := registerAndLogin(t, env, "export-user@example.com", "exportuser", "strong-password")
	rec = doRequest(t, env.router, http.MethodGet, "/api/admin/challenges/export", nil, authHeader(accessUser))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/admin/challenges/export", nil, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var exported struct {
		Version    int `json:"version"`
		Challenges []struct {
			ID                  int64   `json:"id"`
			Title               string  `json:"title"`
			FlagHash            string  `json:"flag_hash"`
			FileKey             *string `json:"file_key"`
			PreviousChallengeID *int64  `json:"previous_challenge_id"`
		} `json:"challenges"`
		RequestedIDs []int64 `json:"requested_ids"`
	}
	decodeJSON(t, rec, &exported)

	if exported.Version != 1 {
		t.Fatalf("expected version 1, got %d", exported.Version)
	}
	if len(exported.RequestedIDs) != 0 {
		t.Fatalf("expected empty requested_ids, got %v", exported.RequestedIDs)
	}
	if len(exported.Challenges) != 2 {
		t.Fatalf("expected 2 challenges, got %d", len(exported.Challenges))
	}
	if exported.Challenges[0].FlagHash == "" || exported.Challenges[1].FlagHash == "" {
		t.Fatalf("expected exported flag hashes")
	}
	if exported.Challenges[1].FileKey == nil || *exported.Challenges[1].FileKey != fileKey {
		t.Fatalf("expected file key %q, got %v", fileKey, exported.Challenges[1].FileKey)
	}
	if exported.Challenges[1].PreviousChallengeID == nil || *exported.Challenges[1].PreviousChallengeID != first.ID {
		t.Fatalf("expected previous challenge id %d, got %v", first.ID, exported.Challenges[1].PreviousChallengeID)
	}

	rec = doRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/admin/challenges/export?ids=%d", second.ID), nil, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	decodeJSON(t, rec, &exported)
	if len(exported.Challenges) != 1 || exported.Challenges[0].ID != second.ID {
		t.Fatalf("expected selected export for challenge %d, got %+v", second.ID, exported.Challenges)
	}
	if len(exported.RequestedIDs) != 1 || exported.RequestedIDs[0] != second.ID {
		t.Fatalf("unexpected requested ids: %v", exported.RequestedIDs)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/admin/challenges/export?ids=abc", nil, authHeader(adminAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/admin/challenges/export?ids=999999", nil, authHeader(adminAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown id, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminImportChallenges(t *testing.T) {
	env := setupTest(t, testCfg)
	admin := ensureAdminUser(t, env)
	adminAccess, _, _ := loginUser(t, env.router, admin.Email, "adminpass")

	originalA := createChallenge(t, env, "Import A", 123, "flag{import-a}", true)
	originalB := createChallenge(t, env, "Import B", 456, "flag{import-b}", false)
	vmSpec := "apiVersion: v1\nkind: Sandbox\nmetadata:\n  name: import-b\nspec:\n  containers:\n    - name: app\n      image: nginx\n"
	fileKey := "imports/import-b.zip"
	fileName := "import-b.zip"
	fileUploadedAt := time.Now().UTC().Add(-2 * time.Minute)
	originalB.PreviousChallengeID = &originalA.ID
	originalB.VMEnabled = true
	originalB.VMSpec = &vmSpec
	originalB.FileKey = &fileKey
	originalB.FileName = &fileName
	originalB.FileUploadedAt = &fileUploadedAt
	if err := env.challengeRepo.Update(context.Background(), originalB); err != nil {
		t.Fatalf("update originalB: %v", err)
	}

	rec := doRequest(t, env.router, http.MethodGet, "/api/admin/challenges/export", nil, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("export status %d: %s", rec.Code, rec.Body.String())
	}

	var bundle map[string]any
	decodeJSON(t, rec, &bundle)

	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/challenges/import", bundle, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	accessUser, _, _ := registerAndLogin(t, env, "import-user@example.com", "importuser", "strong-password")
	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/challenges/import", bundle, authHeader(accessUser))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/challenges/import", bundle, authHeader(adminAccess))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var importedResp struct {
		Imported []struct {
			ID                  int64  `json:"id"`
			Title               string `json:"title"`
			PreviousChallengeID *int64 `json:"previous_challenge_id"`
			VMEnabled           bool   `json:"vm_enabled"`
		} `json:"imported"`
	}
	decodeJSON(t, rec, &importedResp)

	if len(importedResp.Imported) != 2 {
		t.Fatalf("expected 2 imported challenges, got %d", len(importedResp.Imported))
	}
	if importedResp.Imported[0].ID == originalA.ID || importedResp.Imported[1].ID == originalB.ID {
		t.Fatalf("expected imported challenges to get new ids: %+v", importedResp.Imported)
	}
	if importedResp.Imported[1].PreviousChallengeID == nil || *importedResp.Imported[1].PreviousChallengeID != importedResp.Imported[0].ID {
		t.Fatalf("expected remapped previous challenge id, got %+v", importedResp.Imported)
	}
	if !importedResp.Imported[1].VMEnabled {
		t.Fatalf("expected vm_enabled to be preserved")
	}

	importedA, err := env.challengeRepo.GetByID(context.Background(), importedResp.Imported[0].ID)
	if err != nil {
		t.Fatalf("get importedA: %v", err)
	}
	importedB, err := env.challengeRepo.GetByID(context.Background(), importedResp.Imported[1].ID)
	if err != nil {
		t.Fatalf("get importedB: %v", err)
	}

	ok, err := utils.CheckFlag(importedA.FlagHash, "flag{import-a}")
	if err != nil || !ok {
		t.Fatalf("expected importedA hash to be reusable")
	}
	ok, err = utils.CheckFlag(importedB.FlagHash, "flag{import-b}")
	if err != nil || !ok {
		t.Fatalf("expected importedB hash to be reusable")
	}
	if importedB.PreviousChallengeID == nil || *importedB.PreviousChallengeID != importedA.ID {
		t.Fatalf("expected importedB to point at importedA, got %v", importedB.PreviousChallengeID)
	}
	if importedB.FileKey == nil || *importedB.FileKey != fileKey {
		t.Fatalf("expected importedB file key %q, got %v", fileKey, importedB.FileKey)
	}
	if importedB.FileName == nil || *importedB.FileName != fileName {
		t.Fatalf("expected importedB file name %q, got %v", fileName, importedB.FileName)
	}
	if importedB.FileUploadedAt == nil || !importedB.FileUploadedAt.Equal(fileUploadedAt) {
		t.Fatalf("expected importedB file uploaded at %v, got %v", fileUploadedAt, importedB.FileUploadedAt)
	}

	badBundle := map[string]any{
		"version": 1,
		"challenges": []map[string]any{
			{
				"id":             50,
				"title":          "Broken",
				"description":    "desc",
				"category":       "Web",
				"points":         100,
				"minimum_points": 50,
				"flag_hash":      "not-bcrypt",
				"is_active":      true,
				"vm_enabled":     false,
				"created_at":     time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/challenges/import", badBundle, authHeader(adminAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp errorResp
	decodeJSON(t, rec, &errResp)
	assertFieldErrors(t, errResp.Details, map[string]string{"challenges[0].flag_hash": "invalid bcrypt hash"})
}

func TestAdminExportImportDivisions(t *testing.T) {
	env := setupTest(t, testCfg)
	admin := ensureAdminUser(t, env)
	adminAccess, _, _ := loginUser(t, env.router, admin.Email, "adminpass")

	roleID := "123456789012345678"
	channelID := "987654321098765432"
	division := &models.Division{
		Name:                     "Blue",
		DiscordRoleID:            &roleID,
		DiscordAnnounceChannelID: &channelID,
		CreatedAt:                time.Now().UTC().Add(-time.Hour),
	}
	if err := env.divisionRepo.Create(context.Background(), division); err != nil {
		t.Fatalf("create division: %v", err)
	}

	rec := doRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/admin/divisions/export?ids=%d", division.ID), nil, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("export status %d: %s", rec.Code, rec.Body.String())
	}

	var exported struct {
		Version   int `json:"version"`
		Divisions []struct {
			ID                       int64   `json:"id"`
			Name                     string  `json:"name"`
			DiscordRoleID            *string `json:"discord_role_id"`
			DiscordAnnounceChannelID *string `json:"discord_announce_channel_id"`
		} `json:"divisions"`
		RequestedIDs []int64 `json:"requested_ids"`
	}
	decodeJSON(t, rec, &exported)

	if len(exported.Divisions) != 1 || exported.Divisions[0].ID != division.ID {
		t.Fatalf("unexpected exported divisions: %+v", exported.Divisions)
	}
	if exported.Divisions[0].DiscordRoleID == nil || *exported.Divisions[0].DiscordRoleID != roleID {
		t.Fatalf("expected role id %q, got %v", roleID, exported.Divisions[0].DiscordRoleID)
	}

	importBundle := map[string]any{
		"version": 1,
		"divisions": []map[string]any{
			{
				"id":                          777,
				"name":                        "Green",
				"discord_role_id":             roleID,
				"discord_announce_channel_id": channelID,
				"created_at":                  time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/divisions/import", importBundle, authHeader(adminAccess))
	if rec.Code != http.StatusCreated {
		t.Fatalf("import status %d: %s", rec.Code, rec.Body.String())
	}

	var imported struct {
		Imported []models.Division `json:"imported"`
	}
	decodeJSON(t, rec, &imported)
	if len(imported.Imported) != 1 || imported.Imported[0].Name != "Green" {
		t.Fatalf("unexpected imported divisions: %+v", imported.Imported)
	}
	if imported.Imported[0].ID == 777 {
		t.Fatalf("expected new division id, got source id %d", imported.Imported[0].ID)
	}
}

func TestAdminExportImportTeams(t *testing.T) {
	env := setupTest(t, testCfg)
	admin := ensureAdminUser(t, env)
	adminAccess, _, _ := loginUser(t, env.router, admin.Email, "adminpass")

	division := &models.Division{Name: "Special", CreatedAt: time.Now().UTC().Add(-time.Hour)}
	if err := env.divisionRepo.Create(context.Background(), division); err != nil {
		t.Fatalf("create division: %v", err)
	}

	team := &models.Team{Name: "Alpha", DivisionID: division.ID, CreatedAt: time.Now().UTC().Add(-30 * time.Minute)}
	if err := env.teamRepo.Create(context.Background(), team); err != nil {
		t.Fatalf("create team: %v", err)
	}

	rec := doRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/admin/teams/export?ids=%d", team.ID), nil, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("export status %d: %s", rec.Code, rec.Body.String())
	}

	var exported struct {
		Version int `json:"version"`
		Teams   []struct {
			ID           int64  `json:"id"`
			Name         string `json:"name"`
			DivisionName string `json:"division_name"`
		} `json:"teams"`
	}
	decodeJSON(t, rec, &exported)
	if len(exported.Teams) != 1 || exported.Teams[0].DivisionName != division.Name {
		t.Fatalf("unexpected exported teams: %+v", exported.Teams)
	}

	importBundle := map[string]any{
		"version": 1,
		"teams": []map[string]any{
			{
				"id":            999,
				"name":          "Beta Team With Long Imported Name",
				"division_name": division.Name,
				"created_at":    time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/teams/import", importBundle, authHeader(adminAccess))
	if rec.Code != http.StatusCreated {
		t.Fatalf("import status %d: %s", rec.Code, rec.Body.String())
	}

	var imported struct {
		Imported []struct {
			ID         int64  `json:"id"`
			Name       string `json:"name"`
			DivisionID int64  `json:"division_id"`
		} `json:"imported"`
	}
	decodeJSON(t, rec, &imported)
	if len(imported.Imported) != 1 || imported.Imported[0].Name != "Beta Team With Long Imported Name" {
		t.Fatalf("unexpected imported teams: %+v", imported.Imported)
	}
	if imported.Imported[0].DivisionID != division.ID {
		t.Fatalf("expected imported team division_id %d, got %d", division.ID, imported.Imported[0].DivisionID)
	}

	missingDivisionBundle := map[string]any{
		"version": 1,
		"teams": []map[string]any{
			{
				"id":            1000,
				"name":          "Gamma",
				"division_name": "MissingDivision",
				"created_at":    time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/teams/import", missingDivisionBundle, authHeader(adminAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp errorResp
	decodeJSON(t, rec, &errResp)
	assertFieldErrors(t, errResp.Details, map[string]string{"teams[0].division_name": "not found"})
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

func TestAdminExportImportRegistrationKeys(t *testing.T) {
	env := setupTest(t, testCfg)
	admin := ensureAdminUser(t, env)
	adminAccess, _, _ := loginUser(t, env.router, admin.Email, "adminpass")

	team := createTeam(t, env, fmt.Sprintf("ExportTeam-%d", time.Now().UnixNano()))
	key := &models.RegistrationKey{
		Code:      "ABCDEFGHJKLMNPQ2",
		CreatedBy: admin.ID,
		TeamID:    team.ID,
		MaxUses:   3,
		UsedCount: 1,
		CreatedAt: time.Now().UTC().Add(-time.Hour),
	}
	if err := env.regKeyRepo.Create(context.Background(), key); err != nil {
		t.Fatalf("create key: %v", err)
	}

	rec := doRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/admin/registration-keys/export?ids=%d", key.ID), nil, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("export status %d: %s", rec.Code, rec.Body.String())
	}

	var exported struct {
		Version int `json:"version"`
		Keys    []struct {
			ID       int64  `json:"id"`
			Code     string `json:"code"`
			TeamName string `json:"team_name"`
			MaxUses  int    `json:"max_uses"`
		} `json:"registration_keys"`
	}
	decodeJSON(t, rec, &exported)
	if len(exported.Keys) != 1 || exported.Keys[0].Code != key.Code || exported.Keys[0].TeamName != team.Name || exported.Keys[0].MaxUses != key.MaxUses {
		t.Fatalf("unexpected exported keys: %+v", exported.Keys)
	}

	importTeam := createTeam(t, env, fmt.Sprintf("ImportTeam-%d", time.Now().UnixNano()))
	importBundle := map[string]any{
		"version": 1,
		"registration_keys": []map[string]any{
			{
				"id":         999,
				"code":       "ABCDEFGHJKLMNPQ3",
				"team_name":  importTeam.Name,
				"max_uses":   2,
				"created_at": time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/registration-keys/import", importBundle, authHeader(adminAccess))
	if rec.Code != http.StatusCreated {
		t.Fatalf("import status %d: %s", rec.Code, rec.Body.String())
	}

	var imported struct {
		Imported []struct {
			ID        int64  `json:"id"`
			Code      string `json:"code"`
			TeamID    int64  `json:"team_id"`
			TeamName  string `json:"team_name"`
			MaxUses   int    `json:"max_uses"`
			UsedCount int    `json:"used_count"`
		} `json:"imported"`
	}
	decodeJSON(t, rec, &imported)
	if len(imported.Imported) != 1 || imported.Imported[0].Code != "ABCDEFGHJKLMNPQ3" || imported.Imported[0].TeamID != importTeam.ID || imported.Imported[0].TeamName != importTeam.Name || imported.Imported[0].MaxUses != 2 || imported.Imported[0].UsedCount != 0 {
		t.Fatalf("unexpected imported keys: %+v", imported.Imported)
	}

	missingTeamBundle := map[string]any{
		"version": 1,
		"registration_keys": []map[string]any{
			{
				"id":         1000,
				"code":       "ABCDEFGHJKLMNPQ4",
				"team_name":  "Missing Team",
				"max_uses":   1,
				"created_at": time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/registration-keys/import", missingTeamBundle, authHeader(adminAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp errorResp
	decodeJSON(t, rec, &errResp)
	assertFieldErrors(t, errResp.Details, map[string]string{"registration_keys[0].team_name": "not found"})
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

func TestAdminVMManagement(t *testing.T) {
	cfg := testCfg
	cfg.VM = config.VMConfig{
		Enabled:      true,
		MaxPer:       3,
		CreateWindow: time.Minute,
		CreateMax:    1,
	}

	mock := vm.NewOrchestratorMock()
	env := setupVMTest(t, cfg, mock.Client())

	_ = createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)
	adminAccess, _, _ := loginUser(t, env.router, "admin@example.com", "adminpass")
	userAccess, _, _ := registerAndLogin(t, env, "user@example.com", models.UserRole, "strong-pass")
	challenge := createVMChallenge(t, env, "VMChal")

	rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/vm", nil, authHeader(userAccess))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create vm status %d: %s", rec.Code, rec.Body.String())
	}

	var created struct {
		VMID string `json:"vm_id"`
	}
	decodeJSON(t, rec, &created)

	rec = doRequest(t, env.router, http.MethodGet, "/api/admin/vms", nil, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("admin list vms status %d: %s", rec.Code, rec.Body.String())
	}

	var listResp struct {
		VMs []struct {
			VMID string `json:"vm_id"`
		} `json:"vms"`
	}
	decodeJSON(t, rec, &listResp)
	if len(listResp.VMs) != 1 || listResp.VMs[0].VMID != created.VMID {
		t.Fatalf("unexpected admin vms response: %+v", listResp.VMs)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/admin/vms/"+created.VMID, nil, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("admin get vm status %d: %s", rec.Code, rec.Body.String())
	}

	var detailResp struct {
		VMID string `json:"vm_id"`
	}
	decodeJSON(t, rec, &detailResp)
	if detailResp.VMID != created.VMID {
		t.Fatalf("unexpected admin vm detail: %+v", detailResp)
	}

	rec = doRequest(t, env.router, http.MethodDelete, "/api/admin/vms/"+created.VMID, nil, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("admin delete vm status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminVMEndpointsAuth(t *testing.T) {
	env := setupTest(t, testCfg)

	rec := doRequest(t, env.router, http.MethodGet, "/api/admin/vms", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("admin vms unauth status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/admin/vms/vm-missing", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("admin vm detail unauth status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodDelete, "/api/admin/vms/vm-missing", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("admin vm delete unauth status %d: %s", rec.Code, rec.Body.String())
	}

	accessUser, _, _ := registerAndLogin(t, env, "user@example.com", models.UserRole, "strong-pass")

	rec = doRequest(t, env.router, http.MethodGet, "/api/admin/vms", nil, authHeader(accessUser))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("admin vms forbidden status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/admin/vms/vm-missing", nil, authHeader(accessUser))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("admin vm detail forbidden status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodDelete, "/api/admin/vms/vm-missing", nil, authHeader(accessUser))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("admin vm delete forbidden status %d: %s", rec.Code, rec.Body.String())
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
	cfg.VM = config.VMConfig{
		Enabled:      true,
		MaxPer:       3,
		CreateWindow: time.Minute,
		CreateMax:    1,
	}

	mock := vm.NewOrchestratorMock()
	env := setupVMTest(t, cfg, mock.Client())

	_ = createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)
	adminAccess, _, _ := loginUser(t, env.router, "admin@example.com", "adminpass")
	userAccess, _, _ := registerAndLogin(t, env, "user@example.com", models.UserRole, "strong-pass")
	challenge := createVMChallenge(t, env, "VMChal")

	rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/vm", nil, authHeader(userAccess))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create vm status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{vm}"}, authHeader(userAccess))
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

	if _, ok := resp["vms"]; !ok {
		t.Fatalf("expected vms in report")
	}

	if _, ok := resp["leaderboard"]; !ok {
		t.Fatalf("expected leaderboard in report")
	}

	if _, ok := resp["timeline"]; !ok {
		t.Fatalf("expected timeline in report")
	}
}
