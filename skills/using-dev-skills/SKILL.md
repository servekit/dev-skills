---
name: using-dev-skills
description: "Use when starting any coding task — establishes how to find and pick the right dev-skills skill, requiring skill invocation before writing, editing, scaffolding, or reviewing code. Auto-injected at SessionStart; invoke manually only if context is lost."
---

# Using dev-skills

dev-skills is a set of **domain development standards** that Claude Code auto-loads as Skills. They are NOT optional reference material — when you work in their domain you MUST apply them.

## Scope — two tiers

dev-skills is multi-language, but its skills sit in two tiers:

- **General language standards** — `golang-development`, `rust-development`, `ts-development`, `opa-development`, `proto-development`, `gorm-cli-development`. Apply to any project in that language.
- **go-common service toolchain** — `golang-service-development` + `golang-service-docker`. Specific to the `github.com/servekit/go-common` ecosystem (grpcx / configx / lifecycle). Do NOT apply these two to non-go-common Go projects, and there is no non-Go docker scaffold here.

## The rule

Before writing, editing, scaffolding, or reviewing code in any domain below, invoke the matching **Skill** via the Skill tool FIRST, then follow it. If you even suspect a skill applies, invoke it to check — do not rationalize skipping it. Skill content overrides your defaults for that domain; the user's explicit instructions override skills. Apply the skill in full, not just the convenient parts.

## Quick routing — by primary signal

Match the task's primary signal, then invoke. When several signals match, also apply the "compose" rules below.

| Primary signal | Skill |
|---|---|
| Any `.go` file | **golang-development** — always, as Go baseline |
| New `-service` repo on `go-common`; or `pkg/handler` ↔ `internal/service` ↔ `store` layering; thirdcall; `lifecycle.Manager`; `new-service.sh` scaffold | **golang-service-development** |
| A Go project using GORM as its ORM — `store/models` · `gorm gen` · `store/generated` · `store/dal` · type-safe CRUD · transactions | **gorm-cli-development** |
| Any `.proto` · `buf.yaml`/`buf.gen.yaml` · protovalidate · field validation · CEL | **proto-development** |
| The project needs a policy/rules engine — any `.rego` · OPA · Rego · Gatekeeper · policy-as-code · admission control · Envoy/Terraform authz | **opa-development** |
| Any `.rs` | **rust-development** |
| Any `.ts`/`.tsx`/`.js`/`.mjs`/`.cjs` (incl. React/Vue) | **ts-development** |
| Dockerize / build image / compose-stack an **existing grpcx-based Go service** | **golang-service-docker** |

## Go is special — three skills that COMPOSE, not pick-one

Go has three overlapping skills. Each owns a different layer, and inside a `go-common` `-service` they all apply simultaneously:

| Skill | Layer it owns | Scope |
|---|---|---|
| **golang-development** | Go style — naming, errors, concurrency, testing, doc comments, gofmt/goimports/golangci-lint, Config-struct pointer fields | Every `.go`, in any repo |
| **golang-service-development** | Architecture — directory layout (`pkg/internal/cmd/api/gen`), handler↔service↔store split, thirdcall interface/impl, `lifecycle.Manager` resources, proto-enum→DB-int handling, `internal/jobs` cron, the scaffold workflow | `-service` repos whose `go.mod` contains `github.com/servekit/go-common` |
| **gorm-cli-development** | DB layer — `store/{models,generated,dal}`, `gorm gen`, type-safe CRUD, Typed Raw SQL, transactions, associations | Any project using `gorm.io/cli` |

A brand-new service with a proto API + a database is a **stack applied in order**:

**proto-development** (define the API) → **golang-service-development** (scaffold the service) → **gorm-cli-development** (the DAL) → **golang-development** (baseline Go on every `.go`) → **golang-service-docker** (package it).

Disambiguation:
- "Write/review this Go code" (no service/DB context) → `golang-development` alone.
- "Where does this file go — pkg or internal?" / "add a thirdcall" / "scaffold a new service" → `golang-service-development`.
- "Add a model / write a query / open a transaction" → `gorm-cli-development`.

## Router skills — load the entry, it delegates to sub-documents

Four skills use a router pattern: read their `SKILL.md` first, it routes you to the right sub-document. The sub-documents are NOT in the auto-index; the entry tells you when to load them.

- **proto-development** → `proto-best-practices` (writing `.proto`) · `buf-usage` (buf CLI, `buf.yaml`/`buf.gen.yaml`, managed mode, CI) · `protovalidate` (field rules, CEL, Go runtime interceptor).
- **opa-development** → Rego core lives in the entry; sub-docs: `rego-language` (advanced semantics) · `rego-testing-style` · `rego-builtins` (~170 functions) · `opa-operations` (CLI/server/K8s/Envoy/Terraform).
- **rust-development** → entry covers general Rust; sub-docs: `rust-unsafe-rules` · `rust-concurrency-async-rules` · `rust-optimization-testing`.
- **ts-development** → entry covers TS/JS; sub-docs: `react-development` (`.tsx`/React) · `vue-development` (`.vue` SFC).

## Docker boundary

- The `golang-service-development` scaffold ships a basic distroless `Dockerfile` as part of a new service.
- **golang-service-docker** is the dedicated packaging skill: deterministic multi-stage/multi-arch image, a single `docker-compose.yaml` (optional postgres/mysql/redis rendered inline), `grpc_health_probe` healthcheck, `.env.example`, and idempotent Makefile targets. Use it when the user wants to **dockerize / containerize / compose-package an existing grpcx Go service**. It does NOT generate app code, migrations, k8s/Helm, or CI.

## What dev-skills does NOT cover — do not route these here

- **go-common library API** — `configx`, `redisx`, `dbx`, `xerr`, `grpcx`, `signalx`, `cronx`, `lifecycle`, `logging`, etc. These are out of dev-skills' scope; they are documented in the **go-common repo's README** ([github.com/servekit/go-common](https://github.com/servekit/go-common)). If a task is purely "how do I call go-common API X", consult that README — don't improvise from memory.
- Migrations, Kubernetes/Helm manifests, and CI pipelines (golang-service-docker scope stops at image + compose).

## Priority when several skills apply

The **most specific** skill leads; baselines support. Example: writing DB code inside a Go service → **gorm-cli-development** leads, supported by **golang-service-development** (where the file lives) and **golang-development** (baseline Go, always).

## Complete inventory

A full auto-generated list of every skill (name + description + trigger keywords) is appended below this guide by the `session-start` hook. When the list and this routing table disagree: trust the **list** for *what exists*, and this **table** for *which one to pick*.
