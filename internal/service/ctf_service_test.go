package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"smctf/internal/db"
	"smctf/internal/models"
	"smctf/internal/repo"
	"smctf/internal/stack"
	"smctf/internal/storage"
	"smctf/internal/utils"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type errorFileStore struct {
	uploadErr   error
	downloadErr error
	deleteErr   error
}

func (e errorFileStore) PresignUpload(ctx context.Context, key, contentType string) (storage.PresignedPost, error) {
	if e.uploadErr != nil {
		return storage.PresignedPost{}, e.uploadErr
	}

	return storage.PresignedPost{URL: "https://example.com/upload", Fields: map[string]string{"key": key}}, nil
}

func (e errorFileStore) PresignDownload(ctx context.Context, key, filename string) (storage.PresignedURL, error) {
	if e.downloadErr != nil {
		return storage.PresignedURL{}, e.downloadErr
	}

	return storage.PresignedURL{URL: "https://example.com/download/" + key}, nil
}

func (e errorFileStore) Delete(ctx context.Context, key string) error {
	return e.deleteErr
}

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

	challenge, err := env.ctfSvc.CreateChallenge(context.Background(), "Title", "Desc", "Misc", 100, 80, "FLAG{1}", true, false, nil, nil, nil)
	if err != nil {
		t.Fatalf("create challenge: %v", err)
	}

	if challenge.ID == 0 || challenge.Title != "Title" || !challenge.IsActive {
		t.Fatalf("unexpected challenge: %+v", challenge)
	}

	if challenge.MinimumPoints != 80 || challenge.InitialPoints != 100 {
		t.Fatalf("unexpected points metadata: %+v", challenge)
	}

	ok, err := utils.CheckFlag(challenge.FlagHash, "FLAG{1}")
	if err != nil || !ok {
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
	_, err := env.ctfSvc.CreateChallenge(context.Background(), "", "", "Nope", -1, 0, "", true, false, nil, nil, nil)

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}

	_, err = env.ctfSvc.CreateChallenge(context.Background(), "Title", "Desc", "Misc", 100, 200, "FLAG{X}", true, false, nil, nil, nil)
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error for minimum_points, got %v", err)
	}

	podSpec := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: test\nspec:\n  containers:\n    - name: app\n      image: nginx\n      ports:\n        - containerPort: 80\n"
	_, err = env.ctfSvc.CreateChallenge(context.Background(), "Stack", "Desc", "Web", 100, 80, "FLAG{S}", true, true, nil, &podSpec, nil)
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error for stack_target_ports, got %v", err)
	}

	missingPrev := int64(9999)
	_, err = env.ctfSvc.CreateChallenge(context.Background(), "Locked", "Desc", "Misc", 100, 80, "FLAG{P}", true, false, nil, nil, &missingPrev)
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error for previous_challenge_id, got %v", err)
	}
}

func TestCTFServiceStackTargetPortsValidation(t *testing.T) {
	env := setupServiceTest(t)
	podSpec := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: test\nspec:\n  containers:\n    - name: app\n      image: nginx\n      ports:\n        - containerPort: 80\n"

	invalidProtocol := stack.TargetPortSpecs{{ContainerPort: 80, Protocol: "ICMP"}}
	_, err := env.ctfSvc.CreateChallenge(context.Background(), "StackBadProto", "Desc", "Web", 100, 80, "FLAG{P1}", true, true, invalidProtocol, &podSpec, nil)
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error for stack_target_ports protocol, got %v", err)
	}

	duplicatePorts := stack.TargetPortSpecs{
		{ContainerPort: 80, Protocol: "tcp"},
		{ContainerPort: 80, Protocol: "TCP"},
	}
	_, err = env.ctfSvc.CreateChallenge(context.Background(), "StackDup", "Desc", "Web", 100, 80, "FLAG{P2}", true, true, duplicatePorts, &podSpec, nil)
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error for duplicate stack_target_ports, got %v", err)
	}

	mixedProtocols := stack.TargetPortSpecs{
		{ContainerPort: 80, Protocol: "tcp"},
		{ContainerPort: 80, Protocol: "udp"},
	}
	created, err := env.ctfSvc.CreateChallenge(context.Background(), "StackOK", "Desc", "Web", 100, 80, "FLAG{P3}", true, true, mixedProtocols, &podSpec, nil)
	if err != nil {
		t.Fatalf("expected mixed protocols to be allowed, got %v", err)
	}

	if len(created.StackTargetPorts) != 2 {
		t.Fatalf("expected 2 stack_target_ports, got %d", len(created.StackTargetPorts))
	}
}

