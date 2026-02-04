## SMCTF: CTF Platform for everyone, specialized for SMCH[^1]

<div align="left">
    <a href="https://github.com/nullforu/smctf/actions/workflows/backend-test-ci.yaml">
        <img src="https://github.com/nullforu/smctf/actions/workflows/backend-test-ci.yaml/badge.svg" alt="backend-test-ci" />
    </a>
    <a href="https://codecov.io/github/nullforu/smctf">
        <img src="https://codecov.io/github/nullforu/smctf/graph/badge.svg?token=T7HF44RDS8" alt="codecov" />
    </a>
</div>

<br />

<div align="center">
    <img src="./assets/1_challenges_light.jpeg" alt="SMCTF Preview" width="45%" />
    <img src="./assets/3_scoreboard_timeline_detail_dark.jpeg" alt="SMCTF Preview" width="45%" />
</div>

## About

**SMCTF**[^2] is a CTF platform developed by [Null4U](https://github.com/nullforu), a DevOps and Cloud Computing club at Semyeong Computer High School (SMCH).

When hosting CTF competitions within school security clubs such as [SCA](https://www.instagram.com/smc.sec_sca), we faced several challenges:

- Using existing open source CTF platforms involved a steep learning curve.
- They required complex initial configurations, such as plugins for provisioning individual instances or VMs for each challenge.
- Existing platforms were provided only as container images or source code, requiring us to design an architecture tailored to our infrastructure.
- We also found that logging, monitoring, and user management features were insufficient.

As a result, we decided to develop our own CTF platform as a long term project. We are releasing it as an open source project so that it can be used in various CTF competitions.

## Features

### Available/Stable features:

- AuthN/AuthZ (JWT), including registration keys management
- Challenge management (Jeopardy CTF style, See [`ctf_service.go`](./internal/service/ctf_service.go) for a list of categories.)
- Flag submission with rate limiting and HMAC verification
- Scoreboard and Timeline (Redis caching support)
- User profile with statistics (Some implementations are still WIP)
- Logging middleware with file logging and webhook support (e.g., Discord, Slack, etc.)
    - Supports queuing and batching for webhooks to prevent rate limiting issues, and splitting long messages.
    - Ref Issue: [#9](https://github.com/nullforu/smctf/issues/9), PR: [#10](https://github.com/nullforu/smctf/pull/10)
- User and Team management (WIP)
    - Ref Issue: [#11](https://github.com/nullforu/smctf/issues/11), [#22](https://github.com/nullforu/smctf/issues/22), PR: [#12](https://github.com/nullforu/smctf/pull/12), [#15](https://github.com/nullforu/smctf/pull/15), [#23](https://github.com/nullforu/smctf/pull/23)
- Dynamic scoring (ref: [CTFd - Dynamic Value](https://docs.ctfd.io/docs/custom-challenges/dynamic-value/))
    - Ref Issue: [#14](https://github.com/nullforu/smctf/issues/14), PR: [#16](https://github.com/nullforu/smctf/pull/16)
- UI customization and detailed configuration options (WIP)
    - Ref Issue: [#18](https://github.com/nullforu/smctf/issues/18), PR: [#19](https://github.com/nullforu/smctf/pull/19)
- Challenge file upload/download support via AWS S3 Presigned URL
    - Ref Issue: [#20](https://github.com/nullforu/smctf/issues/20), PR: [#21](https://github.com/nullforu/smctf/pull/21)

### Planned/Upcoming features:

Also, the following features are planned to be implemented. see [issues](https://github.com/nullforu/smctf/issues) for more details.

- Per challenge individual instance/VM provisioning support via AWS SDK (ECS Fargate or EC2 based)
- Multi language support (i18n) and RTL language support (for global service expansion)
- (WIP) Systematized admin dashboard and behavior log/monitoring system integration
- ... and more features to be added.

## Tech Stacks

- Backend: [Go](https://go.dev/), [Gin](https://github.com/gin-gonic/gin), [Bun ORM](https://bun.uptrace.dev/)
- Frontend: [Svelte](https://svelte.dev/)
- Database, Cache: [PostgreSQL](https://www.postgresql.org/)(instead of MySQL/MariaDB), [Redis](https://redis.io/)
- Testing: [Testcontainers for Go](https://github.com/testcontainers/testcontainers-go)
- Infra, CI/CD (TBD): AWS, Terraform, Cloudflare, GitHub Actions, etc.

## Installation and Usage

See [`/docs`](./docs) for more details. This README only provides a quick start guide.

> [!NOTE]
>
> PostgreSQL and Redis are required. if necessary, use Docker to run them locally. (for development/testing purposes only)
>
> ```shell
> docker compose -f docker-compose.db.yaml up -d
>
> # if `app_db` database does not exist, create it:
> PGPASSWORD=app_password psql -U app_user -d postgres -h localhost -c "CREATE DATABASE app_db;"
> ```
>
> If you need a remote DB server, refer to the configuration values ​​in [docker-compose.db.yaml](./docker-compose.db.yaml).
> tables, indexes, etc. will be automatically migrated when the server starts.

```shell
git clone https://github.com/nullforu/smctf.git
cd smctf

touch .env
```

And add the following environment variables to `.env` file (refer to [`.env.example`](.env.example)):

```ini
APP_ENV=production
HTTP_ADDR=:8080
SHUTDOWN_TIMEOUT=10s
AUTO_MIGRATE=true
# ... (other variables)
```

<details>
<summary>Click to expand <code>.env.example</code> file content. (default values)</summary>

```ini
# App
APP_ENV=local
HTTP_ADDR=:8080
SHUTDOWN_TIMEOUT=10s
AUTO_MIGRATE=true
BCRYPT_COST=12

# PostgreSQL
DB_HOST=localhost
DB_PORT=5432
DB_USER=app_user
DB_PASSWORD=app_password
DB_NAME=app_db
DB_SSLMODE=disable
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=10
DB_CONN_MAX_LIFETIME=30m

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_POOL_SIZE=20

# JWT
JWT_SECRET=change-me
JWT_ISSUER=smctf
JWT_ACCESS_TTL=24h
JWT_REFRESH_TTL=168h

# Security
FLAG_HMAC_SECRET=change-me-too
SUBMIT_WINDOW=1m
SUBMIT_MAX=10

# Cache
TIMELINE_CACHE_TTL=60s

# Logging
LOG_DIR=logs
LOG_FILE_PREFIX=app
LOG_MAX_BODY_BYTES=1048576
LOG_WEBHOOK_QUEUE_SIZE=1000
LOG_WEBHOOK_TIMEOUT=5s
LOG_WEBHOOK_BATCH_SIZE=20
LOG_WEBHOOK_BATCH_WAIT=2s
LOG_WEBHOOK_MAX_CHARS=1800
LOG_DISCORD_WEBHOOK_URL=
LOG_SLACK_WEBHOOK_URL=

# S3 Challenge Files
S3_ENABLED=false
S3_REGION=ap-northeast-2
S3_BUCKET=
S3_ACCESS_KEY_ID=
S3_SECRET_ACCESS_KEY=
S3_ENDPOINT=
S3_FORCE_PATH_STYLE=false
S3_PRESIGN_TTL=15m
```

</details>

> [!IMPORTANT]
>
> Make sure to change `JWT_SECRET` and `FLAG_HMAC_SECRET` to secure random strings in production!

After setting up the environment variables, build and run the server:

```shell
git clone https://github.com/nullforu/smctf.git

# builds frontend assets to ./frontend/dist (Backend will serve these static files)
source ./scripts/build_frontend.sh

go mod download
go build -o smctf ./cmd/server
./smctf

# or: go run ./cmd/server
```

> [!NOTE]
>
> Running in Docker environment will be supported in the future.
> Currently, please use local installation for development and testing. Requires Go and NodeJS, NPM installation.

## Testing

To run the tests, use the following command:

```shell
go test -v ./internal/...
# or with race detector, coverage options
go test -v -race -cover -coverprofile=coverage.out ./internal/...
```

Check the Codecov report for test coverage:

- https://codecov.io/github/nullforu/smctf

## Previews

<div align="center">
    <img src="./assets/1_challenges_light.jpeg" alt="SMCTF Preview" width="45%" />
    <img src="./assets/2_challenges_detail_dark.jpeg" alt="SMCTF Preview" width="45%" />
</div>

<div align="center">
    <img src="./assets/7_groups_dark.jpeg" alt="SMCTF Preview" width="45%" />
    <img src="./assets/5_scoreboard_group_timeline_light.jpeg" alt="SMCTF Preview" width="45%" />
</div>

<details>
<summary>See more preview/screenshots (Click)</summary>

<div align="center">
    <img src="./assets/0_main_light.jpeg" alt="SMCTF Preview" width="45%" />
    <img src="./assets/0_main_dark.jpeg" alt="SMCTF Preview" width="45%" />
</div>

<div align="center">
    <img src="./assets/1_challenges_dark.jpeg" alt="SMCTF Preview" width="45%" />
    <img src="./assets/1_challenges_light.jpeg" alt="SMCTF Preview" width="45%" />
</div>

<div align="center">
    <img src="./assets/2_challenges_detail_light.jpeg" alt="SMCTF Preview" width="45%" />
    <img src="./assets/2_challenges_detail_dark.jpeg" alt="SMCTF Preview" width="45%" />
</div>

<div align="center">
    <img src="./assets/3_scoreboard_timeline_light.jpeg" alt="SMCTF Preview" width="45%" />
    <img src="./assets/3_scoreboard_timeline_dark.jpeg" alt="SMCTF Preview" width="45%" />
</div>

<div align="center">
    <img src="./assets/3_scoreboard_timeline_detail_light.jpeg" alt="SMCTF Preview" width="45%" />
    <img src="./assets/3_scoreboard_timeline_detail_dark.jpeg" alt="SMCTF Preview" width="45%" />
</div>

<div align="center">
    <img src="./assets/4_scoreboard_leaderboard_light.jpeg" alt="SMCTF Preview" width="45%" />
    <img src="./assets/4_scoreboard_leaderboard_dark.jpeg" alt="SMCTF Preview" width="45%" />
</div>

<div align="center">
    <img src="./assets/5_scoreboard_group_timeline_light.jpeg" alt="SMCTF Preview" width="45%" />
    <img src="./assets/5_scoreboard_group_timeline_dark.jpeg" alt="SMCTF Preview" width="45%" />
</div>

<div align="center">
    <img src="./assets/6_scoreboard_group_leaderboard_light.jpeg" alt="SMCTF Preview" width="45%" />
    <img src="./assets/6_scoreboard_group_leaderboard_dark.jpeg" alt="SMCTF Preview" width="45%" />
</div>

<div align="center">
    <img src="./assets/7_groups_light.jpeg" alt="SMCTF Preview" width="45%" />
    <img src="./assets/7_groups_dark.jpeg" alt="SMCTF Preview" width="45%" />
</div>

<div align="center">
    <img src="./assets/8_groups_detail_light.jpeg" alt="SMCTF Preview" width="45%" />
    <img src="./assets/8_groups_detail_dark.jpeg" alt="SMCTF Preview" width="45%" />
</div>

<div align="center">
    <img src="./assets/9_users_light.jpeg" alt="SMCTF Preview" width="45%" />
    <img src="./assets/9_users_dark.jpeg" alt="SMCTF Preview" width="45%" />
</div>

<div align="center">
    <img src="./assets/10_users_detail_light.jpeg" alt="SMCTF Preview" width="45%" />
    <img src="./assets/10_users_detail_dark.jpeg" alt="SMCTF Preview" width="45%" />
</div>

<div align="center">
    <img src="./assets/11_profile_light.jpeg" alt="SMCTF Preview" width="45%" />
    <img src="./assets/11_profile_dark.jpeg" alt="SMCTF Preview" width="45%" />
</div>

<div align="center">
    <img src="./assets/12_admin_create_challenge_light.jpeg" alt="SMCTF Preview" width="45%" />
    <img src="./assets/12_admin_create_challenge_dark.jpeg" alt="SMCTF Preview" width="45%" />
</div>

<div align="center">
    <img src="./assets/13_admin_challenge_management_light.jpeg" alt="SMCTF Preview" width="45%" />
    <img src="./assets/13_admin_challenge_management_dark.jpeg" alt="SMCTF Preview" width="45%" />
</div>

<div align="center">
    <img src="./assets/14_admin_challenge_management_edit_light.jpeg" alt="SMCTF Preview" width="45%" />
    <img src="./assets/14_admin_challenge_management_edit_dark.jpeg" alt="SMCTF Preview" width="45%" />
</div>

<div align="center">
    <img src="./assets/15_admin_registration_keys_light.jpeg" alt="SMCTF Preview" width="45%" />
    <img src="./assets/15_admin_registration_keys_dark.jpeg" alt="SMCTF Preview" width="45%" />
</div>

<div align="center">
    <img src="./assets/16_admin_registration_keys_ip_light.jpeg" alt="SMCTF Preview" width="45%" />
    <img src="./assets/16_admin_registration_keys_ip_dark.jpeg" alt="SMCTF Preview" width="45%" />
</div>

<div align="center">
    <img src="./assets/17_admin_groups_light.jpeg" alt="SMCTF Preview" width="45%" />
    <img src="./assets/17_admin_groups_dark.jpeg" alt="SMCTF Preview" width="45%" />
</div>

<div align="center">
    <img src="./assets/18_mobile_sidebar_light.jpeg" alt="SMCTF Preview" width="45%" />
    <img src="./assets/18_mobile_sidebar_dark.jpeg" alt="SMCTF Preview" width="45%" />
</div>

</details>

## Dummy/Sample SQL Data

For testing purposes, you can populate the database with dummy data using the following script:

```shell
# python3 -m pip install -r ./scripts/generate_dummy_sql/requirements.txt
python3 ./scripts/generate_dummy_sql/main.py
```

You can also use the wrapper script:

```shell
./scripts/generate_dummy_sql.sh
```

Templates and YAML inputs are supported:

```shell
chmod +x ./scripts/generate_dummy_sql.sh

./scripts/generate_dummy_sql.sh --list-templates
./scripts/generate_dummy_sql.sh --template team_only.yaml --template high_solve_rate.yaml
./scripts/generate_dummy_sql.sh --data ./scripts/generate_dummy_sql/defaults/data.yaml --settings ./scripts/generate_dummy_sql/defaults/settings.yaml
```

CLI options:

- `--data`: path to data YAML (users/teams/challenges). Defaults to bundled `data.yaml`.
- `--settings`: path to settings YAML. Merged over defaults. Defaults to bundled `settings.yaml`.
- `--template`: template YAML to apply before settings (repeatable).
- `--output`: override output SQL file path.
- `--seed`: random seed for reproducible output.
- `--list-templates`: list bundled templates.

Available templates:

- `solo_only.yaml`: force users to have no team (no team join / no team assignment for registration keys)
- `team_only.yaml`: force users to always join a team and assign team on registration keys
- `high_solve_rate.yaml`: increase solve probability and attempt counts
- `low_solve_rate.yaml`: decrease solve probability and attempt counts
- `few_attempts.yaml`: lower number of attempts per user
- `many_attempts.yaml`: higher number of attempts per user

This will generate a `dummy.sql` file. You can then import this file into your PostgreSQL database:

```shell
# for docker-compose.db.yaml
PGPASSWORD=app_password psql -U app_user -d app_db -h localhost -f dummy.sql
```

Defaults live in `./scripts/generate_dummy_sql/defaults/` and can be overridden with your own YAML.
It provides sample challenges, 30 users (including admin), and random submissions data from the last ~48 hours.

> [!WARNING]
> 
> **This will TRUNCATE all tables in the database! Use only in development/test environments.**

## FAQ, Troubleshooting

(Not yet)

## License

This project is licensed under the MIT License. See the [LICENSE](./LICENSE) file for details.

## Contributors

| Name/GitHub                          | Role            | Affiliation                           |
| ------------------------------------ | --------------- | ------------------------------------- |
| [@yulmwu](https://github.com/yulmwu) | Main maintainer | Semyeong Computer High School, Null4U |

... and more [Null4U](https://github.com/nullforu) members.

## Too Much Information (Some excerpts)

```diff
> 백엔드 언어를 굳이 Go를 선택한 이유?

< 1. 기존에 쓰던 NodeJS의 NestJS 프레임워크는 너무 무거웠음
< (DI, 복잡한 구조와 런타임 데코레이터, 많은 빌트인 기능으로 인해 무겁고 운영상의 오버헤드가 있었음)
< => 프로젝트 특성 상 이벤트성으로 운영되는 경우가 많았기에 가벼운 프레임워크가 필요했음

< 2. 그렇다고 가벼운 ExpressJS 프레임워크는 너무 자유로워서 유지보수가 어렵다고 판단함
< Fastify도 고려했으나 익숙하지 않았음

< 3. 백엔드 개발을 위한 언어/런타임 중 다룰 수 있는 언어/런타임이 사실상 NodeJS와 Go 언어밖에 없었음
< (Python, Ruby, Java 등은 개인적으로 선호하지 않았음)

< 4. Go 언어는 컴파일링을 거치면 단일 바이너리로 배포 가능,
< (이론상) 빠름, 정적 타이핑, 쉬운 문법, 나름 생태계가 갖춰짐, 러닝 커브가 완만함
< Go를 처음 접했을 2019년 당시엔 Go 언어의 생태계가 살짝 부족하다고 판단하였으나, 현재는 어느정도 갖춰진 상태라고 판단하였음
< + 거기에 E2E TDD 관련 툴들도 나름 잘 갖춰져 있었음 (특히 testcontainers 등)

< 5. Gin, Fiber, Echo 등의 여러 웹 프레임워크가 있었으나 생태계가 가장 크고 안정적인 Gin 프레임워크를 선택함

< 6. ORM도 여러 후보를 고려했었으나 최종적으로 Bun을 선택하였음
```

```diff
> 프론트엔드 프레임워크를 기존에 쓰던 React에서 Svelte로 바꾼 이유?

< 1. React도 마찬가지로 좀 무거웠음 (의존성이 너무 많고 최종적으로 서빙되는 번들 크기가 좀 큰 듯)

< 2. Svelte는 컴파일 타임에 대부분의 작업이 처리되기 때문에 런타임 오버헤드가 적고,
< 결과물인 번들 크기가 작아지는 경향이 있음 + 거기에 그냥 써보고 싶었음 (5.0의 Rune 기능이 궁금했음)
< => 근데 살짝 후회중.. 굳이 고르라면 React가 더 나았을 듯
```

> \- 프로젝트의 유일 메인테이너이자 동아리 부장 [@yulmwu](https://github.com/yulmwu) 발췌 \-
>
> 이거 유지보수할 사람이 하나밖에 없는게 단점.. Null4U에 종속시키고 졸업할 예정이니 후배님들이 알아서 잘 배워서 유지보수 해주길 바람.

[^1]: SMCH: Semyeong Computer High School (세명컴퓨터고등학교)

[^2]: SMCH(SMC) + CTF = SMCTF
