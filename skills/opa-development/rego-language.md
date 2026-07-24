---
name: rego-language
description: "Rego advanced language features — load when needing operator details, destructuring, rule head variables, every full semantics, with scoping, METADATA full spec, implicit joins, schema type checking, or builtin error handling. Trigger: advanced rego patterns, unification operator, self-join, schema validation, metadata annotations."
---

# Rego 高级语言特性

本文件包含 Rego 语言的高级特性和详细语义。**仅在需要时加载**，基础语法见入口文件 SKILL.md。

---

## 赋值、比较与统一化（详细）

Rego 有三种"相等"操作符，语义不同：

| 操作符 | 含义 | 特点 |
|--------|------|------|
| `:=` | 赋值 | 变量必须未使用过，右侧可引用已绑定变量 |
| `==` | 比较 | 不绑定变量，仅检查值相等 |
| `=` | 统一化 | 同时赋值和比较，顺序无关 |

```rego
# := 赋值（推荐，编译时检查）
x := 5
y := x + 1

# == 比较（不绑定变量）
input.role == "admin"

# = 统一化（组合赋值和比较）
# 以下两种写法等价：
input.servers[i].id == "app"
input.servers[i].id = "app"
```

**赋值约束**：
- 变量不能在赋值之前出现在查询中（编译错误）
- 同一变量不能被赋值两次（编译错误）

## 数组解构

```rego
# 解构赋值
[_, _, city, country] := ["John", "Smith", "Paris", "FR"]

# 对象解构
{["id"]: id, ["protocols"]: protos} := input.servers[0]
```

---

## `default` 关键字完整语义

- `default` 可用于规则**和函数**：`default permission_level(_) := "none"`
- 函数的 `default` 参数必须是纯变量、arity 必须与非 default 定义相同
- `default` 的值可以是标量、复合值或推导式，但**不能是变量或引用**

```rego
default permission_level(_) := "none"

permission_level(user) := "admin" if {
    user.role == "admin"
}
```

---

## 规则头引用中的变量

规则头引用可以使用变量来动态构建嵌套集合：

```rego
# 常量路径
fruit.apple.seeds := 12
fruit.banana.seeds := 0

# 变量路径（动态键）
users_by_role[role][id] := user if {
    some role in ["admin", "viewer"]
    some id, user in input.users
    user.role == role
}
```

**冲突检测**：动态范围（dynamic extent）内的规则冲突在编译时检测。规则不允许修改或替换其他规则定义的值。

---

## `else` 链

```rego
authz_decision := "allow" if {
    input.user.role == "admin"
}

else := "deny" if {
    input.user.role == "guest"
}

else := "review" if {
    true  # 兜底
}
```

`else` 特别适用于从 iptables 等顺序敏感系统迁移策略的场景。

---

## `every` 完整语义

```rego
# 基本用法
no_telnet_exposed if {
    every server in input.servers {
        not "telnet" in server.protocols
    }
}
```

**限制与语义**：
- **禁止取反**：`not every x in xs { p(x) }` 是非法的，应使用 `some x in xs; not p(x)` 替代
- **空域为 true**：当域为空时，`every` 语句结果为 true（vacuous truth）
- **不引入新绑定**：`every` 内部的变量不会向外部规则评估引入新绑定
- **可选 key 参数**：`every i, x in [1, 2, 3] { x > i }` 支持双参数形式（索引 + 值）
- 对对象同理：`every k, v in {"a": 1} { v > 0 }`

---

## 函数高级模式

**参数可以是复合值**，实现模式匹配式参数解构：

```rego
# 复合参数模式匹配
user_role([_, {"role": role}]) := role

# 调用
role := user_role(["skip", {"role": "admin"}])  # => "admin"
```

---

## `in` 高级模式

```rego
# 双左参数形式（索引 + 值）
3, "baz" in ["foo", "bar", "baz"]

# 复合值模式匹配
some {"bar": x}, {"foo": y} in [{"bar": 1}, {"foo": 2}]

# 在列表上下文中需要括号包裹
some (3, "baz") in [["a", 1], [3, "baz"]]
```

**`in` 始终返回布尔值**，即使参数不是集合/数组（如 `3 in "three"` → `false`）。

---

## `with` 完整语义

```rego
# 替换 input
allow with input as {"user": "alice", "role": "admin"}

# 替换 data
allow with data.acl as {"alice": ["read"]}

# 替换内置函数（mock http.send）
allow with http.send as {"body": {"allowed": true}}

# 在 mock 中调用原始函数（不递归）
allow with http.send as http_send_raw({"url": "https://..."})

# 替换为函数引用
allow with http.send as my_http_send
```

