package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"smctf/internal/config"
	"smctf/internal/models"
	"smctf/internal/repo"
	"smctf/internal/utils"
	"smctf/internal/vm"

	"golang.org/x/crypto/bcrypt"
)

const testVMSpec = `apiVersion: sandboxd.o/v1
kind: Sandbox
id: placeholder
spec:
  egress: true
  ttl_seconds: 3600
  ports:
    - host_port: 0
      container_port: 31337
      protocol: tcp
  containers:
    - name: app
      image: nginx:latest
      resource:
        cpu: 50m
        memory: 64Mi
`

func createVMChallenge(t *testing.T, env serviceEnv, title string) *models.Challenge {
	t.Helper()
	spec := testVMSpec
	challenge := &models.Challenge{
		Title:       title,
		Description: "desc",
		Category:    "Web",
		Points:      100,
		VMEnabled:   true,
		VMSpec:      &spec,
		IsActive:    true,
		CreatedAt:   time.Now().UTC(),
	}

	hash, err := utils.HashFlag("flag", bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash flag: %v", err)
	}
	challenge.FlagHash = hash

	if err := env.challengeRepo.Create(context.Background(), challenge); err != nil {
		t.Fatalf("create challenge: %v", err)
	}

	return challenge
}

func newVMServiceForTest(env serviceEnv, client vm.API, cfg config.VMConfig) (*VMService, *repo.VMRepo) {
	vmRepo := repo.NewVMRepo(env.db)
	return NewVMService(cfg, vmRepo, env.challengeRepo, env.submissionRepo, client, env.redis), vmRepo
}

func TestVMServiceGetOrCreateVMRewritesManifestID(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "vm-user@example.com", "vm-user", "pass", models.UserRole)
	challenge := createVMChallenge(t, env, "vm")
	var requestedID string

	client := &vm.MockClient{
		CreateSandboxFn: func(ctx context.Context, id string, specYAML string) (*vm.Sandbox, error) {
			requestedID = id
			_, req, err := vm.RenderManifestWithID(specYAML, id)
			if err != nil {
				t.Fatalf("render manifest: %v", err)
			}

			if req.ID != id {
				t.Fatalf("manifest id not rewritten: got %q want %q", req.ID, id)
			}

			exp := time.Now().UTC().Add(time.Hour)
			return &vm.Sandbox{ID: id, Status: vm.SandboxStatus{Phase: "Pending", ExpireAt: &exp}}, nil
		},
		GetSandboxFn: func(ctx context.Context, id string) (*vm.Sandbox, error) {
			return &vm.Sandbox{ID: id, Status: vm.SandboxStatus{Phase: "Running", ExternalIP: "127.0.0.1", AssignedPorts: []vm.PortMapping{{HostPort: 31000, ContainerPort: 31337, Protocol: "tcp"}}}}, nil
		},
		DeleteSandboxFn: func(ctx context.Context, id string) error { return nil },
	}
	svc, _ := newVMServiceForTest(env, client, config.VMConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 5})

	model, err := svc.GetOrCreateVM(context.Background(), user.ID, challenge.ID)
	if err != nil {
		t.Fatalf("GetOrCreateVM: %v", err)
	}

	if model.VMID == "" || requestedID != model.VMID {
		t.Fatalf("unexpected vm id: requested=%q model=%q", requestedID, model.VMID)
	}
}

func TestVMServiceRefreshDoesNotDeleteFailedVM(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "vm-failed@example.com", "vm-failed", "pass", models.UserRole)
	challenge := createVMChallenge(t, env, "vm-failed")
	svc, vmRepo := newVMServiceForTest(env, &vm.MockClient{
		GetSandboxFn: func(ctx context.Context, id string) (*vm.Sandbox, error) {
			return &vm.Sandbox{ID: id, Status: vm.SandboxStatus{Phase: "Failed", LastError: "image pull failed"}}, nil
		},
		DeleteSandboxFn: func(ctx context.Context, id string) error { return nil },
	}, config.VMConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 5})

	now := time.Now().UTC()
	if err := vmRepo.Create(context.Background(), &models.VM{UserID: user.ID, ChallengeID: challenge.ID, VMID: "vm-failed-1", Status: "Running", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("create vm: %v", err)
	}

	got, err := svc.GetVM(context.Background(), user.ID, challenge.ID)
	if err != nil {
		t.Fatalf("GetVM: %v", err)
	}

	if got.Status != "Failed" || got.LastError == nil || *got.LastError != "image pull failed" {
		t.Fatalf("expected failed vm with error, got %+v", got)
	}

	if _, err := vmRepo.GetByVMID(context.Background(), "vm-failed-1"); err != nil {
		t.Fatalf("vm should remain in db: %v", err)
	}
}

