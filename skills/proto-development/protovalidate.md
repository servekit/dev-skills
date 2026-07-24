---
name: protovalidate
description: "Sub-document of proto-development skill. Loaded by SKILL.md when adding field validation to .proto files. Covers all standard rules, custom CEL expressions, predefined reusable rules, CEL syntax and extensions, Go runtime, gRPC interceptor."
---

# Protovalidate 字段验证指南

基于 [Protovalidate 官方文档](https://protovalidate.com) 编写。引用的所有规则和 API 均来自官方文档。

## 1. 概述

Protobuf 提供类型安全，Protovalidate 提供**数据正确性**。在 proto 文件中定义验证规则，所有语言（Go、Java、Python、C++、TypeScript）一致执行。

**核心设计：**
- 所有标准规则底层都是 CEL 表达式
- 标准规则覆盖大部分场景，CEL 自定义规则处理跨字段验证
- 规则编译到 descriptor 中，运行时库直接执行，无需代码生成插件

**选择策略：**
- **单字段验证** → 优先用标准规则（更简洁、已优化）
- **跨字段验证** → 用自定义 CEL 规则
- **重复使用的规则组** → 用预定义规则

## 2. Buf 配置

### buf.yaml 添加依赖

```yaml
version: v2
modules:
  - path: proto
deps:
  - buf.build/bufbuild/protovalidate    # 必须
  - buf.build/googleapis/googleapis      # 如需 google.type.* 或 google.api.*
```

### buf.gen.yaml 排除 managed mode

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
      module: buf.build/bufbuild/protovalidate   # 必须，否则编译失败
    - file_option: go_package
      module: buf.build/googleapis/googleapis
plugins:
  - remote: buf.build/protocolbuffers/go
    out: gen/go
    opt: paths=source_relative
  - remote: buf.build/grpc/go
    out: gen/go
    opt: paths=source_relative
```

### 安装和生成

```bash
buf dep update
buf generate
go get buf.build/go/protovalidate
go mod tidy
```

## 3. 通用字段规则

### required

要求字段必须被设置。适用于有 presence tracking 的字段（`optional`、message 类型）。

```protobuf
message FieldsWithPresence {
  // optional string：required 确保非空（包括空字符串也算"已设置"）
  optional string link = 1 [(buf.validate.field).required = true];

  // message 类型：确保消息被设置
  SomeMessage msg = 2 [(buf.validate.field).required = true];
}
```

**注意：** 无 presence 的字段（implicit string、repeated、map）始终被验证，不需要 `required`。但对这些字段加 `required` 表示值不能是零值。

### ignore

当字段值为零值时跳过验证：

```protobuf
message UpdateRequest {
  // uri 规则只在 url 非空时生效
  string url = 1 [
    (buf.validate.field).ignore = IGNORE_IF_ZERO_VALUE,
    (buf.validate.field).string.uri = true
  ];
}
```

### example

提供示例值，不影响验证，仅作文档用途：

```protobuf
message MyMessage {
  int32 value = 1 [
    (buf.validate.field).int32.example = 1,
    (buf.validate.field).int32.example = -10
  ];
}
```

## 4. 标准规则 — String

所有 string 规则也适用于 `google.protobuf.StringValue`。

### 长度

```protobuf
message Examples {
  // 精确字符长度（Unicode code points，非字节）
  string country_code = 1 [(buf.validate.field).string.len = 2];

  // 字符长度范围
  string username = 2 [(buf.validate.field).string = {
    min_len: 3,
    max_len: 32
  }];

  // 字节长度（适用于含非 ASCII 字符串）
  string bio = 3 [(buf.validate.field).string = {
    min_bytes: 1,
    max_bytes: 1024
  }];

  // 精确字节长度
  string token = 4 [(buf.validate.field).string.len_bytes = 32];
}
```

### 子串匹配

```protobuf
message Examples {
  string prefix_str = 1 [(buf.validate.field).string.prefix = "user_"];
  string suffix_str = 2 [(buf.validate.field).string.suffix = ".com"];
  string contains_at = 3 [(buf.validate.field).string.contains = "@"];
  string no_spam = 4 [(buf.validate.field).string.not_contains = "spam"];
}
```

### 正则

```protobuf
string pattern = 1 [(buf.validate.field).string.pattern = "^[a-z0-9]+$"];
```

RE2 语法。反斜杠需要双重转义：`\\d`。

### 格式验证

```protobuf
message Formats {
  string email = 1 [(buf.validate.field).string.email = true];
  string hostname = 2 [(buf.validate.field).string.hostname = true];
  string uri = 3 [(buf.validate.field).string.uri = true];
  string uri_ref = 4 [(buf.validate.field).string.uri_ref = true];     // 允许相对 URI
  string ip = 5 [(buf.validate.field).string.ip = true];              // v4 或 v6
  string ipv4 = 6 [(buf.validate.field).string.ipv4 = true];
  string ipv6 = 7 [(buf.validate.field).string.ipv6 = true];
  string uuid = 8 [(buf.validate.field).string.uuid = true];
  string tuuid = 9 [(buf.validate.field).string.tuuid = true];         // 无横线 UUID
  string address = 10 [(buf.validate.field).string.address = true];    // hostname 或 IP
}
```

### IP 前缀

```protobuf
message IPPrefixes {
  string ip_prefix = 1 [(buf.validate.field).string.ip_with_prefixlen = true];     // 如 192.168.5.21/16
  string ipv4_prefix = 2 [(buf.validate.field).string.ipv4_with_prefixlen = true];
  string ipv6_prefix = 3 [(buf.validate.field).string.ipv6_with_prefixlen = true];
  string net = 4 [(buf.validate.field).string.ip_prefix = true];                   // 掩码位必须全零
  string net4 = 5 [(buf.validate.field).string.ipv4_prefix = true];
  string net6 = 6 [(buf.validate.field).string.ipv6_prefix = true];
}
```

### 其他

```protobuf
message Other {
  // 主机:端口
  string host_port = 1 [(buf.validate.field).string.host_and_port = "example.com:8080"];
  // ULID
  string ulid = 2 [(buf.validate.field).string.ulid = true];
  // 枚举值
  string const_val = 3 [(buf.validate.field).string.const = "hello"];
  // 白名单/黑名单
  string in_list = 4 [(buf.validate.field).string.in = "apple"];
  string not_in_list = 5 [(buf.validate.field).string.not_in = "banned"];
}
```

**String 规则完整列表：**

| 规则 | 说明 |
|------|------|
| `const` | 精确等于 |
| `len` / `min_len` / `max_len` | 字符长度（Unicode code points） |
| `len_bytes` / `min_bytes` / `max_bytes` | 字节长度 |
| `pattern` | RE2 正则 |
| `prefix` / `suffix` | 前缀/后缀 |
| `contains` / `not_contains` | 包含/不包含子串 |
| `in` / `not_in` | 在/不在列表中 |
| `email` | 邮箱（HTML 标准定义） |
| `hostname` | 主机名 |
| `ip` / `ipv4` / `ipv6` | IP 地址 |
| `ip_with_prefixlen` / `ipv4_with_prefixlen` / `ipv6_with_prefixlen` | IP/前缀长度 |
| `ip_prefix` / `ipv4_prefix` / `ipv6_prefix` | IP 前缀（掩码位全零） |
| `uri` / `uri_ref` | URI / URI 引用 |
| `address` | 主机名或 IP |
| `uuid` / `tuuid` | UUID / 无横线 UUID |
| `ulid` | ULID |
| `host_and_port` | 主机:端口 |
| `well_known_regex` | 已知正则（如 HTTP_HEADER_VALUE） |
| `strict` | 严格模式（用于 well_known_regex） |
| `example` | 示例值（文档用） |

## 5. 标准规则 — 数值类型

所有整数规则也适用于对应的 Wrapper 类型（如 `Int32Value`、`UInt64Value` 等）。

**支持的数值类型：** `int32`, `int64`, `uint32`, `uint64`, `sint32`, `sint64`, `fixed32`, `fixed64`, `sfixed32`, `sfixed64`, `float`, `double`

### 通用规则

所有数值类型共享以下规则（以 `int32` 为例）：

```protobuf
message NumericExamples {
  // 精确值
  int32 const_val = 1 [(buf.validate.field).int32.const = 42];

  // 范围
  int32 quantity = 2 [(buf.validate.field).int32 = { gte: 1, lte: 100 }];

  // 大于/小于
  int32 positive = 3 [(buf.validate.field).int32.gt = 0];
  int32 under_limit = 4 [(buf.validate.field).int32.lt = 1000];

  // 白名单/黑名单
  int32 allowed = 5 [(buf.validate.field).int32 = { in: [1, 2, 3] }];
  int32 forbidden = 6 [(buf.validate.field).int32 = { not_in: [0, -1] }];
}
```

### 排除范围

当 `gt` > `lt` 时，值必须**在范围外**：

```protobuf
// 值必须 > 10 或 < 5（不在 5-10 范围内）
int32 outside = 1 [(buf.validate.field).int32 = { gt: 10, lt: 5 }];
```

### Float/Double 额外规则

```protobuf
message FloatExamples {
  double price = 1 [(buf.validate.field).double = { gt: 0.0, finite: true }];
  // finite: 不允许 Inf 和 NaN
}
```

**数值规则完整列表：**

| 规则 | 说明 |
|------|------|
| `const` | 精确等于 |
| `lt` / `lte` | 小于 / 小于等于 |
| `gt` / `gte` | 大于 / 大于等于 |
| `in` / `not_in` | 在/不在列表中 |
| `finite` | 有限值（仅 float/double） |
| `example` | 示例值 |

## 6. 标准规则 — 其他类型

### Bool

```protobuf
bool accepted = 1 [(buf.validate.field).bool.const = true];  // 必须为 true
```

### Bytes

所有 bytes 规则也适用于 `google.protobuf.BytesValue`。

```protobuf
message BytesExamples {
  bytes hash = 1 [(buf.validate.field).bytes.len = 32];
  bytes data = 2 [(buf.validate.field).bytes = { min_len: 1, max_bytes: 1024 }];
  bytes magic = 3 [(buf.validate.field).bytes.prefix = "\x89PNG"];
  bytes ip_bytes = 4 [(buf.validate.field).bytes.ip = true];      // IP 地址（字节格式）
  bytes uuid_bytes = 5 [(buf.validate.field).bytes.uuid = true];  // 16 字节 UUID
  bytes pattern = 6 [(buf.validate.field).bytes.pattern = "^[a-zA-Z0-9]+$"];  // 需 UTF-8
}
```

### Enum

```protobuf
enum Status {
  STATUS_UNSPECIFIED = 0;
  STATUS_ACTIVE = 1;
  STATUS_INACTIVE = 2;
}

message EnumExamples {
  // 只允许已定义值
  Status status = 1 [(buf.validate.field).enum.defined_only = true];
  // 精确值
  Status exact = 2 [(buf.validate.field).enum.const = 1];
  // 白名单/黑名单（用数字值）
  Status allowed = 3 [(buf.validate.field).enum = { in: [1, 2] }];
}
```

### Repeated

```protobuf
message RepeatedExamples {
  // 数量限制
  repeated string tags = 1 [(buf.validate.field).repeated = {
    min_items: 1,
    max_items: 50
  }];

  // 元素唯一
  repeated string ids = 2 [(buf.validate.field).repeated.unique = true];

  // 验证每个元素
  repeated string emails = 3 [(buf.validate.field).repeated = {
    min_items: 1,
    items: {
      string: { email: true, max_len: 254 }
    }
  }];
}
```

**注意：** `repeated.items` 中不需要 `required`，因为 repeated 元素始终存在。

### Map

```protobuf
message MapExamples {
  // 数量限制
  map<string, string> labels = 1 [(buf.validate.field).map = {
    min_pairs: 1,
    max_pairs: 50
  }];

  // 验证 key 和 value
  map<string, int32> scores = 2 [(buf.validate.field).map = {
    min_pairs: 1,
    keys: { string: { min_len: 1, max_len: 32 } },
    values: { int32: { gte: 0, lte: 100 } }
  }];
}
```

**注意：** `map.keys` 和 `map.values` 中不需要 `required`。

### Duration

```protobuf
message DurationExamples {
  // 精确值
  google.protobuf.Duration exact = 1 [(buf.validate.field).duration.const = "5s"];

  // 范围
  google.protobuf.Duration timeout = 2 [(buf.validate.field).duration = {
    gte: { seconds: 1 },
    lte: { seconds: 3600 }
  }];

  // 小于
  google.protobuf.Duration max = 3 [(buf.validate.field).duration.lt = { seconds: 60 }];

  // 白名单
  google.protobuf.Duration allowed = 4 [(buf.validate.field).duration.in = ["1s", "5s", "10s"]];
}
```

Duration 字符串格式：`"1h30m"`, `"5s"`, `"100ms"`, `"500us"`, `"10ns"`

### Timestamp

```protobuf
message TimestampExamples {
  // 范围
  google.protobuf.Timestamp created_at = 1 [(buf.validate.field).timestamp = {
    gte: { seconds: 1672444800 },  // 2023-01-01
    lt: { seconds: 1672531200 }    // 2023-01-02
  }];

  // 必须在过去
  google.protobuf.Timestamp past = 2 [(buf.validate.field).timestamp.lt_now = true];

  // 必须在未来
  google.protobuf.Timestamp future = 3 [(buf.validate.field).timestamp.gt_now = true];

  // 在当前时间的某个范围内
  google.protobuf.Timestamp recent = 4 [(buf.validate.field).timestamp = {
    gt_now: true,
    within: { seconds: 3600 }    // 1 小时内
  }];
}
```

### Any

```protobuf
message AnyExamples {
  google.protobuf.Any detail = 1 [(buf.validate.field).any = {
    in: ["type.googleapis.com/MyType1", "type.googleapis.com/MyType2"]
  }];
}
```

### FieldMask

```protobuf
message FieldMaskExamples {
  google.protobuf.FieldMask mask = 1 [(buf.validate.field).field_mask = {
    in: ["name", "email", "address.city"]
  }];
}
```

## 7. 消息级规则

### cel — 完整 CEL 规则

```protobuf
message ScheduleEventRequest {
  google.protobuf.Timestamp start_time = 1 [(buf.validate.field).required = true];
  google.protobuf.Timestamp end_time = 2 [(buf.validate.field).required = true];

  // 跨字段验证
  option (buf.validate.message).cel = {
    id: "end_after_start"
    message: "end_time must be after start_time"
    expression: "this.end_time > this.start_time"
  };
}
```

### cel_expression — 简写形式

当不需要自定义 `id` 和 `message` 时使用（id 自动等于 expression）：

```protobuf
message MyMessage {
  option (buf.validate.message).cel_expression = "this.foo > 42";
  option (buf.validate.message).cel_expression = "this.foo < 84";
  optional int32 foo = 1;
}
```

### 多条规则

```protobuf
message DateRange {
  string start_date = 1;
  string end_date = 2;

  option (buf.validate.message).cel = {
    id: "range.order"
    message: "start_date must be before end_date"
    expression: "this.start_date < this.end_date"
  };
  option (buf.validate.message).cel = {
    id: "range.max"
    message: "date range cannot exceed 90 days"
    expression: "this.end_date <= this.start_date + duration('2160h')"
  };
}
```

### 消息级 oneof 约束

比 proto 的 `oneof` 更灵活：允许 repeated/map 字段，允许隐式 presence 字段：

```protobuf
message MyMessage {
  // field1 和 field2 最多只能有一个
  option (buf.validate.message).oneof = { fields: ["field1", "field2"] };
  // field3 和 field4 必须恰好有一个
  option (buf.validate.message).oneof = { fields: ["field3", "field4"], required: true };

  string field1 = 1;
  bytes field2 = 2;
  bool field3 = 3;
  int32 field4 = 4;
}
```

### proto oneof 的 required

```protobuf
message MyMessage {
  oneof value {
    option (buf.validate.oneof).required = true;  // 必须设置一个
    string a = 1 [(buf.validate.field).string.min_len = 1];
    string b = 2;
  }
}
```

## 8. 自定义 CEL 规则

### 字段级自定义规则

```protobuf
message PlaceWholeSaleOrderRequest {
  uint32 quantity = 2 [(buf.validate.field).cel = {
    id: "minimum_whole_sale_quantity"
    message: "order quantity must be 100 or greater"
    expression: "this >= 100"    // this = 当前字段值
  }];
}
```

### 组合多个规则

```protobuf
message DeviceInfo {
  string hostname = 1 [
    (buf.validate.field).string.min_len = 1,      // 标准规则
    (buf.validate.field).cel = {
      id: "hostname.valid"
      message: "hostname must be valid"
      expression: "this.isHostname()"             // CEL 自定义
    },
    (buf.validate.field).cel = {
      id: "hostname.not_localhost"
      message: "localhost is not permitted"
      expression: "this != 'localhost'"            // 另一个 CEL 规则
    }
  ];
}
```

### 消息级自定义规则

`this` 指向整个消息：

```protobuf
message IndirectFlightRequest {
  option (buf.validate.message).cel = {
    id: "trip.duration.maximum"
    message: "the entire trip must be less than 48 hours"
    expression:
      "this.first_flight_duration"
      "+ this.second_flight_duration"
      "+ this.layover_duration < duration('48h')"
  };
  google.protobuf.Duration first_flight_duration = 1;
  google.protobuf.Duration layover_duration = 2;
  google.protobuf.Duration second_flight_duration = 3;
}
```

## 9. CEL 表达式详解

### 变量

| 变量 | 可用范围 | 说明 |
|------|---------|------|
| `this` | 所有规则 | 字段规则中 = 字段值；消息规则中 = 消息本身 |
| `now` | 仅预定义规则 | 当前时间戳（每次表达式只计算一次） |
| `rule` | 仅预定义规则 | 预定义规则的赋值 |
| `rules` | 仅预定义规则 | 底层规则消息实例 |

### Protobuf 到 CEL 类型映射

比较 `this` 与字面量时必须使用正确的类型后缀：

| Protobuf 类型 | CEL 类型 | 字面量示例 |
|--------------|---------|----------|
| `bool` | `bool` | `true`, `false` |
| `int32`, `int64`, `sint32`, `sint64` | `int` | `1`, `-42`, `0` |
| `uint32`, `uint64`, `fixed32`, `fixed64` | `uint` | `1u`, `0u`, `100u` |
| `float`, `double` | `double` | `1.0`, `-3.14`, `0.0` |
| `string` | `string` | `"hello"`, `'world'` |
| `bytes` | `bytes` | `b"hello"`, `b'\x00'` |
| `enum` | `int` | `0`, `1`, `2` |
| `repeated T` | `list` | `[1, 2, 3]` |
| `map<K, V>` | `map` | `{"key": "value"}` |
| `google.protobuf.Duration` | `duration` | `duration("1h30m")` |
| `google.protobuf.Timestamp` | `timestamp` | `timestamp("2024-01-01T00:00:00Z")` |
| Wrapper 类型 | nullable | `null`（未设置时） |

**类型转换函数：** `int(x)`, `uint(x)`, `double(x)`, `string(x)`, `bytes(x)`

### 运算符

| 运算符 | 说明 | 示例 |
|--------|------|------|
| `==`, `!=` | 相等 | `this == "active"` |
| `<`, `<=`, `>`, `>=` | 比较 | `this >= 1u && this <= 100u` |
| `&&` | 逻辑 AND（短路） | `size(this) >= 3 && size(this) <= 50` |
| `\|\|` | 逻辑 OR（短路） | `this == "admin" \|\| this == "editor"` |
| `!` | 逻辑 NOT | `!this.contains("test")` |
| `+` | 加法/字符串拼接/列表拼接 | `this.first + " " + this.last` |
| `in` | 列表/map 成员 | `this in ["USD", "EUR"]` |
| `? :` | 三元条件 | `this > 0 ? "" : "must be positive"` |

### 常用函数

| 函数 | 说明 | 示例 |
|------|------|------|
| `size()` | 字符串/字节/列表/map 长度 | `size(this) > 0` |
| `contains()` | 字符串包含子串 | `this.contains("@")` |
| `startsWith()` | 字符串前缀 | `this.startsWith("https://")` |
| `endsWith()` | 字符串后缀 | `this.endsWith(".com")` |
| `matches()` | RE2 正则匹配 | `this.matches("^[a-z]+$")` |
| `has()` | 字段是否设置 | `has(this.nickname)` |
| `all()` | 列表所有元素满足条件 | `this.all(x, x > 0)` |
| `exists()` | 列表至少一个满足 | `this.exists(x, x > 100)` |
| `exists_one()` | 列表恰好一个满足 | `this.exists_one(p, p.is_captain)` |
| `filter()` | 过滤列表元素 | `this.filter(x, x > 0)` |
| `map()` | 转换列表元素 | `this.map(a, a.name)` |
| `unique()` | 列表元素是否唯一 | `this.unique()` |

### Protovalidate 扩展函数

| 函数 | 适用类型 | 说明 |
|------|---------|------|
| `isEmail()` | string | 有效邮箱 |
| `isHostname()` | string | 有效主机名 |
| `isIp()` / `isIp(4)` / `isIp(6)` | string | IP 地址 |
| `isIpPrefix()` / `isIpPrefix(int, bool)` | string | IP 前缀 |
| `isUri()` | string | 有效绝对 URI |
| `isUriRef()` | string | 有效 URI（含相对） |
| `isHostAndPort(bool)` | string | 主机:端口，bool=是否要求端口 |
| `isNan()` | double | 是否 NaN |
| `isInf()` | double, int | 是否无穷 |

## 10. CEL 实用模式

### 字符串和数值

```protobuf
// 长度检查
expression: "size(this) >= 3 && size(this) <= 50"

// 跨字段子串检查
expression: "this.callback_url.startsWith(this.allowed_origin)"

// 跨字段正则（根据国家选择不同格式）
expression:
  "(this.country != 'US' || this.tax_id.matches('^[0-9]{2}-[0-9]{7}$'))"
  "&& (this.country != 'GB' || this.tax_id.matches('^[0-9]{9,10}$'))"

// 数值比较（注意 uint 后缀 u）
expression: "this.min_bedrooms <= this.max_bedrooms"
```

### 集合操作

```protobuf
// 列表成员检查
expression: "this.primary_contact in this.team_members"

// 所有元素满足条件
expression: "this.all(email, email.contains('@'))"

// 至少一个满足
expression: "this.exists(p, p.is_captain)"

// 过滤 + 计数
expression: "this.filter(a, a.size() > 0).size() >= 3"

// Map 遍历（all 遍历 key）
expression: "this.all(key, this[key].size() > 0)"

// 转换 + 唯一性（嵌套消息属性唯一）
expression: "this.map(it, it.product_id + '-' + string(it.unit_price)).unique()"

// 字段存在检查（可选字段）
expression: "!has(this.nickname) || this.nickname.size() >= 2"
```

### 逻辑和条件

```protobuf
// 条件字段验证：可选字段如果设置则必须满足条件
expression: "!has(this.nickname) || (this.nickname.size() >= 2 && this.nickname.size() <= 30)"

// 三元运算符
expression: "this.amount <= this.balance ? '' : 'insufficient balance'"

// 动态错误消息（返回 string 而非 bool）
expression:
  "this.amount <= this.balance ? ''"
  ": 'cannot transfer ' + string(this.amount)"
  "  + ' with a balance of ' + string(this.balance)"
```

### Timestamp 和 Duration

```protobuf
// 时间范围
expression: "timestamp('1800-01-01T00:00:00+00:00') <= this && this < timestamp('1900-01-01T00:00:00+00:00')"

// 时间属性
expression: "this.getDayOfWeek() != 1"  // 不允许周一

// Duration 加法
expression: "this.first_flight_duration + this.second_flight_duration + this.layover_duration < duration('48h')"

// 时间差（必须提前 24 小时预订）
expression: "duration('24h') <= this - now"

// 时间 + Duration = 新时间
expression: "this.departure_time + this.duration == this.arrival_time"
```

### 防止运行时错误

**整数除零：** 用短路 `&&` 防护：
```protobuf
expression: "this.count != 0u && this.total / this.count >= 10u"
```

**列表越界：** 先检查 size：
```protobuf
expression: "size(this.tags) > 0 && this.tags[0] == 'primary'"
```

**Map key 不存在：** 用 `in` 检查：
```protobuf
expression: "'env' in this.labels && this.labels['env'] == 'production'"
```

**Wrapper 类型 null：** 用 `has()` 检查：
```protobuf
expression: "!has(this.display_name) || this.display_name.size() >= 3"
```

**三元 fallback：**
```protobuf
expression: "this.count != 0u ? this.total / this.count : 0u"
expression: "has(this.nickname) ? this.nickname : this.full_name"
```

### 嵌套消息验证

```protobuf
// 深层字段存在检查
expression: "has(this.contact.full_name.first_name)"

// repeated message 的属性验证
expression: "this.all(m, m.is_verified)"

// repeated message 属性唯一
expression: "this.map(author, author.name).unique()"

// 列表拼接 + 成员检查
expression: "this.daily_special in this.hot_beverages + this.cold_beverages"
```

## 11. CEL 调试技巧

### 使用 cel_expression 快速原型

`cel_expression` 是 `cel` 的简写形式，id 自动等于 expression 本身。适合快速测试验证逻辑：

```protobuf
// 开发调试时先用 cel_expression 快速验证
option (buf.validate.message).cel_expression = "this.start_time < this.end_time";

// 确认逻辑正确后，再升级为带 id/message 的完整形式
option (buf.validate.message).cel = {
  id: "time_range.valid"
  message: "start_time must be before end_time"
  expression: "this.start_time < this.end_time"
};
```

### 常见 CEL 错误排查

| 错误现象 | 可能原因 | 解决方案 |
|---------|---------|---------|
| `no such overload` | 类型不匹配 | 检查类型后缀：uint 加 `u`，double 加 `.0` |
| 运行时 panic | 整数除零、列表越界 | 添加防护：`this.count != 0u && ...` |
| 验证结果不符合预期 | `this` 指向错误 | 字段规则中 `this` = 字段值，消息规则中 `this` = 消息 |
| optional 字段始终验证失败 | 零值被验证 | 加 `ignore: IGNORE_IF_ZERO_VALUE` 或用 `has()` 检查 |
| repeated 元素验证不生效 | 语法错误 | 用 `repeated.items` 嵌套规则，不要在 repeated 字段上用字段级 CEL |

### 逐步构建复杂 CEL

将复杂表达式拆分为多条简单规则，每条有独立的 id 和 message：

```protobuf
// 不要写一条巨大的 CEL
// 推荐：拆分为多条可独立定位的规则
option (buf.validate.message).cel = {
  id: "order.quantity_positive"
  message: "quantity must be positive"
  expression: "this.quantity > 0u"
};
option (buf.validate.message).cel = {
  id: "order.total_matches"
  message: "total must equal quantity * unit_price"
  expression: "this.total == this.quantity * this.unit_price"
};
```

## 12. 性能考量

Protovalidate 的验证发生在应用进程内（非代理层），以下建议适用于高吞吐热点接口：

### 标准规则 vs CEL

- **标准规则**（如 `string.email`、`int32.gte`）编译为高效的检查代码，性能开销极小
- **CEL 表达式**需要解析和求值，复杂表达式的开销比标准规则高 10-100 倍
- 单次验证通常在微秒级，对普通 API 调用无感知；但对每秒处理百万级的 RPC，CEL 开销可能显著

### 优化建议

| 场景 | 建议 |
|------|------|
| 热点接口 + 简单验证 | 用标准规则，不用 CEL |
| 热点接口 + 复杂业务规则 | 在应用代码中用原生语言验证，protovalidate 只做格式/范围检查 |
| 普通接口 | 标准规则 + CEL 均可，可读性优先 |
| 批量操作（如导入 CSV） | 考虑跳过 protovalidate，用专用批量验证逻辑 |
| gRPC 拦截器 | 验证在拦截器中执行，不进入业务逻辑；热点接口可考虑在特定 RPC 上禁用 |

### Validator 复用

`protovalidate.New()` 会编译所有已知的验证规则。在服务启动时创建一次，全局复用，**不要每次请求创建新的 Validator**：

```go
// 正确：全局创建一次
var validator *protovalidate.Validator

func init() {
    var err error
    validator, err = protovalidate.New()
    if err != nil {
        log.Fatal(err)
    }
}

// 错误：每次请求创建
// validator, err := protovalidate.New() // 不要这样做
```

## 13. 预定义规则（可复用）

当相同的规则组合在多个字段重复出现时，提取为预定义规则。

### 定义

创建 `proto2` 或 `Edition 2023` 语法的文件：

```protobuf
// predefined_string_rules.proto
syntax = "proto2";

package acme.common.v1;

import "buf/validate/validate.proto";

extend buf.validate.StringRules {
  // 简单布尔规则
  optional bool long_name = 50000 [(buf.validate.predefined).cel = {
    id: "string.long_name"
    message: "value must have between 1 and 50 characters"
    expression: "this.size() > 0 && this.size() <= 50"
  }];

  // 参数化规则（使用 rule 变量访问赋值）
  optional int32 name_component = 50001 [(buf.validate.predefined).cel = {
    id: "string.name_component"
    expression:
      "(this.size() > 0 && this.size() <= rule)"
      "? ''"
      ": 'value must have between 1 and ' + string(rule) + ' characters'"
  }];
}
```

**编号规则：**
- 私有 schema：50000-99999
- 公开 schema：在 Protovalidate Extension Registry 注册
- 不同规则类型可以复用相同编号（`FloatRules` 的 50100 和 `Int32Rules` 的 50100 不冲突）

### 使用

在 `proto3` 文件中导入并使用：

```protobuf
// person.proto
syntax = "proto3";

package acme.common.v1;

import "buf/validate/validate.proto";
import "acme/common/v1/predefined_string_rules.proto";

message Person {
  // 布尔规则
  string given_name = 1 [(buf.validate.field).string.(long_name) = true];

  // 参数化规则（传入最大长度）
  string family_name = 2 [(buf.validate.field).string.(name_component) = 50];
  optional string title = 3 [(buf.validate.field).string.(name_component) = 25];

  // 组合预定义规则 + 标准规则
  string email_local = 4 [(buf.validate.field).string = {
    [name_component]: 100,
    not_contains: "@"
  }];
}
```

### 与标准规则冲突解决

用 `rules` 变量检查标准规则是否存在：

```protobuf
extend buf.validate.StringRules {
  optional int32 name_component = 50001 [(buf.validate.predefined).cel = {
    id: "string.name_component"
    expression:
      "("
      "  (has(rules.min_len) ? true : this.size() > 0) && "
      "  this.size() <= rule"
      ")"
      "? ''"
      ": 'value must have between ' + "
      "  string(has(rules.min_len) ? rules.min_len : 1u) + ' and ' + "
      "  string(rule) + ' characters'"
  }];
}
```

## 14. Go 运行时集成

### 基本验证

```go
import (
    "buf.build/go/protovalidate"
    pb "github.com/acme/pet-store/gen/go/acme/pet/v1"
)

validator, err := protovalidate.New()
if err != nil {
    log.Fatal(err)
}

pet := &pb.CreatePetRequest{Name: "Buddy"}

if err := validator.Validate(pet); err != nil {
    // ValidationError 包含结构化信息
    var valErr *protovalidate.ValidationError
    if errors.As(err, &valErr) {
        for _, v := range valErr.Violations {
            fmt.Printf("field: %s, rule: %s, message: %s\n",
                v.FieldPath, v.RuleID, v.Message)
        }
    }
}
```

### 一行验证

```go
if err := protovalidate.Validate(message); err != nil {
    // Handle failure.
}
```

### gRPC 拦截器

```go
import (
    "buf.build/go/protovalidate"
    protovalidate_middleware "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/protovalidate"
    "google.golang.org/grpc"
)

func main() {
    validator, err := protovalidate.New()
    if err != nil {
        log.Fatal(err)
    }

    // 服务端拦截器
    s := grpc.NewServer(
        grpc.UnaryInterceptor(protovalidate_middleware.UnaryServerInterceptor(validator)),
        grpc.StreamInterceptor(protovalidate_middleware.StreamServerInterceptor(validator)),
    )

    // 验证失败自动返回 INVALID_ARGUMENT
}
```

安装拦截器：
```bash
go get github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/protovalidate
```

### 测试验证规则

```go
type violationSpec struct {
    ruleID    string
    fieldPath string
    message   string
}

func TestCreatePet(t *testing.T) {
    testCases := map[string]struct {
        producer   func(*pb.CreatePetRequest) *pb.CreatePetRequest
        violations []violationSpec
    }{
        "valid request": {
            producer: func(req *pb.CreatePetRequest) *pb.CreatePetRequest { return req },
        },
        "name is required": {
            producer: func(req *pb.CreatePetRequest) *pb.CreatePetRequest {
                req.Name = ""
                return req
            },
            violations: []violationSpec{{
                ruleID:    "string.min_len",
                fieldPath: "name",
                message:   "value must be at least 1 character(s)",
            }},
        },
    }
    // ... run tests
}
```

## 15. 完整示例

```protobuf
syntax = "proto3";

package acme.invoice.v1;

import "buf/validate/validate.proto";
import "google/protobuf/timestamp.proto";

message Invoice {
  // UUID 格式
  string invoice_id = 1 [(buf.validate.field).string.uuid = true];

  string account_id = 2 [(buf.validate.field).string.min_len = 1];

  google.protobuf.Timestamp invoice_date = 3 [(buf.validate.field).required = true];

  // 至少一个 LineItem，且 product_id + unit_price 组合唯一
  repeated LineItem line_items = 4 [
    (buf.validate.field).repeated.min_items = 1,
    (buf.validate.field).cel = {
      id: "line_items.unique"
      message: "line items must have unique product_id and unit_price"
      expression: "this.map(it, it.product_id + '-' + string(it.unit_price)).unique()"
    }
  ];
}

message LineItem {
  string product_id = 1 [(buf.validate.field).string.min_len = 1];
  uint64 quantity = 2 [(buf.validate.field).uint64.gt = 0];
  uint64 unit_price = 3 [(buf.validate.field).uint64.gt = 0];
}
```
