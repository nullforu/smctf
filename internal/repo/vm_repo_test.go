package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"smctf/internal/models"
	vmpkg "smctf/internal/vm"
)

func createVMRow(t *testing.T, vmRepo *VMRepo, userID, challengeID int64, vmID, status string, createdAt time.Time) *models.VM {
	t.Helper()
	row := &models.VM{
		UserID:       userID,
		ChallengeID:  challengeID,
		VMID:         vmID,
		Status:       status,
		Ports:        vmpkg.PortMappings{{HostPort: 10000, ContainerPort: 31337, Protocol: "tcp"}},
		TTLExpiresAt: ptrTime(createdAt.Add(time.Hour)),
		CreatedAt:    createdAt,
		UpdatedAt:    createdAt,
	}
	if err := vmRepo.Create(context.Background(), row); err != nil {
		t.Fatalf("create vm: %v", err)
	}

	return row
}

func ptrTime(t time.Time) *time.Time { return &t }

func TestVMRepoCRUD(t *testing.T) {
	env := setupRepoTest(t)
	vmRepo := NewVMRepo(env.db)
	user := createUserWithNewTeam(t, env, "vmcrud@example.com", "vmcrud", "pass", models.UserRole)
	challenge := createChallenge(t, env, "VM CRUD", 100, "flag{vmcrud}", true)

	now := time.Now().UTC()
	created := createVMRow(t, vmRepo, user.ID, challenge.ID, "vm-crud-1", "Pending", now)

	got, err := vmRepo.GetByUserAndChallenge(context.Background(), user.ID, challenge.ID)
	if err != nil {
		t.Fatalf("GetByUserAndChallenge: %v", err)
	}

	if got.VMID != created.VMID || got.Username != user.Username || got.ChallengeTitle != challenge.Title {
		t.Fatalf("unexpected vm row: %+v", got)
	}

	got.Status = "Running"
	lastErr := "none"
	got.LastError = &lastErr
	if err := vmRepo.Update(context.Background(), got); err != nil {
		t.Fatalf("Update: %v", err)
	}

	updated, err := vmRepo.GetByVMID(context.Background(), created.VMID)
	if err != nil {
		t.Fatalf("GetByVMID: %v", err)
	}

	if updated.Status != "Running" {
		t.Fatalf("expected updated status Running, got %s", updated.Status)
	}

	if err := vmRepo.Delete(context.Background(), updated); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := vmRepo.GetByVMID(context.Background(), created.VMID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestVMRepoListAndCountByUser(t *testing.T) {
	env := setupRepoTest(t)
	vmRepo := NewVMRepo(env.db)
	user := createUserWithNewTeam(t, env, "vmlist@example.com", "vmlist", "pass", models.UserRole)
	challenge1 := createChallenge(t, env, "VM List 1", 100, "flag{vmlist1}", true)
	challenge2 := createChallenge(t, env, "VM List 2", 100, "flag{vmlist2}", true)

	now := time.Now().UTC()
	createVMRow(t, vmRepo, user.ID, challenge1.ID, "vm-old", "Running", now.Add(-time.Minute))
	createVMRow(t, vmRepo, user.ID, challenge2.ID, "vm-new", "Pending", now)

	list, err := vmRepo.ListByUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("expected 2 vm rows, got %d", len(list))
	}

	if list[0].VMID != "vm-new" {
		t.Fatalf("expected newest row first, got %+v", list)
	}

	count, err := vmRepo.CountByUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("CountByUser: %v", err)
	}

	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}
}