**完整语义**：
- **作用域**：`with` 只影响附加的表达式，后续表达式看到未修改的值
- **替换函数的值类型**：支持值、函数引用、内置函数引用、规则引用
- **arity 匹配**：替换函数的 arity 必须与原函数相同
- **不能部分定义虚拟文档**：不能对 `data.foo.bar` 做 `with data.foo.bar.baz as ...`

```rego
# 作用域示例
inner := 1 with data.x as 2   # inner = 1（data.x 被替换）
middle := inner + data.x       # data.x 恢复原值
```

---

## 隐式连接（Implicit Joins）

当变量在多个表达式中出现时，Rego 自动执行连接操作。这是 Rego 的核心机制。

```rego
# 隐式连接：变量同时出现在多个表达式中，自动 join
violating_servers[i] := server if {
    some i, server in input.servers
    some j, port in input.ports
    server.ports[_] == port.id      # 通过 port.id 连接
    some network in input.networks
    port.network == network.id      # 通过 network.id 连接
    network.public                  # 条件：公开网络
}
```

### 自连接（Self-Join）

在同一数据源上使用不同键变量：

```rego
# 找出共享同一网络的服务器对
server_pairs[[s1.id, s2.id]] if {
    some s1 in input.servers
    some s2 in input.servers
    s1.id != s2.id
    some p1 in input.ports
    some p2 in input.ports
    s1.ports[_] == p1.id
    s2.ports[_] == p2.id
    p1.network == p2.network
}
```

---

## METADATA 完整规范

```rego
# METADATA
# scope: rule
# title: "Authorization Policy"
# description: "Controls access to resources"
# authors:
#   - name: Team
# related_resources:
#   - url: https://example.com
#     description: "Reference"
# schemas:
#   - input: schema["input"]
# custom:
#   priority: high
# entrypoint: true
# organizations:
#   - team: platform
```

**完整字段说明**：

| 字段 | 说明 |
|------|------|
| `scope` | 作用范围：`rule`（默认）、`document`（整个输出文档）、`package`（当前包）、`subpackages`（当前包及子包） |
| `title` | 简短标题 |
| `description` | 详细描述 |
| `authors` | 作者列表 |
| `related_resources` | 相关资源 URL 列表 |
| `schemas` | 与 Schema 类型检查系统集成（见下文） |
| `custom` | 自定义键值映射 |
| `entrypoint` | 标记为入口点（布尔） |
| `organizations` | 组织/团队信息 |

**注意事项**：
- 注解必须从第 1 列开始（无缩进）
- 运行时可通过 `rego.metadata.rule()` 和 `rego.metadata.chain()` 访问元数据（详见 rego-builtins.md）

---

## Schema 类型检查

OPA 支持使用 JSON Schema 增强 Rego 的类型检查，在编译时发现类型错误。

```bash
# 使用 schema 文件检查
opa check -s schema.json policy.rego

# 在 eval 中使用 schema
opa eval -s schema.json -i input.json -d policy.rego "data.example.allow"
```

### Schema 注解（内联）

```rego
# METADATA
# schemas:
#   - input: schema["input"]
#   - data.acl: schema["acl"]

allow if {
    input.user.role == "admin"  # schema 定义了 role 字段时，编译时检查
}
```

### Schema 注解（引用格式）

```rego
# METADATA
# schemas:
#   - input.recipe[input.location].parameters with data.components as schema["components"]
```

**要点**：
- Schema 文件通过 `-s` 标志传递
- 支持 `anyOf`/`allOf` JSON Schema 组合关键字
- 支持 remote schema 引用（可通过 `--disable-imports` 或 `--set=services.default.allow_net=false` 控制）
- 严格模式下，未通过 schema 检查的字段访问会报编译警告

---

## 内置函数错误处理

默认情况下，内置函数运行时错误（如类型不匹配、除零）返回 `undefined`，**不会中断评估**。这使得策略对异常数据具有容错性。

```rego
# 如果 input.value 不是数字，to_number 返回 undefined，规则整体为 undefined（不报错）
is_valid if {
    to_number(input.value) > 0
}
```

**严格模式**：使用 `--strict-builtin-errors` 标志将内置函数错误视为异常（中断评估并返回错误）。

```bash
# CLI
opa eval --strict-builtin-errors -i input.json -d policy.rego "data.example.allow"

# REST API
curl "localhost:8181/v1/data/example/allow?strict-builtin-errors=true"
```

各接口对应标志：
- CLI：`--strict-builtin-errors`
- REST API：`?strict-builtin-errors=true` 查询参数
- Go module：`rego.StrictBuiltinErrors(true)` 选项
