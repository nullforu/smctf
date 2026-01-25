package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"smctf/internal/config"
	"smctf/internal/models"
	"smctf/internal/repo"
	"smctf/internal/utils"

	"github.com/redis/go-redis/v9"
)

type CTFService struct {
	cfg            config.Config
	challengeRepo  *repo.ChallengeRepo
	submissionRepo *repo.SubmissionRepo
	redis          *redis.Client
}

func NewCTFService(cfg config.Config, challengeRepo *repo.ChallengeRepo, submissionRepo *repo.SubmissionRepo, redis *redis.Client) *CTFService {
	return &CTFService{cfg: cfg, challengeRepo: challengeRepo, submissionRepo: submissionRepo, redis: redis}
}

func (s *CTFService) ListChallenges(ctx context.Context) ([]models.Challenge, error) {
	challenges, err := s.challengeRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("ctf.ListChallenges: %w", err)
	}
	return challenges, nil
}

func (s *CTFService) CreateChallenge(ctx context.Context, title, description string, points int, flag string, active bool) (*models.Challenge, error) {
	title = normalizeTrim(title)
	description = normalizeTrim(description)
	flag = normalizeTrim(flag)
	validator := newFieldValidator()
	validator.Required("title", title)
	validator.Required("description", description)
	validator.Required("flag", flag)
	validator.NonNegative("points", points)
	if err := validator.Error(); err != nil {
		return nil, err
	}

	ch := &models.Challenge{
		Title:       title,
		Description: description,
		Points:      points,
		FlagHash:    utils.HMACFlag(s.cfg.Security.FlagHMACSecret, flag),
		IsActive:    active,
		CreatedAt:   time.Now().UTC(),
	}
	if err := s.challengeRepo.Create(ctx, ch); err != nil {
		return nil, fmt.Errorf("ctf.CreateChallenge: %w", err)
	}
	return ch, nil
}

func (s *CTFService) SubmitFlag(ctx context.Context, userID, challengeID int64, flag string) (bool, error) {
	flag = normalizeTrim(flag)
	validator := newFieldValidator()
	validator.Required("flag", flag)
	validator.PositiveID("challenge_id", challengeID)
	if err := validator.Error(); err != nil {
		return false, err
	}
	if err := s.rateLimit(ctx, userID); err != nil {
		return false, err
	}

	ch, err := s.challengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return false, ErrChallengeNotFound
		}
		return false, fmt.Errorf("ctf.SubmitFlag lookup: %w", err)
	}
	if !ch.IsActive {
		return false, ErrChallengeNotFound
	}

	already, err := s.submissionRepo.HasCorrect(ctx, userID, challengeID)
	if err != nil {
		return false, fmt.Errorf("ctf.SubmitFlag check: %w", err)
	}
	if already {
		return true, ErrAlreadySolved
	}

	flagHash := utils.HMACFlag(s.cfg.Security.FlagHMACSecret, flag)
	correct := utils.SecureCompare(flagHash, ch.FlagHash)

	sub := &models.Submission{
		UserID:      userID,
		ChallengeID: challengeID,
		Provided:    trimTo(flag, 128),
		Correct:     correct,
		SubmittedAt: time.Now().UTC(),
	}
	if err := s.submissionRepo.Create(ctx, sub); err != nil {
		return false, fmt.Errorf("ctf.SubmitFlag create: %w", err)
	}
	return correct, nil
}

func (s *CTFService) SolvedChallenges(ctx context.Context, userID int64) ([]models.SolvedChallenge, error) {
	rows, err := s.submissionRepo.SolvedChallenges(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ctf.SolvedChallenges: %w", err)
	}
	return rows, nil
}
