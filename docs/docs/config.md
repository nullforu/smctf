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
    "header_title": "CTF",
    "header_description": "Capture The Flag",
    "ctf_start_at": "2099-12-31T10:00:00Z",
    "ctf_end_at": "2099-12-31T18:00:00Z",
    "updated_at": "2026-01-26T12:00:00Z"
}
```

Notes:

- Response includes `ETag` and `Cache-Control: public, max-age=60` for caching.
- `ctf_start_at` and `ctf_end_at` are RFC3339 timestamps. Empty values mean the CTF is always active.

Errors:

- 500 `internal error`
