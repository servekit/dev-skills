---
name: gorm-cli-development
description: MUST use when writing, editing, or reviewing Go code that touches the database layer in projects using gorm.io/cli — defining models, generating code, implementing the dal layer, or writing type-safe CRUD/transactions.
---

# GORM CLI 开发指南

基于 [GORM CLI 官方文档](https://gorm.io/cli/index.html) 蒸馏，覆盖 **如何用 gorm/cli 生成代码** 与 **如何写类型安全的数据库操作** 两件事。

## 核心原则

1. **类型安全优先** — 所有 DB 操作必须使用 generated 的类型辅助器（`generated.User.ID.Eq(1)`），手写条件字符串（`Where("id = ?", 1)`）只有在 generated 无法表达时才允许，并且要在代码注释里写清楚原因。
2. **代码生成而非手写** — `gorm gen` 一次性产出字段辅助器与查询方法，`store/generated/` 下的代码不要手改。
3. **Model 即 schema** — `store/models/` 里的 struct 既驱动代码生成，也驱动 migration，所以 tag 要写完整。

## 1. 项目结构

每个使用数据库的服务统一使用如下结构。**外层目录名项目自定**（可能是 `cmd/api/`、`internal/user/`、`app/`、`<service-name>/` 等），下面的 `store/` 与 `service/` 两个目录的名字与平级关系是固定的。图里的 `<service-root>/` 是占位符，**不是真实目录名**。

```
<service-root>/                 # 外层目录名项目自定
├── service/                    # service-layer Go package：业务逻辑、开事务
│   └── login.go
└── store/                      # DB 访问层（目录名固定为 store）
    ├── models/                 # 手写：表结构定义（一个文件 = 一张表）
    │   ├── user.go
    │   └── login_log.go
    ├── generated/              # gorm gen 生成（禁止手改，会被覆盖）
    │   ├── user.gen.go
    │   ├── login_log.gen.go
    │   └── query.gen.go
    └── dal/                    # 手写：业务数据访问层（一个文件 = 一个 model 文件）
        ├── user.go
        └── login_log.go
```

`<service-root>/` 是占位符，**实际项目里换成你自己的目录名**。`store/` 和 `service/` 这两个名字和它们之间的平级关系不要改。

约定：

- `models/` 与 `dal/` 文件名一一对应：`models/login_log.go` ↔ `dal/login_log.go`。
- `dal/` 文件只写自己对应那张表的操作。**跨表交互放在更上层（service/biz）通过组合多个 dal 实现**，不要在一个 dal 文件里 join 多张表。
- `generated/` 要进 git、不要手改；改完 model 后跑 `gorm gen` 重新生成、把变更一起提交。

### 参考骨架

完整可复制的最小项目骨架位于本 skill 同目录下的 [`skeleton/`](./skeleton/)。骨架的 `skeleton/` 目录本身扮演上面图里的 `<service-root>/` 占位符 —— 复制到真实项目时，把 `skeleton/` 里的内容（`store/`、`service/`、`go.mod`、`Makefile`）放到你自己项目的服务根目录下，**不要把 `skeleton/` 这个名字也复制过去**。骨架里的 [`skeleton/README.md`](./skeleton/README.md) 列出了每个文件示范的是哪条约定，以及替换 module path / 服务名时需要改哪里。**给新服务搭数据库层时优先复制骨架再改**，而不是从空白目录开始。

## 2. CLI 工作流

### 安装

```bash
go install gorm.io/cli/gorm@latest
```

### 生成命令

```bash
# 在服务根目录（<service-root>/）下执行
gorm gen -i ./store/models -o ./store/generated
```

参数：

| 参数 | 含义 |
|------|------|
| `-i` | 输入目录，包含 model（和 interface）的包路径 |
| `-o` | 输出目录，生成的代码会落到这里 |

默认输出严格类型化的泛型 API（`gorm.G[T]` + generated 字段辅助器）。**不要用 `--typed=false`** —— 它会回退到允许混写原生 SQL 的标准 API，违背类型安全原则。

何时跑：

- 新增/修改 model 后必须重新跑
- 拉 main 后如果 `generated/` 有冲突，重新跑而不是手动 merge
- CI 流水线里跑一次 + `git diff --exit-code` 确保提交的 generated 与 models 同步

## 3. Model 定义规范

### 显式声明四个标准字段

每个 model 显式声明 `ID`、`CreatedAt`、`UpdatedAt`、`DeletedAt` 四个字段。**不要嵌入 `gorm.Model`**——它把 `ID` 类型硬编码为 `uint`，遇到雪花 ID（`int64`）、UUID（`string`）、复合主键时必须绕开，还会触发 GORM schema 解析的同名字段冲突（见 [go-gorm/gorm#6517](https://github.com/go-gorm/gorm/issues/6517)）。统一显式声明更简单，主键类型想换就换。

```go
package models

import (
    "time"

    "gorm.io/gorm"
)

type User struct {
    ID        uint           `gorm:"primaryKey"` // 主键，类型按需选
    Name      string
    Age       int
    CreatedAt time.Time                          // 创建时 GORM 自动写入
    UpdatedAt time.Time                          // 创建/更新时 GORM 自动写入
    DeletedAt gorm.DeletedAt `gorm:"index"`      // 软删除
}
```

| 字段 | 行为 |
|------|------|
| `ID <T>` | 主键。`<T>` 自由选择：`uint`、`int64`、`string` 都行；整型默认 `autoIncrement`，赋非零值时 GORM 用你的值 |
| `CreatedAt time.Time` | 创建时自动写入 |
| `UpdatedAt time.Time` | 创建/更新时自动写入 |
| `DeletedAt gorm.DeletedAt` | 软删除：`Delete()` 设置此字段，查询自动过滤 `deleted_at IS NULL` |

`CreatedAt` / `UpdatedAt` 靠**字段名约定**自动识别（GORM 看到这两个名字 + `time.Time` 类型就当时时间戳字段），无需 tag。

默认精度按数据库而异（**MySQL 和 PostgreSQL 不同**，换库会踩坑）：

| 数据库 | GORM 默认列类型 | 实际精度 | 是否 round NowFunc |
|--------|----------------|---------|-------------------|
| MySQL | `datetime(3)` | **毫秒**（3 位小数） | 是，round 到毫秒 |
| PostgreSQL | `timestamptz` | **微秒**（PG 默认 6 位） | 否 |

源码出处：`gorm.io/driver/mysql` 里 `defaultDatetimePrecision = 3` + `NowFunc` 用 `time.Now().Round(time.Millisecond)`；`gorm.io/driver/postgres` 不设精度，列类型直接 `timestamptz`，交给 PG 默认。MySQL 5.6 之前 driver 自动关掉精度（`DisableDatetimePrecision: true`），回退到秒。

**MySQL 上想要更高精度**（微秒/纳秒）必须改 driver 配置——单靠 `autoCreateTime:nano` 没用，因为 `NowFunc` 已经 round 到毫秒了：

```go
// 启动时调高 driver 的默认精度（最高 6 = 微秒，MySQL DATETIME 最大支持 6 位）
db, _ := gorm.Open(mysql.New(mysql.Config{
    DSN:                       dsn,
    DefaultDatetimePrecision:  ptrInt(6), // ← 关键
}), &gorm.Config{})
```

要纳秒精度只能改用 `int64` 字段（MySQL `DATETIME` 最大就是 6 位小数）：

```go
CreatedNs int64 `gorm:"autoCreateTime:nano"` // unix 纳秒，存 BIGINT
```

`gorm.DeletedAt` 是个特殊类型（不是普通 `time.Time`），GORM 通过类型识别并启用软删除语义，也无需 tag；`gorm:"index"` 是为了给 `deleted_at` 加索引，加速软删除过滤查询。

**字段顺序**：`ID` 在最前 → 业务字段 → `CreatedAt/UpdatedAt/DeletedAt` 放最后。元数据下沉到底部，阅读 struct 时第一眼看到的是业务字段。

主键类型按需选：

```go
// 雪花 ID（应用层生成，BIGINT）
type Order struct {
    ID        int64          `gorm:"primaryKey"`
    // ...业务字段
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"`
}

// UUID（应用层生成，CHAR(36)）
type Session struct {
    ID        string         `gorm:"primaryKey;size:36"`
    // ...业务字段
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"`
}
```

### 不要复述已由标准字段表达的语义

`CreatedAt` / `UpdatedAt` / `DeletedAt` 已经覆盖"创建时间 / 修改时间 / 是否软删"三个最常见语义。**不要再加冗余字段表达同一件事**：

| ❌ 冗余字段 | ✅ 复用标准字段 |
|------------|---------------|
| `CreatedTime`、`CreatedDate` | `CreatedAt` |
| `ModifiedAt`、`LastModified`、`UpdateTime` | `UpdatedAt` |
| `IsDeleted bool`、`Status = "deleted"` | `DeletedAt`（`IS NULL` 表达"未删"） |
| `IsActive bool`（当语义等价于"未软删"时） | `DeletedAt` |

**业务时间戳不算冗余**：`LastLoginAt`、`PublishedAt`、`ShippedAt` 这类表达具体业务事件的时间点，跟"行创建/修改/软删"无关，该加就加。

### Tag 是契约

Model 既驱动生成代码，也驱动 migration（`db.AutoMigrate(&models.User{})`）。所有需要表达给数据库的语义都必须用 tag 写出来：

```go
type User struct {
    ID        uint           `gorm:"primaryKey"`
    Name   string `gorm:"column:name;size:64;not null;index"`
    Email  string `gorm:"uniqueIndex;size:128"`
    Age    int    `gorm:"default:18"`
    Score  int64  `gorm:"default:0"`
    Status string `gorm:"default:active;size:16"`
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"`
}
```

常用 tag：

| Tag | 用途 |
|-----|------|
| `column:xxx` | 自定义列名 |
| `size:N` | 字符串/字节数组长度 |
| `primarykey` | 自定义主键 |
| `default:xxx` | 默认值 |
| `not null` | 非空约束 |
| `index` / `uniqueIndex` | 索引（可命名：`index:idx_name`） |
| `autoCreateTime` / `autoUpdateTime` | 自定义时间戳列，支持 `:milli` / `:nano` |
| `foreignkey:xxx` | 关联外键字段 |
| `many2many:join_table` | 多对多中间表 |

Unix 时间戳列：

```go
type Event struct {
    ID        uint           `gorm:"primaryKey"`
    Created   int64 `gorm:"autoCreateTime"`       // unix 秒
    Updated   int64 `gorm:"autoUpdateTime:milli"` // unix 毫秒
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"`
}
```

### 命名约定

| 对象 | 规则 | 示例 |
|------|------|------|
| 文件名 | 表名蛇形小写 | `login_log.go` |
| struct 名 | **服务前缀 + 表名（驼峰）** | `UserLoginLog` |
| 表名（DB） | struct 名的蛇形复数 | `user_login_logs` |
| 列名 | 字段名蛇形 | `CreatedAt` → `created_at` |

**服务前缀的目的是避免数据库内表重名**。例如当前是 `user` 服务、对应的表语义是 `login_log`，那么：

- struct：`UserLoginLog`
- 文件：`store/models/login_log.go`（**文件名不加前缀**）
- 表名：自动推导为 `user_login_logs`（GORM 蛇形复数）

如需自定义表名：

```go
func (UserLoginLog) TableName() string { return "user_login_logs" }
```

### 支持的字段类型

| Go 类型 | 生成的辅助器 | 典型谓词 |
|---------|------------|---------|
| `int{8..64}`, `uint{8..64}`, `float{32,64}` | Number | `Eq` / `Gt` / `Between` / `In` / `Incr` |
| `string` | String | `Eq` / `Like` / `Regexp` / `In` / `Concat` |
| `bool` | Bool | `Eq` / `Neq` |
| `time.Time` | Time | `Eq` / `Gt` / `Between` / `Add` / `Now` |
| `[]byte` | Bytes | `Eq` / `In` |
| `*T`、`sql.NullXXX` | 对应类型 + `IsNull` / `IsNotNull` | |
| 关联字段（struct/slice） | Association | `Create` / `Update` / `Unlink` / `Delete` |

自定义类型（实现 `sql.Scanner`/`driver.Valuer`）需要在 `models` 包的 `genconfig.Config` 里通过 `FieldTypeMap` 显式映射，否则生成不出可用辅助器。详情见 [Field Helpers 文档](https://gorm.io/cli/field_helpers.html)。

## 4. 类型安全 CRUD

下面所有示例假设 `ctx context.Context`、`db *gorm.DB` 已就绪，并 `import` 了 `gorm` 和项目里的 `generated` 包。

### 增

```go
// 单条
user := &models.User{Name: "alice", Age: 18}
err := gorm.G[models.User](db).Create(ctx, user)
// user.ID 已回填

// 批量
users := []*models.User{{Name: "a"}, {Name: "b"}}
err := gorm.G[models.User](db).Create(ctx, users...)
```

字段级零值创建（struct 无法表达"显式设置为零值"，用 Set）：

```go
err := gorm.G[models.User](db).
    Set(
        generated.User.Name.Set("alice"),
        generated.User.Age.Set(0),            // 显式零值也会写入
        generated.User.Status.Set("active"),
    ).
    Create(ctx)
```

### 删

```go
// 按 ID 删
err := gorm.G[models.User](db).
    Where(generated.User.ID.Eq(1)).
    Delete(ctx)

// 按条件删（软删除：写 deleted_at）
err := gorm.G[models.User](db).
    Where(generated.User.Status.Eq("inactive")).
    Delete(ctx)
```

软删除由 `gorm.DeletedAt` 字段类型启用，每个 model 显式声明后即可使用。

### 改

**特定字段更新**（推荐，类型安全、避免覆盖整行）：

```go
err := gorm.G[models.User](db).
    Where(generated.User.Name.Eq("alice")).
    Set(
        generated.User.Name.Set("jinzhu"),
        generated.User.IsAdult.Set(false),
        generated.User.Score.Set(sql.NullInt64{}),
        generated.User.Count.Incr(1),
    ).
    Update(ctx)
```

**全字段更新**（用 struct）：

```go
user := models.User{Name: "alice", Age: 20}
err := gorm.G[models.User](db).
    Where(generated.User.Name.Eq("alice")).
    Update(ctx, user)
```

注意：`Update(ctx, struct)` 会**忽略零值字段**（GORM 默认行为）。如果需要写入零值，回到 `Set(...)` 形式。

### 查

```go
// 多条
users, err := gorm.G[models.User](db).
    Where(generated.User.Age.Gt(18)).
    Find(ctx)

// 单条（不存在返回 ErrRecordNotFound）
user, err := gorm.G[models.User](db).
    Where(generated.User.ID.Eq(1)).
    Take(ctx)

// 排序 + 分页
users, err := gorm.G[models.User](db).
    Where(generated.User.Age.Gt(18)).
    Order(generated.User.CreatedAt.Desc()).
    Offset(0).Limit(20).
    Find(ctx)
```

### Select 用类型辅助器

不要手写列名字符串，用 `Column().Name`：

```go
users, err := gorm.G[models.User](db).
    Select(
        generated.User.ID.Column().Name,
        generated.User.Name.Column().Name,
    ).
    Where(generated.User.Age.Gt(18)).
    Find(ctx)
```

## 5. 类型辅助器速查

> 详细 API 见 [Field Helpers](https://gorm.io/cli/field_helpers.html)。

### 通用谓词

| 谓词 | 等价 SQL |
|------|---------|
| `.Eq(v)` | `= v` |
| `.Neq(v)` | `!= v` |
| `.IsNull()` | `IS NULL` |
| `.IsNotNull()` | `IS NOT NULL` |
| `.In(vs...)` | `IN (...)` |
| `.NotIn(vs...)` | `NOT IN (...)` |
| `.Between(a, b)` | `BETWEEN a AND b` |

### String 专属

```go
generated.User.Name.Eq("jinzhu")          // name = 'jinzhu'
generated.User.Name.Like("%jinzhu%")      // name LIKE '%jinzhu%'
generated.User.Name.NotLike("%test%")     // name NOT LIKE '%test%'
generated.User.Name.ILike("%Jin%")        // name ILIKE '%Jin%' (PostgreSQL)
generated.User.Name.Regexp("^[A-Z]")      // name REGEXP '^[A-Z]'
generated.User.Name.Between("A", "Z")     // name BETWEEN 'A' AND 'Z'

// 更新操作
generated.User.Name.Set("new")            // SET name = 'new'
generated.User.Name.Concat("_x")          // SET name = CONCAT(name, '_x')
generated.User.Name.Upper()               // SET name = UPPER(name)
```

### Number 专属

```go
generated.User.Age.Gt(18)                 // age > 18
generated.User.Age.Between(18, 65)        // age BETWEEN 18 AND 65
generated.User.Age.In(25, 30, 35)         // age IN (25,30,35)

// 更新操作
generated.User.Age.Set(20)                // SET age = 20
generated.User.Age.Incr(1)                // SET age = age + 1
generated.User.Age.Decr(1)                // SET age = age - 1
generated.User.Age.Mul(2)                 // SET age = age * 2
```

### Time 专属

```go
generated.User.CreatedAt.Gt(start)        // created_at > ?
generated.User.CreatedAt.Between(s, e)    // created_at BETWEEN ? AND ?
generated.User.DeletedAt.IsNull()         // deleted_at IS NULL

// 更新操作
generated.User.UpdatedAt.Set(now)         // SET updated_at = ?
generated.User.UpdatedAt.Now()            // SET updated_at = NOW()
generated.User.UpdatedAt.Add(time.Hour)   // SET updated_at = DATE_ADD(...)
```

> ⚠️ **不要在普通查询里手写 `Where(generated.X.DeletedAt.IsNull())`**。model 只要声明了 `gorm.DeletedAt` 字段，GORM 就自动给所有 `Find/Take/First/Count` 等查询追加 `WHERE deleted_at IS NULL`，再写一遍纯属冗余，还会让代码看起来像没启用软删除。`DeletedAt.IsNull()` 这个辅助器存在的意义是给 generated 内部 / `.Unscoped()` 场景用，**不是给业务查询加的**。要查包含已软删数据在内的全部行，正确做法是 `.Unscoped()`，而不是把 `IsNull()` 加回去。

### 复合条件

```go
// OR
gorm.G[User](db).Where(
    gorm.Or(
        generated.User.Age.Lt(18),
        generated.User.Age.Gt(65),
    ),
).Find(ctx)

// AND（多个 Where 默认就是 AND）
gorm.G[User](db).
    Where(generated.User.Age.Gt(18)).
    Where(generated.User.Status.Eq("active")).
    Find(ctx)
```

## 6. dal 层规范

`store/dal/` 是业务数据访问层，承担"把业务诉求翻译成 generated 类型安全调用"。

**规则**：

1. **文件命名与 models 一一对应**：`models/login_log.go` ↔ `dal/login_log.go`。
2. **单文件 = 单表**：`dal/login_log.go` 只能写 `LoginLog` 的操作，不要在里面组合 `User` 的更新。
3. **跨表辅助内容集中放 `dal/common.go`**：常量、共用查询参数 struct、错误变量、类型别名等不属于某张具体表的辅助内容，一律放 `dal/common.go` 一个文件。**不要新建 `constants.go`、`types.go`、`query_params.go`、`consts/` 等多个文件**——会破坏跟 `models/` 的一一对应关系。**跟单表强绑定的查询参数 struct**（如只服务于该表查询的 `LoginLogQuery`）跟着表走，放 `dal/<table>.go`。
4. **方法名带表名前缀**：`CreateLoginLog`、`GetUserByID`、`UpdateUserAge`。一个 service 通常有多张表共用同一个 `dal` 包，前缀让调用点（`dal.CreateUser` vs `dal.CreateLoginLog`）一眼区分目标表，也避免方法重名。
5. **方法语义清晰**：方法名表达意图（`CreateLoginLog`、`ListLoginLogByUserAndTimeRange`），调用方不需要知道用了哪些字段辅助器。
6. **接收 `ctx` 和 `*gorm.DB`（或事务的 tx）**：dal 不负责连接管理，由调用方传入。
7. **错误直接返回**：dal 不吞错误，由 service 层决定 wrap/转换。

示例：

```go
// store/dal/login_log.go
package dal

import (
    "context"
    "time"

    "gorm.io/gorm"

    "yourmod/store/generated"
    "yourmod/store/models"
)

// CreateLoginLog inserts a login log entry.
func CreateLoginLog(ctx context.Context, tx *gorm.DB, log *models.UserLoginLog) error {
    return gorm.G[models.UserLoginLog](tx).Create(ctx, log)
}

// ListLoginLogByUserAndTimeRange returns login logs of a user within [start, end).
func ListLoginLogByUserAndTimeRange(
    ctx context.Context,
    tx *gorm.DB,
    userID uint,
    start, end time.Time,
) ([]*models.UserLoginLog, error) {
    return gorm.G[models.UserLoginLog](tx).
        Where(generated.UserLoginLog.UserID.Eq(userID)).
        Where(generated.UserLoginLog.CreatedAt.Gte(start)).
        Where(generated.UserLoginLog.CreatedAt.Lt(end)).
        Order(generated.UserLoginLog.CreatedAt.Desc()).
        Find(ctx)
}
```

调用方在 service 层组合多张表的 dal：

```go
// service/login.go
type LoginService struct {
    db *gorm.DB
}

func NewLoginService(db *gorm.DB) *LoginService { return &LoginService{db: db} }

func (s *LoginService) RecordLogin(ctx context.Context, userID uint, /* ... */) error {
    return s.db.Transaction(func(tx *gorm.DB) error {
        if err := dal.UpdateUserLastLogin(ctx, tx, userID, time.Now()); err != nil {
            return err
        }
        return dal.CreateLoginLog(ctx, tx, &models.UserLoginLog{UserID: userID /* ... */})
    })
}
```

## 7. Typed Raw SQL

**官方文档**：<https://gorm.io/cli/sql_templates.html>

当 builder 表达不了查询（复杂聚合、动态条件拼接、子查询、跨方言 SQL），优先用 Typed Raw SQL —— 在 interface 方法上用 SQL 注释定义模板，`gorm gen` 生成类型安全的 Go 方法。**不要直接写裸 SQL 字符串拼到 `db.Raw()` 里**，那样会丢失类型安全。

### 何时用

- 动态条件拼接（`{{where}}` / `{{set}}`，可选参数组合）
- 复杂 SQL（子查询、窗口函数、聚合、UNION）
- 固定查询模式想做成可复用方法

### 模板语法要点

```go
// store/models/login_log_query.go
package models

import "time"

type LoginLogQuery[T any] interface {
    // SELECT * FROM @@table
    // {{where}}
    //   {{if userID > 0}} user_id = @userID {{end}}
    //   {{if !start.IsZero()}} AND created_at >= @start {{end}}
    //   {{if !end.IsZero()}}   AND created_at <  @end   {{end}}
    // {{end}}
    // ORDER BY created_at DESC
    Filter(userID uint, start, end time.Time) ([]T, error)
}
```

占位符：

| 语法 | 用途 |
|------|------|
| `@@table` | model 对应的表名 |
| `@@<name>` | 从参数引用的表名/列名（动态标识符） |
| `@<name>` | SQL 绑定参数 |
| `{{where}}` ... `{{end}}` | 智能去除多余 `AND`/`OR`/逗号 |
| `{{set}}` ... `{{end}}` | UPDATE 智能去除多余逗号 |
| `{{if}}/{{else if}}/{{else}}` | 条件分支 |
| `{{for}}` | 集合迭代 |

### 调用

```go
logs, err := generated.Query[models.UserLoginLog](db).
    Filter(ctx, userID, start, end)
```

`ctx context.Context` 不需要在方法签名里写，CLI 会自动注入。

## 8. 事务

**规则**：

1. **单条增删改不要开事务** —— GORM 单语句本身就是原子的。
2. **事务在 service 层开**，不在 dal 层开。dal 接收调用方传入的 `*gorm.DB`（可能是 `tx`）。
3. **`*gorm.DB` 通过构造函数注入到 service struct**（`type XService struct { db *gorm.DB }` + `NewXService(db)`），不要在每个方法签名里传。方法签名只暴露业务参数。
4. 用 `s.db.Transaction(func(tx *gorm.DB) error { ... })`，错误自动回滚。

```go
type OrderService struct {
    db *gorm.DB
}

func NewOrderService(db *gorm.DB) *OrderService {
    return &OrderService{db: db}
}

func (s *OrderService) Ship(ctx context.Context, orderID uint) error {
    return s.db.Transaction(func(tx *gorm.DB) error {
        // 用 generated 类型安全调用，传入 tx
        order, err := gorm.G[models.Order](tx).
            Where(generated.Order.ID.Eq(orderID)).
            Take(ctx)
        if err != nil {
            return err
        }

        return gorm.G[models.Order](tx).
            Where(generated.Order.ID.Eq(order.ID)).
            Set(
                generated.Order.Status.Set("shipped"),
                generated.Order.ShippedAt.Set(time.Now()),
            ).
            Update(ctx)
    })
}
```

不要这样写：

- ❌ 在 dal 里包事务：破坏 dal/service 边界，调用方无法控制事务粒度
- ❌ 单条更新包事务：无意义的开销
- ❌ 不传 ctx：丢失超时/追踪链路
- ❌ service 方法签名里传 `db *gorm.DB`：每个方法都暴露连接管理细节，应该构造函数注入到 struct

## 9. 关联操作

**官方文档**：<https://gorm.io/cli/tutorials_associations.html>

关联辅助器在 generated 包里以 `field.Struct[T]` / `field.Slice[T]` 出现，组合进 `Set(...)`：

```go
// Has Many：创建 user 时一起创建 pets
err := gorm.G[models.User](db).
    Set(
        generated.User.Name.Set("alice"),
        generated.User.Pets.Create(
            generated.Pet.Name.Set("fido"),
        ),
    ).
    Create(ctx)

// Has One：更新关联账号
err := gorm.G[models.User](db).
    Where(generated.User.ID.Eq(1)).
    Set(
        generated.User.Account.Update(
            generated.Account.LastLogin.Set(time.Now()),
        ),
    ).
    Update(ctx)
```

操作语义：

| 操作 | Has One / Has Many | Belongs To | Many2Many |
|------|---------------------|------------|-----------|
| `.Create` | 创建子行 + 关联 | 创建关联行 | 新增 join 行 |
| `.Update` | 更新子行 | 更新关联行 | — |
| `.Unlink` | 清空子表 FK | 清空父表 FK | 删 join 行（双方保留） |
| `.Delete` | 删除子行 | 删除关联行 | 删 join 行（双方保留） |

更多细节（多态、自引用、Many2Many 中间表自定义）见官方教程。

## 10. 最佳实践

### 推荐

- 所有 model 显式声明 `ID/CreatedAt/UpdatedAt/DeletedAt`，不嵌入 `gorm.Model`
- migration 用到的约束（`size`、`not null`、`index`、`default`）全部写到 tag 里
- CRUD 一律走 `gorm.G[T]` + generated 辅助器
- struct 更新出现零值被忽略 → 切换到 `Set(...)` 表达
- 复杂查询优先用 Typed Raw SQL（interface + SQL 注释），其次才是 `db.Raw()` 并写明原因
- 事务在 service 层开，单条不开
- CI 跑 `gorm gen` + `git diff --exit-code` 确保 generated 与 models 同步

### 避免

- ❌ 加 `IsDeleted`/`IsActive`/`CreatedTime`/`ModifiedAt` 等冗余字段 —— `DeletedAt`/`CreatedAt`/`UpdatedAt` 已表达同样语义；业务时间戳（`LastLoginAt` 等）除外
- ❌ 在普通查询里手写 `Where(generated.X.DeletedAt.IsNull())` —— model 声明 `gorm.DeletedAt` 字段后 GORM 已自动过滤软删行，重复加是冗余；要查包含软删行的数据用 `.Unscoped()`
- ❌ 手改 `store/generated/` 下任何文件
- ❌ 用 `Where("name = ?", "alice")` 这种字符串条件 —— generated 表达不了的，注释里写明原因
- ❌ `db.Raw()` 拼裸 SQL —— 用 Typed Raw SQL
- ❌ 用 `--typed=false` 退回标准 API —— 失去类型安全
- ❌ 在 `dal/` 下新建 `constants.go`/`types.go`/`query_params.go` 等多文件存常量/struct —— 跨表辅助内容一律放 `dal/common.go`；跟单表强绑定的查询 struct 跟着表走放 `dal/<table>.go`
- ❌ dal 文件里组合多张表操作 —— 上移到 service 层
- ❌ 在 dal 层开事务 —— service 层控制
- ❌ 单条写操作包事务 —— 无意义开销
- ❌ 忽略 `error` 返回值
