---
title: Auth
nav_order: 2
---

## Register

`POST /api/auth/register`

Request

```json
{
    "email": "user@example.com",
    "username": "user1",
    "password": "strong-password",
    "registration_key": "ABCDEFGHJKLMNPQ2"
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

Validation notes:

- `username` must be at most 10 characters (counted by Unicode code points).
- `password` must be at most 72 bytes (bcrypt input limit).

`registration_key` must be an admin-created alphanumeric code.
Keys can be reused up to their configured `max_uses` and assign the user to the key's team.

---

## Login

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
    "user": {
        "id": 1,
        "email": "user@example.com",
        "username": "user1",
        "role": "user",
        "team_id": 3,
        "team_name": "team-alpha",
        "division_id": 2,
        "division_name": "고등부",
        "vm_count": 0,
        "vm_limit": 3,
        "blocked_reason": null,
        "blocked_at": null
    }
}
```

Errors:

- 400 `invalid input`
- 401 `invalid credentials`

Notes:

- `vm_count` and `vm_limit` reflect the configured scope. If `VMS_MAX_SCOPE=team`, these values are team-wide.
- `access_token` and `refresh_token` are issued as `HttpOnly` cookies.
- `csrf_token` is issued as a readable cookie for double-submit CSRF protection.

---

## Refresh Token

`POST /api/auth/refresh`

Request: send `refresh_token` cookie (and `X-CSRF-Token` header matching `csrf_token` cookie).

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

## Logout

`POST /api/auth/logout`

Request: send `refresh_token` cookie (and `X-CSRF-Token` header matching `csrf_token` cookie).

Response 200

```json
{
    "status": "ok"
}
```

Errors:

- 400 `invalid input`
- 401 `invalid credentials`
