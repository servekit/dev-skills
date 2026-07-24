---
name: golang-development
description: MUST use when writing, editing, or reviewing ANY Go code (.go files). Go development guide based on Google Go Style Guide — covers naming, error handling, concurrency, testing, documentation, lint and format verification (gofmt, goimports, golangci-lint).
---

# Go 编码最佳实践

基于 [Google Go Style Guide](https://gocn.github.io/styleguide/docs/01-overview/) 提炼，适用于 AI 辅助 Go 开发。

## 核心原则

优先级从高到低：**清晰 > 简约 > 简洁 > 可维护性 > 一致性**

- **Clarity**：代码意图一目了然，不需要解释
- **Simplicity**：最简单的方案优先，避免过度抽象
- **Conciseness**：消除冗余，但不牺牲清晰
- **Maintainability**：方便未来修改
- **Consistency**：与项目现有风格保持一致

## 1. 命名规范

### 包名
- 全小写、单个单词、无下划线：`http`, `json`, `user`
- 避免信息重复：`user.User` → 直接用 `user.Info` 或 `user.Record`
- 不用 `util`, `common`, `base` 等泛化名称

### 变量和函数
- **MixedCaps**（不用下划线）：`maxLength`, `httpRequest`, `ReaderAt`
- 缩写词全大写或全小写：`HTTP`, `API`, `userID`, `tcpConn`
- 局部变量用短名：`r` for reader, `i` for index（作用域小时）
- 包级变量用描述性名称

### 常量
- 不强制全大写，遵循 MixedCaps：`maxRetries`, `DefaultTimeout`
- 未导出的枚举常量用类型前缀：`type Role int; roleAdmin`

### 接口
- 单方法接口用 `-er` 后缀：`Reader`, `Stringer`, `Formatter`
- 方法名与接口名语义一致

### Receiver
- 短且一致：`(r *Registry)`, `(s *Server)`
- 类型名 1-2 个字母：`(u *User)` 不要 `(user *User)`

### Getter
- 不加 `Get` 前缀：`Name()` 而非 `GetName()`
- Setter 可以用 `SetName()` 形式

### 函数和方法命名
- 名词式：`NewReader`, `NewScanner` — 构造函数包含返回类型名
- 动词式：`Write`, `Execute`, `Parse`
- `New` 构造函数应体现返回类型：`NewServer()` 返回 `*Server`

### 测试替身命名
- `Stub`：返回固定值，不做验证
- `Fake`：有真正的工作实现（如内存数据库）
- `Spy`：记录调用以便后续断言
- `Mock`：验证交互行为
- 包名用 `fake` / `mock` / `stub`，类型名用具体行为：`AlwaysCharges`, `FakeService`

### 遮蔽 vs 覆盖
- 遮蔽（shadowing）：内层作用域声明同名变量 — 小范围内可接受
- 覆盖（stomping）：对同作用域变量重新赋值 — 通常要避免
- 特别注意 `err` 遮蔽：用更具体的名称如 `parseErr`

### 避免重复的3种情况
- 包名 vs 符号名：`flag.Flag` 可以，但 `user.User` 应改为 `user.Record`
- 变量名 vs 类型名：`var client Client` 可接受，`var userClient UserClient` 冗余
- 外部上下文：从调用者角度判断是否冗余

## 2. 错误处理

### 返回错误
```go
// Good: 缩进错误处理路径
f, err := os.Open(path)
if err != nil {
    return fmt.Errorf("open config %s: %w", path, err)
}
defer f.Close()
```

### 错误字符串格式
- 不首字母大写、不以句号结尾：`"not found"` 而非 `"Not found."`
- 错误字符串是短语，不是完整句子

### 错误包装
- 用 `fmt.Errorf("context: %w", err)` 添加上下文
- `%w` 放在末尾：`fmt.Errorf("read config: %w", err)` 而非 `fmt.Errorf("%w: read config", err)`
- 库代码不用 `%w`（避免暴露内部实现），用 `fmt.Errorf("...: %v", err)`
- 避免冗余包装：上层已包含上下文时不要重复

### 带内错误（In-band errors）
- 优先返回 error 而非用特殊值表示失败（如 `-1`, `nil`, `""`）
- 如果无法避免，在文档中明确说明返回值的语义

### 结构化错误
- 简单情况用 sentinel error：`var ErrNotFound = errors.New("not found")`（包级别定义）
- 需要携带额外信息时用自定义错误类型实现 `error` 接口
- 用 `errors.Is()` 和 `errors.As()` 检查

### 避免 panic
- 库代码中绝不 panic，返回 error
- `Must` 前缀函数例外：`template.Must()`, `regexp.MustCompile()`
- 测试中可以用 `Must*` 函数简化 setup
- 程序初始化中的不可恢复错误可以用 `log.Exit`

### 错误日志
- 在错误最终处理处（通常是顶层 handler）记录日志
- 不要在每个 return 处都 log — 会造成重复日志
- 日志中不包含 PII（个人身份信息）
- 避免高频错误日志导致日志爆炸

## 3. 函数设计

### 参数
- 超过 3-4 个参数时用 Option Struct：
```go
type ServerOption struct {
    Host    string
    Port    int
    Timeout time.Duration
}
func NewServer(opt ServerOption) *Server { ... }
```

- 可选参数用 Variadic Option Pattern：
```go
type Option func(*Server)
func WithTimeout(d time.Duration) Option { return func(s *Server) { s.timeout = d } }
func NewServer(addr string, opts ...Option) *Server { ... }
```

### 返回值
- 有意义的命名返回值（尤其在 defer 中需要用时）
- 不要用裸 return（降低可读性）

### 接口设计
- 在使用方定义接口，而非实现方
- 保持接口小：单方法接口最灵活
- 不要为了 mock 而加接口 — 只在需要多态时才加

## 4. 并发

### Goroutine 生命周期
- 每个启动的 goroutine 都必须有明确的退出机制
- 用 context 取消或 done channel 控制生命周期
```go
ctx, cancel := context.WithCancel(parentCtx)
defer cancel()
go func() {
    select {
    case <-ctx.Done():
        return
    case result <-ch:
        // process
    }
}()
```

### Channel
- 声明方向：`chan<- int`（只写）和 `<-chan int`（只读）
- nil slice 和 nil map 用 `make` 初始化时带 size hint：
```go
ch := make(chan int, bufferCap)
s := make([]string, 0, estimatedSize)
```

### 不要启动不知道何时结束的 goroutine

## 5. 测试

### 结构
- 表驱动测试优先：
```go
tests := []struct {
    name    string
    input   string
    want    int
    wantErr bool
}{
    {name: "positive", input: "42", want: 42},
    {name: "negative", input: "-1", want: -1},
    {name: "invalid", input: "abc", wantErr: true},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got, err := Parse(tt.input)
        if (err != nil) != tt.wantErr {
            t.Fatalf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
        }
        if got != tt.want {
            t.Errorf("Parse(%q) = %d, want %d", tt.input, got, tt.want)
        }
    })
}
```

### Helper 函数
- 用 `t.Helper()` 标记测试辅助函数
- `t.Error` 用于非致命断言（继续执行），`t.Fatal` 用于无法继续的情况
- **辅助函数不应直接调用 `t.Error`/`t.Fatal`**，应返回错误让 Test 函数决定
- **不要在 goroutine 中调用 `t.Fatal`** — 通过 channel 传递错误回主 goroutine

### 断言与比较
- 使用 testify `require` 做控制流断言（如 `require.NoError`、`require.Error`、`require.Contains`）
- 使用 testify `assert` 做非致命断言（如 `assert.True`、`assert.False`）
- **结构体比较用 `cmp.Diff`**，不要逐字段比较，失败时打印完整差异
- 简单标量值（字符串、数字、布尔）可直接用 `require.Equal`
- 比较稳定结果，避免依赖不确定输出（如 map 遍历顺序）

### 测试组织
- 子测试名称应简短描述性，不含输入数据的完整表示
- 测试用例必须有唯一标识符（`name` 字段）
- 同包测试（`_test.go`）vs 不同包测试（`_test` 包）按需选择
- 用反引号定义多行字符串常量

### 测试失败信息
- 包含：输入、实际输出、期望输出
- 格式：`got X, want Y` 或 `Function(x) = z, want y`

### 其他
- 用 `TestMain` 做全局 setup/teardown
- 用 `sync.Once` 做昂贵资源的延迟初始化
- 测试中也可以用真实传输（real transport）而非 mock

## 6. 注释与文档

### Doc Comments
- 每个导出的标识符都要有 doc comment
- 以标识符名称开头：`// Open opens the named file...`
- 完整句子，首字母大写（除非以标识符开头）

### 包注释
- 在 `package` 声明前写包级文档
- 说明包的用途和整体功能

### 注释内容
- 解释 **Why** 而非 What
- 删掉无用的注释（代码本身应该说明 What）
- `TODO` 格式：`// TODO(username): description`

### 函数文档
- 描述参数和返回值的语义（而不是类型，类型在签名中已有）
- 注明是否持有锁、是否阻塞、是否安全并发调用
- 说明 context 取消时的行为
- 说明资源清理行为（如 `Close()` 的效果）
- 可运行示例（`Example*` 测试函数）作为包文档的一部分

### godoc 格式
- 段落用空行分隔
- 代码块用缩进 2 空格
- 标题用 `##` 前缀
- 用 `godoc` 工具预览文档效果

## 7. 代码格式

### 基本规则
- 始终用 `gofmt` / `goimports` 格式化
- 行长度没有硬限制，但建议 80-120 字符
- 缩进用 Tab

### Import 分组
```go
import (
    // 标准库
    "fmt"
    "os"

    // 第三方库
    "go.uber.org/zap"

    // 项目内部
    "myproject/pkg/user"
)
```

- 不要用 dot import（测试文件中的 testify 除外）
- 避免别名，除非解决命名冲突
- protobuf 代码用 `pb` 别名，gRPC 代码用 `grpc` 别名
- 空导入（blank import）只在驱动注册等标准模式中使用

### 文件内声明顺序

`.go` 文件从上到下按以下顺序声明（对齐 golangci-lint `decorder` 默认 + Uber Function Grouping）：

1. `package` + 包/文件注释
2. `import`（见上一节分组规则）
3. **类型声明**（`type` / `struct` / `interface`）
4. **构造函数** `NewXYZ()` — 紧贴类型
5. **常量** `const`（错误哨兵、枚举、默认值）
6. **包级变量** `var` — 越少越好（参见 [Practical Go — 避免 package-level state](https://dave.cheney.net/practical-go/presentations/qcon-china.html)）
7. **导出方法**（按 receiver 分组）
8. **导出函数**（非 method）
9. **非导出方法**（按 receiver 分组）
10. **非导出 / 工具函数**（文件底部）

**类型在前，不是常量在前**：包级 `var` / `const` 经常引用类型（`var Default = Config{...}`），先看到数据形状再读值，符合自上而下阅读。这是 Go 业界主流（decorder 默认、Uber style、标准库样本都这么做），不是 Java/C 的"常量在顶"。

**方法按 receiver 分组**：同一 receiver 的方法集中放，不要散到文件多处。读完 `*Server` 的所有方法再看 `*Listener`。

**函数按调用顺序排，不按字母序**：入口/构造在前，被调用的工具函数在后（Uber: "rough call order"）。读者从上往下扫，先看到重要的。

```go
// Good
package server

import (...)

type Server struct { ... }                      // 3. 类型

func NewServer() *Server { ... }                // 4. 构造，紧跟类型

const DefaultAddr = ":8080"                     // 5. 常量

var defaultLogger = newLogger()                 // 6. 包级 var

func (s *Server) Start() error { ... }          // 7. 导出方法
func (s *Server) Stop()  error { ... }

func (s *Server) listen() { ... }               // 9. 非导出方法

func parseAddr(addr string) (string, error) { ... }  // 10. 工具函数，文件底部
```

**特殊规则**：

- `init()` 必须在所有其它函数之前（`decorder` 默认 `disable-init-func-first-check: false` 强制）
- 同类常量合并成一个 `const` 块，不要散开多条 `const`：

```go
// Good
const (
    DefaultAddr    = ":8080"
    DefaultTimeout = 30 * time.Second
)

// Bad
const DefaultAddr = ":8080"
const DefaultTimeout = 30 * time.Second
```

- `Test*` 函数（`_test.go`）顺序不强制，但表驱动测试的 `tests` slice 应紧跟测试函数声明

**强制方式**：在 `.golangci.yml` 启用 `decorder`：

```yaml
linters:
  enable:
    - decorder
linters-settings:
  decorder:
    dec-order:
      - type
      - const
      - var
      - func
    disable-dec-order-check: false      # 强制声明顺序
    disable-init-func-first-check: false # init() 必须在最前
    disable-dec-num-check: false         # 每类声明只允许一个块（强制 const(...) 合并）
```

## 8. 语言特性

### 字面量格式化
- 结构体必须写字段名：`T{Name: "foo"}` 而非 `T{"foo"}`
- 省略零值字段：`T{Name: "foo"}` 而非 `T{Name: "foo", Count: 0}`
- 短结构体可写一行：`Point{x: 1, y: 2}`
- 省略映射/切片中的重复类型名
- `}` 与开头的 `T{` 对齐

### 变量声明
- 用 `:=` 声明并初始化
- 明确零值时用 `var`：`var buf bytes.Buffer`
- 指针用 `&T{}` 而非 `new(T)`
- 复合字面量优于 `new(T)`：`&User{Name: "foo"}` 比 `new(User)` 清晰

### Slice
- nil slice 是合法的，不需要 `make([]T, 0)`
- 预知大小时用 `make([]T, 0, cap)` 减少 GC 压力

### Map
- 必须用 `make` 初始化：`m := make(map[string]int)`
- 检查 key 存在：`v, ok := m[key]`

### 字符串拼接
- 少量拼接：`+` 操作符
- 格式化：`fmt.Sprintf`
- 循环中大量拼接：`strings.Builder`
- 复杂模板：`text/template`

### Switch
- 不需要 `break`（Go 自动 break）
- 用 `default` 处理未匹配情况

### 泛型
- 只在真正需要类型参数时使用
- 如果接口能满足，优先用接口
- 类型约束用 `interface{}` 组合或 constraint 包

### 类型别名
- 只为兼容性使用 `type X = Y`
- 不要为了方便而滥用别名

### 其他格式偏好
- 用 `%q` 而非 `"%s"` 格式化字符串
- 用 `any` 而非 `interface{}`
- 同步函数优先：当同步 API 已满足需求时，不提供异步版本
- 函数参数中声明 channel 方向：`chan<- int`（只写）、`<-chan int`（只读）

### 复制
- 值接收者方法会隐式复制：大结构体用指针接收者
-嵌入值类型 = 复制语义，嵌入指针 = 共享语义

## 9. Receiver 类型选择

| 场景 | 用值接收者 | 用指针接收者 |
|------|----------|------------|
| 小结构体，无修改 | ✅ | |
| 需要修改接收者 | | ✅ |
| 大结构体（> 几十个字节） | | ✅ |
| 包含 sync 类型 | | ✅ |
| 需要一致性（某些方法需指针） | | ✅ |

**规则：如果任何一个方法需要指针接收者，所有方法都用指针接收者。**

## 10. 通用库

### 日志
- 使用标准库 `log/slog`（Go 1.21+），不引入第三方日志库
- 使用结构化字段：`slog.Info("msg", "key", value)` 或 `slog.String("key", value)`
- 初始化时设置 handler 和级别：
  ```go
  logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
  slog.SetDefault(logger)
  ```
- 避免 `log.Fatal` 和 `log.Exit`（应返回错误让调用方决定）
- 库代码不应直接打日志，应返回错误
- HTTP 中间件用 `slog-http` 等适配包自动记录请求信息

### 命令行
- CLI flag 用 `snake_case`：`--max-retries`
- 库代码不应使用 `flag` 包

### 随机数
- 优先用 `crypto/rand` 而非 `math/rand`

## 11. Context

- 第一个参数传 `ctx context.Context`
- 不要把 context 放在 struct 里
- 不要传 nil context，用 `context.TODO()` 代替
- 用 context 传递请求级数据，不传业务参数
- 不要创建自定义 context 类型，只用 `context.Context`

## 12. 常见反模式

### 避免
- ❌ 在 init() 中做复杂逻辑或网络请求
- ❌ 裸 return（无命名返回值时）
- ❌ 过度使用 interface（接口不是越少越好，也不是越多越好）
- ❌ 忽略 error（不用 `_ = mayFail()`）
- ❌ 用 panic 做流程控制
- ❌ 导出未文档化的标识符
- ❌ `defer` 在循环中使用（会累积到函数返回才执行）
- ❌ 用 `map` 做有序集合
- ❌ 把 error string 做字符串比较判断（用 `errors.Is`）
- ❌ 逐字段比较结构体（用 `cmp.Diff` 做完整比较）
- ❌ 在 goroutine 中调用 `t.Fatal`
- ❌ `interface{}` 代替 `any`（Go 1.18+）
- ❌ `math/rand` 代替 `crypto/rand`（需要安全性时）
- ❌ 创建自定义 context 类型

### 推荐
- ✅ 构造函数返回 error 而非 panic
- ✅ 用 `defer` 做资源清理
- ✅ 小接口、组合优于继承
- ✅ 明确的包边界和职责划分
- ✅ 包级函数 vs 方法：需要状态用方法，纯逻辑用函数

## 13. 代码质量验证

**写完或修改 Go 代码后，必须依次执行以下验证步骤。**

### 第一步：格式化

```bash
# 格式化当前修改的文件
gofmt -w <file.go>
goimports -w <file.go>
```

- `gofmt`：统一缩进、空格、换行等格式
- `goimports`：在 gofmt 基础上自动管理 import（增删、分组排序）
- 两者都带 `-w` 直接修改文件

### 第二步：Lint 检查

**需要 golangci-lint v2（`version: "2"` 配置格式）。** 安装方式：

```bash
# Homebrew (macOS)
brew install golangci-lint

# go install (跨平台)
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
```

```bash
# 对修改的文件或包运行 lint
golangci-lint run ./...

# 只检查特定文件
golangci-lint run <file.go>

# 只运行特定 linter
golangci-lint run --enable-only=staticcheck,errcheck,govet ./...
```

常用 linter 说明：

| Linter | 检查内容 |
|--------|---------|
| `staticcheck` | 未使用的变量/函数、冗余代码、常见 bug（v2 已合并 gosimple/unused/typecheck） |
| `govet` | 可疑代码结构（lock copy、wrong printf args） |
| `errcheck` | 未检查的错误返回值 |
| `revive` | 风格问题、命名规范、注释缺失 |
| `gocritic` | 性能与风格建议 |

### 第三步：修复与迭代

1. 如果 lint 报错，**先修复所有问题再继续**
2. 修复后重新跑 `gofmt -w` + `goimports -w`（修复过程可能改变格式）
3. 重新跑 `golangci-lint` 确认通过
4. 循环直到全部通过

### 项目级配置

如果项目根目录有 `.golangci.yml`，遵循项目配置（linter 选择、排除规则等）。不要在已有配置的项目中覆盖项目级 lint 规则。

如果没有配置文件，读取本目录下的 [.golangci.yml](.golangci.yml) 模板，将 `MODULE_NAME` 替换为项目的 `go.mod` 中的 module 名后复制到项目根目录。

> **注意**：模板使用 golangci-lint v2 配置格式（`version: "2"`），需要 golangci-lint v2.x。v1 版本报错：`you are using a configuration file for golangci-lint v2 with golangci-lint v1`。

### 何时跳过

- 只修改注释或文档时，可跳过 lint（但格式化仍建议执行）
- `goimports` 不可用时，退回 `gofmt`
- `golangci-lint` 未安装时，至少执行 `go vet ./...` 作为最小检查

## 14. 配置 struct 设计

顶层 `Config` struct 里的**子配置 struct 字段用指针**，不要直接嵌套值类型：

```go
// Good
type Config struct {
    Server     *ServerConfig
    Database   *dbx.Config
    Redis      *redisx.Config       // 可选依赖
    Storage    *StorageConfig       // 可选依赖
    ThirdParty *ThirdPartyConfig
    Log        *logging.Config
}

// Bad — AI 默认产出风格，全部值类型嵌套
type Config struct {
    Server     ServerConfig
    Database   dbx.Config
    Redis      redisx.Config        // 无法表达"未配置 Redis"
    Storage    StorageConfig
    ThirdParty ThirdPartyConfig
    Log        logging.Config
}
```

**为什么**：

1. **跟整体风格一致** — `Load()` 返回的就是 `*Config`（指针）；调用方传 cfg 也用 `*Config`。子配置再用值类型会让 struct 内部跟外部传递风格不一致。
2. **跟 functional options 模式一致** — `option.Options` 里 `DB *gorm.DB` / `GIDService thirdcall.GIDService` 都是 nil 表达"未注入"。Config 子配置走指针跟 option 风格统一。
3. **修改子配置类型不 breaking** — 子配置从单值变 struct（如 `Redis string` → `Redis *RedisConfig`），父字段已经是指针时只是字段类型变；如果是值，所有传 `cfg.Redis` 的调用方都要改。
4. **避免大 struct 拷贝** — 配置传值（罕见但会发生，如复制 cfg 后修改），指针只拷贝 8 字节；值类型拷贝整个子树。

**判据**：

| 字段类型 | 用指针？ | 例 |
|---------|---------|-----|
| 子配置 struct | ✅ 必须 | `Server *ServerConfig`、`Database *dbx.Config`、`Redis *redisx.Config` |
| 标量字段 | ❌ 直接值类型 | `GRPCAddr string`、`Timeout time.Duration`、`MachineID int64` |

库提供的配置类型（`dbx.Config`、`logging.Config`、`redisx.Config`）由库定义无法改字段类型，但在 `Config` 里**仍然用指针包装**（`*dbx.Config`）。

**业务侧使用** — Go 自动解引用单层指针，字段访问语法不变：

```go
// 改造前后业务侧代码完全一致
addr := cfg.Server.GRPCAddr  // cfg.Server *ServerConfig，自动解引用
db := dbx.New(cfg.Database)  // cfg.Database *dbx.Config，直接传指针
```

**⚠️ 不要用 `cfg.X == nil` 判断"未配置"** — configx（viper）解析时**总是给指针字段分配**：即使配置文件里没出现该 key，也会得到 `&T{}`（零值指针，非 nil）。所以 `if cfg.Redis != nil` 永远为 true，无法区分"未配置"和"零值配置"。

表达"可选依赖"应该用子配置里的 `Enabled bool` 字段：

```go
type RedisConfig struct {
    Enabled bool  // 业务侧用 if cfg.Redis.Enabled 判断是否启用
    Addr    string
    // ...
}
```

而不是依赖指针 nil 判断。

### New 函数签名也用指针

构造函数接收配置参数时**也用指针**，不要把整个 Config 值拷贝过去：

```go
// Good
func New(cfg *Config) *Service
func NewServer(cfg *ServerConfig) *Server
func NewRedisClient(cfg *redisx.Config) *redis.Client

// Bad — AI 默认产出风格，把 Config 当值拷
func New(cfg Config) *Service
func NewServer(cfg ServerConfig) *Server
```

**为什么**：

1. **跟 §14 主规则一致** — Config 字段已经是指针（`*ServerConfig`），New 也用指针是一致性延伸。否则规则只在 struct 定义层成立，函数签名层又退回值类型，自相矛盾。
2. **避免拷贝** — Config struct 通常带 nested 子树，值传递拷贝整棵树；指针拷贝 8 字节。
3. **修改 cfg 不影响调用方** — 如果 New 内部需要修改 cfg（如填默认值），指针让修改对调用方可见；值类型只能改到本地副本。

**适用范围**：所有 `New*` / `New*Client` / `New*Server` 构造函数，无论 cfg 是顶层 `*Config` 还是子配置 `*ServerConfig` / `*GIDConfig`。子配置内部的标量字段（`Addr string`、`Timeout time.Duration`）作为普通参数传递时仍然用值类型 — 这条只约束 struct 类型的配置参数。

