---
title: Leaderboard & Timeline
nav_order: 5
---

## Get Leaderboard

`GET /api/leaderboard`

Response 200

```json
[
    {
        "user_id": 1,
        "username": "user1",
        "score": 300
    }
]
```

Returns all users sorted by score (descending).

---

## Get Team Leaderboard

`GET /api/leaderboard/teams`

Response 200

```json
[
    {
        "team_id": 1,
        "team_name": "서울고등학교",
        "score": 1200
    },
    {
        "team_id": null,
        "team_name": "not affiliated",
        "score": 200
    }
]
```

Returns all teams sorted by score (descending). Users without a team are shown as "not affiliated".

---

## Get Timeline

`GET /api/timeline?window=60`

Query

- `window`: lookback window in minutes (optional, when omitted returns all time)

Response 200

```json
{
    "submissions": [
        {
            "timestamp": "2026-01-24T12:00:00Z",
            "user_id": 1,
            "username": "user1",
            "points": 300,
            "challenge_count": 2
        }
    ]
}
```

Returns all submissions teamed by user and 10 minute intervals.
If multiple challenges are solved by the same user within 10 minutes, they are teamed together with cumulative points and challenge count.
`points` is dynamically calculated based on solves.

Errors:

- 400 `invalid input`

---

## Get Team Timeline

`GET /api/timeline/teams?window=60`

Query

- `window`: lookback window in minutes (optional, when omitted returns all time)

Response 200

```json
{
    "submissions": [
        {
            "timestamp": "2026-01-24T12:00:00Z",
            "team_id": 1,
            "team_name": "서울고등학교",
            "points": 300,
            "challenge_count": 2
        }
    ]
}
```

Returns all submissions teamed by team and 10 minute intervals.
Users without a team are teamed under "not affiliated".

`points` is dynamically calculated based on solves.

Errors:

- 400 `invalid input`
