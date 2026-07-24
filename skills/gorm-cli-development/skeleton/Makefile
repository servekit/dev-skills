.PHONY: gen migrate

# Regenerate type-safe query code from store/models/.
# Re-run after every model change. CI should check that this produces no diff:
#   make gen && git diff --exit-code
gen:
	gorm gen -i ./store/models -o ./store/generated

# Apply schema migrations. Run on deploy / local setup.
# In production, prefer a real migration tool (golang-migrate, atlas, etc.)
# over AutoMigrate — AutoMigrate only adds columns/indexes, never drops them.
migrate:
	go run ./cmd/migrate
