package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"smctf/internal/models"
)

func TestUserRepoGetDivisionID(t *testing.T) {
	env := setupRepoTest(t)
	division := createDivision(t, env, "Alpha")
	team := createTeamInDivision(t, env, "TeamA", division.ID)
	user := createUserWithTeam(t, env, "user@example.com", "user", "pass", models.UserRole, team.ID)

	got, err := env.userRepo.GetDivisionID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("get division id: %v", err)
	}

	if got != division.ID {
		t.Fatalf("expected division id %d, got %d", division.ID, got)
	}
}

func TestUserRepoGetDivisionIDNotFound(t *testing.T) {
	env := setupRepoTest(t)
	if _, err := env.userRepo.GetDivisionID(context.Background(), 999999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestUserRepoGetByEmailIncludesDivision(t *testing.T) {
	env := setupRepoTest(t)
	division := createDivision(t, env, "Alpha")
	team := createTeamInDivision(t, env, "TeamA", division.ID)
	user := &models.User{
		Email:        "a@example.com",
		Username:     "alpha",
		PasswordHash: "hash",
		Role:         models.UserRole,
		TeamID:       team.ID,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := env.userRepo.Create(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	got, err := env.userRepo.GetByEmail(context.Background(), user.Email)
	if err != nil {
		t.Fatalf("get by email: %v", err)
	}

	if got.DivisionID != division.ID || got.DivisionName != division.Name {
		t.Fatalf("expected division fields populated, got %+v", got)
	}
}
