---
name: golang-service-architecture
description: "Sub-document of golang-service-development skill. Loaded by SKILL.md when you need architecture details: directory layout (pkg/internal/cmd/api/gen), the pkg/handler ↔ internal/service layering rule, three-mode runtime (standalone gRPC / HTTP gateway / in-process module), thirdcall inject-or-build pattern (interface internal + raw-handler injection), functional options + lifecycle.Manager resource management, project conventions, and the acceptance checklist. Read this when adding a new domain, changing startup/thirdcall/option code, or reviewing a service."
---

# Go 微服务架构

本文件是 `golang-service-development` 的**架构主文档**。何时读本文件、以及与其他子文档的关系，见入口 `SKILL.md` 的「子文档路由」表。

本节是**架构层规则**——具体 Go 风格、proto 写法、GORM 用法、go-common API 都在各自专门的 skill 里，本文件只负责把它们组装成一个完整的服务。

## 1. 目录布局

```
{service_name}-service/
├── api/proto/{svc}/v1/         # proto 定义（路径 = package 路径）
├── api/swagger/{svc}/v1/       # buf 生成的 Swagger 2.0 文档（committed，供前端/客户端消费）
├── bin/                        # 编译产物（gitignore）
├── cmd/
│   └── server/                 # 服务入口：serve（默认）+ migrate 子命令（单二进制）
├── gen/                        # buf 生成产物（committed）
├── internal/                   # 业务实现，外部不可 import
│   ├── provider/               # 辅助业务：mqtt/kafka/jobs 等
│   ├── service/                # 业务逻辑（一个领域 = 一个子包；service.go 是本体 + facade，helper.go 是资源 resolve，详见 §2）
│   ├── store/                  # DB 访问（遵循 gorm-cli-development）
│   │   ├── generated/          # gorm gen 产物
│   │   ├── models/             # 表 struct
│   │   └── dal/                # 类型安全 CRUD
│   └── thirdcall/              # 第三方调用：接口 + 实现都在这里（全 internal）
│       └── gid_service/        # 一个第三方 = 一个子目录
│           ├── gid.go          # 接口（GIDService: NextID + Close），仅包内可见
│           ├── grpc.go         # gRPC 后端（dial 是 sketch；module 模式才用真 dep）
│           └── module.go       # in-process 后端（wrap 外部注入的 raw *Handler）
├── pkg/                        # 公共能力，可作为 module 被 import
│   ├── config/                 # 配置
│   ├── handler/                # ★ proto service 的薄壳实现
│   ├── option/                 # functional options（注入 raw *Handler，非内部接口）
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

**handler 不写业务，也不做协议转换**。Service 直接接受 proto 类型，转换发生在 service 内部的 store 边界（见 §6 枚举处理，位于 `enum.md`）。

### 反模式：handler 和 service 合并到一个 struct

不要把 gRPC stub 和业务方法都挂在同一个 struct 上：

```go
// ❌ 反模式：service.go 里同时装 stub + 业务
type DemoService struct {
    demov1.UnimplementedDemoServiceServer  // embed 让 *DemoService 满足 gRPC 接口
    db *gorm.DB
    gid gid_service.GIDService
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
- `New()` 构造函数（调用 resolveDB / resolveGID 等资源注入 helper —— 这些 helper 集中在 `helper.go`，不在 service.go）
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
    gid gid_service.GIDService

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
    gid gid_service.GIDService
}

