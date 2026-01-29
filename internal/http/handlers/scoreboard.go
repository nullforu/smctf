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

type rawGroupSubmission struct {
	SubmittedAt time.Time `bun:"submitted_at"`
	GroupID     *int64    `bun:"group_id"`
	GroupName   string    `bun:"group_name"`
	Points      int       `bun:"points"`
}

type groupTimelineSubmission struct {
	Timestamp      time.Time `json:"timestamp"`
	GroupID        *int64    `json:"group_id,omitempty"`
	GroupName      string    `json:"group_name"`
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

func groupGroupSubmissions(raw []rawGroupSubmission) []groupTimelineSubmission {
	if len(raw) == 0 {
		return []groupTimelineSubmission{}
	}

	type groupKey struct {
		groupID  int64
		hasGroup bool
		bucket   time.Time
	}

	groups := make(map[groupKey]*groupTimelineSubmission)

	for _, sub := range raw {
		bucket := sub.SubmittedAt.Truncate(10 * time.Minute)
		key := groupKey{bucket: bucket}
		if sub.GroupID != nil {
			key.groupID = *sub.GroupID
			key.hasGroup = true
		}

		if group, exists := groups[key]; exists {
			group.Points += sub.Points
			group.ChallengeCount++
		} else {
			groups[key] = &groupTimelineSubmission{
				Timestamp:      bucket,
				GroupID:        sub.GroupID,
				GroupName:      sub.GroupName,
				Points:         sub.Points,
				ChallengeCount: 1,
			}
		}
	}

	result := make([]groupTimelineSubmission, 0, len(groups))
	for _, group := range groups {
		result = append(result, *group)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Timestamp.Equal(result[j].Timestamp) {
			if result[i].GroupName == result[j].GroupName {
				return result[i].GroupID != nil && (result[j].GroupID == nil || *result[i].GroupID < *result[j].GroupID)
			}

			return result[i].GroupName < result[j].GroupName
		}

		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result
}
