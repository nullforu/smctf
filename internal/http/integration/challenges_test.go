package http_test

import (
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

	var resp []map[string]interface{}
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
