package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"smctf/internal/config"
	"smctf/internal/models"
	"smctf/internal/repo"
	"smctf/internal/stack"
	"smctf/internal/storage"
	"smctf/internal/utils"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	redisSubmitPrefix = "submit:"
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

func normalizeStackTargetPorts(ports stack.TargetPortSpecs, validator *fieldValidator) stack.TargetPortSpecs {
	if len(ports) == 0 {
		validator.fields = append(validator.fields, FieldError{Field: "stack_target_ports", Reason: "required"})
		return nil
	}

	normalized := make(stack.TargetPortSpecs, 0, len(ports))
	seen := make(map[string]struct{})
	for _, port := range ports {
		if port.ContainerPort <= 0 || port.ContainerPort > 65535 {
			validator.fields = append(validator.fields, FieldError{Field: "stack_target_ports", Reason: "invalid"})
			continue
		}

		protocol := strings.ToUpper(strings.TrimSpace(port.Protocol))
		if protocol != "TCP" && protocol != "UDP" {
			validator.fields = append(validator.fields, FieldError{Field: "stack_target_ports", Reason: "invalid"})
			continue
		}

		key := fmt.Sprintf("%d/%s", port.ContainerPort, protocol)
		if _, exists := seen[key]; exists {
			validator.fields = append(validator.fields, FieldError{Field: "stack_target_ports", Reason: "invalid"})
			continue
		}
		seen[key] = struct{}{}

		normalized = append(normalized, stack.TargetPortSpec{
			ContainerPort: port.ContainerPort,
			Protocol:      protocol,
		})
	}

	return normalized
}

type CTFService struct {
	cfg            config.Config
	challengeRepo  *repo.ChallengeRepo
	submissionRepo *repo.SubmissionRepo
	redis          *redis.Client
	fileStore      storage.ChallengeFileStore
}

func NewCTFService(cfg config.Config, challengeRepo *repo.ChallengeRepo, submissionRepo *repo.SubmissionRepo, redis *redis.Client, fileStore storage.ChallengeFileStore) *CTFService {
	return &CTFService{cfg: cfg, challengeRepo: challengeRepo, submissionRepo: submissionRepo, redis: redis, fileStore: fileStore}
}

func (s *CTFService) ListChallenges(ctx context.Context, divisionID *int64) ([]models.Challenge, error) {
	challenges, err := s.challengeRepo.ListActive(ctx)

	if err != nil {
		return nil, fmt.Errorf("ctf.ListChallenges: %w", err)
	}

	ptrs := make([]*models.Challenge, 0, len(challenges))
	for i := range challenges {
		ptrs = append(ptrs, &challenges[i])
	}

	if err := s.applyDynamicPoints(ctx, ptrs, divisionID); err != nil {
		return nil, fmt.Errorf("ctf.ListChallenges score: %w", err)
	}

	return challenges, nil
}

func (s *CTFService) GetChallengeByID(ctx context.Context, id int64) (*models.Challenge, error) {
	if id <= 0 {
		return nil, ErrInvalidInput
	}

	challenge, err := s.challengeRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrChallengeNotFound
		}

		return nil, fmt.Errorf("ctf.GetChallengeByID: %w", err)
	}

	return challenge, nil
}

func (s *CTFService) CreateChallenge(ctx context.Context, title, description, category string, points int, minimumPoints int, flag string, active bool, stackEnabled bool, stackTargetPorts stack.TargetPortSpecs, stackPodSpec *string, previousChallengeID *int64) (*models.Challenge, error) {
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
	validator.NonNegative("minimum_points", minimumPoints)
	if previousChallengeID != nil {
		validator.PositiveID("previous_challenge_id", *previousChallengeID)
	}

	if minimumPoints > points {
		validator.fields = append(validator.fields, FieldError{Field: "minimum_points", Reason: "must be <= points"})
	}

	if _, ok := challengeCategories[category]; category != "" && !ok {
		validator.fields = append(validator.fields, FieldError{Field: "category", Reason: "invalid"})
	}

	if stackEnabled {
		stackTargetPorts = normalizeStackTargetPorts(stackTargetPorts, validator)
		if stackPodSpec == nil || normalizeTrim(*stackPodSpec) == "" {
			validator.fields = append(validator.fields, FieldError{Field: "stack_pod_spec", Reason: "required"})
		}
	}

	if err := validator.Error(); err != nil {
		return nil, err
	}

	if previousChallengeID != nil {
		if _, err := s.challengeRepo.GetByID(ctx, *previousChallengeID); err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				return nil, NewValidationError(FieldError{Field: "previous_challenge_id", Reason: "invalid"})
			}

			return nil, fmt.Errorf("ctf.CreateChallenge previous challenge: %w", err)
		}
	}

	podSpec := (*string)(nil)
	if stackEnabled && stackPodSpec != nil {
		trimmed := normalizeTrim(*stackPodSpec)
		podSpec = &trimmed
	} else if !stackEnabled {
		stackTargetPorts = nil
	}

	challenge := &models.Challenge{
		Title:               title,
		Description:         description,
		Category:            category,
		Points:              points,
		MinimumPoints:       minimumPoints,
		PreviousChallengeID: previousChallengeID,
		StackEnabled:        stackEnabled,
		StackTargetPorts:    stackTargetPorts,
		StackPodSpec:        podSpec,
		IsActive:            active,
		CreatedAt:           time.Now().UTC(),
	}

	flagHash, err := utils.HashFlag(flag, s.cfg.BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("ctf.CreateChallenge hash flag: %w", err)
	}

	challenge.FlagHash = flagHash

	if err := s.challengeRepo.Create(ctx, challenge); err != nil {
		return nil, fmt.Errorf("ctf.CreateChallenge: %w", err)
	}

	if err := s.applyDynamicPoints(ctx, []*models.Challenge{challenge}, nil); err != nil {
		return nil, fmt.Errorf("ctf.CreateChallenge score: %w", err)
	}

	return challenge, nil
}

