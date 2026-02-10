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
	"smctf/internal/stack"

	"github.com/redis/go-redis/v9"
)

type StackService struct {
	cfg            config.StackConfig
	stackRepo      *repo.StackRepo
	challengeRepo  *repo.ChallengeRepo
	submissionRepo *repo.SubmissionRepo
	client         *stack.Client
	redis          *redis.Client
}

func NewStackService(cfg config.StackConfig, stackRepo *repo.StackRepo, challengeRepo *repo.ChallengeRepo, submissionRepo *repo.SubmissionRepo, client *stack.Client, redisClient *redis.Client) *StackService {
	return &StackService{
		cfg:            cfg,
		stackRepo:      stackRepo,
		challengeRepo:  challengeRepo,
		submissionRepo: submissionRepo,
		client:         client,
		redis:          redisClient,
	}
}

func (s *StackService) ListUserStacks(ctx context.Context, userID int64) ([]models.Stack, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}

	stacks, err := s.stackRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	updated := make([]models.Stack, 0, len(stacks))
	for i := range stacks {
		stackModel := stacks[i]
		refreshed, err := s.refreshStack(ctx, &stackModel)
		if err != nil {
			if errors.Is(err, ErrStackNotFound) {
				continue
			}

			return nil, err
		}

		updated = append(updated, *refreshed)
	}

	return updated, nil
}

func (s *StackService) GetOrCreateStack(ctx context.Context, userID, challengeID int64) (*models.Stack, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}

	challenge, podSpec, err := s.loadChallengeSpec(ctx, challengeID)
	if err != nil {
		return nil, err
	}

	if err := s.ensureNotSolved(ctx, userID, challengeID); err != nil {
		return nil, err
	}

	existing, err := s.findExistingStack(ctx, userID, challengeID)
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

	stackModel, err := s.createStack(ctx, userID, challengeID, challenge.StackTargetPort, podSpec)
	if err != nil {
		return nil, err
	}

	return stackModel, nil
}

func (s *StackService) GetStack(ctx context.Context, userID, challengeID int64) (*models.Stack, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}

	existing, err := s.stackRepo.GetByUserAndChallenge(ctx, userID, challengeID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrStackNotFound
		}

		return nil, fmt.Errorf("stack.GetStack lookup: %w", err)
	}

	return s.refreshStack(ctx, existing)
}

func (s *StackService) DeleteStack(ctx context.Context, userID, challengeID int64) error {
	if err := s.ensureEnabled(); err != nil {
		return err
	}

	existing, err := s.stackRepo.GetByUserAndChallenge(ctx, userID, challengeID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrStackNotFound
		}

		return fmt.Errorf("stack.DeleteStack lookup: %w", err)
	}

	if err := s.client.DeleteStack(ctx, existing.StackID); err != nil && !errors.Is(err, stack.ErrNotFound) {
		return mapProvisionerError(err)
	}

	if err := s.stackRepo.Delete(ctx, existing); err != nil {
		return fmt.Errorf("stack.DeleteStack delete: %w", err)
	}

	return nil
}

func (s *StackService) DeleteStackByUserAndChallenge(ctx context.Context, userID, challengeID int64) error {
	if err := s.ensureEnabled(); err != nil {
		return err
	}

	existing, err := s.stackRepo.GetByUserAndChallenge(ctx, userID, challengeID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrStackNotFound
		}

		return fmt.Errorf("stack.DeleteStackByUserAndChallenge lookup: %w", err)
	}

	if err := s.client.DeleteStack(ctx, existing.StackID); err != nil && !errors.Is(err, stack.ErrNotFound) {
		return mapProvisionerError(err)
	}

	if err := s.stackRepo.Delete(ctx, existing); err != nil {
		return fmt.Errorf("stack.DeleteStackByUserAndChallenge delete: %w", err)
	}

	return nil
}

func (s *StackService) ensureEnabled() error {
	if !s.cfg.Enabled {
		return ErrStackDisabled
	}

	return nil
}

func (s *StackService) loadChallengeSpec(ctx context.Context, challengeID int64) (*models.Challenge, string, error) {
	challenge, err := s.challengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, "", ErrChallengeNotFound
		}

		return nil, "", fmt.Errorf("stack.GetOrCreateStack challenge: %w", err)
	}

	if !challenge.StackEnabled {
		return nil, "", ErrStackNotEnabled
	}

	podSpec := ""
	if challenge.StackPodSpec != nil {
		podSpec = *challenge.StackPodSpec
	}

	if strings.TrimSpace(podSpec) == "" || challenge.StackTargetPort <= 0 {
		return nil, "", ErrStackInvalidSpec
	}

	return challenge, podSpec, nil
}

