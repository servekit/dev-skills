---
name: opa-operations
description: "OPA operations reference — CLI commands, REST API endpoints, server configuration, integration patterns (Kubernetes, Envoy, Terraform), and debugging techniques. Load when deploying, configuring, integrating, or debugging OPA."
---

# OPA 运维与集成

## CLI 命令速查

### opa eval — 评估查询

```bash
# 基本评估
opa eval "1*2+3"

# 评估策略
opa eval -i input.json -d policy.rego "data.example.allow"

# 格式化输出
opa eval -f pretty -i input.json -d policy.rego "data.example.violation[x]"

# 用退出码判断（undefined 返回非零）
opa eval --fail-defined -i input.json -d policy.rego "data.example.violation[x]"

# 优化级别
opa eval -O=1 -i input.json -d policy.rego "data.example.allow"
```

| 标志 | 简写 | 说明 |
|------|------|------|
| `--data` | `-d` | 加载策略或数据文件 |
| `--bundle` | `-b` | 加载 bundle 文件或目录 |
| `--input` | `-i` | 加载 input 文件 |
| `--format` | `-f` | 输出格式：`json`(默认)/`values`/`bindings`/`pretty`/`source`/`raw`/`discard` |
| `--fail` | | undefined 时返回非零退出码 |
| `--fail-defined` | | 非 undefined 时返回非零退出码 |
| `--coverage` | | 报告覆盖率 |
| `--profile` | | 启用性能分析 |
| `--explain` | | 求值追踪：`off`/`full`/`notes`/`fails`/`debug` |
| `--partial` | | 部分评估模式 |

### opa test — 运行测试

```bash
# 运行所有测试
opa test .

# 带覆盖率
opa test --coverage .

# 详细输出
opa test -v .

# 正则过滤
opa test -r "test_admin" .

# 基准测试
opa test --bench .

# 覆盖率阈值
opa test --threshold 0.9 .

# 自动重跑（开发时）
opa test --watch .
```

### opa check — 检查语法

```bash
# 基本检查（解析+编译）
opa check policy.rego

# 严格模式（检查未使用变量/import）
opa check --strict policy.rego

# Rego v1 模式
opa check --rego-v1 policy.rego
```

### opa fmt — 格式化

```bash
# 格式化并输出
opa fmt policy.rego

# 原地覆盖
opa fmt -w policy.rego

# CI 检查（有变更返回非零）
opa fmt --fail policy.rego

# 显示 diff
opa fmt -d policy.rego
```

### opa build — 构建 bundle

```bash
# 构建 bundle
opa build -b ./policies/

# 指定入口点和优化
opa build -b ./policies/ -e example/allow -O=1

# 编译为 Wasm
opa build -b ./policies/ -e example/allow -t wasm

# 指定 revision
opa build -b ./policies/ --revision v1.0.0

# 签名
opa build -b ./policies/ --signing-key mykey.pem
```

### opa run — 启动 REPL 或服务器

```bash
# 交互式 REPL
opa run

# REPL 加载数据
opa run input.json

# REPL 加载策略和 input
opa run example.rego repl.input:input.json

# 服务器模式
opa run --server -b ./bundle/

# 自定义监听地址
opa run --server -a 0.0.0.0:8181 -b ./bundle/

# 启用认证
opa run --server --authentication=token --authorization=basic -b ./bundle/

# 启用 TLS
opa run --server \
  --tls-cert-file cert.pem \
  --tls-private-key-file key.pem \
  -b ./bundle/

# 安全部署建议
# 1. 绑定 localhost（默认行为）：--addr localhost:8181
# 2. 限制 API 访问：--authentication=token --authorization=basic
# 3. 最小授权策略（只允许 POST /）：
#    default allow := false
#    allow if { input.method == "POST"; input.path == [""] }
# 4. 使用 Diagnostic Listener 分离健康检查端口：
#    --diagnostic-addr :8282
# 5. 以非 root 用户运行
# 6. 使用 --set-file 加载 secrets
```

### opa bench — 基准测试

```bash
opa bench -i input.json -d policy.rego "data.example.allow"
```

### opa inspect — 检查 bundle 内容

```bash
opa inspect bundle.tar.gz
```

### opa deps — 查看依赖

```bash
opa deps -d policy.rego "data.example.allow"
```

---

## REST API

