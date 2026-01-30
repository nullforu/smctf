package repo

import (
	"context"
	"testing"
	"time"
)

func TestSubmissionRepoCreateAndHasCorrect(t *testing.T) {
	env := setupRepoTest(t)
	user := createUser(t, env, "u1@example.com", "u1", "pass", "user")
	ch := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)

	if ok, err := env.submissionRepo.HasCorrect(context.Background(), user.ID, ch.ID); err != nil {
		t.Fatalf("HasCorrect: %v", err)
	} else if ok {
		t.Fatalf("expected no correct submissions")
	}

	createSubmission(t, env, user.ID, ch.ID, false, time.Now().Add(-time.Minute))
	if ok, err := env.submissionRepo.HasCorrect(context.Background(), user.ID, ch.ID); err != nil {
		t.Fatalf("HasCorrect after incorrect: %v", err)
	} else if ok {
		t.Fatalf("expected no correct submissions")
	}

	createSubmission(t, env, user.ID, ch.ID, true, time.Now())
	if ok, err := env.submissionRepo.HasCorrect(context.Background(), user.ID, ch.ID); err != nil {
		t.Fatalf("HasCorrect after correct: %v", err)
	} else if !ok {
		t.Fatalf("expected correct submission")
	}
}

func TestSubmissionRepoHasCorrectTeam(t *testing.T) {
	env := setupRepoTest(t)
	team := createTeam(t, env, "Alpha")
	user1 := createUserWithTeam(t, env, "u1@example.com", "u1", "pass", "user", &team.ID)
	user2 := createUserWithTeam(t, env, "u2@example.com", "u2", "pass", "user", &team.ID)
	ch := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)

	createSubmission(t, env, user1.ID, ch.ID, true, time.Now().UTC())

	if ok, err := env.submissionRepo.HasCorrect(context.Background(), user2.ID, ch.ID); err != nil {
		t.Fatalf("HasCorrect teammate: %v", err)
	} else if !ok {
		t.Fatalf("expected teammate solved submission")
	}
}

func TestSubmissionRepoSolvedChallenges(t *testing.T) {
	env := setupRepoTest(t)
	user := createUser(t, env, "u1@example.com", "u1", "pass", "user")
	ch1 := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)
	ch2 := createChallenge(t, env, "ch2", 50, "FLAG{2}", true)

	createSubmission(t, env, user.ID, ch1.ID, true, time.Now().Add(-2*time.Minute))
	createSubmission(t, env, user.ID, ch1.ID, true, time.Now().Add(-1*time.Minute))
	createSubmission(t, env, user.ID, ch2.ID, true, time.Now().Add(-30*time.Second))

	rows, err := env.submissionRepo.SolvedChallenges(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("SolvedChallenges: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	if rows[0].ChallengeID != ch1.ID {
		t.Fatalf("expected first solved to be ch1, got %+v", rows[0])
	}

	if rows[1].ChallengeID != ch2.ID {
		t.Fatalf("expected second solved to be ch2, got %+v", rows[1])
	}
}

func TestSubmissionRepoSolvedChallengesEmpty(t *testing.T) {
	env := setupRepoTest(t)
	rows, err := env.submissionRepo.SolvedChallenges(context.Background(), 123)
	if err != nil {
		t.Fatalf("SolvedChallenges: %v", err)
	}

	if len(rows) != 0 {
		t.Fatalf("expected empty rows, got %d", len(rows))
	}
}
