---
name: opa-development
description: "MUST load when writing, editing, reviewing, or debugging ANY Rego/OPA policy file (.rego), or any OPA/Rego/Open Policy Agent related task, including policy-as-code, admission control, authorization, Terraform plan validation, Kubernetes Gatekeeper policies, Envoy external auth, or CI/CD policy checks. This is the ONLY entry point for the opa-development skill set. Read this file first, then load sub-documents as needed. Trigger keywords: opa, rego, open policy agent, policy as code, admission control, gatekeeper, .rego, opa eval, opa test, opa build, opa run, constraint template, policy engine, authorization policy, kube-mgmt, terraform validate, envoy authz"
---

# OPA/Rego 开发指南

**这是 opa-development skill set 的唯一入口。Agent 必须先读本文件，再按需加载子文档。**

## 版本锁定

| 工具/规范 | 版本 |
|----------|------|
| OPA | v0.60+ / v1.x（支持 `import rego.v1`） |
| Rego 语法 | v1（`import rego.v1` 启用所有新关键字） |
| Gatekeeper | v3.14+ |

如果项目使用不同版本，Agent 应先读取项目中的 `.rego` 文件观察 import 和语法风格，保持一致性。

## 上下文传承

Agent 在应用本 skill 之前，**必须先检查项目现有状态**：

1. **如果项目已有 `.rego` 文件** → 先观察现有 package 命名风格、import 习惯（是否用 `import rego.v1`）、测试结构，保持一致性
2. **如果项目已有 `opa` 或 `conftest` 配置** → 先读取现有配置
3. **如果是 Kubernetes 策略** → 确认使用 Gatekeeper（ConstraintTemplate + Constraint）还是 Plain OPA + kube-mgmt
4. **如果是全新项目** → 按本 skill set 的推荐结构创建

## 子文档路由

本 skill set 包含四个子文档，按需加载：

| 子文档 | 内容 | 何时加载 |
|--------|------|----------|
| [rego-language.md](rego-language.md) | 高级语言特性：操作符细节、解构、`every` 完整语义、`with` 作用域、隐式连接、Schema、METADATA 完整规范 | 需要高级模式、详细语义、或遇到编译错误时 |
| [rego-testing-style.md](rego-testing-style.md) | 测试编写、编码风格、性能优化 | 编写测试、优化策略、配置 CI 时 |
| [rego-builtins.md](rego-builtins.md) | 内置函数参考（~170 个函数） | 需要字符串/正则/集合/编码/时间/加密/网络等操作时 |
| [opa-operations.md](opa-operations.md) | CLI 命令、REST API、服务器配置、集成模式、调试 | 使用 CLI、配置服务器、集成 K8s/Envoy/Terraform 时 |

---

# Rego 语言核心

## 基础概念

Rego 是声明式策略语言，灵感来自 Datalog。策略作者关注"查询应该返回什么"而非"如何执行查询"。

### 数据模型

- **`input`** — 每次查询的外部输入（JSON），只读
- **`data`** — OPA 加载的所有策略和数据文档，可跨 package 引用
- 规则计算的结果也是 `data` 的一部分：`data.<package-path>.<rule-name>`

### 标量值与字符串

```rego
"hello"              # string
`raw\s+string`       # raw string（不解释转义，适合正则）
42                    # number
3.14                  # number
true                  # boolean
null                  # null

# 字符串插值
greeting := sprintf("Hello %s!", [name])   # 传统方式
greeting := $"Hello {name}!"                # 插值方式（推荐，对 undefined 安全）
```

### 复合值

```rego
# 数组（有序，零索引）
ports := ["p1", "p2", "p3"]

# 对象（无序键值，键不限于字符串——可以是数字、布尔值等任意类型）
server := {"id": "app", "protocols": ["https"]}
ips_by_port := {80: ["app"], 443: ["web"]}

# 集合（无序且唯一，查找 O(1)）
protocols := {"https", "ssh"}

# 空值
empty_set := set()     # 空集合
empty_obj := {}        # 空对象（不是空集合！）
```

**性能**：优先使用集合而非数组做成员检查。`"https" in protocols`（集合 O(1)）vs `ports[_] == "https"`（数组 O(n)）。

### 变量

```rego
# 声明变量
some i, j
some server in input.servers

# 赋值（推荐用 :=）
port_id := input.ports[i].id
```

**推荐用 `some` 声明**：避免变量捕获——如果包内新增规则定义了同名变量，不用 `some` 的规则可能意外改变行为。

### 操作符速查

| 操作符 | 含义 | 用途 |
|--------|------|------|
| `:=` | 赋值 | 定义新变量，编译时检查 |
| `==` | 比较 | 检查值相等，不绑定变量 |
| `=` | 统一化 | 同时赋值和比较（高级用法，详见 rego-language.md） |

### 引用与迭代

```rego
input.servers[0].id                          # 点号访问
input.servers[i].protocols[j] == "http"      # 变量键 = 隐式迭代
input.servers[_].protocols[_] == "http"      # 下划线 = 不关心索引
input.ports[{"id": "p1", "network": "net1"}] # 复合键
```

## 规则（Rules）

### 完整规则

生成单一值。同名规则只能有一个为真（否则冲突错误）。