func (s *StackService) ensureNotSolved(ctx context.Context, userID, challengeID int64) error {
	if s.submissionRepo == nil {
		return nil
	}

	solved, err := s.submissionRepo.HasCorrect(ctx, userID, challengeID)
	if err != nil {
		return fmt.Errorf("stack.GetOrCreateStack solved: %w", err)
	}

	if !solved {
		return nil
	}

	existing, err := s.stackRepo.GetByUserAndChallenge(ctx, userID, challengeID)
	if err == nil {
		_ = s.client.DeleteStack(ctx, existing.StackID)
		_ = s.stackRepo.Delete(ctx, existing)
	}

	return ErrAlreadySolved
}

func (s *StackService) findExistingStack(ctx context.Context, userID, challengeID int64) (*models.Stack, error) {
	existing, err := s.stackRepo.GetByUserAndChallenge(ctx, userID, challengeID)
	if err == nil {
		refreshed, refreshErr := s.refreshStack(ctx, existing)
		if refreshErr == nil {
			return refreshed, nil
		}

		if errors.Is(refreshErr, ErrStackNotFound) {
			return nil, nil
		}

		return nil, refreshErr
	}

	if !errors.Is(err, repo.ErrNotFound) {
		return nil, fmt.Errorf("stack.GetOrCreateStack lookup: %w", err)
	}

	return nil, nil
}

func (s *StackService) applyRateLimit(ctx context.Context, userID int64) error {
	if s.redis == nil {
		return nil
	}

	key := stackRateLimitKey(userID)
	return rateLimit(ctx, s.redis, key, s.cfg.CreateWindow, s.cfg.CreateMax)
}

func (s *StackService) ensureUserLimit(ctx context.Context, userID int64) error {
	activeStacks, err := s.ListUserStacks(ctx, userID)
	if err != nil {
		return fmt.Errorf("stack.GetOrCreateStack list: %w", err)
	}

	if len(activeStacks) >= s.cfg.MaxPerUser {
		return ErrStackLimitReached
	}

	return nil
}

func (s *StackService) createStack(ctx context.Context, userID, challengeID int64, targetPort int, podSpec string) (*models.Stack, error) {
	info, err := s.client.CreateStack(ctx, targetPort, podSpec)
	if err != nil {
		return nil, mapProvisionerError(err)
	}

	now := time.Now().UTC()
	stackModel := &models.Stack{
		UserID:       userID,
		ChallengeID:  challengeID,
		StackID:      info.StackID,
		Status:       info.Status,
		NodePublicIP: nullIfEmpty(info.NodePublicIP),
		NodePort:     intPtrOrNil(info.NodePort),
		TargetPort:   info.TargetPort,
		TTLExpiresAt: timePtr(info.TTLExpiresAt),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.stackRepo.Create(ctx, stackModel); err != nil {
		return nil, fmt.Errorf("stack.GetOrCreateStack create: %w", err)
	}

	return stackModel, nil
}

func (s *StackService) refreshStack(ctx context.Context, existing *models.Stack) (*models.Stack, error) {
	status, err := s.client.GetStackStatus(ctx, existing.StackID)
	if err != nil {
		if errors.Is(err, stack.ErrNotFound) {
			_ = s.stackRepo.Delete(ctx, existing)

			return nil, ErrStackNotFound
		}

		return nil, mapProvisionerError(err)
	}

	if isTerminalStackStatus(status.Status) {
		_ = s.stackRepo.Delete(ctx, existing)
		return nil, ErrStackNotFound
	}

	existing.Status = status.Status
	existing.NodePublicIP = nullIfEmpty(status.NodePublicIP)
	existing.NodePort = intPtrOrNil(status.NodePort)
	existing.TargetPort = status.TargetPort
	existing.TTLExpiresAt = timePtr(status.TTL)
	existing.UpdatedAt = time.Now().UTC()

	if err := s.stackRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("stack.refreshStack update: %w", err)
	}

	return existing, nil
}

func isTerminalStackStatus(status string) bool {
	switch status {
	case "stopped", "failed", "node_deleted":
		return true
	default:
		return false
	}
}

func mapProvisionerError(err error) error {
	switch {
	case errors.Is(err, stack.ErrNotFound):
		return ErrStackNotFound
	case errors.Is(err, stack.ErrInvalid):
		return ErrStackInvalidSpec
	case errors.Is(err, stack.ErrUnavailable):
		return ErrStackProvisionerDown
	default:
		return fmt.Errorf("stack provisioner: %w", err)
	}
}

func nullIfEmpty(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	return &value
}

func intPtrOrNil(value int) *int {
	if value == 0 {
		return nil
	}

	return &value
}

func timePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}

	return &value
}

func stackRateLimitKey(userID int64) string {
	return "stack:create:" + strconv.FormatInt(userID, 10)
}
