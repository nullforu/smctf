package handlers

import (
	"sort"
	"time"
)

type rawSubmission struct {
	SubmittedAt time.Time `bun:"submitted_at"`
	UserID      int64     `bun:"user_id"`
	Username    string    `bun:"username"`
	Points      int       `bun:"points"`
}

type timelineSubmission struct {
	Timestamp      time.Time `json:"timestamp"`
	UserID         int64     `json:"user_id"`
	Username       string    `json:"username"`
	Points         int       `json:"points"`
	ChallengeCount int       `json:"challenge_count"`
}

type rawTeamSubmission struct {
	SubmittedAt time.Time `bun:"submitted_at"`
	TeamID      *int64    `bun:"team_id"`
	TeamName    string    `bun:"team_name"`
	Points      int       `bun:"points"`
}

type teamTimelineSubmission struct {
	Timestamp      time.Time `json:"timestamp"`
	TeamID         *int64    `json:"team_id,omitempty"`
	TeamName       string    `json:"team_name"`
	Points         int       `json:"points"`
	ChallengeCount int       `json:"challenge_count"`
}

func teamSubmissions(raw []rawSubmission) []timelineSubmission {
	if len(raw) == 0 {
		return []timelineSubmission{}
	}

	type teamKey struct {
		userID int64
		bucket time.Time
	}

	teams := make(map[teamKey]*timelineSubmission)

	for _, sub := range raw {
		bucket := sub.SubmittedAt.Truncate(10 * time.Minute)
		key := teamKey{userID: sub.UserID, bucket: bucket}

		if team, exists := teams[key]; exists {
			team.Points += sub.Points
			team.ChallengeCount++
		} else {
			teams[key] = &timelineSubmission{
				Timestamp:      bucket,
				UserID:         sub.UserID,
				Username:       sub.Username,
				Points:         sub.Points,
				ChallengeCount: 1,
			}
		}
	}

	result := make([]timelineSubmission, 0, len(teams))
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

func teamTeamSubmissions(raw []rawTeamSubmission) []teamTimelineSubmission {
	if len(raw) == 0 {
		return []teamTimelineSubmission{}
	}

	type teamKey struct {
		teamID  int64
		hasTeam bool
		bucket  time.Time
	}

	teams := make(map[teamKey]*teamTimelineSubmission)

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
			teams[key] = &teamTimelineSubmission{
				Timestamp:      bucket,
				TeamID:         sub.TeamID,
				TeamName:       sub.TeamName,
				Points:         sub.Points,
				ChallengeCount: 1,
			}
		}
	}

	result := make([]teamTimelineSubmission, 0, len(teams))
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
