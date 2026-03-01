---
title: Admin
nav_order: 6
---

## Update Site Configuration

`PUT /api/admin/config`

Headers

```
Authorization: Bearer <access_token>
```

Request

All fields are optional. Only provided fields are validated and updated.
To keep existing values, omit the field entirely.

Field behavior:

| Field                | Type             | Omit          | null        | Empty/Whitespace String | Other String    |
| -------------------- | ---------------- | ------------- | ----------- | ----------------------- | --------------- |
| `title`              | string           | Keep existing | Error       | Allowed                 | Allowed         |
| `description`        | string           | Keep existing | Error       | Allowed                 | Allowed         |
| `header_title`       | string           | Keep existing | Error       | Allowed                 | Allowed         |
| `header_description` | string           | Keep existing | Error       | Allowed                 | Allowed         |
| `ctf_start_at`       | string (RFC3339) | Keep existing | Clear value | Error                   | Must be RFC3339 |
| `ctf_end_at`         | string (RFC3339) | Keep existing | Clear value | Error                   | Must be RFC3339 |

```json
{
    "title": "My CTF",
    "description": "Hello",
    "header_title": "SM CTF",
    "header_description": "Join the challenge",
    "ctf_start_at": "2099-12-31T10:00:00Z",
    "ctf_end_at": "2099-12-31T18:00:00Z"
}
```

Response 200

```json
{
    "title": "My CTF",
    "description": "Hello",
    "header_title": "SM CTF",
    "header_description": "Join the challenge",
    "ctf_start_at": "2099-12-31T10:00:00Z",
    "ctf_end_at": "2099-12-31T18:00:00Z",
    "updated_at": "2026-01-26T12:00:00Z"
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `forbidden`

Notes:

- `ctf_start_at` and `ctf_end_at` are RFC3339 timestamps. Empty values mean the CTF is always active.

---

## Admin Report

`GET /api/admin/report`

Headers

```
Authorization: Bearer <access_token>
```

Response 200

```json
{
    "challenges": [
        {
            "id": 1,
            "title": "Challenge",
            "description": "...",
            "category": "Web",
            "points": 100,
            "initial_points": 100,
            "minimum_points": 50,
            "solve_count": 3,
            "is_active": true,
            "file_key": null,
            "file_name": null,
            "file_uploaded_at": null,
            "stack_enabled": false,
            "stack_target_ports": [],
            "stack_pod_spec": null,
            "created_at": "2026-02-17T12:00:00Z"
        }
    ],
    "divisions": [
        {
            "id": 2,
            "name": "고등부",
            "created_at": "2026-02-17T09:00:00Z"
        }
    ],
    "teams": [
        {
            "id": 1,
            "name": "Alpha",
            "division_id": 2,
            "division_name": "고등부",
            "created_at": "2026-02-17T10:00:00Z",
            "member_count": 2,
            "total_score": 200
        }
    ],
    "users": [
        {
            "id": 1,
            "email": "user@example.com",
            "username": "user",
            "role": "user",
            "team_id": 1,
            "team_name": "Alpha",
            "division_id": 2,
            "division_name": "고등부",
            "blocked_reason": null,
            "blocked_at": null,
            "created_at": "2026-02-17T10:00:00Z",
            "updated_at": "2026-02-17T10:00:00Z"
        }
    ],
    "stacks": [],
    "registration_keys": [],
    "submissions": [],
    "app_config": [],
    "timeline": {
        "submissions": []
    },
    "team_timeline": {
        "submissions": []
    },
    "leaderboard": {
        "challenges": [],
        "entries": []
    },
    "team_leaderboard": {
        "challenges": [],
        "entries": []
    }
}
```

Notes:

- Password hashes are excluded from user records.
- Challenge flag data is excluded from the report.
- Submission provided flag data are excluded from the report.
- Challenge `points` in the report reflect global dynamic scoring (all divisions combined), not per-division scoring.
- See [report.schema.json](./report.schema.json) for the full schema. (there may be slight differences from the actual response)

Errors:

- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `forbidden`

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
    "team_id": 1,
    "max_uses": 3
}
```

`team_id` is required.
`max_uses` defaults to 1.

Response 201

```json
[
    {
        "id": 10,
        "code": "ABCDEFGHJKLMNPQ2",
        "created_by": 2,
        "created_by_username": "admin",
        "team_id": 1,
        "team_name": "서울고등학교",
        "max_uses": 3,
        "used_count": 0,
        "created_at": "2026-01-26T12:00:00Z",
        "last_used_at": null
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
        "code": "ABCDEFGHJKLMNPQ2",
        "created_by": 2,
        "created_by_username": "admin",
        "team_id": 1,
        "team_name": "서울고등학교",
        "max_uses": 3,
        "used_count": 2,
        "created_at": "2026-01-26T12:00:00Z",
        "last_used_at": "2026-01-26T12:30:00Z",
        "uses": [
            {
                "used_by": 5,
                "used_by_username": "user1",
                "used_by_ip": "203.0.113.7",
                "used_at": "2026-01-26T12:30:00Z"
            }
        ]
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
    "name": "서울고등학교",
    "division_id": 2
}
```

