package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"smctf/internal/config"
	"smctf/internal/models"
	"smctf/internal/repo"
	"smctf/internal/vm"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type VMService struct {
	cfg            config.VMConfig
	vmRepo         *repo.VMRepo
	challengeRepo  *repo.ChallengeRepo
	submissionRepo *repo.SubmissionRepo
	client         vm.API
	redis          *redis.Client
}

func NewVMService(cfg config.VMConfig, vmRepo *repo.VMRepo, challengeRepo *repo.ChallengeRepo, submissionRepo *repo.SubmissionRepo, client vm.API, redisClient *redis.Client) *VMService {
	return &VMService{
		cfg:            cfg,
		vmRepo:         vmRepo,
		challengeRepo:  challengeRepo,
		submissionRepo: submissionRepo,
		client:         client,
		redis:          redisClient,
	}
}

func (s *VMService) UserVMSummary(ctx context.Context, userID int64) (int, int, error) {
	if !s.cfg.Enabled {
		return 0, 0, nil
	}

	limit := s.cfg.MaxPer
	if userID <= 0 {
		return 0, limit, nil
	}

	count, err := s.countByScope(ctx, userID)
	if err != nil {
		return 0, limit, err
	}

	return count, limit, nil
}

func (s *VMService) ListUserVMs(ctx context.Context, userID int64) ([]models.VM, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}

	if s.scopeIsTeam() {
		return s.vmRepo.ListByTeamUser(ctx, userID)
	}

	return s.vmRepo.ListByUser(ctx, userID)
}

func (s *VMService) ListAdminVMs(ctx context.Context) ([]models.AdminVMSummary, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}

	return s.vmRepo.ListAdmin(ctx)
}

func (s *VMService) ListAllVMs(ctx context.Context) ([]models.VM, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}

	return s.vmRepo.ListAll(ctx)
}

func (s *VMService) GetVMByVMID(ctx context.Context, vmID string) (*models.VM, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}

	existing, err := s.vmRepo.GetByVMID(ctx, vmID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrVMNotFound
		}

		return nil, fmt.Errorf("vm.GetVMByVMID lookup: %w", err)
	}

	return s.refreshVM(ctx, existing)
}

func (s *VMService) DeleteVMByVMID(ctx context.Context, vmID string) error {
	if err := s.ensureEnabled(); err != nil {
		return err
	}

	existing, err := s.vmRepo.GetByVMID(ctx, vmID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrVMNotFound
		}

		return fmt.Errorf("vm.DeleteVMByVMID lookup: %w", err)
	}

	if err := s.client.DeleteSandbox(ctx, existing.VMID); err != nil && !errors.Is(err, vm.ErrNotFound) {
		return mapVMOrchestratorError(err)
	}

	if err := s.vmRepo.Delete(ctx, existing); err != nil {
		return fmt.Errorf("vm.DeleteVMByVMID delete: %w", err)
	}

	return nil
}

func (s *VMService) GetOrCreateVM(ctx context.Context, userID, challengeID int64) (*models.VM, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}

	challenge, spec, err := s.loadChallengeSpec(ctx, challengeID)
	if err != nil {
		return nil, err
	}

	if err := s.ensureUnlocked(ctx, userID, challenge); err != nil {
		return nil, err
	}

	if err := s.ensureNotSolved(ctx, userID, challengeID); err != nil {
		return nil, err
	}

	existing, err := s.findExistingVM(ctx, userID, challengeID)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		return existing, nil
	}

	if err := s.applyRateLimit(ctx, userID); err != nil {
		return nil, err
	}

	if err := s.ensureUserLimit(ctx, userID); err != nil {
		return nil, err
	}

	return s.createVM(ctx, userID, challengeID, spec)
}

func (s *VMService) GetVM(ctx context.Context, userID, challengeID int64) (*models.VM, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}

	existing, err := s.getByScope(ctx, userID, challengeID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrVMNotFound
		}

		return nil, fmt.Errorf("vm.GetVM lookup: %w", err)
	}

	return s.refreshVM(ctx, existing)
}

func (s *VMService) DeleteVM(ctx context.Context, userID, challengeID int64) error {
	if err := s.ensureEnabled(); err != nil {
		return err
	}

	existing, err := s.getByScope(ctx, userID, challengeID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrVMNotFound
		}

		return fmt.Errorf("vm.DeleteVM lookup: %w", err)
	}

	if err := s.client.DeleteSandbox(ctx, existing.VMID); err != nil && !errors.Is(err, vm.ErrNotFound) {
		return mapVMOrchestratorError(err)
	}

	if err := s.vmRepo.Delete(ctx, existing); err != nil {
		return fmt.Errorf("vm.DeleteVM delete: %w", err)
	}

	return nil
}

