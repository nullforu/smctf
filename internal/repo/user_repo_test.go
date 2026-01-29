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

	if got.GroupName == nil || *got.GroupName != "not affiliated" {
		t.Fatalf("expected default group name, got %+v", got.GroupName)
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

	if users[0].GroupName == nil || *users[0].GroupName != "not affiliated" {
		t.Fatalf("expected default group name, got %+v", users[0].GroupName)
	}
}

func TestUserRepoNotFound(t *testing.T) {
	env := setupRepoTest(t)
	_, err := env.userRepo.GetByID(context.Background(), 123)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUserRepoLeaderboardAndTimeline(t *testing.T) {
	env := setupRepoTest(t)
	group := createGroup(t, env, "Alpha")
	user1 := createUserWithGroup(t, env, "u1@example.com", "u1", "pass", "user", &group.ID)
	user2 := createUser(t, env, "u2@example.com", "u2", "pass", "user")

	ch1 := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)
	ch2 := createChallenge(t, env, "ch2", 50, "FLAG{2}", true)

	createSubmission(t, env, user1.ID, ch1.ID, true, time.Now().Add(-3*time.Minute))
	createSubmission(t, env, user1.ID, ch2.ID, true, time.Now().Add(-2*time.Minute))
	createSubmission(t, env, user2.ID, ch2.ID, false, time.Now().Add(-1*time.Minute))

	leaderboard, err := env.userRepo.Leaderboard(context.Background())
	if err != nil {
		t.Fatalf("Leaderboard: %v", err)
	}

	if len(leaderboard) != 2 {
		t.Fatalf("expected 2 leaderboard rows, got %d", len(leaderboard))
	}

	if leaderboard[0].UserID != user1.ID || leaderboard[0].Score != 150 {
		t.Fatalf("unexpected leaderboard first row: %+v", leaderboard[0])
	}

	if leaderboard[1].UserID != user2.ID || leaderboard[1].Score != 0 {
		t.Fatalf("unexpected leaderboard second row: %+v", leaderboard[1])
	}

	since := time.Now().Add(-2*time.Minute - time.Second)
	rows, err := env.userRepo.TimelineSubmissions(context.Background(), &since)
	if err != nil {
		t.Fatalf("TimelineSubmissions: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("expected 1 timeline row, got %d", len(rows))
	}

	if rows[0].UserID != user1.ID {
		t.Fatalf("unexpected timeline row: %+v", rows[0])
	}
}

func TestUserRepoGroupLeaderboardAndTimeline(t *testing.T) {
	env := setupRepoTest(t)
	groupA := createGroup(t, env, "Alpha")
	groupB := createGroup(t, env, "Beta")
	user1 := createUserWithGroup(t, env, "u1@example.com", "u1", "pass", "user", &groupA.ID)
	user2 := createUserWithGroup(t, env, "u2@example.com", "u2", "pass", "user", &groupB.ID)
	user3 := createUser(t, env, "u3@example.com", "u3", "pass", "user")

	ch1 := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)
	ch2 := createChallenge(t, env, "ch2", 50, "FLAG{2}", true)

	createSubmission(t, env, user1.ID, ch1.ID, true, time.Now().Add(-3*time.Minute))
	createSubmission(t, env, user2.ID, ch2.ID, true, time.Now().Add(-2*time.Minute))
	createSubmission(t, env, user3.ID, ch2.ID, true, time.Now().Add(-1*time.Minute))

	leaderboard, err := env.userRepo.GroupLeaderboard(context.Background())
	if err != nil {
		t.Fatalf("GroupLeaderboard: %v", err)
	}

	if len(leaderboard) != 3 {
		t.Fatalf("expected 3 group rows, got %d", len(leaderboard))
	}

	if leaderboard[0].GroupName != "Alpha" || leaderboard[0].Score != 100 {
		t.Fatalf("unexpected group leaderboard first row: %+v", leaderboard[0])
	}

	if leaderboard[2].GroupName != "not affiliated" || leaderboard[2].Score != 50 {
		t.Fatalf("unexpected group leaderboard last row: %+v", leaderboard[2])
	}

	rows, err := env.userRepo.TimelineGroupSubmissions(context.Background(), nil)
	if err != nil {
		t.Fatalf("TimelineGroupSubmissions: %v", err)
	}

	if len(rows) != 3 {
		t.Fatalf("expected 3 group timeline rows, got %d", len(rows))
	}

	if rows[0].GroupName == "" {
		t.Fatalf("expected group name in row")
	}
}

func TestUserRepoTimelineNoSince(t *testing.T) {
	env := setupRepoTest(t)
	user := createUser(t, env, "u1@example.com", "u1", "pass", "user")
	ch := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)

	createSubmission(t, env, user.ID, ch.ID, true, time.Now().Add(-time.Minute))

	rows, err := env.userRepo.TimelineSubmissions(context.Background(), nil)
	if err != nil {
		t.Fatalf("TimelineSubmissions: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
}

func TestUserRepoTimelineOrdering(t *testing.T) {
	env := setupRepoTest(t)
	user := createUser(t, env, "u1@example.com", "u1", "pass", "user")
	ch := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)

	now := time.Now().UTC()
	createSubmission(t, env, user.ID, ch.ID, true, now.Add(-2*time.Minute))
	createSubmission(t, env, user.ID, ch.ID, true, now.Add(-time.Minute))

	rows, err := env.userRepo.TimelineSubmissions(context.Background(), nil)
	if err != nil {
		t.Fatalf("TimelineSubmissions: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	if rows[0].SubmittedAt.After(rows[1].SubmittedAt) {
		t.Fatalf("expected ascending order")
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

func TestUserRepoLeaderboardTieBreak(t *testing.T) {
	env := setupRepoTest(t)
	user1 := createUser(t, env, "u1@example.com", "u1", "pass", "user")
	user2 := createUser(t, env, "u2@example.com", "u2", "pass", "user")

	ch := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)
	createSubmission(t, env, user1.ID, ch.ID, true, time.Now().Add(-time.Minute))
	createSubmission(t, env, user2.ID, ch.ID, true, time.Now().Add(-time.Minute))

	rows, err := env.userRepo.Leaderboard(context.Background())
	if err != nil {
		t.Fatalf("Leaderboard: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	if rows[0].UserID != user1.ID {
		t.Fatalf("expected lower id first in tie, got %+v", rows[0])
	}
}

func TestUserRepoTimelineIncludesUsername(t *testing.T) {
	env := setupRepoTest(t)
	user := createUser(t, env, "u1@example.com", "u1", "pass", "user")
	ch := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)

	createSubmission(t, env, user.ID, ch.ID, true, time.Now().Add(-time.Minute))

	rows, err := env.userRepo.TimelineSubmissions(context.Background(), nil)
	if err != nil {
		t.Fatalf("TimelineSubmissions: %v", err)
	}

	if rows[0].Username == "" {
		t.Fatalf("expected username in row")
	}
}

func TestUserRepoCreateDuplicateEmail(t *testing.T) {
	env := setupRepoTest(t)
	user := &models.User{
		Email:        "dup@example.com",
		Username:     "dup1",
		PasswordHash: "hash",
		Role:         "user",
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
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := env.userRepo.Create(context.Background(), user2); err == nil {
		t.Fatalf("expected error for duplicate email")
	}
}
