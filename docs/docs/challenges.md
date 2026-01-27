---
title: Challenges
nav_order: 4
---

## List Challenges

`GET /api/challenges`

Response 200

```json
[
    {
        "id": 1,
        "title": "Warmup",
        "description": "...",
        "category": "Web",
        "points": 100,
        "is_active": true
    }
]
```

---

## Submit Flag

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