func (s *VMService) DeleteVMByUserAndChallenge(ctx context.Context, userID, challengeID int64) error {
	err := s.DeleteVM(ctx, userID, challengeID)
	if errors.Is(err, ErrVMNotFound) {
		return nil
	}
	return err
}

func (s *VMService) ensureEnabled() error {
	if !s.cfg.Enabled {
		return ErrVMDisabled
	}

	return nil
}

func (s *VMService) loadChallengeSpec(ctx context.Context, challengeID int64) (*models.Challenge, string, error) {
	challenge, err := s.challengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, "", ErrChallengeNotFound
		}

		return nil, "", fmt.Errorf("vm.GetOrCreateVM challenge: %w", err)
	}

	if !challenge.VMEnabled {
		return nil, "", ErrVMNotEnabled
	}

	if challenge.VMSpec == nil || strings.TrimSpace(*challenge.VMSpec) == "" {
		return nil, "", ErrVMInvalidSpec
	}

	return challenge, *challenge.VMSpec, nil
}

func (s *VMService) ensureNotSolved(ctx context.Context, userID, challengeID int64) error {
	if s.submissionRepo == nil {
		return nil
	}

	solved, err := s.submissionRepo.HasCorrect(ctx, userID, challengeID)
	if err != nil {
		return fmt.Errorf("vm.GetOrCreateVM solved: %w", err)
	}

	if solved {
		return ErrAlreadySolved
	}

	return nil
}

func (s *VMService) ensureUnlocked(ctx context.Context, userID int64, challenge *models.Challenge) error {
	if challenge.PreviousChallengeID == nil || *challenge.PreviousChallengeID <= 0 {
		return nil
	}

	if userID <= 0 || s.submissionRepo == nil {
		return ErrChallengeLocked
	}

	solved, err := s.submissionRepo.HasCorrect(ctx, userID, *challenge.PreviousChallengeID)
	if err != nil {
		return fmt.Errorf("vm.ensureUnlocked: %w", err)
	}

	if !solved {
		return ErrChallengeLocked
	}

	return nil
}

func (s *VMService) findExistingVM(ctx context.Context, userID, challengeID int64) (*models.VM, error) {
	existing, err := s.getByScope(ctx, userID, challengeID)
	if err == nil {
		return s.refreshVM(ctx, existing)
	}

	if !errors.Is(err, repo.ErrNotFound) {
		return nil, fmt.Errorf("vm.GetOrCreateVM lookup: %w", err)
	}

	return nil, nil
}

func (s *VMService) applyRateLimit(ctx context.Context, userID int64) error {
	if s.redis == nil {
		return nil
	}

	return rateLimit(ctx, s.redis, vmRateLimitKey(userID), s.cfg.CreateWindow, s.cfg.CreateMax)
}

func (s *VMService) ensureUserLimit(ctx context.Context, userID int64) error {
	count, err := s.countByScope(ctx, userID)
	if err != nil {
		return fmt.Errorf("vm.GetOrCreateVM count: %w", err)
	}

	if count >= s.cfg.MaxPer {
		return ErrVMLimitReached
	}

	return nil
}

