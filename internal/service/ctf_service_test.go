package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"smctf/internal/db"
	"smctf/internal/repo"
	"smctf/internal/storage"
	"smctf/internal/utils"

	"github.com/uptrace/bun"
)

func newClosedServiceDB(t *testing.T) *bun.DB {
	t.Helper()
	conn, err := db.New(serviceCfg.DB, "test")
	if err != nil {
		t.Fatalf("new db: %v", err)
	}

	_ = conn.Close()
	return conn
}

func TestCTFServiceCreateAndListChallenges(t *testing.T) {
	env := setupServiceTest(t)

	challenge, err := env.ctfSvc.CreateChallenge(context.Background(), "Title", "Desc", "Misc", 100, 80, "FLAG{1}", true)
	if err != nil {
		t.Fatalf("create challenge: %v", err)
	}

	if challenge.ID == 0 || challenge.Title != "Title" || !challenge.IsActive {
		t.Fatalf("unexpected challenge: %+v", challenge)
	}

	if challenge.MinimumPoints != 80 || challenge.InitialPoints != 100 {
		t.Fatalf("unexpected points metadata: %+v", challenge)
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
	_, err := env.ctfSvc.CreateChallenge(context.Background(), "", "", "Nope", -1, 0, "", true)

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}

	_, err = env.ctfSvc.CreateChallenge(context.Background(), "Title", "Desc", "Misc", 100, 200, "FLAG{X}", true)
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error for minimum_points, got %v", err)
	}
}

