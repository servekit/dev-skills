---
name: using-dev-skills
description: "Use when starting any coding task — establishes how to find and pick the right dev-skills skill, requiring skill invocation before writing, editing, scaffolding, or reviewing code. Auto-injected at SessionStart; invoke manually only if context is lost."
---

# Using dev-skills

dev-skills is a set of **domain development standards** that Claude Code auto-loads as Skills. They are NOT optional reference material — when you work in their domain you MUST apply them.

## Scope — two tiers

dev-skills is multi-language, but its skills sit in two tiers:

- **General language standards** — `golang-development`, `rust-development`, `ts-development`, `opa-development`, `proto-development`, `gorm-cli-development`. Apply to any project in that language.
- **Go `-service` toolchain** — `golang-service-development` + `golang-service-docker`. For Go services that follow the `-service` architecture (`pkg/internal/cmd` layout, grpcx, `lifecycle.Manager`). They default to the `github.com/servekit/go-common` library — use it preferentially; with another stack the architecture still applies, just adapt the API calls. (The docker scaffold is grpcx/Go-only.)

## The rule

Before writing, editing, scaffolding, or reviewing code in any domain below, invoke the matching **Skill** via the Skill tool FIRST, then follow it. If you even suspect a skill applies, invoke it to check — do not rationalize skipping it. Skill content overrides your defaults for that domain; the user's explicit instructions override skills. Apply the skill in full, not just the convenient parts.

## Quick routing — by primary signal

Match the task's primary signal, then invoke. When several signals match, also apply the "compose" rules below.

| Primary signal | Skill |
|---|---|
| Any `.go` file | **golang-development** — always loaded as the Go *baseline* (style/lint); NOT the lead when scaffolding a new service |
| **Creating / scaffolding a new Go service** — any gRPC / grpc-gateway / HTTP backend (named `*-service` by convention, but `pay`, `order`, `userapi` also count); the scaffold generates `go.mod`, so do NOT gate on it; or an existing service repo; or `pkg/handler` ↔ `internal/service` ↔ `store` layering; thirdcall; `lifecycle.Manager` | **golang-service-development** — LEAD for new services |
| A Go project using GORM as its ORM — `store/models` · `gorm gen` · `store/generated` · `store/dal` · type-safe CRUD · transactions | **gorm-cli-development** |
| Any `.proto` · `buf.yaml`/`buf.gen.yaml` · protovalidate · field validation · CEL | **proto-development** |
| The project needs a policy/rules engine — any `.rego` · OPA · Rego · Gatekeeper · policy-as-code · admission control · Envoy/Terraform authz | **opa-development** |
| Any `.rs` | **rust-development** |
| Any `.ts`/`.tsx`/`.js`/`.mjs`/`.cjs` (incl. React/Vue) | **ts-development** |
| Dockerize / build image / compose-stack an **existing grpcx-based Go service** | **golang-service-docker** |

### Creating a new Go service — default to golang-service-development

The most common routing mistake: a "create a Go project / service" request lands on `golang-development` alone. In this stack a new Go service is built with **golang-service-development** as the LEAD (scaffold + architecture); `golang-development` is only the baseline. Decide by **intent, not name**:

- The user wants a Go **service** — a long-running gRPC / grpc-gateway / HTTP backend, a microservice, anything with an API — → **golang-service-development LEADS**. The servekit convention names them `*-service`, but `pay`, `order`, `userapi` serve an API and qualify too. The scaffold creates `go.mod`, so do NOT gate on it (it doesn't exist yet at creation).
- The user wants a clearly non-service Go program — a CLI tool, a library, a script — → **golang-development** alone.
- Ambiguous ("create a golang project named X") → **default to golang-service-development** and confirm with the user. Loading it needlessly is cheap; scaffolding a service with only the Go style guide is expensive.

## Go is special — three skills that COMPOSE, not pick-one

Go has three overlapping skills. Each owns a different layer, and inside a `-service` repo they all apply simultaneously:

| Skill | Layer it owns | Scope |
|---|---|---|
| **golang-development** | Go style — naming, errors, concurrency, testing, doc comments, gofmt/goimports/golangci-lint, Config-struct pointer fields | Every `.go`, in any repo |
| **golang-service-development** | Architecture — directory layout (`pkg/internal/cmd/api/gen`), handler↔service↔store split, thirdcall interface/impl, `lifecycle.Manager` resources, proto-enum→DB-int handling, `internal/jobs` cron, the scaffold workflow | Go `-service` repos (go-common by default) |
| **gorm-cli-development** | DB layer — `store/{models,generated,dal}`, `gorm gen`, type-safe CRUD, Typed Raw SQL, transactions, associations | Any project using `gorm.io/cli` |

A brand-new `-service` is built in this order — **scaffold first, then layer on**:

1. **golang-service-development** — scaffold the skeleton with `new-service.sh`. It produces a runnable service with a baseline proto, the directory layout, and lifecycle wiring already in place.
2. **proto-development** — *if the API needs to change*: edit the proto and regenerate the Go code.
3. **gorm-cli-development** — *if there's a database*: build the DAL (`store/models`, `store/dal`).
4. **golang-development** — write the business code on the generated `.go` files; baseline Go style applies throughout.
5. **golang-service-docker** — *when packaging*: confirm the project has the docker config, then build the image / compose stack.

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
