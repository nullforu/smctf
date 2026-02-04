package http_test

import (
	"context"
	"net/http"
	"smctf/internal/service"
	"testing"
)

func TestListChallenges(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createChallenge(t, env, "Active 1", 100, "flag{1}", true)
	_ = createChallenge(t, env, "Inactive", 50, "flag{2}", false)
	_ = createChallenge(t, env, "Active 2", 200, "flag{3}", true)

	rec := doRequest(t, env.router, http.MethodGet, "/api/challenges", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp []map[string]any
	decodeJSON(t, rec, &resp)

	if len(resp) != 3 {
		t.Fatalf("expected 3 challenges, got %d", len(resp))
	}

	expectedTitles := []string{"Active 1", "Inactive", "Active 2"}
	expectedActive := []bool{true, false, true}
	expectedCategories := []string{"Misc", "Misc", "Misc"}

	for i, row := range resp {
		if row["title"] != expectedTitles[i] {
			t.Fatalf("expected title %q, got %q", expectedTitles[i], row["title"])
		}

		if row["category"] != expectedCategories[i] {
			t.Fatalf("expected category %q, got %q", expectedCategories[i], row["category"])
		}

		if isActive, ok := row["is_active"].(bool); !ok || isActive != expectedActive[i] {
			t.Fatalf("expected is_active to be %v for %q, got %v", expectedActive[i], row["title"], isActive)
		}
	}
}

func TestSubmitFlag(t *testing.T) {
	t.Run("missing auth", func(t *testing.T) {
		env := setupTest(t, testCfg)
		challenge := createChallenge(t, env, "Warmup", 100, "flag{ok}", true)

		rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{ok}"}, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		env := setupTest(t, testCfg)
		access, _, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")

		rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/abc/submit", map[string]string{"flag": "flag{ok}"}, authHeader(access))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		if resp.Error != service.ErrInvalidInput.Error() {
			t.Fatalf("unexpected error: %s", resp.Error)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		env := setupTest(t, testCfg)
		access, _, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")
		challenge := createChallenge(t, env, "Warmup", 100, "flag{ok}", true)

		rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{}, authHeader(access))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		assertFieldErrors(t, resp.Details, map[string]string{"flag": "required"})
	})

	t.Run("challenge not found", func(t *testing.T) {
		env := setupTest(t, testCfg)
		access, _, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")

		rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/999/submit", map[string]string{"flag": "flag{ok}"}, authHeader(access))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		if resp.Error != service.ErrChallengeNotFound.Error() {
			t.Fatalf("unexpected error: %s", resp.Error)
		}
	})

	t.Run("inactive challenge", func(t *testing.T) {
		env := setupTest(t, testCfg)
		access, _, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")
		challenge := createChallenge(t, env, "Warmup", 100, "flag{ok}", false)

		rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{ok}"}, authHeader(access))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("correct and wrong", func(t *testing.T) {
		env := setupTest(t, testCfg)
		access, _, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")
		challenge := createChallenge(t, env, "Warmup", 100, "flag{ok}", true)

		rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{nope}"}, authHeader(access))
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var wrongResp struct {
			Correct bool `json:"correct"`
		}
		decodeJSON(t, rec, &wrongResp)

		if wrongResp.Correct {
			t.Fatalf("expected incorrect flag")
		}

		rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{ok}"}, authHeader(access))
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var correctResp struct {
			Correct bool `json:"correct"`
		}
		decodeJSON(t, rec, &correctResp)

		if !correctResp.Correct {
			t.Fatalf("expected correct flag")
		}
	})

	t.Run("already solved", func(t *testing.T) {
		env := setupTest(t, testCfg)
		access, _, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")
		challenge := createChallenge(t, env, "Warmup", 100, "flag{ok}", true)

		rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{ok}"}, authHeader(access))
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{ok}"}, authHeader(access))
		if rec.Code != http.StatusConflict {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		if resp.Error != service.ErrAlreadySolved.Error() {
			t.Fatalf("unexpected error: %s", resp.Error)
		}
	})

	t.Run("team already solved", func(t *testing.T) {
		env := setupTest(t, testCfg)
		team := createTeam(t, env, "Alpha")
		user1 := createUserWithTeam(t, env, "u1@example.com", "u1", "pass", "user", team.ID)
		user2 := createUserWithTeam(t, env, "u2@example.com", "u2", "pass", "user", team.ID)
		access1, _, _ := loginUser(t, env.router, user1.Email, "pass")
		access2, _, _ := loginUser(t, env.router, user2.Email, "pass")
		challenge := createChallenge(t, env, "Warmup", 100, "flag{ok}", true)

		rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{ok}"}, authHeader(access1))
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{ok}"}, authHeader(access2))
		if rec.Code != http.StatusConflict {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		if resp.Error != service.ErrAlreadySolved.Error() {
			t.Fatalf("unexpected error: %s", resp.Error)
		}

		rec = doRequest(t, env.router, http.MethodGet, "/api/users/"+itoa(user2.ID)+"/solved", nil, authHeader(access2))
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var solvedPersonal []struct {
			ChallengeID int64 `json:"challenge_id"`
		}
		decodeJSON(t, rec, &solvedPersonal)

		if len(solvedPersonal) != 0 {
			t.Fatalf("expected personal solved list empty, got %+v", solvedPersonal)
		}

		rec = doRequest(t, env.router, http.MethodGet, "/api/teams/"+itoa(team.ID)+"/solved", nil, authHeader(access2))
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var solvedTeam []struct {
			ChallengeID int64 `json:"challenge_id"`
		}
		decodeJSON(t, rec, &solvedTeam)

		if len(solvedTeam) != 1 || solvedTeam[0].ChallengeID != challenge.ID {
			t.Fatalf("unexpected team solved list: %+v", solvedTeam)
		}
	})

	t.Run("rate limited", func(t *testing.T) {
		env := setupTest(t, testCfg)
		access, _, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")
		challenge := createChallenge(t, env, "Warmup", 100, "flag{ok}", true)

		for i := 0; i < env.cfg.Security.SubmissionMax; i++ {
			rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{nope}"}, authHeader(access))
			if rec.Code != http.StatusOK {
				t.Fatalf("status %d at attempt %d: %s", rec.Code, i+1, rec.Body.String())
			}
		}

		rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{nope}"}, authHeader(access))
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		if resp.Error != service.ErrRateLimited.Error() || resp.RateLimit == nil {
			t.Fatalf("unexpected rate limit response: %+v", resp)
		}

		if resp.RateLimit.Limit != env.cfg.Security.SubmissionMax || resp.RateLimit.Remaining != 0 {
			t.Fatalf("unexpected rate limit info: %+v", resp.RateLimit)
		}

		if rec.Header().Get("X-RateLimit-Limit") == "" || rec.Header().Get("X-RateLimit-Remaining") == "" || rec.Header().Get("X-RateLimit-Reset") == "" {
			t.Fatalf("missing rate limit headers")
		}
	})
}

func TestChallengesDynamicScoring(t *testing.T) {
	env := setupTest(t, testCfg)
	team := createTeam(t, env, "Alpha")
	userTeam := createUserWithTeam(t, env, "team@example.com", "team", "pass123", "user", team.ID)
	userSolo := createUser(t, env, "solo@example.com", "solo", "pass123", "user")

	challenge := createChallenge(t, env, "Dynamic", 500, "flag{dynamic}", true)
	challenge.MinimumPoints = 100
	if err := env.challengeRepo.Update(context.Background(), challenge); err != nil {
		t.Fatalf("update challenge: %v", err)
	}

	login := func(email, password string) string {
		rec := doRequest(t, env.router, http.MethodPost, "/api/auth/login", map[string]string{
			"email":    email,
			"password": password,
		}, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("login status %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			AccessToken string `json:"access_token"`
		}
		decodeJSON(t, rec, &resp)

		return resp.AccessToken
	}

	accessTeam := login(userTeam.Email, "pass123")
	accessSolo := login(userSolo.Email, "pass123")

	rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{dynamic}"}, authHeader(accessTeam))
	if rec.Code != http.StatusOK {
		t.Fatalf("team submit status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{dynamic}"}, authHeader(accessSolo))
	if rec.Code != http.StatusOK {
		t.Fatalf("solo submit status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status %d: %s", rec.Code, rec.Body.String())
	}

	var resp []map[string]any
	decodeJSON(t, rec, &resp)

	if len(resp) != 1 {
		t.Fatalf("expected 1 challenge, got %d", len(resp))
	}

	row := resp[0]
	if row["points"].(float64) != 100 {
		t.Fatalf("expected dynamic points 100, got %v", row["points"])
	}

	if row["solve_count"].(float64) != 2 {
		t.Fatalf("expected solve_count 2, got %v", row["solve_count"])
	}
}

func TestChallengeFileUploadDownloadDelete(t *testing.T) {
	env := setupTest(t, testCfg)
	admin := ensureAdminUser(t, env)
	access, _, _ := loginUser(t, env.router, admin.Email, "adminpass")

	challenge := createChallenge(t, env, "FileChallenge", 100, "flag{file}", true)

	uploadRec := doRequest(
		t,
		env.router,
		http.MethodPost,
		"/api/admin/challenges/"+itoa(challenge.ID)+"/file/upload",
		map[string]string{"filename": "bundle.zip"},
		authHeader(access),
	)
	if uploadRec.Code != http.StatusOK {
		t.Fatalf("upload status %d: %s", uploadRec.Code, uploadRec.Body.String())
	}

	var uploadResp struct {
		Challenge struct {
			ID       int64   `json:"id"`
			HasFile  bool    `json:"has_file"`
			FileName *string `json:"file_name"`
		} `json:"challenge"`
		Upload struct {
			URL    string            `json:"url"`
			Fields map[string]string `json:"fields"`
		} `json:"upload"`
	}

	decodeJSON(t, uploadRec, &uploadResp)
	if !uploadResp.Challenge.HasFile {
		t.Fatalf("expected has_file true")
	}

	if uploadResp.Challenge.FileName == nil || *uploadResp.Challenge.FileName != "bundle.zip" {
		t.Fatalf("expected file_name bundle.zip, got %v", uploadResp.Challenge.FileName)
	}

	if uploadResp.Upload.URL == "" || len(uploadResp.Upload.Fields) == 0 {
		t.Fatalf("expected upload payload")
	}

	downloadRec := doRequest(
		t,
		env.router,
		http.MethodPost,
		"/api/challenges/"+itoa(challenge.ID)+"/file/download",
		nil,
		authHeader(access),
	)
	if downloadRec.Code != http.StatusOK {
		t.Fatalf("download status %d: %s", downloadRec.Code, downloadRec.Body.String())
	}

	deleteRec := doRequest(
		t,
		env.router,
		http.MethodDelete,
		"/api/admin/challenges/"+itoa(challenge.ID)+"/file",
		nil,
		authHeader(access),
	)

	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete status %d: %s", deleteRec.Code, deleteRec.Body.String())
	}

	var deleteResp struct {
		HasFile bool `json:"has_file"`
	}
	decodeJSON(t, deleteRec, &deleteResp)

	if deleteResp.HasFile {
		t.Fatalf("expected has_file false after delete")
	}
}

func TestChallengeFileUploadRejectsNonZip(t *testing.T) {
	env := setupTest(t, testCfg)
	admin := ensureAdminUser(t, env)
	access, _, _ := loginUser(t, env.router, admin.Email, "adminpass")

	challenge := createChallenge(t, env, "FileChallenge", 100, "flag{file}", true)

	rec := doRequest(
		t,
		env.router,
		http.MethodPost,
		"/api/admin/challenges/"+itoa(challenge.ID)+"/file/upload",
		map[string]string{"filename": "bundle.txt"},
		authHeader(access),
	)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestChallengeFileDownloadMissing(t *testing.T) {
	env := setupTest(t, testCfg)
	admin := ensureAdminUser(t, env)
	access, _, _ := loginUser(t, env.router, admin.Email, "adminpass")

	challenge := createChallenge(t, env, "FileChallenge", 100, "flag{file}", true)

	rec := doRequest(
		t,
		env.router,
		http.MethodPost,
		"/api/challenges/"+itoa(challenge.ID)+"/file/download",
		nil,
		authHeader(access),
	)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}
