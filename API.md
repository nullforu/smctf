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

Errors:
- 400 `invalid input`
- 409 `user already exists`

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

Errors:
- 400 `invalid input`
- 401 `invalid credentials`

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

Errors:
- 400 `invalid input`
- 401 `invalid credentials`

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

Errors:
- 400 `invalid input`
- 401 `invalid credentials`

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

Errors:
- 401 `invalid token` or `missing authorization` or `invalid authorization`

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

Errors:
- 401 `invalid token` or `missing authorization` or `invalid authorization`

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

Errors:
- 400 `invalid input`
- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 404 `challenge not found`
- 409 `challenge already solved`
- 429 `too many submissions`

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

Errors:
- 400 `invalid input`

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

Errors:
- 400 `invalid input`
- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `forbidden`

---

## Error Format

All error responses are JSON and may include structured details.

### Common Response
```json
{
  "error": "message",
  "details": [
    { "field": "field_name", "reason": "reason" }
  ]
}
```

`details` is omitted when not applicable.

### Validation Errors (400)
Examples:
```json
{
  "error": "invalid input",
  "details": [
    { "field": "email", "reason": "required" },
    { "field": "password", "reason": "required" }
  ]
}
```

```json
{
  "error": "invalid input",
  "details": [
    { "field": "email", "reason": "invalid format" }
  ]
}
```

```json
{
  "error": "invalid input",
  "details": [
    { "field": "body", "reason": "invalid json" }
  ]
}
```

```json
{
  "error": "invalid input",
  "details": [
    { "field": "flag", "reason": "required" }
  ]
}
```

### Auth Errors (401)
Examples:
```json
{ "error": "missing authorization" }
```

```json
{ "error": "invalid authorization" }
```

```json
{ "error": "invalid token" }
```

```json
{ "error": "invalid credentials" }
```

### Not Found (404)
Examples:
```json
{ "error": "challenge not found" }
```

### Conflict (409)
Examples:
```json
{ "error": "user already exists" }
```

```json
{ "error": "challenge already solved" }
```

### Rate Limit (429)
Examples:
```json
{ "error": "too many submissions" }
```

### Forbidden (403)
Examples:
```json
{ "error": "forbidden" }
```
