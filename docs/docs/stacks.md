---
title: Stacks
nav_order: 8
---

## List My Stacks

`GET /api/stacks`

Headers

```
Authorization: Bearer <access_token>
```

Response 200

```json
{
    "ctf_state": "active",
    "stacks": [
        {
            "stack_id": "stack-716b6384dd477b0b",
            "challenge_id": 12,
            "status": "running",
            "node_public_ip": "12.34.56.78",
            "node_port": 31538,
            "target_port": 80,
            "ttl_expires_at": "2026-02-10T04:02:26Z",
            "created_at": "2026-02-10T02:02:26Z",
            "updated_at": "2026-02-10T02:07:29Z",
            "ctf_state": "active"
        }
    ]
}
```

Errors:

- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 503 `stack feature disabled`
- If `ctf_state` is `not_started`, the response only includes `ctf_state`.

---

## Create Stack For Challenge

`POST /api/challenges/{id}/stack`

Headers

```
Authorization: Bearer <access_token>
```

Response 201

```json
{
    "stack_id": "stack-716b6384dd477b0b",
    "challenge_id": 12,
    "status": "creating",
    "node_public_ip": "12.34.56.78",
    "node_port": 31538,
    "target_port": 80,
    "ttl_expires_at": "2026-02-10T04:02:26Z",
    "created_at": "2026-02-10T02:02:26Z",
    "updated_at": "2026-02-10T02:02:26Z",
    "ctf_state": "active"
}
```

Errors:

- 400 `invalid input` or `stack not enabled for challenge`
- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 404 `challenge not found`
- 409 `stack limit reached` or `challenge already solved`
- 429 `too many submissions` (rate limited)
- 503 `stack feature disabled` or `stack provisioner unavailable`
- If `ctf_state` is `not_started` or `ended`, the response only includes `ctf_state`.

Notes:

- Stack creation is rate-limited per user. Configure via `STACKS_CREATE_WINDOW` and `STACKS_CREATE_MAX`.

---

## Get Stack For Challenge

`GET /api/challenges/{id}/stack`

Headers

```
Authorization: Bearer <access_token>
```

Response 200

```json
{
    "stack_id": "stack-716b6384dd477b0b",
    "challenge_id": 12,
    "status": "running",
    "node_public_ip": "12.34.56.78",
    "node_port": 31538,
    "target_port": 80,
    "ttl_expires_at": "2026-02-10T04:02:26Z",
    "created_at": "2026-02-10T02:02:26Z",
    "updated_at": "2026-02-10T02:07:29Z",
    "ctf_state": "active"
}
```

Errors:

- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 404 `stack not found`
- 503 `stack feature disabled` or `stack provisioner unavailable`
- If `ctf_state` is `not_started`, the response only includes `ctf_state`.

---

## Delete Stack For Challenge

`DELETE /api/challenges/{id}/stack`

Headers

```
Authorization: Bearer <access_token>
```

Response 200

```json
{
    "status": "ok",
    "ctf_state": "active"
}
```

Errors:

- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 404 `stack not found`
- 503 `stack feature disabled` or `stack provisioner unavailable`
- If `ctf_state` is `not_started`, the response only includes `ctf_state`.
