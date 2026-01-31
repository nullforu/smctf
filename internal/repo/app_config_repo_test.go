package repo

import (
	"context"
	"testing"
)

func TestAppConfigRepoUpsertAndList(t *testing.T) {
	env := setupRepoTest(t)
	repo := NewAppConfigRepo(env.db)

	if _, err := repo.Upsert(context.Background(), "title", "My CTF"); err != nil {
		t.Fatalf("Upsert title: %v", err)
	}
	if _, err := repo.Upsert(context.Background(), "description", "Hello"); err != nil {
		t.Fatalf("Upsert description: %v", err)
	}

	rows, err := repo.GetAll(context.Background())
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	if rows[0].Key != "description" || rows[1].Key != "title" {
		t.Fatalf("expected key order description/title, got %q/%q", rows[0].Key, rows[1].Key)
	}
}

func TestAppConfigRepoUpsertMany(t *testing.T) {
	env := setupRepoTest(t)
	repo := NewAppConfigRepo(env.db)

	rows, err := repo.UpsertMany(context.Background(), map[string]string{
		"title":       "My CTF",
		"description": "Hello",
	})
	if err != nil {
		t.Fatalf("UpsertMany: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	rows, err = repo.GetAll(context.Background())
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
}
