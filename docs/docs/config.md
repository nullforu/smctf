---
title: Config
nav_order: 7
---

## Get Site Configuration

`GET /api/config`

Response 200

```json
{
    "title": "Welcome to SMCTF.",
    "description": "Check out the repository for setup instructions.",
    "updated_at": "2026-01-26T12:00:00Z"
}
```

Notes:

- Response includes `ETag` and `Cache-Control: public, max-age=60` for caching.

Errors:

- 500 `internal error`
