---
name: golang-service-enum
description: "Sub-document of golang-service-development skill. Loaded by SKILL.md whenever proto enum ↔ DB int conversion is involved: defining enums in proto, storing int32 in models/dal, converting at the service store boundary, and the boundary cases (i18n labels, YAML parsing). Read this before writing ANY enum conversion helper — proto already generates the API."
---

# 枚举处理（proto enum → DB int）

本文件是 `golang-service-development` 的枚举处理专题。何时读本文件见入口 `SKILL.md` 的「子文档路由」表。

**核心规则**：枚举统一定义在 proto 文件中，DB 存 `int32`，应用层**只能**用 proto 编译器生成的内置函数做转换。**禁止自定义任何枚举转换函数**——`int↔enum`、`string↔enum`、`enum↔string`、`enum↔int slice` 一律不许写 helper，全部走下表的 proto 内置 API。本文档是枚举处理的真相之源。

## Proto 编译器已生成的内置函数（不要重新发明）

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

**典型 service 路径只用前两个**（DB 存取时的类型断言）。`String()` 方法和两个 map 只在日志标签、admin UI、YAML 配置反序列化等少数边界场景用到——见本文档末尾「边界用例」。

> **判定准则**：如果你写的函数输入/输出都是裸 `int32` 或 `string`，那就是绕开 proto、重新发明转换，**禁止**。合法的业务封装（如中文标签）输入或输出**必须有一个是 proto enum 类型**——见本文档「边界用例」。

## Proto 定义

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

## DB 层（model + dal）

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

## Service 层（直接吃 proto，store 边界转换）

Service 方法直接接受 proto 类型。proto enum → int32 的转换发生在调用 dal **之前**，作为字段抽取的一部分。**注意**：方法定义在领域子包（`internal/service/demo/demo.go`），`s` 是子包的 `*demo.Service`（领域子包规则见 `architecture.md` §2）：

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

## Handler 层（不碰枚举）

Handler 是一行委托，**完全不知道枚举存在**：

```go
// pkg/handler/demo.go
func (h *Handler) CreateDemo(ctx context.Context, req *demov1.CreateDemoRequest) (*demov1.Demo, error) {
    return h.svc.CreateDemo(ctx, req)  // req 透传给 service，handler 零转换
}
```

## 反模式（禁止）

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

## 边界用例（基于 proto 内置 map，不重新发明）

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

**判定准则（再说一次，因为这是 AI 最容易滑过去的口子）**：函数的输入或输出**必须有一个是 proto enum 类型**（`demov1.DemoStatus`，不是裸 `int32`/`string`），才不算"重新发明转换"。如果输入输出都是 `int32`/`string`，那就是把 proto 内置能力重写一遍，**禁止**——直接用本文档顶部的速查表。

## 关联

- **入口**：`SKILL.md`（何时用、高频任务快速路径、子文档路由）
- **同级子文档**：`architecture.md`（分层 / 领域子包）、`jobs.md`、`scaffold.md`
