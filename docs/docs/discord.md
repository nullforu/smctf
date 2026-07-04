---
title: Discord
nav_order: 12
---

Notes:

- Discord account linking uses the OAuth2 authorization-code flow on the backend (`identify`, optionally `guilds.join`).
- Guild membership and role changes are performed by a separate **invite-bot** server over an internal HTTP API; the backend never holds the Discord bot token. The bot token and guild id live on the bot server only; the role and announcement channel are configured per division in the backend and passed to the bot per request.
- Access/refresh tokens are **not** stored. Role granting uses the bot token on the bot server, not the user's OAuth token.
- The acting user is identified by the JWT `access_token` cookie. `connect`, `callback`, and `status` require login; `sync-role` and `unlink` additionally require an unblocked (active) user.
- For authenticated `POST`, `PUT`, `PATCH`, and `DELETE` requests, send both `csrf_token` cookie and matching `X-CSRF-Token` header.
- All endpoints return `503 discord feature disabled` when `DISCORD_ENABLED=false`.

## Per-Division Discord Configuration

The Discord role granted on link and the solve-announcement channel are configured
**per division** (Admin > Divisions, `discord_role_id` and
`discord_announce_channel_id`), not as bot environment variables. Both are
optional. The backend resolves the acting user's division and passes the relevant
role id / channel id to the invite-bot on each request; the bot no longer holds a
`DISCORD_VERIFIED_ROLE_ID` or `DISCORD_ANNOUNCE_CHANNEL_ID`.

- **Role grant** — on link, the member is granted the division's `discord_role_id`.
  If the division has no role configured, linking is still marked `VERIFIED` and no
  role is granted.
- **Announcement channel** — solves post to the division's
  `discord_announce_channel_id`. If unset, that division's announcements are
  disabled.

## Solve Announcements & Nickname Sync

When `DISCORD_ENABLED=true`, the backend performs two best-effort side effects
through the invite-bot (failures are logged, never surfaced to the user):

- **Solve announcements** — on a correct flag submission, a message is sent to the
  solving user's division announcement channel (no-op when the division has none):
    - Normal solve: `**division_team** — username solved **Challenge Title**`
    - First blood: prefixed with 🩸 `First Blood!`

    Announcements are independent of whether the solving user has linked their own
    Discord account.

- **Guild nickname sync** — when a user links Discord or changes their username,
  the linked guild member's nickname is set to `division_team_username`. Because
  `username`, team `name`, and division `name` are each capped at 10 characters,
  this always fits Discord's 32-character nickname limit
  (10 + 10 + 10 + two underscores).

## Discord Status Schema

`discordStatusResponse` fields (omitted when empty):

- `connected`: whether the current user has a linked Discord account.
- `discord_user_id`: linked Discord user id (snowflake).
- `discord_username`: Discord unique handle.
- `discord_global_name`: Discord display name.
- `discord_avatar`: Discord avatar hash.
- `role_status`: one of `CONNECTED`, `VERIFIED`, `NOT_IN_GUILD`, `ROLE_FAILED`, `REVOKED`, `LEFT_GUILD`.
- `connected_at`, `verified_at`: timestamps (UTC).
- `invite_url`: guild invite link to show users who are not yet in the guild (from `DISCORD_INVITE_URL`).

## Start Linking

`GET /api/discord/connect`

Headers

```
Cookie: access_token=<jwt>
```

Response 302 — redirect to the Discord authorize URL. The backend generates a single-use `state` (random, TTL `DISCORD_STATE_TTL`) bound to the current user and stores it in Redis.

```
Location: https://discord.com/oauth2/authorize?response_type=code&client_id=...&scope=identify+guilds.join&state=...&redirect_uri=...&prompt=consent
```

Errors:

- 401 `invalid token` or `missing access_token cookie`
- 503 `discord feature disabled`
- 500 internal error

---

## OAuth Callback

`GET /api/discord/callback?code=<code>&state=<state>`

Headers

```
Cookie: access_token=<jwt>
```

Response 302 — redirect to `DISCORD_SUCCESS_REDIRECT` with a `?discord=<result>` query. If `DISCORD_SUCCESS_REDIRECT` is empty, returns `200 {"discord":"<result>"}` instead.

`<result>` values:

- `verified`: account linked, joined the guild, verified role granted.
- `connected_not_joined`: linked, but not a guild member (`NOT_IN_GUILD` / `LEFT_GUILD`).
- `role_failed`: linked, but the role could not be granted (bot permission / role hierarchy).
- `already_linked`: that Discord account is already linked to another user.
- `state_invalid`: `state` was missing, expired, forged, or did not match the user.
- `error`: token exchange / profile fetch failed, or another unexpected error.

```
Location: https://smctf.example.com/profile?discord=verified
```

