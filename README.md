# Secure Observable Go Backend Starter Kit

[![CI](https://github.com/sandeepkv93/secure-observable-go-backend-starter-kit/actions/workflows/ci.yml/badge.svg)](https://github.com/sandeepkv93/secure-observable-go-backend-starter-kit/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.24.13-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Production-oriented Go backend starter with:

- Google OAuth login
- Cookie-based JWT session flow (access + refresh)
- Session/device management APIs (`/api/v1/me/sessions`)
- RBAC authorization
- OpenTelemetry metrics, traces, and logs
- Local tri-signal stack (Grafana + Tempo + Loki + Mimir + OTel Collector)
- Bazel + Gazelle + Task + Wire development workflow

## What This Repository Provides

- API server in `cmd/api`
- Operational CLIs in `cmd/migrate`, `cmd/seed`, `cmd/loadgen`, `cmd/obscheck`
- Layered internal packages (`internal/*`) with DI composition through Wire
- Docker compose local stack for DB + observability
- CI + local hooks enforcing build/test/generation hygiene

## Architecture Overview

Request path:

1. Chi router + middleware chain (`internal/http/router`)
2. Handler layer (`internal/http/handler`)
3. Service layer (`internal/service`)
4. Repository layer (`internal/repository`)
5. GORM + Postgres (`internal/database`)

Cross-cutting:

- Security middleware for headers, CSRF, request ID, rate limiting
- Structured logging with trace/span correlation fields
- OTel tracing, metrics (with exemplars), and logs export via collector

Dependency Injection:

- Providers and wiring in `internal/di`
- Regenerated via `task wire`
- Checked via `task wire-check`

```mermaid
flowchart LR
    User[Web or API Client] --> Router[Chi Router + Middleware]
    Router --> Handlers[HTTP Handlers]
    Handlers --> Services[Service Layer]
    Services --> Repos[Repository Layer]
    Repos --> DB[(PostgreSQL)]
    Services --> Redis[(Redis)]

    Handlers --> OAuth[Google OAuth Provider]
    OAuth --> Handlers

    Router -. request logs, metrics, traces .-> OTelSDK[OTel SDK]
    Services -. cache and auth metrics .-> OTelSDK
    Repos -. db telemetry .-> OTelSDK

    OTelSDK --> Collector[OTel Collector]
    Collector --> Tempo[Tempo Traces]
    Collector --> Loki[Loki Logs]
    Collector --> Mimir[Mimir Metrics]

    Grafana[Grafana] --> Tempo
    Grafana --> Loki
    Grafana --> Mimir

    Loadgen[cmd/loadgen] --> Router
    Obscheck[cmd/obscheck] --> Grafana
```

## Quick Start

Prerequisites:

- Go `1.24.13`
- [Task](https://taskfile.dev/)
- [Bazelisk](https://github.com/bazelbuild/bazelisk)
- Docker + Docker Compose

Run locally:

```bash
task docker-up
task migrate
task seed
task run
```

Useful commands:

```bash
task test
task ci
task obs-generate-traffic
task obs-validate
```

## Documentation

- Project guide (full documentation): `docs/project-guide.md`
- Architecture and flow diagrams: `docs/diagrams.md`
- Audit taxonomy: `docs/audit-taxonomy.md`

Key folders:

- API server: `cmd/api`
- Internal app packages: `internal/`
- Configuration and observability stack: `configs/`
- Integration tests: `test/integration/`
- Task definitions: `taskfiles/`

## License

MIT. See `LICENSE`.
