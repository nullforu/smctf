package repo

import (
	"context"
	"errors"
	"testing"
)

func TestChallengeRepoCRUD(t *testing.T) {
	env := setupRepoTest(t)

	ch := createChallenge(t, env, "challenge", 100, "FLAG{1}", true)

	got, err := env.challengeRepo.GetByID(context.Background(), ch.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.Title != ch.Title {
		t.Fatalf("unexpected title: %s", got.Title)
	}

	list, err := env.challengeRepo.ListActive(context.Background())
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}

	if len(list) != 1 {
		t.Fatalf("expected 1 challenge, got %d", len(list))
	}

	got.Title = "updated"
	if err := env.challengeRepo.Update(context.Background(), got); err != nil {
		t.Fatalf("Update: %v", err)
	}

	updated, err := env.challengeRepo.GetByID(context.Background(), ch.ID)
	if err != nil {
		t.Fatalf("GetByID updated: %v", err)
	}

	if updated.Title != "updated" {
		t.Fatalf("expected updated title, got %s", updated.Title)
	}

	if err := env.challengeRepo.Delete(context.Background(), updated); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := env.challengeRepo.GetByID(context.Background(), ch.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestChallengeRepoNotFound(t *testing.T) {
	env := setupRepoTest(t)
	_, err := env.challengeRepo.GetByID(context.Background(), 123)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
