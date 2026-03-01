package service

import (
	"context"
	"errors"
	"testing"
)

func TestDivisionServiceCreateListGet(t *testing.T) {
	env := setupServiceTest(t)

	if _, err := env.divisionSvc.CreateDivision(context.Background(), ""); err == nil {
		t.Fatalf("expected validation error")
	}

	division, err := env.divisionSvc.CreateDivision(context.Background(), "Alpha")
	if err != nil {
		t.Fatalf("create division: %v", err)
	}

	if _, err := env.divisionSvc.CreateDivision(context.Background(), "Alpha"); err == nil {
		t.Fatalf("expected duplicate error")
	}

	list, err := env.divisionSvc.ListDivisions(context.Background())
	if err != nil {
		t.Fatalf("list divisions: %v", err)
	}

	if len(list) < 2 {
		t.Fatalf("expected at least 2 divisions, got %d", len(list))
	}

	got, err := env.divisionSvc.GetDivision(context.Background(), division.ID)
	if err != nil {
		t.Fatalf("get division: %v", err)
	}

	if got.ID != division.ID || got.Name != division.Name {
		t.Fatalf("unexpected division: %+v", got)
	}
}

func TestDivisionServiceGetDivisionErrors(t *testing.T) {
	env := setupServiceTest(t)

	if _, err := env.divisionSvc.GetDivision(context.Background(), 0); err == nil {
		t.Fatalf("expected validation error")
	}

	if _, err := env.divisionSvc.GetDivision(context.Background(), 999999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}
