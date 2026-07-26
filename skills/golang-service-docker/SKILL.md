---
name: golang-service-docker
description: Build and package an existing grpcx-based Go service, including services that expose an optional grpc-gateway HTTP port, as a reproducible Linux Docker image. Generate a validated docker-compose development stack, environment example, and idempotent Makefile targets. Use this skill when the user wants to dockerize, containerize, build an image for, or add Docker/Compose packaging to a Go gRPC or grpc-gateway service that uses go-common/grpcx. Prefer the source-based multi-stage build; use prebuilt mode only when the user already has a statically linked Linux ELF binary.
---

# golang-service-docker

Package a grpcx-based Go service into a reproducible Linux image and generate a local Compose stack around it.

## Scope

This skill generates:

- a multi-stage `Dockerfile` that cross-compiles the service for the target container platform;
- `Dockerfile.dockerignore` and `.dockerignore` files aligned with the selected build mode;
- one final `docker-compose.yaml`, with optional postgres, mysql, and redis services rendered inline;
- a required non-secret config template at `/app/config.yaml` by default, plus runtime environment injection for every `${VAR}` placeholder it references;
- `.env.example` — read from the service when present (it owns app-env defaults and the renderer only adds compose inline defaults from it); generated with image, port, healthcheck, discovered config variables, and dependency overrides only when absent;
- idempotent Makefile targets for build, push, up, down, reset, logs, and health checks.

The generated image contains `grpc_health_probe` and checks the standard `grpc.health.v1.Health` endpoint registered by `go-common/grpcx`. A grpc-gateway service keeps this gRPC healthcheck and additionally exposes its HTTP listener.

This skill does not generate application code, migrations, Kubernetes manifests, Helm charts, or CI pipelines.

## Build modes

### Source mode — default

Use a BuildKit multi-stage build. The `golang:<version>-bookworm` builder contains the Go toolchain and runs on `BUILDPLATFORM`; the final `alpine` runtime contains neither the compiler nor the source. The builder receives `TARGETPLATFORM`, `TARGETOS`, `TARGETARCH`, and `TARGETVARIANT`, sets `CGO_ENABLED=0`, and produces a Linux-compatible static binary inside Docker. This avoids host OS, CPU architecture, and glibc/musl mismatches.

Map Docker architecture variants to Go tuning variables where they differ: `linux/amd64/v1..v4` maps to `GOAMD64`, and `linux/arm/v7` maps to `GOARM=7`. Accept only platforms available in both the runtime and bundled probe images: amd64, arm64/v8, arm/v7, ppc64le, and s390x. Fail explicitly for non-Linux or unsupported targets instead of silently building a mislabeled image.

Prefer the exact `toolchain` directive from `go.mod`; otherwise preserve the complete `go` directive rather than dropping its patch component. `GOTOOLCHAIN=local`, `-mod=readonly`, `-trimpath`, `-buildvcs=false`, and an empty linker build ID prevent undeclared toolchain downloads, module-file mutation, host-path leakage, and incidental VCS/build-ID variation.

Source mode works with standalone modules and monorepos. For a module with local `replace ../...` directives, choose a build context that contains both the service and every replaced module.

### Prebuilt mode — explicit opt-in

Use prebuilt mode only when the user intentionally supplies `bin/<binary>` and accepts responsibility for building it. Before running Docker, verify that it is:

- an ELF executable for Linux;
- built for the target architecture;
- statically linked, normally with `CGO_ENABLED=0`.

The generated Makefile performs the ELF and dynamic-link checks. Never copy an unchecked macOS or Windows executable into the Linux image.

## Gather inputs

Infer values from the target repository where possible. Ask only for values that cannot be determined safely.

| Input | How to determine | Example |
|---|---|---|
| service name | repository name or deployment name, kebab-case | `message-service` |
| binary name | main package directory or existing Makefile | `server` |
| build package | Go main package passed to `go build` | `./cmd/server` |
| Go image version | exact `toolchain`, then complete `go` directive | `1.26.1` |
| gRPC port | grpcx server config; default 9000 | `9000` |
| HTTP port | grpcx `GatewayAddr`/HTTP server config; omit when gateway is disabled | `8080` |
| env prefix | actual config loader prefix, not merely an assumed derivation | `MESSAGE_SERVICE` |
| build context | `.` unless local replace directives require a common ancestor | `.` or `../..` |
| database | `none`, `postgres`, or `mysql` | `postgres` |
| database name | actual logical database name; default service name normalized to snake_case | `message_service` |
| redis | enabled only when the application really uses it | enabled |
| config mode | `copy`, `mount`, or `none`; default `copy` | `copy` |
| config source | infer `config.example.yaml`, then `config.yaml` | `config.example.yaml` |