```rego
allow := true if { count(violation) == 0 }  # 完整语法
violation_exists if { count(violation) > 0 } # 省略值（默认 true）
max_protocols := 5                            # 常量（无规则体）
default allow := false                        # 默认值
```

`default` 也可用于函数：`default permission_level(_) := "none"`。值不能是变量或引用。

### 部分集合规则

生成集合，同名规则贡献不同元素（OR 语义）。

```rego
violation contains msg if {
    some server in input.servers
    "http" in server.protocols
    msg := sprintf("server %s exposes http", [server.id])
}
```

### 部分对象规则

生成对象，同名规则贡献不同键值对。

```rego
apps_by_hostname[hostname] := app if {
    some app in input.apps
    some hostname in app.hostnames
}
```

### `else` 链

```rego
authz_decision := "allow" if { input.user.role == "admin" }
else := "deny" if { input.user.role == "guest" }
else := "review" if { true }
```

## 逻辑控制

### AND（规则体内） / OR（同名规则）

```rego
# AND：规则体内多个表达式
allow if {
    input.user == "alice"
    input.method == "GET"
}

# OR：同名规则增量定义
shell_accessible if { input.protocol == "telnet" }
shell_accessible if { input.protocol == "ssh" }
```

### 否定（NOT）

```rego
deny if { not authenticated_user }

authenticated_user if {
    some user in input.users
    user.name == input.request.user
    user.active
}
```

**注意**：否定中的变量必须也在非否定表达式中出现（安全约束）。

### 全称量词（every）

```rego
no_telnet_exposed if {
    every server in input.servers {
        not "telnet" in server.protocols
    }
}
```

**常见错误**：`some` 迭代 + `!=` 不是全称量词！

```rego
# BUG: 只检查"存在一个不是 telnet 的"
all_not_telnet if {
    some server in input.servers
    server.protocol != "telnet"  # WRONG!
}

# CORRECT
all_not_telnet if {
    every server in input.servers {
        server.protocol != "telnet"
    }
}
```

**`every` 限制**：禁止取反（`not every ...` 非法）、空域为 true、不引入新绑定。详见 [rego-language.md](rego-language.md)。

## 函数

```rego
permission_level(user) := "admin" if { user.role == "admin" }
permission_level(user) := "read" if { user.role == "viewer" }

level := permission_level(input.user)  # 调用
```

不支持按参数数量重载。参数可以是复合值用于模式匹配。详见 [rego-language.md](rego-language.md)。

## 推导式（Comprehensions）

```rego
doubled := [n * 2 | some n in [1, 2, 3]]                  # 数组
unique_ids := {server.id | some server in input.servers}   # 集合
port_map := {p.id: p.network | some p in input.ports}      # 对象
```

**性能**：优先使用部分规则而非推导式，更易调试和复用。

## 关键字

### `import rego.v1`（推荐）

```rego
package example
import rego.v1
# 启用：if, contains, some, every, in
```

### `in`（成员检查和迭代）

```rego
allow if "admin" in user.roles              # 成员检查
some server in input.servers                # 迭代
some key, value in input.labels             # 键值迭代
```

高级模式（双左参数、复合值匹配）详见 [rego-language.md](rego-language.md)。

### `with`（模拟/替换，主要用于测试）

```rego
allow with input as {"user": "alice"}       # 替换 input
allow with data.acl as {"alice": ["read"]}  # 替换 data
allow with http.send as {"body": {"allowed": true}}  # mock 内置函数
```

完整语义（作用域、arity 匹配、限制）详见 [rego-language.md](rego-language.md)。

## 模块结构

```rego
package myapp.authz  # 必须是第一条语句

import rego.v1       # 推荐：启用 v1 语法
import data.users    # 按需导入其他 package
import data.example.servers as my_servers  # 别名导入

# METADATA（可选，工具可解析）
# METADATA
# title: "Authorization Policy"
# description: "Controls access to resources"
# entrypoint: true

default allow := false

allow if {
    some user in authenticated_users
    user_has_permission(user, "read", input.path)
}
```

**包名匹配文件路径**：`myapp/authz.rego` → `package myapp.authz`
**METADATA 完整字段**详见 [rego-language.md](rego-language.md)。

## 保留名称

以下**不能**用作变量名、规则名：

```
as, contains, data, default, else, every, false, if, in,
import, input, not, null, package, some, true, with
```

## `box[x]` vs `box contains x`

常见陷阱：

- 有 `contains` → 总是部分集合规则
- 有 `[x]` + 有 `if` + 无 `.` → 完整文档规则（生成对象）
- 有 `[x]` + 无 `if` + 无 `.` → 向后兼容，部分集合规则

---

# 常见陷阱速查

| 陷阱 | 正确做法 |
|------|----------|
| `some` + `!=` 当全称量词 | 用 `every` |
| `not p["foo"]` 当 `p[_] != "foo"` | 前者检查键不存在，后者找任何不等于的 |
| `not every ...` | 非法，用 `some x in xs; not p(x)` |
| 空集合 vs 空对象 | `set()` 是空集合，`{}` 是空对象 |
| 否定中未绑定变量 | 否定表达式中的变量必须也在非否定表达式中出现 |