OPA 服务器默认监听 `localhost:8181`。

### Data API

```bash
# 获取文档
GET /v1/data/{path}

# 带输入查询
POST /v1/data/{path}
Content-Type: application/json
{"input": {...}}

# Webhook 风格（v0，body 直接是 input）
POST /v0/data/{path}
Content-Type: application/json
{...}

# 写入/覆盖文档
PUT /v1/data/{path}
Content-Type: application/json
{...}

# JSON Patch 更新
PATCH /v1/data/{path}
Content-Type: application/json-patch+json
[{"op": "replace", "path": "/key", "value": "new"}]

# 删除文档
DELETE /v1/data/{path}
```

### Policy API

```bash
# 列出策略
GET /v1/policies

# 获取策略
GET /v1/policies/{id}

# 创建/更新策略
PUT /v1/policies/{id}
Content-Type: text/plain
package ...

# 删除策略
DELETE /v1/policies/{id}
```

### Query API

```bash
# Ad hoc 查询
GET /v1/query?q=<url-encoded-query>

# POST 查询
POST /v1/query
Content-Type: application/json
{"query": "data.example.allow", "input": {...}}
```

### Compile API（部分评估）

```bash
POST /v1/compile
Content-Type: application/json
{"query": "data.example.allow", "input": {...}, "unknowns": ["input"]}
```

### 运维 API

```bash
# 健康检查
GET /health
GET /health?bundles=true&plugins=true

# 自定义健康检查
GET /health/{rule}  # 使用 system.health.{rule} 策略

# 运行状态
GET /status

# 配置
GET /config

# Prometheus 指标
GET /metrics

# 默认决策（无需路径）
POST /
Content-Type: application/json
{...}
```

### 查询参数

| 参数 | 说明 |
|------|------|
| `explain=full\|notes\|fails\|debug` | 求值追踪 |
| `metrics=true` | 性能指标 |
| `provenance=true` | 构建版本和 bundle revision |
| `pretty=true` | 格式化输出 |

### 健康检查详细参数

```bash
# 排除特定插件
GET /health?exclude-plugin=bundle

# 自定义健康检查（使用 system.health.{rule} 策略）
GET /health/{rule}

# 自定义健康检查收到的 input：
# input.plugins_ready  — 所有插件是否就绪
# input.plugin_state.<plugin_name>  — 特定插件状态
```

### 错误响应格式

所有 API 错误返回统一 JSON 结构：

```json
{
  "code": "invalid_parameter",
  "message": "parameter 'q' is missing",
  "errors": [...],
  "location": {"file": "policy.rego", "row": 10, "col": 5}
}
```

### Trace Events 结构

使用 `explain` 参数时，响应中包含 trace events：

```json
{
  "explain": [
    {
      "op": "Enter",
      "query_id": 0,
      "parent_id": -1,
      "type": "body",
      "node": {...},
      "locals": [...]
    }
  ]
}
```

事件类型：`Enter`（进入）、`Exit`（退出）、`Eval`（评估）、`Fail`（失败）、`Redo`（重做）。

### 性能指标名称

`metrics=true` 返回的关键指标：

| 指标 | 说明 |
|------|------|
| `timer_rego_input_parse_ns` | 输入解析耗时 |
| `timer_rego_query_compile_ns` | 查询编译耗时 |
| `timer_rego_query_eval_ns` | 查询评估耗时 |
| `timer_rego_external_resolve_ns` | 外部数据解析耗时 |
| `counter_server_query_cache_hit` | 查询缓存命中次数 |

`instrument=true` 提供更细粒度的编译阶段指标。

---

## 服务器配置

配置文件支持 YAML/JSON，通过 `-c` 参数指定。

### 最小配置

```yaml
services:
  acme:
    url: https://bundles.acme.com

bundles:
  authz:
    service: acme
    resource: bundles/bundle.tar.gz

decision_logs:
  console: true                        # 见下方 decision_logs 详细配置
```

### 认证方式

