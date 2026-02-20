package service

import (
	"context"
	"smctf/internal/models"
	"testing"
)

func TestUserServiceMoveUserTeam(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "move@example.com", "move", "pass", models.UserRole)
	newTeam := createTeam(t, env, "new-team")

	updated, err := env.userSvc.MoveUserTeam(context.Background(), user.ID, newTeam.ID)
	if err != nil {
		t.Fatalf("move user team: %v", err)
	}

	if updated.TeamID != newTeam.ID {
		t.Fatalf("expected team id %d, got %d", newTeam.ID, updated.TeamID)
	}

	if updated.TeamName != newTeam.Name {
		t.Fatalf("expected team name %q, got %q", newTeam.Name, updated.TeamName)
	}

	if _, err := env.userSvc.MoveUserTeam(context.Background(), user.ID, 999999); err == nil {
		t.Fatalf("expected error for invalid team")
	}
}

func TestUserServiceBlockUnblock(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "block@example.com", "block", "pass", models.UserRole)

	blocked, err := env.userSvc.BlockUser(context.Background(), user.ID, "bad")
	if err != nil {
		t.Fatalf("block user: %v", err)
	}

	if blocked.Role != models.BlockedRole || blocked.BlockedReason == nil {
		t.Fatalf("expected blocked user, got %+v", blocked)
	}

	unblocked, err := env.userSvc.UnblockUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("unblock user: %v", err)
	}

	if unblocked.Role != models.UserRole || unblocked.BlockedReason != nil {
		t.Fatalf("expected unblocked user, got %+v", unblocked)
	}

	admin := createUserWithNewTeam(t, env, "admin@example.com", models.AdminRole, "pass", models.AdminRole)
	if _, err := env.userSvc.BlockUser(context.Background(), admin.ID, "bad"); err == nil {
		t.Fatalf("expected admin block error")
	}

	if _, err := env.userSvc.UnblockUser(context.Background(), admin.ID); err == nil {
		t.Fatalf("expected admin unblock error")
	}
}

func TestUserServiceGetByIDListUpdateProfile(t *testing.T) {
	env := setupServiceTest(t)
	team := createTeam(t, env, "team-a")
	user := createUserWithTeam(t, env, "user@example.com", models.UserRole, "pass", models.UserRole, team.ID)

	got, err := env.userSvc.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.ID != user.ID || got.TeamName != team.Name {
		t.Fatalf("unexpected user: %+v", got)
	}

	users, err := env.userSvc.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(users) != 1 || users[0].ID != user.ID {
		t.Fatalf("unexpected list: %+v", users)
	}

	updated, err := env.userSvc.UpdateProfile(context.Background(), user.ID, ptr("newname"))
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}

	if updated.Username != "newname" {
		t.Fatalf("expected username updated, got %s", updated.Username)
	}
}

func ptr(value string) *string {
	return &value
}
