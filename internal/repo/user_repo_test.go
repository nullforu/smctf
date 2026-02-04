package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"smctf/internal/models"
)

func TestUserRepoCRUD(t *testing.T) {
	env := setupRepoTest(t)

	user := createUser(t, env, "user@example.com", "user1", "pass", "user")

	got, err := env.userRepo.GetByEmail(context.Background(), "user@example.com")
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}

	if got.ID != user.ID {
		t.Fatalf("expected user id %d, got %d", user.ID, got.ID)
	}

	got, err = env.userRepo.GetByEmailOrUsername(context.Background(), "nope@example.com", "user1")
	if err != nil {
		t.Fatalf("GetByEmailOrUsername: %v", err)
	}

	if got.ID != user.ID {
		t.Fatalf("expected user id %d, got %d", user.ID, got.ID)
	}

	got, err = env.userRepo.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.Email != user.Email {
		t.Fatalf("unexpected email: %s", got.Email)
	}

	team, err := env.teamRepo.GetByID(context.Background(), got.TeamID)
	if err != nil {
		t.Fatalf("expected team lookup: %v", err)
	}
	if got.TeamName != team.Name {
		t.Fatalf("expected team name %q, got %+v", team.Name, got.TeamName)
	}

	got.Username = "user2"
	if err := env.userRepo.Update(context.Background(), got); err != nil {
		t.Fatalf("Update: %v", err)
	}

	updated, err := env.userRepo.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetByID updated: %v", err)
	}

	if updated.Username != "user2" {
		t.Fatalf("expected updated username, got %s", updated.Username)
	}

	users, err := env.userRepo.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}

	team, err = env.teamRepo.GetByID(context.Background(), users[0].TeamID)
	if err != nil {
		t.Fatalf("expected team lookup: %v", err)
	}
	if users[0].TeamName != team.Name {
		t.Fatalf("expected team name %q, got %+v", team.Name, users[0].TeamName)
	}
}

func TestUserRepoNotFound(t *testing.T) {
	env := setupRepoTest(t)
	_, err := env.userRepo.GetByID(context.Background(), 123)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUserRepoGetByIDTeamName(t *testing.T) {
	env := setupRepoTest(t)
	team := createTeam(t, env, "Alpha")
	user := createUserWithTeam(t, env, "u1@example.com", "u1", "pass", "user", team.ID)

	got, err := env.userRepo.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.TeamName != team.Name {
		t.Fatalf("expected team name %q, got %+v", team.Name, got.TeamName)
	}
}

func TestUserRepoListOrdering(t *testing.T) {
	env := setupRepoTest(t)
	_ = createUser(t, env, "u1@example.com", "u1", "pass", "user")
	_ = createUser(t, env, "u2@example.com", "u2", "pass", "user")

	users, err := env.userRepo.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}

	if users[0].ID > users[1].ID {
		t.Fatalf("expected ascending id order")
	}
}

func TestUserRepoCreateDuplicateEmail(t *testing.T) {
	env := setupRepoTest(t)
	team := createTeam(t, env, "Dup Team")
	user := &models.User{
		Email:        "dup@example.com",
		Username:     "dup1",
		PasswordHash: "hash",
		Role:         "user",
		TeamID:       team.ID,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := env.userRepo.Create(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	user2 := &models.User{
		Email:        "dup@example.com",
		Username:     "dup2",
		PasswordHash: "hash",
		Role:         "user",
		TeamID:       team.ID,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := env.userRepo.Create(context.Background(), user2); err == nil {
		t.Fatalf("expected error for duplicate email")
	}
}

func TestUserRepoCreateError(t *testing.T) {
	closedDB := newClosedRepoDB(t)
	repo := NewUserRepo(closedDB)

	team := &models.Team{
		ID:   1,
		Name: "Err Team",
	}
	user := &models.User{
		Email:        "err@example.com",
		Username:     "erruser",
		PasswordHash: "hash",
		Role:         "user",
		TeamID:       team.ID,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := repo.Create(context.Background(), user); err == nil {
		t.Fatalf("expected error from Create")
	}
}
