// Package handler implements demo.v1.DemoServiceServer as a thin shim over
// internal/service. Each method is a one-line delegation — service takes the
// proto request directly (per project convention: avoid unnecessary struct
// allocation; convert at the store boundary instead).
//
// Handlers hold NO business logic and NO conversion logic. Anything beyond
// `return h.svc.X(ctx, req)` belongs in internal/service.
//
// Handler also implements signalx.Service (Start/Stop) by delegating to the
// underlying Service. This lets in-process module users manage lifecycle
// (background goroutines, owned resource cleanup) via the same object they
// call RPC methods on — no separate service handle to track.
//
// If the service grows a second proto service (e.g., AdminService), add
// admin.go alongside this file; the Handler struct gains a second set of
// methods. Keep one file per proto service.
package handler

import (
	"context"

	demov1 "demo-service/gen/demo/v1"
	"demo-service/internal/service"

	"google.golang.org/protobuf/types/known/emptypb"
)

// Handler implements demo.v1.DemoServiceServer.
//
// It holds no mutable state — the embedded *service.Service owns all
// business state and lifecycle. Construction-time injection only.
type Handler struct {
	demov1.UnimplementedDemoServiceServer

	svc *service.Service
}

// New constructs a Handler wrapping svc.
func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// Compile-time assertion: Handler implements the gRPC server interface.
var _ demov1.DemoServiceServer = (*Handler)(nil)

// Start starts service-internal components (background goroutines for owned
// resources like cron, message consumers, etc.). Safe to call from in-process
// module users before invoking RPCs.
func (h *Handler) Start() error { return h.svc.Start() }

// Stop releases resources owned by the service (DB pool, gRPC client conns,
// stops background goroutines). After Stop, the Handler must not be used.
func (h *Handler) Stop() error { return h.svc.Stop() }

// CreateDemo delegates to service.CreateDemo.
func (h *Handler) CreateDemo(ctx context.Context, req *demov1.CreateDemoRequest) (*demov1.Demo, error) {
	return h.svc.CreateDemo(ctx, req)
}

// GetDemo delegates to service.GetDemo.
func (h *Handler) GetDemo(ctx context.Context, req *demov1.GetDemoRequest) (*demov1.Demo, error) {
	return h.svc.GetDemo(ctx, req.GetId())
}

// ListDemos delegates to service.ListDemos.
func (h *Handler) ListDemos(ctx context.Context, req *demov1.ListDemosRequest) (*demov1.ListDemosResponse, error) {
	return h.svc.ListDemos(ctx, int(req.GetPageSize()), req.GetPageToken())
}

// UpdateDemo delegates to service.UpdateDemo.
func (h *Handler) UpdateDemo(ctx context.Context, req *demov1.UpdateDemoRequest) (*demov1.Demo, error) {
	return h.svc.UpdateDemo(ctx, req)
}

// DeleteDemo delegates to service.DeleteDemo.
func (h *Handler) DeleteDemo(ctx context.Context, req *demov1.DeleteDemoRequest) (*emptypb.Empty, error) {
	if err := h.svc.DeleteDemo(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
