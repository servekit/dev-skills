---
name: proto-best-practices
description: "Sub-document of proto-development skill. Loaded by SKILL.md when writing or editing .proto files. Covers proto3 syntax, naming conventions, scalar types, enums, message design, 1-1-1 rule, service design, wire compatibility, Well-Known Types, JSON mapping, project structure."
---

# Protobuf 项目开发指南

基于 [Proto3 Language Guide](https://protobuf.dev/programming-guides/proto3/)、[Style Guide](https://protobuf.dev/programming-guides/style/)、[JSON Mapping](https://protobuf.dev/programming-guides/json/)、[Best Practices](https://protobuf.dev/best-practices/dos-donts/)、[1-1-1 Rule](https://protobuf.dev/best-practices/1-1-1/)、[Protovalidate](https://protovalidate.com)、[Buf](https://buf.build/docs) 编写。

本文档分三大部分：

- **第一部分：编写规范的 proto 文件** — 核心内容，始终适用
- **第二部分：添加字段验证** — 简要说明 + 引用 protovalidate skill
- **第三部分：安装和使用 Buf** — 简要说明 + 引用 buf-usage skill

---

# 第一部分：编写规范的 proto 文件

## 0. 项目目录结构

在编写 proto 文件之前，先规划好项目结构。以下是三种常见场景的推荐布局。

### 单服务项目（最常见）

proto 定义和业务代码在同一仓库：

```
pet-service/
├── buf.yaml                          # Buf 模块配置
├── buf.gen.yaml                      # 代码生成配置
├── buf.lock                          # 依赖锁定
├── proto/                            # proto 源文件（语言无关）
│   └── acme/pet/v1/                  # 路径 = package（acme.pet.v1）
│       ├── pet_type.proto            # 枚举
│       ├── pet.proto                 # 消息
│       ├── create_pet_request.proto  # 1-1-1：每文件一个顶层元素
│       ├── create_pet_response.proto
│       ├── get_pet_request.proto
│       ├── get_pet_response.proto
│       └── pet_service.proto         # 服务定义
├── gen/                              # 生成代码（不提交 / 或 CI 提交）
│   └── go/
├── cmd/
│   └── server/
├── internal/
├── go.mod
└── go.sum
```

**要点：**
- `proto/` 是语言无关的纯净 proto 源文件，不包含任何语言选项
- `gen/` 是生成的代码，通过 `buf generate` 生成，`buf.gen.yaml` 中 `clean: true` 每次清空重建
- `proto/` 下的目录路径必须匹配 `package`：`proto/acme/pet/v1/` → `package acme.pet.v1`
- `buf.yaml` 中 `modules.path` 设为 `proto`
- `buf.gen.yaml` 中 `inputs.directory` 设为 `proto`

### 多服务 Monorepo

多个服务共享同一仓库，proto 定义集中管理：

```
platform/
├── buf.yaml                          # 工作区配置（多个模块）
├── buf.lock
├── buf.gen.yaml
├── proto/                            # 模块 1：业务服务
│   └── acme/
│       ├── user/v1/                  # 用户服务
│       │   ├── user.proto
│       │   └── user_service.proto
│       └── order/v1/                 # 订单服务
│           ├── order.proto
│           └── order_service.proto
├── common/                           # 模块 2：共享类型（可选独立模块）
│   └── acme/common/v1/
│       ├── money.proto
│       └── address.proto
├── gen/
│   └── go/
├── services/
│   ├── user-service/
│   │   ├── cmd/
│   │   └── internal/
│   └── order-service/
│       ├── cmd/
│       └── internal/
├── go.mod
└── go.sum
```

**要点：**
- `buf.yaml` 中声明多个模块（`modules.path: proto` 和 `modules.path: common`）
- 各服务独立 package 路径：`acme.user.v1`、`acme.order.v1`、`acme.common.v1`
- 共享类型放 `common/` 模块，避免循环依赖
- `gen/` 集中生成，各服务 import 同一个 gen 目录

### 独立 Proto 仓库

proto 定义单独一个仓库，各语言消费方通过 BSR 或 Git 依赖：

```
acme-proto/                           # 独立仓库
├── buf.yaml
├── buf.gen.yaml                      # 可选：CI 中推送到各语言 artifact
├── buf.lock
├── acme/
│   ├── pet/v1/
│   │   ├── pet_type.proto
│   │   ├── pet.proto
│   │   └── pet_service.proto
│   └── user/v1/
│       └── user_service.proto
└── gen/                              # CI 生成并推送到 artifact registry
    ├── go/                           # → Go module
    ├── java/                         # → Maven package
    └── ts/                           # → npm package
```

**要点：**
- 无 `proto/` 子目录，proto 文件直接在根目录下（`buf.yaml` 中 `modules` 默认 path `.`）
- 每个服务独立版本化（`buf.yaml` 中各模块设 `name`）
- 可推送到 BSR（`buf push`），消费方通过 `deps` 声明依赖
- CI 负责 `buf generate` 并推送到各语言的 artifact registry

### 核心原则

| 原则 | 说明 |
|------|------|
| proto 和 gen 分离 | proto 是源文件，gen 是生成产物，不要混放 |
| proto 是语言无关的 | 不在 proto 文件中硬编码 `go_package` 等语言选项（用 managed mode） |
| 目录路径 = package | `proto/acme/pet/v1/` 对应 `package acme.pet.v1` |
| buf 配置在根目录 | `buf.yaml`、`buf.gen.yaml`、`buf.lock` 放在项目根目录 |
| 1-1-1 影响文件数 | package 内每个 message/enum/service 独立一个文件 |
| gen 目录可 .gitignore | 生成代码由 `buf generate` 重建，不需要手动编辑 |

## 1. 文件结构

### 文件命名

文件名使用 `lower_snake_case.proto`：
```
pet_service.proto
create_pet_request.proto
student_id.proto
```

### 文件内部顺序

```protobuf
// 1. Syntax（必须是第一个非空非注释行）
syntax = "proto3";

// 2. Package
package acme.pet.v1;

// 3. Imports（按字母排序，buf format 会自动排）
import "buf/validate/validate.proto";
import "google/protobuf/timestamp.proto";
import "acme/pet/v1/pet_type.proto";

// 4. Options（managed mode 下不需要写 go_package 等）
// option go_package = "...";  // 用 managed mode，不手动写

// 5. 消息、枚举、服务定义
```

### 格式规范

- 行长度不超过 80 字符
- 缩进 2 个空格
- 字符串使用双引号

## 2. 命名规范

### 标识符命名风格

| 风格 | 规则 | 示例 |
|------|------|------|
| `TitleCase` | 首字母大写，每个单词首字母大写 | `CreatePetRequest` |
| `lower_snake_case` | 全小写，单词间下划线分隔 | `pet_name` |
| `UPPER_SNAKE_CASE` | 全大写，单词间下划线分隔 | `PET_TYPE_DOG` |

**缩写当作单个单词处理**：`GetDnsRequest` 而非 `GetDNSRequest`，`dns_request` 而非 `d_n_s_request`。

**下划线规则**：下划线不能作为名称的首尾字符，下划线后必须跟字母（不能是数字或另一个下划线）。用 `XYZ_V2` 而非 `XYZ_2`。

### 各元素命名

| 元素 | 风格 | 示例 |
|------|------|------|
| Package | `lower_snake_case` 点分隔 | `acme.pet.v1` |
| Message | `TitleCase` | `CreatePetRequest` |
| Field | `lower_snake_case`，repeated 用复数 | `pet_id`, `pets` |
| Enum Type | `TitleCase` | `PetType` |
| Enum Value | `UPPER_SNAKE_CASE`，前缀为枚举名 | `PET_TYPE_DOG` |
| Service | `TitleCase` | `PetStoreService` |
| RPC Method | `TitleCase` | `CreatePet` |
| Oneof | `lower_snake_case` | `pet_id` |

### Package 命名

- 使用点分隔的 `lower_snake_case`：`acme.pet.v1`
- 不使用 Java 风格的反域名：`com.company.x.y` → 用 `x.y` 作为 package，`java_package` 选项单独设置
- 不与目录路径耦合，保持简短但唯一
- 最后一段必须是版本后缀：`v1`、`v1alpha`、`v1beta`

### 目录结构匹配 package

`package acme.pet.v1` 必须在 `acme/pet/v1/` 目录下。

## 3. 字段类型选择

### 标量类型

| 类型 | 说明 | Go 类型 |
|------|------|---------|
| `double` | IEEE 754 双精度 | `float64` |
| `float` | IEEE 754 单精度 | `float32` |
| `int32` | 变长编码，负数效率低 | `int32` |
| `int64` | 变长编码，负数效率低 | `int64` |
| `uint32` | 变长编码，无符号 | `uint32` |
| `uint64` | 变长编码，无符号 | `uint64` |
| `sint32` | 变长编码，**负数高效**（ZigZag） | `int32` |
| `sint64` | 变长编码，**负数高效**（ZigZag） | `int64` |
| `fixed32` | 固定 4 字节，大于 2^28 时比 uint32 高效 | `uint32` |
| `fixed64` | 固定 8 字节，大于 2^56 时比 uint64 高效 | `uint64` |
| `sfixed32` | 固定 4 字节 | `int32` |
| `sfixed64` | 固定 8 字节 | `int64` |
| `bool` | 布尔值 | `bool` |
| `string` | UTF-8 或 7-bit ASCII，最长 2^32 | `string` |
| `bytes` | 任意字节序列，最长 2^32 | `[]byte` |

**选择建议：**
- 字段可能有负值 → 用 `sint32`/`sint64` 而非 `int32`/`int64`
- 值经常大于 2^28（32 位）或 2^56（64 位）→ 用 `fixed32`/`fixed64`

### 字段编号

- 范围：1 到 536,870,911（29 位，3 位用于 wire format）
- 19000 到 19999 保留给 Protobuf 内部使用
- 1-15 编码为 1 字节，16-2047 编码为 2 字节 → 常用字段用小编号
- **编号一旦使用就永远不能改**

### 默认值

| 类型 | 默认值 |
|------|--------|
| string | `""` |
| bytes | `b""` |
| bool | `false` |
| 数值类型 | `0` |
| message | 未设置（`null`） |
| enum | 第一个定义的值（必须是 0） |
| repeated | 空列表 |
| map | 空映射 |

### 字段基数（Cardinality）

- **`optional`（推荐）**：可区分"未设置"和"设置为默认值"。与 proto2 和 editions 最大兼容。
- **implicit（不推荐）**：无显式标签。非 message 类型的字段无法区分"未设置"和"设置为默认值"。
- **`repeated`**：零或多个值，顺序保留。proto3 中标量数值类型默认 packed 编码。
- **`map`**：键值对，key 不能是浮点数/bytes/enum/message。

## 4. 枚举

### 基本规则

```protobuf
enum PetType {
  PET_TYPE_UNSPECIFIED = 0;   // 零值必须是第一个，无语义含义
  PET_TYPE_DOG = 1;
  PET_TYPE_CAT = 2;
  PET_TYPE_FISH = 3;
}
```

- 第一个值**必须**是 0，名称应为 `ENUM_NAME_UNSPECIFIED` 或 `ENUM_NAME_UNKNOWN`
- 零值不应有语义含义，它表示"未指定"
- 编号应**密集递增**，只在删除值时才出现间隔
- 避免负值（varint 编码效率低）

### 枚举值前缀

枚举值在语义上不被包含它的枚举名限定作用域。同名枚举值在不同枚举中会导致编译冲突：

```protobuf
// 错误：两个枚举都有 SET，会冲突
enum CollectionType { SET = 1; }
enum TennisVictoryType { SET = 2; }  // 编译失败
```

**必须用枚举名作为前缀**：

```protobuf
enum CollectionType {
  COLLECTION_TYPE_UNSPECIFIED = 0;
  COLLECTION_TYPE_SET = 1;
  COLLECTION_TYPE_MAP = 2;
}
```

前缀去掉后，剩余部分仍应是合法的枚举值名：
```protobuf
// 错误：去掉 DEVICE_TIER_ 后只剩数字 1，不是合法标识符
enum DeviceTier {
  DEVICE_TIER_1 = 1;
}

// 正确
enum DeviceTier {
  DEVICE_TIER_TIER1 = 1;
}
```

### 别名

```protobuf
enum Status {
  option allow_alias = true;
  STATUS_UNSPECIFIED = 0;
  STATUS_STARTED = 1;
  STATUS_RUNNING = 1;    // 别名，序列化时使用第一个（STARTED）
  STATUS_FINISHED = 2;
}
```

### 删除枚举值

必须 reserve 编号和名称：
```protobuf
enum Foo {
  reserved 2, 15, 9 to 11, 40 to max;
  reserved "FOO", "BAR";
}
```

## 5. 消息设计

### 1-1-1 规则

每个 `.proto` 文件一个顶层元素（message、enum、extension）。每个文件对应一个 build target。

**好处：**
- **简化重构**：移动文件比从大文件中提取 message 容易得多
- **减少传递依赖**：只需依赖用到的类型，不会意外拉入大量依赖
- **改善构建时间**：更小的编译单元

**示例结构：**

```
proto/acme/pet/v1/
├── pet_type.proto                # enum PetType
├── pet.proto                     # message Pet
├── create_pet_request.proto      # message CreatePetRequest
├── create_pet_response.proto     # message CreatePetResponse
├── get_pet_request.proto         # message GetPetRequest
├── get_pet_response.proto        # message GetPetResponse
├── list_pets_request.proto       # message ListPetsRequest
├── list_pets_response.proto      # message ListPetsResponse
├── delete_pet_request.proto      # message DeletePetRequest
├── delete_pet_response.proto     # message DeletePetResponse
└── pet_service.proto             # service PetStoreService
```

**何时可以放宽：**
- 循环依赖时（不可能拆分）
- 概念强耦合的消息（放在一起可读性更好）
- 无 import 的文件（不存在传递依赖问题）

### 域类型复用

将跨多个 Request/Response 使用的类型提取为独立的域类型 message：

```protobuf
// student_id.proto — 可跨文件复用
message StudentId {
  string value = 1;
}

// full_name.proto — PII 标记只需在一处
message FullName {
  string family_name = 1;
  string given_name = 2;
}

// student.proto — 组合域类型
message Student {
  StudentId id = 1;
  FullName name = 2;
}
```

好处：
- 修改只需更新一处
- 对 PII 字段统一标记敏感
- 添加字段（如 `middle_name`）时，所有引用自动获得更新

### 嵌套类型

```protobuf
message SearchResponse {
  message Result {             // 嵌套定义
    string url = 1;
    string title = 2;
  }
  repeated Result results = 1;
}

// 外部引用
message SomeOtherMessage {
  SearchResponse.Result result = 1;
}
```

### Oneof

互斥字段，最多只有一个被设置：

```protobuf
message Payment {
  oneof method {
    CreditCard credit_card = 1;
    BankTransfer bank_transfer = 2;
  }
}
```

注意：
- 设置一个字段会自动清除其他字段
- 不能是 `repeated`
- 不能将 `map` 字段放入 oneof
- 但可以放入一个包含 `repeated` 的 message

### Maps

```protobuf
map<string, Project> projects = 3;
```

- key 不能是浮点数、bytes、enum、message
- value 不能是另一个 map
- 不能是 `repeated`
- 迭代顺序未定义
- 重复 key 时，最后一个生效
- wire format 等价于 `repeated MapFieldEntry { key_type key = 1; value_type value = 2; }`

### Any 类型

用于嵌入任意 message（不需要提前知道 .proto 定义）：

```protobuf
import "google/protobuf/any.proto";

message ErrorStatus {
  string message = 1;
  repeated google.protobuf.Any details = 2;
}
```

**优先使用 extensions 而非 Any**。Any 有设计缺陷，只在基础设施需要传播完全任意 message 的罕见场景中使用。

#### Any 类型的权衡

Any 看似灵活，但在实际业务中引入显著的复杂性：

**不推荐 Any 的原因：**
- **丧失类型安全**：消费方必须手动解包（`UnpackTo`），编译器无法检查类型正确性
- **自描述格式开销**：Any 在 wire format 中包含 type URL + 完整消息体，体积比普通字段大
- **难以文档化和治理**：API 消费方无法从 proto 定义推断 Any 中可以放什么类型
- **JSON 互操作性差**：`@type` 字段在 JSON 中嵌入，破坏了普通的 JSON 结构

**Any 合理使用的场景（仅限基础设施层）：**
- 错误详情传播：`google.rpc.Status.details` 是 `repeated Any`，这是 gRPC 标准模式
- 中间件/代理需要透传不透明消息，不关心内容
- 插件式架构，消息类型在编译时确实无法确定

**推荐替代方案：**

| 场景 | 替代方案 |
|------|---------|
| 多种可选载荷 | `oneof` — 类型安全，消费方可以 switch |
| 可扩展的元数据 | `map<string, string>` 或 `google.protobuf.Struct` |
| 跨服务通用类型 | 定义明确的域类型，通过 import 共享 |
| 真正需要开放扩展 | proto extensions（Edition 2023+） |

### 消息大小

不要创建包含数百个字段的消息。每个字段在 C++ 中增加约 65 位内存占用，过多字段可能导致生成代码无法编译（如 Java 方法大小限制）。

### RPC 与存储使用不同消息

```protobuf
// RPC API 消息
message CreateUserRequest {
  string name = 1;
  string email = 2;
}

// 存储消息（独立演进）
message UserRecord {
  string id = 1;
  string name = 2;
  string email = 3;
  google.protobuf.Timestamp created_at = 4;
}
```

为 API 和长期存储使用不同的消息类型，即使初始时大量重复。这给了修改存储格式而不影响外部客户端的自由。

## 6. 服务设计

### gRPC 服务

```protobuf
service PetStoreService {
  rpc CreatePet(CreatePetRequest) returns (CreatePetResponse);
  rpc GetPet(GetPetRequest) returns (GetPetResponse);
  rpc ListPets(ListPetsRequest) returns (ListPetsResponse);
  rpc DeletePet(DeletePetRequest) returns (DeletePetResponse);
}
```

- 每个 RPC 使用独立的 Request/Response 消息（STANDARD lint 规则强制）
- Request 命名为 `<Method>Request` 或 `<Service><Method>Request`
- Response 同理

### 不要用布尔值表示可能扩展为多状态的事物

```protobuf
// Bad
message Photo {
  bool gif = 1;    // 未来可能还有 WebP、PNG...
}

// Good
enum PhotoType {
  PHOTO_TYPE_UNSPECIFIED = 0;
  PHOTO_TYPE_GIF = 1;
  PHOTO_TYPE_WEBP = 2;
  PHOTO_TYPE_PNG = 3;
}

message Photo {
  PhotoType type = 1;
}
```

## 7. 兼容性规则

### 二进制 Wire-unsafe 变更（永远不要做）

- 改变任何现有字段的编号（等同于删除+新建）
- 将字段移入已存在的 `oneof`

### 二进制 Wire-safe 变更（安全）

- 添加新字段（旧代码忽略新字段，新代码对旧数据使用默认值）
- 删除字段（**必须 reserve 编号和名称**）
- 添加枚举值
- 将单个 `optional` 字段改为**新建的** `oneof` 的成员
- 只含一个字段的 `oneof` 改回 `optional`
- 将字段改为同编号同类型的 extension

### 二进制 Wire-compatible 变更（条件安全）

需要仔细管控部署顺序：

- `int32`/`uint32`/`int64`/`uint64`/`bool` 之间互相兼容（溢出时 C++ 风格截断）
- `sint32` 和 `sint64` 互相兼容，但**不兼容**其他整数类型
- `string` 和 `bytes` 兼容（前提是 bytes 是合法 UTF-8）
- `fixed32` 和 `sfixed32` 兼容，`fixed64` 和 `sfixed64` 兼容
- `enum` 与 `int32`/`uint32`/`int64`/`uint64` 兼容
- `map<K,V>` 与对应的 `repeated MapFieldEntry` 二进制兼容

### JSON Wire 安全

**Wire-unsafe（会破坏解析）：**
- 在 `string` 和 `bytes` 之间改
- 在 message 类型和 `bytes` 之间改
- 从 `optional` 改为 `repeated`（反之不行）
- 在 `map<K,V>` 和对应 `repeated` 之间改
- 将字段移入已存在的 `oneof`

**Wire-safe（安全）：**
- `int32`/`sint32`/`sfixed32`/`fixed32` 之间互相改
- `int64`/`sint64`/`sfixed64`/`fixed64` 之间互相改
- 添加字段和枚举值（但旧客户端遇到新数据会解析失败，因为 JSON 不传播 unknown fields）

### 核心原则

- **永远不要复用 tag 编号**：即使"没人用了"也不能复用，序列化数据可能存在于日志中
- **删除字段必须 reserve**：`reserved 2, 3;`（编号）+ `reserved "old_name";`（名称，JSON 兼容）
- **不要改字段类型**：几乎永远不安全，除了上面列出的兼容类型对
- **不要从 repeated 改为 scalar**：会丢失数据。JSON 中丢失整个 message
- **不要用布尔值表示可能扩展的状态**：用枚举
- **推荐 `deprecated` 而非删除字段**：`string old_field = 1 [deprecated = true];`
- **不要依赖序列化稳定性**：不要用序列化结果做缓存 key

## 8. Well-Known Types

不要自己定义已有的通用类型：

| 类型 | 用途 | 导入 |
|------|------|------|
| `google.protobuf.Timestamp` | 时间点（RFC 3339） | `google/protobuf/timestamp.proto` |
| `google.protobuf.Duration` | 时间段 | `google/protobuf/duration.proto` |
| `google.protobuf.Empty` | 空响应 | `google/protobuf/empty.proto` |
| `google.protobuf.Any` | 任意类型（优先用 extensions） | `google/protobuf/any.proto` |
| `google.protobuf.FieldMask` | 字段掩码（部分更新） | `google/protobuf/field_mask.proto` |
| `google.protobuf.Struct` | 任意 JSON 对象 | `google/protobuf/struct.proto` |
| `google.protobuf.Value` | 任意 JSON 值 | `google/protobuf/struct.proto` |

Common Types（需依赖 googleapis）：

| 类型 | 用途 |
|------|------|
| `google.type.Date` | 日历日期 |
| `google.type.Money` | 货币金额 |
| `google.type.LatLng` | 经纬度 |
| `google.type.Color` | RGBA 颜色 |
| `google.type.Interval` | 时间区间 |
| `google.type.DayOfWeek` | 星期 |
| `google.type.TimeOfDay` | 一天中的时间 |

## 9. JSON 映射

### 类型映射

| Protobuf 类型 | JSON 表示 | 示例 |
|--------------|----------|------|
| message | object | `{"fooBar": v}` |
| enum | string | `"FOO_BAR"` |
| map | object | `{"k": v}` |
| repeated | array | `[v1, v2]` |
| bool | boolean | `true` |
| string | string | `"hello"` |
| bytes | base64 string | `"YWJj"` |
| int32, fixed32, uint32 | number | `1, -10` |
| int64, fixed64, uint64 | **string** | `"1", "-10"` |
| float, double | number | `1.1, "NaN", "Infinity"` |

**int64 在 JSON 中是字符串**：JSON 数字精度只到 2^53，超过会丢失精度。

### 字段名映射

- proto 字段名 `snake_case` 自动转为 JSON `lowerCamelCase` 作为 key
- 可用 `json_name` 选项自定义
- 解析器同时接受 lowerCamelCase 和原始 proto 字段名

### null 处理

- 序列化时不输出 `null`
- 解析时接受 `null`，等同于字段未设置
- `null` 不允许出现在 repeated 数组内

### 默认值行为

- 有 presence 的字段（`optional`、message 类型）：未设置时不输出
- 无 presence 的字段（implicit scalar）：默认值时不输出
- 可通过选项强制输出默认值

### JSON 选项

| 选项 | 说明 |
|------|------|
| Always emit fields without presence | 输出包含默认值的字段 |
| Ignore unknown fields | 解析时忽略未知字段 |
| Use proto field name | 用 proto 原始字段名而非 lowerCamelCase |
| Emit enum values as integers | 枚举值输出数字而非字符串 |

### 不要用 Text/JSON 格式做数据交换

Text format 和 JSON 中字段名和枚举值名以字符串形式序列化。旧代码解析新数据时，重命名的字段或枚举值会导致解析失败。

- 二进制格式用于数据交换
- Text/JSON 格式仅用于调试和人工编辑

## 10. Proto 文件存放

- proto 文件放在语言无关的目录中（如 `proto/`），不要和特定语言源码混放
- 文件路径相对于 proto_path 必须全局唯一
- 如果对外发布，路径中应包含唯一的库名避免文件名冲突

---

# 第二部分：添加字段验证（Protovalidate）

## 什么时候需要字段验证

当你的 proto 字段需要以下约束时：

- **格式验证**：email、UUID、IP、URI、hostname
- **范围限制**：数值范围、字符串长度、repeated 元素数量
- **必填字段**：`optional` 字段必须被设置
- **跨字段验证**：开始时间 < 结束时间、密码确认匹配
- **自定义业务规则**：条件性验证、嵌套消息属性唯一

## 基本用法

在 proto 文件中 import 并使用 `(buf.validate.field)` 注解：

```protobuf
syntax = "proto3";
package acme.user.v1;

import "buf/validate/validate.proto";

message CreateUserRequest {
  // 邮箱格式 + 必填
  string email = 1 [(buf.validate.field).string.email = true];

  // 长度范围
  string name = 2 [(buf.validate.field).string = { min_len: 1, max_len: 100 }];

  // 数值范围
  int32 age = 3 [(buf.validate.field).int32 = { gte: 0, lte: 150 }];

  // 跨字段验证（消息级 CEL 规则）
  option (buf.validate.message).cel = {
    id: "password.match"
    message: "password and confirmation must match"
    expression: "this.password == this.confirm_password"
  };
}
```

## 更深入的验证需求

当你需要以下内容时，**必须阅读 [protovalidate.md](protovalidate.md)** 获取完整文档：

| 需求 | 参考章节 |
|------|---------|
| 所有标准规则（string/int/float/enum/repeated/map/duration/timestamp/bytes） | 标准规则章节 |
| 自定义 CEL 表达式（跨字段验证、动态错误消息） | 自定义 CEL 规则章节 |
| CEL 语法、函数、类型映射 | CEL 表达式详解章节 |
| 可复用预定义规则 | 预定义规则章节 |
| Go 运行时 / gRPC 拦截器 / 测试 | Go 运行时集成章节 |

---

# 第三部分：安装和使用 Buf

## 为什么用 Buf

Buf 是 protoc 的现代替代品，解决以下问题：

- **零本地安装**：远程插件从 BSR 拉取，不需要本地安装 protoc-gen-go 等
- **Managed mode**：自动设置 go_package 等语言选项，proto 文件保持干净
- **Lint + Breaking check**：内置 proto 规范检查和破坏性变更检测
- **依赖管理**：像 Go mod 一样管理 proto 依赖（buf.yaml deps + buf.lock）

## 什么时候需要 Buf

| 场景 | 是否需要 Buf |
|------|-------------|
| 编写 .proto 文件 | 不需要，但建议用 `buf format -w` 格式化 |
| 构建/验证 proto | `buf build` |
| 生成代码（Go/Java/...） | `buf generate` |
| proto 规范检查 | `buf lint` |
| 破坏性变更检测 | `buf breaking` |
| 格式化 proto 文件 | `buf format -w` |
| 设置 proto CI | buf-usage 的 CI 章节 |

## 基本项目结构

```
project/
├── buf.yaml              # 模块/工作区配置
├── buf.gen.yaml          # 代码生成配置
├── buf.lock              # 依赖锁定
├── proto/                # proto 文件
│   └── acme/pet/v1/
└── gen/                  # 生成代码
    └── go/
```

## 快速开始

```bash
# 1. 安装
brew install bufbuild/buf/buf

# 2. 初始化项目
buf config init

# 3. 构建验证
buf build

# 4. Lint 检查
buf lint

# 5. 格式化
buf format -w

# 6. 生成代码
buf generate
```

## 更详细的 Buf 使用指南

当你需要以下内容时，**必须阅读 [buf-usage.md](buf-usage.md)** 获取完整文档：

| 需求 | 参考章节 |
|------|---------|
| 安装 buf（macOS/Linux/Go/Docker） | 安装章节 |
| 配置 buf.yaml（单模块/多模块/依赖） | 模块与工作区章节 |
| 配置 buf.gen.yaml（插件/managed mode） | 代码生成章节 |
| Managed mode 详解（override/disable） | 代码生成 > Managed Mode 章节 |
| 远程插件 vs 本地插件 | 代码生成 > 插件章节 |
| Lint 规则（STANDARD/BASIC/MINIMAL） | Lint 规则章节 |
| Breaking check（FILE/PACKAGE/WIRE） | Breaking Check 章节 |
| GitHub Actions CI 配置 | CI 集成章节 |
| 命令速查和常见问题 | 命令速查 + 常见问题章节 |
