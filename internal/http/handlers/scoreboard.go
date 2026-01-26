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

	sort.Slice(result, func(i, j int) bool {
		if result[i].Timestamp.Equal(result[j].Timestamp) {
			return result[i].UserID < result[j].UserID
		}

		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result
}
