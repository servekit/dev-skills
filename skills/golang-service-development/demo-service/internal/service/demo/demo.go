// Package demo contains the demo domain business logic.
//
// Layering contract (see golang-service-development skill §2):
//   - This is a SUBPACKAGE under internal/service/. Demo methods are
//     NOT on the outer *service.Service; the outer service exposes them via
//     one-line facade methods (see ../service.go).
//   - Methods take proto types DIRECTLY (e.g., *demov1.CreateDemoRequest)
//     and return proto types — no intermediate Go structs. Conversion to
//     Go-native types happens at the store boundary (proto → models.Demo
//     before dal call; models → proto via demoToProto before return).
//   - Resources (db) are injected via New(); the subpackage does NOT
//     hold a reference to the parent *service.Service (avoids import cycle).
//   - The subpackage does NOT manage resource lifecycle — that's the parent
//     service.Service's job via lifecycle.Manager.
package demo

import (
	"context"
	"errors"
	"strconv"

	"gorm.io/gorm"

	demov1 "demo-service/gen/demo/v1"
	"demo-service/internal/store/dal"
	"demo-service/internal/store/models"
	"demo-service/pkg/xcodes"
	gidservice "github.com/servekit/gid-service/pkg"
)

// Service is the demo domain service. Resources are injected at
// construction; the subpackage does not manage their lifecycle.
type Service struct {
	db  *gorm.DB
	gid gidservice.Service
}

// New constructs a Demo domain service with injected resources.
func New(db *gorm.DB, gid gidservice.Service) *Service {
	return &Service{db: db, gid: gid}
}

// CreateDemo inserts a new demo record.
//
// The ID is generated up front via gid-service (snowflake) and
// set on the record before insert; the model's ID column is NOT autoIncrement
// in this config. CreatedAt/UpdatedAt are backfilled by GORM on
// insert. Status is the proto enum value, cast to int32 for storage.
func (s *Service) CreateDemo(ctx context.Context, req *demov1.CreateDemoRequest) (*demov1.Demo, error) {
	record := &models.Demo{
		Name:        req.GetName(),
		Description: req.GetDescription(),
		Status:      int32(req.GetStatus()),
	}

	id, err := gidservice.NextID(ctx, s.gid)
	if err != nil {
		return nil, xcodes.ErrDemoInternal.Wrapf(err, "generate demo id")
	}
	record.ID = id

	if err := dal.CreateDemo(ctx, s.db, record); err != nil {
		return nil, xcodes.ErrDemoInternal.Wrapf(err, "insert demo")
	}

	return demoToProto(record), nil
}

// GetDemo returns a single demo by ID.
//
// Returns xcodes.ErrDemoNotFound (wrapped) when the record is missing;
// xcodes.ErrDemoInternal for any other failure.
func (s *Service) GetDemo(ctx context.Context, id int64) (*demov1.Demo, error) {
	record, err := dal.GetDemo(ctx, s.db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, xcodes.ErrDemoNotFound.Wrapf(err, "demo id=%d", id)
		}
		return nil, xcodes.ErrDemoInternal.Wrapf(err, "get demo id=%d", id)
	}
	return demoToProto(record), nil
}

// DefaultPageSize bounds List when the caller doesn't pass a page size.
const DefaultPageSize = 50

// ListDemos returns up to pageSize demos, ordered by ID descending,
// skipping offset rows.
//
// pageSize <= 0 or > 100 falls back to DefaultPageSize. pageToken is the
// stringified offset returned by a previous call; pass "" for the first page.
//
// Note: OFFSET-based pagination is fine for demo / low-cardinality tables.
// For high-volume tables, switch to cursor-based (WHERE id > $last_id ORDER BY
// id LIMIT N) in the dal layer.
func (s *Service) ListDemos(ctx context.Context, pageSize int, pageToken string) (*demov1.ListDemosResponse, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = DefaultPageSize
	}

	offset := 0
	if pageToken != "" {
		if v, err := strconv.Atoi(pageToken); err == nil && v > 0 {
			offset = v
		}
	}

	// Fetch one extra to determine if a next page exists.
	records, err := dal.ListDemos(ctx, s.db, pageSize+1, offset)
	if err != nil {
		return nil, xcodes.ErrDemoInternal.Wrapf(err, "list demos offset=%d", offset)
	}

	hasNext := len(records) > pageSize
	if hasNext {
		records = records[:pageSize]
	}

	out := &demov1.ListDemosResponse{Demos: make([]*demov1.Demo, 0, len(records))}
	for _, r := range records {
		out.Demos = append(out.Demos, demoToProto(r))
	}
	if hasNext {
		out.NextPageToken = strconv.Itoa(offset + pageSize)
	}
	return out, nil
}

// UpdateDemo overwrites name, description, and status for the given ID.
//
// Returns xcodes.ErrDemoNotFound when the record is missing. Update + re-read
// pattern: dal's Update doesn't surface ErrRecordNotFound for zero-row
// updates, so we re-read both to detect missing and to return current row.
func (s *Service) UpdateDemo(ctx context.Context, req *demov1.UpdateDemoRequest) (*demov1.Demo, error) {
	id := req.GetId()
	if err := dal.UpdateDemo(ctx, s.db, id, req.GetName(), req.GetDescription(), int32(req.GetStatus())); err != nil {
		return nil, xcodes.ErrDemoInternal.Wrapf(err, "update id=%d", id)
	}

	record, err := dal.GetDemo(ctx, s.db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, xcodes.ErrDemoNotFound.Wrapf(err, "demo id=%d", id)
		}
		return nil, xcodes.ErrDemoInternal.Wrapf(err, "re-read after update id=%d", id)
	}

	return demoToProto(record), nil
}

// DeleteDemo soft-deletes the given demo (sets deleted_at).
//
// Idempotent: deleting an already-soft-deleted demo succeeds. Returns
// xcodes.ErrDemoInternal for any dal failure.
func (s *Service) DeleteDemo(ctx context.Context, id int64) error {
	if err := dal.DeleteDemo(ctx, s.db, id); err != nil {
		return xcodes.ErrDemoInternal.Wrapf(err, "delete demo id=%d", id)
	}
	return nil
}

// demoToProto converts a DB record to its proto response. Centralized so all
// CRUD methods produce identical timestamp encoding and enum cast.
//
// Enum conversion is a Go type cast on the proto-generated type
// (type DemoStatus int32). Do NOT write custom int↔enum / string↔enum
// helpers — proto already generates String(), DemoStatus_name and
// DemoStatus_value. See golang-service-development skill §6.
func demoToProto(d *models.Demo) *demov1.Demo {
	if d == nil {
		return nil
	}
	return &demov1.Demo{
		Id:          d.ID,
		Name:        d.Name,
		Description: d.Description,
		Status:      demov1.DemoStatus(d.Status),
		CreatedAt:   d.CreatedAt.UnixMilli(),
		UpdatedAt:   d.UpdatedAt.UnixMilli(),
	}
}
