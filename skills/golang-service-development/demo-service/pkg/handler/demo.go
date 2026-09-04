// Package handler implements demo.v1.DemoServiceServer as a thin shim over
// internal/service. Each method is a one-line delegation — service takes the
// proto request directly (convert at the store boundary, not here).
//
// Handlers hold NO business logic and NO conversion logic. Anything beyond
// `return h.svc.X(ctx, req)` belongs in internal/service.
//
// Handler also implements signalx.Service (Start/Stop) by delegating to the
// underlying Service, so in-process module users manage lifecycle via the same
// object they call RPC methods on.
package handler

import (
	"context"

	demov1 "demo-service/gen/demo/v1"
	"demo-service/internal/service"

	"google.golang.org/protobuf/types/known/emptypb"
)

// Handler implements demo.v1.DemoServiceServer. It holds no mutable
// state — the embedded *service.Service owns all business state and lifecycle.
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
// resources like cron, message consumers, etc.).
func (h *Handler) Start() error { return h.svc.Start() }

// Stop releases resources owned by the service. After Stop, the Handler must
// not be used.
func (h *Handler) Stop() error { return h.svc.Stop() }

// Ping is a health-check RPC, always generated so the grpc-gateway has at
// least one HTTP endpoint and pkg/server.go can always register the gateway
// handler. (A proto service with zero RPCs produces no HandlerFromEndpoint,
// which would silently disable the HTTP gateway.)
func (h *Handler) Ping(ctx context.Context, _ *emptypb.Empty) (*demov1.Pong, error) {
	return h.svc.Ping(ctx)
}

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
