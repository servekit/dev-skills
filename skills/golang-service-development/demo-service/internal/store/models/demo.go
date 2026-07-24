// Package models defines GORM-backed table structs for the demo service.
//
// Conventions (see gorm-cli-development skill):
//   - Explicit ID/CreatedAt/UpdatedAt/DeletedAt fields (no gorm.Model embedding)
//   - Primary table has no service prefix; secondary tables do (e.g. DemoAuditLog)
//   - One file per table; struct name == PascalCase of table semantic
package models

import (
	"time"

	"gorm.io/gorm"
)

// Demo is the primary table of the demo service.
type Demo struct {
	// ID is auto-incremented by the database (BIGSERIAL). GORM backfills the
	// assigned value into the struct after insert.
	ID          int64  `gorm:"primaryKey;autoIncrement"`
	Name        string `gorm:"size:100;not null;index"`
	Description string `gorm:"size:500"`
	// Status stores a proto enum value as int32 (see api/proto/demo/v1/demo.proto
	// DemoStatus). DB layer stays numeric; the int32↔enum cast happens at
	// the store boundary inside the service layer (never in pkg/handler, never
	// in store). Do NOT write a custom int↔enum / string↔enum helper — proto
	// already generates String(), DemoStatus_name, DemoStatus_value.
	// See golang-service-development skill §6.
	Status    int32 `gorm:"not null;default:1;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName explicitly returns the plural table name so new-service.sh's
// ${plural} token substitution has a visible target. GORM would derive the
// same string from the struct name, but the explicit return makes the
// template's table-name placeholder obvious.
func (Demo) TableName() string { return "demos" }
