---
title: Challenges
nav_order: 4
---

## List Challenges

`GET /api/challenges?division_id={id}`

`division_id` is required.

Response 200

```json
{
    "ctf_state": "active",
    "challenges": [
        {
            "id": 1,
            "title": "Warmup",
            "description": "...",
            "category": "Web",
            "points": 100,
            "initial_points": 200,
            "minimum_points": 50,
            "solve_count": 12,
            "is_active": true,
            "is_locked": false,
            "has_file": true,
            "file_name": "challenge.zip",
            "stack_enabled": false,
            "stack_target_ports": []
        }
    ]
}
```

Notes:

- `points` is dynamically calculated based on solves.
- `has_file` indicates whether a challenge file is available.
- `stack_enabled` indicates if a stack instance is supported for this challenge. Scope is controlled by `STACKS_MAX_SCOPE` (user or team).
- If a challenge is locked, the response includes only `id`, `title`, `category`, `points`, `initial_points`, `minimum_points`, `solve_count`, `previous_challenge_id`, `previous_challenge_title`, `previous_challenge_category`, `is_active`, and `is_locked`.
- If `ctf_state` is `not_started`, the response only includes `ctf_state`.

Errors:

- 400 `invalid input` (`division_id` required or invalid)

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
    "correct": true,
    "ctf_state": "active"
}
```

Notes:

- A challenge is considered already solved once any teammate solves it.
- If `ctf_state` is `not_started` or `ended`, the response only includes `ctf_state`.

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `challenge locked`
- 404 `challenge not found`
- 409 `challenge already solved`
- 429 `too many submissions`

---

## Download Challenge File

`POST /api/challenges/{id}/file/download`

Headers

```
Authorization: Bearer <access_token>
```

Response 200

```json
{
    "url": "https://s3.example.com/...",
    "expires_at": "2025-01-01T00:00:00Z",
    "ctf_state": "active"
}
```

Errors:

- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `challenge locked`
- 404 `challenge not found` or `challenge file not found`
- If `ctf_state` is `not_started`, the response only includes `ctf_state`.
