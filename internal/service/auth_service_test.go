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
	key := createRegistrationKey(t, env, "ABCDEFGHJKLMNPQ2", admin.ID)

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

	if stored.UsedCount != 1 {
		t.Fatalf("expected used_count 1, got %d", stored.UsedCount)
	}

	var uses []models.RegistrationKeyUse
	if err := env.db.NewSelect().Model(&uses).Where("registration_key_id = ?", stored.ID).Scan(context.Background()); err != nil {
		t.Fatalf("load uses: %v", err)
	}
	if len(uses) != 1 || uses[0].UsedBy != user.ID || uses[0].UsedByIP != "127.0.0.1" {
		t.Fatalf("unexpected uses: %+v", uses)
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
	_ = createRegistrationKey(t, env, "ABCDEFGHJKLMNPQ3", admin.ID)
	_ = createUser(t, env, "user@example.com", "user1", "pass", "user")

	_, err := env.authSvc.Register(context.Background(), "user@example.com", "newuser", "pass", "ABCDEFGHJKLMNPQ3", "")
	if !errors.Is(err, ErrUserExists) {
		t.Fatalf("expected ErrUserExists, got %v", err)
	}
}

func TestAuthServiceCreateRegistrationKeys(t *testing.T) {
	env := setupServiceTest(t)
	admin := createUser(t, env, "admin@example.com", "admin", "pass", "admin")
	team := createTeam(t, env, "Alpha")

	if _, err := env.authSvc.CreateRegistrationKeys(context.Background(), admin.ID, 0, team.ID, 1); err == nil {
		t.Fatalf("expected validation error")
	}
	if _, err := env.authSvc.CreateRegistrationKeys(context.Background(), admin.ID, 1, team.ID, 0); err == nil {
		t.Fatalf("expected validation error")
	}

	keys, err := env.authSvc.CreateRegistrationKeys(context.Background(), admin.ID, 2, team.ID, 2)
	if err != nil {
		t.Fatalf("create keys: %v", err)
	}

	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}

	if keys[0].Code == keys[1].Code || len(keys[0].Code) != 16 || len(keys[1].Code) != 16 {
		t.Fatalf("unexpected key codes: %+v", keys)
	}
	if keys[0].MaxUses != 2 {
		t.Fatalf("expected max_uses 2, got %d", keys[0].MaxUses)
	}
}

func TestAuthServiceCreateRegistrationKeysWithTeam(t *testing.T) {
	env := setupServiceTest(t)
	admin := createUser(t, env, "admin@example.com", "admin", "pass", "admin")
	team := createTeam(t, env, "Alpha")

	keys, err := env.authSvc.CreateRegistrationKeys(context.Background(), admin.ID, 1, team.ID, 1)
	if err != nil {
		t.Fatalf("create keys: %v", err)
	}

	if len(keys) != 1 || keys[0].TeamID != team.ID {
		t.Fatalf("expected team on key, got %+v", keys)
	}
}

