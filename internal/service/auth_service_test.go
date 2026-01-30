package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"smctf/internal/auth"
	"smctf/internal/models"

	"github.com/redis/go-redis/v9"
)

func TestAuthServiceRegisterSuccess(t *testing.T) {
	env := setupServiceTest(t)
	admin := createUser(t, env, "admin@example.com", "admin", "pass", "admin")
	key := createRegistrationKey(t, env, "123456", admin.ID)

	user, err := env.authSvc.Register(context.Background(), "USER@Example.com", "  user1  ", "pass1", key.Code, "127.0.0.1")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if user.ID == 0 || user.Email != "user@example.com" || user.Username != "user1" {
		t.Fatalf("unexpected user: %+v", user)
	}

	stored, err := env.regKeyRepo.GetByCodeForUpdate(context.Background(), env.db, key.Code)
	if err != nil {
		t.Fatalf("fetch key: %v", err)
	}

	if stored.UsedBy == nil || *stored.UsedBy != user.ID {
		t.Fatalf("expected used_by to be set, got %+v", stored.UsedBy)
	}

	if stored.UsedByIP == nil || *stored.UsedByIP != "127.0.0.1" {
		t.Fatalf("expected used_by_ip to be set, got %+v", stored.UsedByIP)
	}

	if stored.UsedAt == nil {
		t.Fatalf("expected used_at to be set")
	}
}

func TestAuthServiceRegisterValidation(t *testing.T) {
	env := setupServiceTest(t)

	_, err := env.authSvc.Register(context.Background(), "bad", "", "", "12345", "")
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestAuthServiceRegisterUserExists(t *testing.T) {
	env := setupServiceTest(t)
	admin := createUser(t, env, "admin@example.com", "admin", "pass", "admin")
	_ = createRegistrationKey(t, env, "111111", admin.ID)
	_ = createUser(t, env, "user@example.com", "user1", "pass", "user")

	_, err := env.authSvc.Register(context.Background(), "user@example.com", "newuser", "pass", "111111", "")
	if !errors.Is(err, ErrUserExists) {
		t.Fatalf("expected ErrUserExists, got %v", err)
	}
}

func TestAuthServiceCreateRegistrationKeys(t *testing.T) {
	env := setupServiceTest(t)
	admin := createUser(t, env, "admin@example.com", "admin", "pass", "admin")

	if _, err := env.authSvc.CreateRegistrationKeys(context.Background(), admin.ID, 0, nil); err == nil {
		t.Fatalf("expected validation error")
	}

	keys, err := env.authSvc.CreateRegistrationKeys(context.Background(), admin.ID, 2, nil)
	if err != nil {
		t.Fatalf("create keys: %v", err)
	}

	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}

	if keys[0].Code == keys[1].Code || len(keys[0].Code) != 6 || len(keys[1].Code) != 6 {
		t.Fatalf("unexpected key codes: %+v", keys)
	}
}

func TestAuthServiceCreateRegistrationKeysWithTeam(t *testing.T) {
	env := setupServiceTest(t)
	admin := createUser(t, env, "admin@example.com", "admin", "pass", "admin")
	team := createTeam(t, env, "Alpha")

	keys, err := env.authSvc.CreateRegistrationKeys(context.Background(), admin.ID, 1, &team.ID)
	if err != nil {
		t.Fatalf("create keys: %v", err)
	}

	if len(keys) != 1 || keys[0].TeamID == nil || *keys[0].TeamID != team.ID {
		t.Fatalf("expected team on key, got %+v", keys)
	}
}

func TestAuthServiceRegisterAssignsTeam(t *testing.T) {
	env := setupServiceTest(t)
	admin := createUser(t, env, "admin@example.com", "admin", "pass", "admin")
	team := createTeam(t, env, "Alpha")
	key := createRegistrationKeyWithTeam(t, env, "654321", admin.ID, &team.ID)

	user, err := env.authSvc.Register(context.Background(), "user@example.com", "user1", "pass1", key.Code, "")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if user.TeamID == nil || *user.TeamID != team.ID {
		t.Fatalf("expected user team assigned, got %+v", user.TeamID)
	}
}

