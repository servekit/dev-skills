package models

import (
	"time"

	"gorm.io/gorm"
)

// User is the main account record in the user service.
//
// The service itself is named "user", so the struct keeps the bare name
// "User" (no service prefix needed — the service IS user). Other tables in
// this service that risk colliding with names from other services get a
// "User" prefix on the struct (see login_log.go for an example).
//
// Standard four fields (ID/CreatedAt/UpdatedAt/DeletedAt) are declared
// explicitly rather than via gorm.Model embedding — see skill §3 for the
// reasoning (avoids the gorm.Model uint ID type lock-in + GORM schema
// conflict bugs).
type User struct {
	ID uint `gorm:"primaryKey"` // primary key, type can be changed freely

	Name      string `gorm:"size:64;not null;index"`          // display name
	Email     string `gorm:"size:128;uniqueIndex"`            // unique login email
	Age       int    `gorm:"default:0"`                       // defaults to 0
	Status    string `gorm:"size:16;not null;default:active"` // active / suspended / ...
	LastLogin *time.Time                                    // pointer → nullable

	CreatedAt time.Time                          // GORM auto-fills on insert
	UpdatedAt time.Time                          // GORM auto-fills on insert + update
	DeletedAt gorm.DeletedAt `gorm:"index"`      // soft delete
}

func (User) TableName() string { return "users" }
