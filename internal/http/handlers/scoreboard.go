package handlers

import (
	"sort"
	"time"

	"smctf/internal/models"
)

func indexUsers(users []models.ScoreEntry) ([]int64, map[int64]string) {
	userIDs := make([]int64, 0, len(users))
	usernames := make(map[int64]string, len(users))

	for _, u := range users {
		userIDs = append(userIDs, u.UserID)
		usernames[u.UserID] = u.Username
	}

	return userIDs, usernames
}

func buildScoreTimelineBuckets(rows []models.ScoreTimelineRow, userIDs []int64, usernames map[int64]string) []models.ScoreTimelineBucket {
	bucketMap := make(map[time.Time]map[int64]int)

	for _, row := range rows {
		if _, ok := bucketMap[row.Bucket]; !ok {
			bucketMap[row.Bucket] = make(map[int64]int)
		}

		bucketMap[row.Bucket][row.UserID] += row.Score
	}

	buckets := make([]time.Time, 0, len(bucketMap))
	for bucket := range bucketMap {
		buckets = append(buckets, bucket)
	}

	sort.Slice(buckets, func(i, j int) bool { return buckets[i].Before(buckets[j]) })

	cumulative := make(map[int64]int, len(userIDs))
	respBuckets := make([]models.ScoreTimelineBucket, 0, len(buckets))

	for _, bucket := range buckets {
		scores := make([]models.ScoreEntry, 0, len(userIDs))

		for _, id := range userIDs {
			cumulative[id] += bucketMap[bucket][id]
			scores = append(scores, models.ScoreEntry{
				UserID:   id,
				Username: usernames[id],
				Score:    cumulative[id],
			})
		}

		respBuckets = append(respBuckets, models.ScoreTimelineBucket{
			Bucket: bucket,
			Scores: scores,
		})
	}

	return respBuckets
}