func (s *CTFService) UpdateChallenge(ctx context.Context, id int64, title, description, category *string, points *int, minimumPoints *int, flag *string, active *bool, stackEnabled *bool, stackTargetPorts *[]stack.TargetPortSpec, stackPodSpec *string, previousChallengeID *int64, previousChallengeSet bool) (*models.Challenge, error) {
	validator := newFieldValidator()
	validator.PositiveID("id", id)

	var normalizedTitle *string
	if title != nil {
		value := *title
		normalizedTitle = &value
	}

	var normalizedDescription *string
	if description != nil {
		value := *description
		normalizedDescription = &value
	}

	var normalizedCategory *string
	if category != nil {
		value := strings.TrimSpace(*category)
		normalizedCategory = &value
		if value == "" {
			validator.fields = append(validator.fields, FieldError{Field: "category", Reason: "required"})
		} else if _, ok := challengeCategories[value]; !ok {
			validator.fields = append(validator.fields, FieldError{Field: "category", Reason: "invalid"})
		}
	}

	var normalizedFlag *string
	if flag != nil {
		value := strings.TrimSpace(*flag)
		if value == "" {
			validator.fields = append(validator.fields, FieldError{Field: "flag", Reason: "required"})
		} else {
			normalizedFlag = &value
		}
	}

	var normalizedPodSpec *string
	if stackPodSpec != nil {
		value := strings.TrimSpace(*stackPodSpec)
		normalizedPodSpec = &value
	}

	if points != nil {
		validator.NonNegative("points", *points)
	}

	if minimumPoints != nil {
		validator.NonNegative("minimum_points", *minimumPoints)
	}
	if previousChallengeSet && previousChallengeID != nil {
		validator.PositiveID("previous_challenge_id", *previousChallengeID)
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

	if previousChallengeSet && previousChallengeID != nil {
		if *previousChallengeID == challenge.ID {
			return nil, NewValidationError(FieldError{Field: "previous_challenge_id", Reason: "invalid"})
		}

		if _, err := s.challengeRepo.GetByID(ctx, *previousChallengeID); err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				return nil, NewValidationError(FieldError{Field: "previous_challenge_id", Reason: "invalid"})
			}

			return nil, fmt.Errorf("ctf.UpdateChallenge previous challenge: %w", err)
		}
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
	if minimumPoints != nil {
		challenge.MinimumPoints = *minimumPoints
	}

	if previousChallengeSet {
		challenge.PreviousChallengeID = previousChallengeID
	}

	if active != nil {
		challenge.IsActive = *active
	}

	if stackEnabled != nil {
		challenge.StackEnabled = *stackEnabled
		if !*stackEnabled {
			challenge.StackTargetPorts = nil
			challenge.StackPodSpec = nil
		}
	}

	if stackTargetPorts != nil {
		if !challenge.StackEnabled {
			return nil, NewValidationError(FieldError{Field: "stack_target_ports", Reason: "stack disabled"})
		}

		validator := newFieldValidator()
		normalized := normalizeStackTargetPorts(stack.TargetPortSpecs(*stackTargetPorts), validator)
		if err := validator.Error(); err != nil {
			return nil, err
		}

		challenge.StackTargetPorts = normalized
	}

	if normalizedPodSpec != nil {
		if !challenge.StackEnabled {
			return nil, NewValidationError(FieldError{Field: "stack_pod_spec", Reason: "stack disabled"})
		}

		if *normalizedPodSpec == "" {
			challenge.StackPodSpec = nil
		} else {
			challenge.StackPodSpec = normalizedPodSpec
		}
	}

	if normalizedFlag != nil {
		flagHash, err := utils.HashFlag(*normalizedFlag, s.cfg.BcryptCost)
		if err != nil {
			return nil, fmt.Errorf("ctf.UpdateChallenge hash flag: %w", err)
		}

		challenge.FlagHash = flagHash
	}

	if challenge.StackEnabled {
		if len(challenge.StackTargetPorts) == 0 {
			return nil, NewValidationError(FieldError{Field: "stack_target_ports", Reason: "required"})
		}

		if challenge.StackPodSpec == nil || normalizeTrim(*challenge.StackPodSpec) == "" {
			return nil, NewValidationError(FieldError{Field: "stack_pod_spec", Reason: "required"})
		}
	}

	if challenge.MinimumPoints > challenge.Points {
		return nil, NewValidationError(FieldError{Field: "minimum_points", Reason: "must be <= points"})
	}

	if err := s.challengeRepo.Update(ctx, challenge); err != nil {
		return nil, fmt.Errorf("ctf.UpdateChallenge update: %w", err)
	}

	if err := s.applyDynamicPoints(ctx, []*models.Challenge{challenge}, nil); err != nil {
		return nil, fmt.Errorf("ctf.UpdateChallenge score: %w", err)
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

	if err := s.ensureUnlocked(ctx, userID, challenge); err != nil {
		return false, err
	}

	if err := s.rateLimit(ctx, userID); err != nil {
		return false, err
	}

	already, err := s.submissionRepo.HasCorrect(ctx, userID, challengeID)
	if err != nil {
		return false, fmt.Errorf("ctf.SubmitFlag check: %w", err)
	}

	if already {
		return true, ErrAlreadySolved
	}

	correct, err := utils.CheckFlag(challenge.FlagHash, flag)
	if err != nil {
		return false, fmt.Errorf("ctf.SubmitFlag check flag: %w", err)
	}

	sub := &models.Submission{
		UserID:      userID,
		ChallengeID: challengeID,
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

func (s *CTFService) RequestChallengeFileUpload(ctx context.Context, id int64, filename string) (*models.Challenge, storage.PresignedPost, error) {
	filename = normalizeTrim(filename)
	validator := newFieldValidator()
	validator.PositiveID("id", id)
	validator.Required("filename", filename)

	if !strings.HasSuffix(strings.ToLower(filename), ".zip") {
		validator.fields = append(validator.fields, FieldError{Field: "filename", Reason: "must be a .zip file"})
	}

	if err := validator.Error(); err != nil {
		return nil, storage.PresignedPost{}, err
	}

	if s.fileStore == nil {
		return nil, storage.PresignedPost{}, ErrStorageUnavailable
	}

	challenge, err := s.challengeRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, storage.PresignedPost{}, ErrChallengeNotFound
		}
		return nil, storage.PresignedPost{}, fmt.Errorf("ctf.RequestChallengeFileUpload lookup: %w", err)
	}

	key := uuid.NewString() + ".zip"
	upload, err := s.fileStore.PresignUpload(ctx, key, "application/zip")
	if err != nil {
		return nil, storage.PresignedPost{}, fmt.Errorf("ctf.RequestChallengeFileUpload presign: %w", err)
	}

	previousKey := challenge.FileKey
	now := time.Now().UTC()
	challenge.FileKey = &key
	challenge.FileName = &filename
	challenge.FileUploadedAt = &now

	if err := s.challengeRepo.Update(ctx, challenge); err != nil {
		return nil, storage.PresignedPost{}, fmt.Errorf("ctf.RequestChallengeFileUpload update: %w", err)
	}

	if previousKey != nil && *previousKey != "" && s.fileStore != nil {
		// Best-effort cleanup. Keep the new file active even if cleanup fails. See https://github.com/nullforu/smctf/pull/38 for more details.
		_ = s.fileStore.Delete(ctx, *previousKey)
	}

	return challenge, upload, nil
}

func (s *CTFService) RequestChallengeFileDownload(ctx context.Context, userID, id int64) (storage.PresignedURL, error) {
	validator := newFieldValidator()
	validator.PositiveID("id", id)
	if err := validator.Error(); err != nil {
		return storage.PresignedURL{}, err
	}

	if s.fileStore == nil {
		return storage.PresignedURL{}, ErrStorageUnavailable
	}

	challenge, err := s.challengeRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return storage.PresignedURL{}, ErrChallengeNotFound
		}
		return storage.PresignedURL{}, fmt.Errorf("ctf.RequestChallengeFileDownload lookup: %w", err)
	}

	if err := s.ensureUnlocked(ctx, userID, challenge); err != nil {
		return storage.PresignedURL{}, err
	}

	if challenge.FileKey == nil || *challenge.FileKey == "" {
		return storage.PresignedURL{}, ErrChallengeFileNotFound
	}

	filename := ""
	if challenge.FileName != nil {
		filename = *challenge.FileName
	}
	download, err := s.fileStore.PresignDownload(ctx, *challenge.FileKey, filename)
	if err != nil {
		return storage.PresignedURL{}, fmt.Errorf("ctf.RequestChallengeFileDownload presign: %w", err)
	}

	return download, nil
}

