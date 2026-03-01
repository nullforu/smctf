package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"smctf/internal/db"
	"smctf/internal/models"
	"smctf/internal/repo"
)

type DivisionService struct {
	divisionRepo *repo.DivisionRepo
}

func NewDivisionService(divisionRepo *repo.DivisionRepo) *DivisionService {
	return &DivisionService{divisionRepo: divisionRepo}
}

func (s *DivisionService) CreateDivision(ctx context.Context, name string) (*models.Division, error) {
	name = strings.TrimSpace(name)
	validator := newFieldValidator()
	validator.Required("name", name)
	if err := validator.Error(); err != nil {
		return nil, err
	}

	division := &models.Division{
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.divisionRepo.Create(ctx, division); err != nil {
		if db.IsUniqueViolation(err) {
			return nil, NewValidationError(FieldError{Field: "name", Reason: "duplicate"})
		}
		return nil, fmt.Errorf("division.CreateDivision: %w", err)
	}

	return division, nil
}

func (s *DivisionService) ListDivisions(ctx context.Context) ([]models.Division, error) {
	rows, err := s.divisionRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("division.ListDivisions: %w", err)
	}

	return rows, nil
}

func (s *DivisionService) GetDivision(ctx context.Context, id int64) (*models.Division, error) {
	validator := newFieldValidator()
	validator.PositiveID("id", id)
	if err := validator.Error(); err != nil {
		return nil, err
	}

	division, err := s.divisionRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("division.GetDivision: %w", err)
	}

	return division, nil
}
