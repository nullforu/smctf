package discord

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBotClientGrantRoleSuccess(t *testing.T) {
	var gotPath, gotAuth string
	var body roleRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewBotClient(srv.URL, "secret", time.Second)
	if err := client.GrantRole(context.Background(), "123", "role-9"); err != nil {
		t.Fatalf("GrantRole: %v", err)
	}

	if gotPath != "/internal/guild/members/123/role" {
		t.Errorf("path = %q", gotPath)
	}

	if gotAuth != "Bearer secret" {
		t.Errorf("auth = %q", gotAuth)
	}

	if body.RoleID != "role-9" {
		t.Errorf("role id = %q", body.RoleID)
	}
}

func TestBotClientKickMember(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewBotClient(srv.URL, "secret", time.Second)
	if err := client.KickMember(context.Background(), "123"); err != nil {
		t.Fatalf("KickMember: %v", err)
	}

	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q", gotMethod)
	}

	if gotPath != "/internal/guild/members/123" {
		t.Errorf("path = %q", gotPath)
	}
}

func TestBotClientGrantRoleMapsCodes(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		code    string
		wantErr error
	}{
		{"not in guild via code", http.StatusNotFound, "NOT_IN_GUILD", ErrNotInGuild},
		{"permission via code", http.StatusForbidden, "BOT_PERMISSION", ErrBotPermission},
		{"rate limited via code", http.StatusTooManyRequests, "RATE_LIMITED", ErrRateLimited},
		{"not in guild via status", http.StatusNotFound, "", ErrNotInGuild},
		{"unavailable", http.StatusServiceUnavailable, "", ErrUnavailable},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				if tc.code != "" {
					_, _ = w.Write([]byte(`{"code":"` + tc.code + `","error":"boom"}`))
				}
			}))
			defer srv.Close()

			client := NewBotClient(srv.URL, "", time.Second)
			err := client.GrantRole(context.Background(), "123", "role-1")
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestBotClientJoinGuildSendsAccessToken(t *testing.T) {
	var body joinRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s", r.Method)
		}

		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewBotClient(srv.URL, "", time.Second)
	if err := client.JoinGuild(context.Background(), "u1", "tok-abc", "role-5"); err != nil {
		t.Fatalf("JoinGuild: %v", err)
	}

	if body.AccessToken != "tok-abc" {
		t.Errorf("access token = %q", body.AccessToken)
	}

	if body.RoleID != "role-5" {
		t.Errorf("role id = %q", body.RoleID)
	}
}

func TestBotClientGetMember(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"in_guild":true,"has_role":false}`))
	}))
	defer srv.Close()

	client := NewBotClient(srv.URL, "", time.Second)
	member, err := client.GetMember(context.Background(), "u1")
	if err != nil {
		t.Fatalf("GetMember: %v", err)
	}

	if !member.InGuild || member.HasRole {
		t.Errorf("member = %+v", member)
	}
}

func TestBotClientUnavailableOnDialError(t *testing.T) {
	client := NewBotClient("http://127.0.0.1:0", "", 200*time.Millisecond)
	err := client.GrantRole(context.Background(), "1", "role-1")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("got %v, want ErrUnavailable", err)
	}
}

func TestBotClientAnnounce(t *testing.T) {
	var gotMethod, gotPath string
	var body announceRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewBotClient(srv.URL, "secret", time.Second)
	if err := client.Announce(context.Background(), "chan-1", "hello world"); err != nil {
		t.Fatalf("Announce: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/internal/announce" {
		t.Errorf("method=%q path=%q", gotMethod, gotPath)
	}

	if body.Content != "hello world" {
		t.Errorf("content = %q", body.Content)
	}

	if body.ChannelID != "chan-1" {
		t.Errorf("channel id = %q", body.ChannelID)
	}
}

func TestBotClientSetNickname(t *testing.T) {
	var gotMethod, gotPath string
	var body nicknameRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewBotClient(srv.URL, "secret", time.Second)
	if err := client.SetNickname(context.Background(), "123", "Div_Team_neo"); err != nil {
		t.Fatalf("SetNickname: %v", err)
	}

	if gotMethod != http.MethodPatch || gotPath != "/internal/guild/members/123/nickname" {
		t.Errorf("method=%q path=%q", gotMethod, gotPath)
	}

	if body.Nickname != "Div_Team_neo" {
		t.Errorf("nickname = %q", body.Nickname)
	}
}