func (s *CTFService) TeamSolvedChallengeIDs(ctx context.Context, userID int64) (map[int64]struct{}, error) {
	if userID <= 0 || s.submissionRepo == nil {
		return map[int64]struct{}{}, nil
	}

	ids, err := s.submissionRepo.TeamSolvedChallengeIDs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ctf.TeamSolvedChallengeIDs: %w", err)
	}

	return ids, nil
}

func (s *CTFService) ensureUnlocked(ctx context.Context, userID int64, challenge *models.Challenge) error {
	if challenge.PreviousChallengeID == nil || *challenge.PreviousChallengeID <= 0 {
		return nil
	}

	if userID <= 0 || s.submissionRepo == nil {
		return ErrChallengeLocked
	}

	solved, err := s.submissionRepo.HasCorrect(ctx, userID, *challenge.PreviousChallengeID)
	if err != nil {
		return fmt.Errorf("ctf.ensureUnlocked: %w", err)
	}

	if !solved {
		return ErrChallengeLocked
	}

	return nil
}

func (s *CTFService) DeleteChallengeFile(ctx context.Context, id int64) (*models.Challenge, error) {
	validator := newFieldValidator()
	validator.PositiveID("id", id)
	if err := validator.Error(); err != nil {
		return nil, err
	}

	if s.fileStore == nil {
		return nil, ErrStorageUnavailable
	}

	challenge, err := s.challengeRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrChallengeNotFound
		}
		return nil, fmt.Errorf("ctf.DeleteChallengeFile lookup: %w", err)
	}

	if challenge.FileKey == nil || *challenge.FileKey == "" {
		return nil, ErrChallengeFileNotFound
	}

	if err := s.fileStore.Delete(ctx, *challenge.FileKey); err != nil {
		return nil, fmt.Errorf("ctf.DeleteChallengeFile delete: %w", err)
	}

	challenge.FileKey = nil
	challenge.FileName = nil
	challenge.FileUploadedAt = nil

	if err := s.challengeRepo.Update(ctx, challenge); err != nil {
		return nil, fmt.Errorf("ctf.DeleteChallengeFile update: %w", err)
	}

	return challenge, nil
}

