package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"smctf/internal/auth"
	"smctf/internal/models"

	"github.com/redis/go-redis/v9"
)

func TestAuthServiceRegisterSuccess(t *testing.T) {
	env := setupServiceTest(t)
	admin := createUserWithNewTeam(t, env, "admin@example.com", models.AdminRole, "pass", models.AdminRole)
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

func TestAuthServiceRegisterPasswordTooLong(t *testing.T) {
	env := setupServiceTest(t)
	admin := createUserWithNewTeam(t, env, "admin@example.com", models.AdminRole, "pass", models.AdminRole)
	key := createRegistrationKey(t, env, "ABCDEFGHJKLMNPQ7", admin.ID)

	_, err := env.authSvc.Register(context.Background(), "user@example.com", "user1", strings.Repeat("a", 73), key.Code, "")
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}

	if len(ve.Fields) == 0 || ve.Fields[0].Field != "password" || ve.Fields[0].Reason != "max bytes is 72" {
		t.Fatalf("unexpected validation details: %+v", ve.Fields)
	}
}

func TestAuthServiceRegisterUsernameTooLong(t *testing.T) {
	env := setupServiceTest(t)
	admin := createUserWithNewTeam(t, env, "admin@example.com", models.AdminRole, "pass", models.AdminRole)
	key := createRegistrationKey(t, env, "ABCDEFGHJKLMNPQ7", admin.ID)

	_, err := env.authSvc.Register(context.Background(), "user@example.com", "ThisNameIsWayTooLong", "pass1", key.Code, "")
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}

	found := false
	for _, f := range ve.Fields {
		if f.Field == "username" {
			found = true
		}
	}

	if !found {
		t.Fatalf("expected username length error, got %+v", ve.Fields)
	}
}

func TestAuthServiceRegisterUserExists(t *testing.T) {
	env := setupServiceTest(t)
	admin := createUserWithNewTeam(t, env, "admin@example.com", models.AdminRole, "pass", models.AdminRole)
	_ = createRegistrationKey(t, env, "ABCDEFGHJKLMNPQ3", admin.ID)
	_ = createUserWithNewTeam(t, env, "user@example.com", "user1", "pass", models.UserRole)

	_, err := env.authSvc.Register(context.Background(), "user@example.com", "newuser", "pass", "ABCDEFGHJKLMNPQ3", "")
	if !errors.Is(err, ErrUserExists) {
		t.Fatalf("expected ErrUserExists, got %v", err)
	}
}

