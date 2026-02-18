package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"smctf/internal/models"
	"smctf/internal/repo"
)

type ScoreboardService struct {
	repo *repo.ScoreboardRepo
}

func NewScoreboardService(scoreRepo *repo.ScoreboardRepo) *ScoreboardService {
	return &ScoreboardService{repo: scoreRepo}
}

func (s *ScoreboardService) Leaderboard(ctx context.Context) (models.LeaderboardResponse, error) {
	rows, err := s.repo.Leaderboard(ctx)
	if err != nil {
		return models.LeaderboardResponse{}, fmt.Errorf("scoreboard.Leaderboard: %w", err)
	}

	return rows, nil
}

func (s *ScoreboardService) TeamLeaderboard(ctx context.Context) (models.TeamLeaderboardResponse, error) {
	rows, err := s.repo.TeamLeaderboard(ctx)
	if err != nil {
		return models.TeamLeaderboardResponse{}, fmt.Errorf("scoreboard.TeamLeaderboard: %w", err)
	}

	return rows, nil
}

func (s *ScoreboardService) UserTimeline(ctx context.Context, since *time.Time) ([]models.TimelineSubmission, error) {
	raw, err := s.repo.TimelineSubmissions(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("scoreboard.UserTimeline: %w", err)
	}

	return aggregateUserTimeline(raw), nil
}

func (s *ScoreboardService) TeamTimeline(ctx context.Context, since *time.Time) ([]models.TeamTimelineSubmission, error) {
	raw, err := s.repo.TimelineTeamSubmissions(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("scoreboard.TeamTimeline: %w", err)
	}

	return aggregateTeamTimeline(raw), nil
}

func aggregateUserTimeline(raw []models.UserTimelineRow) []models.TimelineSubmission {
	if len(raw) == 0 {
		return []models.TimelineSubmission{}
	}

	type teamKey struct {
		userID int64
		bucket time.Time
	}

	teams := make(map[teamKey]*models.TimelineSubmission)

	for _, sub := range raw {
		bucket := sub.SubmittedAt.Truncate(10 * time.Minute)
		key := teamKey{userID: sub.UserID, bucket: bucket}

		if team, exists := teams[key]; exists {
			team.Points += sub.Points
			team.ChallengeCount++
		} else {
			teams[key] = &models.TimelineSubmission{
				Timestamp:      bucket,
				UserID:         sub.UserID,
				Username:       sub.Username,
				Points:         sub.Points,
				ChallengeCount: 1,
			}
		}
	}

	result := make([]models.TimelineSubmission, 0, len(teams))
	for _, team := range teams {
		result = append(result, *team)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Timestamp.Equal(result[j].Timestamp) {
			return result[i].UserID < result[j].UserID
		}

		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result
}

func aggregateTeamTimeline(raw []models.TeamTimelineRow) []models.TeamTimelineSubmission {
	if len(raw) == 0 {
		return []models.TeamTimelineSubmission{}
	}

	type teamKey struct {
		teamID int64
		bucket time.Time
	}

	teams := make(map[teamKey]*models.TeamTimelineSubmission)

	for _, sub := range raw {
		bucket := sub.SubmittedAt.Truncate(10 * time.Minute)
		key := teamKey{teamID: sub.TeamID, bucket: bucket}

		if team, exists := teams[key]; exists {
			team.Points += sub.Points
			team.ChallengeCount++
		} else {
			teams[key] = &models.TeamTimelineSubmission{
				Timestamp:      bucket,
				TeamID:         sub.TeamID,
				TeamName:       sub.TeamName,
				Points:         sub.Points,
				ChallengeCount: 1,
			}
		}
	}

	result := make([]models.TeamTimelineSubmission, 0, len(teams))
	for _, team := range teams {
		result = append(result, *team)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Timestamp.Equal(result[j].Timestamp) {
			if result[i].TeamName == result[j].TeamName {
				return result[i].TeamID < result[j].TeamID
			}

			return result[i].TeamName < result[j].TeamName
		}

		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result
}
