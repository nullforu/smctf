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
	admin := createUser(t, env, "admin@example.com", "admin", "pass", "admin")
	user := createUser(t, env, "user@example.com", "user", "pass", "user")

	usedBy := user.ID
	usedAt := time.Now().UTC()
	usedByIP := "203.0.113.10"

	key := &models.RegistrationKey{
		Code:      "123456",
		CreatedBy: admin.ID,
		CreatedAt: time.Now().UTC(),
		UsedBy:    &usedBy,
		UsedAt:    &usedAt,
		UsedByIP:  &usedByIP,
	}
	if err := env.regKeyRepo.Create(context.Background(), key); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := env.regKeyRepo.GetByCodeForUpdate(context.Background(), env.db, "123456")
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

	if rows[0].UsedByUsername == nil || *rows[0].UsedByUsername != user.Username {
		t.Fatalf("expected used by username, got %+v", rows[0].UsedByUsername)
	}
}

func TestRegistrationKeyRepoNotFound(t *testing.T) {
	env := setupRepoTest(t)
	_, err := env.regKeyRepo.GetByCodeForUpdate(context.Background(), env.db, "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
