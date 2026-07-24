// Package service contains demo-service business logic.
//
// Layering contract (see golang-service-development skill §2):
//   - This is the SERVICE ROOT. It holds Service struct + New + Start/Stop +
//     resource resolve helpers + one-line facade methods (one per RPC).
//   - Business logic lives in SUBPACKAGES (internal/service/<domain>/). This
//     file does NOT contain CRUD implementations — only delegations.
//   - handler calls service.X; service.X is a one-line facade that calls
//     s.<domain>.X in the subpackage. handler never imports the subpackage.
//   - Service methods take proto types DIRECTLY (e.g., *demov1.CreateDemoRequest)
//     and return proto types — no intermediate Go structs at any layer.
//   - Resources (db, demo) are constructed here from cfg (or injected via
//     option) and passed to subpackage constructors. The subpackages do NOT
//     manage resource lifecycle — this Service does via lifecycle.Manager.
//
// Lifecycle:
//   - Service holds a *lifecycle.Manager that owns all "heavy" resources
//     (DB, DemoService, future cron/mqtt/etc). Each owned resource is wrapped as
//     a lifecycle.Stopper and registered with the manager; New does NOT close
//     them on success, Stop stops all of them in reverse registration order.
//   - Injected resources (via option.WithDB etc.) are NOT registered — caller
//     owns their lifecycle.
//   - Handler exposes Start/Stop by delegating to Service, so in-process
//     module users get one-call lifecycle management.
//   - Cleanup-path close errors are logged via slog.Warn rather than returned
//     (Stopper-registered funcs run inside lifecycle.Manager; their errors
//     would be lost anyway since StopFunc drops them). This is the only place
//     service code logs directly — business code still returns errors.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"gorm.io/gorm"

	demov1 "demo-service/gen/demo/v1"
	"demo-service/internal/jobs"
	"demo-service/internal/service/demo"
	"demo-service/pkg/config"
	"demo-service/pkg/option"
	"demo-service/pkg/thirdcall"

	"github.com/servekit/go-common/cronx"
	"github.com/servekit/go-common/dbx"
	"github.com/servekit/go-common/lifecycle"
)

// Service holds demo-service business state.
//
// db and demoSvc are convenience references kept on the root Service for
// resolveXxx helpers — they point at the same instances tracked by mgr and
// injected into subpackages. Each domain lives in its own subpackage field
// (demo *demo.Service); subpackages do NOT reference this struct.
//
// Named demoSvc (not demo) to avoid colliding with the demo domain
// field below when demo happens to be "demo". Same pattern applies to
// any third-party service whose name matches a domain name.
type Service struct {
	cfg *config.Config
	mgr *lifecycle.Manager

	db      *gorm.DB
	demoSvc thirdcall.DemoService

	// One field per domain subpackage. Add fields here as new domains appear.
	demo *demo.Service
}

// New constructs a Service from config and functional options.
//
// Resources not injected via options are created from cfg, wrapped as
// lifecycle.Stoppers, and registered with the internal Manager. Stop will
// stop them in reverse order. Injected resources are NOT registered — caller
// owns their lifecycle.
//
// After resources are resolved, each domain subpackage is constructed with
// the resolved db. Subpackages do NOT participate in lifecycle.Manager
// (they have no goroutines of their own); their resources are owned by this
// Service via mgr.
//
// On partial failure (any resolve returns an error), already-registered
// components are stopped via mgr.Stop() before returning the error.
func New(cfg *config.Config, opts ...option.Option) (*Service, error) {
	o := option.Apply(opts...)
	mgr := lifecycle.NewManager()

	db, err := resolveDB(cfg, o.DB, mgr)
	if err != nil {
		// mgr has nothing registered yet, but Stop is still well-defined
		// (returns nil). Surface any error from Stop alongside the resolve
		// error in case future refactors add components before resolveDB.
		if cerr := mgr.Stop(); cerr != nil {
			err = errors.Join(err, cerr)
		}
		return nil, err
	}

	demoSvc, err := resolveDemo(cfg, o.DemoService, mgr)
	if err != nil {
		if cerr := mgr.Stop(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("rollback: %w", cerr))
		}
		return nil, err
	}

	svc := &Service{
		cfg: cfg,
		mgr: mgr,
		db:  db,
		demoSvc: demoSvc,

		// Construct each domain subpackage with injected resources.
		demo: demo.New(db),
	}

	// jobs.Scheduler owns the cron instance; setupJobs builds it, registers
	// it on mgr, and wires periodic jobs (empty by default — add jobs inside
	// setupJobs as scheduler.AddFunc calls). See golang-service-development
	// skill §10 for the jobs pattern.
	if err := svc.setupJobs(); err != nil {
		if cerr := mgr.Stop(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("rollback: %w", cerr))
		}
		return nil, err
	}

	return svc, nil
}

