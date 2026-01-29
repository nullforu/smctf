package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"smctf/internal/repo"
)

func TestGroupServiceCreateAndList(t *testing.T) {
	env := setupServiceTest(t)

	if _, err := env.groupSvc.CreateGroup(context.Background(), ""); err == nil {
		t.Fatalf("expected validation error")
	}

	group, err := env.groupSvc.CreateGroup(context.Background(), "Alpha")
	if err != nil {
		t.Fatalf("create group: %v", err)
	}

	if group.ID == 0 || group.Name != "Alpha" {
		t.Fatalf("unexpected group: %+v", group)
	}

	rows, err := env.groupSvc.ListGroups(context.Background())
	if err != nil {
		t.Fatalf("list groups: %v", err)
	}

	if len(rows) != 1 || rows[0].MemberCount != 0 || rows[0].TotalScore != 0 {
		t.Fatalf("unexpected group list: %+v", rows)
	}
}

func TestGroupServiceStatsMembersSolved(t *testing.T) {
	env := setupServiceTest(t)
	group := createGroup(t, env, "Alpha")
	other := createGroup(t, env, "Beta")
	user1 := createUserWithGroup(t, env, "u1@example.com", "u1", "pass", "user", &group.ID)
	user2 := createUserWithGroup(t, env, "u2@example.com", "u2", "pass", "user", &group.ID)
	_ = createUserWithGroup(t, env, "u3@example.com", "u3", "pass", "user", &other.ID)

	ch1 := createChallenge(t, env, "Ch1", 100, "flag{1}", true)
	ch2 := createChallenge(t, env, "Ch2", 50, "flag{2}", true)

	createSubmission(t, env, user1.ID, ch1.ID, true, time.Now().Add(-2*time.Minute))
	createSubmission(t, env, user2.ID, ch2.ID, true, time.Now().Add(-time.Minute))

	stats, err := env.groupSvc.GetGroup(context.Background(), group.ID)
	if err != nil {
		t.Fatalf("get group: %v", err)
	}

	if stats.MemberCount != 2 || stats.TotalScore != 150 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	members, err := env.groupSvc.ListMembers(context.Background(), group.ID)
	if err != nil {
		t.Fatalf("list members: %v", err)
	}

	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}

	solved, err := env.groupSvc.ListSolvedChallenges(context.Background(), group.ID)
	if err != nil {
		t.Fatalf("list solved: %v", err)
	}

	if len(solved) != 2 {
		t.Fatalf("expected 2 solved challenges, got %d", len(solved))
	}
}

func TestGroupServiceNotFound(t *testing.T) {
	env := setupServiceTest(t)
	_, err := env.groupSvc.GetGroup(context.Background(), 999)
	if !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGroupServiceMembersInvalidID(t *testing.T) {
	env := setupServiceTest(t)
	_, err := env.groupSvc.ListMembers(context.Background(), 0)
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestGroupServiceSolvedInvalidID(t *testing.T) {
	env := setupServiceTest(t)
	_, err := env.groupSvc.ListSolvedChallenges(context.Background(), 0)
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
