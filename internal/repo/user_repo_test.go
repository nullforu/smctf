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

func TestUserRepoCreateGetByIDUpdate(t *testing.T) {
	env := setupRepoTest(t)
	team := createTeam(t, env, "team-a")

	user := createUserWithTeam(t, env, "update@example.com", "update-user", "pass", models.UserRole, team.ID)

	got, err := env.userRepo.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}

	if got.ID != user.ID || got.TeamID != team.ID {
		t.Fatalf("unexpected user: %+v", got)
	}

	got.Username = "updated-name"
	if err := env.userRepo.Update(context.Background(), got); err != nil {
		t.Fatalf("update user: %v", err)
	}

	updated, err := env.userRepo.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("get updated user: %v", err)
	}

	if updated.Username != "updated-name" {
		t.Fatalf("expected updated username, got %q", updated.Username)
	}
}

func TestUserRepoGetByEmailOrUsername(t *testing.T) {
	env := setupRepoTest(t)
	user := createUserWithNewTeam(t, env, "lookup@example.com", "lookup-user", "pass", models.UserRole)

	got, err := env.userRepo.GetByEmailOrUsername(context.Background(), "lookup@example.com", "nope")
	if err != nil {
		t.Fatalf("get by email or username (email): %v", err)
	}

	if got.ID != user.ID {
		t.Fatalf("expected user by email, got %+v", got)
	}

	got, err = env.userRepo.GetByEmailOrUsername(context.Background(), "missing@example.com", "lookup-user")
	if err != nil {
		t.Fatalf("get by email or username (username): %v", err)
	}

	if got.ID != user.ID {
		t.Fatalf("expected user by username, got %+v", got)
	}
}

func TestUserRepoListWithDivisionFilter(t *testing.T) {
	env := setupRepoTest(t)
	divisionA := createDivision(t, env, "DivisionA")
	divisionB := createDivision(t, env, "DivisionB")
	teamA := createTeamInDivision(t, env, "TeamA", divisionA.ID)
	teamB := createTeamInDivision(t, env, "TeamB", divisionB.ID)

	userA := createUserWithTeam(t, env, "a@example.com", "user-a", "pass", models.UserRole, teamA.ID)
	_ = createUserWithTeam(t, env, "b@example.com", "user-b", "pass", models.UserRole, teamB.ID)

	allUsers, err := env.userRepo.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("list users: %v", err)
	}

	if len(allUsers) != 2 {
		t.Fatalf("expected 2 users, got %d", len(allUsers))
	}

	filtered, err := env.userRepo.List(context.Background(), &divisionA.ID)
	if err != nil {
		t.Fatalf("list users by division: %v", err)
	}

	if len(filtered) != 1 || filtered[0].ID != userA.ID {
		t.Fatalf("unexpected filtered users: %+v", filtered)
	}
}

func TestUserRepoNotFoundCases(t *testing.T) {
	env := setupRepoTest(t)

	if _, err := env.userRepo.GetByID(context.Background(), 123456); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected GetByID not found, got %v", err)
	}

	if _, err := env.userRepo.GetByEmail(context.Background(), "missing@example.com"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected GetByEmail not found, got %v", err)
	}

	if _, err := env.userRepo.GetByEmailOrUsername(context.Background(), "missing@example.com", "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected GetByEmailOrUsername not found, got %v", err)
	}
}
