---
name: golang-service-development
description: MUST use when creating, scaffolding, or architecting a new Go microservice in this monorepo (any repo ending in -service whose go.mod contains github.com/servekit/go-common). Covers directory layout (pkg/internal/cmd/api/gen), the pkg/handler ↔ internal/service layering rule, three-mode runtime (standalone gRPC / HTTP gateway / in-process module), the thirdcall interface/impl split, functional options + lifecycle.Manager resource management, proto enum → DB int handling, and the new-service.sh scaffold workflow. Trigger keywords: new service, scaffold, grpc-gateway project layout, pkg/handler, internal/service, thirdcall pattern, cmd/server, in-process module, lifecycle.Manager.
---

# Go 微服务开发指南

定义本团队所有 Go 微服务的目录结构、分层、bootstrap 模板。**这是架构层规则**——具体 Go 风格、proto 写法、GORM 用法、go-common API 都在各自专门的 skill 里，本 skill 只负责把它们组装成一个完整的服务。

## 0. 何时用本 skill

**必加载**：

- 新建任何 `-service` 后缀的 Go 项目（**第一次**用 scaffold 生成骨架，之后手写演进，详见 §8）
- 在已有服务里加 `pkg/handler`、`internal/service`、`internal/thirdcall/` 等结构
- 讨论"这个文件应该放 pkg 还是 internal"
- 用 `new-service.sh` 脚手架

**不加载**（用别的 skill）：

| 任务 | Skill |
|------|-------|
| Go 命名/错误处理/并发/测试 | golang-development |
| 写 .proto、配 buf | proto-development |
| store/{models,generated,dal}、gorm gen | gorm-cli-development |
| configx/redisx/dbx/xerr/grpcx 等基础库 API | go-common-usage |

## 1. 目录布局

```
{service_name}-service/
├── api/proto/{svc}/v1/         # proto 定义（路径 = package 路径）
├── bin/                        # 编译产物（gitignore）
├── cmd/
│   └── server/                 # 服务入口：serve（默认）+ migrate 子命令（单二进制）
├── gen/                        # buf 生成产物（committed）
├── internal/                   # 业务实现，外部不可 import
│   ├── provider/               # 辅助业务：mqtt/kafka/jobs 等
│   ├── service/                # 业务逻辑（一个领域 = 一个子包；service.go 是本体 + facade，详见 §2）
│   ├── store/                  # DB 访问（遵循 gorm-cli-development）
│   │   ├── generated/          # gorm gen 产物
│   │   ├── models/             # 表 struct
│   │   └── dal/                # 类型安全 CRUD
│   └── thirdcall/              # 第三方调用实现
│       └── {name}/             # 一个第三方 = 一个子目录
│           ├── grpc.go
│           ├── module.go
│           └── http.go (可选)
├── pkg/                        # 公共能力，可作为 module 被 import
│   ├── config/                 # 配置
│   ├── handler/                # ★ proto service 的薄壳实现
│   ├── option/                 # functional options
│   ├── thirdcall/              # 第三方接口 + 工厂
│   ├── xcodes/                 # 错误码（按域分文件）
│   ├── client.go               # gRPC 客户端
│   ├── module.go               # in-process 入口
│   └── server.go               # gRPC + gateway server
├── buf.yaml / buf.gen.yaml     # buf v2 配置
├── Makefile / Dockerfile / docker-compose.yaml / .golangci.yml / CLAUDE.md / README.md
├── config.example.yaml         # 纯结构，值全 ${VAR}（由 configx WithExpandEnv 展开）
├── .env.example                # docker 取向默认值（scaffold 生成，render.sh 读取）
└── go.mod
```

### 为什么这样切

- **`pkg/` 是公共 API**：被外部模块 import 时（in-process 模式），它定义稳定的契约。改 `pkg/` 等于 breakage。
- **`internal/` 是实现细节**：可以自由重构，外部不会感知。
- **`api/` 和 `gen/` 分离**：source (`api/proto/`) 和 derived (`gen/`) 都进 git，但分别由人和工具维护。
- **`cmd/` 严格薄**：只装配 + 启动，业务逻辑零容忍。

### `pkg/handler` 的角色（核心）

**handler 是 proto service 的薄壳**——每个 RPC 方法一行委托：

```go
// pkg/handler/demo.go
func (h *Handler) CreateDemo(ctx context.Context, req *demov1.CreateDemoRequest) (*demov1.Demo, error) {
    return h.svc.CreateDemo(ctx, req)
}
```

**handler 不写业务，也不做协议转换**。Service 直接接受 proto 类型，转换发生在 service 内部的 store 边界（见 §6 枚举处理）。

### 反模式：handler 和 service 合并到一个 struct

不要把 gRPC stub 和业务方法都挂在同一个 struct 上：

```go
// ❌ 反模式：service.go 里同时装 stub + 业务
type DemoService struct {
    demov1.UnimplementedDemoServiceServer  // embed 让 *DemoService 满足 gRPC 接口
    db *gorm.DB
    gid thirdcall.GIDService
}

// gRPC stub（public，导出）
func (s *DemoService) CreateDemo(ctx context.Context, req *demov1.CreateDemoRequest) (*demov1.Demo, error) {
    return s.createDemo(ctx, req)  // 委托给私有方法
}

// 业务实现（private，小写）
func (s *DemoService) createDemo(ctx context.Context, req *demov1.CreateDemoRequest) (*demov1.Demo, error) {
    // 业务逻辑
}
```

**问题**：
1. **handler 和 service 边界消失**：gRPC 框架替换（比如换成 connect-go）要改 service 代码；in-process 调用方也得通过这个混合 struct
2. **stub + 业务方法成对**（`CreateDemo` + `createDemo`）：每加一个 RPC 要加两个方法，重复且容易漂移
3. **interface satisfaction 隐式**：`*DemoService` 因为 embed 而满足 `DemoServiceServer`，编译器不会提醒你"这只是 stub"

**正确做法**（本 skill 推荐）：handler 和 service 严格分离。
- `pkg/handler.*Handler` 只实现 `demov1.DemoServiceServer` 接口（一行委托）
- `internal/service.*Service` 持有所有业务状态 + 业务方法，**直接接受 proto 类型**，在调用 dal 前把 proto 字段拆到 `models.Demo`

