---
name: rust-concurrency-async-rules
description: Detailed concurrency and async rules from Rust Coding Guidelines — covers locks, lock-free programming, and async patterns. Use when writing or reviewing multi-threaded or async Rust code.
---

# 并发与异步详细规则

基于 [Rust 编码规范](https://rust-coding-guidelines.github.io/rust-coding-guidelines-zh/) 提炼。

---

## 多线程 — 锁（Locks）

### 简单类型优先使用原子操作而非 Mutex
- **级别**：建议
- `bool`、`&T` 等简单类型用 `AtomicBool`、`AtomicPtr` 更高效
- `Mutex` 有锁开销（系统调用、线程阻塞），原子操作通常更快
- 适用场景：简单状态标记、引用计数

### 使用 Arc<str>/Arc<[T]> 而非 Arc<String>/Arc<Vec<T>>
- **级别**：建议
- `Arc<String>` 会导致双重分配：`Arc` 分配 + `String` 内部堆分配
- `Arc<str>` 只需一次分配，`Arc` 直接包裹数据
- 同理：`Arc<[T]>` 优于 `Arc<Vec<T>>`

### 优先使用 parking_lot 而非 std::sync
- **级别**：建议
- `parking_lot` 优势：
  - 更小的内存占用
  - 无 poisoning（锁持有者 panic 不再阻止后续获取）
  - 更好的性能（基于 parking 许可证机制）
  - 支持 `RawMutex` 等底层原语

### 优先使用 crossbeam channel 而非 std::sync::mpsc
- **级别**：建议
- `crossbeam` channel 优势：
  - 支持 multi-producer multi-consumer（MPMC）
  - 更好的性能
  - `select!` 宏支持多通道选择
  - 更丰富的 API（bounded/unbounded/after/tick）

### 注意锁竞争和死锁
- **级别**：要求
- **锁排序**：多个锁必须按固定顺序获取，防止死锁
- **try_lock 策略**：使用 `try_lock` 避免无限等待
- **锁粒度**：持锁时间尽量短，不要在持锁期间做 I/O 或耗时操作
- **避免嵌套锁**：如必须嵌套，确保全局一致的获取顺序

---

## 多线程 — 无锁（Lock-free）

### 除非必要，不要使用无锁编程
- **级别**：建议
- 原子操作比 `Mutex` 快约 4 倍，但推理难度更大
- 大多数场景下 `Mutex` 已经足够高效
- 只在极端性能要求或无法使用锁的场景（如信号处理器）中使用无锁

### 使用正确的内存排序
- **级别**：要求
- 内存排序从弱到强：
  1. **Relaxed**：最佳性能，无同步保证，仅保证原子性
  2. **Release**：写操作用，确保之前的写操作对其他线程可见
  3. **Acquire**：读操作用，确保之后的读操作看到最新数据
  4. **AcqRel**：Acquire + Release 的组合
  5. **SeqCst**：最安全但性能最差，全局顺序一致
- Acquire/Release 成对使用建立因果关系
- 不确定时用 `SeqCst`，确定后降级

---

## 异步编程（Async）

### 不要忘记 .await
- **级别**：要求
- 忘记 `.await` 意味着 Future 永远不会执行
- 编译器会发出 "unused must_use" 警告
- 异步函数调用返回 `Future`，必须 `.await` 才会执行

### 跨 await 点处理 RefCell 引用
- **级别**：要求
- 在 `.await` 之前 drop `RefCell`/`MutexGuard`
- 跨 `.await` 持有 `Ref`/`RefMut` 会导致 panic
- 编译器会检测到 `&mut RefCell<T>` 跨 `.await` 并报错

### 没有异步代码的函数不要标记为 async
- **级别**：建议
- `async fn` 有运行时开销（状态机生成、Future 分配）
- 纯同步计算不需要 async
- 如果函数内部没有任何 `.await`，就不应该是 `async fn`

### 异步上下文中使用运行时的等价 API
- **级别**：要求
- 不要在 async 代码中使用阻塞 I/O
- 使用 `tokio::fs` 代替 `std::fs`
- 使用 `tokio::net` 代替 `std::net`
- 使用 `tokio::time` 代替 `std::thread::sleep`
- CPU 密集任务用 `spawn_blocking` 卸载到线程池

---

## 并发工具推荐

### 线程间通信
- **crossbeam**：多线程 channel、无锁并发结构
- **flume**：高性能多生产者多消费者 channel

### 并行计算
- **rayon**：并行迭代器，轻松并行化数据处理

### 异步运行时
- **tokio**：功能全面，生态丰富（推荐用于服务端）
- **async-std**：与标准库 API 一致

### 测试与调试
- **loom**：Tokio 提供的并发代码测试工具，支持 C11 内存模型
- **console**：Tokio 提供的异步诊断工具，检测性能问题和错误模式
