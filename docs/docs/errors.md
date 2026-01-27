---
title: Error Format
nav_order: 7
---

## Common Response

```json
{
    "error": "message",
    "details": [{ "field": "field_name", "reason": "reason" }]
}
```

`details` is omitted when not applicable.

---

## Validation Errors (400)

```json
{
    "error": "invalid input",
    "details": [
        { "field": "email", "reason": "required" },
        { "field": "password", "reason": "required" }
    ]
}
```

---

## Auth Errors (401)

```json
{ "error": "invalid token" }
```

---

## Not Found (404)

```json
{ "error": "challenge not found" }
```

---

## Conflict (409)

```json
{ "error": "user already exists" }
```

---

## Rate Limit (429)

```json
{
    "error": "too many submissions",
    "rate_limit": {
        "limit": 10,
        "remaining": 0,
        "reset_seconds": 42
    }
}
```

Headers:

```
X-RateLimit-Limit
X-RateLimit-Remaining
X-RateLimit-Reset
```

---

## Forbidden (403)

```json
{ "error": "forbidden" }
```
