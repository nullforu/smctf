package handlers

import (
	"sort"
	"time"

	"smctf/internal/models"
	"smctf/internal/repo"
)

func teamSubmissions(raw []repo.RawSubmission) []models.TimelineSubmission {
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

func teamTeamSubmissions(raw []repo.RawTeamSubmission) []models.TeamTimelineSubmission {
	if len(raw) == 0 {
		return []models.TeamTimelineSubmission{}
	}

	type teamKey struct {
		teamID  int64
		hasTeam bool
		bucket  time.Time
	}

	teams := make(map[teamKey]*models.TeamTimelineSubmission)

	for _, sub := range raw {
		bucket := sub.SubmittedAt.Truncate(10 * time.Minute)
		key := teamKey{bucket: bucket}
		if sub.TeamID != nil {
			key.teamID = *sub.TeamID
			key.hasTeam = true
		}

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
				return result[i].TeamID != nil && (result[j].TeamID == nil || *result[i].TeamID < *result[j].TeamID)
			}

			return result[i].TeamName < result[j].TeamName
		}

		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result
}
