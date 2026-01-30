package http_test

import (
	"net/http"
	"smctf/internal/models"
	"smctf/internal/service"
	"testing"
)

func TestRegister(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		env := setupTest(t, testCfg)
		admin := ensureAdminUser(t, env)
		key := createRegistrationKey(t, env, admin.ID)
		body := map[string]string{
			"email":            "user@example.com",
			"username":         "user1",
			"password":         "strong-password",
			"registration_key": key.Code,
		}

		rec := doRequest(t, env.router, http.MethodPost, "/api/auth/register", body, nil)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			ID       int64  `json:"id"`
			Email    string `json:"email"`
			Username string `json:"username"`
		}
		decodeJSON(t, rec, &resp)

		if resp.ID == 0 || resp.Email != body["email"] || resp.Username != body["username"] {
			t.Fatalf("unexpected response: %+v", resp)
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		env := setupTest(t, testCfg)
		rec := doRequest(t, env.router, http.MethodPost, "/api/auth/register", map[string]string{}, nil)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		if resp.Error != service.ErrInvalidInput.Error() {
			t.Fatalf("unexpected error: %s", resp.Error)
		}

		assertFieldErrors(t, resp.Details, map[string]string{
			"email":            "required",
			"username":         "required",
			"password":         "required",
			"registration_key": "required",
		})
	})

	t.Run("invalid key format", func(t *testing.T) {
		env := setupTest(t, testCfg)
		body := map[string]string{
			"email":            "user@example.com",
			"username":         "user1",
			"password":         "strong-password",
			"registration_key": "abc123",
		}

		rec := doRequest(t, env.router, http.MethodPost, "/api/auth/register", body, nil)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		assertFieldErrors(t, resp.Details, map[string]string{
			"registration_key": "invalid",
		})
	})

	t.Run("invalid key", func(t *testing.T) {
		env := setupTest(t, testCfg)
		body := map[string]string{
			"email":            "user@example.com",
			"username":         "user1",
			"password":         "strong-password",
			"registration_key": "123456",
		}

		rec := doRequest(t, env.router, http.MethodPost, "/api/auth/register", body, nil)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		assertFieldErrors(t, resp.Details, map[string]string{
			"registration_key": "invalid",
		})
	})

	t.Run("used key", func(t *testing.T) {
		env := setupTest(t, testCfg)
		admin := ensureAdminUser(t, env)
		key := createRegistrationKey(t, env, admin.ID)
		body := map[string]string{
			"email":            "user@example.com",
			"username":         "user1",
			"password":         "strong-password",
			"registration_key": key.Code,
		}

		rec := doRequest(t, env.router, http.MethodPost, "/api/auth/register", body, nil)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		rec = doRequest(t, env.router, http.MethodPost, "/api/auth/register", map[string]string{
			"email":            "user2@example.com",
			"username":         "user2",
			"password":         "strong-password",
			"registration_key": key.Code,
		}, nil)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		assertFieldErrors(t, resp.Details, map[string]string{
			"registration_key": "used",
		})
	})

	t.Run("duplicate", func(t *testing.T) {
		env := setupTest(t, testCfg)
		admin := ensureAdminUser(t, env)
		key := createRegistrationKey(t, env, admin.ID)
		body := map[string]string{
			"email":            "user@example.com",
			"username":         "user1",
			"password":         "strong-password",
			"registration_key": key.Code,
		}

		rec := doRequest(t, env.router, http.MethodPost, "/api/auth/register", body, nil)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		secondKey := createRegistrationKey(t, env, admin.ID)
		body["registration_key"] = secondKey.Code

		rec = doRequest(t, env.router, http.MethodPost, "/api/auth/register", body, nil)
		if rec.Code != http.StatusConflict {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		if resp.Error != service.ErrUserExists.Error() {
			t.Fatalf("unexpected error: %s", resp.Error)
		}
	})
}

func TestLogin(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		env := setupTest(t, testCfg)
		access, refresh, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")

		if access == "" || refresh == "" {
			t.Fatalf("tokens should not be empty")
		}
	})

	t.Run("invalid password", func(t *testing.T) {
		env := setupTest(t, testCfg)
		_, _, _ = registerAndLogin(t, env, "user@example.com", "user1", "strong-password")
		body := map[string]string{"email": "user@example.com", "password": "wrong"}

		rec := doRequest(t, env.router, http.MethodPost, "/api/auth/login", body, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		if resp.Error != service.ErrInvalidCreds.Error() {
			t.Fatalf("unexpected error: %s", resp.Error)
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		env := setupTest(t, testCfg)
		rec := doRequest(t, env.router, http.MethodPost, "/api/auth/login", map[string]string{"email": "user@example.com"}, nil)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		assertFieldErrors(t, resp.Details, map[string]string{
			"password": "required",
		})
	})
}

func TestRefreshAndLogout(t *testing.T) {
	env := setupTest(t, testCfg)
	_, refresh, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")

	rec := doRequest(t, env.router, http.MethodPost, "/api/auth/refresh", map[string]string{"refresh_token": refresh}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var refreshResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	decodeJSON(t, rec, &refreshResp)

	if refreshResp.AccessToken == "" || refreshResp.RefreshToken == "" {
		t.Fatalf("tokens should not be empty")
	}

	if refreshResp.RefreshToken == refresh {
		t.Fatalf("refresh token should rotate")
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/auth/refresh", map[string]string{"refresh_token": refresh}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/auth/logout", map[string]string{"refresh_token": refreshResp.RefreshToken}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/auth/refresh", map[string]string{"refresh_token": refreshResp.RefreshToken}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMe(t *testing.T) {
	env := setupTest(t, testCfg)
	access, refresh, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")

	rec := doRequest(t, env.router, http.MethodGet, "/api/me", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/me", nil, map[string]string{"Authorization": "Token " + access})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/me", nil, authHeader(refresh))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/me", nil, authHeader(access))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		ID       int64  `json:"id"`
		Email    string `json:"email"`
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	decodeJSON(t, rec, &resp)

	if resp.Email != "user@example.com" || resp.Username != "user1" || resp.Role != "user" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestUpdateMe(t *testing.T) {
	env := setupTest(t, testCfg)
	access, _, userID := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")

	rec := doRequest(t, env.router, http.MethodPut, "/api/me", map[string]string{"username": "newuser"}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPut, "/api/me", map[string]string{"username": "newuser"}, authHeader(access))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		ID       int64  `json:"id"`
		Email    string `json:"email"`
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	decodeJSON(t, rec, &resp)

	if resp.ID != userID || resp.Email != "user@example.com" || resp.Username != "newuser" || resp.Role != "user" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestMeSolved(t *testing.T) {
	env := setupTest(t, testCfg)
	access, _, userID := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")
	challenge := createChallenge(t, env, "Warmup", 100, "flag{ok}", true)

	rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{ok}"}, authHeader(access))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/users/"+itoa(userID)+"/solved", nil, authHeader(access))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var solved []models.SolvedChallenge
	decodeJSON(t, rec, &solved)

	if len(solved) != 1 {
		t.Fatalf("expected 1 solved, got %d", len(solved))
	}

	if solved[0].ChallengeID != challenge.ID || solved[0].Points != 100 || solved[0].Title != "Warmup" {
		t.Fatalf("unexpected solved entry: %+v", solved[0])
	}

	if solved[0].ChallengeID == 0 || solved[0].SolvedAt.IsZero() {
		t.Fatalf("expected solved timestamp and id, got %+v for user %d", solved[0], userID)
	}
}
