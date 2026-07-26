#!/usr/bin/env bash
set -euo pipefail

test_script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
renderer="${test_script_dir}/render.sh"
test_root="$(mktemp -d /tmp/service-docker-scaffold-test.XXXXXX)"
# Images built by the SERVICE_DOCKER_SCAFFOLD_BUILD_TEST block. The EXIT trap
# removes them so the regression suite leaves no local image artifacts behind.
test_built_images=""
keep_built_images="${SERVICE_DOCKER_SCAFFOLD_KEEP_IMAGE:-0}"

cleanup() {
  rm -rf "$test_root"
  if [[ -n "$test_built_images" && "$keep_built_images" != 1 ]]; then
    for test_image in $test_built_images; do
      if docker rmi "$test_image" >/dev/null 2>&1; then
        printf 'removed test image %s\n' "$test_image"
      fi
    done
  fi
}
trap cleanup EXIT

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

new_fixture() {
  local name="$1"
  local fixture="${test_root}/${name}"
  mkdir -p "${fixture}/cmd/server"
  printf 'module example.com/%s\n\ngo 1.26.1\n' "$name" > "${fixture}/go.mod"
  printf 'package main\n\nfunc main() {}\n' > "${fixture}/cmd/server/main.go"
  printf 'server:\n  grpc_addr: ":9000"\nintegration:\n  token: ${EXTERNAL_API_TOKEN}\n' > "${fixture}/config.example.yaml"
  printf '.PHONY: test\n\ntest:\n\t@true\n' > "${fixture}/Makefile"
  printf '%s\n' "$fixture"
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  grep -Fq -- "$pattern" "$file" || fail "$file does not contain: $pattern"
}

assert_not_contains() {
  local file="$1"
  local pattern="$2"
  if grep -Fq -- "$pattern" "$file"; then
    fail "$file unexpectedly contains: $pattern"
  fi
}

assert_clean_render() {
  local fixture="$1"
  if grep -ER '\{\{[^}]+\}\}|# --- (IF|END) ' \
    "${fixture}/Dockerfile" \
    "${fixture}/Dockerfile.dockerignore" \
    "${fixture}/.dockerignore" \
    "${fixture}/docker-compose.yaml" \
    "${fixture}/.env.example" \
    "${fixture}/Makefile" >/dev/null; then
    fail "unresolved template syntax in $fixture"
  fi
  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    (cd "$fixture" && docker compose config --quiet)
  fi
}

solo_fixture="$(new_fixture solo-service)"
bash "$renderer" \
  --target "$solo_fixture" \
  --service-name solo-service \
  --binary-name server \
  --env-prefix SOLO_SERVICE >/dev/null
