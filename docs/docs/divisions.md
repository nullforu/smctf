---
title: Divisions
nav_order: 4
---

## List Divisions

`GET /api/divisions`

Response 200

```json
[
    {
        "id": 2,
        "name": "고등부",
        "discord_role_id": "1522163303982563458",
        "discord_announce_channel_id": "1522218332806447225",
        "created_at": "2026-01-26T12:00:00Z"
    }
]
```

Notes:

- Division identifiers are numeric `id` values (slugs are not supported).
- `discord_role_id` and `discord_announce_channel_id` are the per-division Discord
  role and announcement channel. They are omitted when unset and are managed by
  admins via Create / Update Division. See [Admin](admin.md) and [Discord](discord.md).