这样 handler 可以被任何 gRPC/gateway 框架替换，service 可以在 in-process module 模式下被直接调用（`hdl.GetDemo(ctx, req)` 走的就是 service 方法），同时不需要在 handler ↔ service 之间分配中间结构体。

## 2. Service 文件粒度（每个领域一个子包）

`internal/service/` 下的代码按**领域**拆分，**一个领域 = 一个子包**。`service.go` 是 service 本体 + 对 handler 的 facade 收口；业务实现一律在 `internal/service/<domain>/` 子包。**不存在**单文件领域。

### 目录结构

```
internal/service/
├── service.go              # Service struct + New + Start/Stop + resolveXxx + facade 方法
└── demo/                   # 每个领域一个子包
    └── demo.go             # type Service + New + 业务方法 + 内部 helper（xxxToProto / 常量）
```

每个子包：

- 有自己的 `type Service struct{...}` + `New(...)` 构造函数 + 业务方法
- 内部 helper（`xxxToProto`、常量、状态机等）藏在子包里，不对外暴露
- 资源（db / gid 等）通过 `New(...)` **注入**，不持有父 `*service.Service` 引用（避免循环依赖）

### 反例：按方法拆分

```text
❌ internal/service/
   ├── create.go            # Demo 的 Create
   ├── get.go               # Demo 的 Get
   ├── list.go              # Demo 的 List
   ├── update.go            # Demo 的 Update
   └── delete.go            # Demo 的 Delete
```

5 个文件装的都是 Demo 这个领域的 CRUD——按方法拆而不是按领域拆。正确做法：合并到 `demo/demo.go` 子包主文件。

### 反例：领域散在多个子包

```text
❌ internal/service/
   └── audit/
       ├── audit.go
       └── audit_snapshots.go   # 都是 audit，凭什么拆？
```

合并到 `audit/audit.go`。如果 audit 真的复杂到要拆，**在 audit 子包内部**用 `snapshots.go`、`gc.go` 等命名拆分，**不是**再开一个 audit 相关的子包。

### 子包内部文件太大怎么办

子包主文件 `<domain>/<domain>.go` 长到难以一眼看懂（经验值：超过 ~500 行），**在子包内部**拆分：

```
internal/service/
└── upload/
    ├── upload.go           # 主入口（New + service.go 调用的公开方法）
    ├── session.go          # session 状态机（仅子包内可见）
    ├── sts.go              # STS 凭证生成
    └── gc.go               # 后台清理
```

子包内部怎么拆，handler 完全不需要知道——它只看 `service.go` 这层 facade。

### `service.go` 的内容

`service.go` 只放三类东西，**不写业务逻辑**：

**1. Service 本体**：
- `Service` struct 定义（持有子包实例）
- `New()` 构造函数（含 resolveDB / resolveGID 等资源注入 helper）
- `Start()` / `Stop()` 生命周期方法

**2. facade 方法**（每个 RPC 一个，**强制**）：
- 与 proto RPC **一一对应**的公开方法
- 一行委托到子包

**3. 跨域编排 facade**（如果某 RPC 跨多个领域）：
- 在 facade 方法里组合多个子包

```go
// internal/service/service.go
type Service struct {
    cfg *config.Config
    mgr *lifecycle.Manager

    db  *gorm.DB
    gid thirdcall.GIDService

    // 每个领域一个子包实例
    demo      *demo.Service
    order     *order.Service
    inventory *inventory.Service
    payment   *payment.Service
}

// 单领域 RPC —— 一行委托
func (s *Service) CreateDemo(ctx context.Context, req *demov1.CreateDemoRequest) (*demov1.Demo, error) {
    return s.demo.CreateDemo(ctx, req)
}

// 跨域 RPC —— 在 facade 里组合多个子包
func (s *Service) PlaceOrder(ctx context.Context, req *v1.PlaceOrderRequest) (*v1.Order, error) {
    order, err := s.order.Create(ctx, req)
    if err != nil { return nil, err }
    if err := s.inventory.Reserve(ctx, order.Items); err != nil { return nil, err }
    if err := s.payment.Charge(ctx, order); err != nil { return nil, err }
    return order, nil
}
```

### 资源注入到子包

子包 `New` 通过参数注入资源，不持有父 `*Service` 引用：

```go
// internal/service/demo/demo.go
package demo

type Service struct {
    db  *gorm.DB
    gid thirdcall.GIDService
}

func New(db *gorm.DB, gid thirdcall.GIDService) *Service {
    return &Service{db: db, gid: gid}
}
```

父 `Service.New` resolve 完 db / gid 后构造所有子包：

```go
// internal/service/service.go
func New(cfg *config.Config, opts ...option.Option) (*Service, error) {
    // ...resolve db, gid...
    return &Service{
        // ...本体字段...
        demo: demo.New(db, gid),
    }, nil
}
```

### 为什么所有领域都强制子包（不保留单文件选项）

1. **handler↔service 契约稳定** — handler 永远调 `h.svc.X`。领域从 5 个方法长到 50 个方法、从单子包主文件拆成 5 个子包文件，handler 调用方式零改变。
2. **service.go 职责单一** — 只装本体 + facade，业务逻辑零容忍。读 service.go 一眼能看到所有 RPC 入口。
3. **跨域编排有自然位置** — service.go facade 方法直接组合多个子包，不用纠结"塞哪个领域文件"。
4. **领域隔离** — 子包是 Go 的可见性边界，domain A 的内部 helper 不会污染 domain B。

代价是每个领域多一层 `s.demo.X` 跳转 + 初始多一个子包目录。**可接受**——demo 这种简单服务也走这个模式，一致性比省一个目录重要。

### 子包的后台 goroutine

子包通常**不需要**自己的 `Start` / `Stop` —— 它没有后台 goroutine，资源（DB/gID）由父 `Service.mgr` 统一管。

如果将来某子包需要 consumer/cron，父 `Service.New` 把它作为 `lifecycle.Starter` 注册到 `mgr`：

```go
mgr.Add("demo-consumer", demo.NewConsumer(...))
```

**不要**给子包 struct 加 `Start`/`Stop` 方法然后从父转发——会让子包侵入 lifecycle 管理，违背"资源由父统一管"的原则。

