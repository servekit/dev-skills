// Package option defines functional options for constructing the demo
// service with optional dependency injection.
//
// Each "heavy" resource follows the own* rule:
//   - Injected via WithXxx(...)  → own=false → service won't Stop it
//   - Not injected               → own=true  → service creates from config
//     and Stops it on shutdown
//
// This prevents double-Close and lifecycle confusion when the service runs
// embedded in a parent process that owns the resources.
//
// # Available go-common resources (menu)
//
// This scaffold wires DB + DemoService by default. Other go-common resources
// can be slotted in by mirroring the resolveDB pattern in internal/service/:
//
//	dbx       → *gorm.DB              (already wired; resolveDB)
//	redisx    → *redis.Client         (Options.Redis below; add resolveRedis)
//	cronx     → *cron.Cron            (Options.Cron below; scaffold uses jobs.Scheduler instead)
//	captcha   → purpose-scoped, create per use case (no global injection)
//	ratelimit → purpose-scoped, create per use case (no global injection)
//	lifecycle → *lifecycle.Manager    (internal to service.Service)
//	signalx   → signal handling       (owned by cmd/server/main.go)
//	grpcx     → gRPC server utilities (owned by cmd/server/main.go)
//
// Stateless utilities (xerr, ptr, jsonx, gorx, logging) are imported directly
// where needed — no injection.
package option

import (
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"demo-service/pkg/thirdcall"
)

// Option mutates Options.
type Option func(*Options)

// Options holds resolved dependencies. Nil means "not injected — service
// creates from cfg and Stops it on shutdown".
type Options struct {
	// DB is the primary relational store. resolveDB builds it from cfg.Database.
	DB *gorm.DB

	// Redis is optional. Add Redis to config.Config + a resolveRedis helper
	// in internal/service/ to auto-create; otherwise inject via WithRedis.
	Redis *redis.Client

	// Cron is optional. The scaffold already wires jobs.Scheduler (built on
	// cronx) in service.setupJobs — most periodic-task needs should extend
	// that rather than inject a separate *cron.Cron. This option exists for
	// advanced cases (e.g., a parent process sharing its scheduler).
	Cron *cron.Cron

	// DemoService is the placeholder third-party service showing the dual-mode
	// (gRPC / module) integration pattern. Replace with real services as needed.
	DemoService thirdcall.DemoService
}

// WithDB injects an existing *gorm.DB. Caller owns its lifecycle.
func WithDB(db *gorm.DB) Option { return func(o *Options) { o.DB = db } }

// WithRedis injects an existing *redis.Client. Caller owns its lifecycle.
// Add a Redis field to config.Config + a resolveRedis helper to auto-create
// from cfg instead of injecting.
func WithRedis(c *redis.Client) Option { return func(o *Options) { o.Redis = c } }

// WithCron injects an existing *cron.Cron. Caller owns its lifecycle. Most
// periodic-task needs should extend the scaffold's jobs.Scheduler instead.
func WithCron(c *cron.Cron) Option { return func(o *Options) { o.Cron = c } }

// WithDemoService injects an existing DemoService. Caller owns its lifecycle.
func WithDemoService(d thirdcall.DemoService) Option {
	return func(o *Options) { o.DemoService = d }
}

// Apply evaluates all options and returns the resolved Options. A nil field
// means "not injected — service owns it and will Stop it on shutdown":
//
//	opts := Apply(opts...)
//	if opts.DB == nil {
//	    opts.DB, err = buildDB(cfg)  // service owns; Stop will Close it
//	}
func Apply(opts ...Option) Options {
	var o Options
	for _, opt := range opts {
		opt(&o)
	}
	return o
}