func TestAuthServiceCreateRegistrationKeys(t *testing.T) {
	env := setupServiceTest(t)
	admin := createUserWithNewTeam(t, env, "admin@example.com", models.AdminRole, "pass", models.AdminRole)
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
	admin := createUserWithNewTeam(t, env, "admin@example.com", models.AdminRole, "pass", models.AdminRole)
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
	admin := createUserWithNewTeam(t, env, "admin@example.com", models.AdminRole, "pass", models.AdminRole)

	_, err := env.authSvc.CreateRegistrationKeys(context.Background(), admin.ID, 1, 9999, 1)
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestAuthServiceRegisterAssignsTeam(t *testing.T) {
	env := setupServiceTest(t)
	admin := createUserWithNewTeam(t, env, "admin@example.com", models.AdminRole, "pass", models.AdminRole)
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
	admin := createUserWithNewTeam(t, env, "admin@example.com", models.AdminRole, "pass", models.AdminRole)
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
	admin := createUserWithNewTeam(t, env, "admin@example.com", models.AdminRole, "pass", models.AdminRole)
	user := createUserWithNewTeam(t, env, "user@example.com", "user1", "pass", models.UserRole)

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

func TestAuthServiceExportImportRegistrationKeys(t *testing.T) {
	env := setupServiceTest(t)
	admin := createUserWithNewTeam(t, env, "admin@example.com", models.AdminRole, "pass", models.AdminRole)
	team := createTeam(t, env, "Key Team")
	key := createRegistrationKeyWithTeam(t, env, "ABCDEFGHJKLMNPQ8", admin.ID, team.ID)

	exported, err := env.authSvc.ExportRegistrationKeys(context.Background(), []int64{key.ID})
	if err != nil {
		t.Fatalf("export keys: %v", err)
	}

	if len(exported.Keys) != 1 || exported.Keys[0].Code != key.Code || exported.Keys[0].TeamName != team.Name || exported.Keys[0].MaxUses != key.MaxUses {
		t.Fatalf("unexpected export bundle: %+v", exported)
	}

	otherTeam := createTeam(t, env, "Imported Key Team")
	imported, err := env.authSvc.ImportRegistrationKeys(context.Background(), admin.ID, RegistrationKeyExportBundle{
		Version: 1,
		Keys: []RegistrationKeyExportItem{{
			ID:        999,
			Code:      "ABCDEFGHJKLMNPQ9",
			TeamName:  otherTeam.Name,
			MaxUses:   2,
			CreatedAt: time.Now().UTC(),
		}},
	})
	if err != nil {
		t.Fatalf("import keys: %v", err)
	}

	if len(imported) != 1 || imported[0].Code != "ABCDEFGHJKLMNPQ9" || imported[0].TeamName != otherTeam.Name || imported[0].MaxUses != 2 || imported[0].UsedCount != 0 {
		t.Fatalf("unexpected imported keys: %+v", imported)
	}
}

func TestAuthServiceImportRegistrationKeysValidation(t *testing.T) {
	env := setupServiceTest(t)
	admin := createUserWithNewTeam(t, env, "admin@example.com", models.AdminRole, "pass", models.AdminRole)
	team := createTeam(t, env, "Dup Team")
	_ = createRegistrationKeyWithTeam(t, env, "ABCDEFGHJKLMNPQ4", admin.ID, team.ID)

	_, err := env.authSvc.ImportRegistrationKeys(context.Background(), admin.ID, RegistrationKeyExportBundle{
		Version: 1,
		Keys: []RegistrationKeyExportItem{{
			ID:        1,
			Code:      "ABCDEFGHJKLMNPQ4",
			TeamName:  team.Name,
			MaxUses:   1,
			CreatedAt: time.Now().UTC(),
		}},
	})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}

	_, err = env.authSvc.ImportRegistrationKeys(context.Background(), admin.ID, RegistrationKeyExportBundle{
		Version: 1,
		Keys: []RegistrationKeyExportItem{{
			ID:        2,
			Code:      "ABCDEFGHJKLMNPRA",
			TeamName:  "Missing Team",
			MaxUses:   1,
			CreatedAt: time.Now().UTC(),
		}},
	})
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestAuthServiceLoginRefreshLogout(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "user@example.com", "user1", "pass", models.UserRole)

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
	user := createUserWithNewTeam(t, env, "blocked@example.com", models.BlockedRole, "pass", models.UserRole)
	user.Role = models.BlockedRole
	if err := env.userRepo.Update(context.Background(), user); err != nil {
		t.Fatalf("update user: %v", err)
	}

	if _, _, _, err := env.authSvc.Login(context.Background(), "blocked@example.com", "pass"); err != nil {
		t.Fatalf("expected login success, got %v", err)
	}
}

func TestAuthServiceRefreshBlocked(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "user@example.com", "user1", "pass", models.UserRole)

	_, refresh, _, err := env.authSvc.Login(context.Background(), "user@example.com", "pass")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	user.Role = models.BlockedRole
	if err := env.userRepo.Update(context.Background(), user); err != nil {
		t.Fatalf("update user: %v", err)
	}

	if _, _, err := env.authSvc.Refresh(context.Background(), refresh); err != nil {
		t.Fatalf("expected refresh success, got %v", err)
	}
}

func TestAuthServiceRefreshUserNotFound(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "user@example.com", "user1", "pass", models.UserRole)

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

