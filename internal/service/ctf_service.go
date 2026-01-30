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

const (
	redisSubmitPrefix = "submit:"
	maxFlagLength     = 128
)

var challengeCategories = map[string]struct{}{
	"Web":         {},
	"Web3":        {},
	"Pwnable":     {},
	"Reversing":   {},
	"Crypto":      {},
	"Forensics":   {},
	"Network":     {},
	"Cloud":       {},
	"Misc":        {},
	"Programming": {},
	"Algorithms":  {},
	"Math":        {},
	"AI":          {},
	"Blockchain":  {},
}

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

func (s *CTFService) CreateChallenge(ctx context.Context, title, description, category string, points int, flag string, active bool) (*models.Challenge, error) {
	title = normalizeTrim(title)
	description = normalizeTrim(description)
	category = normalizeTrim(category)
	flag = normalizeTrim(flag)
	validator := newFieldValidator()
	validator.Required("title", title)
	validator.Required("description", description)
	validator.Required("category", category)
	validator.Required("flag", flag)
	validator.NonNegative("points", points)
	if _, ok := challengeCategories[category]; category != "" && !ok {
		validator.fields = append(validator.fields, FieldError{Field: "category", Reason: "invalid"})
	}

	if err := validator.Error(); err != nil {
		return nil, err
	}

	challenge := &models.Challenge{
		Title:       title,
		Description: description,
		Category:    category,
		Points:      points,
		FlagHash:    utils.HMACFlag(s.cfg.Security.FlagHMACSecret, flag),
		IsActive:    active,
		CreatedAt:   time.Now().UTC(),
	}

	if err := s.challengeRepo.Create(ctx, challenge); err != nil {
		return nil, fmt.Errorf("ctf.CreateChallenge: %w", err)
	}

	return challenge, nil
}

func (s *CTFService) UpdateChallenge(ctx context.Context, id int64, title, description, category *string, points *int, flag *string, active *bool) (*models.Challenge, error) {
	normalizeOptionalString := func(value *string) *string {
		if value == nil {
			return nil
		}
		normalized := normalizeTrim(*value)
		return &normalized
	}

	normalizedTitle := normalizeOptionalString(title)
	normalizedDescription := normalizeOptionalString(description)
	normalizedCategory := normalizeOptionalString(category)

	validator := newFieldValidator()
	validator.PositiveID("id", id)

	if flag != nil {
		return nil, NewValidationError(FieldError{Field: "flag", Reason: "immutable"})
	}

	if normalizedTitle != nil {
		validator.Required("title", *normalizedTitle)
	}

	if normalizedDescription != nil {
		validator.Required("description", *normalizedDescription)
	}

	if normalizedCategory != nil {
		validator.Required("category", *normalizedCategory)
		if *normalizedCategory != "" {
			if _, ok := challengeCategories[*normalizedCategory]; !ok {
				validator.fields = append(validator.fields, FieldError{Field: "category", Reason: "invalid"})
			}
		}
	}

	if points != nil {
		validator.NonNegative("points", *points)
	}

	if err := validator.Error(); err != nil {
		return nil, err
	}

	challenge, err := s.challengeRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrChallengeNotFound
		}
		return nil, fmt.Errorf("ctf.UpdateChallenge lookup: %w", err)
	}

	if normalizedTitle != nil {
		challenge.Title = *normalizedTitle
	}

	if normalizedDescription != nil {
		challenge.Description = *normalizedDescription
	}

	if normalizedCategory != nil {
		challenge.Category = *normalizedCategory
	}

	if points != nil {
		challenge.Points = *points
	}

	if active != nil {
		challenge.IsActive = *active
	}

	if err := s.challengeRepo.Update(ctx, challenge); err != nil {
		return nil, fmt.Errorf("ctf.UpdateChallenge update: %w", err)
	}

	return challenge, nil
}

func (s *CTFService) DeleteChallenge(ctx context.Context, id int64) error {
	challenge, err := s.challengeRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrChallengeNotFound
		}
		return fmt.Errorf("ctf.DeleteChallenge lookup: %w", err)
	}

	if err := s.challengeRepo.Delete(ctx, challenge); err != nil {
		return fmt.Errorf("ctf.DeleteChallenge delete: %w", err)
	}

	return nil
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

	challenge, err := s.challengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return false, ErrChallengeNotFound
		}
		return false, fmt.Errorf("ctf.SubmitFlag lookup: %w", err)
	}

	if !challenge.IsActive {
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
	correct := utils.SecureCompare(flagHash, challenge.FlagHash)

	sub := &models.Submission{
		UserID:      userID,
		ChallengeID: challengeID,
		Provided:    trimTo(flag, maxFlagLength),
		Correct:     correct,
		SubmittedAt: time.Now().UTC(),
	}

	if correct {
		inserted, err := s.submissionRepo.CreateCorrectIfNotSolvedByTeam(ctx, sub)
		if err != nil {
			return false, fmt.Errorf("ctf.SubmitFlag create: %w", err)
		}

		if !inserted {
			return true, ErrAlreadySolved
		}
	} else if err := s.submissionRepo.Create(ctx, sub); err != nil {
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