func TestAuthServiceListRegistrationKeys(t *testing.T) {
	env := setupServiceTest(t)
	admin := createUser(t, env, "admin@example.com", "admin", "pass", "admin")
	user := createUser(t, env, "user@example.com", "user1", "pass", "user")

	usedBy := user.ID
	usedAt := time.Now().UTC()
	usedByIP := "192.0.2.1"
	key := &models.RegistrationKey{
		Code:      "222222",
		CreatedBy: admin.ID,
		CreatedAt: time.Now().UTC(),
		UsedBy:    &usedBy,
		UsedAt:    &usedAt,
		UsedByIP:  &usedByIP,
	}

	if err := env.regKeyRepo.Create(context.Background(), key); err != nil {
		t.Fatalf("create key: %v", err)
	}

	rows, err := env.authSvc.ListRegistrationKeys(context.Background())
	if err != nil {
		t.Fatalf("list keys: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("expected 1 key, got %d", len(rows))
	}

	if rows[0].CreatedByUsername != admin.Username || rows[0].UsedByUsername == nil || *rows[0].UsedByUsername != user.Username {
		t.Fatalf("unexpected key summary: %+v", rows[0])
	}
}

func TestAuthServiceLoginRefreshLogout(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "user@example.com", "user1", "pass", "user")

	if _, _, _, err := env.authSvc.Login(context.Background(), "user@example.com", "wrong"); !errors.Is(err, ErrInvalidCreds) {
		t.Fatalf("expected ErrInvalidCreds, got %v", err)
	}

	access, refresh, got, err := env.authSvc.Login(context.Background(), "user@example.com", "pass")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if access == "" || refresh == "" || got.ID != user.ID {
		t.Fatalf("unexpected login response")
	}

	claims, err := auth.ParseToken(env.cfg.JWT, refresh)
	if err != nil {
		t.Fatalf("parse refresh: %v", err)
	}

	val, err := env.redis.Get(context.Background(), refreshKey(claims.ID)).Result()
	if err != nil || val == "" {
		t.Fatalf("expected refresh token stored, err %v val %s", err, val)
	}

	if _, _, err := env.authSvc.Refresh(context.Background(), "bad-token"); !errors.Is(err, ErrInvalidCreds) {
		t.Fatalf("expected ErrInvalidCreds, got %v", err)
	}

	newAccess, newRefresh, err := env.authSvc.Refresh(context.Background(), refresh)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}

	if newAccess == "" || newRefresh == "" {
		t.Fatalf("expected new tokens")
	}

	if _, err := env.redis.Get(context.Background(), refreshKey(claims.ID)).Result(); !errors.Is(err, redis.Nil) {
		t.Fatalf("expected old refresh revoked, got %v", err)
	}

	if err := env.authSvc.Logout(context.Background(), "bad-token"); !errors.Is(err, ErrInvalidCreds) {
		t.Fatalf("expected ErrInvalidCreds, got %v", err)
	}

	newClaims, err := auth.ParseToken(env.cfg.JWT, newRefresh)
	if err != nil {
		t.Fatalf("parse new refresh: %v", err)
	}

	if err := env.authSvc.Logout(context.Background(), newRefresh); err != nil {
		t.Fatalf("logout: %v", err)
	}

	if _, err := env.redis.Get(context.Background(), refreshKey(newClaims.ID)).Result(); !errors.Is(err, redis.Nil) {
		t.Fatalf("expected refresh revoked, got %v", err)
	}
}

func TestAuthServiceRegisterMissingKey(t *testing.T) {
	env := setupServiceTest(t)
	_, err := env.authSvc.Register(context.Background(), "user@example.com", "user1", "pass", "123456", "")
	if err == nil {
		t.Fatalf("expected error")
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
