All routes require `Authorization: Bearer <DISCORD_INTERNAL_SECRET>`.

| Method   | Path                                   | Body                        | Success                      |
| -------- | -------------------------------------- | --------------------------- | ---------------------------- |
| `PUT`    | `/internal/guild/members/:userId`      | `{ "access_token": "..." }` | `204` (join)                 |
| `PUT`    | `/internal/guild/members/:userId/role` | –                           | `204` (grant role)           |
| `DELETE` | `/internal/guild/members/:userId`      | –                           | `204` (kick)                 |
| `GET`    | `/internal/guild/members/:userId`      | –                           | `200 { in_guild, has_role }` |
| `GET`    | `/healthz`                             | –                           | `200`                        |

Errors return `{ "error": "...", "code": "..." }` where `code` is one of `NOT_IN_GUILD`, `BOT_PERMISSION`, `RATE_LIMITED`, `INVALID`, `UPSTREAM`.

```bash
npm install
cp .env.example .env

npm run dev
npm run build
npm start
```