assert_clean_render "$solo_fixture"
assert_contains "${solo_fixture}/Dockerfile" 'ARG GO_VERSION=1.26.1'
assert_contains "${solo_fixture}/Dockerfile" 'FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-bookworm AS builder'
assert_contains "${solo_fixture}/Dockerfile" 'FROM alpine:3.24'
assert_contains "${solo_fixture}/Dockerfile" 'FROM ghcr.io/grpc-ecosystem/grpc-health-probe:v0.4.53 AS grpc-health-probe'
assert_not_contains "${solo_fixture}/Dockerfile" 'FROM alpine:3.20'
assert_contains "${solo_fixture}/Dockerfile" 'ARG TARGETPLATFORM'
assert_contains "${solo_fixture}/Dockerfile" 'ARG TARGETVARIANT'
assert_contains "${solo_fixture}/Dockerfile" 'amd64/v1|amd64/v2|amd64/v3|amd64/v4) export GOAMD64="${TARGETVARIANT}"'
assert_contains "${solo_fixture}/Dockerfile" 'arm/|arm/v7) export GOARM=7'
assert_contains "${solo_fixture}/Dockerfile" 'unsupported target platform: ${TARGETPLATFORM}'
assert_contains "${solo_fixture}/Dockerfile" 'mkdir -p /out'
assert_contains "${solo_fixture}/Dockerfile" 'CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}"'
assert_contains "${solo_fixture}/Dockerfile" '-mod=readonly'
assert_contains "${solo_fixture}/Dockerfile" '-buildvcs=false'
assert_contains "${solo_fixture}/Dockerfile" '-ldflags="-s -w -buildid="'
assert_contains "${solo_fixture}/docker-compose.yaml" 'image: ${IMAGE_REPOSITORY:-solo-service}:${IMAGE_TAG:-latest}'
assert_contains "${solo_fixture}/docker-compose.yaml" 'SOLO_SERVICE_SERVER_GRPC_ADDR=${SOLO_SERVICE_SERVER_GRPC_ADDR:-:9000}'
assert_not_contains "${solo_fixture}/docker-compose.yaml" 'postgres:'
assert_not_contains "${solo_fixture}/docker-compose.yaml" 'redis:'
assert_not_contains "${solo_fixture}/docker-compose.yaml" 'HOST_HTTP_PORT'
assert_not_contains "${solo_fixture}/docker-compose.yaml" 'SOLO_SERVICE_SERVER_HTTP_ADDR'
assert_contains "${solo_fixture}/Dockerfile" 'COPY config.example.yaml /app/config.yaml'
assert_contains "${solo_fixture}/docker-compose.yaml" 'EXTERNAL_API_TOKEN=${EXTERNAL_API_TOKEN:-}'
assert_contains "${solo_fixture}/.env.example" 'EXTERNAL_API_TOKEN='
assert_not_contains "${solo_fixture}/docker-compose.yaml" 'command:'
assert_not_contains "${solo_fixture}/docker-compose.yaml" ':/app/config.yaml:ro'
assert_not_contains "${solo_fixture}/Makefile" 'not a Linux ELF executable'
assert_contains "${solo_fixture}/Makefile" '$(DOCKER_COMPOSE) down'
assert_contains "${solo_fixture}/Makefile" '$(DOCKER_COMPOSE) down -v'
assert_contains "${solo_fixture}/Makefile" 'DOCKER_DEFAULT_PLATFORM=$(TARGET_PLATFORM)'
[[ -n "$(find "${solo_fixture}/Dockerfile" -prune -perm 0644 -print)" ]] || fail 'generated files must be mode 0644'
make -n -C "$solo_fixture" docker-build docker-up docker-down docker-reset docker-logs docker-health >/dev/null
make -n -C "$solo_fixture" TARGET_PLATFORM=linux/amd64 docker-build >/dev/null

solo_before="$(cksum "${solo_fixture}/Makefile" "${solo_fixture}/docker-compose.yaml")"
bash "$renderer" \
  --target "$solo_fixture" \
  --service-name solo-service \
  --binary-name server \
  --env-prefix SOLO_SERVICE >/dev/null
solo_after="$(cksum "${solo_fixture}/Makefile" "${solo_fixture}/docker-compose.yaml")"
[[ "$solo_before" == "$solo_after" ]] || fail 'rerender is not idempotent'
[[ "$(grep -c '^# BEGIN service-docker-scaffold$' "${solo_fixture}/Makefile")" == 1 ]] || fail 'Makefile block duplicated'

gateway_fixture="$(new_fixture gateway-service)"
printf 'server:\n  grpc_addr: ":9000"\n  http_addr: ":8080"\nintegration:\n  token: ${GATEWAY_SERVICE_VENDOR_TOKEN}\n' > "${gateway_fixture}/config.example.yaml"
bash "$renderer" \
  --target "$gateway_fixture" \
  --service-name gateway-service \
  --binary-name server \
  --grpc-port 9000 \
  --http-port 8080 \
  --env-prefix GATEWAY_SERVICE >/dev/null