func TestAuthServiceCreateRegistrationKeysInvalidTeam(t *testing.T) {
	env := setupServiceTest(t)
	admin := createUser(t, env, "admin@example.com", "admin", "pass", "admin")

	_, err := env.authSvc.CreateRegistrationKeys(context.Background(), admin.ID, 1, 9999, 1)
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestAuthServiceRegisterAssignsTeam(t *testing.T) {
	env := setupServiceTest(t)
	admin := createUser(t, env, "admin@example.com", "admin", "pass", "admin")
	team := createTeam(t, env, "Alpha")
	key := createRegistrationKeyWithTeam(t, env, "ABCDEFGHJKLMNPQ4", admin.ID, team.ID)

	user, err := env.authSvc.Register(context.Background(), "user@example.com", "user1", "pass1", key.Code, "")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if user.TeamID != team.ID {
		t.Fatalf("expected user team assigned, got %+v", user.TeamID)
	}
}

func TestAuthServiceRegisterUsedRegistrationKey(t *testing.T) {
	env := setupServiceTest(t)
	admin := createUser(t, env, "admin@example.com", "admin", "pass", "admin")
	team := createTeam(t, env, "Key Team")

	key := &models.RegistrationKey{
		Code:      "ABCDEFGHJKLMNPQ5",
		CreatedBy: admin.ID,
		TeamID:    team.ID,
		MaxUses:   1,
		UsedCount: 1,
		CreatedAt: time.Now().UTC(),
	}
	if err := env.regKeyRepo.Create(context.Background(), key); err != nil {
		t.Fatalf("create key: %v", err)
	}

	_, err := env.authSvc.Register(context.Background(), "user@example.com", "user1", "pass1", key.Code, "")
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestAuthServiceListRegistrationKeys(t *testing.T) {
	env := setupServiceTest(t)
	admin := createUser(t, env, "admin@example.com", "admin", "pass", "admin")
	user := createUser(t, env, "user@example.com", "user1", "pass", "user")

	usedAt := time.Now().UTC()
	key := &models.RegistrationKey{
		Code:      "ABCDEFGHJKLMNPQ6",
		CreatedBy: admin.ID,
		TeamID:    createTeam(t, env, "Key Team").ID,
		MaxUses:   3,
		UsedCount: 1,
		CreatedAt: time.Now().UTC(),
	}

	if err := env.regKeyRepo.Create(context.Background(), key); err != nil {
		t.Fatalf("create key: %v", err)
	}

	use := &models.RegistrationKeyUse{
		RegistrationKeyID: key.ID,
		UsedBy:            user.ID,
		UsedByIP:          "192.0.2.1",
		UsedAt:            usedAt,
	}
	if _, err := env.db.NewInsert().Model(use).Exec(context.Background()); err != nil {
		t.Fatalf("create use: %v", err)
	}

	rows, err := env.authSvc.ListRegistrationKeys(context.Background())
	if err != nil {
		t.Fatalf("list keys: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("expected 1 key, got %d", len(rows))
	}

	if rows[0].CreatedByUsername != admin.Username || len(rows[0].Uses) != 1 || rows[0].Uses[0].UsedByUsername != user.Username {
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

func TestAuthServiceLoginBlocked(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "blocked@example.com", "blocked", "pass", "user")
	user.Role = "blocked"
	if err := env.userRepo.Update(context.Background(), user); err != nil {
		t.Fatalf("update user: %v", err)
	}

	if _, _, _, err := env.authSvc.Login(context.Background(), "blocked@example.com", "pass"); err != nil {
		t.Fatalf("expected login success, got %v", err)
	}
}

func TestAuthServiceRefreshBlocked(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "user@example.com", "user1", "pass", "user")

	_, refresh, _, err := env.authSvc.Login(context.Background(), "user@example.com", "pass")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	user.Role = "blocked"
	if err := env.userRepo.Update(context.Background(), user); err != nil {
		t.Fatalf("update user: %v", err)
	}

	if _, _, err := env.authSvc.Refresh(context.Background(), refresh); err != nil {
		t.Fatalf("expected refresh success, got %v", err)
	}
}

func TestAuthServiceRefreshUserNotFound(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "user@example.com", "user1", "pass", "user")

	_, refresh, _, err := env.authSvc.Login(context.Background(), "user@example.com", "pass")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if _, err := env.db.NewDelete().Model(&models.User{}).Where("id = ?", user.ID).Exec(context.Background()); err != nil {
		t.Fatalf("delete user: %v", err)
	}

	if _, _, err := env.authSvc.Refresh(context.Background(), refresh); !errors.Is(err, ErrInvalidCreds) {
		t.Fatalf("expected ErrInvalidCreds, got %v", err)
	}
}

func TestAuthServiceRegisterMissingKey(t *testing.T) {
	env := setupServiceTest(t)
	_, err := env.authSvc.Register(context.Background(), "user@example.com", "user1", "pass", "MISSING1", "")
	if err == nil {
		t.Fatalf("expected error")
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