func TestAuthServiceLoginUserNotFound(t *testing.T) {
	env := setupServiceTest(t)

	if _, _, _, err := env.authSvc.Login(context.Background(), "missing@example.com", "pass"); !errors.Is(err, ErrInvalidCreds) {
		t.Fatalf("expected ErrInvalidCreds, got %v", err)
	}
}

func TestAuthServiceParseRefreshTokenValidation(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "parse@example.com", "parse", "pass", models.UserRole)

	access, err := auth.GenerateAccessToken(env.cfg.JWT, user.ID, user.Role)
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}

	if _, err := env.authSvc.parseRefreshToken(access); !errors.Is(err, ErrInvalidCreds) {
		t.Fatalf("expected ErrInvalidCreds for access token, got %v", err)
	}

	if _, err := env.authSvc.parseRefreshToken("not-a-token"); !errors.Is(err, ErrInvalidCreds) {
		t.Fatalf("expected ErrInvalidCreds for malformed token, got %v", err)
	}
}

func TestAuthServiceAssertRefreshValidCases(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "assert@example.com", "assert", "pass", models.UserRole)
	key := "custom-jti"

	if err := env.authSvc.assertRefreshValid(context.Background(), key, user.ID); !errors.Is(err, ErrInvalidCreds) {
		t.Fatalf("expected ErrInvalidCreds when key missing, got %v", err)
	}

	if err := env.redis.Set(context.Background(), refreshKey(key), "", time.Minute).Err(); err != nil {
		t.Fatalf("seed redis empty value: %v", err)
	}

	if err := env.authSvc.assertRefreshValid(context.Background(), key, user.ID); !errors.Is(err, ErrInvalidCreds) {
		t.Fatalf("expected ErrInvalidCreds for empty stored value, got %v", err)
	}

	if err := env.redis.Set(context.Background(), refreshKey(key), strconv.FormatInt(user.ID+1, 10), time.Minute).Err(); err != nil {
		t.Fatalf("seed redis mismatched value: %v", err)
	}

	if err := env.authSvc.assertRefreshValid(context.Background(), key, user.ID); !errors.Is(err, ErrInvalidCreds) {
		t.Fatalf("expected ErrInvalidCreds for mismatched user id, got %v", err)
	}

	if err := env.redis.Set(context.Background(), refreshKey(key), strconv.FormatInt(user.ID, 10), time.Minute).Err(); err != nil {
		t.Fatalf("seed redis valid value: %v", err)
	}

	if err := env.authSvc.assertRefreshValid(context.Background(), key, user.ID); err != nil {
		t.Fatalf("expected valid refresh state, got %v", err)
	}
}

func TestAuthServiceRefreshOldTokenRevoked(t *testing.T) {
	env := setupServiceTest(t)
	createUserWithNewTeam(t, env, "refresh-old@example.com", "refresh-old", "pass", models.UserRole)

	_, refresh, _, err := env.authSvc.Login(context.Background(), "refresh-old@example.com", "pass")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if _, _, err := env.authSvc.Refresh(context.Background(), refresh); err != nil {
		t.Fatalf("first refresh: %v", err)
	}

	if _, _, err := env.authSvc.Refresh(context.Background(), refresh); !errors.Is(err, ErrInvalidCreds) {
		t.Fatalf("expected ErrInvalidCreds for revoked refresh token, got %v", err)
	}
}

func TestAuthServiceRegisterTrimsNormalization(t *testing.T) {
	env := setupServiceTest(t)
	admin := createUserWithNewTeam(t, env, "admin2@example.com", models.AdminRole, "pass", models.AdminRole)
	key := createRegistrationKey(t, env, "ABCDEFGHJKLMNPQ7", admin.ID)

	user, err := env.authSvc.Register(context.Background(), "  MiXeD@Example.com  ", "  trim-user  ", "pass1", "  "+strings.ToLower(key.Code)+"  ", " 192.0.2.5 ")
	if err != nil {
		t.Fatalf("register with normalization: %v", err)
	}

	if user.Email != "mixed@example.com" || user.Username != "trim-user" {
		t.Fatalf("unexpected normalized user: %+v", user)
	}
}