assert_clean_render "$gateway_fixture"
assert_contains "${gateway_fixture}/Dockerfile" 'EXPOSE 9000'
assert_contains "${gateway_fixture}/Dockerfile" 'EXPOSE 8080'
assert_contains "${gateway_fixture}/docker-compose.yaml" 'GATEWAY_SERVICE_SERVER_GRPC_ADDR=${GATEWAY_SERVICE_SERVER_GRPC_ADDR:-:9000}'
assert_contains "${gateway_fixture}/docker-compose.yaml" 'GATEWAY_SERVICE_SERVER_HTTP_ADDR=${GATEWAY_SERVICE_SERVER_HTTP_ADDR:-:8080}'
assert_contains "${gateway_fixture}/docker-compose.yaml" '${HOST_GRPC_PORT:-9000}:9000'
assert_contains "${gateway_fixture}/docker-compose.yaml" '${HOST_HTTP_PORT:-8080}:8080'
assert_contains "${gateway_fixture}/docker-compose.yaml" 'grpc_health_probe", "-addr=127.0.0.1:9000"'
assert_not_contains "${gateway_fixture}/docker-compose.yaml" 'curl'
assert_contains "${gateway_fixture}/.env.example" '# HOST_HTTP_PORT=8080'
assert_contains "${gateway_fixture}/.env.example" '# GATEWAY_SERVICE_SERVER_HTTP_ADDR=:8080'
[[ "$(grep -Fc -- '- GATEWAY_SERVICE_SERVER_HTTP_ADDR=${GATEWAY_SERVICE_SERVER_HTTP_ADDR:-:8080}' "${gateway_fixture}/docker-compose.yaml")" == 1 ]] || fail 'managed HTTP address was duplicated'

if bash "$renderer" \
  --target "$gateway_fixture" \
  --service-name gateway-service \
  --binary-name server \
  --grpc-port 9000 \
  --http-port 9000 \
  --env-prefix GATEWAY_SERVICE >/dev/null 2>&1; then
  fail 'renderer accepted identical gRPC and HTTP ports'
fi

postgres_fixture="$(new_fixture postgres-service)"
printf 'database:\n  password: ${POSTGRES_SERVICE_DATABASE_PASSWORD}\n' >> "${postgres_fixture}/config.example.yaml"
bash "$renderer" \
  --target "$postgres_fixture" \
  --service-name postgres-service \
  --binary-name server \
  --env-prefix POSTGRES_SERVICE \
  --database postgres >/dev/null
assert_clean_render "$postgres_fixture"
assert_contains "${postgres_fixture}/docker-compose.yaml" 'postgres:'
assert_contains "${postgres_fixture}/docker-compose.yaml" 'condition: service_healthy'
assert_contains "${postgres_fixture}/docker-compose.yaml" 'POSTGRES_PASSWORD=${POSTGRES_SERVICE_DATABASE_PASSWORD:-postgres}'
assert_contains "${postgres_fixture}/docker-compose.yaml" 'POSTGRES_DB=${POSTGRES_SERVICE_DATABASE_DBNAME:-postgres_service}'
assert_contains "${postgres_fixture}/docker-compose.yaml" 'POSTGRES_SERVICE_DATABASE_DBNAME=${POSTGRES_SERVICE_DATABASE_DBNAME:-postgres_service}'
assert_contains "${postgres_fixture}/.env.example" 'POSTGRES_SERVICE_DATABASE_DBNAME=postgres_service'
[[ "$(grep -Fc -- '- POSTGRES_SERVICE_DATABASE_PASSWORD=${POSTGRES_SERVICE_DATABASE_PASSWORD:-postgres}' "${postgres_fixture}/docker-compose.yaml")" == 1 ]] || fail 'managed config variable was duplicated'
assert_not_contains "${postgres_fixture}/docker-compose.yaml" 'include:'

mysql_redis_fixture="$(new_fixture mysql-redis-service)"
printf 'server:\n  grpc_addr: ":9000"\nvendor:\n  token: ${MYSQL_REDIS_SERVICE_VENDOR_TOKEN}\n' > "${mysql_redis_fixture}/config.docker.yaml"
bash "$renderer" \
  --target "$mysql_redis_fixture" \
  --service-name mysql-redis-service \
  --binary-name server \
  --env-prefix MYSQL_REDIS_SERVICE \
  --database mysql \
  --database-name application_data \
  --redis \
  --config-mode mount \
  --config-source config.docker.yaml >/dev/null