func TestVMServiceRefreshReturnsUnavailableWithoutLastError(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "vm-unavailable@example.com", "vm-unavailable", "pass", models.UserRole)
	challenge := createVMChallenge(t, env, "vm-unavailable")
	svc, vmRepo := newVMServiceForTest(env, &vm.MockClient{
		GetSandboxFn: func(ctx context.Context, id string) (*vm.Sandbox, error) {
			return nil, vm.ErrUnavailable
		},
	}, config.VMConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 5})

	now := time.Now().UTC()
	if err := vmRepo.Create(context.Background(), &models.VM{UserID: user.ID, ChallengeID: challenge.ID, VMID: "vm-unavailable-1", Status: "Pending", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("create vm: %v", err)
	}

	if _, err := svc.GetVM(context.Background(), user.ID, challenge.ID); !errors.Is(err, ErrVMOrchestratorDown) {
		t.Fatalf("expected ErrVMOrchestratorDown, got %v", err)
	}

	got, err := vmRepo.GetByVMID(context.Background(), "vm-unavailable-1")
	if err != nil {
		t.Fatalf("vm should remain in db: %v", err)
	}

	if got.LastError != nil {
		t.Fatalf("network errors should not be stored as last error: %q", *got.LastError)
	}
}

func TestVMServiceRefreshStoresOrchestratorHTTPError(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "vm-http-error@example.com", "vm-http-error", "pass", models.UserRole)
	challenge := createVMChallenge(t, env, "vm-http-error")
	svc, vmRepo := newVMServiceForTest(env, &vm.MockClient{
		GetSandboxFn: func(ctx context.Context, id string) (*vm.Sandbox, error) {
			return nil, &vm.StatusError{StatusCode: 409, Message: "sandbox is terminating"}
		},
	}, config.VMConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 5})

	now := time.Now().UTC()
	if err := vmRepo.Create(context.Background(), &models.VM{UserID: user.ID, ChallengeID: challenge.ID, VMID: "vm-http-error-1", Status: "Running", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("create vm: %v", err)
	}

	got, err := svc.GetVM(context.Background(), user.ID, challenge.ID)
	if err != nil {
		t.Fatalf("GetVM should keep HTTP response errors in last_error: %v", err)
	}

	if got.LastError == nil || *got.LastError != "sandbox is terminating" {
		t.Fatalf("expected orchestrator message in last_error, got %+v", got.LastError)
	}

	if got.Status != "Error" {
		t.Fatalf("expected status Error on orchestrator HTTP error, got %q", got.Status)
	}
}

func TestVMServiceRefreshDeletesMissingVM(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "vm-missing@example.com", "vm-missing", "pass", models.UserRole)
	challenge := createVMChallenge(t, env, "vm-missing")
	svc, vmRepo := newVMServiceForTest(env, &vm.MockClient{
		GetSandboxFn: func(ctx context.Context, id string) (*vm.Sandbox, error) {
			return nil, &vm.StatusError{StatusCode: 404, Message: "not found"}
		},
	}, config.VMConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 5})

	now := time.Now().UTC()
	if err := vmRepo.Create(context.Background(), &models.VM{UserID: user.ID, ChallengeID: challenge.ID, VMID: "vm-missing-1", Status: "Running", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("create vm: %v", err)
	}

	if _, err := svc.GetVM(context.Background(), user.ID, challenge.ID); !errors.Is(err, ErrVMNotFound) {
		t.Fatalf("expected ErrVMNotFound, got %v", err)
	}

	if _, err := vmRepo.GetByVMID(context.Background(), "vm-missing-1"); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("expected vm row to be deleted, got %v", err)
	}
}

func TestVMServiceAdminListDoesNotRefreshRows(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "vm-admin-refresh@example.com", "vm-admin-refresh", "pass", models.UserRole)
	challenge := createVMChallenge(t, env, "vm-admin-refresh")
	svc, vmRepo := newVMServiceForTest(env, &vm.MockClient{
		GetSandboxFn: func(ctx context.Context, id string) (*vm.Sandbox, error) {
			return nil, &vm.StatusError{StatusCode: 404, Message: "not found"}
		},
	}, config.VMConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 5})

	now := time.Now().UTC()
	if err := vmRepo.Create(context.Background(), &models.VM{UserID: user.ID, ChallengeID: challenge.ID, VMID: "vm-admin-refresh-1", Status: "Running", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("create vm: %v", err)
	}

	rows, err := svc.ListAdminVMs(context.Background())
	if err != nil {
		t.Fatalf("ListAdminVMs: %v", err)
	}

	if len(rows) != 1 || rows[0].VMID != "vm-admin-refresh-1" {
		t.Fatalf("expected vm row to remain in list without refresh, got %+v", rows)
	}

	if _, err := vmRepo.GetByVMID(context.Background(), "vm-admin-refresh-1"); err != nil {
		t.Fatalf("expected vm row to remain in db without refresh, got %v", err)
	}
}
