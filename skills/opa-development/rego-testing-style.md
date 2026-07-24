---
name: rego-testing-style
description: "Rego testing, style guide, and performance optimization — load when writing tests (_test.rego), applying coding conventions, optimizing policy performance, or setting up CI checks. Trigger: opa test, test_, _test.rego, opa fmt, opa check --strict, performance, benchmark, coverage."
---

# Rego 测试、风格与性能

本文件包含策略测试编写、编码风格指南和性能优化。**在编写测试、优化策略或配置 CI 时加载**。

---

# 策略测试

## 测试格式

测试规则以 `test_` 前缀命名，推荐放在 `_test.rego` 后缀的文件中。

```rego
package myapp.authz_test  # 推荐 _test 后缀

import rego.v1

test_admin_allowed if {
    allow with input as {
        "user": "alice",
        "role": "admin",
        "action": "read",
        "resource": "data"
    }
}

test_guest_denied if {
    not allow with input as {
        "user": "guest",
        "role": "viewer",
        "action": "write",
        "resource": "data"
    }
}
```

## 参数化测试（数据驱动）

```rego
test_permissions if {
    some tc in [
        {"user": "alice", "role": "admin", "action": "read", "expected": true},
        {"user": "bob", "role": "viewer", "action": "write", "expected": false},
        {"user": "carol", "role": "editor", "action": "read", "expected": true},
    ]
    result := allow with input as tc
    result == tc.expected
}
```

## Mock 内置函数

```rego
test_with_mock_http if {
    allow with http.send as {"status_code": 200, "body": {"allowed": true}}
    with input as {"token": "valid"}
}
```

## 运行测试

CLI 命令详见 [opa-operations.md](opa-operations.md) 的 `opa test` 部分。

## 待办测试（SKIPPED）

以 `todo_` 前缀的测试标记为跳过：

```rego
todo_test_future_feature if {
    # 待实现
    false
}
```

---

# 风格指南

以下建议**补充**入口文件未覆盖的最佳实践，入口文件已述的原则不再重复。

## 通用原则

- **优化可读性，非性能**：让 OPA 担心性能，策略作者关注清晰
- **使用 `opa fmt`**：统一格式化，CI 中用 `opa fmt --fail` 强制
- **使用严格模式**：`opa check --strict` 检查未使用的变量和 import（具体检查项：未使用的本地赋值、未使用的 imports）

## 命名

- **snake_case** 命名规则和变量：`is_admin`、`user_name`
- **前导下划线**标记内部规则/函数：`_internal_helper(x)`
- **不用 `get_`/`list_` 前缀**：Rego 本身无副作用
- **行长度 <= 120 字符**

## 规则组织

- **优先在规则头赋值**：`full_name := concat(" ", [first, last])` 而非在规则体中赋值
- **使用辅助规则和函数**拆分复杂规则

## 函数

- **函数应只依赖参数**：不直接引用 `input`/`data`，更易测试和复用
- **避免用最后参数作为返回值**

## 导入

- **优先导入包而非单个规则**：`import data.user` 然后 `user.is_admin`
- **避免导入 `input`**：保持 `input` 来源明显
- **使用别名避免冲突**：`import data.example.servers as my_servers`

---

# 性能优化

## 核心原则

- **用对象替代数组**：有唯一标识符时用对象（key-value），查找从 O(n) 变为 O(1)
- **最小化迭代和搜索**：避免深层嵌套的 `[_]` 迭代
- **使用可索引的语句**：等式语句（`==`）和 glob 语句（`glob.match`）可被 OPA 特殊索引

## 优化级别

- `-O=0`（默认）：不优化
- `-O=1`（推荐）：部分评估，不依赖 unknown 的规则被内联
- `-O=2`（激进）：更激进的内联，包括复制传播和否定语句

CLI 命令详见 [opa-operations.md](opa-operations.md) 的 `opa eval` 部分。

## 性能分析与基准测试

详见 [opa-operations.md](opa-operations.md) 的 `opa eval --profile`、`opa bench` 和调试技巧部分。

## Early Exit

完整文档规则和函数规则只产生单一值时，OPA 找到第一个匹配后立即停止。利用此特性优化：

```rego
# 好：Early Exit 生效
allow if {
    input.role == "admin"     # 第一个不匹配就跳过
    input.action == "read"    # 第二个不匹配就跳过
}

# 差：全量迭代
allow if {
    some role in input.roles
    role == "admin"
    some action in input.actions
    action == "read"
}
```

## 推导式索引（Comprehension Indexing）

对需要 group-by 的操作，OPA 自动记忆推导式结果，将 O(n²) 降为 O(n)。需满足：
- 使用赋值语句（`:=`）
- 无 `with`
- 非否定
- 变量安全
- 闭包变量

## 存储优化

`opa run --server --optimize-store-for-read-speed` 预计算 AST 表示，牺牲写入性能换取读取性能。
