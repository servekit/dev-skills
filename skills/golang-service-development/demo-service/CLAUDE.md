# CLAUDE.md — demo-service

## 项目定位

Demo service — dev-skills 的 golang-service-development skill 自带的参考实现。
完整跑通 grpc-gateway 服务三层结构（handler / service / store）、thirdcall、option、生命周期。

**这是 skill 的"golden sample"**：脚本 `./scripts/new-service.sh <name>` 拷贝此目录生成新服务。
修改任何结构请同步更新 `skills/golang-service-development/SKILL.md`。

## 技术栈约定

### gRPC / Proto
- Proto 在 `api/proto/demo/v1/`
- buf v2 配置在 `buf.yaml` / `buf.gen.yaml`
- 生成代码到 `gen/`（committed），由 `make proto` 生成

### 数据库 / GORM
- PostgreSQL（通过 `dbx.New`）
- `internal/store/{models,generated,dal}` 遵循 `gorm-cli-development` skill
- 迁移：`cmd/migrate/` + GORM AutoMigrate

### 错误处理
- 错误码在 `pkg/xcodes/demo.go`，按域分文件
- 业务错误用 `xcodes.ErrDemoXxx.Wrap(err)` / `.New()` 包装

### 基础库
- 全部 `github.com/servekit/go-common/*`，参考 `go-common/skills/go-common-usage`

## 三种运行模式

1. **standalone gRPC**: `make run` → listen :9000
2. **HTTP gateway**: 同上自动启用，:8080（除非 `server.http_addr` 为空）
3. **in-process module**: 其它服务 `import "demo-service/pkg"` → `pkg.NewModule(cfg, opts...)`

## 常用命令

```bash
make proto       # buf generate
make generate    # gorm gen
make migrate     # AutoMigrate
make run         # 启动服务
make test        # 跑测试
make lint        # golangci-lint
```
