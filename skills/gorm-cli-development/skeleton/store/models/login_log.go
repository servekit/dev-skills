package models

import (
	"time"

	"gorm.io/gorm"
)

// UserLoginLog records a single login attempt by a user.
//
// Naming convention (see gorm-cli-development §3):
//   - service is "user", table semantic is "login_log"
//   - struct carries the service prefix → UserLoginLog (avoids table-name
//     collisions when multiple services share a database)
//   - file name does NOT carry the prefix → login_log.go (snake of the table
//     semantic)
//   - DB table derived from the struct name → user_login_logs
type UserLoginLog struct {
	ID uint `gorm:"primaryKey"`

	UserID  uint   `gorm:"index;not null"`                  // FK to users.id
	IP      string `gorm:"size:45;not null"`                // IPv6 max length is 45
	Agent   string `gorm:"size:256"`                        // User-Agent header
	Success bool   `gorm:"not null;default:false"`          // whether auth succeeded

	// Custom unix-millisecond timestamps in addition to the time.Time ones
	// below. They coexist with CreatedAt/UpdatedAt — they do NOT replace them.
	// OPT-IN: delete these two fields and their tags if your service doesn't
	// need integer millisecond columns.
	CreatedMs int64 `gorm:"autoCreateTime:milli"` // populated on insert, ms since epoch
	UpdatedMs int64 `gorm:"autoUpdateTime:milli"` // populated on insert + update

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (UserLoginLog) TableName() string { return "user_login_logs" }
