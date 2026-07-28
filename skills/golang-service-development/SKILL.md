---
name: golang-service-development
description: "MUST use when creating, scaffolding, OR evolving a Go service on the go-common stack (grpcx / configx / lifecycle) — adding an RPC, a domain, a field, or a thirdcall to an EXISTING service counts just as much as scaffolding a new one. For iteration on an existing service, follow this entry's quick-path (hand-written, ~4 files); do NOT re-run the scaffold. Read SKILL.md first; it routes to sub-documents (architecture / enum / jobs / scaffold) loaded on demand. The servekit naming convention is *-service, but the trigger is 'it's a service', not the suffix: a project named pay, order, or userapi that serves an API also qualifies. Do NOT wait for go.mod to exist: at creation time it does not exist yet, and the scaffold generates it. Also use on an existing service whose go.mod contains github.com/servekit/go-common. Trigger keywords: create/scaffold a Go service, new microservice, grpc backend, grpc-gateway service, -service project, pay/order/*-api backend, add RPC, add endpoint, add API, pkg/handler, internal/service, thirdcall pattern, cmd/server, in-process module, lifecycle.Manager, proto enum to DB int."
---

# Go 微服务开发指南（入口）

本文件是 `golang-service-development` skill set 的**唯一入口**。**先读本文件**，按需加载子文档（`architecture.md` / `enum.md` / `jobs.md` / `scaffold.md`）。子文档不在自动索引里，只有本文件指引你读时才读——别一开始就把它们全读了。

这是**架构层规则**——具体 Go 风格、proto 写法、GORM 用法、go-common API 都在各自专门的 skill 里，本 skill 只负责把它们组装成一个完整的服务。

## 0. 第一步：判断任务类型（决定走哪条路径）

90% 的日常需求是**在已有服务上加接口/字段**，不是从零搭服务。先判断：

| 你的任务 | 路径 | 加载 |
|---------|------|------|
| **在已有服务加 RPC / 加字段 / 加领域**（go.mod 已存在） | §1 快速路径，**手写**，~4 个文件 | 本文件够了；涉及枚举读 `enum.md` |
| **从零新建一个 `-service`**（go.mod 还不存在） | §2 一条命令 scaffold | 本文件 + `scaffold.md` |

**最常见的错误**：把"加一个接口"当成"搭一个服务"，去跑 scaffold 或通读全部架构文档——结果 30 分钟。加接口走 §1，5 分钟。

---

## 1. 快速路径：加一个 RPC（已有服务，最高频）

以在 `demo` 服务的 `demo` 领域上加 `UpdateDemo` 为例。**只改下面这几个文件，其他一律别碰**：

### 第 1 步 · proto 加方法 → `make proto`

编辑 `api/proto/demo/v1/demo.proto`：在 `service DemoService { }` 加 `rpc UpdateDemo(UpdateDemoRequest) returns (Demo);`，并定义 `UpdateDemoRequest`。然后：

```bash
make proto     # 重新生成 gen/（gen/ 是产物，别手改）
```

### 第 2 步 · handler 加一行委托

编辑 `pkg/handler/demo.go`：

```go
func (h *Handler) UpdateDemo(ctx context.Context, req *demov1.UpdateDemoRequest) (*demov1.Demo, error) {
    return h.svc.UpdateDemo(ctx, req)
}
```

### 第 3 步 · service.go 加一行 facade

编辑 `internal/service/service.go`（只加一行委托到子包）：

```go
func (s *Service) UpdateDemo(ctx context.Context, req *demov1.UpdateDemoRequest) (*demov1.Demo, error) {
    return s.demo.UpdateDemo(ctx, req)
}
```

### 第 4 步 · 子包加业务方法

编辑 `internal/service/demo/demo.go`（业务方法 + `xxxToProto` 合本文件）：

```go
func (s *Service) UpdateDemo(ctx context.Context, req *demov1.UpdateDemoRequest) (*demov1.Demo, error) {
    // 业务逻辑；enum 在 proto 定义、DB 存 int32，转换用 int32(req.GetStatus()) / demov1.DemoStatus(x)，别写 helper（见 enum.md）
    return demoToProto(record), nil
}
```

**涉及 DB 时**还要改（否则跳过）：`internal/store/models/demo.go`（加字段）→ `internal/store/dal/demo.go`（加查询）→ `make generate`。新错误码加到 `pkg/xcodes/demo.go`。

### 验证（最小闭环，别上全套）

```bash
make proto && make generate && go build ./...
```

通过即可。**不要**一上来就 `make run` + grpcurl + curl——那是新建服务验收用的（见 `architecture.md` §7），日常迭代用上面三件套足够。

### 加 RPC 时读什么、不读什么

