#!/usr/bin/env bash
# new-service.sh — thin wrapper that runs the scaffold generator.
#
# Usage:
#   ./scripts/new-service.sh <name>              # 默认生成到 dev-skills 平级目录
#   ./scripts/new-service.sh <name> /some/path/  # 生成到 /some/path/<name>-service/
#   ./scripts/new-service.sh --regen-demo        # 从模板重生成 demo-service/
#
# The generated go.mod consumes github.com/servekit/go-common as a normal
# remote module (require + version, no local replace); run `make tidy` in the
# new service to resolve and pin the version.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SCAFFOLD_DIR="$SKILL_ROOT/scaffold"

# Handle --regen-demo: rebuild demo-service/ inside the skill.
if [[ "${1:-}" == "--regen-demo" ]]; then
    shift
    cd "$SCAFFOLD_DIR"
    exec go run . --force --db --redis --thirdcall --example demo "$SKILL_ROOT"
fi

# Prefer a pre-built binary if SCAFFOLD_BIN is set; otherwise `go run`.
if [[ -n "${SCAFFOLD_BIN:-}" ]]; then
    exec "$SCAFFOLD_BIN" "$@"
fi

cd "$SCAFFOLD_DIR"
exec go run . "$@"