### handler 的契约

handler **永远只调** `service.go` 这一层的方法。不 import `internal/service/<domain>` 子包，不感知 service.go 哪些方法是 facade、哪些是编排——它只看 service 这一层暴露的 API。这是 service 层给 handler 的契约边界。

## 3. 启动三件套

### cmd/server/main.go（服务 + 迁移入口，单二进制）

服务启动和数据库迁移合并到同一个二进制 `cmd/server`，通过子命令区分：

- 无参 / `serve` —— 启动 gRPC + HTTP 服务（默认）
- `migrate` —— 跑 GORM AutoMigrate 后退出
- 其他 —— 打印用法，exit 2

`main.go` 只做 dispatch，真正的启动/迁移逻辑在 `runServer` / `runMigrate`（迁移逻辑放 `cmd/server/migrate.go`，只依赖 DB，不碰 Redis/gRPC）。两者都返回 error、由 `main` 统一 `os.Exit`——revive `deep-exit` 要求 `os.Exit` 只能在 `main`：

```go
// cmd/server/main.go
func main() {
    switch subcommand() {
    case "", "serve":
        if err := runServer(); err != nil {
            slog.Error("serve failed", "error", err)
            os.Exit(1)
        }
    case "migrate":
        if err := runMigrate(); err != nil {
            slog.Error("migrate failed", "error", err)
            os.Exit(1)
        }
    default:
        fmt.Fprintf(os.Stderr, "usage: %s [serve|migrate]\n", os.Args[0])
        os.Exit(2)
    }
}

func runServer() error {
    cfg, err := config.Load()
    if err != nil { return fmt.Errorf("load config: %w", err) }
    logging.Setup(cfg.Log)
    srv, err := pkg.NewServer(cfg)
    if err != nil { return fmt.Errorf("init server: %w", err) }
    if err := signalx.RunWithForceQuit(srv); err != nil {
        return fmt.Errorf("run server: %w", err)
    }
    return nil
}

// cmd/server/migrate.go
func runMigrate() error {
    cfg, err := config.Load()
    if err != nil { return fmt.Errorf("load config: %w", err) }
    logging.Setup(cfg.Log)
    db, err := dbx.New(cfg.Database)
    if err != nil { return fmt.Errorf("init database: %w", err) }
    return runMigration(db)
}

func runMigration(db *gorm.DB) error {
    if err := dbx.AutoMigrate(db, models.AllModels()...); err != nil {
        return fmt.Errorf("auto-migrate: %w", err)
    }
    return nil
}
```

`signalx.RunWithForceQuit` 处理 SIGTERM 优雅停机 + SIGINT 强制退出。迁移**与发版解耦**——部署时先单独跑 `./<svc> migrate`（或 `make migrate`），再启动服务；Docker 镜像 ENTRYPOINT 即该二进制，`docker run <img> migrate` 直接跑迁移。

**注意**：AutoMigrate 只做加法（建表、加字段、加索引）。删字段/删表写独立脚本。

### pkg/server.go（gRPC + HTTP server）

`*Server` 同时持有 `*service.Service` 和 `*handler.Handler`：
- Server 负责组合 service.Start + gRPC server.Start（含 HTTP gateway）
- Handler 是无状态壳（仅 gRPC stub + 委托 + 自身 Start/Stop 转发给 service）

Start/Stop 的关键模式：**失败时回滚已启动组件**。

```go
func (s *Server) Start() error {
    if err := s.svc.Start(); err != nil { return err }
    if err := s.grpcSrv.Start(); err != nil {
        return errors.Join(err, s.svc.Stop())  // rollback
    }
    return nil
}
```

`svc.Start()` 现在会触发 lifecycle.Manager 并发启动所有注册的 Starter（DB/GID 这种 close-only 资源 Start 是 no-op，cron/consumer 这种会真正起 goroutine）。

### pkg/client.go（gRPC 客户端）

embedding `demov1.DemoServiceClient` 让调用方直接调用 RPC 方法。

### pkg/module.go（in-process 入口）

```go
func NewModule(cfg *config.Config, opts ...option.Option) (*Handler, error)
```

**只返回 `*Handler`**——Handler 就是对外能力，调用方不需要也不应该拿到 service handle。如果需要控制资源（DB、GID）生命周期，通过 `option.WithDB` / `option.WithGIDService` 注入，让父进程拥有资源、负责清理：

```go
hdl, err := demopkg.NewModule(cfg, option.WithDB(parentDB))
if err != nil { panic(err) }
demo, err := hdl.GetDemo(ctx, &demov1.GetDemoRequest{Id: 1})
// parentDB.Close() by the parent — module never owns it
```

注意：`Service.Start()` 在 demo 里是 no-op（没有后台 goroutine），所以 in-process 用户不需要调 Start。如果未来 service 加了 cron / 消费者，再考虑把 Start/Stop 也提升到 Handler 上。

## 4. Thirdcall 双层模式

### pkg/thirdcall/（接口 + 工厂）

```go
type GIDService interface {
    NextID() (int64, error)
}

func NewGIDService(cfg *config.RemoteServiceConfig[config.SnowflakeConfig]) (GIDService, error) {
    switch cfg.Mode {
    case "grpc":     return gidservice.NewGRPC(cfg.Target)
    case "module","": return gidservice.NewModule(&cfg.Config)
    }
}
```

### internal/thirdcall/{name}/（实现）

每个第三方一个子目录：
- `grpc.go` — gRPC 客户端实现
- `module.go` — in-process 实现
- `http.go`（可选）— HTTP 客户端实现

**依赖方向**：`pkg/thirdcall/` 定义接口，`internal/thirdcall/` 实现。包外只能看到接口；切后端（gRPC ↔ module）只需改 config。

### 关键约束

- `internal/thirdcall/<name>/` 是**唯一**允许 import 第三方 proto/gen 的地方
- service / handler / pkg 都 import `pkg/thirdcall` 的**接口**，不直接 import 实现

## 5. Option + lifecycle.Manager 资源管理

`pkg/option/option.go` 定义 functional options：

```go
type Option func(*Options)

type Options struct {
    DB         *gorm.DB
    GIDService thirdcall.GIDService
}

func WithDB(db *gorm.DB) Option { return func(o *Options) { o.DB = db } }
func WithGIDService(g thirdcall.GIDService) Option { ... }
func Apply(opts ...Option) Options { ... }
```

