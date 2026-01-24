package service

import (
	"context"
	"strconv"
	"strings"
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
	return s.challengeRepo.ListActive(ctx)
}

func (s *CTFService) CreateChallenge(ctx context.Context, title, description string, points int, flag string, active bool) (*models.Challenge, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	flag = strings.TrimSpace(flag)
	var fields []FieldError
	if title == "" {
		fields = append(fields, FieldError{Field: "title", Reason: "required"})
	}
	if description == "" {
		fields = append(fields, FieldError{Field: "description", Reason: "required"})
	}
	if flag == "" {
		fields = append(fields, FieldError{Field: "flag", Reason: "required"})
	}
	if points < 0 {
		fields = append(fields, FieldError{Field: "points", Reason: "must be >= 0"})
	}
	if len(fields) > 0 {
		return nil, NewValidationError(fields...)
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
		return nil, err
	}
	return ch, nil
}

func (s *CTFService) SubmitFlag(ctx context.Context, userID, challengeID int64, flag string) (bool, error) {
	flag = strings.TrimSpace(flag)
	if flag == "" {
		return false, NewValidationError(FieldError{Field: "flag", Reason: "required"})
	}
	if err := s.rateLimit(ctx, userID); err != nil {
		return false, err
	}

	ch, err := s.challengeRepo.GetByID(ctx, challengeID)
	if err != nil || !ch.IsActive {
		return false, ErrChallengeNotFound
	}

	already, err := s.submissionRepo.HasCorrect(ctx, userID, challengeID)
	if err != nil {
		return false, err
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
		return false, err
	}
	return correct, nil
}

func (s *CTFService) SolvedChallenges(ctx context.Context, userID int64) ([]models.SolvedChallenge, error) {
	return s.submissionRepo.SolvedChallenges(ctx, userID)
}

func (s *CTFService) rateLimit(ctx context.Context, userID int64) error {
	key := "submit:" + itoa(userID)
	pipe := s.redis.TxPipeline()
	cnt := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, s.cfg.Security.SubmissionWindow)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}
	if cnt.Val() > int64(s.cfg.Security.SubmissionMax) {
		return ErrRateLimited
	}
	return nil
}

func trimTo(v string, max int) string {
	if len(v) <= max {
		return v
	}
	return v[:max]
}

func itoa(v int64) string {
	return strconv.FormatInt(v, 10)
}