Callback behavior:

- `state` is validated with `GETDEL` and must match the user that started `connect` (CSRF protection); invalid `state` redirects with `discord=state_invalid`.
- The code is exchanged for an access token, then `/users/@me` is read to capture `discord_user_id`, `username`, `global_name`, `avatar`.
- The connection is upserted. `discord_user_id` is unique — reusing one already linked elsewhere redirects with `discord=already_linked`.
- When `DISCORD_AUTO_JOIN=true`, the backend asks the invite-bot to add the user to the guild (`guilds.join`) using the access token, which is forwarded once and never stored.
- The verified role is then requested through the invite-bot; `role_status` is persisted as `VERIFIED`, `NOT_IN_GUILD`, or `ROLE_FAILED`.

Errors:

- 401 `invalid token` or `missing access_token cookie`
- 503 `discord feature disabled`

(Other failures are surfaced as the `?discord=<result>` redirect query above, not as HTTP error codes.)

---

## Get Link Status

`GET /api/discord/status`

Headers

```
Cookie: access_token=<jwt>
```

Response 200 — linked

```json
{
    "connected": true,
    "discord_user_id": "900000000000000001",
    "discord_username": "neo",
    "discord_global_name": "Neo",
    "discord_avatar": "a1b2c3d4e5f6",
    "role_status": "VERIFIED",
    "connected_at": "2026-06-23T11:00:00Z",
    "verified_at": "2026-06-23T11:00:02Z",
    "invite_url": "https://discord.gg/example"
}
```

Response 200 — not linked

```json
{
    "connected": false,
    "invite_url": "https://discord.gg/example"
}
```

Errors:

- 401 `invalid token` or `missing access_token cookie`
- 503 `discord feature disabled`
- 500 internal error

---

## Re-Check Role

`POST /api/discord/sync-role`

Headers

```
Cookie: access_token=<jwt>
X-CSRF-Token: <csrf>
```

Re-attempts guild join / verified-role grant for an already-linked user (used after the user joins the guild). Returns the updated `discordStatusResponse`.

Response 200 — same schema as `GET /api/discord/status` (linked).

Errors:

- 401 `invalid token` or `missing access_token cookie`
- 403 `user blocked`
- 404 `discord account not connected`
- 429 `discord rate limited`
- 503 `discord feature disabled` or `discord bot server unavailable`
- 500 internal error

---

## Unlink

`DELETE /api/discord/unlink`

Headers

```
Cookie: access_token=<jwt>
X-CSRF-Token: <csrf>
```

Response 200

```json
{
    "status": "ok"
}
```

Unlink behavior:

- The invite-bot is asked to kick the user from the guild. This is best-effort: any error (bot down, missing permission, already gone) is logged and ignored.
- The connection row is then deleted.

Errors:

- 401 `invalid token` or `missing access_token cookie`
- 403 `user blocked`
- 404 `discord account not connected`
- 503 `discord feature disabled`
- 500 internal error

---

## Configuration (ENV)

Discord service options (backend):

- `DISCORD_ENABLED` (default: `false`)
- `DISCORD_CLIENT_ID`, `DISCORD_CLIENT_SECRET`
- `DISCORD_REDIRECT_URI` — must match a redirect registered in the Discord Developer Portal exactly.
- `DISCORD_OAUTH_SCOPES` (default: `identify guilds.join`)
- `DISCORD_STATE_TTL` (default: `5m`)
- `DISCORD_OAUTH_TIMEOUT` (default: `10s`)
- `DISCORD_SUCCESS_REDIRECT` — frontend page to return to after the callback.
- `DISCORD_INVITE_URL` — guild invite shown to users not yet in the guild.
- `DISCORD_AUTO_JOIN` (default: `true`)
- `DISCORD_BOT_BASE_URL` (default: `http://localhost:8083`)
- `DISCORD_BOT_SECRET` — shared bearer secret for the invite-bot internal API (must equal the bot's `DISCORD_INTERNAL_SECRET`).
- `DISCORD_BOT_TIMEOUT` (default: `5s`)

The verified role and solve-announcement channel are **not** environment
variables — configure them per division (Admin > Divisions).

Validation rules when `DISCORD_ENABLED=true`:

- `DISCORD_CLIENT_ID` and `DISCORD_CLIENT_SECRET` must be set
- `DISCORD_REDIRECT_URI` must not be empty
- `DISCORD_OAUTH_SCOPES` must not be empty
- `DISCORD_BOT_BASE_URL` must not be empty
- `DISCORD_STATE_TTL > 0`
- `DISCORD_BOT_TIMEOUT > 0`
- `DISCORD_OAUTH_TIMEOUT > 0`

The Discord bot token and guild id are configured on the invite-bot server, not here.
The verified role and announcement channel are configured per division in the backend.
