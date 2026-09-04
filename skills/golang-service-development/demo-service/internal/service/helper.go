package service

// This file holds the resource resolve helpers used by service.New. They were
// extracted from service.go to keep that file focused on the Service struct,
// New/Start/Stop/Ping, and the RPC facade delegations.
//
// Every dependency follows the platform's inject-or-build contract: an
// injected one (option.With…) is used as-is with the caller owning its
// lifecycle; otherwise it is built from cfg via the provider's Connect
// (dbx/redisx for resources, the dependency's pkg.Connect for services),
// which also registers its lifecycle with the Manager.

import (
	"fmt"

	"github.com/redis/go-redis/v9"

	"gorm.io/gorm"

	gidservice "github.com/servekit/gid-service/pkg"
	gidconfig "github.com/servekit/gid-service/pkg/config"

	"demo-service/pkg/config"
	"demo-service/pkg/option"

	"github.com/servekit/go-common/dbx"
	"github.com/servekit/go-common/lifecycle"
	"github.com/servekit/go-common/redisx"
)

// resolveDB returns the DB to use: an injected one as-is (caller owns
// lifecycle), otherwise built from cfg with a Stopper registered on mgr via
// dbx.Connect.
func resolveDB(o *option.Options, cfg *config.Config, mgr *lifecycle.Manager) (*gorm.DB, error) {
	return dbx.Connect(cfg.Database, o.DB, mgr)
}

// resolveRedis returns the Redis client to use: an injected one as-is
// (caller owns lifecycle), otherwise built from cfg with a Stopper registered
// on mgr via redisx.Connect.
func resolveRedis(o *option.Options, cfg *config.Config, mgr *lifecycle.Manager) (*redis.Client, error) {
	return redisx.Connect(cfg.Redis, o.Redis, mgr)
}

// resolveGID returns the gid dependency. Construction delegates to
// gidservice.Connect, which owns the mode switch and lifecycle registration;
// only the adoption of a parent-injected Handler stays here — it reads this
// service's own options and the parent owns that lifecycle.
func resolveGID(o *option.Options, cfg *config.RemoteServiceConfig[*gidconfig.Config], mgr *lifecycle.Manager) (gidservice.Service, error) {
	if o.GIDHandler != nil {
		return o.GIDHandler, nil // injected → borrowed; parent owns lifecycle
	}
	if cfg == nil {
		return nil, fmt.Errorf("third_party.gid: not configured")
	}
	gid, _, err := gidservice.Connect(gidservice.ConnectConfig{
		Mode:   cfg.Mode,
		Target: cfg.Target,
		Config: cfg.Config,
	}, mgr)
	return gid, err
}
