---
name: golang-service-scaffold
description: "Sub-document of golang-service-development skill. Loaded by SKILL.md ONLY when creating a brand-new service from scratch, regenerating demo-service (--regen-demo), or editing the scaffold templates themselves. Covers the new-service.sh one-shot workflow, template variables (Name/Pascal/Module/Plural/NameUpper), the text/template gotchas, and the regen-demo loop. NOT for iterating an existing service — that is hand-written per architecture.md."
---

# 脚手架：scaffold + templates

本文件是 `golang-service-development` 的脚手架专题。何时读本文件见入口 `SKILL.md` 的「子文档路由」表。

## 使用边界（重要）

scaffold **只用于全新项目的初始化**，不用于已有项目的迭代：

| 场景 | 用 scaffold？ | 怎么做 |
|------|--------------|--------|
| 全新服务，从零开始搭框架 | ✅ 用 | `./scripts/new-service.sh <name>` 生成骨架 |
| 已有服务，加新领域/RPC/字段 | ❌ 不用 | 按 `architecture.md` + `enum.md` 的约定**手写代码**，加在对应目录 |
| 已有服务，重命名/重构 | ❌ 不用 | 手改，scaffold 不知道你的业务变更 |
| 改 skill 模板本身（影响未来生成） | ✅ 用 `--regen-demo` | 改 `scaffold/templates/`，重生成 demo-service 验证 |

scaffold 是 **one-shot bootstrapping 工具**，不是代码维护工具。生成的代码归你所有，之后所有的演进都按 skill 文档手写——加新 RPC 就在 proto 里加 + 在 `service.go` 加 facade + 在 `internal/service/<domain>/` 子包实现业务 + handler 加委托方法（详见入口 `SKILL.md` 的「加一个 RPC」快速路径），不要回头让 scaffold 重新生成。

为什么有这条限制：scaffold 重生成是覆盖式的（`--force` 会清掉 target 再写）。如果对已有项目跑 scaffold，所有业务代码、单元测试、自定义配置都会丢。

## 总体设计

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

## 用法

```bash
# 新建服务（默认 target = dev-skills 平级目录）；不传能力 flag = 最小空壳
./skills/golang-service-development/scripts/new-service.sh <name> [capability flags] [target-parent]

# 示例
./skills/golang-service-development/scripts/new-service.sh ping                              # 最小空壳（无 DB，开箱跑）
./skills/golang-service-development/scripts/new-service.sh pay /tmp/ --db --example           # CRUD + PostgreSQL
./skills/golang-service-development/scripts/new-service.sh cache --redis                      # Redis
./skills/golang-service-development/scripts/new-service.sh order --db --redis --thirdcall     # 全家桶

# 重生成 demo-service（改完模板后跑；全开 4 flag 保 golden sample）
./skills/golang-service-development/scripts/new-service.sh --regen-demo
```

**能力 flag**（默认全 off = 最小空壳，无 Postgres 开箱跑）：

| flag | 生成 |
|---|---|
| `--db` | PostgreSQL：`store/{models,dal,generated}` + migrate 子命令 + resolveDB + compose postgres |
| `--redis` | Redis：resolveRedis + config.Redis + compose redis |
| `--thirdcall` | 第三方调用占位：`pkg/thirdcall` + `internal/thirdcall` + option/config 接线 |
| `--example` | CRUD 起点领域（`{Name}` 的 Create/Get/List/Update/Delete），隐含 `--db` |
| `--no-X` | 关闭（如 `--db --no-redis`）；多个 flag 后者胜出 |

**交互收参**：不传任何能力 flag 时——stdin 是 tty 则逐个问（默认 N）；非交互（管道 / agent 已传 flag）则最小空壳 + stderr 提示。agent 推断出答案就**直接传 flag**，别走交互。

## 模板变量

`scaffold/main.go` 的 `Spec` struct 定义了所有可用变量，模板里通过 `{{.X}}` 引用：

