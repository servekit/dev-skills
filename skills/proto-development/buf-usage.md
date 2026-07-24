---
name: buf-usage
description: "Sub-document of proto-development skill. Loaded by SKILL.md when configuring buf.yaml, buf.gen.yaml, running buf CLI commands, code generation, managed mode, lint, breaking check, format, or proto CI. Covers installation, modules/workspaces, remote plugins, managed mode, lint rules, breaking check, GitHub Actions."
---

# Buf 工具使用指南

基于 [Buf 官方文档](https://buf.build/docs) 编写。

## 1. 安装 buf

当前版本：**1.68.4**

```bash
# macOS / Linux (推荐)
brew install bufbuild/buf/buf

# Go install（官方不推荐 tools.go 方式）
GOBIN=/usr/local/bin go install github.com/bufbuild/buf/cmd/buf@v1.68.4

# npm
npm install @bufbuild/buf

# Docker（不包含 protoc 和本地插件，remote 插件无需额外配置）
docker run --volume "$(pwd):/workspace" --workdir /workspace bufbuild/buf lint

# Windows
winget install bufbuild.buf
```

验证：`buf --version`（protovalidate 需要 >= 1.54.0）

## 2. 模块与工作区

### 核心概念

- **模块 (Module)**：一个目录树下的 `.proto` 文件集合，是 buf 构建、版本化、发布的基本单元。每个模块对应 BSR 上的一个仓库。模块内 import 相对于模块根目录解析。
- **工作区 (Workspace)**：一个或多个模块的集合，由单个 `buf.yaml` 配置。共享 lint/breaking 默认规则和外部依赖。工作区内模块可以互相 import，无需显式声明依赖。
- 即使只有一个模块，它也在一个工作区中。

### 初始化

```bash
buf config init    # 在当前目录创建 buf.yaml（v2 格式）
```

生成的默认配置：
```yaml
version: v2
lint:
  use:
    - STANDARD
breaking:
  use:
    - FILE
```

### 单模块（proto 在子目录）

最常见的情况。proto 文件在 `proto/` 子目录下，buf 配置在项目根目录：

```
project/
├── buf.yaml
├── buf.gen.yaml
├── buf.lock
├── proto/                    # 模块
│   └── acme/pet/v1/
│       ├── pet.proto
│       └── pet_service.proto
└── gen/                      # 生成代码
    └── go/
```

buf.yaml：
```yaml
version: v2
modules:
  - path: proto               # proto 文件所在目录
```

### 单模块（proto 在根目录）

简单项目，proto 直接在项目根目录：

```
project/
├── buf.yaml
├── buf.gen.yaml
├── acme/pet/v1/
│   └── pet.proto
└── gen/
```

buf.yaml 可以省略 `modules`（默认 path 为 `.`）：
```yaml
version: v2
# modules 未指定时默认 path: .
```

或使用顶层简写（适用于单模块）：
```yaml
version: v2
name: buf.build/acme/petapis   # 可选，推送 BSR 时需要
lint:
  use:
    - STANDARD
breaking:
  use:
    - FILE
deps:
  - buf.build/googleapis/googleapis
```

### 多模块工作区

大型项目，多个服务独立版本化：

```
workspace_root/
├── buf.yaml
├── buf.lock
├── buf.gen.yaml
├── proto/
│   └── acme/weather/v1/
│       └── api.proto
└── vendor/
    └── units/v1/
        └── metric.proto
```

buf.yaml：
```yaml
version: v2
modules:
  - path: proto
    name: buf.build/acme/weatherapi
  - path: vendor
    name: buf.build/acme/units
    lint:
      use:
        - MINIMAL              # 模块级覆盖，完全替换工作区默认值（不合并）
    breaking:
      use:
        - PACKAGE
deps:
  - buf.build/googleapis/googleapis
lint:
  use:
    - STANDARD                # 工作区默认，被模块覆盖的不生效
breaking:
  use:
    - FILE                    # 工作区默认
```

**重要规则：**
- 工作区内每个 `.proto` 文件的路径（相对于模块根目录）必须唯一
- 模块级 lint/breaking 设置**完全替换**工作区默认值，不合并
- `deps` 中不需要声明工作区内模块间的依赖，buf 自动解析

### 依赖管理

```bash
buf dep update    # 更新依赖，生成/更新 buf.lock
buf dep prune     # 清理未使用的依赖
```

常用 BSR 依赖：

| 依赖 | 用途 |
|------|------|
| `buf.build/googleapis/googleapis` | `google/api/annotations.proto`、HTTP 注解 |
| `buf.build/bufbuild/protovalidate` | `buf/validate/validate.proto` 字段验证 |

`google/protobuf/` 下的 Well-Known Types（`timestamp`、`empty`、`any`、`duration` 等）内置支持，无需声明依赖。

## 3. 代码生成

### buf.gen.yaml 配置

buf.gen.yaml 是独立的配置文件，和 buf.yaml 在同一目录（项目根目录）。它定义输入、插件和输出位置。

**Go + gRPC + grpc-gateway + protovalidate 的完整配置：**

```yaml
version: v2
clean: true
inputs:
  - directory: proto           # 输入：proto 目录（必须指定，否则 generate 找不到文件）
managed:
  enabled: true
  override:
    - file_option: go_package_prefix
      value: github.com/acme/pet-store/gen/go
  disable:
    - file_option: go_package
      module: buf.build/bufbuild/protovalidate
    - file_option: go_package
      module: buf.build/googleapis/googleapis
plugins:
  - remote: buf.build/protocolbuffers/go
    out: gen/go
    opt: paths=source_relative
  - remote: buf.build/grpc/go
    out: gen/go
    opt: paths=source_relative
  - remote: buf.build/grpc-ecosystem/gateway
    out: gen/go
    opt: paths=source_relative
  - remote: buf.build/bufbuild/validate-go
    out: gen/go
    opt: paths=source_relative
```

**关键字段说明：**

| 字段 | 说明 |
|------|------|
| `clean: true` | 每次生成前清空输出目录 |
| `inputs` | **必须指定**。`directory` 相对于 buf.gen.yaml 所在目录 |
| `managed` | managed mode 配置（见下节） |
| `plugins[].remote` | BSR 远程插件，零本地安装 |
| `plugins[].local` | 本地安装的插件（需要在 $PATH 中） |
| `plugins[].out` | 输出目录，相对于 buf.gen.yaml 所在目录 |
| `plugins[].opt` | 插件选项 |

### Managed Mode

Managed mode 自动设置语言相关的 file option（如 `go_package`、`java_package`），让 proto 文件保持干净，不需要硬编码语言选项。

**Go 语言的 managed mode：**
- Go 没有默认的 `go_package`，**必须**在 `override` 中设置 `go_package_prefix` 或 `go_package`
- `go_package_prefix`：`<prefix>/<proto_package_path>`，配合 `paths=source_relative` 使用时目录结构清晰
- `go_package`：直接设置固定值

**`disable` 规则：**
- 必须对第三方模块（googleapis、protovalidate）disable `go_package`，否则生成代码编译失败
- `disable` 优先于 `override`
- 可按 `module`、`path`、`file_option` 精确控制

**disable 与 override 优先级：**
- 如果同一个选项同时有 `disable` 和 `override`，`disable` 优先
- 多个 `override` 规则修改同一选项时，**最后指定的生效**

### Managed Mode 调试

如果生成代码的 `go_package` 或其他语言选项不符合预期，用以下命令检查 buf 实际生成的描述符：

```bash
buf build --as-file-descriptor-set -o debug.json
```

检查输出中对应 `.proto` 文件的 `options` 字段，确认 managed mode 是否正确设置了 `go_package` 等选项。常见问题：

| 症状 | 可能原因 |
|------|---------|
| go_package 为空 | 缺少 `override` 中的 `go_package_prefix` |
| 第三方模块 go_package 不对 | 缺少对应的 `disable` 规则 |
| override 没生效 | 检查 `disable` 是否意外覆盖了 `override` |

**完整 disable 示例：**
```yaml
managed:
  enabled: true
  disable:
    # 完全不对 googleapis 做 managed mode
    - module: buf.build/googleapis/googleapis
    # 不修改 csharp_namespace（任何文件）
    - file_option: csharp_namespace
    # 只对特定路径的特定选项 disable
    - module: buf.build/acme/weather
      path: weather/v1beta1/
      file_option: java_package
```

### 插件

| 插件 | 生成文件 | 何时使用 |
|------|---------|---------|
| `buf.build/protocolbuffers/go` | `.pb.go` | 始终 |
| `buf.build/grpc/go` | `_grpc.pb.go` | 用 gRPC 时 |
| `buf.build/grpc-ecosystem/gateway` | `.pb.gw.go` | 用 grpc-gateway 时 |
| `buf.build/bufbuild/validate-go` | `.pb.validate.go` | 用 protovalidate 时 |
| `buf.build/grpc-ecosystem/openapiv2` | `.swagger.json` | 需要 OpenAPI 时 |

**本地插件 vs 远程插件：**
- 远程插件：`remote: buf.build/protocolbuffers/go`，buf 从 BSR 拉取执行，零本地安装
- 本地插件：`local: protoc-gen-go`，需要在 `$PATH` 中安装
- 远程插件可锁版本：`remote: buf.build/protocolbuffers/go:v1.36.11`（BSR 自己的版本 scheme）
- 不锁版本则用最新

> **CI/容器环境警告：** 在 CI pipeline 或 Docker 容器中，**强烈推荐远程插件**。本地插件需要额外安装步骤（`go install` 或下载二进制），增加构建时间和维护成本，且容易因版本不一致导致 CI 与本地生成结果不同。远程插件由 BSR 分发，buf 自动处理版本和缓存，零配置。

### 运行 buf generate

```bash
# 在含 buf.gen.yaml 的目录下运行
buf generate

# 指定配置文件
buf generate --template buf.gen.go.yaml
buf generate --template templates/gen.yaml

# 只生成特定路径
buf generate --path proto/acme/pet --path proto/acme/order

# 排除特定路径
buf generate --exclude-path proto/acme/internal

# 生成到指定目录（在 out 前加前缀）
buf generate -o output/

# 从 BSR 模块生成
buf generate buf.build/acme/petapis

# 从 GitHub 仓库生成
buf generate https://github.com/acme/petapis.git

# 无需配置文件，直接命令行指定
buf generate --template '{"version":"v2","plugins":[{"protoc_builtin":"go","out":"gen/go"}]}'

# 生成时包含依赖和 Well-Known Types
buf generate --include-imports --include-wkt
```

**关键：** `buf generate` 必须在含 buf.gen.yaml 的目录下运行（或用 `--template` 指定路径）。

### OpenAPI 单独配置

创建 `buf.gen.openapi.yaml`：
```yaml
version: v2
clean: true
inputs:
  - directory: proto
managed:
  enabled: true
  override:
    - file_option: go_package_prefix
      value: github.com/acme/pet-store/gen/go
  disable:
    - file_option: go_package
      module: buf.build/bufbuild/protovalidate
    - file_option: go_package
      module: buf.build/googleapis/googleapis
plugins:
  - remote: buf.build/grpc-ecosystem/openapiv2
    out: gen/swagger
```

只对有 HTTP 注解的服务生成：
```bash
buf generate --template buf.gen.openapi.yaml --path proto/acme/pet
```

### Go 模块依赖

生成代码后，Go 依赖由 `go mod tidy` 自动解析：

```bash
go mod init github.com/acme/pet-store
go mod tidy    # 自动解析 buf.build Go 模块版本
go build ./gen/go/...
```

**不要手动写 buf.build Go 模块版本号**，`go mod tidy` 会自动从 BSR 解析正确的版本。

## 4. Lint 规则

### 规则类别

| 类别 | 严格程度 | 说明 |
|------|---------|------|
| `MINIMAL` | 最低 | 目录结构与 package 匹配、package 定义、无循环依赖 |
| `BASIC` | 中低 | MINIMAL + 标准命名风格（snake_case 字段、PascalCase 消息等） |
| `STANDARD` | 推荐 | BASIC + 枚举前缀、版本后缀、Request/Response 唯一、Service 后缀、protovalidate 验证 |
| `COMMENTS` | 额外 | 强制各种元素有注释 |
| `UNARY_RPC` | 额外 | 禁止 streaming RPC |

**推荐始终使用 `STANDARD`。**

### STANDARD 新增的关键规则

| 规则 | 说明 |
|------|------|
| `ENUM_VALUE_PREFIX` | 枚举值必须以枚举名作前缀：`PET_TYPE_DOG` |
| `ENUM_ZERO_VALUE_SUFFIX` | 枚举零值后缀：`_UNSPECIFIED`（可配置） |
| `FILE_LOWER_SNAKE_CASE` | 文件名必须是 lower_snake_case |
| `PACKAGE_VERSION_SUFFIX` | package 最后一段必须是版本：`v1`、`v1alpha` |
| `RPC_REQUEST_RESPONSE_UNIQUE` | 每个 RPC 的 Request/Response 必须唯一 |
| `RPC_REQUEST_STANDARD_NAME` | Request 必须命名为 `MethodRequest` 或 `ServiceMethodRequest` |
| `RPC_RESPONSE_STANDARD_NAME` | Response 同上 |
| `SERVICE_SUFFIX` | 服务名必须以 `Service` 结尾（后缀可配置） |
| `PROTOVALIDATE` | protovalidate 约束必须有效（v2 配置） |

### 配置

```yaml
# buf.yaml
lint:
  use:
    - STANDARD
  except:                     # 排除特定规则
    - SERVICE_SUFFIX
  ignore:                     # 忽略整个文件或目录
    - proto/google/type/
  ignore_only:                # 特定文件忽略特定规则
    RPC_REQUEST_STANDARD_NAME:
      - proto/legacy/v1/api.proto
  enum_zero_value_suffix: _UNSPECIFIED
  service_suffix: Service
  # disallow_comment_ignores: true  # 生产环境推荐，禁止注释跳过规则
```

### 注释忽略（行级）

```protobuf
// buf:lint:ignore PACKAGE_VERSION_SUFFIX
package legacy;
```

### 迁移现有项目

```bash
# 生成忽略配置，粘贴到 buf.yaml 中
buf lint --error-format=config-ignore-yaml
```

### 运行

```bash
buf lint                             # lint 当前工作区
buf lint --error-format json         # JSON 输出
buf lint --path proto/acme/pet       # 只 lint 特定路径
buf lint buf.build/acme/petapis      # lint BSR 上的模块
```

## 5. Breaking Check

### 规则类别

| 类别 | 检测范围 | 适用场景 |
|------|---------|---------|
| `FILE` | 最严格（默认） | 保护所有生成语言的源码兼容性 |
| `PACKAGE` | 包级别 | 允许类型在包内文件间移动 |
| `WIRE_JSON` | 线路 + JSON | 你控制所有客户端时 |
| `WIRE` | 仅线路二进制 | 你就是自己的客户端 |

**推荐选择一个类别而非混合排除特定规则。有疑问选 `FILE`。**

### 关键规则

| 规则 | 类别 | 说明 |
|------|------|------|
| `ENUM_VALUE_SAME_NAME` | FILE, PACKAGE, WIRE_JSON | 枚举值不能改名 |
| `FIELD_SAME_NAME` | FILE, PACKAGE, WIRE_JSON | 字段名不能改 |
| `FIELD_SAME_TYPE` | FILE, PACKAGE | 字段类型不能改 |
| `FIELD_NO_DELETE` | FILE, PACKAGE | 不能删字段，用 deprecated |
| `ENUM_VALUE_NO_DELETE` | FILE, PACKAGE | 不能删枚举值，用 deprecated |
| `SERVICE_NO_DELETE` | FILE | 不能删服务，用 deprecated |
| `RPC_NO_DELETE` | FILE, PACKAGE | 不能删 RPC，用 deprecated |

### 运行

```bash
# 对比本地 Git 分支
buf breaking --against '.git#branch=main'
buf breaking --against '.git#tag=v1.0.0'

# proto 在子目录时
buf breaking --against '.git#branch=main,subdir=proto'

# 对比 BSR 模块
buf breaking --against buf.build/acme/petapis

# 对比工作区内所有模块（每个模块需有 name）
buf breaking --against-registry

# 对比远程 Git 仓库
buf breaking --against 'https://github.com/acme/petapis.git'

# 对比 GitHub archive
buf breaking --against "https://github.com/acme/petapis/archive/${COMMIT}.tar.gz#strip_components=1"

# 只检查特定文件
buf breaking --against '.git#branch=main' --path proto/acme/pet/v1/pet.proto

# JSON 输出
buf breaking --against '.git#branch=main' --error-format=json
```

### 配置

```yaml
# buf.yaml
breaking:
  use:
    - FILE
  except:
    - RPC_NO_DELETE           # 按需排除
  ignore:
    - proto/legacy/
  ignore_only:
    FIELD_SAME_JSON_NAME:
      - proto/acme/pet/v1/pet.proto
  ignore_unstable_packages: true  # 忽略 v1alpha/v1beta 包
```

## 6. Format

`buf format` 统一 proto 文件风格：

- `package` 移到 `import` 上方
- import 按字母排序
- 缩进统一为 2 空格
- RPC 名称与括号间去掉空格

```bash
buf format                    # 输出到 stdout（预览）
buf format -w                 # 原地修改
buf format -d                 # 显示 diff
buf format --exit-code        # 有 diff 时返回非零退出码（CI 用）
buf format -w --exit-code     # 原地修改 + 退出码
buf format proto -o formatted # 输出到另一个目录
```

## 7. CI 集成（GitHub Actions）

```yaml
name: Proto CI
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  proto:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0           # breaking check 需要完整 git 历史
      - uses: bufbuild/buf-setup-action@v1
        with:
          version: '1.68.4'
      - name: Lint
        run: buf lint
      - name: Format check
        run: buf format --exit-code
      - name: Breaking check
        if: github.event_name == 'pull_request'
        run: buf breaking --against '.git#branch=main'
      - name: Generate and verify
        run: |
          buf generate
          git diff --exit-code gen/  # 确保生成代码已提交
```

## 8. 常用命令速查

```bash
# === 初始化 ===
buf config init                         # 创建 buf.yaml

# === 依赖 ===
buf dep update                          # 更新依赖
buf dep prune                           # 清理未使用依赖

# === 构建 ===
buf build                               # 构建验证（无输出 = 成功）

# === 代码生成（必须在含 buf.gen.yaml 的目录运行）===
buf generate                            # 用 buf.gen.yaml 生成
buf generate --template buf.gen.go.yaml # 指定配置文件
buf generate --path proto/acme/pet      # 只生成特定路径

# === Lint ===
buf lint                                # 检查所有文件
buf lint --error-format json            # JSON 输出
buf lint --error-format=config-ignore-yaml  # 生成忽略配置（迁移用）

# === Breaking ===
buf breaking --against '.git#branch=main'
buf breaking --against '.git#tag=v1.0.0,subdir=proto'
buf breaking --against-registry         # 对比 BSR

# === Format ===
buf format -w                           # 原地格式化
buf format -d --exit-code               # CI：检查格式

# === 其他 ===
buf export buf.build/googleapis/googleapis --output ~/3rd_proto  # 导出第三方 proto
buf config ls-lint-rules                # 列出所有 lint 规则
buf config ls-breaking-rules            # 列出所有 breaking 规则
```

## 9. 常见问题

### go_package 不对 / 编译失败
- 检查 `managed.override` 的 `go_package_prefix` 是否正确
- 第三方模块（googleapis、protovalidate）必须在 `managed.disable` 中排除

### buf generate 找不到文件 / 输出为空
- 确认 `buf.gen.yaml` 中有 `inputs: - directory: proto`
- 确认在项目根目录（含 buf.yaml 和 buf.gen.yaml）运行

### grpc-gateway 生成空文件
- 确认 RPC 方法上有 `google.api.http` 注解
- 确认 `buf.yaml` 的 deps 包含 `buf.build/googleapis`

### protovalidate 编译失败
- `managed.disable` 必须排除 `buf.build/bufbuild/protovalidate` 模块

### buf.build Go 模块版本 404
- 不要手动写版本号，用 `go mod tidy` 自动解析