func New(db *gorm.DB, gid gid_service.GIDService) *Service {
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

**不要**给子包 struct 加 `Start`/`Stop` 方法然后从父转发——会让子包侵入 lifecycle 管理，违背"资源由父统一管"的原则。周期性触发的 cron 任务统一走 `internal/jobs/`，详见 `jobs.md`。

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

**只返回 `*Handler`**——Handler 就是对外能力，调用方不需要也不应该拿到 service handle。如果需要控制资源（DB、GID）生命周期，通过 `option.WithDB` / `option.WithGIDHandler` 注入（注入的是 raw `*Handler`，不是内部接口），让父进程拥有资源、负责清理：

```go
hdl, err := demopkg.NewModule(cfg, option.WithDB(parentDB))
if err != nil { panic(err) }
demo, err := hdl.GetDemo(ctx, &demov1.GetDemoRequest{Id: 1})
// parentDB.Close() by the parent — module never owns it
```

注意：`Service.Start()` 在 demo 里是 no-op（没有后台 goroutine），所以 in-process 用户不需要调 Start。如果未来 service 加了 cron / 消费者，再考虑把 Start/Stop 也提升到 Handler 上。

## 4. Thirdcall 双层模式

## 4. Thirdcall：接口 internal + 注入或自建

第三方依赖（如 gid-service）的接入采用**接口内置 + 注入或自建**模式。canonical 实现是 `user-service`，照抄即可。

### 接口 + 两个后端，都在 `internal/thirdcall/<name>/`

接口**不**放 `pkg/`（没有 `pkg/thirdcall/`）——它是实现细节，对外只暴露 raw `*Handler`（见下方 option）。整个三方依赖的接口 + 实现都收在 `internal/thirdcall/gid_service/`：

```go
// internal/thirdcall/gid_service/gid.go
package gid_service

type GIDService interface {
    NextID(ctx context.Context) (int64, error)
    Close() error  // grpc→client.Close()；module→no-op（Handler 由 mgr.Add 管 Start/Stop）；resolveGID 接 lifecycle
}
```

```go
// internal/thirdcall/gid_service/module.go —— wrap 一个已建好的 raw Handler
func NewModule(h *gidservice.Handler) GIDService { return &moduleGID{Handler: h} }
func (m *moduleGID) Close() error { return nil }  // no-op：Handler 生命周期由 mgr.Add 管，见 resolveGID

// internal/thirdcall/gid_service/grpc.go —— dial 远端（scaffold 里 dial 是 sketch）
func NewGRPC(target string) (GIDService, error) { ... }
func (g *grpcGID) Close() error { return g.client.Close() }
```

关键：`NewModule` / `NewGRPC` 只 **wrap**，不 **build**。建 raw Handler（或从父进程透传一个）是 service 根的活（`resolveGID`），不是这个包的。

### option 注入 raw `*Handler`，不是接口

`pkg/option` 暴露的是 gid-service 的 raw `*gidservice.Handler`，调用方不需要知道本服务的 `GIDService` 接口：

```go
type Options struct {
    DB         *gorm.DB
    GIDHandler *gidservice.Handler  // raw handler；service 内部 wrap 成 GIDService
}
func WithGIDHandler(h *gidservice.Handler) Option { ... }
```

### resolveGID：注入或自建（在 `helper.go`）

`internal/service/helper.go` 的 `resolveGID` 是接线的核心 —— grpc / module / 注入三分支：grpc 注册 Stopper（关连接）；module 自建把 raw Handler 注册成 `mgr.Add`（管它的 Start/Stop）；注入不注册：

```go
func resolveGID(o *option.Options, cfg *config.RemoteServiceConfig[*gidconfig.Config], mgr *lifecycle.Manager) (gid_service.GIDService, error) {
    switch cfg.Mode {
    case "grpc":
        gid, err := gid_service.NewGRPC(cfg.Target)        // 自建连接 → own
        mgr.AddStopper("gid", lifecycle.StopFunc(func() { _ = gid.Close() }))
        return gid, nil
    case "module":
        if o.GIDHandler != nil {
            return gid_service.NewModule(o.GIDHandler), nil  // 父进程注入 → 不 own，不注册
        }
        hdl, err := gidservice.NewModule(cfg.Config)         // 自建 Handler → own
        gid := gid_service.NewModule(hdl)
        mgr.Add("gid", hdl)  // raw Handler 实现 lifecycle.Service，mgr 管它的 Start/Stop
        return gid, nil
    }
}
```

- **grpc** → 注册 Stopper，`mgr.Stop` 关连接
- **module+自建** → raw Handler 注册成 `mgr.Add`，`mgr` 管它的 Start/Stop（不再走 Stopper，所以 Handler 内部的 cron/consumer 也会真正 Start）
- **module+注入**（`WithGIDHandler`）→ 不注册，父进程 own 生命周期

### 关键约束

- `internal/thirdcall/<name>/` 是**唯一**允许 import 第三方（`github.com/servekit/gid-service/...`）的地方；service / handler 只依赖内部的 `gid_service.GIDService` 接口
- **不存在 `pkg/thirdcall/`**；option 注入的是 raw `*Handler`，不是本服务接口
- `New{Module,GRPC}` 只 wrap；build raw Handler 在 `resolveGID`（module+自建分支）或由父进程完成（注入分支）
- config：`ThirdParty.GID *RemoteServiceConfig[*gidconfig.Config]`（泛型 `[T]` 复用，加新三方一行）

## 5. Option + lifecycle.Manager 资源管理

`pkg/option/option.go` 定义 functional options：

```go
type Option func(*Options)

type Options struct {
    DB         *gorm.DB
    GIDHandler *gidservice.Handler  // raw handler（不是内部 GIDService 接口）
}

func WithDB(db *gorm.DB) Option { return func(o *Options) { o.DB = db } }
func WithGIDHandler(h *gidservice.Handler) Option { ... }
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
    gid gid_service.GIDService
}

func New(cfg *config.Config, opts ...option.Option) (*Service, error) {
    o := option.Apply(opts...)
    mgr := lifecycle.NewManager()

    db, err := resolveDB(&o, cfg, mgr)        // 注入 → 不注册；自建 → 注册为 Stopper
    if err != nil {
        return nil, errors.Join(err, mgr.Stop())  // 已注册的回滚
    }
    // resolveGID 在 helper.go：grpc / module+自建 → own（注册 Stopper）；
    // module+注入(WithGIDHandler) → 父进程 own，不注册。详见 §4。
    gid, err := resolveGID(&o, cfg.ThirdParty.GID, mgr)
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
// resolveDB 里（helper.go）：Close 用 lifecycle.StopFunc；cleanup error 按 CLAUDE.md
// 例外用 _ = 丢弃（资源清理 Close 允许 _ =，无 actionable 价值就不造类型 preserve）
mgr.AddStopper("db", lifecycle.StopFunc(func() {
    if sqlDB, e := db.DB(); e == nil && sqlDB != nil {
        _ = sqlDB.Close()
    }
}))

// resolveGID 里：grpc 分支用 StopFunc 包 GIDService.Close（接口自带 Close，无需类型断言）；
// module 分支不同——raw *Handler 自带 Start+Stop，直接 mgr.Add("gid", hdl) 注册成 lifecycle.Service
mgr.AddStopper("gid", lifecycle.StopFunc(func() { _ = gid.Close() }))  // grpc 分支
```

**关于 cleanup error**：CLAUDE.md 允许资源清理（`Close()` 等）用 `_ =` 忽略 error。Stopper 注册到 lifecycle.Manager 后，Stop 时由 Manager 调用；cleanup path 的 close 失败基本无 actionable 价值，直接 `_ =` 即可（scaffold 和 user-service 都是这个风格）。要保留可见性也可改 `slog.Warn`，但不要在 service 里造 closer 类型去 preserve error。

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
| in-process 下游 Handler（thirdcall module 自建）| `mgr.Add("gid", hdl)` | raw Handler 自带 Start+Stop，注册成 lifecycle.Service（区别于上面 close-only 资源）|
| Redis client | `mgr.AddStopper("redis", lifecycle.StopFunc(...))` | 同上 |
| Cron (周期任务) | `mgr.Add("jobs", scheduler)` via `svc.setupJobs()` | 详见 `jobs.md` |
| Kafka consumer | `mgr.Add("consumer", &consumerComponent{...})` | Start 启动 goroutine；Stop 停 + 等 in-flight |
| MQTT client | `mgr.Add("mqtt", &mqttComponent{...})` | 同上 |

注意 lifecycle.Manager 的语义：**Start 并发**调用（每个 Service 在自己 goroutine 里跑 Start，所以 blocking Starts 不会互相阻塞），**Stop 反序串行**。如果某个组件的 Start 必须等另一个先 ready，把它放在 mgr 里更后面注册是不够的——需要显式同步，或者拆成两步（构造时建立连接，Start 时启动 goroutine）。

> **后台 cron jobs**（`internal/jobs/` + `svc.setupJobs()`）是独立主题，详见 `jobs.md`。本节只覆盖 close-only 资源（DB / gRPC / Redis）和 consumer 类组件的注册。

## 6. 项目级约定

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

`buf.gen.yaml`：v2 + managed mode + `go_package_prefix: <svc>-service/gen`，四个插件：`protocolbuffers/go`、`grpc/go`、`grpc-ecosystem/gateway`（gRPC↔HTTP 网关 stub，输出 `gen/`）、`grpc-ecosystem/openapiv2`（从 `google.api.http` 注解派生 **Swagger 2.0 文档**，输出 `api/swagger/`，供 Swagger UI / Redoc / 前端客户端代码生成器消费）。openapiv2 是 remote plugin，**不进 `buf.lock`**（`buf.lock` 只记 proto module 依赖）；改完 `buf.gen.yaml` 跑 `make proto` 即生成。无 `google.api.http` 注解的 RPC 默认不出现在 swagger 里。

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

## 7. 验收检查清单

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

> 注意：日常**迭代**（加一个 RPC / 加字段）不需要跑完整个验收清单，用入口 `SKILL.md` 的「最小验证闭环」即可。本清单是**新建服务**后的完整验收。

## 关联

- **入口**：`SKILL.md`（何时用、高频任务快速路径、子文档路由）
- **同级子文档**：`enum.md`（枚举处理）、`jobs.md`（后台 cron）、`scaffold.md`（脚手架）
- **同级 skills**：`golang-development`、`gorm-cli-development`、`proto-development`
- **参考实现**：本目录下的 `demo-service/`
