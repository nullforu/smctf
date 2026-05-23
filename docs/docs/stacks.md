---
title: Stacks
nav_order: 8
---

## List My Stacks

`GET /api/stacks`

Headers

```
Cookie: access_token=<jwt>
```

Response 200

```json
{
    "ctf_state": "active",
    "stacks": [
        {
            "stack_id": "stack-716b6384dd477b0b",
            "challenge_id": 12,
            "challenge_title": "SQLi 101",
            "status": "running",
            "node_public_ip": "12.34.56.78",
            "ports": [
                {
                    "container_port": 80,
                    "protocol": "TCP",
                    "node_port": 31538
                }
            ],
            "ttl_expires_at": "2026-02-10T04:02:26Z",
            "created_at": "2026-02-10T02:02:26Z",
            "updated_at": "2026-02-10T02:07:29Z",
            "created_by_user_id": 17,
            "created_by_username": "alice",
            "ctf_state": "active"
        }
    ]
}
```

Errors:

- 401 `invalid token` or `missing access_token cookie`
- 503 `stack feature disabled`
- If `ctf_state` is `not_started`, the response only includes `ctf_state`.
  Notes:

- Blocked users can access this endpoint (read-only).
- If `STACKS_MAX_SCOPE=team`, this list includes stacks created by any member of the same team.

---

## Create Stack For Challenge

`POST /api/challenges/{id}/stack`

Headers

```
Cookie: access_token=<jwt>
```

Response 201

```json
{
    "stack_id": "stack-716b6384dd477b0b",
    "challenge_id": 12,
    "challenge_title": "SQLi 101",
    "status": "creating",
    "node_public_ip": "12.34.56.78",
    "ports": [
        {
            "container_port": 80,
            "protocol": "TCP",
            "node_port": 31538
        }
    ],
    "ttl_expires_at": "2026-02-10T04:02:26Z",
    "created_at": "2026-02-10T02:02:26Z",
    "updated_at": "2026-02-10T02:02:26Z",
    "created_by_user_id": 17,
    "created_by_username": "alice",
    "ctf_state": "active"
}
```

Errors:

- 400 `invalid input` or `stack not enabled for challenge`
- 401 `invalid token` or `missing access_token cookie`
- 403 `challenge locked`
- 404 `challenge not found`
- 409 `stack limit reached` or `challenge already solved`
- 429 `too many submissions` (rate limited)
- 503 `stack feature disabled` or `stack provisioner unavailable`
- If `ctf_state` is `not_started` or `ended`, the response only includes `ctf_state`.

Notes:

- Stack creation is rate-limited by scope. Configure via `STACKS_CREATE_WINDOW` and `STACKS_CREATE_MAX`.
- If `STACKS_MAX_SCOPE=team`, creating a stack counts against the team limit and team rate limit.

---

## Get Stack For Challenge

`GET /api/challenges/{id}/stack`

Headers

```
Cookie: access_token=<jwt>
```

Response 200

```json
{
    "stack_id": "stack-716b6384dd477b0b",
    "challenge_id": 12,
    "challenge_title": "SQLi 101",
    "status": "running",
    "node_public_ip": "12.34.56.78",
    "ports": [
        {
            "container_port": 80,
            "protocol": "TCP",
            "node_port": 31538
        }
    ],
    "ttl_expires_at": "2026-02-10T04:02:26Z",
    "created_at": "2026-02-10T02:02:26Z",
    "updated_at": "2026-02-10T02:07:29Z",
    "created_by_user_id": 17,
    "created_by_username": "alice",
    "ctf_state": "active"
}
```

Errors:

- 401 `invalid token` or `missing access_token cookie`
- 404 `stack not found`
- 503 `stack feature disabled` or `stack provisioner unavailable`
- If `ctf_state` is `not_started`, the response only includes `ctf_state`.
  Notes:

- Blocked users can access this endpoint (read-only).
- If `STACKS_MAX_SCOPE=team`, this returns the team stack for the challenge (if any).

---

## Delete Stack For Challenge

`DELETE /api/challenges/{id}/stack`

Headers

```
Cookie: access_token=<jwt>
```

Response 200

```json
{
    "status": "ok",
    "ctf_state": "active"
}
```

Errors:

- 401 `invalid token` or `missing access_token cookie`
- 404 `stack not found`
- 503 `stack feature disabled` or `stack provisioner unavailable`
- If `ctf_state` is `not_started`, the response only includes `ctf_state`.
