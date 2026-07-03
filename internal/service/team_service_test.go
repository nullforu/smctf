package service

import (
	"context"
	"errors"
	"smctf/internal/models"
	"testing"
	"time"
)

func TestTeamServiceCreateAndList(t *testing.T) {
	env := setupServiceTest(t)

	if _, err := env.teamSvc.CreateTeam(context.Background(), "", env.defaultDivisionID); err == nil {
		t.Fatalf("expected validation error")
	}

	if _, err := env.teamSvc.CreateTeam(context.Background(), "ThisNameIsWayTooLong", env.defaultDivisionID); err == nil {
		t.Fatalf("expected max-length validation error")
	}

	_, err := env.teamSvc.CreateTeam(context.Background(), "Alpha", env.defaultDivisionID+999)
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if len(ve.Fields) != 1 || ve.Fields[0].Field != "division_id" || ve.Fields[0].Reason != "not found" {
		t.Fatalf("unexpected validation error details: %+v", ve.Fields)
	}

	team, err := env.teamSvc.CreateTeam(context.Background(), "Alpha", env.defaultDivisionID)
	if err != nil {
		t.Fatalf("create team: %v", err)
	}

	if team.ID == 0 || team.Name != "Alpha" {
		t.Fatalf("unexpected team: %+v", team)
	}

	rows, err := env.teamSvc.ListTeams(context.Background(), nil)
	if err != nil {
		t.Fatalf("list teams: %v", err)
	}

	if len(rows) != 1 || rows[0].MemberCount != 0 || rows[0].TotalScore != 0 {
		t.Fatalf("unexpected team list: %+v", rows)
	}
}

func TestTeamServiceStatsMembersSolved(t *testing.T) {
	env := setupServiceTest(t)
	team := createTeam(t, env, "Alpha")
	other := createTeam(t, env, "Beta")
	user1 := createUserWithTeam(t, env, "u1@example.com", "u1", "pass", models.UserRole, team.ID)
	user2 := createUserWithTeam(t, env, "u2@example.com", "u2", "pass", models.UserRole, team.ID)
	_ = createUserWithTeam(t, env, "u3@example.com", "u3", "pass", models.UserRole, other.ID)

	ch1 := createChallenge(t, env, "Ch1", 100, "flag{1}", true)
	ch2 := createChallenge(t, env, "Ch2", 50, "flag{2}", true)

	createSubmission(t, env, user1.ID, ch1.ID, true, time.Now().Add(-2*time.Minute))
	createSubmission(t, env, user2.ID, ch2.ID, true, time.Now().Add(-time.Minute))

	stats, err := env.teamSvc.GetTeam(context.Background(), team.ID)
	if err != nil {
		t.Fatalf("get team: %v", err)
	}

	if stats.MemberCount != 2 || stats.TotalScore != 150 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	members, err := env.teamSvc.ListMembers(context.Background(), team.ID)
	if err != nil {
		t.Fatalf("list members: %v", err)
	}

	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}

	solved, err := env.teamSvc.ListSolvedChallenges(context.Background(), team.ID)
	if err != nil {
		t.Fatalf("list solved: %v", err)
	}

	if len(solved) != 2 {
		t.Fatalf("expected 2 solved challenges, got %d", len(solved))
	}
}

func TestTeamServiceNotFound(t *testing.T) {
	env := setupServiceTest(t)
	_, err := env.teamSvc.GetTeam(context.Background(), 999)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestTeamServiceMembersInvalidID(t *testing.T) {
	env := setupServiceTest(t)
	_, err := env.teamSvc.ListMembers(context.Background(), 0)
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestTeamServiceSolvedInvalidID(t *testing.T) {
	env := setupServiceTest(t)
	_, err := env.teamSvc.ListSolvedChallenges(context.Background(), 0)
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestTeamServiceCreateDuplicateTeam(t *testing.T) {
	env := setupServiceTest(t)

	if _, err := env.teamSvc.CreateTeam(context.Background(), "Alpha", env.defaultDivisionID); err != nil {
		t.Fatalf("create first team: %v", err)
	}

	_, err := env.teamSvc.CreateTeam(context.Background(), "Alpha", env.defaultDivisionID)
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestTeamServiceListTeamsDivisionFilterAndValidation(t *testing.T) {
	env := setupServiceTest(t)
	otherDivision := createDivision(t, env, "Other")

	_, _ = env.teamSvc.CreateTeam(context.Background(), "Alpha", env.defaultDivisionID)
	_, _ = env.teamSvc.CreateTeam(context.Background(), "Beta", otherDivision.ID)

	rows, err := env.teamSvc.ListTeams(context.Background(), &env.defaultDivisionID)
	if err != nil {
		t.Fatalf("list teams by division: %v", err)
	}

	if len(rows) != 1 || rows[0].Name != "Alpha" {
		t.Fatalf("unexpected filtered rows: %+v", rows)
	}

	invalid := int64(0)
	if _, err := env.teamSvc.ListTeams(context.Background(), &invalid); err == nil {
		t.Fatalf("expected validation error for division id")
	}
}

func TestTeamServiceGetTeamValidation(t *testing.T) {
	env := setupServiceTest(t)

	if _, err := env.teamSvc.GetTeam(context.Background(), 0); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestTeamServiceMembersAndSolvedNotFound(t *testing.T) {
	env := setupServiceTest(t)

	if _, err := env.teamSvc.ListMembers(context.Background(), 999999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if _, err := env.teamSvc.ListSolvedChallenges(context.Background(), 999999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