```yaml
services:
  acme:
    url: https://bundles.acme.com
    credentials:
      # 1. Bearer Token
      bearer:
        token: "secret-token"           # 或 token_path: /path/to/token
        scheme: "Bearer"                # 可选，默认 "Bearer"

      # 2. Client TLS 证书
      # client_tls:
      #   cert: /path/to/cert.pem
      #   private_key: /path/to/key.pem

      # 3. OAuth2 Client Credentials（基础）
      # oauth2:
      #   token_url: https://oauth.example.com/token
      #   client_id: id
      #   client_secret: secret
      #   scopes: ["read", "write"]

      # 4. OAuth2 JWT Authentication
      # oauth2:
      #   token_url: https://oauth.example.com/token
      #   grant_type: jwt_bearer                     # JWT Bearer Grant
      #   signing_key: /path/to/key.pem
      #   claims: {"aud": "https://oauth.example.com/token"}
      #   # 或 AWS KMS 签名：
      #   # aws_kms:
      #   #   key_id: alias/my-key
      #   #   region: us-east-1
      #   # 或 Azure Key Vault 签名：
      #   # azure_keyvault:
      #   #   key_name: my-key
      #   #   vault_url: https://myvault.vault.azure.net
      #   # 或 Azure Workload Identity：
      #   # azure_workload_identity:
      #   #   resource: https://vault.azure.net

      # 5. AWS Signature v4
      # s3_signing:
      #   environment: aws
      #   aws_region: us-east-1
      #   # 多种凭证来源：
      #   # environment_credentials: {}                # 环境变量
      #   # named_credentials/profile: my-profile       # Named Profile
      #   # sso_credentials/account_id: "123"/role_name: "MyRole"  # AWS SSO
      #   # metadata_credentials:                       # EC2/ECS Metadata
      #   #   iam_role: "my-role"                       # EC2
      #   #   region: us-east-1
      #   # assume_role_credentials:                    # STS AssumeRole
      #   #   role_arn: "arn:aws:iam::123:role/MyRole"
      #   #   session_name: "opa-session"
      #   #   external_id: "ext-id"
      #   # web_identity_credentials:                   # EKS Web Identity
      #   #   role_arn: "arn:aws:iam::123:role/MyRole"
      #   #   web_identity_token_path: /var/run/secrets/eks.amazonaws.com/serviceaccount/token

      # 6. GCP Metadata Token
      # gcp_metadata:
      #   audience: https://bundles.example.com
      #   # access_token: true                          # 使用 access token（默认 identity token）

      # 7. Azure Managed Identity
      # azure_managed_identity:
      #   endpoint: "http://169.254.169.254/metadata/identity/oauth2/token"
      #   resource: "https://vault.azure.net"

      # 8. 自定义插件认证
      # plugin: my_auth_plugin
```

### OCI Registry（Bundle 来源）

OCI 注册表作为 bundle 来源（支持 GHCR、AWS ECR 等）：

```yaml
# type: oci
# url: ghcr.io/my-org/my-bundle
```

### Bundle 配置

```yaml
bundles:
  authz:
    service: acme
    resource: bundles/bundle.tar.gz
    polling:
      min_delay_seconds: 60            # 最小轮询间隔
      max_delay_seconds: 300           # 最大轮询间隔
    signing:
      keyid: mykey                     # 签名验证密钥 ID
    persist: true                       # 持久化到磁盘
    trigger:
      type: periodic                   # periodic | manual（手动触发）
    size_limit_bytes: 1073741824       # 单文件大小限制（默认 1GB）

keys:
  mykey:
    algorithm: RS256
    key: |                             # 或 remote: {url: ...}
      -----BEGIN PUBLIC KEY-----
      ...
      -----END PUBLIC KEY-----
```

### 决策日志

```yaml
decision_logs:
  console: true                        # 开发调试用
  service: acme                        # 上报到远程服务
  resource: /v1/logs                   # 自定义日志上报路径
  reporting:
    min_delay_seconds: 30
    max_delay_seconds: 300
  buffer_type: size                    # size（按字节）| event（按事件数）
  buffer_size_limit_bytes: 1048576     # size 模式缓冲上限
  buffer_size_limit_events: 1000       # event 模式缓冲上限
  max_decisions_per_second: 1000       # 速率限制
  upload_size_limit_bytes: 1048576     # 单次上传大小限制
  trigger:
    type: periodic                     # periodic | immediate | manual
  masking:
    - type: "remove"                   # 移除敏感字段
      oneof:
        - path: ["input", "password"]
        - path: ["input", "ssn"]
  mask_decision: system.mask.decision  # 自定义 mask 路径（策略控制）
  drop_decision: system.drop.decision  # 自定义 drop 路径（策略控制）
  request_context:
    http:
      headers: ["X-Request-ID"]        # 记录请求头到日志
```