func TestVMRepoListAdmin(t *testing.T) {
	env := setupRepoTest(t)
	vmRepo := NewVMRepo(env.db)
	user := createUserWithNewTeam(t, env, "vmadmin@example.com", "vmadmin", "pass", models.UserRole)
	challenge := createChallenge(t, env, "VM Admin", 300, "flag{vmadmin}", true)

	createVMRow(t, vmRepo, user.ID, challenge.ID, "vm-admin-1", "Running", time.Now().UTC())

	rows, err := vmRepo.ListAdmin(context.Background())
	if err != nil {
		t.Fatalf("ListAdmin: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	if rows[0].VMID != "vm-admin-1" || rows[0].Username != user.Username || rows[0].ChallengeTitle != challenge.Title {
		t.Fatalf("unexpected admin row: %+v", rows[0])
	}
}

func TestVMRepoNotFound(t *testing.T) {
	env := setupRepoTest(t)
	vmRepo := NewVMRepo(env.db)

	if _, err := vmRepo.GetByVMID(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if _, err := vmRepo.GetByUserAndChallenge(context.Background(), 999, 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestVMRepoTeamScopeQueries(t *testing.T) {
	env := setupRepoTest(t)
	vmRepo := NewVMRepo(env.db)

	team := createTeam(t, env, "TeamScope")
	user1 := createUserWithTeam(t, env, "team1@example.com", "team1", "pass", models.UserRole, team.ID)
	user2 := createUserWithTeam(t, env, "team2@example.com", "team2", "pass", models.UserRole, team.ID)
	other := createUserWithNewTeam(t, env, "other@example.com", "other", "pass", models.UserRole)

	ch1 := createChallenge(t, env, "Team Challenge 1", 100, "flag{t1}", true)
	ch2 := createChallenge(t, env, "Team Challenge 2", 100, "flag{t2}", true)
	chOther := createChallenge(t, env, "Other Challenge", 100, "flag{o}", true)

	now := time.Now().UTC()
	createVMRow(t, vmRepo, user1.ID, ch1.ID, "vm-team-1", "Running", now.Add(-time.Minute))
	createVMRow(t, vmRepo, user2.ID, ch2.ID, "vm-team-2", "Pending", now)
	createVMRow(t, vmRepo, other.ID, chOther.ID, "vm-other-1", "Running", now.Add(-2*time.Minute))

	list, err := vmRepo.ListByTeamUser(context.Background(), user1.ID)
	if err != nil {
		t.Fatalf("ListByTeamUser: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("expected 2 team VMs, got %d", len(list))
	}

	if list[0].VMID != "vm-team-2" {
		t.Fatalf("expected newest team VM first, got %+v", list)
	}

	count, err := vmRepo.CountByTeamUser(context.Background(), user1.ID)
	if err != nil {
		t.Fatalf("CountByTeamUser: %v", err)
	}

	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}

	got, err := vmRepo.GetByTeamUserAndChallenge(context.Background(), user1.ID, ch2.ID)
	if err != nil {
		t.Fatalf("GetByTeamUserAndChallenge: %v", err)
	}

	if got.VMID != "vm-team-2" {
		t.Fatalf("expected vm-team-2, got %+v", got)
	}

	if _, err := vmRepo.GetByTeamUserAndChallenge(context.Background(), user1.ID, chOther.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for other team VM, got %v", err)
	}
}

func TestVMRepoListAll(t *testing.T) {
	env := setupRepoTest(t)
	vmRepo := NewVMRepo(env.db)

	user := createUserWithNewTeam(t, env, "all1@example.com", "all1", "pass", models.UserRole)
	user2 := createUserWithNewTeam(t, env, "all2@example.com", "all2", "pass", models.UserRole)
	ch1 := createChallenge(t, env, "All 1", 100, "flag{a1}", true)
	ch2 := createChallenge(t, env, "All 2", 100, "flag{a2}", true)

	now := time.Now().UTC()
	createVMRow(t, vmRepo, user.ID, ch1.ID, "vm-all-1", "Running", now.Add(-time.Minute))
	createVMRow(t, vmRepo, user2.ID, ch2.ID, "vm-all-2", "Pending", now)

	rows, err := vmRepo.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	if rows[0].VMID != "vm-all-2" {
		t.Fatalf("expected newest row first, got %+v", rows)
	}
}