### 用 lifecycle.Manager 而不是 ownX bool

不要在 Service struct 上为每个资源加 `ownX bool`——依赖一多就爆炸（`ownDB`、`ownGID`、`ownRedis`、`ownCron`、`ownMQTT`...）。改用 go-common 的 `lifecycle.Manager`：

```go
type Service struct {
    cfg *config.Config
    mgr *lifecycle.Manager  // tracks every owned resource

    // 直接引用（CRUD 方法用），跟 mgr 里的实例是同一份
    db  *gorm.DB
    gid thirdcall.GIDService
}

func New(cfg *config.Config, opts ...option.Option) (*Service, error) {
    o := option.Apply(opts...)
    mgr := lifecycle.NewManager()

    db, err := resolveDB(cfg, o.DB, mgr)        // 注入 → 不注册；自建 → 注册为 Stopper
    if err != nil {
        return nil, errors.Join(err, mgr.Stop())  // 已注册的回滚
    }
    gid, err := resolveGID(cfg, o.GIDService, mgr)
    if err != nil {
        return nil, errors.Join(err, mgr.Stop())
    }
    return &Service{cfg: cfg, mgr: mgr, db: db, gid: gid}, nil
}

func (s *Service) Start() error { return s.mgr.Start() }  // 并发启动所有 Starter
func (s *Service) Stop() error  { return s.mgr.Stop() }   // 反序停止所有 Stopper
```

go-common 的 `lifecycle` 包已经提供了适配器，直接用就行——不要自己再造 `closeStopper` 类型：

- `lifecycle.StopFunc(func())` —— 把 `func()` 包成 Stopper（Start no-op）。Close 类资源的 error 用 `slog.Warn` 记录即可，**不要**在 service 里造类型去 preserve error——cleanup path 的 close 失败基本无 actionable 价值，记日志足够
- `lifecycle.StartFunc(func() error)` —— 把 `func() error` 包成 Starter（Stop no-op），用于只 Start 不 Stop 的场景
- 真有 Start + Stop 两个阶段（cron / consumer），写个实现 `lifecycle.Service` 的小 struct，用 `mgr.Add(name, svc)` 注册

```go
// resolveDB 里：Close 用 lifecycle.StopFunc + slog.Warn
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

// resolveGID 里：gRPC client 的 Close 同理
if closer, ok := gid.(interface{ Close() error }); ok {
    mgr.AddStopper("gid", lifecycle.StopFunc(func() {
        if err := closer.Close(); err != nil {
            slog.Warn("close gid", "error", err)
        }
    }))
}
```

**关于"service 不打日志"的例外**：全局规则是库代码（`internal/` 业务逻辑）不直接打日志，通过返回 error 交给调用方。但 cleanup path 是例外——Stopper 注册到 lifecycle.Manager 后，Stop 时由 Manager 调用，error 即使返回也会被 `StopFunc` 丢弃。所以 close 错误只能用 slog 记录，没别的出口。

- **调用方注入 (`WithDB(db)`)** → resolveXxx 直接返回，不注册到 mgr → Stop 不管它，调用方负责清理
- **调用方没注入** → resolveXxx 从 cfg 自建 + 注册到 mgr → Stop 按 LIFO 反序清理

### Handler 满足 signalx.Service

Handler 实现 Start/Stop（委托给 service），同时满足 `demov1.DemoServiceServer` 和 `signalx.Service` 两个接口。这让 in-process module 调用方在同一个对象上既调 RPC 又管生命周期：

```go
hdl, err := demopkg.NewModule(cfg, option.WithDB(parentDB))
if err != nil { panic(err) }
if err := hdl.Start(); err != nil { panic(err) }  // 启动后台 goroutine（cron 等）
defer hdl.Stop()                                    // 清理 owned 资源
demo, err := hdl.GetDemo(ctx, &demov1.GetDemoRequest{Id: 1})
```

如果 service 内部没有任何后台 goroutine（demo 当前就是），`Start` 等价于 no-op，但 API 已经为将来加 cron / consumer 留好了位置。

### 资源类型对应注册方式

| 资源 | 注册方式 | 说明 |
|------|---------|------|
| DB pool | `mgr.AddStopper("db", lifecycle.StopFunc(...))` | 只需 Close，没 Start |
| gRPC client conn | `mgr.AddStopper("gid", lifecycle.StopFunc(...))` | 同上 |
| Redis client | `mgr.AddStopper("redis", lifecycle.StopFunc(...))` | 同上 |
| Cron (周期任务) | `mgr.Add("jobs", scheduler)` via `svc.setupJobs()` | 见下方"后台 cron jobs"小节 |
| Kafka consumer | `mgr.Add("consumer", &consumerComponent{...})` | Start 启动 goroutine；Stop 停 + 等 in-flight |
| MQTT client | `mgr.Add("mqtt", &mqttComponent{...})` | 同上 |

注意 lifecycle.Manager 的语义：**Start 并发**调用（每个 Service 在自己 goroutine 里跑 Start，所以 blocking Starts 不会互相阻塞），**Stop 反序串行**。如果某个组件的 Start 必须等另一个先 ready，把它放在 mgr 里更后面注册是不够的——需要显式同步，或者拆成两步（构造时建立连接，Start 时启动 goroutine）。

### 后台 cron jobs（jobs.Scheduler）

凡是需要周期性触发的逻辑（清理过期会话、汇总统计、重算 quota、cache 失效扫描、心跳上报……），统一走 `internal/jobs/` 包，不要把 cron 实例散落到各业务子包。

#### 包结构与职责

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

#### API

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

#### service 装配（setupJobs 方法）

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

#### 为什么不把 cron 散到各子包

| 反模式 | 问题 |
|--------|------|
| 每个子包持有 `*cron.Cron` + 各自 RegisterXxx | 调度配置散落，cron 实例归属混乱 |
| 子包既写业务 RPC 又写后台任务入口 | 职责错位——RPC 是同步路径，cron 是后台路径 |
| `pkg/server.go` / `pkg/module.go` 各装配一份 jobs | 入口重复，被迫给 service 加 `Upload()` / `AddExtra()` 之类的 public API |
| 在 service.New 里 inline `cron.New + AddFunc` 长串 | New 函数膨胀，每次加任务都要改 New |

