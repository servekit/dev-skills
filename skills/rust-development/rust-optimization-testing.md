---
name: rust-optimization-testing
description: Rust performance optimization guide and testing best practices — covers profiling tools, benchmark methodology, optimization techniques, compilation tuning, and testing patterns. Use when optimizing Rust code performance or setting up test infrastructure.
---

# Rust 性能优化与测试指南

基于 [Rust 编码规范](https://rust-coding-guidelines.github.io/rust-coding-guidelines-zh/) 附录提炼。

---

## 优化总则

### 原则一：不要过早优化
- "过早优化是万恶之源" — Donald Knuth
- 先完成再完美，优化代码可读性是持续要做的
- Rust 显式语义要求命名体现类型语义

### 原则二：不要过度优化
- 性能够用就好
- 案例：第一版 Rust 代码已比 Python 快 20 倍，第二版优化收益极小
- 过度优化浪费时间

### 原则三：性能、安全、编译速度和编译大小需要权衡
- 使用 Unsafe 提升性能可能牺牲安全性
- 优化可能导致编译变慢或二进制膨胀

---

## 性能测试准备

### 基准测试（Benchmark）
- **工具**：`cargo bench` + `criterion` crate
- `criterion` 支持 Stable Rust（Rust 自带基准测试仅限 Nightly）
- 配置示例：
  ```toml
  [dev-dependencies]
  criterion = { version = "0.3.5", features = ["async_tokio", "html_reports"] }

  [[bench]]
  name = "find"
  harness = false
  ```
- `criterion` 生成 HTML 报告，可视化性能变化和回归
- 每次运行自动与上次比较，检测性能回归

### 压力/负载测试
- 基准测试是开发期间的预判，发布后需要实际负载测试
- **goose**（Rust 实现）：分布式负载测试工具，每核产生流量至少是 Locust 的 11 倍
- 其他工具：`wrk`、`locust`

### 高性能系统标准
- 性能 = 产出 / 资源消耗
- 产出 = 事务次数（QPS）+ 吞吐数据量
- 消耗 = CPU 时间片、磁盘/网络 I/O
- 高性能设计标准：
  1. 最大化利用资源
  2. 使用流水线技术减少任务总耗时

### 常见瓶颈类型
1. **CPU**：占用过高 → 减少计算；负载过高 → 检查线程数和切换频率
2. **I/O**：
   - IOPS 达上限 → 减少读写次数，提高 cache 命中率
   - 带宽达上限 → 紧凑数据格式，减少读写放大
   - 并发达上限 → 使用异步 I/O
   - 锁、计时器、分页/交换阻塞

---

## 性能剖析工具

### On-CPU 分析

#### Perf + 火焰图
- **perf**：Linux CPU 性能采样工具
- **flamegraph** crate：与 cargo 集成的火焰图生成器
- 使用步骤：
  ```bash
  # Cargo.toml 中 [profile.release] debug = true
  cargo build --release
  perf record -g target/release/my-app
  perf report
  # 或直接生成火焰图
  cargo flamegraph --bin my-app -o flamegraph.svg
  ```
- 火焰图阅读：底部开始向上，宽度 = 时间占比，宽矩形 = 性能瓶颈

#### 检查内存泄漏和不必要分配
- **Valgrind**：检查内存泄漏和不必要的堆分配
- **Rust nightly**：`RUSTFLAGS=-Zprint-type-sizes cargo build --release` 查看数据结构大小
- 异步程序占用过多栈空间 → 考虑平衡同步/异步代码组合

#### 其他工具
- **Hotspot** / **Firefox Profiler**：查看 perf 数据
- **Cachegrind** / **Callgrind**：指令数和缓存模拟
- **DHAT**：找到大量分配的代码位置
- **heaptrack**：堆分析
- **counts**：即席剖析（eprintln + 频率分析）
- **Coz**：因果分析，衡量优化潜力
- **VTune**（Intel）：高级 CPU 性能剖析

### Off-CPU 分析
- **bcc** 工具包中的 `offcputime-bpfcc`
- 原理：记录进程离开 CPU 到下次调度的时间差
- 需要开启 `RUSTFLAGS="-C force-frame-pointers=yes"` 以便栈展开
- 生成 Off-CPU 火焰图：
  ```bash
  ./target/debug/mytest & sudo offcputime-bpfcc -p $(pgrep -nx mytest) 5 > out.stacks
  ./flamegraph.pl --color=io --title="Off-CPU Time Flame Graph" < out.stacks > out.svg
  ```

---

## 日常优化技巧

### 1. 优化占运行时间最长的函数
- 只调用一次的函数（如读配置文件）不需要优化
- 优先优化被频繁调用的函数

### 2. 优先改进算法
- 性能不佳往往是算法问题而非实现问题
- 减少 `collect` 次数（每次至少遍历一次集合）
- 警惕标准库和第三方库方法内部的隐藏循环

### 3. 理解数据结构的内存布局
- 栈 vs 堆：`String`/`Vec`/`HashMap`/`Box` 分配在堆上
- 栈数据移动是按位复制，考虑 Copy 成本
- 堆数据避免深拷贝（显式 Clone）
- 缓存数据避免频繁分配（如 `slab` crate）

### 4. 避免 Box 动态分发
- 大多数代码可用 `&mut Trait` 代替 `Box<Trait>`
- 用 Enum 代替 trait 对象（`enum_dispatch` crate）

### 5. 使用基于栈的可变长度类型
- `smallvec`、`smallstring`、`tendril`：少量元素存储在栈上
- 提升缓存局部性，减少堆分配

### 6. 合理使用断言消除数组越界检查
- 编译器自动为每个数组索引插入边界检查
- 手工插入一次 `assert!` 消除多次自动检查：
  ```rust
  fn process(array: &[u8]) -> u8 {
      assert!(array.len() >= 6); // 一次检查
      array[0] + array[1] + array[2] + array[3] + array[4] + array[5] // 不再重复检查
  }
  ```

### 7. 使用 LTO（链接时优化）
- 允许跨 crate 内联
- 代价：编译时间变慢，但值得用编译时间换性能

### 8. 不要使用 #[inline(always)]
- 让编译器自行决定内联时机
- 除非函数调用极其频繁
- 显式指定会导致二进制膨胀

### 9. 避免显式 Clone
- 尽可能使用引用
- Clone 可能伴随内存分配

### 10. 使用 Unsafe 方法消除不必要的安全检查
- 标准库中 `_unchecked` 后缀的方法跳过安全检查
- `String::from_utf8` vs `String::from_utf8_unchecked`
- 确保调用环境安全时才使用 unchecked 版本

### 11. 并发/并行化
- **rayon**：并行迭代器
- **crossbeam/flume**：多线程 channel
- **tokio**：异步运行时
- **loom**：并发测试
- **console**：异步诊断

### 12. 合理使用锁或无锁数据结构
- 读多写少 → 读写锁代替互斥锁
- `parking_lot` 代替标准库锁
- 高质量无锁实现可能优于锁同步

### 13. 使用 Clippy
- Clippy 包含性能改进 lint
- 遵循 Rust 编码规范中的 Clippy 建议

---

## 编译大小和编译时间优化

### 编译大小优化
- `codegen-units = 1`：减少分割单元，利于内联优化（但可能增大二进制）
- `panic = "abort"`：缩减二进制大小
- 优化等级 `opt-level = "z"`：最小二进制体积
- 评估泛型和宏的使用，精简不必要的

### 编译时间优化
- 使用 `cargo check` 代替 `cargo build`
- 使用最新 Rust 工具链
- 使用 Rust Analyzer 而非 RLS
- 删除未使用的依赖，替换依赖过多的库
- 使用 workspace 拆分 crate，并行编译
- 测试拆分为独立文件，集成测试合并到同一文件
- 使用 `sccache` 缓存依赖
- 切换更快链接器：`mold`（Linux）/ `zld`（macOS）
- macOS 增量编译：`split-debuginfo = "unpacked"`
- 避免过程宏 crates（特别是含 `syn` 的）
- 避免过多泛型单态化
- 使用 `cargo -Z timings` 分析编译步骤

---

## 测试指南

### 单元测试
- 内部函数：测试代码放在同模块下
- 外部接口：测试代码放在独立的 `tests/` 目录
- 文档测试：对所有对外接口进行文档测试（`cargo test --doc`）
- 编译测试：通过 `compiletest` 测试某些代码无法编译
- 随机测试：使用 `proptest` crate 进行属性测试
- 覆盖率：使用 `tarpaulin` 检测代码覆盖率（仅 Linux x86_64）

### 基准测试工作流
1. 使用 `criterion` 建立基线
2. 通过 `cargo flamegraph` 识别瓶颈
3. 尝试解决瓶颈
4. 重新运行基准测试验证改进
5. 重复以上步骤

### 压力测试
- 基准测试后用真实负载验证
- `wrk`：简单 HTTP 压测
- `goose`：Rust 实现的分布式负载测试

### 模糊测试
- 使用 `cargo-fuzz` 进行模糊测试
- 发现安全性和稳定性问题
- 对处理外部输入的代码特别重要

---

## 编程技巧（Best Practices）

### 构建者模式（Builder Pattern）
- 需要多个构造函数或很多可选配置时使用
- Rust 没有默认构造函数，构建者模式提供灵活的对象构造
- 使用 `Default` derive 简化构建者实现

### 善用迭代器适配器
- `enumerate()` 代替手动计数器循环
- `flatten()` 代替 `filter_map(|x| x)` 和 `flat_map(|x| x)`
- `find()` 代替 `filter().next()`
- `filter_map()` 代替 `flat_map(|x| x.parse().ok())`

### 使用 Cow 减少拷贝
- `Cow<'a, str>` 借用或拥有的灵活选择
- 读多写少场景特别适合
- 未修改时使用 `&str`，修改时才转为 `String`

### 错误处理策略
- **应用代码**：使用 `Box<dyn Error>` 或 `anyhow`
- **库代码**：返回自定义错误类型，方便下游处理
- 使用 `thiserror` 简化错误定义

### 嵌入式（no_std）共享库
- 将公用的类型、函数、宏集中到自定义 `baremetal-std`
- 积累 no-std 下常用的公共库
