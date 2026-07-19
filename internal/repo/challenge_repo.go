package repo

import (
	"context"

	"smctf/internal/models"

	"github.com/uptrace/bun"
)

type ChallengeRepo struct {
	db *bun.DB
}

func NewChallengeRepo(db *bun.DB) *ChallengeRepo {
	return &ChallengeRepo{db: db}
}

func (r *ChallengeRepo) ListActive(ctx context.Context) ([]models.Challenge, error) {
	challenges := make([]models.Challenge, 0)

	if err := r.db.NewSelect().
		Model(&challenges).
		Where("is_active = true").
		Order("id ASC").
		Scan(ctx); err != nil {
		return nil, wrapError("challengeRepo.ListActive", err)
	}

	return challenges, nil
}

func (r *ChallengeRepo) ListAll(ctx context.Context) ([]models.Challenge, error) {
	challenges := make([]models.Challenge, 0)

	if err := r.db.NewSelect().
		Model(&challenges).
		Order("id ASC").
		Scan(ctx); err != nil {
		return nil, wrapError("challengeRepo.ListAll", err)
	}

	return challenges, nil
}

func (r *ChallengeRepo) ListByIDs(ctx context.Context, ids []int64) ([]models.Challenge, error) {
	challenges := make([]models.Challenge, 0)
	if len(ids) == 0 {
		return challenges, nil
	}

	if err := r.db.NewSelect().
		Model(&challenges).
		Where("id IN (?)", bun.In(ids)).
		Order("id ASC").
		Scan(ctx); err != nil {
		return nil, wrapError("challengeRepo.ListByIDs", err)
	}

	return challenges, nil
}

func (r *ChallengeRepo) GetByID(ctx context.Context, id int64) (*models.Challenge, error) {
	challenge := new(models.Challenge)

	if err := r.db.NewSelect().Model(challenge).Where("id = ?", id).Scan(ctx); err != nil {
		return nil, wrapNotFound("challengeRepo.GetByID", err)
	}

	return challenge, nil
}

func (r *ChallengeRepo) Create(ctx context.Context, challenge *models.Challenge) error {
	if _, err := r.db.NewInsert().Model(challenge).Exec(ctx); err != nil {
		return wrapError("challengeRepo.Create", err)
	}

	return nil
}

func (r *ChallengeRepo) Update(ctx context.Context, challenge *models.Challenge) error {
	if _, err := r.db.NewUpdate().Model(challenge).WherePK().Exec(ctx); err != nil {
		return wrapError("challengeRepo.Update", err)
	}

	return nil
}

func (r *ChallengeRepo) Delete(ctx context.Context, challenge *models.Challenge) error {
	if _, err := r.db.NewDelete().Model(challenge).WherePK().Exec(ctx); err != nil {
		return wrapError("challengeRepo.Delete", err)
	}

	return nil
}

func (r *ChallengeRepo) ImportChallenges(ctx context.Context, challenges []models.Challenge) ([]models.Challenge, error) {
	if len(challenges) == 0 {
		return []models.Challenge{}, nil
	}

	imported := make([]models.Challenge, 0, len(challenges))
	if err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		idMap := make(map[int64]int64, len(challenges))
		sourcePreviousIDs := make(map[int64]*int64, len(challenges))

		for _, item := range challenges {
			sourceID := item.ID
			sourcePreviousIDs[sourceID] = item.PreviousChallengeID

			item.ID = 0
			item.PreviousChallengeID = nil

			if _, err := tx.NewInsert().Model(&item).Exec(ctx); err != nil {
				return wrapError("challengeRepo.ImportChallenges insert", err)
			}

			idMap[sourceID] = item.ID
			imported = append(imported, item)
		}

		for i := range imported {
			sourcePreviousID := sourcePreviousIDs[challenges[i].ID]
			if sourcePreviousID == nil {
				continue
			}

			mappedPreviousID, ok := idMap[*sourcePreviousID]
			if !ok {
				continue
			}

			imported[i].PreviousChallengeID = &mappedPreviousID
			if _, err := tx.NewUpdate().Model(&imported[i]).Column("previous_challenge_id").WherePK().Exec(ctx); err != nil {
				return wrapError("challengeRepo.ImportChallenges relink", err)
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return imported, nil
}

func (r *ChallengeRepo) DynamicPoints(ctx context.Context, divisionID *int64) (map[int64]int, error) {
	points, err := dynamicPointsMap(ctx, r.db, divisionID)
	if err != nil {
		return nil, wrapError("challengeRepo.DynamicPoints", err)
	}

	return points, nil
}

func (r *ChallengeRepo) SolveCounts(ctx context.Context, divisionID *int64) (map[int64]int, error) {
	counts, err := challengeSolveCounts(ctx, r.db, divisionID)
	if err != nil {
		return nil, wrapError("challengeRepo.SolveCounts", err)
	}

	return counts, nil
}
