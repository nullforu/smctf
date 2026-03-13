---
title: Users
nav_order: 3
---

## Me

`GET /api/me`

Headers

```
Authorization: Bearer <access_token>
```

Response 200

```json
{
    "id": 1,
    "email": "user@example.com",
    "username": "user1",
    "role": "user",
    "team_id": 1,
    "team_name": "서울고등학교",
    "division_id": 2,
    "division_name": "고등부",
    "stack_count": 0,
    "stack_limit": 3,
    "blocked_reason": null,
    "blocked_at": null
}
```

Errors:

- 401 `invalid token` or `missing authorization` or `invalid authorization`

Notes:

- `stack_count` and `stack_limit` reflect the configured scope. If `STACKS_MAX_SCOPE=team`, these values are team-wide.

---

## Update Me

`PUT /api/me`

Headers

```
Authorization: Bearer <access_token>
```

Request

```json
{
    "username": "new_username"
}
```

Response 200

```json
{
    "id": 1,
    "email": "user@example.com",
    "username": "new_username",
    "role": "user",
    "team_id": 1,
    "team_name": "서울고등학교",
    "division_id": 2,
    "division_name": "고등부",
    "stack_count": 0,
    "stack_limit": 3,
    "blocked_reason": null,
    "blocked_at": null
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `user blocked`

Notes:

- `stack_count` and `stack_limit` reflect the configured scope. If `STACKS_MAX_SCOPE=team`, these values are team-wide.

---

## Solved Challenges

Use `GET /api/me` to fetch the current user ID, then call `GET /api/users/{id}/solved`.

## List Users

`GET /api/users`

Optional query:

- `division_id` (number): filter users to a division.

If `division_id` is omitted, returns users from all divisions.

Response 200

```json
[
    {
        "id": 1,
        "username": "user1",
        "role": "user",
        "team_id": 1,
        "team_name": "서울고등학교",
        "division_id": 2,
        "division_name": "고등부",
        "blocked_reason": null,
        "blocked_at": null
    },
    {
        "id": 2,
        "username": "admin",
        "role": "admin",
        "team_id": 2,
        "team_name": "운영팀",
        "division_id": 2,
        "division_name": "대학부",
        "blocked_reason": null,
        "blocked_at": null
    }
]
```

Errors:

- 400 `invalid input` (invalid `division_id`)

---

## Get User

`GET /api/users/{id}`

Response 200

```json
{
    "id": 1,
    "username": "user1",
    "role": "user",
    "team_id": 1,
    "team_name": "서울고등학교",
    "division_id": 2,
    "division_name": "고등부",
    "blocked_reason": null,
    "blocked_at": null
}
```

Errors:

- 400 `invalid input`
- 404 `not found`

---

## Get User Solved Challenges

`GET /api/users/{id}/solved`

Returns only the user's own solved challenges (not team-shared).

Response 200

```json
[
    {
        "challenge_id": 1,
        "title": "Warmup",
        "points": 100,
        "solved_at": "2026-01-24T12:00:00Z"
    }
]
```

Notes:

- `points` is dynamically calculated based on solves.
- Blocked users are excluded from solved challenge stats.

Errors:

- 400 `invalid input`
- 404 `not found`

---

## Team Solved Challenges (My Team)

Use `GET /api/me` to fetch the current user's `team_id`, then call `GET /api/teams/{team_id}/solved`.
