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

Returns only the user's own solved challenges (not team-shared).

Response 200

```json
{
    "id": 1,
    "email": "user@example.com",
    "username": "user1",
    "role": "user",
    "team_id": 1,
    "team_name": "서울고등학교"
}
```

Errors:

- 401 `invalid token` or `missing authorization` or `invalid authorization`

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
    "team_name": "서울고등학교"
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing authorization` or `invalid authorization`

---

## Solved Challenges

Use `GET /api/me` to fetch the current user ID, then call `GET /api/users/{id}/solved`.

## List Users

`GET /api/users`

Response 200

```json
[
    {
        "id": 1,
        "username": "user1",
        "role": "user",
        "team_id": 1,
        "team_name": "서울고등학교"
    },
    {
        "id": 2,
        "username": "admin",
        "role": "admin",
        "team_id": null,
        "team_name": "not affiliated"
    }
]
```

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
    "team_name": "서울고등학교"
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

Errors:

- 400 `invalid input`
- 404 `not found`

---

## Team Solved Challenges (My Team)

Use `GET /api/me` to fetch the current user's `team_id`, then call `GET /api/teams/{team_id}/solved`.