`setupJobs` 是 receiver-only method，**签名恒定**——未来加任务只改 setupJobs 内部，`New()` 永远是 `if err := svc.setupJobs(); err != nil { ... }` 一行。

#### 何时不用

- **一次性引导任务**（启动时跑一次）：直接在 `New()` 里 `go func() { ... }()`，不需要 cron
- **事件驱动而非时间驱动**（消息队列触发）：用 Kafka/MQTT consumer，注册到 mgr 而非 jobs
- **完全无后台任务的小服务**：scaffold 仍然生成 jobs 包（轻量，~100 行），不用就在 setupJobs 内不加 AddFunc 即可

#### 配置

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

## 6. 枚举处理（proto enum → DB int）

**核心规则**：枚举统一定义在 proto 文件中，DB 存 `int32`，应用层**只能**用 proto 编译器生成的内置函数做转换。**禁止自定义任何枚举转换函数**——`int↔enum`、`string↔enum`、`enum↔string`、`enum↔int slice` 一律不许写 helper，全部走下表的 proto 内置 API。本节是枚举处理的真相之源。

### Proto 编译器已生成的内置函数（不要重新发明）

每个 proto enum，`protoc-gen-go` 都会在 `gen/<svc>/v1/*.pb.go` 自动生成下列 API——**直接调用，不要重新实现**。以 `demo.v1.DemoStatus` 为例：

| 需求 | 内置 API | 示例结果 |
|------|---------|---------|
| `int32` → enum | `demov1.DemoStatus(i)` 类型断言 | `demov1.DemoStatus(1)` → `DemoStatus_DEMO_STATUS_ACTIVE` |
| enum → `int32` | `int32(e)` 类型断言 | `int32(demov1.DemoStatus_DEMO_STATUS_ACTIVE)` → `1` |
| enum → 字符串名 | `e.String()` 方法 | `demov1.DemoStatus_DEMO_STATUS_ACTIVE.String()` → `"DEMO_STATUS_ACTIVE"` |
| `int32` → 字符串名 | `demov1.DemoStatus_name` map | `demov1.DemoStatus_name[1]` → `"DEMO_STATUS_ACTIVE"` |
| 字符串名 → `int32` | `demov1.DemoStatus_value` map | `demov1.DemoStatus_value["DEMO_STATUS_ACTIVE"]` → `1` |
| 字符串名 → enum | `demov1.DemoStatus(demov1.DemoStatus_value[name])` | `demov1.DemoStatus(demov1.DemoStatus_value["DEMO_STATUS_ACTIVE"])` → `DemoStatus_DEMO_STATUS_ACTIVE` |
| 检查 int32 是否合法 enum 值 | 查 `_name` map | `_, ok := demov1.DemoStatus_name[i]; return ok` |

**典型 service 路径只用前两个**（DB 存取时的类型断言）。`String()` 方法和两个 map 只在日志标签、admin UI、YAML 配置反序列化等少数边界场景用到——见 §6 末尾「边界用例」。

> **判定准则**：如果你写的函数输入/输出都是裸 `int32` 或 `string`，那就是绕开 proto、重新发明转换，**禁止**。合法的业务封装（如中文标签）输入或输出**必须有一个是 proto enum 类型**——见 §6 边界用例。

### Proto 定义

```proto
enum DemoStatus {
  DEMO_STATUS_UNSPECIFIED = 0;
  DEMO_STATUS_ACTIVE = 1;
  DEMO_STATUS_ARCHIVED = 2;
}

message Demo {
  // ...
  DemoStatus status = 6;
}
```

protovalidate 用 `enum: {defined_only: true}` 拒绝未定义值（如 99）。注意 `enum` 和 `not_in` 是 FieldRules 的互斥子字段，不能写在同一个 `(buf.validate.field)` 里——要排除 UNSPECIFIED 必须用 CEL 表达式或 service 层显式校验。

### DB 层（model + dal）

```go
// internal/store/models/demo.go
type Demo struct {
    // ...
    Status int32 `gorm:"not null;default:1;index"`  // 数字存储，不是 enum 类型
}
```

dal 层**永远**用 `int32`，**永远不** import proto。GORM gen 会为 `int32` 字段生成 `field.Number[int32]` 辅助器：

```go
generated.Demo.Status.Set(status)  // status 是 int32
```

### Service 层（直接吃 proto，store 边界转换）

Service 方法直接接受 proto 类型。proto enum → int32 的转换发生在调用 dal **之前**，作为字段抽取的一部分。**注意**：方法定义在领域子包（`internal/service/demo/demo.go`），`s` 是子包的 `*demo.Service`：

```go
// internal/service/demo/demo.go
func (s *Service) CreateDemo(ctx context.Context, req *demov1.CreateDemoRequest) (*demov1.Demo, error) {
    id, err := s.gid.NextID()
    if err != nil {
        return nil, xcodes.ErrDemoInternal.Wrapf(err, "generate id")
    }

    // proto → models（store 边界）：int32(req.GetStatus()) 是 Go 类型断言
    record := &models.Demo{
        ID:     id,
        Name:   req.GetName(),
        Status: int32(req.GetStatus()),
    }
    if err := dal.CreateDemo(ctx, s.db, record); err != nil { ... }
    return demoToProto(record), nil  // models → proto，合在本文件内（见下）
}
```

反向（DB 读出来 → 返回给调用方）的 `demoToProto` 跟业务方法放**同一个子包主文件**，不单独建 `convert.go`：

```go
// internal/service/demo/demo.go（同上文件，紧邻 CRUD 方法）
func demoToProto(d *models.Demo) *demov1.Demo {
    return &demov1.Demo{
        // ...
        Status: demov1.DemoStatus(d.Status),  // int32 → proto enum（Go 类型断言）
    }
}
```

**service.go 的一行 facade 委托到子包**，handler 不知道子包存在：

```go
// internal/service/service.go
func (s *Service) CreateDemo(ctx context.Context, req *demov1.CreateDemoRequest) (*demov1.Demo, error) {
    return s.demo.CreateDemo(ctx, req)
}
```

