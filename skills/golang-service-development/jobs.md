---
name: golang-service-jobs
description: "Sub-document of golang-service-development skill. Loaded by SKILL.md when adding periodic/background work to a service: cron jobs, scheduled cleanup, quota recompute, cache invalidation sweeps, heartbeat reporters. Covers the internal/jobs/ Scheduler package, the receiver-only svc.setupJobs() pattern, ownsCron semantics, and the CronConfig layout. NOT for one-shot bootstrap tasks or event-driven (MQ) consumers."
---

# 后台 cron jobs（internal/jobs + setupJobs）

本文件是 `golang-service-development` 的后台定时任务专题。何时读本文件见入口 `SKILL.md` 的「子文档路由」表。资源管理的基础（`lifecycle.Manager`、close-only 资源注册）见 `architecture.md` §5。

凡是需要周期性触发的逻辑（清理过期会话、汇总统计、重算 quota、cache 失效扫描、心跳上报……），统一走 `internal/jobs/` 包，不要把 cron 实例散落到各业务子包。

## 包结构与职责

```
internal/jobs/
└── jobs.go    # Scheduler + New + AddFunc + Start/Stop
```

`Scheduler` 是一个**纯调度器**：
- 持有一个 `*cron.Cron`（自建或外部注入）
- 实现 `lifecycle.Service`（Start 启动 cron，Stop 等 in-flight 任务排空）
- 暴露 `AddFunc(spec, cmd)` 给 caller 注册任务
- **不 import 任何业务子包**——它不知道要跑什么任务

业务子包（`internal/service/<domain>/`）提供"执行一次清理/汇总"的方法（如 `upload.ReapExpiredSessions(ctx)`），由 service 装配层决定何时调用。

## API

```go
type Scheduler struct { ... }

type Deps struct {
    Config *cronx.Config  // 用于自建 cron；为 nil 时 Cron 必须非 nil
    Cron   *cron.Cron     // 可选注入；非 nil 时 Config 被忽略，caller 自管 lifecycle
}

func New(d *Deps) (*Scheduler, error)

// 暴露 cron.AddFunc，caller 决定挂什么任务
func (s *Scheduler) AddFunc(spec string, cmd func()) error

// 实现 lifecycle.Service
func (s *Scheduler) Start() error
func (s *Scheduler) Stop() error
```

**`ownsCron` 字段是关键**：当 `Deps.Cron == nil`（Scheduler 自建）时 ownsCron=true，Start/Stop 控制底层 cron；当 `Deps.Cron != nil`（注入）时 ownsCron=false，Start/Stop 是 no-op，caller 全权管 lifecycle。这是为了避免 robfig/cron 的 `Stop()` 非幂等性（每次调用都起一个 goroutine 等 jobWaiter）， borrower 不能调 owner 的 Stop。

## service 装配（setupJobs 方法）

在 `internal/service/service.go` 加一个 **receiver-only** 的 setupJobs 方法：

```go
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

    // 在这里加 scheduler.AddFunc。s.cfg / s.<domain> 全从 receiver 取，
    // 签名恒定不变——未来加 N 个任务，调用方 svc.setupJobs() 一行不变。
    //
    //   if err := scheduler.AddFunc(s.cfg.Upload.ReapCronSpec, func() {
    //       ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    //       defer cancel()
    //       if err := s.upload.ReapExpiredSessions(ctx); err != nil {
    //           slog.Error("upload reap", "error", err)
    //       }
    //   }); err != nil {
    //       return fmt.Errorf("register upload reap: %w", err)
    //   }
    return nil
}
```

`New()` 末尾调一次：

```go
svc := &Service{ ... }
if err := svc.setupJobs(); err != nil {
    if cerr := mgr.Stop(); cerr != nil {
        err = errors.Join(err, fmt.Errorf("rollback: %w", cerr))
    }
    return nil, err
}
return svc, nil
```

启动链是间接的：`svc.Start() → mgr.Start() → scheduler.Start() → cron.Start()`。和 db/redis/gid 一样走 lifecycle 一致路径，service.go 看不到显式 `cron.Start()`。

## 为什么不把 cron 散到各子包

| 反模式 | 问题 |
|--------|------|
| 每个子包持有 `*cron.Cron` + 各自 RegisterXxx | 调度配置散落，cron 实例归属混乱 |
| 子包既写业务 RPC 又写后台任务入口 | 职责错位——RPC 是同步路径，cron 是后台路径 |
| `pkg/server.go` / `pkg/module.go` 各装配一份 jobs | 入口重复，被迫给 service 加 `Upload()` / `AddExtra()` 之类的 public API |
| 在 service.New 里 inline `cron.New + AddFunc` 长串 | New 函数膨胀，每次加任务都要改 New |

`setupJobs` 是 receiver-only method，**签名恒定**——未来加任务只改 setupJobs 内部，`New()` 永远是 `if err := svc.setupJobs(); err != nil { ... }` 一行。

## 何时不用

- **一次性引导任务**（启动时跑一次）：直接在 `New()` 里 `go func() { ... }()`，不需要 cron
- **事件驱动而非时间驱动**（消息队列触发）：用 Kafka/MQTT consumer，注册到 mgr 而非 jobs
- **完全无后台任务的小服务**：scaffold 仍然生成 jobs 包（轻量，~100 行），不用就在 setupJobs 内不加 AddFunc 即可

## 配置

`pkg/config/config.go` 已经声明 `CronConfig`：

```go
type Config struct {
    ...
    Cron *CronConfig
}

type CronConfig struct {
    Timezone string `default:"Asia/Shanghai"`
}
```

YAML：

```yaml
cron:
  timezone: "Asia/Shanghai"
```

每个具体任务的 cron spec（如 upload reap 的 `*/5 * * * *`）放在**该任务所属领域的配置**下（如 `cfg.Upload.ReapCronSpec`），不在 `CronConfig` 里——`CronConfig` 只放调度器本身的设置（timezone）。

## 关联

- **入口**：`SKILL.md`（何时用、高频任务快速路径、子文档路由）
- **基础**：`architecture.md` §5（lifecycle.Manager、资源注册方式表）
- **同级子文档**：`enum.md`、`scaffold.md`
