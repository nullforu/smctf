package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"smctf/internal/utils"
)

func TestCTFServiceCreateAndListChallenges(t *testing.T) {
	env := setupServiceTest(t)

	challenge, err := env.ctfSvc.CreateChallenge(context.Background(), "Title", "Desc", "Misc", 100, "FLAG{1}", true)
	if err != nil {
		t.Fatalf("create challenge: %v", err)
	}

	if challenge.ID == 0 || challenge.Title != "Title" || !challenge.IsActive {
		t.Fatalf("unexpected challenge: %+v", challenge)
	}

	if challenge.FlagHash != utils.HMACFlag(env.cfg.Security.FlagHMACSecret, "FLAG{1}") {
		t.Fatalf("unexpected flag hash")
	}

	list, err := env.ctfSvc.ListChallenges(context.Background())
	if err != nil {
		t.Fatalf("list challenges: %v", err)
	}

	if len(list) != 1 || list[0].ID != challenge.ID {
		t.Fatalf("unexpected list: %+v", list)
	}
}

func TestCTFServiceCreateChallengeValidation(t *testing.T) {
	env := setupServiceTest(t)
	_, err := env.ctfSvc.CreateChallenge(context.Background(), "", "", "Nope", -1, "", true)

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestCTFServiceUpdateChallenge(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createChallenge(t, env, "Old", 50, "FLAG{2}", true)

	newTitle := "New"
	newDesc := "New Desc"
	newCat := "Crypto"
	newPoints := 150
	newActive := false

	updated, err := env.ctfSvc.UpdateChallenge(context.Background(), challenge.ID, &newTitle, &newDesc, &newCat, &newPoints, nil, &newActive)
	if err != nil {
		t.Fatalf("update challenge: %v", err)
	}

	if updated.Title != newTitle || updated.Description != newDesc || updated.Category != newCat || updated.Points != newPoints || updated.IsActive != newActive {
		t.Fatalf("unexpected updated challenge: %+v", updated)
	}

	flag := "FLAG{IMMUTABLE}"
	if _, err := env.ctfSvc.UpdateChallenge(context.Background(), challenge.ID, nil, nil, nil, nil, &flag, nil); err == nil {
		t.Fatalf("expected flag immutable error")
	}

	badCat := "Bad"
	if _, err := env.ctfSvc.UpdateChallenge(context.Background(), challenge.ID, nil, nil, &badCat, nil, nil, nil); err == nil {
		t.Fatalf("expected validation error")
	}

	if _, err := env.ctfSvc.UpdateChallenge(context.Background(), 9999, &newTitle, nil, nil, nil, nil, nil); !errors.Is(err, ErrChallengeNotFound) {
		t.Fatalf("expected ErrChallengeNotFound, got %v", err)
	}
}

func TestCTFServiceDeleteChallenge(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createChallenge(t, env, "Delete", 50, "FLAG{3}", true)

	if err := env.ctfSvc.DeleteChallenge(context.Background(), challenge.ID); err != nil {
		t.Fatalf("delete challenge: %v", err)
	}

	if err := env.ctfSvc.DeleteChallenge(context.Background(), challenge.ID); !errors.Is(err, ErrChallengeNotFound) {
		t.Fatalf("expected ErrChallengeNotFound, got %v", err)
	}
}

func TestCTFServiceSubmitFlag(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createChallenge(t, env, "Solve", 100, "FLAG{4}", true)

	if _, err := env.ctfSvc.SubmitFlag(context.Background(), 0, challenge.ID, "flag"); err == nil {
		t.Fatalf("expected validation error")
	}

	if _, err := env.ctfSvc.SubmitFlag(context.Background(), 1, 0, ""); err == nil {
		t.Fatalf("expected validation error")
	}

	correct, err := env.ctfSvc.SubmitFlag(context.Background(), 1, challenge.ID, "WRONG")
	if err != nil {
		t.Fatalf("submit wrong: %v", err)
	}

	if correct {
		t.Fatalf("expected incorrect submission")
	}

	correct, err = env.ctfSvc.SubmitFlag(context.Background(), 1, challenge.ID, "FLAG{4}")
	if err != nil {
		t.Fatalf("submit correct: %v", err)
	}

	if !correct {
		t.Fatalf("expected correct submission")
	}

	correct, err = env.ctfSvc.SubmitFlag(context.Background(), 1, challenge.ID, "FLAG{4}")
	if !errors.Is(err, ErrAlreadySolved) || !correct {
		t.Fatalf("expected already solved, got %v correct %v", err, correct)
	}

	inactive := createChallenge(t, env, "Inactive", 50, "FLAG{5}", false)
	if _, err := env.ctfSvc.SubmitFlag(context.Background(), 1, inactive.ID, "FLAG{5}"); !errors.Is(err, ErrChallengeNotFound) {
		t.Fatalf("expected ErrChallengeNotFound, got %v", err)
	}
}

func TestCTFServiceSolvedChallenges(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createChallenge(t, env, "Solved", 100, "FLAG{6}", true)
	now := time.Now().UTC()
	_ = createSubmission(t, env, 1, challenge.ID, true, now.Add(-time.Minute))

	rows, err := env.ctfSvc.SolvedChallenges(context.Background(), 1)
	if err != nil {
		t.Fatalf("solved challenges: %v", err)
	}

	if len(rows) != 1 || rows[0].ChallengeID != challenge.ID {
		t.Fatalf("unexpected solved rows: %+v", rows)
	}
}
