---
name: rust-unsafe-rules
description: Detailed Unsafe Rust rules from Rust Coding Guidelines — covers FFI, memory, raw pointers, unions, safe abstractions, and IO safety. Use as reference when writing or reviewing unsafe Rust code.
---

# Unsafe Rust 详细规则

基于 [Rust 编码规范](https://rust-coding-guidelines.github.io/rust-coding-guidelines-zh/) 提炼，涵盖 unsafe Rust 的所有子领域详细规则。

## 通用规则

### 不要滥用 unsafe
- **级别**：要求
- 不要使用 unsafe 来绕过编译器的安全检查，只在绝对必要时使用
- unsafe 应该用于：FFI、性能关键路径（且有充分理由）、与底层硬件交互

### 精确限定 unsafe 块的作用域
- **级别**：要求
- unsafe 块应尽可能小，只包裹真正需要的操作
- 大 unsafe 块可能意外包含不需要 unsafe 的代码，增加出错概率

### 不要为含 "unsafe" 的类型创建别名
- **级别**：建议
- 不要为 `UnsafeCell` 等类型创建别名，直接使用原名以保持代码清晰度

---

## FFI（Foreign Function Interface）

### 避免 FFI 中的格式字符串漏洞
- **级别**：要求
- 不要将公共 Rust API 的字符串直接传递给 C 的格式化函数
- 必须在 FFI 边界做参数检查，防止格式字符串攻击（format string vulnerability）
- 示例：C 的 `printf` 系列函数不应直接接收用户输入的格式字符串

### 为包装 C 指针的 Rust 类型实现 Drop
- **级别**：要求
- 当使用 C 指针管理内存时，为包装的 Rust 类型实现 `Drop` trait
- 通过 C-ABI 回调函数安全释放资源
- 确保即使发生 panic 也能正确释放 C 资源

### 跨 FFI 边界时处理 Panic
- **级别**：要求
- 使用 `catch_unwind` 捕获 panic（仅对 `UnwindSafe` 类型有效）
- 或返回错误码代替 panic
- **跨 FFI 边界 panic 导致未定义行为（UB）**
- `catch_unwind` 不能捕获所有 panic（如 `abort`）

### FFI 中使用可移植类型别名
- **级别**：要求
- 使用 `std::os::raw` 或 `libc` crate 中的类型别名
- 不要使用平台特定的类型（如 `c_long` 在不同平台大小不同）
- 示例：`c_int`, `c_long`, `c_char`, `c_void`

### FFI 中正确处理 C 字符串
- **级别**：要求
- 使用 `c_char` 表示 C 字符
- FFI 字符串必须以 `\0` 结尾
- 注意 UTF-8 与 C 编码的差异
- 示例：`"/proc/uptime\0".as_ptr().cast()` — 字面量必须包含 null 终止符
- 使用 `CString`/`CStr` 进行安全的字符串转换

### FFI 中的错误处理策略
- **级别**：要求
- 三种策略：
  1. **无字段枚举** → 数值返回码（如 `0 = Success, 1 = Error`）
  2. **携带数据的枚举** → 整数码 + 错误描述函数
  3. **自定义错误类型** → `#[repr(C)]` C 兼容布局

### 导出函数必须线程安全
- **级别**：要求
- 导出的 Rust 函数默认必须可跨线程安全调用
- 除非能保证在单线程环境中使用，否则必须确保 `Send`/`Sync`

### 避免 repr(packed) 的未对齐引用
- **级别**：要求
- 引用 `#[repr(packed)]` 结构体字段可能导致未对齐访问（UB）
- 使用 `ptr::read_unaligned`/`ptr::write_unaligned` 或将字段拷贝到局部变量（需要 `Copy`）
- **Lint**: `unaligned_references`

### 为 C 传入参数编写不变量文档
- **级别**：要求
- 两种场景：
  1. 参数保证有效 → 在类型不变量文档中说明 + 只提供 unsafe 构造函数
  2. 参数不确定 → 在 safe 构造函数中验证（如 null 检查）
- doc comment 中说明参数的安全约束

### FFI 边界确保一致的数据布局
- **级别**：要求
- 使用 `#[repr(C)]` 确保 FFI 边界数据布局一致
- **不适合 FFI 的类型**：无 repr 的自定义类型、DST、胖指针、`str`、元组、闭包
- 这些类型没有稳定的内存布局

### FFI 类型必须有稳定布局
- **级别**：要求
- FFI 类型必须使用 `#[repr(C)]` 或 `#[repr(transparent)]`
- 不要在 FFI 中使用 ZST（零大小类型）
- 不透明类型用 `libc::c_void` 或 `{ _unused: [u8; 0] }`

### 在 FFI 边界验证非健壮类型
- **级别**：要求
- 需要验证的外部值类型：`bool`、引用、函数指针、枚举、浮点数、包含这些的复合类型
- C 端可以传入任何比特模式，必须验证

### 将 Rust 闭包传递给 C
- **级别**：建议
- 分离数据（`Box::into_raw`）和代码（`extern "C" fn`）
- 确保生命周期有效、`UnwindSafe`、`Send`
- 使用回调模式：C 端调用 extern "C" 函数，该函数解引用 Box 并调用闭包

### 使用专用不透明类型而非 c_void 指针
- **级别**：建议
- 使用 `#[repr(C)] struct Foo { _private: [u8; 0] }` 作为不透明类型
- 比 `*const c_void` 更类型安全
- 编译器可以区分不同不透明类型的指针

### 避免向 C 传递 trait 对象
- **级别**：建议
- trait 对象没有稳定的 ABI（胖指针布局可能变化）
- 替代方案：使用带指针的枚举，或 `thin_trait_object` 模式

---

## 内存操作（Memory）

### 使用 MaybeUninit 处理未初始化内存
- **级别**：要求
- 使用 `MaybeUninit<T>` 处理未初始化内存
- **绝不**使用 `mem::zeroed()` 获取引用（引用不能为 0）
- **绝不**使用 `mem::uninitialized()` 获取 `bool`（可能不是 `true`/`false`）

### 使用 #[repr] 控制数据布局
- **级别**：建议
- 选择合适的 `#[repr]` 属性控制结构体/元组/枚举的内存布局
- `#[repr(C)]`：C 兼容布局
- `#[repr(transparent)]`：单字段类型透明包装
- `#[repr(u8)]`/`#[repr(i32)]` 等：枚举判别式大小

### 不要修改其他进程/共享库的内存
- **级别**：要求
- 修改其他进程或动态库的内存会导致 `SIGSEGV`
- 示例：`sqlite3_libversion()` 返回指向 `.so` 静态字符串的指针，不应修改

### 不要让 String/Vec Drop 释放其他进程的内存
- **级别**：要求
- `String`/`Vec` 的 `Drop` 会释放当前进程的内存
- 如果数据来自其他进程/动态库，不能让 Rust 的 `Drop` 去释放
- 使用 `mem::forget` 或 `ManuallyDrop`，或先将数据拷贝到当前进程内存

### 使用可重入版本的 C API
- **级别**：要求
- 非可重入函数会写入库中的静态变量，导致线程安全问题
- 使用带 `_r` 后缀的版本：`ctime_r`, `gmtime_r`, `localtime_r`, `gethostbyname_r`
- 可重入函数使用调用者提供的缓冲区，不依赖静态状态

### 使用第三方库处理位域
- **级别**：建议
- Rust 没有内建位域（bit field）支持
- 推荐库（均支持 no-std）：
  - **bitvec**：功能最丰富，支持任意位级操作
  - **bitflags**：便捷宏语法，适合标志位集合
  - **modular-bitfield**：完全 safe Rust，编译时检查

---

## 裸指针（Raw Pointer）

### 不要解引用错误对齐的指针
- **级别**：建议
- 将 `*const u8` 转换为 `*const u16` 并解引用会导致 UB
- 必须确保指针类型对齐：`u16` 需要 2 字节对齐，`u32` 需要 4 字节对齐
- **Lint**: `cast_ptr_alignment`（Clippy, style, warn）

### 不要手动将 const 指针转为 mut 指针
- **级别**：要求
- `*const _ as *mut _` 导致 UB — Rust 的引用规则不允许通过共享引用修改数据
- 使用 `UnsafeCell<T>` 代替
- **Lint**: `cast_ref_to_mut`（Clippy, correctness, deny）
- 例外：当风险已被理解并用 `#[allow(clippy::cast_ref_to_mut)]` 文档说明时

### 使用 pointer::cast 而非 as 进行指针转换
- **级别**：要求
- `pointer::cast` 更安全 — 不会意外改变可变性或将指针转为其他类型
- `as` 可能做意外的类型转换
- **Lint**: `ptr_as_ptr`（Clippy, correctness, deny）

### 使用 NonNull 替代 *mut T
- **级别**：建议
- `NonNull<T>` 优势：
  1. 非空保证 + 自动 null 检查
  2. 协变性（covariance），用于安全抽象时不需要额外的 `PhantomData`
- 适用于构建安全抽象时的内部指针

### 使用 PhantomData 指定变性和所有权
- **级别**：要求
- 构建包含裸指针的泛型结构体时，使用 `PhantomData<T>` 指定对 `T` 的变性和所有权
- 没有 `PhantomData`，`Drop` 可能导致 UB
- 示例：`struct Vec<T> { data: *const T, _marker: PhantomData<T> }`

---

## 联合体（Union）

### 除了 C FFI，不要使用 Union
- **级别**：要求
- Union 只用于 C FFI 兼容场景
- Rust 中优先使用 `enum` 或 `struct` 替代
- Union 的变体应使用 `Copy` 类型和 `ManuallyDrop` 包装

### Union 变体的借用共享生命周期
- **级别**：要求
- Union 变体的借用共享同一生命周期
- 不能同时拥有不同字段的可变借用
- 同一时间只能安全地访问一个字段

---

## 安全抽象（Safe Abstraction）

### 公共 unsafe 函数必须有 Safety 文档
- **级别**：要求
- 所有公共 `unsafe fn` 必须有 `# Safety` 文档段落
- 说明调用者需要保证的安全边界条件
- **Lint**: `missing_safety_doc`（Clippy, Style, warn）

### unsafe 函数中使用 assert! 而非 debug_assert!
- **级别**：要求
- `debug_assert!` 在 Release 模式下（`-C debug-assertions`）会被禁用
- 安全检查在 Release 中失效会导致 UB
- **Lint**: `debug_assert_with_mut_call`（Clippy, nursery, allow）

### 注意 Panic 导致的内存安全问题
- **级别**：建议
- Panic 触发栈展开，调用活跃变量的析构函数
- 可能导致双重释放或未初始化内存
- CVE-2020-36317：`String::retain()` bug，panic 后字符串处于不一致状态

### unsafe 代码编写者必须验证安全不变量
- **级别**：要求
- 检查三个方面：
  1. **逻辑一致性**：操作在逻辑上是正确的
  2. **纯度**：相同输入 → 相同输出（无副作用干扰）
  3. **语义约束**：参数必须是有效的
- CVE-2020-36323：不一致的 `Borrow` 实现暴露了未初始化字节

### 不要在公共 API 中暴露未初始化内存
- **级别**：要求
- 暴露未初始化内存可能导致 UB
- 修复：在传递给用户提供的 `Read` 实现前初始化缓冲区（如 `resize(0)`）

### 避免 Panic 导致的双重释放
- **级别**：要求
- 使用 `ptr::read` 后接可能 panic 的操作时，用 `ManuallyDrop` 防护
- `mem::forget` 必须放在 panic 安全的代码之后
- 示例流程：`ptr::read` → 可能 panic 的操作 → `mem::forget(src)`

### 手动实现 Send/Sync 时充分考虑安全性
- **级别**：要求
- CVE-2020-35905：`MappedMutexGuard` 的 `Send`/`Sync` 只考虑了 `T` 但守卫了 `U`
- 修复：添加 `PhantomData<&'a mut U>` 标记
- 手动实现 `Send`/`Sync` 时必须考虑所有涉及的类型参数

### 不要在公共 API 中随意暴露裸指针
- **级别**：建议
- 用户可能将指针设为 null 导致段错误
- 真实漏洞：`cache` crate 的 `Cached` 变体暴露了 `*const ManuallyDrop`

### 为安全抽象提供 unsafe _unchecked 版本
- **级别**：建议
- 遵循标准库模式：`get`（safe，检查 null）+ `get_unchecked`（unsafe，假设有效）
- 示例：`io_read_u32`（safe）+ `io_read_u32_unchecked`（unsafe）
- safe 版本做完整检查，unsafe 版本跳过检查以提升性能

---

## IO Safety

### 原始句柄没有访问限制
- **级别**：建议
- `AsRawFd`/`as_raw_fd` 返回的原始句柄（`RawFd`）没有访问限制
- `RawFd` 实现了 `AsRawFd`，任何人可以传入任意 FD 值
- Rust 1.63+ 引入了 `AsFd`/`OwnedFd`/`BorrowedFd` 用于 IO 安全
- 优先使用 owned/borrowed 句柄类型而非 raw 句柄
