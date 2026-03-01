---
title: Leaderboard & Timeline
nav_order: 5
---

## Get Leaderboard

`GET /api/leaderboard?division_id={id}`

`division_id` is required.

Response 200

```json
{
    "challenges": [
        {
            "id": 1,
            "title": "pwn-101",
            "category": "Pwn",
            "points": 300
        }
    ],
    "entries": [
        {
            "user_id": 1,
            "username": "user1",
            "score": 300,
            "solves": [
                {
                    "challenge_id": 1,
                    "solved_at": "2026-01-24T12:00:00Z",
                    "is_first_blood": true
                }
            ]
        }
    ]
}
```

Returns all users in the division sorted by score (descending).
`solves` includes earliest solve timestamp per challenge and `is_first_blood` for the first solver.
Blocked users are excluded from leaderboard scores and solves.

Errors:

- 400 `invalid input` (`division_id` required or invalid)

---

## Get Team Leaderboard

`GET /api/leaderboard/teams?division_id={id}`

Response 200

```json
{
    "challenges": [
        {
            "id": 1,
            "title": "pwn-101",
            "category": "Pwn",
            "points": 300
        }
    ],
    "entries": [
        {
            "team_id": 1,
            "team_name": "서울고등학교",
            "score": 1200,
            "solves": [
                {
                    "challenge_id": 1,
                    "solved_at": "2026-01-24T12:00:00Z",
                    "is_first_blood": true
                }
            ]
        }
    ]
}
```

Returns all teams in the division sorted by score (descending).
`solves` includes earliest solve timestamp per challenge and `is_first_blood` for the first solver.
Blocked users are excluded from team scores and solves.

Errors:

- 400 `invalid input` (`division_id` required or invalid)

---

## Get Timeline

`GET /api/timeline?division_id={id}`

Response 200

```json
{
    "submissions": [
        {
            "timestamp": "2026-01-24T12:00:00Z",
            "user_id": 1,
            "username": "user1",
            "points": 300,
            "challenge_count": 2
        }
    ]
}
```

Returns all submissions in the division teamed by user and 10 minute intervals.
If multiple challenges are solved by the same user within 10 minutes, they are teamed together with cumulative points and challenge count.
`points` is dynamically calculated based on solves.
Blocked users are excluded.

Errors:

- 400 `invalid input` (`division_id` required or invalid)

---

## Get Team Timeline

`GET /api/timeline/teams?division_id={id}`

Response 200

```json
{
    "submissions": [
        {
            "timestamp": "2026-01-24T12:00:00Z",
            "team_id": 1,
            "team_name": "서울고등학교",
            "points": 300,
            "challenge_count": 2
        }
    ]
}
```

Returns all submissions in the division teamed by team and 10 minute intervals.

`points` is dynamically calculated based on solves.
Blocked users are excluded.

Errors:

- 400 `invalid input` (`division_id` required or invalid)

---

## Scoreboard Stream (SSE)

`GET /api/scoreboard/stream`

Opens a Server-Sent Events (SSE) stream that notifies clients when the scoreboard
data has been rebuilt and cached. This endpoint is public (no auth).

### Events

- `ready`: sent immediately after connection is established.
- `scoreboard`: emitted after caches are refreshed. Clients should re-fetch
  `/api/leaderboard`, `/api/leaderboard/teams`, `/api/timeline`, and
  `/api/timeline/teams` with the same `division_id`.

Payload notes:

- `division_ids` is included when the rebuild targets specific divisions.
- If `division_ids` is omitted, the rebuild applies to all divisions.
- `division_ids` may contain multiple IDs when the server debounces events and merges
  updates for more than one division.
- `scope` is `"division"` when `division_ids` is present, otherwise `"all"`.
- `reason` may be `"batch"` when multiple event reasons were merged during debounce.

Payload schema:

```json
{
    "scope": "division",
    "reason": "submission_correct",
    "division_ids": [1, 2],
    "ts": "2026-02-27T18:00:00Z"
}
```

## Scoreboard-Affecting APIs

The following API actions trigger a scoreboard SSE event. Each event includes a
`reason` and either `division_ids` (division-scoped) or `scope: "all"`.

| Action              | API                                  | Reason                | Scope    | Notes                                          |
| ------------------- | ------------------------------------ | --------------------- | -------- | ---------------------------------------------- |
| User registration   | `POST /api/register`                 | `user_registered`     | division | Uses the new user's division.                  |
| User profile update | `PUT /api/me`                        | `user_profile_update` | division | Uses the current user's division.              |
| Correct submission  | `POST /api/challenges/{id}/submit`   | `submission_correct`  | division | Uses the submitting user's division.           |
| Challenge created   | `POST /api/admin/challenges`         | `challenge_created`   | all      | Challenges are shared across divisions.        |
| Challenge updated   | `PUT /api/admin/challenges/{id}`     | `challenge_updated`   | all      | Challenges are shared across divisions.        |
| Challenge deleted   | `DELETE /api/admin/challenges/{id}`  | `challenge_deleted`   | all      | Challenges are shared across divisions.        |
| Move user team      | `PUT /api/admin/users/{id}/team`     | `user_team_moved`     | division | Includes both old and new division if changed. |
| Block user          | `POST /api/admin/users/{id}/block`   | `user_blocked`        | division | Uses the affected user's division.             |
| Unblock user        | `POST /api/admin/users/{id}/unblock` | `user_unblocked`      | division | Uses the affected user's division.             |
| Create team         | `POST /api/admin/teams`              | `team_created`        | division | Uses the new team's division.                  |

Example stream:

```
event: ready
data: {}

event: scoreboard
data: {"scope":"division","reason":"submission_correct","division_ids":[1],"ts":"2026-02-27T18:00:00Z"}

event: scoreboard
data: {"scope":"division","reason":"batch","division_ids":[1,2],"ts":"2026-02-27T18:00:01Z"}
```

### Client Reconnect

SSE connections can be closed by server or proxy timeouts. Clients should be
prepared to reconnect and re-subscribe to `/api/scoreboard/stream`.

Example (browser EventSource):

```js
let es

const connect = () => {
    es = new EventSource('/api/scoreboard/stream')

    es.addEventListener('scoreboard', (event) => {
        const payload = JSON.parse(event.data || '{}')

        if (payload.scope === 'all' || !payload.division_ids || payload.division_ids.length === 0) {
            const divisions = [1, 2] // for example, fetch division list from /api/divisions

            divisions.forEach((divisionId) => {
                fetch(`/api/leaderboard?division_id=${divisionId}`)
                fetch(`/api/leaderboard/teams?division_id=${divisionId}`)
                fetch(`/api/timeline?division_id=${divisionId}`)
                fetch(`/api/timeline/teams?division_id=${divisionId}`)
            })
            return
        }

        payload.division_ids.forEach((divisionId) => {
            fetch(`/api/leaderboard?division_id=${divisionId}`)
            fetch(`/api/leaderboard/teams?division_id=${divisionId}`)
            fetch(`/api/timeline?division_id=${divisionId}`)
            fetch(`/api/timeline/teams?division_id=${divisionId}`)
        })
    })

    es.onerror = () => {
        es.close()
        setTimeout(connect, 1000)
    }
}

connect()
```

### Proxy/Server Timeouts

If a reverse proxy is in front of the API, configure longer timeouts for the
SSE endpoint (`/api/scoreboard/stream`) while keeping normal API timeouts for
other routes.
