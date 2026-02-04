package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"smctf/internal/db"

	"github.com/uptrace/bun"
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

func TestChallengeRepoDynamicPointsAndSolveCounts(t *testing.T) {
	env := setupRepoTest(t)

	team := createTeam(t, env, "Alpha")
	userTeam := createUserWithTeam(t, env, "team@example.com", "team", "pass", "user", team.ID)
	userSolo := createUser(t, env, "solo@example.com", "solo", "pass", "user")

	challenge := createChallenge(t, env, "Dynamic", 500, "FLAG{DYN}", true)
	challenge.MinimumPoints = 100
	if err := env.challengeRepo.Update(context.Background(), challenge); err != nil {
		t.Fatalf("update challenge minimum: %v", err)
	}

	other := createChallenge(t, env, "Static", 200, "FLAG{STATIC}", true)

	now := time.Now().UTC()
	createSubmission(t, env, userTeam.ID, challenge.ID, true, now.Add(-time.Minute))
	createSubmission(t, env, userSolo.ID, challenge.ID, true, now)

	points, err := env.challengeRepo.DynamicPoints(context.Background())
	if err != nil {
		t.Fatalf("DynamicPoints: %v", err)
	}

	if points[challenge.ID] != 100 {
		t.Fatalf("expected dynamic challenge to be 100, got %d", points[challenge.ID])
	}

	if points[other.ID] != other.Points {
		t.Fatalf("expected static challenge to be %d, got %d", other.Points, points[other.ID])
	}

	solveCounts, err := env.challengeRepo.SolveCounts(context.Background())
	if err != nil {
		t.Fatalf("SolveCounts: %v", err)
	}

	if solveCounts[challenge.ID] != 2 {
		t.Fatalf("expected solve count 2, got %d", solveCounts[challenge.ID])
	}

	if _, ok := solveCounts[other.ID]; ok {
		t.Fatalf("expected no solve count entry for unsolved challenge")
	}
}

func TestChallengeRepoDynamicPointsError(t *testing.T) {
	closedDB := newClosedRepoDB(t)
	repo := NewChallengeRepo(closedDB)

	if _, err := repo.DynamicPoints(context.Background()); err == nil {
		t.Fatalf("expected error from DynamicPoints")
	}
}

func newClosedRepoDB(t *testing.T) *bun.DB {
	t.Helper()
	conn, err := db.New(repoCfg.DB, "test")
	if err != nil {
		t.Fatalf("new db: %v", err)
	}

	_ = conn.Close()
	return conn
}
