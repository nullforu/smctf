package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"smctf/internal/models"
)

func TestGroupRepoCRUD(t *testing.T) {
	env := setupRepoTest(t)

	group := &models.Group{
		Name:      "Alpha School",
		CreatedAt: time.Now().UTC(),
	}

	if err := env.groupRepo.Create(context.Background(), group); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := env.groupRepo.GetByID(context.Background(), group.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.ID != group.ID || got.Name != group.Name {
		t.Fatalf("unexpected group: %+v", got)
	}

	list, err := env.groupRepo.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(list) != 1 {
		t.Fatalf("expected 1 group, got %d", len(list))
	}
}

func TestGroupRepoNotFound(t *testing.T) {
	env := setupRepoTest(t)

	_, err := env.groupRepo.GetByID(context.Background(), 999)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
