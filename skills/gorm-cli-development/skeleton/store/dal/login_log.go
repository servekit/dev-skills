package dal

import (
	"context"
	"time"

	"gorm.io/gorm"

	"example.com/myorg/userservice/store/generated"
	"example.com/myorg/userservice/store/models"
)

// CreateLoginLog inserts a login log entry. log.ID is backfilled on success.
func CreateLoginLog(ctx context.Context, tx *gorm.DB, log *models.UserLoginLog) error {
	return gorm.G[models.UserLoginLog](tx).Create(ctx, log)
}

// ListLoginLogByUser returns login logs for a user within [start, end),
// newest first, capped at limit. Demonstrates:
//   - chained Where (AND by default)
//   - time predicates (Gte / Lt)
//   - typed Order on a generated field
//   - pagination via Offset/Limit (Offset omitted here because the caller
//     always wants the most recent page)
func ListLoginLogByUser(
	ctx context.Context,
	tx *gorm.DB,
	userID uint,
	start, end time.Time,
	limit int,
) ([]*models.UserLoginLog, error) {
	return gorm.G[models.UserLoginLog](tx).
		Where(generated.UserLoginLog.UserID.Eq(userID)).
		Where(generated.UserLoginLog.CreatedAt.Gte(start)).
		Where(generated.UserLoginLog.CreatedAt.Lt(end)).
		Order(generated.UserLoginLog.CreatedAt.Desc()).
		Limit(limit).
		Find(ctx)
}

// CountFailedAttemptsSince returns the number of failed logins for a user
// since the given cutoff. Demonstrates a bool predicate (Eq) and the typed
// Count() terminal. Note: Success is a non-nullable bool in the model, so the
// generated helper only exposes Eq/Neq (no IsNull/IsNotNull).
func CountFailedAttemptsSince(
	ctx context.Context,
	tx *gorm.DB,
	userID uint,
	since time.Time,
) (int64, error) {
	return gorm.G[models.UserLoginLog](tx).
		Where(generated.UserLoginLog.UserID.Eq(userID)).
		Where(generated.UserLoginLog.Success.Eq(false)).
		Where(generated.UserLoginLog.CreatedAt.Gte(since)).
		Count(ctx)
}
