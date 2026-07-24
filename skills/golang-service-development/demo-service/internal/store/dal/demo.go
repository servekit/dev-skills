// Package dal provides type-safe data-access functions over generated GORM
// helpers. One file per table; cross-table composition lives in the service
// layer (see gorm-cli-development skill §1).
package dal

import (
	"context"

	"gorm.io/gorm"

	"demo-service/internal/store/generated"
	"demo-service/internal/store/models"
)

// CreateDemo inserts a new demo record. d.ID is backfilled on success.
//
// dal functions accept *gorm.DB (which may be a transaction passed down from
// the service layer); they NEVER open transactions themselves.
func CreateDemo(ctx context.Context, tx *gorm.DB, d *models.Demo) error {
	return gorm.G[models.Demo](tx).Create(ctx, d)
}

// GetDemo returns the demo with the given primary key, or gorm.ErrRecordNotFound.
//
// gorm.G[T].Take returns a value type T (not *T); we take its address so the
// caller receives a pointer consistent with CreateDemo's input.
func GetDemo(ctx context.Context, tx *gorm.DB, id int64) (*models.Demo, error) {
	d, err := gorm.G[models.Demo](tx).
		Where(generated.Demo.ID.Eq(id)).
		Take(ctx)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// ListDemos returns up to limit demos ordered by ID descending, skipping
// offset rows. Callers handle pagination tokens in the service layer.
//
// gorm.G[T].Find returns []T (not []*T); we convert to []*T to match the
// pointer-style convention used elsewhere in the dal layer.
func ListDemos(ctx context.Context, tx *gorm.DB, limit, offset int) ([]*models.Demo, error) {
	demos, err := gorm.G[models.Demo](tx).
		Order(generated.Demo.ID.Desc()).
		Limit(limit).
		Offset(offset).
		Find(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*models.Demo, len(demos))
	for i := range demos {
		out[i] = &demos[i]
	}
	return out, nil
}

// UpdateDemo sets name, description, and status atomically.
//
// TEMPLATE: This function is demo-specific. When adapting this template to a
// new service, either delete it or rewrite it to match your schema's updateable
// fields. The pattern (Set expressions on generated field helpers) is what
// matters — copy that, not the specific fields.
//
// Uses Set expressions (not struct Update) to avoid zero-value drop: passing
// an empty description string still writes NULL/' to the column.
//
// status is the raw int32 of a proto enum value — the dal layer stays numeric
// and never imports proto. The int32↔enum cast lives in the service layer
// (never in pkg/handler, never in store). See golang-service-development
// skill §6.
func UpdateDemo(ctx context.Context, tx *gorm.DB, id int64, name, description string, status int32) error {
	_, err := gorm.G[models.Demo](tx).
		Where(generated.Demo.ID.Eq(id)).
		Set(
			generated.Demo.Name.Set(name),
			generated.Demo.Description.Set(description),
			generated.Demo.Status.Set(status),
		).
		Update(ctx)
	return err
}

// DeleteDemo soft-deletes the demo (sets deleted_at; field type gorm.DeletedAt).
//
// Single-record dal convention: returns error only. See storage-service.DeleteFile.
func DeleteDemo(ctx context.Context, tx *gorm.DB, id int64) error {
	_, err := gorm.G[models.Demo](tx).
		Where(generated.Demo.ID.Eq(id)).
		Delete(ctx)
	return err
}
