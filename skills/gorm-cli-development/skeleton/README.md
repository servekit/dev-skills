# skeleton

A minimal Go project skeleton that follows every rule in
`SKILL.md`. **This is not a separate module that compiles on its
own** — it is a copy-paste template an AI agent can drop into a new Go service
that needs database access.

## What `skeleton/` represents

`skeleton/` plays the role of the **`<service-root>/`** placeholder used in the
skill's directory diagram. The outer directory name is **project-defined** — in
a real codebase it might be `cmd/api/`, `internal/user/`, `app/`, or just the
service's repo root. **Do not copy the `skeleton/` directory name itself**; copy
its *contents* (`store/`, `service/`, `go.mod`, `Makefile`) into whatever your
project's service root is.

Only two directory names are fixed by the convention:

- `store/` — the DB access layer root (always this name)
- `service/` — the service-layer Go package (always this name, sits next to `store/`)

Everything else is up to your project.

## What this demonstrates

| Convention in skill | Where to look |
| --- | --- |
| `store/{models,generated,dal}` layout | directory tree below |
| Explicit ID/CreatedAt/UpdatedAt/DeletedAt fields (no gorm.Model embedding) — soft delete enabled | `store/models/user.go` |
| Service-prefix model naming (`UserLoginLog`) + snake file name (`login_log.go`) | `store/models/login_log.go` |
| Migration-grade struct tags (`size`, `not null`, `index`, `uniqueIndex`, `default`) | both model files |
| Custom unix-millisecond timestamp via `autoCreateTime:milli` / `autoUpdateTime:milli` | `store/models/login_log.go` |
| `generated/` committed to git, never hand-edited, CI checks for drift | `store/generated/README.md` |
| Type-safe CRUD via `gorm.G[T]` + `generated.X.Y.Z(...)` | all `dal/*.go` files |
| Partial-field updates use `Set(...)` to avoid zero-value drop | `dal/user.go` `UpdateUserAge` |
| dal accepts `*gorm.DB`, never opens transactions itself | all `dal/*.go` files |
| Transactions wrap multiple dal calls in the service layer (db injected into a service struct via constructor) | `service/login.go` |

## How to use this for a new service

1. **Copy the *contents* of `skeleton/`** (i.e. `store/`, `service/`, `go.mod`,
   `Makefile` — not the `skeleton/` directory itself) into your new service's
   root directory. That root directory's name is **project-defined** — it might
   be `cmd/api/`, `internal/user/`, `app/`, or the repo root; pick whatever your
   project already uses. The two fixed names are `store/` and `service/`, which
   sit side-by-side inside it.
2. **Replace the module path `example.com/myorg/userservice` everywhere** — in
   `go.mod` and in every import path under `store/dal/` and `service/`. A
   project-wide search-and-replace is the easiest way to catch them all.
3. **Rename the service name from `user` to the real service name.** Three
   places need updating, and they are NOT symmetrical:
   - **Struct names**: any "secondary" table that is not the service's primary
     table needs the service prefix. The skeleton uses `User` (no prefix,
     because the service is "user") and `UserLoginLog` (carries the prefix).
     If the new service is "payment", rename to `Payment` + `PaymentRefund`
     (or whatever the secondary table is). The primary table itself never has
     a prefix; secondary tables always do.
   - **File names**: do **not** rename based on the service. File names are
     the snake_case of the *table semantic* only — `login_log.go` stays
     `login_log.go` regardless of service, because the prefix lives on the
     struct, not the file.
   - **DB table names**: derived from the struct name by GORM's snake+plural
     rule, so they follow the struct rename automatically. The `TableName()`
     string literals in each model file need to match.
4. **Rename the function names in `store/dal/`** that carry the old struct
   name: `CreateUser` → `CreatePayment`, `GetUser` → `GetPayment`,
   `UpdateUserAge` → `UpdatePaymentAge`, etc. Function names use the struct
   name (with prefix when applicable) — see `dal/user.go` and
   `dal/login_log.go` for the pattern.
5. **Decide whether to keep `CreatedMs` / `UpdatedMs`** in `login_log.go`.
   These unix-millisecond columns are an **opt-in** feature for services that
   need integer timestamps in addition to the `time.Time` ones declared
   explicitly. If you do not need them, delete the two fields and their tags.
6. Run `make gen` to populate `store/generated/`.
7. Wire the database connection at startup (see "Connecting a real DB" below).

## Generating code

```bash
make gen
```

Runs `gorm gen -i ./store/models -o ./store/generated`. Re-run whenever models
change.

CI should run `make gen && git diff --exit-code` to fail when the committed
`generated/` drifts from `models/`.

## Connecting a real DB

The skeleton does not include a connection helper because projects differ on
config loading. Typical wiring in `cmd/<svc>/main.go`:

```go
package main

import (
    "context"
    "database/sql"
    "log"

    "gorm.io/driver/mysql"
    "gorm.io/gorm"

    "example.com/myorg/userservice/store/models"
)

func main() {
    dsn := mustLoadDSN() // from env / config file
    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatalf("open db: %v", err)
    }

    if err := db.AutoMigrate(&models.User{}, &models.UserLoginLog{}); err != nil {
        log.Fatalf("migrate: %v", err)
    }

    // pass *gorm.DB to service-layer constructors
}
```

## Module path placeholder

Every import path uses `example.com/myorg/userservice`. Replace globally before
compiling. The placeholder is intentional — it makes search-and-replace obvious.