func TestCTFServiceListChallengesDynamicPoints(t *testing.T) {
	env := setupServiceTest(t)
	team := createTeam(t, env, "Alpha")
	teamUser := createUserWithTeam(t, env, "t1@example.com", "t1", "pass", "user", &team.ID)
	soloUser := createUser(t, env, "s1@example.com", "s1", "pass", "user")

	challenge, err := env.ctfSvc.CreateChallenge(context.Background(), "Dynamic", "Desc", "Misc", 500, 100, "FLAG{DYN}", true)
	if err != nil {
		t.Fatalf("create challenge: %v", err)
	}

	createSubmission(t, env, teamUser.ID, challenge.ID, true, time.Now().UTC())

	list, err := env.ctfSvc.ListChallenges(context.Background())
	if err != nil {
		t.Fatalf("list challenges: %v", err)
	}

	if len(list) != 1 {
		t.Fatalf("expected 1 challenge, got %d", len(list))
	}

	if list[0].Points != 400 || list[0].InitialPoints != 500 || list[0].MinimumPoints != 100 {
		t.Fatalf("unexpected dynamic points: %+v", list[0])
	}

	createSubmission(t, env, soloUser.ID, challenge.ID, true, time.Now().UTC())
	list, err = env.ctfSvc.ListChallenges(context.Background())
	if err != nil {
		t.Fatalf("list challenges: %v", err)
	}

	if list[0].Points != 100 {
		t.Fatalf("expected minimum after 2 solves, got %d", list[0].Points)
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

	newMin := 40
	updated, err := env.ctfSvc.UpdateChallenge(context.Background(), challenge.ID, &newTitle, &newDesc, &newCat, &newPoints, &newMin, nil, &newActive)
	if err != nil {
		t.Fatalf("update challenge: %v", err)
	}

	if updated.Title != newTitle || updated.Description != newDesc || updated.Category != newCat || updated.Points != newPoints || updated.IsActive != newActive || updated.MinimumPoints != newMin {
		t.Fatalf("unexpected updated challenge: %+v", updated)
	}

	flag := "FLAG{IMMUTABLE}"
	if _, err := env.ctfSvc.UpdateChallenge(context.Background(), challenge.ID, nil, nil, nil, nil, nil, &flag, nil); err == nil {
		t.Fatalf("expected flag immutable error")
	}

	badCat := "Bad"
	if _, err := env.ctfSvc.UpdateChallenge(context.Background(), challenge.ID, nil, nil, &badCat, nil, nil, nil, nil); err == nil {
		t.Fatalf("expected validation error")
	}

	if _, err := env.ctfSvc.UpdateChallenge(context.Background(), 9999, &newTitle, nil, nil, nil, nil, nil, nil); !errors.Is(err, ErrChallengeNotFound) {
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
	user := createUser(t, env, "u1@example.com", "u1", "pass", "user")
	challenge := createChallenge(t, env, "Solve", 100, "FLAG{4}", true)

	if _, err := env.ctfSvc.SubmitFlag(context.Background(), 0, challenge.ID, "flag"); err == nil {
		t.Fatalf("expected validation error")
	}

	if _, err := env.ctfSvc.SubmitFlag(context.Background(), user.ID, 0, ""); err == nil {
		t.Fatalf("expected validation error")
	}

	correct, err := env.ctfSvc.SubmitFlag(context.Background(), user.ID, challenge.ID, "WRONG")
	if err != nil {
		t.Fatalf("submit wrong: %v", err)
	}

	if correct {
		t.Fatalf("expected incorrect submission")
	}

	correct, err = env.ctfSvc.SubmitFlag(context.Background(), user.ID, challenge.ID, "FLAG{4}")
	if err != nil {
		t.Fatalf("submit correct: %v", err)
	}

	if !correct {
		t.Fatalf("expected correct submission")
	}

	correct, err = env.ctfSvc.SubmitFlag(context.Background(), user.ID, challenge.ID, "FLAG{4}")
	if !errors.Is(err, ErrAlreadySolved) || !correct {
		t.Fatalf("expected already solved, got %v correct %v", err, correct)
	}

	team := createTeam(t, env, "Alpha")
	user1 := createUserWithTeam(t, env, "t1@example.com", "t1", "pass", "user", &team.ID)
	user2 := createUserWithTeam(t, env, "t2@example.com", "t2", "pass", "user", &team.ID)
	teamChallenge := createChallenge(t, env, "Team", 120, "FLAG{TEAM}", true)

	if _, err := env.ctfSvc.SubmitFlag(context.Background(), user1.ID, teamChallenge.ID, "FLAG{TEAM}"); err != nil {
		t.Fatalf("team submit correct: %v", err)
	}

	correct, err = env.ctfSvc.SubmitFlag(context.Background(), user2.ID, teamChallenge.ID, "FLAG{TEAM}")
	if !errors.Is(err, ErrAlreadySolved) || !correct {
		t.Fatalf("expected teammate already solved, got %v correct %v", err, correct)
	}

	inactive := createChallenge(t, env, "Inactive", 50, "FLAG{5}", false)
	if _, err := env.ctfSvc.SubmitFlag(context.Background(), user.ID, inactive.ID, "FLAG{5}"); !errors.Is(err, ErrChallengeNotFound) {
		t.Fatalf("expected ErrChallengeNotFound, got %v", err)
	}
}

func TestCTFServiceSolvedChallenges(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "u1@example.com", "u1", "pass", "user")
	challenge := createChallenge(t, env, "Solved", 100, "FLAG{6}", true)
	now := time.Now().UTC()
	_ = createSubmission(t, env, user.ID, challenge.ID, true, now.Add(-time.Minute))

	rows, err := env.ctfSvc.SolvedChallenges(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("solved challenges: %v", err)
	}

	if len(rows) != 1 || rows[0].ChallengeID != challenge.ID {
		t.Fatalf("unexpected solved rows: %+v", rows)
	}
}

func TestCTFServiceSolvedChallengesEmpty(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "u1@example.com", "u1", "pass", "user")

	rows, err := env.ctfSvc.SolvedChallenges(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("solved challenges: %v", err)
	}

	if len(rows) != 0 {
		t.Fatalf("expected empty solved rows, got %+v", rows)
	}
}

func TestCTFServiceListChallengesError(t *testing.T) {
	closedDB := newClosedServiceDB(t)
	challengeRepo := repo.NewChallengeRepo(closedDB)
	submissionRepo := repo.NewSubmissionRepo(closedDB)
	fileStore := storage.NewMemoryChallengeFileStore(10 * time.Minute)
	ctfSvc := NewCTFService(serviceCfg, challengeRepo, submissionRepo, serviceRedis, fileStore)

	if _, err := ctfSvc.ListChallenges(context.Background()); err == nil {
		t.Fatalf("expected error from ListChallenges")
	}
}

func TestCTFServiceSubmitFlagError(t *testing.T) {
	closedDB := newClosedServiceDB(t)
	challengeRepo := repo.NewChallengeRepo(closedDB)
	submissionRepo := repo.NewSubmissionRepo(closedDB)
	fileStore := storage.NewMemoryChallengeFileStore(10 * time.Minute)
	ctfSvc := NewCTFService(serviceCfg, challengeRepo, submissionRepo, serviceRedis, fileStore)

	if _, err := ctfSvc.SubmitFlag(context.Background(), 1, 1, "flag{err}"); err == nil {
		t.Fatalf("expected error from SubmitFlag")
	}
}

