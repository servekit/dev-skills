#!/usr/bin/env bash
set -euo pipefail

scaffold_script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
scaffold_root_dir="$(cd "${scaffold_script_dir}/.." && pwd -P)"
scaffold_template_dir="${scaffold_root_dir}/templates"

die() {
  printf 'service-docker-scaffold: %s\n' "$*" >&2
  exit 1
}

usage() {
  sed -n '/^# Usage:/,/^$/p' "$0" | sed 's/^# \{0,1\}//'
}

# Accept an optional registry[:port]/, lowercase name components, an optional
# :tag, and an optional @sha256 digest. The grammar itself rejects whitespace
# and shell metacharacters, so a value that would break the rendered Dockerfile
# fails validation rather than being written verbatim.
valid_image_ref() {
  local re='^([a-z0-9]+([._-][a-z0-9]+)*(:[0-9]+)/)?[a-z0-9]+([._-][a-z0-9]+)*(/[a-z0-9]+([._-][a-z0-9]+)*)*(:[a-zA-Z0-9_][a-zA-Z0-9._-]{0,127})?(@sha256:[0-9a-f]{64})?$'
  [[ -n "$1" ]] || return 1
  [[ "$1" =~ $re ]]
}

# Usage:
#   render.sh [options]
#
# Required or inferred:
#   --target DIR                    service repository (default: current dir)
#   --service-name NAME             kebab-case deployment name
#   --binary-name NAME              executable filename
#   --build-path PACKAGE            Go main package (default: ./cmd/<binary>)
#   --grpc-port PORT                grpcx port (default: 9000)
#   --http-port PORT                optional grpc-gateway port
#   --env-prefix PREFIX             actual application env prefix
#
# Build and runtime:
#   --build-mode source|prebuilt    default: source
#   --build-context PATH            relative to target; default: .
#   --go-version VERSION            default: go directive in go.mod
#   --image-repository NAME         default: service name
#   --health-probe-image REF        default: ghcr.io/grpc-ecosystem/grpc-health-probe:v0.4.53
#   --runtime-image REF             default: alpine:3.24
#   --database none|postgres|mysql  default: none
#   --database-name NAME            default: snake_case service name
#   --redis                         include Redis without a dormant profile
#   --config-mode copy|mount|none   default: copy
#   --config-source PATH            default: config.example.yaml or config.yaml
#   --force                         overwrite unmanaged generated-file paths
#   --help

# Static defaults (ports, modes, images, switches) live in scripts/defaults.sh
# so they are easy to find and bump without editing renderer logic. render.sh
# sources that file, then applies --flag overrides on top below.
source "${scaffold_script_dir}/defaults.sh"

# Per-service values with no fixed default: initialized empty, then filled from
# the target repository or the matching flag in the code below.
scaffold_service_name=""
scaffold_binary_name=""
scaffold_build_path=""
scaffold_http_port=""
scaffold_env_prefix=""
scaffold_go_version=""
scaffold_image_repository=""
scaffold_database_name=""
scaffold_config_source=""

while (($#)); do
  case "$1" in
    --target) scaffold_target="${2:?missing value for --target}"; shift 2 ;;
    --service-name) scaffold_service_name="${2:?missing value for --service-name}"; shift 2 ;;
    --binary-name) scaffold_binary_name="${2:?missing value for --binary-name}"; shift 2 ;;
    --build-path) scaffold_build_path="${2:?missing value for --build-path}"; shift 2 ;;
    --grpc-port) scaffold_grpc_port="${2:?missing value for --grpc-port}"; shift 2 ;;
    --http-port) scaffold_http_port="${2:?missing value for --http-port}"; shift 2 ;;
    --env-prefix) scaffold_env_prefix="${2:?missing value for --env-prefix}"; shift 2 ;;
    --build-mode) scaffold_build_mode="${2:?missing value for --build-mode}"; shift 2 ;;
    --build-context) scaffold_build_context="${2:?missing value for --build-context}"; shift 2 ;;
    --go-version) scaffold_go_version="${2:?missing value for --go-version}"; shift 2 ;;
    --image-repository) scaffold_image_repository="${2:?missing value for --image-repository}"; shift 2 ;;
    --health-probe-image) scaffold_health_probe_image="${2:?missing value for --health-probe-image}"; shift 2 ;;
    --runtime-image) scaffold_runtime_image="${2:?missing value for --runtime-image}"; shift 2 ;;
    --database) scaffold_database="${2:?missing value for --database}"; shift 2 ;;
    --database-name) scaffold_database_name="${2:?missing value for --database-name}"; shift 2 ;;
    --redis) scaffold_redis=1; shift ;;
    --config-mode) scaffold_config_mode="${2:?missing value for --config-mode}"; shift 2 ;;
    --config-source) scaffold_config_source="${2:?missing value for --config-source}"; shift 2 ;;
    --force) scaffold_force=1; shift ;;
    --help|-h) usage; exit 0 ;;
    *) die "unknown option: $1" ;;
  esac
