---
title: Teams
nav_order: 4
---

## List Teams

`GET /api/teams`

Optional query:

- `division_id` (number): filter teams to a division.

If `division_id` is omitted, returns teams from all divisions.

Response 200

```json
[
    {
        "id": 1,
        "name": "서울고등학교",
        "division_id": 2,
        "division_name": "고등부",
        "created_at": "2026-01-26T12:00:00Z",
        "member_count": 12,
        "total_score": 1200
    }
]
```

Errors:

- 400 `invalid input` (invalid `division_id`)

---

## Get Team

`GET /api/teams/{id}`

Response 200

```json
{
    "id": 1,
    "name": "서울고등학교",
    "division_id": 2,
    "division_name": "고등부",
    "created_at": "2026-01-26T12:00:00Z",
    "member_count": 12,
    "total_score": 1200
}
```

Errors:

- 400 `invalid input`
- 404 `not found`

---

## Get Team Members

`GET /api/teams/{id}/members`

Response 200

```json
[
    {
        "id": 5,
        "username": "user1",
        "role": "user",
        "blocked_reason": null,
        "blocked_at": null
    }
]
```

Blocked users are still listed with `role: "blocked"`, but they are excluded from all score/solve aggregates.

Errors:

- 400 `invalid input`
- 404 `not found`

---

## Get Team Solved Challenges

`GET /api/teams/{id}/solved`

Response 200

```json
[
    {
        "challenge_id": 2,
        "title": "Ch2",
        "points": 200,
        "solve_count": 4,
        "last_solved_at": "2026-01-26T12:30:00Z"
    }
]
```

Notes:

- `points` is dynamically calculated based on solves.
- Blocked users are excluded from solved challenge stats.

Errors:

- 400 `invalid input`
- 404 `not found`