assert_clean_render "$mysql_redis_fixture"
assert_contains "${mysql_redis_fixture}/docker-compose.yaml" 'mysql:'
assert_contains "${mysql_redis_fixture}/docker-compose.yaml" 'redis:'
assert_contains "${mysql_redis_fixture}/docker-compose.yaml" 'MYSQL_DATABASE=${MYSQL_REDIS_SERVICE_DATABASE_DBNAME:-application_data}'
assert_contains "${mysql_redis_fixture}/.env.example" 'MYSQL_REDIS_SERVICE_DATABASE_DBNAME=application_data'
assert_not_contains "${mysql_redis_fixture}/docker-compose.yaml" 'profiles:'
assert_contains "${mysql_redis_fixture}/docker-compose.yaml" '${CONFIG_FILE:-./config.docker.yaml}:/app/config.yaml:ro'
assert_contains "${mysql_redis_fixture}/docker-compose.yaml" 'MYSQL_REDIS_SERVICE_VENDOR_TOKEN=${MYSQL_REDIS_SERVICE_VENDOR_TOKEN:-}'
assert_contains "${mysql_redis_fixture}/.env.example" 'MYSQL_REDIS_SERVICE_VENDOR_TOKEN='
assert_not_contains "${mysql_redis_fixture}/docker-compose.yaml" 'command:'
assert_not_contains "${mysql_redis_fixture}/Dockerfile" 'COPY config.docker.yaml /app/config.yaml'

prebuilt_fixture="$(new_fixture prebuilt-service)"
bash "$renderer" \
  --target "$prebuilt_fixture" \
  --service-name prebuilt-service \
  --binary-name worker \
  --env-prefix PREBUILT_SERVICE \
  --build-mode prebuilt >/dev/null 2>&1
assert_clean_render "$prebuilt_fixture"
assert_contains "${prebuilt_fixture}/Dockerfile" 'COPY bin/worker /app/worker'
assert_contains "${prebuilt_fixture}/Dockerfile" 'FROM alpine:3.24'
assert_contains "${prebuilt_fixture}/Dockerfile" 'FROM ghcr.io/grpc-ecosystem/grpc-health-probe:v0.4.53 AS grpc-health-probe'
assert_contains "${prebuilt_fixture}/Dockerfile" 'COPY config.example.yaml /app/config.yaml'
assert_contains "${prebuilt_fixture}/Dockerfile.dockerignore" '!bin/worker'
assert_contains "${prebuilt_fixture}/Makefile" 'not a Linux ELF executable'
assert_contains "${prebuilt_fixture}/Makefile" 'dynamically linked; rebuild with CGO_ENABLED=0'

none_fixture="$(new_fixture no-config-service)"
bash "$renderer" \
  --target "$none_fixture" \
  --service-name no-config-service \
  --binary-name server \
  --env-prefix NO_CONFIG_SERVICE \
  --config-mode none >/dev/null
assert_clean_render "$none_fixture"
assert_not_contains "${none_fixture}/Dockerfile" 'COPY config.example.yaml /app/config.yaml'
assert_not_contains "${none_fixture}/docker-compose.yaml" 'EXTERNAL_API_TOKEN'

custom_image_fixture="$(new_fixture custom-image-service)"
bash "$renderer" \
  --target "$custom_image_fixture" \
  --service-name custom-image-service \
  --binary-name server \
  --env-prefix CUSTOM_IMAGE_SERVICE \
  --runtime-image registry.example.com/base/alpine:3.24 \
  --health-probe-image registry.example.com/probes/grpc-health-probe:v0.4.53 >/dev/null
assert_clean_render "$custom_image_fixture"
assert_contains "${custom_image_fixture}/Dockerfile" 'FROM registry.example.com/probes/grpc-health-probe:v0.4.53 AS grpc-health-probe'
assert_contains "${custom_image_fixture}/Dockerfile" 'FROM registry.example.com/base/alpine:3.24'
assert_not_contains "${custom_image_fixture}/Dockerfile" 'FROM alpine:3.24'
assert_not_contains "${custom_image_fixture}/Dockerfile" 'ghcr.io/grpc-ecosystem'

monorepo_root="${test_root}/monorepo"
mkdir -p "${monorepo_root}/common" "${monorepo_root}/service/cmd/server"
printf 'module example.com/common\n\ngo 1.26.1\n' > "${monorepo_root}/common/go.mod"
printf 'module example.com/service\n\ngo 1.26.1\n\nreplace example.com/common => ../common\n' > "${monorepo_root}/service/go.mod"
printf 'package main\n\nfunc main() {}\n' > "${monorepo_root}/service/cmd/server/main.go"
printf 'server:\n  grpc_addr: ":9000"\n' > "${monorepo_root}/service/config.example.yaml"
bash "$renderer" \
  --target "${monorepo_root}/service" \
  --service-name monorepo-service \
  --binary-name server \
  --env-prefix MONOREPO_SERVICE \
  --build-context .. >/dev/null
