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

Example stream:

```
event: ready
data: {}

event: scoreboard
data: {"scope":"all","reason":"submission_correct","ts":"2026-02-27T18:00:00Z"}
```

### Client Reconnect

SSE connections can be closed by server or proxy timeouts. Clients should be
prepared to reconnect and re-subscribe to `/api/scoreboard/stream`.

Example (browser EventSource):

```js
let es

const connect = () => {
    es = new EventSource('/api/scoreboard/stream')

    es.addEventListener('scoreboard', () => {
        fetch('/api/leaderboard?division_id=1')
        fetch('/api/leaderboard/teams?division_id=1')
        fetch('/api/timeline?division_id=1')
        fetch('/api/timeline/teams?division_id=1')
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