### 磁盘存储

```yaml
storage:
  disk:
    directory: /path/to/storage        # BadgerDB 数据目录
    auto_create: true
```

### 分布式追踪

```yaml
distributed_tracing:
  type: grpc                           # grpc 或 http
  address: "localhost:4317"
  service_name: "opa"
```

### 环境变量替换

配置中支持 `${VAR_NAME}` 语法引用环境变量。

### CLI 覆盖

```bash
opa run --set=default_decision=example/allow \
        --set=services.acme.url=https://bundles.acme.com

# 从文件读取值（用于 secrets）
opa run --set-file=default_decision=policy.rego \
        --set-file=services.acme.credentials.bearer.token=/run/secrets/token

# Remote bundle 速写（直接传 URL）
opa run -s -c config.yaml https://bundles.acme.com/authz.tar.gz
```

### Discovery 插件（动态配置）

Discovery 插件允许 OPA 从远程动态拉取配置，无需重启。

```yaml
discovery:
  resource: discovery/bundle.tar.gz    # discovery bundle 资源路径
  service: acme                        # 关联的 service
  decision: config                     # bundle 中用作配置的决策路径
  polling:
    min_delay_seconds: 60
    max_delay_seconds: 300
  signing:
    keyid: mykey
  persist: true
  trigger:
    type: periodic                     # periodic | manual
```

**关键行为**：Discovery bundle 中的 trigger 模式会被 bundle/decision_log/status 插件继承。

### Caching 配置

```yaml
caching:
  inter_query_builtin_cache:
    max_size_bytes: 10000000           # 最大缓存大小（字节）
    forced_eviction_threshold_percentage: 90
    stale_entry_eviction_period_seconds: 10
  inter_query_builtin_value_cache:
    max_num_entries: 10000
    named:
      io_jwt:                          # JWT 验证缓存
        max_num_entries: 1000
      graphql:                         # GraphQL schema 缓存
        max_num_entries: 100
```

### Server 配置

```yaml
server:
  decoding:
    max_length: 1048576                # 请求体最大长度（字节）
    gzip:
      max_length: 1048576              # gzip 解压最大长度
  encoding:
    gzip:
      min_length: 1024                 # 响应压缩最小长度
      compression_level: 6             # 1-9
  metrics:
    prom:
      http_request_duration_seconds:
        buckets: [0.1, 0.25, 0.5, 1, 2.5, 5, 10]  # Prometheus histogram buckets
```

### 杂项配置

```yaml
labels:                                # OPA 实例标识标签
  app: my-app
  env: production

default_decision: example/allow        # 默认决策路径（默认 /system/main）
default_authorization_decision: system/authz/allow  # 默认授权决策路径

persistence_directory: /var/run/opa    # 持久化目录
nd_builtin_cache: true                 # 启用非确定性内置函数缓存
```

### Status 插件

```yaml
status:
  service: acme                        # 上报到远程服务
  partition_name: "zone-a"             # 分区名称
  console: true                        # 控制台输出
  prometheus: true                     # 导出 Prometheus 指标
  trigger:
    type: periodic                     # periodic | manual
  plugin: status                       # 自定义状态插件
```

Prometheus 指标：`opa_info`、`plugin_status_gauge`、`bundle_loaded_counter`、`bundle_loading_duration_ns`。

---

## 集成模式

### Kubernetes（Admission Control）

两种模式：

**1. OPA Gatekeeper（推荐）**

原生 CRD：`ConstraintTemplate`（Rego 策略）+ `Constraint`（参数和匹配）。

```yaml
apiVersion: templates.gatekeeper.sh/v1
kind: ConstraintTemplate
metadata:
  name: k8srequiredlabels
spec:
  crd:
    spec:
      names:
        kind: K8sRequiredLabels
      validation:
        openAPIV3Schema:
          type: object
          properties:
            labels:
              type: array
              items:
                type: string
  targets:
    - target: admission.k8s.gatekeeper.sh
      rego: |
        package k8srequiredlabels
        violation[{"msg": msg}] {
          provided := {label | input.review.object.metadata.labels[label]}
          required := {label | label := input.parameters.labels[_]}
          missing := required - provided
          count(missing) > 0
          msg := sprintf("Missing labels: %v", [missing])
        }
```

