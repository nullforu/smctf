# smctf API

Base URL: `http://localhost:8080`

## Auth

### Register
`POST /api/auth/register`

Request
```json
{
  "email": "user@example.com",
  "username": "user1",
  "password": "strong-password"
}
```

Response 201
```json
{
  "id": 1,
  "email": "user@example.com",
  "username": "user1"
}
```

Errors: 400, 409

---

### Login
`POST /api/auth/login`

Request
```json
{
  "email": "user@example.com",
  "password": "strong-password"
}
```

Response 200
```json
{
  "access_token": "<jwt>",
  "refresh_token": "<jwt>",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "username": "user1",
    "role": "user"
  }
}
```

Errors: 400, 401

---

### Refresh Token
`POST /api/auth/refresh`

Request
```json
{
  "refresh_token": "<jwt>"
}
```

Response 200
```json
{
  "access_token": "<jwt>",
  "refresh_token": "<jwt>"
}
```

Errors: 400, 401

---

### Logout
`POST /api/auth/logout`

Request
```json
{
  "refresh_token": "<jwt>"
}
```

Response 200
```json
{
  "status": "ok"
}
```

Errors: 400, 401

---

## User

### Me
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
  "role": "user"
}
```

Errors: 401

---

### Solved Challenges
`GET /api/me/solved`

Headers
```
Authorization: Bearer <access_token>
```

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

Errors: 401

---

## Challenges

### List Challenges
`GET /api/challenges`

Response 200
```json
[
  {
    "id": 1,
    "title": "Warmup",
    "description": "...",
    "points": 100,
    "is_active": true
  }
]
```

---

### Submit Flag
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

Errors: 400, 401, 404, 409, 429

---

## Scoreboard

### Get Scoreboard
`GET /api/scoreboard?limit=50`

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

---

### Scoreboard Timeline
`GET /api/scoreboard/timeline?interval=10&limit=50`

Response 200
```json
{
  "interval_minutes": 10,
  "users": [
    { "user_id": 1, "username": "user1", "score": 300 }
  ],
  "buckets": [
    {
      "bucket": "2026-01-24T12:00:00Z",
      "scores": [
        { "user_id": 1, "username": "user1", "score": 100 }
      ]
    }
  ]
}
```

Errors: 400

---

## Admin

### Create Challenge
`POST /api/admin/challenges`

Headers
```
Authorization: Bearer <access_token>
```

Request
```json
{
  "title": "New Challenge",
  "description": "...",
  "points": 200,
  "flag": "flag{...}",
  "is_active": true
}
```

Response 201
```json
{
  "id": 2,
  "title": "New Challenge",
  "description": "...",
  "points": 200,
  "is_active": true
}
```

Errors: 400, 401, 403

---

## Error Format

```json
{
  "error": "message"
}
```