done

[[ -d "$scaffold_target" ]] || die "target directory does not exist: $scaffold_target"
scaffold_target_abs="$(cd "$scaffold_target" && pwd -P)"

if [[ -z "$scaffold_service_name" ]]; then
  scaffold_service_name="$(basename "$scaffold_target_abs")"
fi
if [[ -z "$scaffold_binary_name" ]]; then
  scaffold_binary_name="$scaffold_service_name"
fi
if [[ -z "$scaffold_build_path" ]]; then
  scaffold_build_path="./cmd/${scaffold_binary_name}"
fi
if [[ -z "$scaffold_env_prefix" ]]; then
  scaffold_env_prefix="$(printf '%s' "$scaffold_service_name" | tr '[:lower:].-' '[:upper:]__')"
fi
if [[ -z "$scaffold_image_repository" ]]; then
  scaffold_image_repository="$scaffold_service_name"
fi
if [[ -z "$scaffold_database_name" ]]; then
  scaffold_database_name="${scaffold_service_name//-/_}"
  if [[ ! "$scaffold_database_name" =~ ^[a-z] ]]; then
    scaffold_database_name="service_${scaffold_database_name}"
  fi
fi
if [[ "$scaffold_config_mode" != none && -z "$scaffold_config_source" ]]; then
  for scaffold_config_candidate in config.example.yaml config.yaml; do
    if [[ -f "${scaffold_target_abs}/${scaffold_config_candidate}" ]]; then
      scaffold_config_source="$scaffold_config_candidate"
      break
    fi
  done
fi

[[ "$scaffold_service_name" =~ ^[a-z0-9]+(-[a-z0-9]+)*$ ]] || die "service name must be kebab-case"
[[ "$scaffold_binary_name" =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]] || die "binary name must be a filename, not a path"
[[ "$scaffold_build_path" =~ ^\.?/[A-Za-z0-9._/-]+$ ]] || die "build path must be a relative Go package path"
[[ "$scaffold_build_path" != *".."* ]] || die "build path must stay inside the service module"
[[ "$scaffold_grpc_port" =~ ^[0-9]+$ ]] || die "gRPC port must be numeric"
((scaffold_grpc_port >= 1 && scaffold_grpc_port <= 65535)) || die "gRPC port must be between 1 and 65535"
if [[ -n "$scaffold_http_port" ]]; then
  [[ "$scaffold_http_port" =~ ^[0-9]+$ ]] || die "HTTP port must be numeric"
  ((scaffold_http_port >= 1 && scaffold_http_port <= 65535)) || die "HTTP port must be between 1 and 65535"
  [[ "$scaffold_http_port" != "$scaffold_grpc_port" ]] || die "HTTP port must differ from the gRPC port"
