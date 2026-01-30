---
title: Admin
nav_order: 6
---

## Create Registration Keys

`POST /api/admin/registration-keys`

Headers

```
Authorization: Bearer <access_token>
```

Request

```json
{
    "count": 5,
    "team_id": 1
}
```

`team_id` is optional. Omit or set to null for unassigned (not affiliated).

Response 201

```json
[
    {
        "id": 10,
        "code": "123456",
        "created_by": 2,
        "created_by_username": "admin",
        "team_id": 1,
        "team_name": "서울고등학교",
        "used_by": null,
        "used_by_username": null,
        "used_by_ip": null,
        "created_at": "2026-01-26T12:00:00Z",
        "used_at": null
    }
]
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `forbidden`

---

## List Registration Keys

`GET /api/admin/registration-keys`

Headers

```
Authorization: Bearer <access_token>
```

Response 200

```json
[
    {
        "id": 10,
        "code": "123456",
        "created_by": 2,
        "created_by_username": "admin",
        "team_id": 1,
        "team_name": "서울고등학교",
        "used_by": 5,
        "used_by_username": "user1",
        "used_by_ip": "203.0.113.7",
        "created_at": "2026-01-26T12:00:00Z",
        "used_at": "2026-01-26T12:30:00Z"
    }
]
```

Errors:

- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `forbidden`

---

## Create Team

`POST /api/admin/teams`

Headers

```
Authorization: Bearer <access_token>
```

Request

```json
{
    "name": "서울고등학교"
}
```

Response 201

```json
{
    "id": 1,
    "name": "서울고등학교",
    "created_at": "2026-01-26T12:00:00Z"
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `forbidden`

---

## Create Challenge

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
    "category": "Web",
    "points": 200,
    "flag": "flag{...}",
    "is_active": true
}
```

Categories

```
Web, Web3, Pwnable, Reversing, Crypto, Forensics, Network, Cloud, Misc,
Programming, Algorithms, Math, AI, Blockchain
```

Response 201

```json
{
    "id": 2,
    "title": "New Challenge",
    "description": "...",
    "category": "Web",
    "points": 200,
    "is_active": true
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `forbidden`

---

## Update Challenge

`PUT /api/admin/challenges/{id}`

Headers

```
Authorization: Bearer <access_token>
```

Request

All fields are optional. Only provided fields are validated and updated.
`flag` cannot be changed via this endpoint.

```json
{
    "title": "Updated Challenge",
    "points": 250,
    "is_active": false
}
```

Response 200

```json
{
    "id": 2,
    "title": "Updated Challenge",
    "description": "...",
    "category": "Crypto",
    "points": 250,
    "is_active": false
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `forbidden`
- 404 `challenge not found`

---

## Delete Challenge

`DELETE /api/admin/challenges/{id}`

Headers

```
Authorization: Bearer <access_token>
```

Response 200

```json
{
    "status": "ok"
}
```

Errors:

- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `forbidden`
- 404 `challenge not found`