**为什么 `int32(x)` 和 `demov1.DemoStatus(x)` 不算"自定义转换函数"**：proto 编译器生成的 `type DemoStatus int32` 让两者都是 Go 语言原生的类型断言（zero-cost，编译期完成），不是函数调用。proto 还生成了 `String()` 方法和 `DemoStatus_name` / `DemoStatus_value` 两个 map（见上面「Proto 编译器已生成的内置函数」表）——只要 DB 存的是 `int32`、应用层不涉及字符串，service 路径只用类型断言；字符串相关的 map 留给日志/admin UI 等边界场景。

### Handler 层（不碰枚举）

Handler 是一行委托，**完全不知道枚举存在**：

```go
// pkg/handler/demo.go
func (h *Handler) CreateDemo(ctx context.Context, req *demov1.CreateDemoRequest) (*demov1.Demo, error) {
    return h.svc.CreateDemo(ctx, req)  // req 透传给 service，handler 零转换
}
```

### 反模式（禁止）

每个 AI 想"自己写一个 helper"的冲动，proto 都已经提供了等价物——下表一一对应。**看到下面左边任意一行，立刻换成右边**：

| 想写的 helper（禁止） | proto 内置等价物 | 说明 |
|---------------------|-----------------|------|
| `func statusToInt(s DemoStatus) int32` | `int32(s)` | 类型断言 |
| `func intToStatus(i int32) DemoStatus` | `demov1.DemoStatus(i)` | 类型断言 |
| `func statusName(s DemoStatus) string` | `s.String()` 或 `demov1.DemoStatus_name[int32(s)]` | proto 内置方法/map |
| `func parseStatus(name string) (DemoStatus, error)` | `demov1.DemoStatus(demov1.DemoStatus_value[name])` + 查 map 判 ok | proto 内置 map |
| `func isValidStatus(i int32) bool` | `_, ok := demov1.DemoStatus_name[i]; return ok` | 查 proto 内置 map |
| `func statusesToIntSlice(ss []DemoStatus) []int32` | 直接 range 用 `int32(x)` | 不要包一层 |
| `func statusValues() []DemoStatus` | `demov1.DemoStatus.Values()`（proto 反射 API） | 不要硬编码常量列表 |

```go
// ❌ 一律禁止：自定义 int↔enum、string↔enum、enum↔string 转换
func statusToString(s int32) string { ... }
func parseStatus(s string) int32 { ... }
func isValidStatus(s int32) bool { ... }

// ❌ 也禁止：在 handler 做协议转换（service 直接吃 proto）
func (h *Handler) CreateDemo(ctx, req) (*Demo, error) {
    in := service.CreateDemoInput{Status: int32(req.GetStatus())}  // 多此一举
    res, _ := h.svc.CreateDemo(ctx, in)
    return toProto(res), nil
}

// ❌ model 不要用 enum 类型字段（store 层会耦合 proto）
type Demo struct {
    Status demov1.DemoStatus
}
```

### 边界用例（基于 proto 内置 map，不重新发明）

只有 proto 提供的 enum name 不够用时（中文标签、YAML 配置反序列化、admin UI 文案），才允许写业务函数。这时**仍然不要重新发明 string↔enum 的解析逻辑**——基于 proto 内置 map 封装一层业务语义即可。

**ok-check 的判定**（什么时候该写 `_, ok := ...`，什么时候一行搞定）：

- **不可信来源**（用户输入、外部 API、YAML 配置、迁移期间的老脏数据）→ 用 comma-ok 检查 key 是否存在，unknown 时返回 error。见下面 `ParseStatusName`。
- **可信来源**（DB 已落库的 enum 字段、内部 API 返回、proto 常量字面量）→ **免 ok-check**，直接 `demov1.DemoStatus(demov1.DemoStatus_value[s])` 一行。`_value[s]` 缺 key 返回 `0`（`_UNSPECIFIED`），不会 panic——可信来源不会触发，免 ok-check 不是 unsafe，是为了不让噪音盖过业务逻辑。

```go
// ✅ 合法：业务层 i18n，proto 不提供中文标签。输入是 proto enum 类型，不是 int32/string
var statusLabels = map[demov1.DemoStatus]string{
    demov1.DemoStatus_DEMO_STATUS_ACTIVE:   "活跃",
    demov1.DemoStatus_DEMO_STATUS_ARCHIVED: "已归档",
}

func StatusLabel(s demov1.DemoStatus) string {
    if label, ok := statusLabels[s]; ok {
        return label
    }
    return "未知"
}

// ✅ 合法：YAML 配置反序列化（不可信来源）。直接用 proto 内置 _value map，不写 switch/case
func ParseStatusName(name string) (demov1.DemoStatus, error) {
    v, ok := demov1.DemoStatus_value[name]
    if !ok {
        return 0, fmt.Errorf("unknown DemoStatus: %s", name)
    }
    return demov1.DemoStatus(v), nil
}

// ✅ 合法：可信来源（DB 已落库的 enum 字段、内部 API 返回）——免 ok-check
// _value map 缺 key 返回 0（UNSPECIFIED），不会 panic；可信数据不会触发
status := demov1.DemoStatus(demov1.DemoStatus_value[row.StatusStr])

// ✅ 合法：日志结构化字段直接用 enum 的 String()，不需要任何 helper
slog.Info("status changed", "from", oldStatus.String(), "to", newStatus.String())
```

**判定准则（再说一次，因为这是 AI 最容易滑过去的口子）**：函数的输入或输出**必须有一个是 proto enum 类型**（`demov1.DemoStatus`，不是裸 `int32`/`string`），才不算"重新发明转换"。如果输入输出都是 `int32`/`string`，那就是把 proto 内置能力重写一遍，**禁止**——直接用 §6 顶部的速查表。

## 7. 项目级约定

### 错误码（pkg/xcodes/）

按**域**分文件，不按类型分：

```
pkg/xcodes/
├── demo.go          # demo 域
├── quota.go         # 配额域
├── audit.go         # 审计域
└── ...
```

通用错误（`ErrInternal`、`ErrBadRequest`、`ErrNotFound`）直接复用 `go-common/xerr` 包；域错误在每个域文件里定义。

### buf 配置