策略关键字段：
- `input.request.kind` — 资源类型
- `input.request.operation` — 操作（CREATE/UPDATE/DELETE）
- `input.request.object` — 资源对象
- `input.request.oldObject` — 更新前的对象
- `input.request.userInfo` — 请求用户

**2. Plain OPA + kube-mgmt**

kube-mgmt sidecar 将 ConfigMap 中的策略加载到 OPA。支持 decision logs/bundles 等管理功能。

### Envoy（External Authorization）

OPA 通过 Envoy ext_authz 过滤器提供策略决策。

```rego
package envoy.authz

import rego.v1
import input.attributes.request.http as http_request

default allow := false

allow if {
    http_request.method == "GET"
    path_allowed(http_request.path)
}
```

部署方式：OPA 以 sidecar 运行在每个 Envoy 旁边。

### Terraform Plan 验证

```bash
# 生成 plan JSON
terraform plan -out=tfplan
terraform show -json tfplan > tfplan.json

# 评估策略
opa eval -i tfplan.json -d policy.rego "data.terraform.deny[x]"
```

关键 input 路径：
- `input.resource_changes` — 资源变更列表
- `input.resource_changes[_].change.actions` — 操作（create/update/delete）
- `input.resource_changes[_].type` — 资源类型
- `input.planned_values` — 计划值

### Go 嵌入

```go
import "github.com/open-policy-agent/opa/rego"

r := rego.New(
    rego.Query("data.example.allow"),
    rego.Load([]string{"./policy.rego"}, nil),
)

query, err := r.PrepareForEval(ctx)
// query 可缓存、跨 goroutine 共享

rs, err := query.Eval(ctx, rego.EvalInput(input))
result := rs[0].Expressions[0].Value
```

### Go SDK（高级 API，推荐用于生产）

```go
import "github.com/open-policy-agent/opa/sdk"

// 从文件系统加载 bundle（无需远程服务器）
opa, err := sdk.New(ctx, sdk.Options{
    ID:     "opa-instance-1",
    Bundle: "./bundle.tar.gz",
    Config: bytes.NewReader(configYAML),
    Ready:  make(chan struct{}),  // 阻塞直到 OPA 就绪
})
defer opa.Stop(ctx)

// 查询决策
result, err := opa.Decision(ctx, sdk.DecisionOptions{
    Path: "example/allow",
    Input: map[string]interface{}{"user": "alice"},
})
```

**低级 rego 包 vs 高级 sdk 包**：

| 维度 | `rego` 包 | `sdk` 包 |
|------|-----------|----------|
| 级别 | 低级 API | 高级 API |
| Bundle 管理 | 手动 | 自动（支持远程拉取） |
| 配置 | 代码中设置 | YAML 配置文件 |
| Decision Logs | 手动实现 | 内置支持 |
| 适合场景 | 嵌入式、单次查询 | 生产部署、完整 OPA 功能 |

### Wasm

```bash
# 编译为 Wasm
opa build -t wasm -e example/allow -b ./policies/
```

JavaScript SDK：`@open-policy-agent/opa-wasm`

### 集成方式比较

| 维度 | REST API（Sidecar） | Go Library | Wasm |
|------|---------------------|------------|------|
| 评估速度 | Fast | Faster | Fastest |
| 语言支持 | 任何语言 | 仅 Go | 任何支持 Wasm 的语言 |
| 运维复杂度 | 需管理和保护 OPA | 需重新部署服务 | 很少更新服务 |
| 安全性 | 必须保护 API 端点 | 按需启用 | 按需启用 |
| 适用场景 | 微服务、多语言 | Go 服务、嵌入控制 | 高性能、边缘计算 |

### REST API 集成（Sidecar 模式）

非 Go 应用的推荐方式：OPA 作为 sidecar，应用通过 HTTP 查询策略。

---

### 授权策略（system.authz）

启用 `--authorization=basic` 时，OPA 使用 `system.authz` 包的策略决定是否允许请求。

