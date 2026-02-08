package repo

import (
	"context"
	"testing"
	"time"

	"smctf/internal/models"
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
	user1 := createUserWithTeam(t, env, "u1@example.com", "u1", "pass", "user", team.ID)
	user2 := createUserWithTeam(t, env, "u2@example.com", "u2", "pass", "user", team.ID)
	ch := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)

	createSubmission(t, env, user1.ID, ch.ID, true, time.Now().UTC())

	if ok, err := env.submissionRepo.HasCorrect(context.Background(), user2.ID, ch.ID); err != nil {
		t.Fatalf("HasCorrect teammate: %v", err)
	} else if !ok {
		t.Fatalf("expected teammate solved submission")
	}
}

func TestSubmissionRepoHasCorrectDifferentTeam(t *testing.T) {
	env := setupRepoTest(t)
	teamA := createTeam(t, env, "Alpha")
	teamB := createTeam(t, env, "Beta")
	user1 := createUserWithTeam(t, env, "u1@example.com", "u1", "pass", "user", teamA.ID)
	user2 := createUserWithTeam(t, env, "u2@example.com", "u2", "pass", "user", teamB.ID)
	ch := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)

	createSubmission(t, env, user1.ID, ch.ID, true, time.Now().UTC())

	if ok, err := env.submissionRepo.HasCorrect(context.Background(), user2.ID, ch.ID); err != nil {
		t.Fatalf("HasCorrect different team: %v", err)
	} else if ok {
		t.Fatalf("expected different team to be unsolved")
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

func TestSubmissionRepoCreateCorrectIfNotSolvedByTeam(t *testing.T) {
	env := setupRepoTest(t)
	team := createTeam(t, env, "Alpha")
	user1 := createUserWithTeam(t, env, "u1@example.com", "u1", "pass", "user", team.ID)
	user2 := createUserWithTeam(t, env, "u2@example.com", "u2", "pass", "user", team.ID)
	ch := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)

	now := time.Now().UTC()
	sub1 := &models.Submission{
		UserID:      user1.ID,
		ChallengeID: ch.ID,
		Provided:    "flag{1}",
		Correct:     true,
		SubmittedAt: now,
	}

	inserted, err := env.submissionRepo.CreateCorrectIfNotSolvedByTeam(context.Background(), sub1)
	if err != nil {
		t.Fatalf("CreateCorrectIfNotSolvedByTeam: %v", err)
	}

	if !inserted {
		t.Fatalf("expected first insert to succeed")
	}

	if !sub1.IsFirstBlood {
		t.Fatalf("expected first solve to be first blood")
	}

	sub2 := &models.Submission{
		UserID:      user2.ID,
		ChallengeID: ch.ID,
		Provided:    "flag{1}",
		Correct:     true,
		SubmittedAt: now.Add(time.Second),
	}
	inserted, err = env.submissionRepo.CreateCorrectIfNotSolvedByTeam(context.Background(), sub2)

	if err != nil {
		t.Fatalf("CreateCorrectIfNotSolvedByTeam second: %v", err)
	}

	if inserted {
		t.Fatalf("expected second insert to be blocked by team solve")
	}

	count, err := env.db.NewSelect().
		Model((*models.Submission)(nil)).
		Where("challenge_id = ?", ch.ID).
		Where("correct = true").
		Count(context.Background())
	if err != nil {
		t.Fatalf("count submissions: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected 1 correct submission, got %d", count)
	}
}

func TestSubmissionRepoFirstBloodAcrossTeams(t *testing.T) {
	env := setupRepoTest(t)
	teamA := createTeam(t, env, "Alpha")
	teamB := createTeam(t, env, "Beta")
	user1 := createUserWithTeam(t, env, "u1@example.com", "u1", "pass", "user", teamA.ID)
	user2 := createUserWithTeam(t, env, "u2@example.com", "u2", "pass", "user", teamB.ID)
	ch := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)

	sub1 := &models.Submission{
		UserID:      user1.ID,
		ChallengeID: ch.ID,
		Provided:    "flag{1}",
		Correct:     true,
		SubmittedAt: time.Now().UTC(),
	}

	inserted, err := env.submissionRepo.CreateCorrectIfNotSolvedByTeam(context.Background(), sub1)
	if err != nil {
		t.Fatalf("CreateCorrectIfNotSolvedByTeam: %v", err)
	}

	if !inserted || !sub1.IsFirstBlood {
		t.Fatalf("expected first solve to be first blood, got %+v", sub1)
	}

	sub2 := &models.Submission{
		UserID:      user2.ID,
		ChallengeID: ch.ID,
		Provided:    "flag{1}",
		Correct:     true,
		SubmittedAt: time.Now().UTC().Add(time.Second),
	}
	inserted, err = env.submissionRepo.CreateCorrectIfNotSolvedByTeam(context.Background(), sub2)
	if err != nil {
		t.Fatalf("CreateCorrectIfNotSolvedByTeam second: %v", err)
	}

	if !inserted {
		t.Fatalf("expected second team solve to be inserted")
	}

	if sub2.IsFirstBlood {
		t.Fatalf("expected second solve to not be first blood")
	}

	count, err := env.db.NewSelect().
		Model((*models.Submission)(nil)).
		Where("challenge_id = ?", ch.ID).
		Where("is_first_blood = true").
		Count(context.Background())
	if err != nil {
		t.Fatalf("count first blood: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 first blood entry, got %d", count)
	}
}

func TestSubmissionRepoCreateCorrectIfNotSolvedByTeamSameUser(t *testing.T) {
	env := setupRepoTest(t)
	user := createUser(t, env, "u1@example.com", "u1", "pass", "user")
	ch := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)

	now := time.Now().UTC()
	sub1 := &models.Submission{
		UserID:      user.ID,
		ChallengeID: ch.ID,
		Provided:    "flag{1}",
		Correct:     true,
		SubmittedAt: now,
	}

	inserted, err := env.submissionRepo.CreateCorrectIfNotSolvedByTeam(context.Background(), sub1)
	if err != nil {
		t.Fatalf("CreateCorrectIfNotSolvedByTeam: %v", err)
	}

	if !inserted {
		t.Fatalf("expected insert to succeed")
	}

	sub2 := &models.Submission{
		UserID:      user.ID,
		ChallengeID: ch.ID,
		Provided:    "flag{1}",
		Correct:     true,
		SubmittedAt: now.Add(time.Second),
	}
	inserted, err = env.submissionRepo.CreateCorrectIfNotSolvedByTeam(context.Background(), sub2)
	if err != nil {
		t.Fatalf("CreateCorrectIfNotSolvedByTeam second: %v", err)
	}

	if inserted {
		t.Fatalf("expected duplicate correct to be blocked")
	}
}

func TestSubmissionRepoCreateCorrectIfNotSolvedByTeamIncorrect(t *testing.T) {
	env := setupRepoTest(t)
	user := createUser(t, env, "u1@example.com", "u1", "pass", "user")
	ch := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)

	sub := &models.Submission{
		UserID:      user.ID,
		ChallengeID: ch.ID,
		Provided:    "flag{wrong}",
		Correct:     false,
		SubmittedAt: time.Now().UTC(),
	}
	inserted, err := env.submissionRepo.CreateCorrectIfNotSolvedByTeam(context.Background(), sub)
	if err != nil {
		t.Fatalf("CreateCorrectIfNotSolvedByTeam incorrect: %v", err)
	}

	if !inserted {
		t.Fatalf("expected incorrect submission to be inserted")
	}
}