**常规加 RPC**（上面四步）只动那 4 个文件，下列**一般不用读**（读了容易绕远、浪费时间）：

- `gen/`（自动生成，改 proto 后 `make proto` 重生）
- `cmd/`、`pkg/config/`、`pkg/option/`、`buf*.yaml`（加 RPC 不用动）
- skill 仓库的 `scaffold/`、`demo-service/`（示例，不在你的服务里）

**但调试启动 / 网关 / 拦截器 / 端口这类问题时，反过来——先读这两处框架代码，通常一眼定位**：

- `pkg/server.go` 的 `grpcx.New(...)`：网关启停、拦截器、端口都在这里。比如 HTTP gateway 没监听，看 `registerGW` / `GatewayAddr` 那几行，别先去猜配置 / env / godotenv。
- `internal/service/service.go` 的 `New()`：资源 resolve、生命周期。

`pkg/handler` 和 `internal/service/<domain>/` 是你的主战场，随便读。

---

## 2. 快速路径：新建一个服务（从零）

**先从需求推断 4 个能力开关**（DB / Redis / thirdcall / example），能定就传 flag、不问用户；推断不出且在 tty 才让脚本交互问。

| 需求信号 | flag |
|---|---|
| 持久化 / CRUD / 记录 | `--db`（PostgreSQL） |
| 缓存 / 锁 / 限流 | `--redis` |
| 调别的服务 | `--thirdcall` |
| 要 CRUD 起点（隐含 `--db`） | `--example` |
| 都不需要（健康检查 / 转发 / 计算） | 不传 = 最小空壳，无 Postgres 开箱跑 |

```bash
./skills/golang-service-development/scripts/new-service.sh <name> [flag...]   # ping / pay --db --example / cache --redis
```

不传 flag：tty 交互问 4 个；非交互（管道 / agent 已传 flag）= 最小空壳 + stderr 提示。

生成后进目录跑（按能力）：

```bash
cd <name>-service
make tidy && make proto
make run                       # 最小空壳无需 Postgres
# 有 --db 时还要：make generate && make migrate
```

命名约束：`<name>` 必须匹配 `^[a-z][a-z0-9]*$`（小写字母数字，无连字符/下划线）。详细用法、模板变量、改模板流程见 `scaffold.md`。

> scaffold 是 **one-shot**：生成一次后归你所有，之后所有演进（加 RPC / 加领域）都走 §1 手写，**绝不**对已有服务重跑 `new-service.sh`（`--force` 会清空 target，业务代码全丢）。

### 生成后的示例代码

scaffold 生成的那套以服务名为名的代码（demo-service 里满眼的 `Demo` / `DemoService`）是**示例样板**——展示各层写法，**无业务意义**。完整样板在 skill 的 `demo-service/`（golden sample，随时对照，别去改它）。

**删或留，任选**：

- **留**（省事）：示例的 CRUD 结构几乎总能改成你的第一个领域——改 proto 字段 + 业务逻辑即可，`server.go` / `client.go` / `module.go` 不用动。
- **删**：先 `make run` + grpcurl 验证骨架通了，再**成套删**（下表），从空壳按 §1/§3 写自己的业务。

| 示例 | 要删的文件 |
|------|-----------|
| 主业务 | `api/proto/<name>/`（service + message）· `pkg/handler/<name>.go` · `internal/service/<name>/` · `internal/store/{models,dal,generated}/<name>*` · `pkg/xcodes/<name>.go` · `store/models/register.go` 的注册行 |
| thirdcall 占位（死代码，没业务调用） | `pkg/thirdcall/<name>*.go` · `internal/thirdcall/<name>*/` · `option.go`（`<name>Service` 字段 + `With<name>()`）· `service.go`（字段 + `resolve<name>()` + New 调用）· `config.go`（`<name>` 配置）· `config.example.yaml` + `.env.example`（`third_party.<name>` 段） |
| 框架引用（删主业务时同步） | `server.go`（`Register<name>` + Handler）· `client.go`（embed + `New<name>Client`）· `module.go`（编译断言） |

删主业务后，proto 至少留一个空 `service <Name>Service {}` 才能编译。**示例不是生成产物，删了不能 `make` / scaffold 重生**。

---

## 3. 快速路径：加一个新领域

1. proto：在现有 service 加 RPC（或新 service）→ `make proto`
2. 新建子包 `internal/service/<newdomain>/<newdomain>.go`：`type Service struct{}` + `New(db, gid)` + 业务方法 + `xxxToProto`
3. `internal/service/service.go`：加子包字段 + `New` 里 `xxx.New(db, gid)` + facade 方法
4. `pkg/handler/<svc>.go`：加 handler 委托方法
5. 涉及 DB：`internal/store/models/<newdomain>.go` + `register.go` 注册 + `dal/` → `make generate`
6. 错误码：`pkg/xcodes/<newdomain>.go`

