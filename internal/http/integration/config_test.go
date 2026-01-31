package http_test

import (
	"net/http"
	"testing"
)

type appConfigResp struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func TestConfigEndpoints(t *testing.T) {
	env := setupTest(t, testCfg)

	rec := doRequest(t, env.router, http.MethodGet, "/api/config", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var publicResp appConfigResp
	decodeJSON(t, rec, &publicResp)
	if publicResp.Title == "" || publicResp.Description == "" {
		t.Fatalf("expected default config, got %+v", publicResp)
	}

	_ = createUser(t, env, "admin@example.com", "admin", "adminpass", "admin")
	adminAccess, _, _ := loginUser(t, env.router, "admin@example.com", "adminpass")

	rec = doRequest(t, env.router, http.MethodPut, "/api/admin/config", map[string]string{
		"title":       "My CTF",
		"description": "Hello from API",
	}, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var adminResp appConfigResp
	decodeJSON(t, rec, &adminResp)
	if adminResp.Title != "My CTF" || adminResp.Description != "Hello from API" {
		t.Fatalf("unexpected admin config: %+v", adminResp)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/config", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	decodeJSON(t, rec, &publicResp)
	if publicResp.Title != "My CTF" || publicResp.Description != "Hello from API" {
		t.Fatalf("unexpected config after update: %+v", publicResp)
	}

	rec = doRequest(t, env.router, http.MethodPut, "/api/admin/config", map[string]string{"title": ""}, authHeader(adminAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
}
