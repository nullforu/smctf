package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"smctf/internal/models"
)

func TestRegistrationKeyRepoCRUD(t *testing.T) {
	env := setupRepoTest(t)
	team := createTeam(t, env, "Alpha")
	admin := createUserWithNewTeam(t, env, "admin@example.com", models.AdminRole, "pass", models.AdminRole)
	user := createUserWithNewTeam(t, env, "user@example.com", models.UserRole, "pass", models.UserRole)

	usedAt := time.Now().UTC()

	key := &models.RegistrationKey{
		Code:      "ABCDEFGHJKLMNPQ2",
		CreatedBy: admin.ID,
		TeamID:    team.ID,
		MaxUses:   3,
		UsedCount: 1,
		CreatedAt: time.Now().UTC(),
	}
	if err := env.regKeyRepo.Create(context.Background(), key); err != nil {
		t.Fatalf("Create: %v", err)
	}

	use := &models.RegistrationKeyUse{
		RegistrationKeyID: key.ID,
		UsedBy:            user.ID,
		UsedByIP:          "203.0.113.10",
		UsedAt:            usedAt,
	}
	if _, err := env.db.NewInsert().Model(use).Exec(context.Background()); err != nil {
		t.Fatalf("Create use: %v", err)
	}

	got, err := env.regKeyRepo.GetByCodeForUpdate(context.Background(), env.db, "ABCDEFGHJKLMNPQ2")
	if err != nil {
		t.Fatalf("GetByCodeForUpdate: %v", err)
	}

	if got.ID != key.ID {
		t.Fatalf("expected key id %d, got %d", key.ID, got.ID)
	}

	rows, err := env.regKeyRepo.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	if rows[0].CreatedByUsername != admin.Username {
		t.Fatalf("expected creator username, got %s", rows[0].CreatedByUsername)
	}

	if rows[0].MaxUses != 3 || rows[0].UsedCount != 1 {
		t.Fatalf("expected usage summary, got %+v", rows[0])
	}

	if len(rows[0].Uses) != 1 || rows[0].Uses[0].UsedByUsername != user.Username {
		t.Fatalf("expected uses list, got %+v", rows[0].Uses)
	}

	if rows[0].TeamID != team.ID || rows[0].TeamName != team.Name {
		t.Fatalf("expected team in key summary, got %+v", rows[0])
	}
}

func TestRegistrationKeyRepoNotFound(t *testing.T) {
	env := setupRepoTest(t)
	_, err := env.regKeyRepo.GetByCodeForUpdate(context.Background(), env.db, "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestRegistrationKeyRepoNotUsed(t *testing.T) {
	env := setupRepoTest(t)
	team := createTeam(t, env, "Beta")
	admin := createUserWithNewTeam(t, env, "admin@example.com", models.AdminRole, "pass", models.AdminRole)

	key := &models.RegistrationKey{
		Code:      "ABCDEFGHJKLMNPQ3",
		CreatedBy: admin.ID,
		TeamID:    team.ID,
		MaxUses:   3,
		UsedCount: 0,
		CreatedAt: time.Now().UTC(),
	}
	if err := env.regKeyRepo.Create(context.Background(), key); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := env.regKeyRepo.GetByCodeForUpdate(context.Background(), env.db, "ABCDEFGHJKLMNPQ3")
	if err != nil {
		t.Fatalf("GetByCodeForUpdate: %v", err)
	}

	if got.ID != key.ID {
		t.Fatalf("expected key id %d, got %d", key.ID, got.ID)
	}
}

func TestRegistrationKeyRepoListEmpty(t *testing.T) {
	env := setupRepoTest(t)
	rows, err := env.regKeyRepo.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(rows))
	}
}