| 变量 | 含义 | demo 渲染为 | note 渲染为 |
|------|------|------------|------------|
| `{{.Name}}` | 小写服务名 | `demo` | `note` |
| `{{.Pascal}}` | 首字母大写 | `Demo` | `Note` |
| `{{.Module}}` | Go module 名 | `demo-service` | `note-service` |
| `{{.Plural}}` | 复数小写 | `demos` | `notes` |
| `{{.NameUpper}}` | 全大写（envPrefix、enum 值、xcodes reason） | `DEMO` | `NOTE` |

生成的新服务 `go.mod` 把 `github.com/servekit/go-common` 作为普通远程依赖（`require` + 版本，**无本地 replace**）；进新服务目录后跑 `make tidy` 解析并钉版本即可。

## 模板示例

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

## service 层模板（facade + 子包）

scaffold 在 `internal/service/` 下生成两个文件，对应 `architecture.md` §2 的规则：

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

func (s *Service) Create{{.Pascal}}(ctx context.Context, req *{{.Name}}v1.Create{{.Name}}Request) (*{{.Name}}v1.{{.Pascal}}, error) {
    // 业务逻辑 + xxxToProto 合在本文件
}
```

加新 RPC 时按这个两文件模式扩展：service.go 加 facade 一行，子包主文件加业务方法。**不要**在 `internal/service/` 根目录新建 `<domain>.go` 单文件——所有领域一律走子包。

## 结构体字面量的 `{` 处理

Go 模板用 `{{ }}` 作为分隔符，遇到 Go 代码里的 `&Foo{Bar: ...}` 这种连续 `{` 会有歧义。解决方法：用 `{{"{"}}` 输出字面 `{`：

```go
// 想生成: &userv1.ListUsersResponse{Users: ...}
// 模板写: &{{.Name}}v1.List{{.Pascal}}sResponse{{"{"}}{{.Pascal}}s: ...
//         ↑                          ↑
//         普通模板变量               字面 "{" 输出，避免 {{{ 歧义
```

## 命名约束

服务名 `^[a-z][a-z0-9]*$`（小写字母数字，不允许连字符/下划线）——否则 Go 标识符/包名会炸。复杂名字（如 `user-profile`）请拆分成单字（`user` + `profile`）或用单字替代。

## 生成后：baseline 代码怎么处理

生成的那套以服务名为名的代码是**两条独立链路**：主业务线（proto→handler→service→store，你的服务本体，**演进它**）和 thirdcall 占位线（纯 dual-mode 教学样本，**没业务调用它**，不调第三方就成套删）。三种处理场景（演进 / 删 thirdcall 占位 / 推倒重做）的完整文件清单见入口 `SKILL.md` §2.1。

## 生成后必跑

```bash
cd <name>-service
make tidy        # 整理 go.sum
make proto       # 生成 gen/
make generate    # 生成 internal/store/generated/
make migrate     # 建表（需要 PostgreSQL）
make run
```

`go-common` 作为普通远程依赖，`make tidy` 会拉取并钉定版本（如需固定到某 tag，先给 go-common 打 tag，再 `go get github.com/servekit/go-common@<tag>`）。

## 修改模板的流程

1. 编辑 `scaffold/templates/*.tmpl`
2. 跑 `./scripts/new-service.sh --regen-demo` 重生成 demo-service/
3. **regen 会清掉 `make` 产物**（`gen/`、`internal/store/generated/`）——恢复：在 skill 目录跑 `git checkout -- demo-service/gen demo-service/internal/store/generated`，或 `cd demo-service && make proto && make generate` 重生。否则第 4 步 build 会断。
4. `cd demo-service && go build ./... && golangci-lint run ./...` 确认能编译能 lint
5. 一起提交 templates/ 和 demo-service/（demo-service 的变更全是由模板重生成产生的）

demo-service/ 里**不要手改**——任何修改都会在下一次 `--regen-demo` 时被覆盖。要改东西就改 templates。

## 关联

- **入口**：`SKILL.md`（何时用、高频任务快速路径、子文档路由）
- **架构约定**：`architecture.md`（模板生成的代码遵循的分层规则）
- **同级子文档**：`enum.md`、`jobs.md`
