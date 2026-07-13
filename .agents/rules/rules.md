# Org-Wide Agent Rules

This file defines conventions shared across all repos in this organisation.
Repo-level `.rules` files reference this document — read it fully before acting.

---

## Infrastructure Overview

All development runs inside Docker containers orchestrated with `docker-compose.yml`.
There is no expectation of anything being installed or runnable directly on the host.

### Shared infrastructure
A dedicated org-wide repo defines all shared resources — do not redefine these
in individual repos:
- **Traefik** reverse proxy and routing
- **JetStream** (NATS) messaging
- A shared external Docker network named `traefik` (and, where messaging is
  needed, `jetstream`) is sourced from that repo — never create a local
  `traefik` or `jetstream` network in a repo's compose file. Each web-exposed
  service joins `traefik` directly.

### Shared infrastructure contract
These values mirror the infra repo's `docker-compose.yml` (the source of truth —
do not diverge). They are the stable, non-secret facts a repo needs to route
traffic and connect to messaging.

**Traefik** (`traefik:v3.0`, Docker provider, dashboard at `local.nathejk.dk`)
- External network: `traefik` (declare it `external: true` in your compose).
- Entrypoints: `web` (:80) and `websecure` (:443), bound to `127.0.0.1` on the
  host.
- **No TLS/ACME is configured** (a `letsencrypt` volume is mounted but unused),
  so local services are served over **HTTP on :80** — use `http://` URLs. Do
  not add `tls` / `certresolver` labels; there is no resolver to satisfy them.
- Routers attach to all entrypoints by default, so **omit** the `entrypoints`
  label (the infra's own services do).
- Required per-service labels: `traefik.enable=true`,
  `traefik.docker.network=traefik`, and
  `traefik.http.routers.<repo>[-<svc>].rule=Host(`<sub>.local.nathejk.dk`)`.
  Add `traefik.http.services.<…>.loadbalancer.server.port` only when the
  container's web port is not 80 (e.g. MailHog → 8025).
- Hostnames: any subdomain of `local.nathejk.dk`.

**JetStream / NATS** (`nats:2.10-alpine -js`)
- Network: `jetstream` (declare it `external: true`).
- Client DSN (dev): `nats://jetstream:4222`; monitoring on `:8222`.
- A NATS UI (`nui`) is available at `jetstream.local.nathejk.dk`.

### Container architecture (standard web repos)
Most repos follow this pattern:

| Stage | Technology | Purpose |
|---|---|---|
| Frontend | Vue 3 (JS/TS) | Single page application |
| BFF | Go | Backend-for-frontend — serves the JSON API and (in prod) the SPA |
| Dev containers | Node or Go (multistage) | One per service, from shared Dockerfile |
| Production | Single bundled container | Frontend assets served by Go binary |

The `docker/Dockerfile` uses multistage builds. The frontend dev target is
`ui-dev` and the backend dev target is `base` (built on an `api-dev` Go toolchain
stage); intermediate `build` / `ui-builder` stages produce the artefacts baked
into the single `prod` target. Always use a dev target during development —
never build the `prod` image for local work.

---

## Docker & Compose conventions

- All services are defined in `docker-compose.yml` at repo root
- Development services must mount source code as volumes for hot reload
- Each service that serves web traffic declares **its own** Traefik config:
  - Join the external `traefik` network (in addition to the repo-local `local`
    network).
  - Set `traefik.enable`, `traefik.docker.network=traefik`, and a router
    `Host(...)` rule. Add a `loadbalancer.server.port` **only** when the
    container listens on a non-80 port (Traefik defaults to 80).
  - Use repo-scoped router/service names (e.g. `<repo>`, `<repo>-<service>`)
    to avoid collisions on the shared Traefik.
  - Never expose ports directly to the host — Traefik handles routing.
- Services reached only internally (e.g. the Go `api` via the frontend's dev
  proxy, the database) stay on `local` only and get no Traefik labels.
- Committed environment defaults live in `docker-compose.yml`; per-developer
  overrides and secrets go in `docker-compose.override.yml`, which is gitignored
- Never hardcode secrets or hostnames — always use environment variables

### Per-service Traefik label pattern
```yaml
<service>:
  networks:
    - local
    - traefik
  labels:
    traefik.enable: true
    traefik.docker.network: traefik
    traefik.http.routers.<repo>-<service>.rule: Host(`<sub>.<repo>.local.nathejk.dk`)
    # only when the container port is not 80:
    # traefik.http.services.<repo>-<service>.loadbalancer.server.port: <port>
networks:
  traefik:
    external: true
```

---

## Developer groups

| Group | Name | Access |
|---|---|---|
| **Core** | Production developers | Full repo access — may write and deploy production code |
| **Support** | Supporting systems | Access limited to infra/, tools/, scripts/, config/ — not application code |

When creating tasks or reviewing changes, note which group the work belongs to.

---

## Task board

Tasks are tracked using the file-based kanban system under `roadmap/tasks/`
(PRDs live alongside them in `roadmap/prd/`). Before picking up, creating, or
updating any task, read `roadmap/tasks/TASKS.md`. The `file-task-board` skill
describes the full workflow.

Commit message format: `task(<id>): <action> — <title>`
Actions: `create` · `pick up` · `update` · `done` · `reopen`

---

## Git conventions

- Branch format: `<task-id>-<short-slug>` — e.g. `003-add-jwt-middleware`
- Commits are small and focused — one logical change per commit
- Never commit directly to `main` — always branch and PR
- `.env` files are always gitignored — `.env.example` is always committed

---

## General agent behaviour

- Always work inside Docker — never run Node, Go, or build tools directly on the host
- Use `docker compose run --rm <service> <command>` for one-off commands
- When in doubt about the dev environment, check `docker-compose.yml` first
- If a shared resource (Traefik network, JetStream) is missing, the org-wide
  infra repo needs to be running — flag this rather than working around it
