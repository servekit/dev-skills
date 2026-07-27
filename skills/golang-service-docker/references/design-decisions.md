# Design Decisions

This document explains the operational choices encoded by `golang-service-docker`. Read the relevant section before changing a template or bypassing the renderer.

## Table of contents

1. [Source builds are the default](#source-builds-are-the-default)
2. [Prebuilt mode is guarded](#prebuilt-mode-is-guarded)
3. [Build context and local replacements](#build-context-and-local-replacements)
4. [Alpine runtime and grpc health probe](#alpine-runtime-and-grpc-health-probe)
5. [gRPC and grpc-gateway ports](#grpc-and-grpc-gateway-ports)
6. [Stable image identity](#stable-image-identity)
7. [One rendered Compose file](#one-rendered-compose-file)
8. [Dependency lifecycle](#dependency-lifecycle)
9. [Configuration modes](#configuration-modes)
10. [Environment defaults](#environment-defaults)
11. [Volume lifecycle](#volume-lifecycle)
12. [Renderer safety and idempotency](#renderer-safety-and-idempotency)

## Source builds are the default

**Decision:** compile the Go service inside a BuildKit multi-stage Docker build with a `golang:<version>-bookworm` builder pinned to `BUILDPLATFORM` and:

```text
CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH}
```

Copying the result of a plain host-side `go build` is unreliable. A binary built on macOS is Mach-O rather than Linux ELF, and a binary built for the wrong CPU architecture fails with an exec-format error. Linux binaries using CGO may also expect glibc and fail in an Alpine/musl runtime.

The builder image contains the Go toolchain; the final Alpine stage does not. Using the Debian Bookworm variant for compilation avoids treating the smaller musl-based `golang:alpine` variant as if it were the plain Alpine runtime. Because CGO is disabled, the resulting static Go executable remains suitable for the Alpine runtime.

The builder redeclares Docker's automatic `TARGETPLATFORM`, `TARGETOS`, `TARGETARCH`, and `TARGETVARIANT` arguments inside the stage. It maps amd64 variants to `GOAMD64` and ARM v7 to `GOARM=7`, and rejects non-Linux or unsupported platforms. The accepted set matches the bundled probe manifest: amd64, arm64/v8, arm/v7, ppc64le, and s390x.

The renderer prefers an exact `toolchain goX.Y.Z` directive and otherwise retains the complete `go` directive. It does not deliberately turn `1.26.1` into the floating `1.26` tag. `GOTOOLCHAIN=local` also prevents an undeclared toolchain download inside the build.

The source tree is mounted read-only during the build. Go's module and compiler caches use BuildKit cache mounts, while the output directory is created explicitly outside the bind mount. `-mod=readonly` rejects missing module metadata instead of editing `go.mod` or `go.sum`; `-trimpath`, `-buildvcs=false`, and `-buildid=` remove incidental host, VCS, and linker identity inputs. This avoids copying host build artifacts and supports both standalone modules and monorepos.

For a single non-native target, run `make docker-build TARGET_PLATFORM=linux/amd64`; the Makefile passes that selection through Docker's `DOCKER_DEFAULT_PLATFORM`. For a multi-platform manifest, invoke `docker buildx build --platform ... --push` with the generated context and Dockerfile. A multi-platform result cannot be loaded into the classic local image store as one image.

**When to deviate:** a service that genuinely requires CGO should use a compatible builder/runtime pair, normally Debian-to-Debian, and install the required shared libraries explicitly. Do not simply remove `CGO_ENABLED=0` while keeping Alpine.

## Prebuilt mode is guarded

**Decision:** retain prebuilt packaging as an explicit mode for CI systems that already produce release binaries.

The selected file is the only `bin/` entry re-included by the generated ignore files. Before building, the Makefile checks that the file exists, that `file(1)` identifies it as ELF, and that it is not dynamically linked.

These checks prevent the common failure where a macOS or Windows development binary is copied successfully but cannot start in Linux. They also reject typical glibc-linked binaries that Alpine cannot load.

The check does not prove the CPU architecture matches the requested Docker platform. CI using prebuilt mode should produce and label one artifact per architecture; source mode is safer when that pipeline does not already exist.

## Build context and local replacements

Go services can contain directives such as:

```go
replace example.com/platform/go-common => ../go-common
```

Docker cannot access files outside its build context. The renderer therefore resolves every local `replace` path and refuses a context that does not contain it. Select the nearest common ancestor with `--build-context`.

The renderer calculates both the service source directory and Dockerfile path relative to that context. A service in `repo/services/message-service` can therefore use `repo` as context without hard-coding repository names in templates.

`Dockerfile.dockerignore` is generated next to the Dockerfile because Docker gives a Dockerfile-specific ignore file precedence. Its patterns use the service's path relative to the context. A conventional `.dockerignore` is also generated for standalone builds.

**Tradeoff:** a broad monorepo context can transfer more data. The ignore rules exclude repository metadata and common scratch output, while BuildKit's bind mount and cache keep layer churn low.

## Alpine runtime and grpc health probe

**Decision:** use a supported Alpine 3.24 runtime with `ca-certificates`, `tzdata`, and a non-root user. Do not scaffold new images from Alpine 3.20, whose normal support window ended in April 2026. Copy `grpc_health_probe` from its versioned official multi-platform image rather than downloading a host-architecture binary in a `RUN` step.

This removes `uname -m` target-detection mistakes, avoids an unverified `wget`, and lets the container registry resolve the probe image for the build target platform.

Both the runtime and probe image references are render-time parameters (`--runtime-image`, `--health-probe-image`) defaulting to Alpine 3.24 and `grpc-health-probe:v0.4.53` (centralized in `scripts/defaults.sh`), so a version bump or a private registry mirror is a single flag or a one-line edit rather than a template change.

The application healthcheck calls:

```text
grpc_health_probe -addr=127.0.0.1:<grpc-port> -connect-timeout=2s
```

`go-common/grpcx` registers `grpc.health.v1.Health`, reports `SERVING` after startup, and reports `NOT_SERVING` during shutdown. The probe therefore checks service readiness rather than merely checking whether a TCP socket accepts connections.

**When to deviate:** if the service does not expose the standard gRPC Health Protocol, use a protocol-appropriate healthcheck or add the protocol to the service. This skill should not silently replace the semantic probe with a weak port check.

## gRPC and grpc-gateway ports

**Decision:** model grpc-gateway as an optional HTTP listener alongside the required gRPC listener. Passing `--http-port` makes the renderer:

- expose both container ports in the image;
- publish both ports in Compose, with independent `HOST_GRPC_PORT` and `HOST_HTTP_PORT` overrides;
- set the grpcx/configx server address environment variables to the selected container ports.

Omitting `--http-port` produces a gRPC-only image and Compose service without an unused HTTP mapping.

The healthcheck does not switch to HTTP for gateway services. grpc-gateway still depends on the underlying gRPC service, while grpcx's standard health service carries readiness and graceful-shutdown state. Keeping `grpc_health_probe` therefore provides a stronger, consistent signal than probing an arbitrary gateway route. HTTP route checks belong in an external smoke test when an application needs end-to-end gateway verification.

## Stable image identity

**Decision:** Compose always declares:

```yaml
image: ${IMAGE_REPOSITORY:-service-name}:${IMAGE_TAG:-latest}
```

Relying on Compose's generated project/service image name makes CI, tagging, and registry pushes unpredictable. `.env.example` and the managed Makefile block expose the same repository and tag variables, so `docker-build`, `docker-up`, and `docker-push` operate on one identity.

Production automation should set immutable version tags and may additionally pin base images by digest.

## One rendered Compose file

**Decision:** render one final `docker-compose.yaml` instead of using `include` files to inject partial definitions for an existing app service.

Compose `include` is designed to import independent application models, and duplicate resource merging has varied across versions. The deterministic renderer already knows which dependencies were selected, so resolving conditional blocks before writing YAML is simpler and version-stable.

Postgres and MySQL remain mutually exclusive through the renderer's `--database` enum. Redis can be combined with either database. The regression suite runs `docker compose config --quiet` for supported combinations whenever Compose is installed.

## Dependency lifecycle

**Decision:** when a dependency is selected, include it as a normal service and gate application startup on `condition: service_healthy`.

Redis is not hidden behind a profile. A profile-gated Redis service combined with unconditional `REDIS_ADDR=redis:6379` lets the app start while its configured dependency is absent. If Redis is optional, omit it during rendering; if selected, start it consistently.

Database and Redis host ports remain configurable for local development. Internal app connections always use Compose DNS names and container ports.

## Configuration modes

**Decision:** treat the configuration file as a required runtime contract while keeping environment-specific values outside the image.

- Copy mode, the default, copies a non-secret template such as `config.example.yaml` into the image as `/app/config.yaml`.
- Mount mode replaces that path with a host file through a read-only Compose volume.
- None mode is an explicit exception for binaries proven to start without a configuration file.

No mode emits `--config`. grpcx/configx services normally discover `./config.yaml`; with `WORKDIR /app`, `/app/config.yaml` is already at the expected location.

The template is structure-only: every value is a `${VAR}` placeholder, never a literal default. This keeps a single source of truth for values (`.env.example` / the runtime env) and makes every field env-overridable. configx expands those placeholders from the process environment at startup **only when the service's `config.Load` passes `configx.WithExpandEnv()`** — the renderer assumes this is enabled (the scaffold turns it on). Without it, `${VAR}` stays literal and the injected env never reaches the config struct. Secrets remain in `.env`, CI, or the deployment secret manager and never enter image layers.

Compose's project `.env` file supplies interpolation values but does not automatically pass every value into the container. The renderer therefore scans the config template for `${VAR}` tokens and emits corresponding app `environment` entries. When the service ships a `.env.example` (the scaffold always does), the renderer reads each variable's default from it and emits `KEY=${KEY:-default}` so `docker compose up` works even before an operator copies `.env.example` to `.env`. Known log/database/Redis variables remain in their explicit sections with docker-topology defaults (e.g. `DATABASE_HOST=postgres`, the compose service name) so local dependency defaults continue to work.

## Environment defaults

Compose uses `${VAR:-default}` for local-development defaults. For variables discovered in the config template, the renderer reads each default from the service's `.env.example` and inlines it, so `docker compose up` works even before an operator copies `.env.example` to `.env`. Compose also reads a local `.env` file when present, which overrides the inline defaults.

`.env.example` is service-owned: the `golang-service-development` scaffold generates it with docker-compose-oriented app-env defaults (hostnames are compose service names). The renderer reads it and leaves it intact — it never regenerates or clobbers a service-owned `.env.example`. It generates one from the template only when the file is absent (standalone use on a non-scaffolded service). Real secrets must come from an ignored `.env`, CI secret store, or deployment secret manager.

Application environment names use the actual configured prefix gathered from the source repository. Deriving a prefix from the service name is only a fallback; it must not override a different config contract.

The logical database name is independent from the deployment service name. By default, the renderer converts `message-service` to `message_service`, avoiding hyphenated SQL identifiers that often require quoting. `--database-name` supports services that use a shared or historical database name.

## Volume lifecycle

**Decision:** normal shutdown preserves data.

```text
make docker-down   -> docker compose down
make docker-reset  -> docker compose down -v
```

Deleting named volumes from a routine `docker-down` target is surprising and can destroy a developer's database. The destructive operation has a separate, explicit name and prints a notice before running.

## Renderer safety and idempotency

The renderer owns files containing the `Generated by golang-service-docker` signature. It can update those files on subsequent runs but refuses to overwrite unmanaged files unless the caller supplies `--force` after reviewing the conflict.

Makefile content is enclosed by:

```make
# BEGIN golang-service-docker
...
# END golang-service-docker
```

Rerendering replaces that block and preserves unrelated targets. Incomplete or duplicate marker blocks cause an error rather than a destructive guess.

All placeholders and conditional markers are checked after rendering. The bundled regression suite covers dependency combinations, config modes, source/prebuilt builds, monorepo replacements, overwrite protection, and repeat rendering.