```rego
package system.authz

import rego.v1

default allow := false

# authz 策略收到的 input 结构：
# input.identity          — 请求者身份（token 或 TLS CN）
# input.client_certificates — TLS 客户端证书链（mTLS 场景）
# input.method            — HTTP 方法
# input.path              — 请求路径数组
# input.params            — 查询参数
# input.headers           — 请求头
# input.body              — 请求体

allow if {
    input.method == "POST"
    input.path == [""]                    # 只允许默认决策端点
    token_is_valid
}

# 支持结构化响应（allow + reason）
result := {
    "allowed": true,
    "reason": "request allowed",
} if {
    input.method == "GET"
    input.path == ["health"]
}

token_is_valid if {
    # Bearer token 验证逻辑
    input.identity == "expected-token"
}
```

### Metrics Export（OTLP）

```yaml
metrics_export:
  type: otlp/grpc                      # otlp/grpc | otlp/http
  address: "localhost:4317"
  export_interval_ms: 30000
  service_name: "opa"
  encryption:
    type: tls                          # off | tls | mtls
  tls_ca_cert_file: /path/to/ca.pem
  tls_cert_file: /path/to/cert.pem
  tls_private_key_file: /path/to/key.pem
```

### Distributed Tracing 完整配置

```yaml
distributed_tracing:
  type: grpc                           # grpc | http
  address: "localhost:4317"
  service_name: "opa"
  sample_percentage: 100               # 采样百分比
  encryption:
    type: tls                          # off | tls | mtls
  resource:
    service_version: "1.0.0"
    service_instance_id: "opa-1"
    service_namespace: "production"
    deployment_environment: "prod"
  batch_span_processor_options:
    blocking: false
    batch_timeout_ms: 5000
    export_timeout_ms: 30000
    max_export_batch_size: 512
    max_queue_size: 2048
```

### OCI 仓库（Bundle 来源）

OPA 支持从 OCI 兼容注册表（GHCR、AWS ECR、Docker Hub 等）拉取 bundle：

```yaml
services:
  ghcr:
    url: ghcr.io
    type: oci
    credentials:
      bearer:
        token: "ghp_xxxx"

bundles:
  authz:
    service: ghcr
    resource: my-org/my-bundle:latest  # org/repo:tag
```

### Compile API 数据过滤器

OPA 可以将策略编译为 SQL WHERE 子句或 UCAST（用于数据库层策略执行）：

```bash
# 编译为 SQL WHERE 子句
POST /v1/compile
Content-Type: application/json
Accept: application/vnd.opa.sql; version=1

{
  "query": "data.example.allow",
  "input": {...},
  "unknowns": ["input"],
  "options": {
    "disableInlining": ["data.example.acl"],
    "targetDialects": ["postgresql"],
    "targetSQLTableMappings": {
      "input.users": {"table": "users", "columns": {"id": "user_id", "role": "role"}}
    }
  }
}
```

返回编译后的 SQL WHERE 子句。支持 `Accept` header 控制目标格式：
- `application/vnd.opa.sql; version=1` — SQL 方言
- `application/vnd.opa.ucast; version=1` — UCAST 变体

---

## 调试技巧

### print 内置函数

```rego
debug if {
    print("user:", input.user, "path:", input.path)
    some role in data.roles[input.user]
    print("role:", role)
}
```

### 求值追踪

```bash
# CLI
opa eval --explain full -i input.json -d policy.rego "data.example.allow"

# REST API
curl localhost:8181/v1/data/example/allow?explain=full \
  -d @input.json -H 'Content-Type: application/json'
```

### REPL 调试

```bash
opa run example.rego repl.input:input.json

# 查看数据
> data.example

# 追踪模式
> trace data.example.allow
```

### Rego Playground

Web 端调试环境，可分享策略和数据：https://play.openpolicyagent.org/

### 性能调试

见上方 `opa eval --profile` 标志和 `opa bench` 命令。

```bash
# OPA 详细日志
opa run --server --log-level debug -b ./bundle/

# 决策日志见上方 decision_logs 配置
# 手动查询 OPA 状态见上方运维 API
```

### 错误排查

常见错误类型：
- `rego_parse_error` — 语法错误
- `rego_type_error` — 类型错误（如 arity mismatch）
- `rego_unsafe_var_error` — 未绑定变量
- `rego_recursion_error` — 递归规则
- `eval_conflict_error` — 完整规则冲突（多个同名规则同时为真）

```bash
# 严格模式检查（见上方 opa check）
# 显示内置函数错误详情
opa eval --show-builtin-errors -i input.json -d policy.rego "data.example.allow"
```