func (s *VMService) createVM(ctx context.Context, userID, challengeID int64, spec string) (*models.VM, error) {
	vmID := newVMID(userID, challengeID)
	sandbox, err := s.client.CreateSandbox(ctx, vmID, spec)
	if err != nil {
		return nil, mapVMOrchestratorError(err)
	}

	now := time.Now().UTC()
	model := &models.VM{
		UserID:       userID,
		ChallengeID:  challengeID,
		VMID:         vmID,
		Status:       sandbox.Status.Phase,
		NodeName:     nullIfEmpty(sandbox.Status.NodeName),
		ExternalIP:   nullIfEmpty(sandbox.Status.ExternalIP),
		Ports:        toVMPortMappings(sandbox.Status.AssignedPorts),
		TTLExpiresAt: sandbox.Status.ExpireAt,
		LastError:    nullIfEmpty(sandbox.Status.LastError),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.vmRepo.Create(ctx, model); err != nil {
		// If concurrent requests race on the unique (user_id, challenge_id) constraint,
		// delete the just-created sandbox and return the winner row.
		if isUniqueVMConflict(err) {
			_ = s.client.DeleteSandbox(ctx, vmID)
			if existing, getErr := s.vmRepo.GetByUserAndChallenge(ctx, userID, challengeID); getErr == nil {
				return existing, nil
			}
		}
		return nil, fmt.Errorf("vm.GetOrCreateVM create: %w", err)
	}

	if reloaded, reloadErr := s.vmRepo.GetByUserAndChallenge(ctx, userID, challengeID); reloadErr == nil {
		return reloaded, nil
	}

	return model, nil
}

func (s *VMService) refreshVM(ctx context.Context, existing *models.VM) (*models.VM, error) {
	sandbox, err := s.client.GetSandbox(ctx, existing.VMID)
	if err != nil {
		var statusErr *vm.StatusError
		if errors.As(err, &statusErr) && !errors.Is(err, vm.ErrUnavailable) {
			if errors.Is(err, vm.ErrNotFound) {
				if deleteErr := s.vmRepo.Delete(ctx, existing); deleteErr != nil {
					return nil, fmt.Errorf("vm.refreshVM delete missing vm row: %w", deleteErr)
				}
				return nil, ErrVMNotFound
			}

			existing.Status = "Error"
			existing.LastError = vmStringPtr(statusErr.Error())
			existing.UpdatedAt = time.Now().UTC()
			if err := s.vmRepo.Update(ctx, existing); err != nil {
				return nil, fmt.Errorf("vm.refreshVM status error update: %w", err)
			}

			return existing, nil
		}

		return nil, mapVMOrchestratorError(err)
	}

	existing.Status = sandbox.Status.Phase
	existing.NodeName = nullIfEmpty(sandbox.Status.NodeName)
	existing.ExternalIP = nullIfEmpty(sandbox.Status.ExternalIP)
	existing.Ports = toVMPortMappings(sandbox.Status.AssignedPorts)
	existing.TTLExpiresAt = sandbox.Status.ExpireAt
	existing.LastError = nullIfEmpty(sandbox.Status.LastError)
	existing.UpdatedAt = time.Now().UTC()

	if err := s.vmRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("vm.refreshVM update: %w", err)
	}

	return existing, nil
}

func mapVMOrchestratorError(err error) error {
	switch {
	case errors.Is(err, vm.ErrNotFound):
		return ErrVMNotFound
	case errors.Is(err, vm.ErrInvalid):
		return ErrVMInvalidSpec
	case errors.Is(err, vm.ErrUnavailable):
		return ErrVMOrchestratorDown
	default:
		return fmt.Errorf("vm orchestrator: %w", err)
	}
}

func toVMPortMappings(ports []vm.PortMapping) vm.PortMappings {
	if len(ports) == 0 {
		return nil
	}

	out := make(vm.PortMappings, 0, len(ports))
	for _, port := range ports {
		out = append(out, vm.PortMapping{
			HostPort:      port.HostPort,
			ContainerPort: port.ContainerPort,
			Protocol:      port.Protocol,
		})
	}

	return out
}

func newVMID(userID, challengeID int64) string {
	return fmt.Sprintf("vm-%d-%d-%s", userID, challengeID, strings.ReplaceAll(uuid.NewString(), "-", "")[:12])
}

func vmRateLimitKey(userID int64) string {
	return "vm:create:" + strconv.FormatInt(userID, 10)
}

func vmStringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	return &value
}

func nullIfEmpty(value string) *string {
	return vmStringPtr(value)
}

func isUniqueVMConflict(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique constraint")
}

func (s *VMService) scopeIsTeam() bool {
	return strings.EqualFold(strings.TrimSpace(s.cfg.MaxScope), "team")
}

func (s *VMService) countByScope(ctx context.Context, userID int64) (int, error) {
	if s.scopeIsTeam() {
		return s.vmRepo.CountByTeamUser(ctx, userID)
	}

	return s.vmRepo.CountByUser(ctx, userID)
}

func (s *VMService) getByScope(ctx context.Context, userID, challengeID int64) (*models.VM, error) {
	if s.scopeIsTeam() {
		return s.vmRepo.GetByTeamUserAndChallenge(ctx, userID, challengeID)
	}

	return s.vmRepo.GetByUserAndChallenge(ctx, userID, challengeID)
}
