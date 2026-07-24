# Static defaults for the renderer, centralized so the default for every option
# lives in one place. render.sh sources this file and then applies --flag
# overrides on top, so a value here is used only when the matching flag is
# omitted on a given render.
#
# Per-service values that are *inferred* (service name, binary name, build path,
# env prefix, image repository, database name, Go version from go.mod, config
# file discovery) are intentionally NOT here — render.sh computes those from the
# target repository.
#
# Dependency images (postgres / mysql / redis) are a separate concern: they are
# runtime-overridable and live inline in templates/docker-compose.yaml.tmpl as
# ${POSTGRES_IMAGE:-} and friends, not here.

# --- Base images (bump versions here) ---
scaffold_health_probe_image="ghcr.io/grpc-ecosystem/grpc-health-probe:v0.4.53"
scaffold_runtime_image="alpine:3.24"

# --- Network / paths ---
scaffold_target="."
scaffold_grpc_port="9000"

# --- Build & dependency defaults (override per service with flags) ---
scaffold_build_mode="source"
scaffold_build_context="."
scaffold_database="none"
scaffold_config_mode="copy"

# --- Switches (off by default; enable per render with the matching flag) ---
scaffold_redis=0
scaffold_force=0