领域子包的完整规则（为什么强制子包、service.go 放什么、后台 goroutine 怎么处理）见 `architecture.md` §2。

---

## 4. 文件分类：哪些能改、哪些能删重生、哪些是示例

在你自己的服务里，文件分四类。**先认清类别，再动手**——这决定你改不改、敢不敢删：

| 类别 | 位置 | 怎么对待 |
|------|------|---------|
| **① 业务代码（你的主战场）** | `api/proto/` · `pkg/handler/` · `pkg/xcodes/` · `internal/service/` · `internal/store/{models,dal}` | 加 RPC / 领域就改这里。**含 scaffold 生成的那套以服务名为名的 baseline**（Create/Get 等）——它是你的起点，演进或删见 §2.1 |
| **② 框架代码（基本不动）** | `cmd/server/` · `pkg/{server,module,client,config,option}.go` · `internal/jobs/` · `buf*.yaml` · `Makefile` · `.golangci.yml` | scaffold 生成。加 RPC 时**不用碰**；改启动 / 加定时任务时才动 |
| **②′ thirdcall 占位（可删）** | `pkg/thirdcall/<name>.go` · `internal/thirdcall/<name>/` · `option.go` 的 `<name>Service` 字段 · `service.go` 的 `resolve<Name>` · config 里的 `<name>` 段 | dual-mode 教学**样本**，**没业务调用它**。不调第三方就成套删；要调就照抄改真实（§2.1） |
| **③ 生成产物（可删重生）** | `gen/` · `api/swagger/` · `internal/store/generated/` | `make proto` / `make generate` 产出。**永远别手改**；改了 proto/model 后重跑生成覆盖即可，删了能重生 |
| **④ 示例（不是你的代码）** | skill 仓库里的 `demo-service/` · `scaffold/` | 只在 skill 仓库存在，**不在你的服务里**。是参考实现 / 模板源，加接口时**别去读** |

> **① ② ②′ 是手写起点，不是 ③ 那种能 `make` 重生的产物**——baseline（含 thirdcall 占位）删了找不回。

---

## 5. 提效铁律（避免 30 分钟变 5 分钟）

1. **先判规模再动手**：go.mod 已存在 → 已有服务 → §1 手写快速路径；go.mod 不存在 → 新建 → §2 scaffold。别把加接口当搭服务。
2. **scaffold 是 one-shot**：已有服务绝不重跑 `new-service.sh`。
3. **最小验证闭环**：加完 RPC 跑 `make proto && make generate && go build ./...`，别一上来 `make run` + grpcurl + curl 全套。
4. **按阶段读文件**：常规加 RPC 只读 §1 列的那几个，`gen/`、`scaffold/`、`demo-service/`、`cmd/`、`pkg/config`、`pkg/option` 一般不用读；但**调试启动 / 网关 / 端口问题**时先读 `pkg/server.go`（`grpcx.New`）和 `internal/service/service.go`（`New`）——通常一眼定位，别先猜配置。
5. **枚举优先 proto**：枚举在 proto 定义、DB 存 int32，用 proto 内置方法转换（`int32(x)` / `demov1.DemoStatus(x)`），**不要写 helper**（详见 `enum.md`）。

---

## 6. 子文档路由（按需加载）

只在本文件不够时才读对应子文档：

| 任务 | 读 |
|------|-----|
| 理解分层 / 目录布局 / 启动三件套 / thirdcall 双层 / option+lifecycle / 项目约定 / 新建服务验收清单 | `architecture.md` |
| proto enum ↔ DB int 转换（几乎所有涉及枚举的场景） | `enum.md` |
| 加后台定时任务（cron / 周期清理 / 心跳） | `jobs.md` |
| 新建服务的详细用法 / 模板变量 / 改模板 / `--regen-demo` | `scaffold.md` |

---

## 7. 何时不用本 skill（交给别的 skill）

| 任务 | Skill |
|------|-------|
| Go 命名 / 错误处理 / 并发 / 测试 / lint | golang-development |
| 写 `.proto`、配 buf、protovalidate | proto-development |
| `store/{models,generated,dal}`、gorm gen、类型安全 CRUD | gorm-cli-development |
| `configx` / `redisx` / `dbx` / `xerr` / `grpcx` 等基础库 API | go-common 仓库 README（不在 dev-skills 范围） |

## 关联

- **子文档**：`architecture.md`（架构全貌）、`enum.md`（枚举）、`jobs.md`（cron）、`scaffold.md`（脚手架）
- **同级 skills**：`golang-development`、`gorm-cli-development`、`proto-development`
- **参考实现**：本目录下的 `demo-service/`（示例，仅在需要对照结构时读）