`buf.yaml`：v2 格式，依赖 `protovalidate` + `googleapis`，lint 用 STANDARD（去除三个 RPC_REQUEST/RESPONSE 规则）。

`buf.gen.yaml`：v2 + managed mode + `go_package_prefix: <svc>-service/gen`，三个插件：`protocolbuffers/go`、`grpc/go`、`grpc-ecosystem/gateway`。

### Makefile 标准目标

每个 `-service` 项目必须提供：`build`、`run`、`test`、`lint`、`fmt`、`vet`、`generate`、`proto`、`migrate`、`tidy`。

### Docker 打包（交接给 golang-service-docker）

`Dockerfile` / `docker-compose.yaml` / `.dockerignore` / Makefile 的 `docker-*` target 由脚手架在生成服务后**自动交给 `golang-service-docker` 的 `render.sh`** 产出：标准多阶段构建（`golang:<ver>-bookworm` 编译 → `alpine:3.24` 运行时，带 `grpc_health_probe` 健康检查 + 平台矩阵）。scaffold **不**自带简陋 Dockerfile 模板，避免和 docker skill 的标准件冲突。

配套的配置契约（三个必须一起）：

- `config.example.yaml` 是**纯结构**——每个值都是 `${VAR}` 占位符，零字面量。
- `pkg/config/config.go` 的 `configx.Load(...)` **必须传 `configx.WithExpandEnv()`**，`${VAR}` 才会在启动时从环境展开（默认不开）。
- `.env.example`（scaffold 生成）是 **docker-compose 取向**的默认值源（host 名是 compose 服务名，如 `DATABASE_HOST=postgres`）。本地 `make run` 需 `cp .env.example .env` 并把 docker host 名改成本地地址；`docker compose up` 直接用默认值即可。

演进时（加 Postgres / 改端口 / 切 prebuilt）重跑 `golang-service-docker` 的 `render.sh`，幂等——它读取 `.env.example`（不覆盖）并用其默认值生成 compose 内联默认。

### .golangci.yml

复用 `dev-skills/skills/golang-development/.golangci.yml` 模板，`local-prefixes` 改成本服务 module 名 + `go-common`。

### grpcx.New 必须传三个东西

```go
grpcSrv := grpcx.New(
    &grpcx.ServerConfig{GRPCAddr: ..., GatewayAddr: ...},
    func(gs *grpc.Server) { demov1.RegisterDemoServiceServer(gs, hdl) },
    demov1.RegisterDemoServiceHandlerFromEndpoint,  // 启用 HTTP gateway
    grpcx.ErrorInterceptor,                          // xerr → gRPC status 映射
    protovalidate_middleware.UnaryServerInterceptor(validator),  // protovalidate
)
```

**漏掉任何一个，对应功能静默失效**：
- registerGW 为 nil → HTTP gateway 不启动（即使 GatewayAddr 已配）
- ErrorInterceptor 缺失 → 所有 xerr 错误变成 `codes.Unknown`
- protovalidate 缺失 → `(buf.validate.field)` 规则不生效

参考 `gid-service/pkg/server.go` 是这套模式的 canonical 实现。

## 8. 脚手架：scaffold + templates

### 使用边界（重要）

scaffold **只用于全新项目的初始化**，不用于已有项目的迭代：

| 场景 | 用 scaffold？ | 怎么做 |
|------|--------------|--------|
| 全新服务，从零开始搭框架 | ✅ 用 | `./scripts/new-service.sh <name>` 生成骨架 |
| 已有服务，加新领域/RPC/字段 | ❌ 不用 | 按 §1-§7 的约定**手写代码**，加在对应目录 |
| 已有服务，重命名/重构 | ❌ 不用 | 手改，scaffold 不知道你的业务变更 |
| 改 skill 模板本身（影响未来生成） | ✅ 用 `--regen-demo` | 改 `scaffold/templates/`，重生成 demo-service 验证 |

scaffold 是 **one-shot bootstrapping 工具**，不是代码维护工具。生成的代码归你所有，之后所有的演进都按 skill 文档手写——加新 RPC 就在 proto 里加 + 在 `service.go` 加 facade + 在 `internal/service/<domain>/` 子包实现业务 + handler 加委托方法，不要回头让 scaffold 重新生成。

为什么有这条限制：scaffold 重生成是覆盖式的（`--force` 会清掉 target 再写）。如果对已有项目跑 scaffold，所有业务代码、单元测试、自定义配置都会丢。

### 总体设计

模板用 Go `text/template`，模板变量类型安全（不会再有 sed 误伤 token 的边角 case）。三层结构：

```
skills/golang-service-development/
├── scaffold/              # Go generator（嵌入所有模板）
│   ├── main.go            # CLI + 模板渲染逻辑
│   ├── go.mod
│   └── templates/         # *.tmpl 文件（templates 是真相之源）
├── demo-service/          # 由 scaffold 生成（不是手维护）
└── scripts/
    └── new-service.sh     # 瘦壳，调 go run ./scaffold
```

**策略 B：templates 是源，demo-service 由 scaffold 重生成**。改模板后跑 `./scripts/new-service.sh --regen-demo` 重新生成 demo-service/，确保两者永远一致。

### 用法

```bash
# 新建服务（默认 target = dev-skills 平级目录）
./skills/golang-service-development/scripts/new-service.sh user

# 在指定父目录创建
./skills/golang-service-development/scripts/new-service.sh pay /some/parent/

# 重生成 demo-service（改完模板后跑）
./skills/golang-service-development/scripts/new-service.sh --regen-demo
```

### 模板变量

`scaffold/main.go` 的 `Spec` struct 定义了所有可用变量，模板里通过 `{{.X}}` 引用：

| 变量 | 含义 | demo 渲染为 | note 渲染为 |
|------|------|------------|------------|
| `{{.Name}}` | 小写服务名 | `demo` | `note` |
| `{{.Pascal}}` | 首字母大写 | `Demo` | `Note` |
| `{{.Module}}` | Go module 名 | `demo-service` | `note-service` |
| `{{.Plural}}` | 复数小写 | `demos` | `notes` |
| `{{.NameUpper}}` | 全大写（envPrefix、enum 值、xcodes reason） | `DEMO` | `NOTE` |