func TestChallengeFileUploadValidation(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createChallenge(t, env, "ZipTest", 100, "flag{zip}", true)

	_, _, err := env.ctfSvc.RequestChallengeFileUpload(context.Background(), challenge.ID, "file.txt")
	if err == nil {
		t.Fatalf("expected error")
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestChallengeFileUploadValidationBadID(t *testing.T) {
	env := setupServiceTest(t)
	_ = createChallenge(t, env, "ZipTest", 100, "flag{zip}", true)

	_, _, err := env.ctfSvc.RequestChallengeFileUpload(context.Background(), -1, "bundle.zip")
	if err == nil {
		t.Fatalf("expected error")
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestChallengeFileUploadAndDownload(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createChallenge(t, env, "ZipTest", 100, "flag{zip}", true)

	updated, upload, err := env.ctfSvc.RequestChallengeFileUpload(context.Background(), challenge.ID, "bundle.zip")
	if err != nil {
		t.Fatalf("upload request: %v", err)
	}

	if upload.URL == "" || len(upload.Fields) == 0 {
		t.Fatalf("expected upload data")
	}

	if updated.FileKey == nil || *updated.FileKey == "" {
		t.Fatalf("expected file key set")
	}

	if updated.FileName == nil || *updated.FileName != "bundle.zip" {
		t.Fatalf("expected file name set")
	}

	download, err := env.ctfSvc.RequestChallengeFileDownload(context.Background(), challenge.ID)
	if err != nil {
		t.Fatalf("download request: %v", err)
	}

	if download.URL == "" {
		t.Fatalf("expected download url")
	}
}

func TestChallengeFileUploadStorageUnavailable(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createChallenge(t, env, "ZipTest", 100, "flag{zip}", true)

	ctfSvc := NewCTFService(env.cfg, env.challengeRepo, env.submissionRepo, env.redis, nil)

	_, _, err := ctfSvc.RequestChallengeFileUpload(context.Background(), challenge.ID, "bundle.zip")
	if !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("expected ErrStorageUnavailable, got %v", err)
	}
}

func TestChallengeFileDownloadStorageUnavailable(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createChallenge(t, env, "ZipTest", 100, "flag{zip}", true)
	_, _, err := env.ctfSvc.RequestChallengeFileUpload(context.Background(), challenge.ID, "bundle.zip")
	if err != nil {
		t.Fatalf("upload request: %v", err)
	}

	ctfSvc := NewCTFService(env.cfg, env.challengeRepo, env.submissionRepo, env.redis, nil)

	_, err = ctfSvc.RequestChallengeFileDownload(context.Background(), challenge.ID)
	if !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("expected ErrStorageUnavailable, got %v", err)
	}
}

func TestChallengeFileDelete(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createChallenge(t, env, "ZipTest", 100, "flag{zip}", true)

	_, _, err := env.ctfSvc.RequestChallengeFileUpload(context.Background(), challenge.ID, "bundle.zip")
	if err != nil {
		t.Fatalf("upload request: %v", err)
	}

	updated, err := env.ctfSvc.DeleteChallengeFile(context.Background(), challenge.ID)
	if err != nil {
		t.Fatalf("delete file: %v", err)
	}

	if updated.FileKey != nil || updated.FileName != nil {
		t.Fatalf("expected file cleared")
	}
}

func TestChallengeFileDeleteStorageUnavailable(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createChallenge(t, env, "ZipTest", 100, "flag{zip}", true)
	_, _, err := env.ctfSvc.RequestChallengeFileUpload(context.Background(), challenge.ID, "bundle.zip")
	if err != nil {
		t.Fatalf("upload request: %v", err)
	}

	ctfSvc := NewCTFService(env.cfg, env.challengeRepo, env.submissionRepo, env.redis, nil)

	_, err = ctfSvc.DeleteChallengeFile(context.Background(), challenge.ID)
	if !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("expected ErrStorageUnavailable, got %v", err)
	}
}

func TestChallengeFileDownloadMissing(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createChallenge(t, env, "NoFile", 100, "flag{zip}", true)

	_, err := env.ctfSvc.RequestChallengeFileDownload(context.Background(), challenge.ID)
	if !errors.Is(err, ErrChallengeFileNotFound) {
		t.Fatalf("expected ErrChallengeFileNotFound, got %v", err)
	}
}

func TestChallengeFileDeleteMissing(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createChallenge(t, env, "NoFile", 100, "flag{zip}", true)

	_, err := env.ctfSvc.DeleteChallengeFile(context.Background(), challenge.ID)
	if !errors.Is(err, ErrChallengeFileNotFound) {
		t.Fatalf("expected ErrChallengeFileNotFound, got %v", err)
	}
}