func (s *CTFService) SolvedChallenges(ctx context.Context, userID int64, divisionID *int64) ([]models.SolvedChallenge, error) {
	rows, err := s.submissionRepo.SolvedChallenges(ctx, userID)

	if err != nil {
		return nil, fmt.Errorf("ctf.SolvedChallenges: %w", err)
	}

	pointsMap, err := s.challengeRepo.DynamicPoints(ctx, divisionID)
	if err != nil {
		return nil, fmt.Errorf("ctf.SolvedChallenges score: %w", err)
	}

	for i := range rows {
		rows[i].Points = pointsMap[rows[i].ChallengeID]
	}

	return rows, nil
}

func (s *CTFService) ListAllSubmissions(ctx context.Context) ([]models.Submission, error) {
	rows, err := s.submissionRepo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("ctf.ListAllSubmissions: %w", err)
	}

	return rows, nil
}

func (s *CTFService) applyDynamicPoints(ctx context.Context, challenges []*models.Challenge, divisionID *int64) error {
	pointsMap, err := s.challengeRepo.DynamicPoints(ctx, divisionID)
	if err != nil {
		return err
	}

	solveCounts, err := s.challengeRepo.SolveCounts(ctx, divisionID)
	if err != nil {
		return err
	}

	for _, challenge := range challenges {
		challenge.InitialPoints = challenge.Points
		if points, ok := pointsMap[challenge.ID]; ok {
			challenge.Points = points
		}

		challenge.SolveCount = solveCounts[challenge.ID]
	}

	return nil
}
