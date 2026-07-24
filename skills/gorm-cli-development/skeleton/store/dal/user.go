package dal

import (
	"context"
	"time"

	"gorm.io/gorm"

	"example.com/myorg/userservice/store/generated"
	"example.com/myorg/userservice/store/models"
)

// CreateUser inserts a new user record. u.ID is backfilled on success.
//
// dal methods accept *gorm.DB (which may be a *gorm.DB tx passed down from a
// service-layer transaction); they NEVER open transactions themselves.
func CreateUser(ctx context.Context, tx *gorm.DB, u *models.User) error {
	return gorm.G[models.User](tx).Create(ctx, u)
}

// GetUser returns the user with the given primary key, or gorm.ErrRecordNotFound.
func GetUser(ctx context.Context, tx *gorm.DB, id uint) (*models.User, error) {
	return gorm.G[models.User](tx).
		Where(generated.User.ID.Eq(id)).
		Take(ctx)
}

// GetUserByEmail looks up a user by their unique email.
// Demonstrates a string field predicate (Eq) and a Select to limit columns.
func GetUserByEmail(ctx context.Context, tx *gorm.DB, email string) (*models.User, error) {
	return gorm.G[models.User](tx).
		Select(
			generated.User.ID.Column().Name,
			generated.User.Name.Column().Name,
			generated.User.Email.Column().Name,
			generated.User.Status.Column().Name,
		).
		Where(generated.User.Email.Eq(email)).
		Take(ctx)
}

// UpdateUserAge sets only the age column for the given user.
//
// Uses Set(...) form rather than Update(ctx, struct) because:
//   1. We are updating one field, not the whole row
//   2. struct Update silently drops zero-value fields — Set never does
func UpdateUserAge(ctx context.Context, tx *gorm.DB, id uint, age int) error {
	return gorm.G[models.User](tx).
		Where(generated.User.ID.Eq(id)).
		Set(generated.User.Age.Set(age)).
		Update(ctx)
}

// UpdateUserLastLogin sets last_login to the given time (clearing it if t is nil
// would need a different API; here we always set a concrete value).
func UpdateUserLastLogin(ctx context.Context, tx *gorm.DB, id uint, t time.Time) error {
	return gorm.G[models.User](tx).
		Where(generated.User.ID.Eq(id)).
		Set(generated.User.LastLogin.Set(t)).
		Update(ctx)
}

// SuspendUser marks the user suspended and bumps age by 1 in one UPDATE.
//
// Multi-field Set example: pass multiple Set expressions inside a single
// Set(...) call. Each can be a plain value, an Incr/Decr/Mul, or a SQL expr.
// Zero values ARE written (unlike struct Update), so passing Set(""),
// Set(0), Set(false) is safe.
func SuspendUser(ctx context.Context, tx *gorm.DB, id uint) error {
	return gorm.G[models.User](tx).
		Where(generated.User.ID.Eq(id)).
		Set(
			generated.User.Status.Set("suspended"),
			generated.User.Age.Incr(1), // exercise: bump age atomically
		).
		Update(ctx)
}

// DeleteUser soft-deletes the user (sets deleted_at; field type gorm.DeletedAt).
func DeleteUser(ctx context.Context, tx *gorm.DB, id uint) error {
	return gorm.G[models.User](tx).
		Where(generated.User.ID.Eq(id)).
		Delete(ctx)
}
