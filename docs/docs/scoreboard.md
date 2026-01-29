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

## Get Group / Organization Leaderboard

`GET /api/leaderboard/groups`

Response 200

```json
[
    {
        "group_id": 1,
        "group_name": "서울고등학교",
        "score": 1200
    },
    {
        "group_id": null,
        "group_name": "not affiliated",
        "score": 200
    }
]
```

Returns all groups sorted by score (descending). Users without a group are shown as "not affiliated".

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

Returns all submissions grouped by user and 10 minute intervals.
If multiple challenges are solved by the same user within 10 minutes, they are grouped together with cumulative points and challenge count.

Errors:

- 400 `invalid input`

---

## Get Group / Organization Timeline

`GET /api/timeline/groups?window=60`

Query

- `window`: lookback window in minutes (optional, when omitted returns all time)

Response 200

```json
{
    "submissions": [
        {
            "timestamp": "2026-01-24T12:00:00Z",
            "group_id": 1,
            "group_name": "서울고등학교",
            "points": 300,
            "challenge_count": 2
        }
    ]
}
```

Returns all submissions grouped by group and 10 minute intervals.
Users without a group are grouped under "not affiliated".

Errors:

- 400 `invalid input`
