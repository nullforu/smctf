package repo

import (
	"context"
	"errors"
	"testing"
)

func TestDivisionRepoCreateGetList(t *testing.T) {
	env := setupRepoTest(t)

	division := createDivision(t, env, "Alpha")
	if division.ID == 0 {
		t.Fatalf("expected division id")
	}

	gotByID, err := env.divisionRepo.GetByID(context.Background(), division.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if gotByID.ID != division.ID || gotByID.Name != division.Name {
		t.Fatalf("unexpected division: %+v", gotByID)
	}

	gotByName, err := env.divisionRepo.GetByName(context.Background(), division.Name)
	if err != nil {
		t.Fatalf("get by name: %v", err)
	}
	if gotByName.ID != division.ID {
		t.Fatalf("unexpected division by name: %+v", gotByName)
	}

	rows, err := env.divisionRepo.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) < 2 {
		t.Fatalf("expected at least 2 divisions, got %d", len(rows))
	}
}

func TestDivisionRepoNotFound(t *testing.T) {
	env := setupRepoTest(t)

	if _, err := env.divisionRepo.GetByID(context.Background(), 999999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}

	if _, err := env.divisionRepo.GetByName(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}
