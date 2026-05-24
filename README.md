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
    <img src="https://github.com/nullforu/smctf-docs/blob/main/src/content/docs/smctf/images/10-theme/image-1.png?raw=true" alt="SMCTF Preview" width="45%" />
    <img src="https://github.com/nullforu/smctf-docs/blob/main/src/content/docs/smctf/images/10-theme/image-3.png?raw=true" alt="SMCTF Preview" width="45%" />
</div>

<br />

<div align="center">
    <a href="https://ctf.null4u.cloud/">
        Docs
    </a>
    | <strong>Backend</strong> |
    <a href="https://github.com/swualabs/sandboxd-o">
        Container Orchestrator
    </a>
    |
    <a href="https://github.com/nullforu/smctfe">
        Frontend
    </a>
    |
    <a href="https://github.com/nullforu/smctf-infra">
        Infrastructure
    </a>
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

See [SMCTF Docs](https://ctf.null4u.cloud/smctf/) for more details. This README only provides a brief overview.

<!-- ### Available/Stable features:

- AuthN/AuthZ (JWT), including registration keys management
- Challenge management (Jeopardy CTF style, See [`ctf_service.go`](./internal/service/ctf_service.go) for a list of categories.)
- Flag submission with rate limiting
- Scoreboard and Timeline (Redis caching support)
- User profile with statistics (Some implementations are still WIP)
- Logging middleware with file logging, structured JSON logging and OpenMetrics endpoint support
    - Ref Issue: [#9](https://github.com/nullforu/smctf/issues/9), PR: [#10](https://github.com/nullforu/smctf/pull/10), PR: [#39](https://github.com/nullforu/smctf/pull/39)
- User and Team management (WIP)
    - Ref Issue: [#11](https://github.com/nullforu/smctf/issues/11), [#22](https://github.com/nullforu/smctf/issues/22), PR: [#12](https://github.com/nullforu/smctf/pull/12), [#15](https://github.com/nullforu/smctf/pull/15), [#23](https://github.com/nullforu/smctf/pull/23)
- Dynamic scoring (ref: [CTFd - Dynamic Value](https://docs.ctfd.io/docs/custom-challenges/dynamic-value/))
    - Ref Issue: [#14](https://github.com/nullforu/smctf/issues/14), PR: [#16](https://github.com/nullforu/smctf/pull/16)
- ~~UI customization and detailed configuration options (WIP)~~
    - ~~Ref Issue: [#18](https://github.com/nullforu/smctf/issues/18), PR: [#19](https://github.com/nullforu/smctf/pull/19)~~
    - Frontend has been moved to a separate repository ([nullforu/smctfe](https://github.com/nullforu/smctfe))
- Challenge file upload/download support via AWS S3 Presigned URL
    - Ref Issue: [#20](https://github.com/nullforu/smctf/issues/20), PR: [#21](https://github.com/nullforu/smctf/pull/21)
- Per challenge individual VM(instance/VM) provisioning support via Kubernetes
    - Ref PR: [#25](https://github.com/nullforu/smctf/pull/25), See [container-orchestrator-k8s](https://github.com/nullforu/container-orchestrator-k8s) and [docs](https://ctf.null4u.cloud/container-orchestrator/) for more details.
- ... and more! (See [docs](https://github.com/nullforu/smctf-docs) for more details) -->

### Planned/Upcoming features:

Also, the following features are planned to be implemented. see [issues](https://github.com/nullforu/smctf/issues) for more details.

- (WIP) Systematized admin dashboard and behavior log/monitoring system integration
- ... and more features to be added.

## Tech VMs

- Backend: [Go](https://go.dev/), [Gin](https://github.com/gin-gonic/gin), [Bun ORM](https://bun.uptrace.dev/)
- Container Orchestrator: [Go (nullforu/container-orchestrator-k8s)](https://github.com/nullforu/container-orchestrator-k8s)
- Frontend: React [(nullforu/smctfe)](https://github.com/nullforu/smctfe)
- Database, Cache: [PostgreSQL](https://www.postgresql.org/)(instead of MySQL/MariaDB), [Redis](https://redis.io/)
- Testing: [Testcontainers for Go](https://github.com/testcontainers/testcontainers-go)
- Infra: AWS, EKS, Helm, Terraform, Cloudflare, etc. (See [nullforu/smctf-infra](https://github.com/nullforu/smctf-infra) for more details)

## Installation and Usage

See [docs](https://ctf.null4u.cloud) for more details. This README only provides a quick start guide.

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
SUBMIT_WINDOW=1m
SUBMIT_MAX=10

# Cache
TIMELINE_CACHE_TTL=60s
LEADERBOARD_CACHE_TTL=60s

# VM (Container Orchestrator)
VMS_ENABLED=true
VMS_MAX_SCOPE=team
VMS_MAX_PER=3
VMS_ORCHESTRATOR_BASE_URL=http://localhost:8081
VMS_ORCHESTRATOR_TIMEOUT=5s
VMS_CREATE_WINDOW=1m
VMS_CREATE_MAX=1

# Logging
LOG_DIR=logs
LOG_FILE_PREFIX=app
LOG_MAX_BODY_BYTES=1048576

# Bootstrap
BOOTSTRAP_ADMIN_TEAM=true
BOOTSTRAP_ADMIN_USER=true
BOOTSTRAP_ADMIN_USERNAME=admin
BOOTSTRAP_ADMIN_EMAIL=
BOOTSTRAP_ADMIN_PASSWORD=

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
> Make sure to change `JWT_SECRET` to a secure random string in production!

After setting up the environment variables, build and run the server:

```shell
git clone https://github.com/nullforu/smctf.git

go mod download
go build -o smctf ./cmd/server
./smctf

# or: go run ./cmd/server
```

> [!NOTE]
>
> Running in Docker environment will be supported in the future.

**Logging Schema**

```json
{
    "$schema": "https://json-schema.org/draft/2020-12/schema",
    "title": "SMCTF Log Event",
    "type": "object",
    "additionalProperties": true,
    "required": ["ts", "level", "msg", "app"],
    "properties": {
        "ts": {
            "type": "string",
            "format": "date-time",
            "description": "RFC3339 timestamp with timezone"
        },
        "level": {
            "type": "string",
            "enum": ["debug", "info", "warn", "error"]
        },
        "msg": { "type": "string" },
        "app": { "type": "string" },
        "legacy": { "type": "boolean" },
        "error": {},
        "stack_trace": { "type": "string" },
        "http": {
            "type": "object",
            "additionalProperties": true,
            "properties": {
                "method": { "type": "string" },
                "path": { "type": "string" },
                "status": { "type": "integer" },
                "latency": { "type": "string" },
                "ip": { "type": "string" },
                "query": { "type": "string" },
                "user_agent": { "type": "string" },
                "content_type": { "type": "string" },
                "content_length": { "type": "integer" },
                "user_id": { "type": "integer" },
                "body": { "type": "string" }
            }
        }
    }
}
```

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

## YAML-to-SQL (Teams/Challenges)

If you want to generate SQL from a YAML file that explicitly defines teams and challenges (and optional per-team users), use the YAML generator:

```shell
# python3 -m pip install -r ./scripts/generate_yaml_sql/requirements.txt
python3 ./scripts/generate_yaml_sql/main.py --data ./scripts/generate_yaml_sql/defaults/data.yaml
```

You can also use the wrapper script:

```shell
./scripts/generate_yaml_sql.sh --data ./scripts/generate_yaml_sql/defaults/data.yaml
```

CLI options:

- `--data`: path to data YAML (teams/challenges). Required.
- `--settings`: path to settings YAML (security/constraints). Merged over defaults.
- `--output`: override output SQL file path. Defaults to `output.sql`.

Defaults live in `./scripts/generate_yaml_sql/defaults/`.

> [!WARNING]
>
> **This script refuses to run if `teams`, `challenges`, or `users` tables are not empty.**

## FAQ, Troubleshooting

(Not yet)

See [SMCTF Docs](https://ctf.null4u.cloud/faq/) for more details.

## License

This project is licensed under the MIT License. See the [LICENSE](./LICENSE) file for details.

## Contributors

| Name/GitHub                          | Role            | Affiliation                           |
| ------------------------------------ | --------------- | ------------------------------------- |
| [@yulmwu](https://github.com/yulmwu) | Main maintainer | Semyeong Computer High School, Null4U |

... and more [Null4U](https://github.com/nullforu) members.

<!-- ## Too Much Information (Some excerpts)

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
> 이거 유지보수할 사람이 하나밖에 없는게 단점.. Null4U에 종속시키고 졸업할 예정이니 후배님들이 알아서 잘 배워서 유지보수 해주길 바람. -->

[^1]: SMCH: Semyeong Computer High School (세명컴퓨터고등학교)

[^2]: SMCH(SMC) + CTF = SMCTF