fi
[[ "$scaffold_env_prefix" =~ ^[A-Z][A-Z0-9_]*$ ]] || die "env prefix must be upper snake case"
[[ "$scaffold_image_repository" =~ ^[a-z0-9][a-z0-9._-]*(:[0-9]+)?(/[a-z0-9][a-z0-9._-]*)*$ ]] || die "image repository contains unsupported characters or an embedded tag"
valid_image_ref "$scaffold_health_probe_image" || die "health-probe image must be a valid image reference: $scaffold_health_probe_image"
valid_image_ref "$scaffold_runtime_image" || die "runtime image must be a valid image reference: $scaffold_runtime_image"
[[ "$scaffold_build_mode" == source || "$scaffold_build_mode" == prebuilt ]] || die "build mode must be source or prebuilt"
[[ "$scaffold_database" == none || "$scaffold_database" == postgres || "$scaffold_database" == mysql ]] || die "database must be none, postgres, or mysql"
[[ "$scaffold_database_name" =~ ^[a-z][a-z0-9_]*$ ]] || die "database name must be lowercase snake_case and start with a letter"
((${#scaffold_database_name} <= 63)) || die "database name must be at most 63 characters"
[[ "$scaffold_config_mode" == copy || "$scaffold_config_mode" == mount || "$scaffold_config_mode" == none ]] || die "config mode must be copy, mount, or none"

if [[ "$scaffold_config_mode" != none ]]; then
  [[ -n "$scaffold_config_source" ]] || die "config mode $scaffold_config_mode requires --config-source or config.example.yaml/config.yaml"
  [[ "$scaffold_config_source" != /* && "$scaffold_config_source" != *".."* ]] || die "config source must stay inside the service repository"
  [[ -f "${scaffold_target_abs}/${scaffold_config_source}" ]] || die "config source does not exist: ${scaffold_config_source}"
fi

if [[ "$scaffold_build_context" == /* ]]; then
  scaffold_context_candidate="$scaffold_build_context"
else
  scaffold_context_candidate="${scaffold_target_abs}/${scaffold_build_context}"
fi
[[ -d "$scaffold_context_candidate" ]] || die "build context does not exist: $scaffold_build_context"
scaffold_context_abs="$(cd "$scaffold_context_candidate" && pwd -P)"

if [[ "$scaffold_context_abs" == "$scaffold_target_abs" ]]; then
  scaffold_source_dir="."
  scaffold_source_prefix=""
  scaffold_dockerfile_path="Dockerfile"
elif [[ "$scaffold_target_abs" == "$scaffold_context_abs"/* ]]; then
  scaffold_source_dir="${scaffold_target_abs#"${scaffold_context_abs}/"}"
  scaffold_source_prefix="${scaffold_source_dir}/"
  scaffold_dockerfile_path="${scaffold_source_dir}/Dockerfile"
else
  die "build context must be the target directory or one of its ancestors"
fi
[[ "$scaffold_source_dir" =~ ^[A-Za-z0-9._/-]+$ ]] || die "service path relative to build context contains unsupported characters"

if [[ "$scaffold_build_mode" == source ]]; then
  [[ -f "${scaffold_target_abs}/go.mod" ]] || die "source mode requires go.mod in the target repository"
  if [[ -z "$scaffold_go_version" ]]; then
    scaffold_go_version="$(awk '$1 == "toolchain" && $2 ~ /^go[0-9]/ { sub(/^go/, "", $2); print $2; exit }' "${scaffold_target_abs}/go.mod")"
    if [[ -z "$scaffold_go_version" ]]; then
      scaffold_go_version="$(awk '$1 == "go" { print $2; exit }' "${scaffold_target_abs}/go.mod")"
    fi
  fi
  [[ "$scaffold_go_version" =~ ^[0-9]+\.[0-9]+(\.[0-9]+)?$ ]] || die "could not infer a valid Go version; pass --go-version"

  while IFS= read -r scaffold_replace_path; do
    [[ -n "$scaffold_replace_path" ]] || continue
    case "$scaffold_replace_path" in
      ./*|../*|/*)
        if [[ "$scaffold_replace_path" == /* ]]; then
          scaffold_replace_candidate="$scaffold_replace_path"
        else
          scaffold_replace_candidate="${scaffold_target_abs}/${scaffold_replace_path}"
        fi
        [[ -d "$scaffold_replace_candidate" ]] || die "local go.mod replacement does not exist: $scaffold_replace_path"
        scaffold_replace_abs="$(cd "$scaffold_replace_candidate" && pwd -P)"
        if [[ "$scaffold_replace_abs" != "$scaffold_context_abs" && "$scaffold_replace_abs" != "$scaffold_context_abs"/* ]]; then
          die "build context does not contain local replacement $scaffold_replace_path; choose a common ancestor with --build-context"
        fi
        ;;
    esac
  done < <(awk '{ for (i = 1; i <= NF; i++) if ($i == "=>") print $(i + 1) }' "${scaffold_target_abs}/go.mod")
else
  scaffold_go_version="0.0"
  if [[ ! -f "${scaffold_target_abs}/bin/${scaffold_binary_name}" ]]; then
    printf 'service-docker-scaffold: warning: bin/%s is absent; docker-build will reject it until a static Linux ELF is provided\n' "$scaffold_binary_name" >&2
  fi
fi

scaffold_dependencies=0
[[ "$scaffold_database" != none || "$scaffold_redis" == 1 ]] && scaffold_dependencies=1
scaffold_gateway=0
[[ -n "$scaffold_http_port" ]] && scaffold_gateway=1

config_env_is_managed() {
  case "$1" in
    "${scaffold_env_prefix}_LOG_LEVEL"|"${scaffold_env_prefix}_SERVER_GRPC_ADDR") return 0 ;;
  esac
  if [[ "$scaffold_gateway" == 1 && "$1" == "${scaffold_env_prefix}_SERVER_HTTP_ADDR" ]]; then
    return 0
  fi
  if [[ "$scaffold_database" != none ]]; then
    case "$1" in
      "${scaffold_env_prefix}_DATABASE_HOST"|"${scaffold_env_prefix}_DATABASE_PORT"|\
      "${scaffold_env_prefix}_DATABASE_USER"|"${scaffold_env_prefix}_DATABASE_PASSWORD"|\
      "${scaffold_env_prefix}_DATABASE_DBNAME"|"${scaffold_env_prefix}_DATABASE_SSLMODE"|\
      "${scaffold_env_prefix}_DATABASE_ROOT_PASSWORD") return 0 ;;
    esac
  fi
  if [[ "$scaffold_redis" == 1 ]]; then
    case "$1" in
      "${scaffold_env_prefix}_REDIS_ADDR"|"${scaffold_env_prefix}_REDIS_USERNAME"|\
      "${scaffold_env_prefix}_REDIS_PASSWORD"|"${scaffold_env_prefix}_REDIS_DB") return 0 ;;
    esac
  fi
  return 1
}

# Bash 3.2 treats an empty array as unset under `set -u`; keep an empty
# sentinel and skip it when rendering for macOS compatibility.
scaffold_config_env_names=("")
if [[ "$scaffold_config_mode" != none ]]; then
  while IFS= read -r scaffold_config_env_name; do
    [[ -n "$scaffold_config_env_name" ]] || continue
    if ! config_env_is_managed "$scaffold_config_env_name"; then
      scaffold_config_env_names+=("$scaffold_config_env_name")
    fi
  done < <(grep -Eo '\$\{[A-Za-z_][A-Za-z0-9_]*' "${scaffold_target_abs}/${scaffold_config_source}" \
    | sed 's/^${//' | LC_ALL=C sort -u || true)
fi

# env_default echoes the value of $1 from the target's existing .env.example,
# or empty when the file or key is absent. The service owns .env.example
# (scaffold-generated, docker-compose oriented); the renderer reads it to seed
# compose ${VAR:-default} inline defaults so docker compose up works even
# before the operator copies .env.example to .env.
env_default() {
  local scaffold_env_file="${scaffold_target_abs}/.env.example"
  [[ -f "$scaffold_env_file" ]] || return 0
  local scaffold_env_line
  scaffold_env_line="$(grep -E "^${1}=" "$scaffold_env_file" 2>/dev/null | head -n1 || true)"
  printf '%s' "${scaffold_env_line#"${1}="}"
}

condition_enabled() {
  case "$1" in
    source) [[ "$scaffold_build_mode" == source ]] ;;
    prebuilt) [[ "$scaffold_build_mode" == prebuilt ]] ;;
    postgres) [[ "$scaffold_database" == postgres ]] ;;
    mysql) [[ "$scaffold_database" == mysql ]] ;;
    redis) [[ "$scaffold_redis" == 1 ]] ;;
    dependencies) [[ "$scaffold_dependencies" == 1 ]] ;;
    gateway) [[ "$scaffold_gateway" == 1 ]] ;;
    config_copy) [[ "$scaffold_config_mode" == copy ]] ;;
    config_mount) [[ "$scaffold_config_mode" == mount ]] ;;
    *) die "unknown template condition: $1" ;;
  esac
}

render_template() {
  local scaffold_template="$1"
  local scaffold_output="$2"
  local scaffold_prefix_override="${3-$scaffold_source_prefix}"
  local scaffold_line scaffold_trimmed scaffold_condition
  local scaffold_active=1
  local scaffold_depth=0
  local -a scaffold_parent_active=()
  local scaffold_tmp
  scaffold_tmp="$(mktemp "${scaffold_target_abs}/.service-docker-scaffold.XXXXXX")"

  while IFS= read -r scaffold_line || [[ -n "$scaffold_line" ]]; do
    scaffold_trimmed="${scaffold_line#"${scaffold_line%%[![:space:]]*}"}"
    if [[ "$scaffold_trimmed" == '# --- IF '*' ---' ]]; then
      scaffold_condition="${scaffold_trimmed#'# --- IF '}"
      scaffold_condition="${scaffold_condition%' ---'}"
      scaffold_parent_active[$scaffold_depth]="$scaffold_active"
      if ((scaffold_active)) && condition_enabled "$scaffold_condition"; then
        scaffold_active=1
      else
        scaffold_active=0
      fi
      ((scaffold_depth += 1))
      continue
    fi
    if [[ "$scaffold_trimmed" == '# --- END '*' ---' ]]; then
      ((scaffold_depth > 0)) || die "unmatched END marker in $scaffold_template"
      ((scaffold_depth -= 1))
      scaffold_active="${scaffold_parent_active[$scaffold_depth]}"
      continue
    fi
    ((scaffold_active)) || continue

    if [[ "$scaffold_trimmed" == '{{CONFIG_ENVIRONMENT_LINES}}' ]]; then
      for scaffold_config_env_name in "${scaffold_config_env_names[@]}"; do
        [[ -n "$scaffold_config_env_name" ]] || continue
        printf '      - %s=${%s:-%s}\n' \
          "$scaffold_config_env_name" "$scaffold_config_env_name" \
          "$(env_default "$scaffold_config_env_name")" >> "$scaffold_tmp"
      done
      continue
    fi
    if [[ "$scaffold_trimmed" == '{{CONFIG_ENV_EXAMPLE_LINES}}' ]]; then
      for scaffold_config_env_name in "${scaffold_config_env_names[@]}"; do
        [[ -n "$scaffold_config_env_name" ]] || continue
        printf '%s=\n' "$scaffold_config_env_name" >> "$scaffold_tmp"
      done
      continue
    fi

    scaffold_line="${scaffold_line//\{\{SERVICE_NAME\}\}/$scaffold_service_name}"
    scaffold_line="${scaffold_line//\{\{ENV_PREFIX\}\}/$scaffold_env_prefix}"
    scaffold_line="${scaffold_line//\{\{BINARY_NAME\}\}/$scaffold_binary_name}"
    scaffold_line="${scaffold_line//\{\{BUILD_PATH\}\}/$scaffold_build_path}"
    scaffold_line="${scaffold_line//\{\{GO_VERSION\}\}/$scaffold_go_version}"
    scaffold_line="${scaffold_line//\{\{EXPOSE_PORT\}\}/$scaffold_grpc_port}"
    scaffold_line="${scaffold_line//\{\{HTTP_PORT\}\}/$scaffold_http_port}"
    scaffold_line="${scaffold_line//\{\{IMAGE_REPOSITORY\}\}/$scaffold_image_repository}"
    scaffold_line="${scaffold_line//\{\{HEALTH_PROBE_IMAGE\}\}/$scaffold_health_probe_image}"
    scaffold_line="${scaffold_line//\{\{RUNTIME_IMAGE\}\}/$scaffold_runtime_image}"
    scaffold_line="${scaffold_line//\{\{DATABASE_NAME\}\}/$scaffold_database_name}"
    scaffold_line="${scaffold_line//\{\{BUILD_CONTEXT\}\}/$scaffold_build_context}"
    scaffold_line="${scaffold_line//\{\{SOURCE_DIR\}\}/$scaffold_source_dir}"
    scaffold_line="${scaffold_line//\{\{SOURCE_PREFIX\}\}/$scaffold_prefix_override}"
    scaffold_line="${scaffold_line//\{\{DOCKERFILE_PATH\}\}/$scaffold_dockerfile_path}"
    scaffold_line="${scaffold_line//\{\{CONFIG_SOURCE\}\}/$scaffold_config_source}"
    printf '%s\n' "$scaffold_line" >> "$scaffold_tmp"
  done < "$scaffold_template"

  ((scaffold_depth == 0)) || die "unclosed IF marker in $scaffold_template"
  if grep -Eq '\{\{[^}]+\}\}|# --- (IF|END) ' "$scaffold_tmp"; then
    die "unresolved template syntax in $(basename "$scaffold_output")"
  fi
  mv "$scaffold_tmp" "$scaffold_output"
  chmod 0644 "$scaffold_output"
}

managed_or_absent() {
  local scaffold_path="$1"
  [[ ! -e "$scaffold_path" ]] && return 0
  grep -q 'Generated by service-docker-scaffold' "$scaffold_path" && return 0
  ((scaffold_force)) && return 0
  die "refusing to overwrite unmanaged file: $scaffold_path (pass --force after reviewing it)"
}

for scaffold_managed_path in \
  "${scaffold_target_abs}/Dockerfile" \
  "${scaffold_target_abs}/Dockerfile.dockerignore" \
  "${scaffold_target_abs}/.dockerignore" \
  "${scaffold_target_abs}/docker-compose.yaml"; do
  managed_or_absent "$scaffold_managed_path"
done

# .env.example is service-owned (scaffold-generated, app env + defaults). When
# present, read it via env_default for compose inline defaults and leave it
# intact — never clobber the service's env contract. Generate from the template
# only when absent (standalone docker skill use on a non-scaffolded service).

if [[ "$scaffold_build_mode" == source ]]; then
  scaffold_dockerfile_template="${scaffold_template_dir}/Dockerfile.tmpl"
else
  scaffold_dockerfile_template="${scaffold_template_dir}/Dockerfile.prebuilt.tmpl"
fi

render_template "$scaffold_dockerfile_template" "${scaffold_target_abs}/Dockerfile"
render_template "${scaffold_template_dir}/.dockerignore.tmpl" "${scaffold_target_abs}/Dockerfile.dockerignore"
render_template "${scaffold_template_dir}/.dockerignore.tmpl" "${scaffold_target_abs}/.dockerignore" ""
render_template "${scaffold_template_dir}/docker-compose.yaml.tmpl" "${scaffold_target_abs}/docker-compose.yaml"
if [[ ! -f "${scaffold_target_abs}/.env.example" ]]; then
  managed_or_absent "${scaffold_target_abs}/.env.example"
  render_template "${scaffold_template_dir}/.env.example.tmpl" "${scaffold_target_abs}/.env.example"
else
  printf 'service-docker-scaffold: read existing .env.example for compose defaults (not regenerated)\n' >&2
fi

scaffold_make_block="$(mktemp "${scaffold_target_abs}/.service-docker-scaffold.make.XXXXXX")"
render_template "${scaffold_template_dir}/Makefile.targets.tmpl" "$scaffold_make_block"
scaffold_makefile="${scaffold_target_abs}/Makefile"
scaffold_make_base="$(mktemp "${scaffold_target_abs}/.service-docker-scaffold.make-base.XXXXXX")"

if [[ -f "$scaffold_makefile" ]]; then
  scaffold_begin_count="$(grep -c '^# BEGIN service-docker-scaffold$' "$scaffold_makefile" || true)"
  scaffold_end_count="$(grep -c '^# END service-docker-scaffold$' "$scaffold_makefile" || true)"
  [[ "$scaffold_begin_count" == "$scaffold_end_count" ]] || die "Makefile has an incomplete service-docker-scaffold block"
  [[ "$scaffold_begin_count" == 0 || "$scaffold_begin_count" == 1 ]] || die "Makefile has duplicate service-docker-scaffold blocks"
  awk '
    /^# BEGIN service-docker-scaffold$/ { skip = 1; next }
    /^# END service-docker-scaffold$/ { skip = 0; next }
    !skip { print }
  ' "$scaffold_makefile" > "$scaffold_make_base"
else
  : > "$scaffold_make_base"
fi

scaffold_make_combined="$(mktemp "${scaffold_target_abs}/.service-docker-scaffold.make-final.XXXXXX")"
awk 'NF { last = NR } { lines[NR] = $0 } END { for (i = 1; i <= last; i++) print lines[i] }' "$scaffold_make_base" > "$scaffold_make_combined"
if [[ -s "$scaffold_make_combined" ]]; then
  printf '\n' >> "$scaffold_make_combined"
fi
sed -n 'p' "$scaffold_make_block" >> "$scaffold_make_combined"
mv "$scaffold_make_combined" "$scaffold_makefile"
chmod 0644 "$scaffold_makefile"
rm -f "$scaffold_make_block" "$scaffold_make_base"

if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  (cd "$scaffold_target_abs" && docker compose -f docker-compose.yaml config --quiet)
else
  printf 'service-docker-scaffold: warning: Docker Compose unavailable; skipped compose validation\n' >&2
fi

printf 'Rendered Docker packaging for %s in %s\n' "$scaffold_service_name" "$scaffold_target_abs"
printf 'Image: %s:${IMAGE_TAG:-latest}\n' "$scaffold_image_repository"
printf 'Build mode: %s; database: %s (%s); redis: %s; config: %s\n' \
  "$scaffold_build_mode" "$scaffold_database" "$scaffold_database_name" "$scaffold_redis" "$scaffold_config_mode"
if [[ "$scaffold_gateway" == 1 ]]; then
  printf 'Ports: gRPC %s; HTTP gateway %s\n' "$scaffold_grpc_port" "$scaffold_http_port"
else
  printf 'Port: gRPC %s; HTTP gateway disabled\n' "$scaffold_grpc_port"
fi