Inspect `go.mod`, the main package, config package, existing Makefile, and README. Business-code-wide env discovery is unnecessary, but the generated environment variable names must match the real config contract.

## gRPC and grpc-gateway ports

Treat grpc-gateway as an optional second listener, not as a replacement for gRPC:

- always configure and expose the gRPC container port; it remains the backend used by grpc-gateway and the target of `grpc_health_probe`;
- when the service config enables `GatewayAddr`/`HTTPAddr`, pass `--http-port` so the Dockerfile and Compose file also expose and publish that container port;
- omit `--http-port` for gRPC-only services, so no unused HTTP mapping is generated;
- use `HOST_GRPC_PORT` and `HOST_HTTP_PORT` only to resolve host-side conflicts. Do not change the container listen ports through those variables.

The renderer sets `<ENV_PREFIX>_SERVER_GRPC_ADDR=:<grpc-port>` and, when enabled, `<ENV_PREFIX>_SERVER_HTTP_ADDR=:<http-port>`. These values keep the process listeners aligned with the generated Compose mappings even if the copied config contains different local defaults.

## Config strategy

The grpcx/configx service shape requires a config file. Choose one strategy explicitly:

- `copy` — default: copy the non-secret config template into the image as `/app/config.yaml`. The program finds it as `./config.yaml` because the runtime `WORKDIR` is `/app`.
- `mount`: do not bake the template into the image; mount `${CONFIG_FILE:-./<source>}` at `/app/config.yaml:ro` through Compose.
- `none`: emit no config file. Use only after confirming the binary genuinely supports starting without one.

The renderer scans the selected YAML file for `${VAR}` references, adds non-dependency variables to the app's Compose `environment`, and lists them in `.env.example`. Database, Redis, and log variables keep their explicit safe defaults and are not duplicated. Compose `.env` is only an interpolation source, so these generated `environment` entries are what actually pass values into the container.

**`WithExpandEnv` contract.** configx only expands `${VAR}` placeholders when the service's `config.Load` passes `configx.WithExpandEnv()`. The renderer assumes this is set (the `golang-service-development` scaffold enables it); without it, `${VAR}` stays literal and the injected env never reaches the config struct.

**`.env.example` ownership.** When the target already has a `.env.example` (the normal case for scaffolded services), the renderer **reads** it for the compose `${VAR:-default}` inline defaults and leaves the file intact — it never clobbers the service's env contract. It generates a `.env.example` from the template only when the file is absent (standalone use on a non-scaffolded service).

Do not add a `--config` argument. `configx` discovers `/app/config.yaml` through its normal `./config.yaml` lookup and expands the placeholders from the container environment at startup.

## Render workflow

Read the template headers, then use the bundled renderer instead of performing substitutions manually:

```bash
bash <skill-dir>/scripts/render.sh \
  --target /path/to/service \
  --service-name message-service \
  --binary-name server \
  --build-path ./cmd/server \
  --grpc-port 9000 \
  --http-port 8080 \
  --env-prefix MESSAGE_SERVICE \
  --database postgres \
  --redis \
  --config-mode copy \
  --config-source config.example.yaml
```

The renderer:

1. validates all inputs and reads the Go version from `go.mod` when omitted;
2. calculates the service path relative to the selected build context;
3. resolves conditional blocks for build mode, dependencies, and config mode;
4. refuses to overwrite unmanaged Docker files unless `--force` is supplied;
5. replaces its marked Makefile block rather than appending duplicate targets;
6. rejects unresolved placeholders or conditional markers;
7. runs `docker compose config --quiet` when Docker Compose is available.

Useful options:

```text
--build-mode source|prebuilt       default: source
--build-context PATH               default: . (relative to target)
--go-version VERSION               default: exact toolchain/go.mod directive
--http-port PORT                   optional grpc-gateway container port
--database none|postgres|mysql     default: none
--database-name NAME               default: service-name converted to snake_case
--redis                            include and start Redis normally
--config-mode copy|mount|none      default: copy
--config-source PATH               default: config.example.yaml or config.yaml
--image-repository NAME            default: service name
--runtime-image REF                default: alpine:3.24
--health-probe-image REF           default: ghcr.io/grpc-ecosystem/grpc-health-probe:v0.4.53
--force                            overwrite unmanaged generated-file paths
```

If `go.mod` contains a local replacement outside the selected build context, choose a broader context and rerun. Do not remove the replace directive merely to make Docker build.

## Generated commands

The managed Makefile block provides:

- `make docker-build` — build the stable `${IMAGE_REPOSITORY}:${IMAGE_TAG}` image for the host platform, or pass `TARGET_PLATFORM=linux/amd64`;
- `make docker-push` — push that tag;
- `make docker-up` — build, start, and wait for healthy services;
- `make docker-migrate` — one-shot migration: builds, starts dependencies, runs `migrate` in a throwaway container, then exits (production-style run-to-completion; does not touch the serving stack);
- `make docker-down` — stop containers while preserving volumes;
- `make docker-reset` — stop containers and deliberately delete volumes;
- `make docker-logs [svc=name]` — follow logs;
- `make docker-health` — run the bundled probe inside the app container.

**Go module proxy.** The builder resolves modules through `GOPROXY` (default `https://proxy.golang.org,direct`, wired via a Dockerfile `ARG` and compose `build.args`). On networks where `proxy.golang.org` is blocked, set `GOPROXY=https://goproxy.cn,direct` in `.env` (or export it) before `make docker-build`.

Redis is not profile-gated. If the user selects Redis, `make docker-up` starts it and waits for its healthcheck. If Redis is optional, omit it during rendering and enable it in a later rerender when needed.

## Verification

After rendering, run:

```bash
docker compose config --quiet
make docker-build
make docker-build TARGET_PLATFORM=linux/amd64
make docker-up
make docker-health
docker compose ps
```

Verify:

- the image has the requested stable repository and tag;
- the builder uses the expected exact Go toolchain and the requested Linux target platform;
- copy mode contains `/app/config.yaml`, while mount mode has a read-only Compose mount;
- every `${VAR}` referenced by the config is present in the rendered container environment or an explicit dependency block;
- gRPC-only services expose only the gRPC mapping, while gateway services expose both gRPC and HTTP mappings;
- gateway services still use `grpc_health_probe` against the gRPC port rather than an HTTP or TCP-only check;
- no synthetic `--config` argument is emitted;
- the app is healthy and `grpc_health_probe` reports `SERVING`;
- each selected dependency is healthy;
- `make docker-down` preserves named volumes;
- `make docker-reset` is the only generated target that deletes volumes;
- no `{{PLACEHOLDER}}` or `--- IF/END` marker remains.

If Docker cannot run in the current environment, still run the bundled render tests and `docker compose config`; report that the actual image build was not executed.

## Templates and implementation

- `templates/Dockerfile.tmpl` — default multi-stage source build.
- `templates/Dockerfile.prebuilt.tmpl` — guarded prebuilt-binary mode.
- `templates/docker-compose.yaml.tmpl` — single final Compose file with renderer conditionals.
- `templates/.dockerignore.tmpl` — source/prebuilt-aware context rules.
- `templates/.env.example.tmpl` — override contract without stale healthcheck branches.
- `templates/Makefile.targets.tmpl` — managed, idempotent target block.
- `scripts/render.sh` — deterministic renderer and validator.
- `scripts/defaults.sh` — sourced image/version defaults (bump versions here).
- `scripts/test-render.sh` — supported-combination regression tests.
- `references/design-decisions.md` — rationale and deviation guidance.

Read `references/design-decisions.md` before changing build mode, runtime base, healthcheck, config handling, or dependency topology.

## Maintenance contract

When changing a template or renderer behavior, run:

```bash
bash skills/golang-service-docker/scripts/test-render.sh
```

For a real image build in CI, set `SERVICE_DOCKER_SCAFFOLD_BUILD_TEST=1`; the suite removes the built image afterward (`SERVICE_DOCKER_SCAFFOLD_KEEP_IMAGE=1` keeps it for inspection).

Keep the renderer idempotent and preserve the `service-docker-scaffold` Makefile markers. Add a regression case for every new dependency or conditional branch.
