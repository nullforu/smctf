package discord

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func newTestOAuthClient(serverURL string) *OAuthClient {
	c := NewOAuthClient(OAuthConfig{
		ClientID:     "cid",
		ClientSecret: "csecret",
		RedirectURI:  "https://example.com/api/discord/callback",
		Scopes:       "identify guilds.join",
	}, time.Second)
	c.httpClient = &http.Client{Transport: rewriteTransport{base: serverURL}}
	return c
}

type rewriteTransport struct {
	base string
}

func (t rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	target, _ := url.Parse(t.base)
	req.URL.Scheme = target.Scheme
	req.URL.Host = target.Host
	return http.DefaultTransport.RoundTrip(req)
}

func TestAuthorizeURLContainsParams(t *testing.T) {
	c := NewOAuthClient(OAuthConfig{
		ClientID:    "cid",
		RedirectURI: "https://example.com/cb",
		Scopes:      "identify guilds.join",
	}, time.Second)

	got := c.AuthorizeURL("xyz-state")
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	q := parsed.Query()
	if q.Get("client_id") != "cid" {
		t.Errorf("client_id = %q", q.Get("client_id"))
	}

	if q.Get("state") != "xyz-state" {
		t.Errorf("state = %q", q.Get("state"))
	}

	if q.Get("scope") != "identify guilds.join" {
		t.Errorf("scope = %q", q.Get("scope"))
	}

	if q.Get("response_type") != "code" {
		t.Errorf("response_type = %q", q.Get("response_type"))
	}
}

func TestExchangeCodeSendsFormEncoded(t *testing.T) {
	var contentType, code, grantType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType = r.Header.Get("Content-Type")
		_ = r.ParseForm()
		code = r.Form.Get("code")
		grantType = r.Form.Get("grant_type")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at-123","token_type":"Bearer","expires_in":604800,"scope":"identify"}`))
	}))
	defer srv.Close()

	c := newTestOAuthClient(srv.URL)
	token, err := c.ExchangeCode(context.Background(), "the-code")
	if err != nil {
		t.Fatalf("ExchangeCode: %v", err)
	}

	if token.AccessToken != "at-123" {
		t.Errorf("access_token = %q", token.AccessToken)
	}

	if !strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		t.Errorf("content-type = %q", contentType)
	}

	if code != "the-code" || grantType != "authorization_code" {
		t.Errorf("form code=%q grant=%q", code, grantType)
	}
}

func TestFetchUserUsesBearer(t *testing.T) {
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"42","username":"neo","global_name":"Neo"}`))
	}))
	defer srv.Close()

	c := newTestOAuthClient(srv.URL)
	user, err := c.FetchUser(context.Background(), "at-123")
	if err != nil {
		t.Fatalf("FetchUser: %v", err)
	}

	if user.ID != "42" || user.Username != "neo" {
		t.Errorf("user = %+v", user)
	}

	if auth != "Bearer at-123" {
		t.Errorf("auth = %q", auth)
	}
}

func TestExchangeCodeErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`invalid_grant`))
	}))
	defer srv.Close()

	c := newTestOAuthClient(srv.URL)
	if _, err := c.ExchangeCode(context.Background(), "bad"); err == nil {
		t.Fatal("expected error")
	}
}