func TestCTFServiceListChallengesDynamicPoints(t *testing.T) {
	env := setupServiceTest(t)
	team := createTeam(t, env, "Alpha")
	teamUser := createUserWithTeam(t, env, "t1@example.com", "t1", "pass", models.UserRole, team.ID)
	soloUser := createUserWithNewTeam(t, env, "s1@example.com", "s1", "pass", models.UserRole)

	challenge, err := env.ctfSvc.CreateChallenge(context.Background(), "Dynamic", "Desc", "Misc", 500, 100, "FLAG{DYN}", true, false, nil, nil, nil)
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

func TestCTFServiceGetChallengeByID(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createChallenge(t, env, "Lookup", 100, "FLAG{LOOK}", true)

	found, err := env.ctfSvc.GetChallengeByID(context.Background(), challenge.ID)
	if err != nil {
		t.Fatalf("GetChallengeByID: %v", err)
	}

	if found.ID != challenge.ID || found.Title != challenge.Title {
		t.Fatalf("unexpected challenge: %+v", found)
	}

	if _, err := env.ctfSvc.GetChallengeByID(context.Background(), 0); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}

	if _, err := env.ctfSvc.GetChallengeByID(context.Background(), 99999); !errors.Is(err, ErrChallengeNotFound) {
		t.Fatalf("expected ErrChallengeNotFound, got %v", err)
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
	updated, err := env.ctfSvc.UpdateChallenge(context.Background(), challenge.ID, &newTitle, &newDesc, &newCat, &newPoints, &newMin, nil, &newActive, nil, nil, nil, nil, false)
	if err != nil {
		t.Fatalf("update challenge: %v", err)
	}

	if updated.Title != newTitle || updated.Description != newDesc || updated.Category != newCat || updated.Points != newPoints || updated.IsActive != newActive || updated.MinimumPoints != newMin {
		t.Fatalf("unexpected updated challenge: %+v", updated)
	}

	emptyFlag := "   "
	if _, err := env.ctfSvc.UpdateChallenge(context.Background(), challenge.ID, nil, nil, nil, nil, nil, &emptyFlag, nil, nil, nil, nil, nil, false); err == nil {
		t.Fatalf("expected empty flag to be rejected")
	}

	newFlag := "FLAG{UPDATED}"
	updatedFlag, err := env.ctfSvc.UpdateChallenge(context.Background(), challenge.ID, nil, nil, nil, nil, nil, &newFlag, nil, nil, nil, nil, nil, false)
	if err != nil {
		t.Fatalf("expected flag update, got %v", err)
	}
	ok, err := utils.CheckFlag(updatedFlag.FlagHash, newFlag)
	if err != nil || !ok {
		t.Fatalf("expected updated flag hash")
	}

	badCat := "Bad"
	if _, err := env.ctfSvc.UpdateChallenge(context.Background(), challenge.ID, nil, nil, &badCat, nil, nil, nil, nil, nil, nil, nil, nil, false); err == nil {
		t.Fatalf("expected validation error")
	}

	whitespaceTitle := "   "
	if _, err := env.ctfSvc.UpdateChallenge(context.Background(), challenge.ID, &whitespaceTitle, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, false); err != nil {
		t.Fatalf("expected whitespace title to be allowed, got %v", err)
	}

	whitespaceDesc := "   "
	if _, err := env.ctfSvc.UpdateChallenge(context.Background(), challenge.ID, nil, &whitespaceDesc, nil, nil, nil, nil, nil, nil, nil, nil, nil, false); err != nil {
		t.Fatalf("expected whitespace description to be allowed, got %v", err)
	}

	whitespaceCat := "   "
	if _, err := env.ctfSvc.UpdateChallenge(context.Background(), challenge.ID, nil, nil, &whitespaceCat, nil, nil, nil, nil, nil, nil, nil, nil, false); err == nil {
		t.Fatalf("expected whitespace category to be rejected")
	}

	negPoints := -1
	if _, err := env.ctfSvc.UpdateChallenge(context.Background(), challenge.ID, nil, nil, nil, &negPoints, nil, nil, nil, nil, nil, nil, nil, false); err == nil {
		t.Fatalf("expected negative points to be rejected")
	}

	negMin := -1
	if _, err := env.ctfSvc.UpdateChallenge(context.Background(), challenge.ID, nil, nil, nil, nil, &negMin, nil, nil, nil, nil, nil, nil, false); err == nil {
		t.Fatalf("expected negative minimum_points to be rejected")
	}

	badMin := 200
	if _, err := env.ctfSvc.UpdateChallenge(context.Background(), challenge.ID, nil, nil, nil, &newPoints, &badMin, nil, nil, nil, nil, nil, nil, false); err == nil {
		t.Fatalf("expected minimum_points > points to be rejected")
	}

	prev := createChallenge(t, env, "Prev", 40, "FLAG{PREV}", true)
	prevID := prev.ID
	if _, err := env.ctfSvc.UpdateChallenge(context.Background(), challenge.ID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &prevID, true); err != nil {
		t.Fatalf("expected previous_challenge_id update, got %v", err)
	}

	if _, err := env.ctfSvc.UpdateChallenge(context.Background(), challenge.ID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &challenge.ID, true); err == nil {
		t.Fatalf("expected self previous_challenge_id to be rejected")
	}

	if _, err := env.ctfSvc.UpdateChallenge(context.Background(), challenge.ID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, true); err != nil {
		t.Fatalf("expected previous_challenge_id clear, got %v", err)
	}

	if _, err := env.ctfSvc.UpdateChallenge(context.Background(), 9999, &newTitle, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, false); !errors.Is(err, ErrChallengeNotFound) {
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
	user := createUserWithNewTeam(t, env, "u1@example.com", "u1", "pass", models.UserRole)
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
	user1 := createUserWithTeam(t, env, "t1@example.com", "t1", "pass", models.UserRole, team.ID)
	user2 := createUserWithTeam(t, env, "t2@example.com", "t2", "pass", models.UserRole, team.ID)
	teamChallenge := createChallenge(t, env, "Team", 120, "FLAG{TEAM}", true)

	if _, err := env.ctfSvc.SubmitFlag(context.Background(), user1.ID, teamChallenge.ID, "FLAG{TEAM}"); err != nil {
		t.Fatalf("team submit correct: %v", err)
	}

	correct, err = env.ctfSvc.SubmitFlag(context.Background(), user2.ID, teamChallenge.ID, "FLAG{TEAM}")
	if !errors.Is(err, ErrAlreadySolved) || !correct {
		t.Fatalf("expected teammate already solved, got %v correct %v", err, correct)
	}

	prev := createChallenge(t, env, "Prev", 30, "FLAG{PREV}", true)
	locked := createChallenge(t, env, "Locked", 60, "FLAG{LOCK}", true)
	locked.PreviousChallengeID = &prev.ID
	if err := env.challengeRepo.Update(context.Background(), locked); err != nil {
		t.Fatalf("update locked challenge: %v", err)
	}

	if _, err := env.ctfSvc.SubmitFlag(context.Background(), user.ID, locked.ID, "FLAG{LOCK}"); !errors.Is(err, ErrChallengeLocked) {
		t.Fatalf("expected ErrChallengeLocked, got %v", err)
	}

	_ = createSubmission(t, env, user.ID, prev.ID, true, time.Now().UTC())
	correct, err = env.ctfSvc.SubmitFlag(context.Background(), user.ID, locked.ID, "FLAG{LOCK}")
	if err != nil || !correct {
		t.Fatalf("expected unlocked solve, got %v correct %v", err, correct)
	}

	inactive := createChallenge(t, env, "Inactive", 50, "FLAG{5}", false)
	if _, err := env.ctfSvc.SubmitFlag(context.Background(), user.ID, inactive.ID, "FLAG{5}"); !errors.Is(err, ErrChallengeNotFound) {
		t.Fatalf("expected ErrChallengeNotFound, got %v", err)
	}
}

func TestCTFServiceSolvedChallenges(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "u1@example.com", "u1", "pass", models.UserRole)
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
	user := createUserWithNewTeam(t, env, "u1@example.com", "u1", "pass", models.UserRole)

	rows, err := env.ctfSvc.SolvedChallenges(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("solved challenges: %v", err)
	}

	if len(rows) != 0 {
		t.Fatalf("expected empty solved rows, got %+v", rows)
	}
}

func TestCTFServiceListAllSubmissions(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "sub-all@example.com", "suball", "pass", models.UserRole)
	challenge := createChallenge(t, env, "SubAll", 100, "flag{sub}", true)

	_ = createSubmission(t, env, user.ID, challenge.ID, true, time.Now().UTC().Add(-time.Minute))
	_ = createSubmission(t, env, user.ID, challenge.ID, false, time.Now().UTC())

	rows, err := env.ctfSvc.ListAllSubmissions(context.Background())
	if err != nil {
		t.Fatalf("ListAllSubmissions: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 submissions, got %d", len(rows))
	}
	if rows[0].SubmittedAt.Before(rows[1].SubmittedAt) {
		t.Fatalf("expected newest submission first")
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

func TestChallengeFileUploadChallengeNotFound(t *testing.T) {
	env := setupServiceTest(t)

	_, _, err := env.ctfSvc.RequestChallengeFileUpload(context.Background(), 9999, "bundle.zip")
	if !errors.Is(err, ErrChallengeNotFound) {
		t.Fatalf("expected ErrChallengeNotFound, got %v", err)
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

	download, err := env.ctfSvc.RequestChallengeFileDownload(context.Background(), 0, challenge.ID)
	if err != nil {
		t.Fatalf("download request: %v", err)
	}

	if download.URL == "" {
		t.Fatalf("expected download url")
	}
}

func TestChallengeFileDownloadLocked(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "locked@example.com", "locked", "pass", models.UserRole)
	prev := createChallenge(t, env, "Prev", 50, "flag{prev}", true)
	locked := createChallenge(t, env, "Locked", 100, "flag{locked}", true)
	locked.PreviousChallengeID = &prev.ID
	locked.FileKey = ptrString("bundle.zip")
	locked.FileName = ptrString("bundle.zip")
	if err := env.challengeRepo.Update(context.Background(), locked); err != nil {
		t.Fatalf("update locked challenge: %v", err)
	}

	if _, err := env.ctfSvc.RequestChallengeFileDownload(context.Background(), user.ID, locked.ID); !errors.Is(err, ErrChallengeLocked) {
		t.Fatalf("expected ErrChallengeLocked, got %v", err)
	}

	_ = createSubmission(t, env, user.ID, prev.ID, true, time.Now().UTC())
	if _, err := env.ctfSvc.RequestChallengeFileDownload(context.Background(), user.ID, locked.ID); err != nil {
		t.Fatalf("expected download after unlock, got %v", err)
	}
}

func TestCTFServiceTeamSolvedChallengeIDs(t *testing.T) {
	env := setupServiceTest(t)
	team := createTeam(t, env, "Alpha")
	user1 := createUserWithTeam(t, env, "u1@example.com", "u1", "pass", models.UserRole, team.ID)
	user2 := createUserWithTeam(t, env, "u2@example.com", "u2", "pass", models.UserRole, team.ID)
	challenge := createChallenge(t, env, "Team", 120, "FLAG{TEAM}", true)

	_ = createSubmission(t, env, user1.ID, challenge.ID, true, time.Now().UTC())

	ids, err := env.ctfSvc.TeamSolvedChallengeIDs(context.Background(), user2.ID)
	if err != nil {
		t.Fatalf("TeamSolvedChallengeIDs: %v", err)
	}

	if len(ids) != 1 {
		t.Fatalf("expected 1 solved challenge, got %d", len(ids))
	}

	if _, ok := ids[challenge.ID]; !ok {
		t.Fatalf("expected challenge to be solved for team")
	}

	ids, err = env.ctfSvc.TeamSolvedChallengeIDs(context.Background(), 0)
	if err != nil {
		t.Fatalf("expected empty ids for user 0: %v", err)
	}

	if len(ids) != 0 {
		t.Fatalf("expected empty ids for user 0, got %d", len(ids))
	}

	ctfSvc := NewCTFService(env.cfg, env.challengeRepo, nil, env.redis, nil)
	ids, err = ctfSvc.TeamSolvedChallengeIDs(context.Background(), user1.ID)
	if err != nil {
		t.Fatalf("expected empty ids when repo nil: %v", err)
	}

	if len(ids) != 0 {
		t.Fatalf("expected empty ids when repo nil, got %d", len(ids))
	}
}

func TestChallengeFileUploadPresignError(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createChallenge(t, env, "ZipTest", 100, "flag{zip}", true)

	ctfSvc := NewCTFService(env.cfg, env.challengeRepo, env.submissionRepo, env.redis, errorFileStore{uploadErr: errors.New("presign fail")})

	_, _, err := ctfSvc.RequestChallengeFileUpload(context.Background(), challenge.ID, "bundle.zip")
	if err == nil || !strings.Contains(err.Error(), "presign") {
		t.Fatalf("expected presign error, got %v", err)
	}
}

func TestChallengeFileUploadDeletePreviousError(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createChallenge(t, env, "ZipTest", 100, "flag{zip}", true)

	prevKey := uuid.NewString() + ".zip"
	challenge.FileKey = &prevKey
	challenge.FileName = ptrString("old.zip")
	now := time.Now().UTC()
	challenge.FileUploadedAt = &now
	if err := env.challengeRepo.Update(context.Background(), challenge); err != nil {
		t.Fatalf("seed update: %v", err)
	}

	ctfSvc := NewCTFService(env.cfg, env.challengeRepo, env.submissionRepo, env.redis, errorFileStore{deleteErr: errors.New("delete fail")})

	if _, _, err := ctfSvc.RequestChallengeFileUpload(context.Background(), challenge.ID, "bundle.zip"); err != nil {
		t.Fatalf("expected upload success despite delete failure, got %v", err)
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

func TestChallengeFileDownloadChallengeNotFound(t *testing.T) {
	env := setupServiceTest(t)

	_, err := env.ctfSvc.RequestChallengeFileDownload(context.Background(), 0, 9999)
	if !errors.Is(err, ErrChallengeNotFound) {
		t.Fatalf("expected ErrChallengeNotFound, got %v", err)
	}
}

func TestChallengeFileDownloadPresignError(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createChallenge(t, env, "ZipTest", 100, "flag{zip}", true)
	_, _, err := env.ctfSvc.RequestChallengeFileUpload(context.Background(), challenge.ID, "bundle.zip")
	if err != nil {
		t.Fatalf("upload request: %v", err)
	}

	ctfSvc := NewCTFService(env.cfg, env.challengeRepo, env.submissionRepo, env.redis, errorFileStore{downloadErr: errors.New("download fail")})

	_, err = ctfSvc.RequestChallengeFileDownload(context.Background(), 0, challenge.ID)
	if err == nil || !strings.Contains(err.Error(), "presign") {
		t.Fatalf("expected presign error, got %v", err)
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

	_, err = ctfSvc.RequestChallengeFileDownload(context.Background(), 0, challenge.ID)
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

func TestChallengeFileDeleteChallengeNotFound(t *testing.T) {
	env := setupServiceTest(t)

	_, err := env.ctfSvc.DeleteChallengeFile(context.Background(), 9999)
	if !errors.Is(err, ErrChallengeNotFound) {
		t.Fatalf("expected ErrChallengeNotFound, got %v", err)
	}
}

func TestChallengeFileDeleteStoreError(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createChallenge(t, env, "ZipTest", 100, "flag{zip}", true)
	_, _, err := env.ctfSvc.RequestChallengeFileUpload(context.Background(), challenge.ID, "bundle.zip")
	if err != nil {
		t.Fatalf("upload request: %v", err)
	}

	ctfSvc := NewCTFService(env.cfg, env.challengeRepo, env.submissionRepo, env.redis, errorFileStore{deleteErr: errors.New("delete fail")})

	_, err = ctfSvc.DeleteChallengeFile(context.Background(), challenge.ID)
	if err == nil || !strings.Contains(err.Error(), "delete") {
		t.Fatalf("expected delete error, got %v", err)
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

	_, err := env.ctfSvc.RequestChallengeFileDownload(context.Background(), 0, challenge.ID)
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

func TestCTFServiceStackFields(t *testing.T) {
	env := setupServiceTest(t)
	podSpec := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: test\nspec:\n  containers:\n    - name: app\n      image: nginx\n      ports:\n        - containerPort: 80\n"

	challenge, err := env.ctfSvc.CreateChallenge(context.Background(), "Stack", "Desc", "Web", 100, 80, "FLAG{STACK}", true, true, stack.TargetPortSpecs{{ContainerPort: 80, Protocol: "TCP"}}, &podSpec, nil)
	if err != nil {
		t.Fatalf("create challenge: %v", err)
	}

	if !challenge.StackEnabled || len(challenge.StackTargetPorts) != 1 || challenge.StackTargetPorts[0].ContainerPort != 80 || challenge.StackPodSpec == nil {
		t.Fatalf("unexpected stack fields: %+v", challenge)
	}

	disable := false
	updated, err := env.ctfSvc.UpdateChallenge(context.Background(), challenge.ID, nil, nil, nil, nil, nil, nil, nil, &disable, nil, nil, nil, false)
	if err != nil {
		t.Fatalf("disable stack: %v", err)
	}

	if updated.StackEnabled || len(updated.StackTargetPorts) != 0 || updated.StackPodSpec != nil {
		t.Fatalf("expected stack cleared, got %+v", updated)
	}

	newPorts := []stack.TargetPortSpec{{ContainerPort: 80, Protocol: "TCP"}}
	if _, err := env.ctfSvc.UpdateChallenge(context.Background(), challenge.ID, nil, nil, nil, nil, nil, nil, nil, nil, &newPorts, nil, nil, false); err == nil {
		t.Fatalf("expected validation error when stack disabled")
	}

	enable := true
	empty := ""
	if _, err := env.ctfSvc.UpdateChallenge(context.Background(), challenge.ID, nil, nil, nil, nil, nil, nil, nil, &enable, &newPorts, &empty, nil, false); err == nil {
		t.Fatalf("expected validation error for empty pod spec")
	} else {
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Fatalf("expected validation error, got %v", err)
		}
	}

	if _, err := env.ctfSvc.UpdateChallenge(context.Background(), challenge.ID, nil, nil, nil, nil, nil, nil, nil, &enable, nil, &podSpec, nil, false); err == nil {
		t.Fatalf("expected validation error for missing stack_target_ports when stack enabled")
	}

	outOfRangePorts := []stack.TargetPortSpec{{ContainerPort: 70000, Protocol: "TCP"}}
	if _, err := env.ctfSvc.UpdateChallenge(context.Background(), challenge.ID, nil, nil, nil, nil, nil, nil, nil, &enable, &outOfRangePorts, &podSpec, nil, false); err == nil {
		t.Fatalf("expected validation error for out-of-range port")
	}

	zeroPorts := []stack.TargetPortSpec{{ContainerPort: 0, Protocol: "TCP"}}
	if _, err := env.ctfSvc.UpdateChallenge(context.Background(), challenge.ID, nil, nil, nil, nil, nil, nil, nil, &enable, &zeroPorts, &podSpec, nil, false); err == nil {
		t.Fatalf("expected validation error for zero port")
	}
}

func ptrString(value string) *string {
	return &value
}
