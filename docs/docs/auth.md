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
    "registration_key": "TESTKEY0001"
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

## Refresh Token

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

## Logout

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
