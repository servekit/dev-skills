# generated/

This directory is populated by `gorm gen`. **Commit it to git.** Generated
diffs in PRs are useful — they let reviewers see the type-safety impact of a
model change at a glance.

**Do not hand-edit files here** — they will be overwritten on the next
generation run. To change what's generated, edit the model in `store/models/`,
run `make gen`, and commit both the model change and the regenerated files
together.

```bash
make gen
```

CI should run `make gen && git diff --exit-code` to fail when someone forgets
to regen after editing a model.

## What ends up here

After `gorm gen` runs against `store/models/`:

- `<model>.gen.go` — one per model struct (e.g. `user.gen.go`, `login_log.gen.go`)
  containing field helpers like `generated.User.ID.Eq(...)`.
- `query.gen.go` — the `gorm.G[T]` builder entry and any typed-query interfaces
  declared in the models package.
