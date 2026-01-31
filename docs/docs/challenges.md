---
title: Challenges
nav_order: 4
---

## List Challenges

`GET /api/challenges`

Response 200

```json
[
    {
        "id": 1,
        "title": "Warmup",
        "description": "...",
        "category": "Web",
        "points": 100,
        "initial_points": 200,
        "minimum_points": 50,
        "solve_count": 12,
        "is_active": true
    }
]
```

Notes:

- `points` is dynamically calculated based on solves.

---

## Submit Flag

`POST /api/challenges/{id}/submit`

Headers

```
Authorization: Bearer <access_token>
```

Request

```json
{
    "flag": "flag{...}"
}
```

Response 200

```json
{
    "correct": true
}
```

Notes:

- If a user belongs to a team, a challenge is considered already solved once any teammate solves it.

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 404 `challenge not found`
- 409 `challenge already solved`
- 429 `too many submissions`
