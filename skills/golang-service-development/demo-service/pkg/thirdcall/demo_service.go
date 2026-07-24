// Package thirdcall defines stable interfaces to external services consumed
// by this service. Implementations live under internal/thirdcall/<name>/.
//
// Splitting interface (pkg/) from implementation (internal/) lets callers
// inject mocks without importing the concrete backend, and keeps the surface
// area minimal for downstream module users.
package thirdcall

import (
	"context"
	"fmt"

	"demo-service/internal/thirdcall/demoservice"
	"demo-service/pkg/config"
)

// DemoService is a self-contained placeholder showing the dual-mode (gRPC /
// module) third-party integration pattern. It deliberately depends on no real
// external service so the scaffold stays generic — copy this shape when
// wiring real services (PaymentService, StorageService, ...).
//
// One method is enough to demonstrate the wiring: interface → factory → two
// backends → config → option injection. Real services add methods as needed.
type DemoService interface {
	// DoDemo echoes input with the configured prefix (module mode) or rounds
	// the call through a remote gRPC server (grpc mode).
	DoDemo(ctx context.Context, input string) (string, error)
}

// NewDemoService constructs a DemoService from config.
//
//	cfg.Mode == "grpc"     → dials a remote demo-service over gRPC
//	cfg.Mode == "module"   → in-process implementation (no external dep)
//	cfg.Mode == ""         → treated as "module" (dev default)
//	anything else          → error
func NewDemoService(cfg *config.RemoteServiceConfig[config.DemoServiceConfig]) (DemoService, error) {
	switch cfg.Mode {
	case "grpc":
		return demoservice.NewGRPC(cfg.Target)
	case "module", "":
		return demoservice.NewModule(&cfg.Config)
	default:
		return nil, fmt.Errorf("unknown demo mode %q (want \"grpc\" or \"module\")", cfg.Mode)
	}
}
