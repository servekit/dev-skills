#!/usr/bin/env bash
# new-service.sh — thin wrapper that runs the scaffold generator.
#
# Usage:
#   ./scripts/new-service.sh <name> [target-parent] [capability flags]
#   ./scripts/new-service.sh --regen-demo
#
# Capability flags (default ALL OFF = minimal shell that runs without Postgres):
#   --db            PostgreSQL via dbx (postgres-only; go-common has no mysql path)
#   --redis         Redis via redisx
#   --thirdcall     gid-service dependency (snowflake ID generator)
#   --example       CRUD starter domain (implies --db)
#   Negate with --no-db / --no-redis / --thirdcall / --no-example; last wins.
#
# If NO capability flag is given:
#   - stdin is a tty  → prompts for each (default N)
#   - stdin not a tty → minimal shell (all off), warns on stderr
# Agents that have inferred the answers MUST pass the flags directly (no prompt).
#
# The generated go.mod consumes github.com/servekit/go-common as a normal
# remote module (require + version, no local replace); run `make tidy` in the
# new service to resolve and pin the version.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SCAFFOLD_DIR="$SKILL_ROOT/scaffold"

# Handle --regen-demo: rebuild demo-service/ inside the skill (all capabilities
# on, so the golden sample stays the full-featured reference).
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

# If no capability flag is present, collect capabilities interactively (tty)
# or fall back to a minimal shell (non-tty). Any capability flag present
# disables prompting entirely (the agent/explicit-user path).
has_cap_flag=false
for a in "$@"; do
    case "$a" in
        --db|--no-db|--redis|--no-redis|--thirdcall|--no-thirdcall|--example|--no-example)
            has_cap_flag=true; break;;
    esac
done

cap_flags=""
if [[ "$has_cap_flag" == "false" ]]; then
    if [[ -t 0 ]]; then
        echo "No capability flags given. Answer to enable (default N):"
        ask() {
            local prompt="$1" flag="$2" ans
            read -r -p "$prompt [y/N] " ans || ans=""
            case "$ans" in y|Y|yes|YES) cap_flags="$cap_flags $flag";; esac
        }
        ask "  Database (PostgreSQL)?" "--db"
        ask "  Redis?" "--redis"
        ask "  Third-party call (gid-service)?" "--thirdcall"
        ask "  Example CRUD domain?" "--example"
        # Mirror scaffold's example->db invariant for clarity in the prompt flow.
        if [[ "$cap_flags" == *"--example"* && "$cap_flags" != *"--db"* ]]; then
            cap_flags="$cap_flags --db"
            echo "  note: --example implies --db (enabling database)" >&2
        fi
    else
        echo "new-service: non-interactive, generating minimal shell (all capabilities off); pass --db/--redis/--thirdcall/--example to enable" >&2
    fi
fi

# shellcheck disable=SC2086 # cap_flags is intentionally word-split (space-separated flags)
exec go run . "$@" $cap_flags