assert_clean_render "${monorepo_root}/service"
assert_contains "${monorepo_root}/service/Dockerfile" 'cd /workspace/service'
assert_contains "${monorepo_root}/service/docker-compose.yaml" 'context: ".."'
assert_contains "${monorepo_root}/service/docker-compose.yaml" 'dockerfile: "service/Dockerfile"'

toolchain_fixture="$(new_fixture toolchain-service)"
printf 'module example.com/toolchain-service\n\ngo 1.25.0\n\ntoolchain go1.26.1\n' > "${toolchain_fixture}/go.mod"
bash "$renderer" \
  --target "$toolchain_fixture" \
  --service-name toolchain-service \
  --binary-name server \
  --env-prefix TOOLCHAIN_SERVICE >/dev/null
assert_clean_render "$toolchain_fixture"
assert_contains "${toolchain_fixture}/Dockerfile" 'ARG GO_VERSION=1.26.1'

unmanaged_fixture="$(new_fixture unmanaged-service)"
printf 'FROM scratch\n' > "${unmanaged_fixture}/Dockerfile"
if bash "$renderer" \
  --target "$unmanaged_fixture" \
  --service-name unmanaged-service \
  --binary-name server \
  --env-prefix UNMANAGED_SERVICE >/dev/null 2>&1; then
  fail 'renderer overwrote an unmanaged Dockerfile without --force'
fi
assert_contains "${unmanaged_fixture}/Dockerfile" 'FROM scratch'

# A pre-existing, service-owned .env.example (as the golang-service-development
# scaffold emits) must be read for compose inline defaults and left intact —
# never regenerated or clobbered.
owned_env_fixture="$(new_fixture owned-env-service)"
printf '# service-owned env template; do not clobber\nEXTERNAL_API_TOKEN=scaffold-default-token\nPAYMENTSVC_ONLY_VAR=kept\n' > "${owned_env_fixture}/.env.example"
owned_env_before="$(cksum "${owned_env_fixture}/.env.example")"
bash "$renderer" \
  --target "$owned_env_fixture" \
  --service-name owned-env-service \
  --binary-name server \
  --env-prefix OWNED_ENV_SERVICE >/dev/null
owned_env_after="$(cksum "${owned_env_fixture}/.env.example")"
[[ "$owned_env_before" == "$owned_env_after" ]] || fail 'renderer clobbered a service-owned .env.example'
assert_contains "${owned_env_fixture}/.env.example" 'service-owned env template; do not clobber'
assert_contains "${owned_env_fixture}/.env.example" 'PAYMENTSVC_ONLY_VAR=kept'
assert_contains "${owned_env_fixture}/docker-compose.yaml" 'EXTERNAL_API_TOKEN=${EXTERNAL_API_TOKEN:-scaffold-default-token}'

if [[ "${SERVICE_DOCKER_SCAFFOLD_BUILD_TEST:-0}" == 1 ]]; then
  if [[ -n "${SERVICE_DOCKER_SCAFFOLD_TARGET_PLATFORM:-}" ]]; then
    make -C "$solo_fixture" \
      TARGET_PLATFORM="$SERVICE_DOCKER_SCAFFOLD_TARGET_PLATFORM" \
      docker-build
  else
    make -C "$solo_fixture" docker-build
  fi
  # Record the built image so the EXIT trap can remove it. Derive the ref from
  # the rendered Makefile so this stays correct if the fixture changes.
  built_repo="$(awk '/^IMAGE_REPOSITORY / {print $3}' "$solo_fixture/Makefile")"
  built_tag="$(awk '/^IMAGE_TAG / {print $3}' "$solo_fixture/Makefile")"
  test_built_images="${built_repo}:${built_tag}"
fi

printf 'PASS: service-docker-scaffold renderer regression suite\n'
