package service

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"example.com/myorg/userservice/store/dal"
	"example.com/myorg/userservice/store/models"
)

// LoginService owns the user-facing login business logic. The *gorm.DB is
// injected at construction time and held on the struct; service methods only
// expose business parameters in their signatures.
type LoginService struct {
	db *gorm.DB
}

// NewLoginService constructs a LoginService. Inject the *gorm.DB from your
// startup wiring (cmd/main.go, DI container, etc.).
func NewLoginService(db *gorm.DB) *LoginService {
	return &LoginService{db: db}
}

// LoginParams describes a single login attempt to be recorded.
type LoginParams struct {
	UserID  uint
	IP      string
	Agent   string
	Success bool
}

// ErrUserNotFound is returned when the user referenced by LoginParams.UserID
// does not exist.
var ErrUserNotFound = errors.New("user not found")

// RecordLogin updates the user's last_login timestamp AND writes a login log
// entry, atomically.
//
// Canonical "multiple writes that must succeed or fail together" shape: the
// transaction lives here in the service layer (per gorm-cli-development §8),
// wrapping calls into the dal layer. dal methods receive the tx and stay
// transaction-agnostic.
//
// Single-statement updates (e.g. dal.UpdateUserAge by itself) do NOT need a
// transaction wrapper — call the dal method directly.
func (s *LoginService) RecordLogin(ctx context.Context, p LoginParams) error {
	// Existence check outside the transaction — it's a read, no atomicity needed.
	if _, err := dal.GetUser(ctx, s.db, p.UserID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	now := time.Now()
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := dal.UpdateUserLastLogin(ctx, tx, p.UserID, now); err != nil {
			return err
		}
		return dal.CreateLoginLog(ctx, tx, &models.UserLoginLog{
			UserID:  p.UserID,
			IP:      p.IP,
			Agent:   p.Agent,
			Success: p.Success,
		})
	})
}