Response 201

```json
{
    "id": 1,
    "name": "서울고등학교",
    "division_id": 2,
    "created_at": "2026-01-26T12:00:00Z"
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `forbidden`

---

## Create Division (Admin)

`POST /api/admin/divisions`

Request

```json
{
    "name": "고등부"
}
```

Response 201

```json
{
    "id": 2,
    "name": "고등부",
    "created_at": "2026-01-26T12:00:00Z"
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `forbidden`

---

## Move User Team

`POST /api/admin/users/:id/team`

Headers

```
Authorization: Bearer <access_token>
```

Request

```json
{
    "team_id": 2
}
```

Response 200

```json
{
    "id": 10,
    "email": "user1@example.com",
    "username": "user1",
    "role": "user",
    "team_id": 2,
    "team_name": "New Team",
    "blocked_reason": null,
    "blocked_at": null
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `forbidden`
- 404 `not found`

---

## Block User

`POST /api/admin/users/:id/block`

Headers

```
Authorization: Bearer <access_token>
```

Request

```json
{
    "reason": "policy violation"
}
```

Response 200

```json
{
    "id": 10,
    "email": "user1@example.com",
    "username": "user1",
    "role": "blocked",
    "team_id": 2,
    "team_name": "New Team",
    "blocked_reason": "policy violation",
    "blocked_at": "2026-01-26T12:00:00Z"
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `forbidden`
- 404 `not found`

---

## Unblock User

`POST /api/admin/users/:id/unblock`

Headers

```
Authorization: Bearer <access_token>
```

Response 200

```json
{
    "id": 10,
    "email": "user1@example.com",
    "username": "user1",
    "role": "user",
    "team_id": 2,
    "team_name": "New Team",
    "blocked_reason": null,
    "blocked_at": null
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `forbidden`
- 404 `not found`

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
    "minimum_points": 50,
    "flag": "flag{...}",
    "previous_challenge_id": 1,
    "is_active": true,
    "stack_enabled": false,
    "stack_target_ports": [
        {
            "container_port": 80,
            "protocol": "TCP"
        }
    ],
    "stack_pod_spec": "apiVersion: v1\nkind: Pod\nmetadata:\n  name: challenge\nspec:\n  containers:\n    - name: app\n      image: nginx:stable\n      ports:\n        - containerPort: 80"
}
```

If `minimum_points` is omitted, it defaults to the same value as `points`.
If `stack_enabled` is true, both `stack_target_ports` and `stack_pod_spec` are required.

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
    "initial_points": 200,
    "minimum_points": 50,
    "solve_count": 0,
    "previous_challenge_id": 1,
    "is_active": true,
    "has_file": false
}
```

Notes:

- `points` is dynamically calculated based on solves. `initial_points` is the configured starting value.

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
To keep existing values, omit the field entirely.

Field behavior:

| Field                   | Type   | Omit          | null         | Empty/Whitespace String | Other                                                     |
| ----------------------- | ------ | ------------- | ------------ | ----------------------- | --------------------------------------------------------- |
| `title`                 | string | Keep existing | Error        | Allowed                 | Allowed                                                   |
| `description`           | string | Keep existing | Error        | Allowed                 | Allowed                                                   |
| `category`              | string | Keep existing | Error        | Error                   | Must be a valid category                                  |
| `points`                | int    | Keep existing | Error        | Error                   | Must be >= 0                                              |
| `minimum_points`        | int    | Keep existing | Error        | Error                   | Must be >= 0 and <= `points`                              |
| `flag`                  | string | Keep existing | Error        | Error                   | Updates flag                                              |
| `previous_challenge_id` | int    | Keep existing | Clears value | Error                   | Must be a valid challenge id (not self)                   |
| `is_active`             | bool   | Keep existing | Error        | Error                   | Sets value                                                |
| `stack_enabled`         | bool   | Keep existing | Error        | Error                   | If `false`, clears `stack_target_ports` + `stack_pod_spec` |
| `stack_target_ports`    | array  | Keep existing | Error        | Error                   | Requires `stack_enabled` true; container port 1-65535 and protocol TCP/UDP |
| `stack_pod_spec`        | string | Keep existing | Error        | Error                   | Requires `stack_enabled` true and non-empty value         |

If `stack_enabled` is true after updates, `stack_target_ports` and `stack_pod_spec` are required (non-empty).
To clear stack fields, set `stack_enabled` to `false` (and omit `stack_target_ports` / `stack_pod_spec`).

```json
{
    "title": "Updated Challenge",
    "points": 250,
    "minimum_points": 100,
    "flag": "flag{rotated}",
    "previous_challenge_id": 1,
    "is_active": false,
    "stack_enabled": true,
    "stack_target_ports": [
        {
            "container_port": 80,
            "protocol": "TCP"
        }
    ],
    "stack_pod_spec": "apiVersion: v1\nkind: Pod\nmetadata:\n  name: challenge\nspec:\n  containers:\n    - name: app\n      image: nginx:stable\n      ports:\n        - containerPort: 80"
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
    "initial_points": 250,
    "minimum_points": 100,
    "solve_count": 12,
    "previous_challenge_id": 1,
    "is_active": false,
    "has_file": true,
    "file_name": "challenge.zip",
    "stack_enabled": true,
    "stack_target_ports": [
        {
            "container_port": 80,
            "protocol": "TCP"
        }
    ]
}
```

Notes:

- `has_file` and `file_name` are not updated via this endpoint. They are managed by the file upload/delete endpoints.

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `forbidden`
- 404 `challenge not found`

---

## Get Challenge Detail (Admin)

`GET /api/admin/challenges/{id}`

Headers

```
Authorization: Bearer <access_token>
```

Response 200

```json
{
    "id": 2,
    "title": "Updated Challenge",
    "description": "...",
    "category": "Crypto",
    "points": 250,
    "initial_points": 250,
    "minimum_points": 100,
    "solve_count": 12,
    "previous_challenge_id": 1,
    "is_active": false,
    "has_file": true,
    "file_name": "challenge.zip",
    "stack_enabled": true,
    "stack_target_ports": [
        {
            "container_port": 80,
            "protocol": "TCP"
        }
    ],
    "stack_pod_spec": "apiVersion: v1\nkind: Pod\nmetadata:\n  name: challenge\nspec:\n  containers:\n    - name: app\n      image: nginx:stable\n      ports:\n        - containerPort: 80"
}
```

Notes:

- `stack_pod_spec` is only returned via this admin-only endpoint.

Errors:

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

---

## Stack Management (Admin)

Headers

```
Authorization: Bearer <access_token>
```

### List All Stacks

`GET /api/admin/stacks`

Response 200

```json
{
    "stacks": [
        {
            "stack_id": "stack-716b6384dd477b0b",
            "ttl_expires_at": "2026-02-10T04:02:26.535664Z",
            "created_at": "2026-02-10T02:02:26.535664Z",
            "updated_at": "2026-02-10T02:06:33.16031Z",
            "user_id": 12,
            "username": "alice",
            "email": "alice@example.com",
            "team_id": 2,
            "team_name": "Red",
            "challenge_id": 5,
            "challenge_title": "Web 1",
            "challenge_category": "Web"
        }
    ]
}
```

Errors:

- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `forbidden`
- 503 `stack feature disabled` or `stack provisioner unavailable`

### Get Stack Detail

`GET /api/admin/stacks/{stack_id}`

Response 200

```json
{
    "stack_id": "stack-716b6384dd477b0b",
    "challenge_id": 5,
    "status": "running",
    "node_public_ip": "12.34.56.78",
    "ports": [
        {
            "container_port": 80,
            "protocol": "TCP",
            "node_port": 31538
        }
    ],
    "ttl_expires_at": "2026-02-10T04:02:26.535664Z",
    "created_at": "2026-02-10T02:02:26.535664Z",
    "updated_at": "2026-02-10T02:06:33.16031Z"
}
```

Errors:

- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `forbidden`
- 404 `stack not found`
- 503 `stack feature disabled` or `stack provisioner unavailable`

### Delete Stack

`DELETE /api/admin/stacks/{stack_id}`

Response 200

```json
{
    "deleted": true,
    "stack_id": "stack-716b6384dd477b0b"
}
```

Errors:

- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `forbidden`
- 404 `stack not found`
- 503 `stack feature disabled` or `stack provisioner unavailable`

---

## Upload Challenge File

`POST /api/admin/challenges/{id}/file/upload`

Headers

```
Authorization: Bearer <access_token>
```

Request

```json
{
    "filename": "challenge.zip"
}
```

Response 200

```json
{
    "challenge": {
        "id": 2,
        "title": "New Challenge",
        "description": "...",
        "category": "Web",
        "points": 200,
        "initial_points": 200,
        "minimum_points": 50,
        "solve_count": 0,
        "is_active": true,
        "has_file": true,
        "file_name": "challenge.zip"
    },
    "upload": {
        "url": "https://s3.example.com/...",
        "fields": {
            "key": "uuid.zip",
            "Content-Type": "application/zip"
        },
        "expires_at": "2025-01-01T00:00:00Z"
    }
}
```

Notes:

- The upload target expects a `.zip` file. The server stores it as `UUID.zip` in the configured bucket.

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `forbidden`
- 404 `challenge not found`
- 503 `storage unavailable`

---

## Delete Challenge File

`DELETE /api/admin/challenges/{id}/file`

Headers

```
Authorization: Bearer <access_token>
```

Response 200

```json
{
    "id": 2,
    "title": "New Challenge",
    "description": "...",
    "category": "Web",
    "points": 200,
    "initial_points": 200,
    "minimum_points": 50,
    "solve_count": 0,
    "is_active": true,
    "has_file": false
}
```

Errors:

- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `forbidden`
- 404 `challenge not found` or `challenge file not found`
- 503 `storage unavailable`