// Start starts all owned components concurrently. Returns the first error
// reported by any component's Start, or nil if all succeeded.
func (s *Service) Start() error { return s.mgr.Start() }

// Stop stops all owned components in reverse registration order. Errors are
// aggregated so a failure in one component doesn't mask others.
func (s *Service) Stop() error { return s.mgr.Stop() }

// --- facade methods (one per RPC, delegate to subpackage) ---
//
// handler calls these; they are one-line delegations to the corresponding
// domain subpackage. Add one facade per new RPC, never business logic here.
// Cross-domain orchestration RPCs also live here (compose multiple
// subpackages inline).

// CreateDemo delegates to the demo subpackage.
func (s *Service) CreateDemo(ctx context.Context, req *demov1.CreateDemoRequest) (*demov1.Demo, error) {
	return s.demo.CreateDemo(ctx, req)
}

// GetDemo delegates to the demo subpackage.
func (s *Service) GetDemo(ctx context.Context, id int64) (*demov1.Demo, error) {
	return s.demo.GetDemo(ctx, id)
}

// ListDemos delegates to the demo subpackage.
func (s *Service) ListDemos(ctx context.Context, pageSize int, pageToken string) (*demov1.ListDemosResponse, error) {
	return s.demo.ListDemos(ctx, pageSize, pageToken)
}

// UpdateDemo delegates to the demo subpackage.
func (s *Service) UpdateDemo(ctx context.Context, req *demov1.UpdateDemoRequest) (*demov1.Demo, error) {
	return s.demo.UpdateDemo(ctx, req)
}

// DeleteDemo delegates to the demo subpackage.
func (s *Service) DeleteDemo(ctx context.Context, id int64) error {
	return s.demo.DeleteDemo(ctx, id)
}

// --- internal helpers ---

// setupJobs builds the jobs.Scheduler, registers it on s.mgr (so its
// lifecycle is managed alongside db/demo), and wires periodic jobs.
// Signature is intentionally receiver-only: future jobs are added inside
// this method as additional scheduler.AddFunc calls — callers never need
// to extend a parameter list. Timezone default lives in config.CronConfig's
// default tag.
func (s *Service) setupJobs() error {
	scheduler, err := jobs.New(&jobs.Deps{
		Config: &cronx.Config{
			Timezone:      s.cfg.Cron.Timezone,
			OverlapPolicy: "skip",
		},
	})
	if err != nil {
		return fmt.Errorf("init jobs: %w", err)
	}
	s.mgr.Add("jobs", scheduler)

	// Register periodic jobs here. Example shape (uncomment + adapt):
	//
	//   if err := scheduler.AddFunc("*/5 * * * *", func() {
	//       ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	//       defer cancel()
	//       if err := s.demo.SomePeriodicOp(ctx); err != nil {
	//           slog.Error("demo periodic op", "error", err)
	//       }
	//   }); err != nil {
	//       return fmt.Errorf("register demo periodic op: %w", err)
	//   }
	return nil
}

// resolveDB returns the *gorm.DB to use. If the caller injected one via
// WithDB, it's returned as-is (caller owns lifecycle). Otherwise a new one
// is built from cfg and registered with mgr as a Stopper.
func resolveDB(cfg *config.Config, injected *gorm.DB, mgr *lifecycle.Manager) (*gorm.DB, error) {
	if injected != nil {
		return injected, nil
	}
	db, err := dbx.New(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	mgr.AddStopper("db", lifecycle.StopFunc(func() {
		sqlDB, err := db.DB()
		if err != nil {
			slog.Warn("get sql db for close", "error", err)
			return
		}
		if err := sqlDB.Close(); err != nil {
			slog.Warn("close db", "error", err)
		}
	}))
	return db, nil
}

// resolveDemo returns the DemoService to use. If the caller injected one via
// WithDemoService, it's returned as-is. Otherwise a new one is built from cfg
// and — if it implements Close() — registered with mgr as a Stopper.
//
// The module-mode demo backend has no resources to Close, so it is NOT
// registered (skipping the type assertion). The gRPC backend does, and will
// be registered automatically when mode=grpc.
func resolveDemo(cfg *config.Config, injected thirdcall.DemoService, mgr *lifecycle.Manager) (thirdcall.DemoService, error) {
	if injected != nil {
		return injected, nil
	}
	demo, err := thirdcall.NewDemoService(&cfg.ThirdParty.Demo)
	if err != nil {
		return nil, fmt.Errorf("init demo: %w", err)
	}
	if closer, ok := demo.(interface{ Close() error }); ok {
		mgr.AddStopper("demo", lifecycle.StopFunc(func() {
			if err := closer.Close(); err != nil {
				slog.Warn("close demo", "error", err)
			}
		}))
	}
	return demo, nil
}
