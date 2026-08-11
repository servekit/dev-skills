# CLAUDE.md

This file provides guidance to Claude Code when working in this repository.

## Project Overview

**dev-skills** is a multi-agent **plugin** — a library of domain development Skills (Go, Rust, TypeScript, OPA, Protobuf standards + a go-common service toolchain). It is NOT a build-script repo. Skills are auto-discovered by Claude Code and ZCode; a SessionStart hook injects a routing guide (`using-dev-skills`) plus an auto-generated skill index, forcing reliable skill usage. Modeled after [superpowers](https://github.com/obra/superpowers).

## Architecture

```
.claude-plugin/        plugin.json (manifest) + marketplace.json (Claude Code delivery form)
.zcode-plugin/         plugin.json (manifest) + marketplace.json (ZCode delivery form)
hooks/
  hooks.json           SessionStart hook registration (startup|clear|compact) — shared by both agents
  session-start        reads skills/using-dev-skills/SKILL.md (routing guide), appends an
                       auto-generated index of every skills/*/SKILL.md, outputs JSON
                       (hookSpecificOutput.additionalContext) → injected into each session
skills/<name>/         one dir per skill; entry is SKILL.md (frontmatter: name + description)
```

The two `.claude-plugin/` and `.zcode-plugin/` manifests are agent-specific delivery forms; they share the same `skills/` and `hooks/` trees. ZCode probes `.zcode-plugin/plugin.json` first (then falls back to `.claude-plugin/`), and its `${CLAUDE_PLUGIN_ROOT}` variable, `SessionStart` event, and `startup|clear|compact` matcher are all compatible with the existing hook — so `hooks/session-start` runs unchanged on both agents.

### How skills reach Claude Code / ZCode (two layers)

1. **Discovery** — each agent auto-scans `skills/*/SKILL.md` for every enabled plugin. No per-skill declaration; the `name` + `description` frontmatter registers each skill in the model's catalog.
2. **Forcing** — the SessionStart hook injects `using-dev-skills` (the routing guide: when to use which skill) + an auto-index of all skills. The index is generated at runtime from SKILL.md frontmatter, so adding a skill needs no hook edit.

### Skills (two tiers + meta)

- **General language standards** — `golang-development`, `rust-development`, `ts-development`, `opa-development`, `proto-development`, `gorm-cli-development`. Apply to any project in that language.
- **go-common service toolchain** — `golang-service-development`, `golang-service-docker`. Specific to `github.com/servekit/go-common` (grpcx / configx / lifecycle). Do not apply to non-go-common projects.
- **meta** — `using-dev-skills`: the routing guide, auto-injected by the hook.

Router skills (`proto/opa/rust/ts-development`) use an entry SKILL.md that delegates to sub-documents loaded on demand; sub-docs are not in the auto-index.

## Working in this repo

- Test the hook locally: `./hooks/session-start` — output must be valid JSON (pipe through `python3 -m json.tool` to check).
- After editing `skills/using-dev-skills/SKILL.md`, re-run the hook to confirm the YAML frontmatter strips cleanly and the index still renders.
