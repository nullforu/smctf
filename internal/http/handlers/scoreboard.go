package handlers

import (
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

func groupSubmissions(raw []rawSubmission) []timelineSubmission {
	if len(raw) == 0 {
		return []timelineSubmission{}
	}

	type groupKey struct {
		userID int64
		bucket time.Time
	}

	groups := make(map[groupKey]*timelineSubmission)

	for _, sub := range raw {
		bucket := sub.SubmittedAt.Truncate(10 * time.Minute)
		key := groupKey{userID: sub.UserID, bucket: bucket}

		if group, exists := groups[key]; exists {
			group.Points += sub.Points
			group.ChallengeCount++
		} else {
			groups[key] = &timelineSubmission{
				Timestamp:      bucket,
				UserID:         sub.UserID,
				Username:       sub.Username,
				Points:         sub.Points,
				ChallengeCount: 1,
			}
		}
	}

	result := make([]timelineSubmission, 0, len(groups))
	for _, group := range groups {
		result = append(result, *group)
	}

	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].Timestamp.After(result[j].Timestamp) ||
				(result[i].Timestamp.Equal(result[j].Timestamp) && result[i].UserID > result[j].UserID) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}