生成的新服务 `go.mod` 把 `github.com/servekit/go-common` 作为普通远程依赖（`require` + 版本，**无本地 replace**）；进新服务目录后跑 `make tidy` 解析并钉版本即可。

### 模板示例

```go
// scaffold/templates/pkg/handler/{{.Name}}.go.tmpl
package handler

import (
    "context"
    {{.Name}}v1 "{{.Module}}/gen/{{.Name}}/v1"
    "{{.Module}}/internal/service"
    "google.golang.org/protobuf/types/known/emptypb"
)

type Handler struct {
    {{.Name}}v1.Unimplemented{{.Pascal}}ServiceServer
    svc *service.Service
}

func (h *Handler) Create{{.Pascal}}(ctx context.Context, req *{{.Name}}v1.Create{{.Pascal}}Request) (*{{.Name}}v1.{{.Pascal}}, error) {
    return h.svc.Create{{.Pascal}}(ctx, req)
}
```

**路径也参与渲染**：`pkg/handler/{{.Name}}.go.tmpl` 渲染路径后变成 `pkg/handler/user.go`，无需在脚本里写 rename 规则。同理 `internal/service/{{.Name}}/{{.Name}}.go.tmpl` 会渲染到 `internal/service/user/user.go`，目录层级由模板路径决定。

### service 层模板（facade + 子包）

scaffold 在 `internal/service/` 下生成两个文件，对应 §2 的规则：

**`internal/service/service.go.tmpl`** — 本体 + facade 方法（每个 RPC 一行委托）：

```go
// scaffold/templates/internal/service/service.go.tmpl
package service

import (
    // ...本体 imports...
    "{{.Module}}/internal/service/{{.Name}}"
)

type Service struct {
    // ...本体字段（cfg / mgr / db / gid）...
    {{.Name}} *{{.Name}}.Service  // 子包实例
}

func New(/* ... */) (*Service, error) {
    // ...resolve db, gid...
    return &Service{
        // ...本体...
        {{.Name}}: {{.Name}}.New(db, gid),
    }, nil
}

// facade —— 一行委托到子包
func (s *Service) Create{{.Pascal}}(ctx context.Context, req *{{.Name}}v1.Create{{.Pascal}}Request) (*{{.Name}}v1.{{.Pascal}}, error) {
    return s.{{.Name}}.Create{{.Pascal}}(ctx, req)
}
```

**`internal/service/{{.Name}}/{{.Name}}.go.tmpl`** — 子包业务实现：

```go
// scaffold/templates/internal/service/{{.Name}}/{{.Name}}.go.tmpl
package {{.Name}}

type Service struct {
    db  *gorm.DB
    gid thirdcall.GIDService
}

func New(db *gorm.DB, gid thirdcall.GIDService) *Service {
    return &Service{db: db, gid: gid}
}

func (s *Service) Create{{.Pascal}}(ctx context.Context, req *{{.Name}}v1.Create{{.Pascal}}Request) (*{{.Name}}v1.{{.Pascal}}, error) {
    // 业务逻辑 + xxxToProto 合在本文件
}
```

加新 RPC 时按这个两文件模式扩展：service.go 加 facade 一行，子包主文件加业务方法。**不要**在 `internal/service/` 根目录新建 `<domain>.go` 单文件——所有领域一律走子包。

### 结构体字面量的 `{` 处理

Go 模板用 `{{ }}` 作为分隔符，遇到 Go 代码里的 `&Foo{Bar: ...}` 这种连续 `{` 会有歧义。解决方法：用 `{{"{"}}` 输出字面 `{`：

```go
// 想生成: &userv1.ListUsersResponse{Users: ...}
// 模板写: &{{.Name}}v1.List{{.Pascal}}sResponse{{"{"}}{{.Pascal}}s: ...
//         ↑                          ↑
//         普通模板变量               字面 "{" 输出，避免 {{{ 歧义
```

### 命名约束

服务名 `^[a-z][a-z0-9]*$`（小写字母数字，不允许连字符/下划线）——否则 Go 标识符/包名会炸。复杂名字（如 `user-profile`）请拆分成单字（`user` + `profile`）或用单字替代。

### 生成后必跑

```bash
cd <name>-service
make tidy        # 整理 go.sum
make proto       # 生成 gen/
make generate    # 生成 internal/store/generated/
make migrate     # 建表（需要 PostgreSQL）
make run
```

`go-common` 作为普通远程依赖，`make tidy` 会拉取并钉定版本（如需固定到某 tag，先给 go-common 打 tag，再 `go get github.com/servekit/go-common@<tag>`）。

### 修改模板的流程

1. 编辑 `scaffold/templates/*.tmpl`
2. 跑 `./scripts/new-service.sh --regen-demo` 重生成 demo-service/
3. `cd demo-service && go build ./... && golangci-lint run ./...` 确认能编译能 lint
4. 一起提交 templates/ 和 demo-service/（demo-service 的变更全是由模板重生成产生的）

demo-service/ 里**不要手改**——任何修改都会在下一次 `--regen-demo` 时被覆盖。要改东西就改 templates。

## 9. 验收检查清单

新服务搭起来后，确认：

- [ ] `go build ./...` 通过
- [ ] `golangci-lint run ./...` 无 error
- [ ] `make proto && git diff --exit-code` —— 生成结果与 committed 一致
- [ ] `make generate && git diff --exit-code` —— 同上
- [ ] grpcurl 能 CreateDemo + GetDemo 跑通
- [ ] curl HTTP gateway 也能跑通
- [ ] in-process module 测试（`pkg.NewModule`）能跑通
- [ ] 每个 RPC 在 `service.go` 都有对应 facade 方法（一一对应）
- [ ] 每个领域在 `internal/service/<domain>/` 子包，`internal/service/` 根目录只有 `service.go`，无 `<domain>.go` 单文件残留
- [ ] 周期任务都在 `internal/jobs/` + `svc.setupJobs()` 内注册，无业务子包直接持有 `*cron.Cron`
- [ ] 无 "demo" 字样残留（`grep -rn demo .`）

## 关联

- **同级 skills**：[[golang-development]]、[[gorm-cli-development]]、[[proto-development]]
- **基础库 skill**：go-common/skills/go-common-usage
- **参考实现**：本目录下的 `demo-service/`
