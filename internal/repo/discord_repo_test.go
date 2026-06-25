package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"smctf/internal/models"
)

func createDiscordConnection(t *testing.T, repo *DiscordRepo, userID int64, discordUserID string) *models.DiscordConnection {
	t.Helper()
	username := "neo"
	globalName := "Neo"
	avatar := "avatarhash"
	conn := &models.DiscordConnection{
		UserID:            userID,
		DiscordUserID:     discordUserID,
		DiscordUsername:   &username,
		DiscordGlobalName: &globalName,
		DiscordAvatar:     &avatar,
		RoleStatus:        models.DiscordStatusConnected,
		ConnectedAt:       time.Now().UTC(),
	}
	if err := repo.Create(context.Background(), conn); err != nil {
		t.Fatalf("create discord connection: %v", err)
	}

	return conn
}

func TestDiscordRepoCRUD(t *testing.T) {
	env := setupRepoTest(t)
	repo := NewDiscordRepo(env.db)
	user := createUserWithNewTeam(t, env, "discord-crud@example.com", "discord-crud", "pass", models.UserRole)

	created := createDiscordConnection(t, repo, user.ID, "900000000000000001")

	byUser, err := repo.GetByUserID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetByUserID: %v", err)
	}

	if byUser.DiscordUserID != created.DiscordUserID || byUser.RoleStatus != models.DiscordStatusConnected {
		t.Fatalf("unexpected row by user: %+v", byUser)
	}

	byDiscord, err := repo.GetByDiscordUserID(context.Background(), created.DiscordUserID)
	if err != nil {
		t.Fatalf("GetByDiscordUserID: %v", err)
	}

	if byDiscord.UserID != user.ID {
		t.Fatalf("unexpected row by discord id: %+v", byDiscord)
	}

	verifiedAt := time.Now().UTC()
	byUser.RoleStatus = models.DiscordStatusVerified
	byUser.VerifiedAt = &verifiedAt
	if err := repo.Update(context.Background(), byUser); err != nil {
		t.Fatalf("Update: %v", err)
	}

	updated, err := repo.GetByUserID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetByUserID after update: %v", err)
	}

	if updated.RoleStatus != models.DiscordStatusVerified || updated.VerifiedAt == nil {
		t.Fatalf("update not persisted: %+v", updated)
	}

	if err := repo.Delete(context.Background(), updated); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := repo.GetByUserID(context.Background(), user.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestDiscordRepoNotFound(t *testing.T) {
	env := setupRepoTest(t)
	repo := NewDiscordRepo(env.db)

	if _, err := repo.GetByUserID(context.Background(), 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByUserID: expected ErrNotFound, got %v", err)
	}

	if _, err := repo.GetByDiscordUserID(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByDiscordUserID: expected ErrNotFound, got %v", err)
	}
}

func TestDiscordRepoDuplicateDiscordUserID(t *testing.T) {
	env := setupRepoTest(t)
	repo := NewDiscordRepo(env.db)
	user1 := createUserWithNewTeam(t, env, "discord-dup1@example.com", "discord-dup1", "pass", models.UserRole)
	user2 := createUserWithNewTeam(t, env, "discord-dup2@example.com", "discord-dup2", "pass", models.UserRole)

	createDiscordConnection(t, repo, user1.ID, "900000000000000002")

	dup := &models.DiscordConnection{
		UserID:        user2.ID,
		DiscordUserID: "900000000000000002",
		RoleStatus:    models.DiscordStatusConnected,
		ConnectedAt:   time.Now().UTC(),
	}
	if err := repo.Create(context.Background(), dup); err == nil {
		t.Fatal("expected unique violation creating duplicate discord_user_id, got nil")
	}
}
