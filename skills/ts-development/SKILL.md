---
name: ts-development
description: MUST use when writing, editing, or reviewing ANY TypeScript (.ts, .tsx) or JavaScript (.js, .mjs, .cjs) code. Development guide based on Google TypeScript & JavaScript Style Guides — covers file structure, naming, type system, language features, control flow, strings, comments, anti-patterns, and lint verification (ESLint).
---

# TypeScript / JavaScript 编码最佳实践

基于 [Google TypeScript Style Guide](https://google.github.io/styleguide/tsguide.html) 和 [Google JavaScript Style Guide](https://google.github.io/styleguide/jsguide.html) 提炼，适用于 AI 辅助 TypeScript / JavaScript 开发。

**所有规则同时适用于 `.ts`/`.tsx` 和 `.js`/`.mjs`/`.cjs` 文件，除非标注为「TS-only」。**

## 关联 Skills

本 skill 覆盖 TS/JS 基础规则。以下框架有专用 skill，检测到对应模式时**必须额外加载**：

- **[react-development.md](react-development.md)** — React 专用规则（Hooks、Effect、状态管理、JSX、组件设计）
- **[vue-development.md](vue-development.md)** — Vue 专用规则（Composition API、模板、响应式、组件通信、SFC 规范）

```
当前文件是？
├── .ts / .js / .mjs / .cjs → 本 skill 足够
├── .tsx / .jsx
│   ├── 包含 React 组件 / Hooks / JSX → 本 skill + react-development
│   └── 非 React 代码 → 本 skill 足够
├── .vue
│   └── Vue SFC → 本 skill + vue-development
└── 不确定 → 先读本 skill，发现框架模式时追加对应 skill
```

## 核心原则

优先级从高到低：**正确性 > 可读性 > 一致性 > 简洁性**

- **Correctness**：类型安全，充分利用 TypeScript 类型系统
- **Readability**：代码意图一目了然，命名清晰
- **Consistency**：与项目现有风格保持一致，跨项目遵循统一规范
- **Conciseness**：消除冗余，但不牺牲可读性

## 1. 文件结构

### 文件编码
- 所有源文件使用 **UTF-8** 编码
- 空白字符仅允许 ASCII 水平空格（`0x20`），其他空白字符必须转义

### 文件组成顺序
```
1. Copyright（如有）
2. @fileoverview JSDoc（如有）
3. Imports
4. Implementation
```
各部分之间用**恰好一个空行**分隔。

### Import 规则
```typescript
// Good: 使用 import type 导入纯类型
import type { Options } from './options';
import { createHandler } from './handler';

// Good: 命名导入优先
import { readFile, writeFile } from 'fs';

// Good: 大型 API 可用命名空间导入
import * as fs from 'fs';

// Bad: 禁用 default export/import
import MyClass from './my-class'; // 不推荐

// Good: 使用具名导入
import { MyClass } from './my-class';
```

- **禁用 default export**：所有导出使用 named export
- **禁用可变导出**：不允许 `export let`，用函数返回值代替
- **相对路径导入**：优先使用 `./foo` 而非绝对路径
- **纯类型导入**：使用 `import type` 和 `export type`
- **禁用 namespace**：使用 ES modules，不用 `namespace` 和 `<reference>`
- **禁用 `require()`**：使用 `import`

## 2. 命名规范

### 命名风格

| 标识符类型 | 风格 | 示例 |
|-----------|------|------|
| 类、接口、类型、枚举、装饰器、类型参数 | `UpperCamelCase` | `HttpClient`, `RequestOptions` |
| 变量、参数、函数、方法、属性、模块别名 | `lowerCamelCase` | `maxLength`, `handleRequest` |
| 全局常量、枚举值 | `CONSTANT_CASE` | `MAX_RETRIES`, `HttpStatus.Ok` |
| 文件名 | `snake_case` | `http_client.ts`, `user_service.ts` |

### 缩写词处理
- 缩写词视为完整单词：`loadHttpUrl` 而非 `loadHTTPURL`
- 但在 CONSTANT_CASE 中全大写：`MAX_HTTP_CONNECTIONS`

### 禁止的命名模式
```typescript
// Bad: 不要在名称中包含类型信息
const nameString = 'Alice';
const userArray: User[] = [];

// Bad: 不要用 _ 前缀表示私有
class Foo {
  _privateField = 0;  // 用 private/private 修饰符
}

// Bad: 不要用 I 前缀命名接口
interface IUserService {}  // 用 UserService

// Bad: 不要用 opt_ 前缀表示可选参数
function render(opt_label?: string) {}  // 用 label?
```

### 命名要点
- **描述性命名**：避免模糊缩写，不用匈牙利命名法
- **短名称允许**：作用域不超过 10 行的变量可用短名（如 `i`, `e`）
- **不要单独使用 `_`**：不用的参数用 `_` 前缀加名字：`_unused`
- **布尔变量**：用 `is`/`has`/`should`/`can` 前缀
- **常量**：仅模块级不可变值用 CONSTANT_CASE，局部变量一律 lowerCamelCase

## 3. 类型系统

### 类型推断
```typescript
// Good: 简单类型靠推断
const name = 'Alice';
const items: string[] = [];

// Good: 复杂表达式显式标注
const config: AppConfig = JSON.parse(raw);

// Good: 空容器显式标注类型
const users: User[] = [];
const cache = new Map<string, User>();
```

- 简单类型依赖推断，不重复标注
- 复杂表达式、空泛型、多态场景必须显式标注
- 返回类型标注由作者/审阅者判断是否需要

### 禁用 `any`
```typescript
// Bad
function process(data: any) {}

// Good: 使用具体类型
function process(data: UserData) {}

// Good: 使用 unknown 需要类型收窄
function process(data: unknown) {
  if (typeof data === 'string') {
    // data: string
  }
}

// Good: 使用 Record<string, unknown> 代替 object
function process(data: Record<string, unknown>) {}
```

### interface 优先于 type
```typescript
// Good: 对象形状用 interface
interface UserInfo {
  name: string;
  age: number;
}

// Good: 联合类型、工具类型用 type
type Status = 'active' | 'inactive';
type ReadonlyUser = Readonly<UserInfo>;
```

- 对象形状定义优先用 `interface`（可扩展、可声明合并）
- 联合类型、交叉类型、工具类型用 `type`

### 数组类型
```typescript
// Good: 简单类型用 T[]
const names: string[] = [];

// Good: 复杂类型用 Array<T>
const matrix: Array<Array<number>> = [];
```

### null 和 undefined
```typescript
// Good: 用可选属性而非 | undefined
interface Config {
  host: string;
  port?: number;  // 优于 port: number | undefined
}

// Good: 类属性优先初始化而非可选
class Service {
  private logger = createLogger();  // 优于 logger?: Logger
}
```

- 两者都允许使用，无全局偏好
- **优先用 `?` 可选标记**而非 `| undefined`
- 类属性优先初始化，减少可选属性
- 类型别名中不包含 `| null` 或 `| undefined`

### 泛型
```typescript
// Good: 类型参数有意义
function merge<T extends object, U extends object>(a: T, b: U): T & U {
  return { ...a, ...b };
}

// Bad: 不要仅用于返回类型推断
function wrap<T>(value: T): { value: T } {
  return { value };
}
// Good: 调用时显式指定
const result = wrap<User>(defaultUser);
```

- 类型参数命名：单字母或 UpperCamelCase（`T`, `TResponse`, `TInput`）
- 调用仅返回类型泛型的 API 时必须显式指定类型参数

### 类型断言
```typescript
// Bad: 避免类型断言
const user = data as User;

// Good: 优先用运行时检查
function isUser(data: unknown): data is User {
  return typeof data === 'object' && data !== null && 'name' in data;
}
const user = isUser(data) ? data : getDefault();

// 如果必须断言，用 as 不用尖括号
const value = result as ExpectedType;

// Good: 双重断言用 unknown 中间过渡
const value = result as unknown as ExpectedType;
```

- 避免类型断言（`as`）和非空断言（`!`），优先运行时检查
- 使用 `as` 语法，不用尖括号 `<Type>value`
- 对象字面量用类型注解（`: Foo`）而非断言（`as Foo`）

### 其他类型规则
- **禁用包装类型**：用 `string` 而非 `String`，`number` 而非 `Number`
- **禁用 `{}`**：用 `unknown`、`Record<string, T>` 或 `object`
- **元组类型**：优于 Pair 风格接口，命名字段考虑内联对象类型
- **Map/Set**：优于 object-as-dict
- **映射/条件类型**：允许，但用最简单的构造

## 4. 语言特性

### 变量声明
```typescript
// Good: 优先 const，需要重赋值时用 let
const name = 'Alice';
let count = 0;

// Bad: 禁用 var
var x = 1;

// Good: 每个声明一个变量
const x = 1;
const y = 2;

// Bad: 不要在声明前使用
console.log(name);  // ReferenceError
const name = 'Alice';
```

- 始终使用 `const` 或 `let`，**禁用 `var`**
- 每个变量单独声明
- 变量必须先声明后使用

### 类
```typescript
class UserService {
  // Good: readonly 用于不重新赋值的属性
  readonly name: string;

  // Good: 在声明处初始化字段
  private users: User[] = [];

  // Good: 使用参数属性简化构造函数
  constructor(
    name: string,
    private readonly config: Config,
  ) {
    this.name = name;
  }

  // Good: getter 是纯函数
  get userCount(): number {
    return this.users.length;
  }

  // Good: 方法之间用空行分隔
  addUser(user: User): void {
    this.users.push(user);
  }

  // Good: 静态方法用模块局部函数替代
  // 而非 private static helper()
}
```

- **禁用 `#private`**：使用 TypeScript `private` 修饰符
- **`readonly`**：不重新赋值的属性必须标记
- **参数属性**：构造函数参数直接声明为类属性
- **getter 必须是纯函数**，不能有副作用
- **字段在声明处初始化**
- **静态方法**：优先用模块局部函数，不用 private static
- **不用 `prototype` 直接操作**（框架代码除外）
- **必须写构造函数括号**，即使空构造函数：`constructor() {}`

### 函数
```typescript
// Good: 顶层函数优先用函数声明
function createRouter(): Router {
  // ...
}

// Good: 方法体内的回调用箭头函数
class Handler {
  items = [1, 2, 3].map((item) => item * 2);
}

// Good: 简洁体与块体按需选择
const double = (n: number) => n * 2;           // 简洁体
const process = (data: string) => {            // 块体
  const result = parse(data);
  return result;
};

// Bad: 禁用 function 表达式（生成器除外）
const handler = function (event: Event) {};

// Good: 生成器可以用 function 表达式
const gen = function* () {
  yield 1;
};

// Good: rest 参数代替 arguments
function sum(...nums: number[]): number {
  return nums.reduce((a, b) => a + b, 0);
}
```

- 顶层函数优先用函数声明
- 回调优先用箭头函数（避免 `this` 绑定问题）
- **禁用 function 表达式**（生成器除外）
- 禁止用 `bind`、`call`、`apply` 重新绑定 `this`
- rest 参数代替 `arguments`
- 箭头函数单参数推荐加括号但非强制

### 解构与展开
```typescript
// Good: 数组解构
const [first, second] = items;

// Good: 对象解构（仅单层、无计算属性）
const { name, age } = user;

// Good: 展开语法（类型匹配）
const copy = { ...original, name: 'updated' };
const merged = [...arr1, ...arr2];

// Bad: 展开类型不匹配
const str: string = ...numArray;  // 类型不匹配
```

### 枚举
```typescript
// Good: 使用普通 enum
enum Direction {
  Up = 'UP',
  Down = 'DOWN',
  Left = 'LEFT',
  Right = 'RIGHT',
}

// Bad: 禁用 const enum
const enum Color { Red, Green, Blue }
```

- 禁用 `const enum`，使用普通 `enum`
- 枚举值用 CONSTANT_CASE
- 枚举类型名用 UpperCamelCase

### 装饰器
- **不要自定义装饰器**
- 仅使用框架提供的装饰器（Angular、NestJS 等）
- 装饰器紧跟在被装饰符号前面

## 5. 控制流

### 块语句
```typescript
// Good: 始终使用花括号
if (isValid) {
  process();
}

// Good: 简短 if 允许省略花括号
if (!item) return null;
```

- `if`/`else`/`for`/`while` 等必须使用花括号
- 例外：单行 `if` 可以省略花括号

### 相等比较
```typescript
// Good: 始终用 ===
if (count === 0) {}

// Good: == null 检查 null 和 undefined
if (value == null) {}  // 等同于 value === null || value === undefined

// Bad: 不要用 ==
if (count == 0) {}
```

- **始终用 `===` 和 `!==`**
- 唯一例外：`== null` 可同时检查 `null` 和 `undefined`

### switch 语句
```typescript
switch (status) {
  case 'active':
    handleActive();
    break;
  case 'inactive':
    handleInactive();
    break;
  default:
    handleUnknown();
}
```

- 必须包含 `default` 分支（放在最后）
- 非空 case 不能 fall-through（必须 `break`/`return`）

### 异常处理
```typescript
// Good: 始终用 new Error()
throw new Error('something went wrong');

// Good: try 块聚焦可能抛出异常的代码
const data = readFile(path);
try {
  const parsed = JSON.parse(data);
  return parsed;
} catch (error) {
  // Good: 假设 catch 到的是 Error
  throw new Error(`parse failed: ${(error as Error).message}`);
}

// Bad: 空 catch 块
try {
  riskyOperation();
} catch (e) {
  // 至少要有注释说明为什么忽略
}

// Bad: 抛出非 Error 值
throw 'something went wrong';
throw 404;
```

- 始终 `throw new Error()` 或 Error 子类
- 不要抛出非 Error 值
- try 块保持聚焦，不抛异常的代码移到外面
- 空 catch 块必须有注释解释原因
- 假设 catch 到的错误是 `Error` 类型

### 赋值在条件语句中
```typescript
// Bad: 避免条件语句中的赋值
if (result = compute()) {}

// 如果确实需要，用双重括号标注意图
if ((result = compute())) {}
```

## 6. 字符串与数字

### 字符串
```typescript
// Good: 使用单引号
const name = 'Alice';

// Good: 复杂拼接用模板字符串
const message = `Hello, ${user.name}! You have ${count} items.`;

// Good: 多行字符串用模板字符串
const query = `
  SELECT *
  FROM users
  WHERE active = true
`;

// Bad: 禁用行续符（反斜杠换行）
const text = 'hello \
world';
```

- **使用单引号 `'`** 而非双引号 `"`
- 复杂拼接用模板字符串（反引号）
- 禁用反斜杠行续符

### 数字
```typescript
// Good: 小写前缀
const hex = 0xff;
const octal = 0o77;
const binary = 0b1010;

// Bad: 不必要的前导零
const num = 0777;
```

### 类型转换
```typescript
// Good: 推荐的转换方式
String(42);           // → string
Boolean(value);       // → boolean
!!value;              // → boolean（简写）
Number('42');         // → number
parseFloat(intStr);   // 仅非十进制时，必须带基数验证

// Good: 模板字符串转字符串
const str = `${42}`;

// Bad: 禁用一元加号转数字
const num = +'42';

// Bad: parseInt/parseFloat 仅用于非十进制
parseInt('42', 10);   // 用 Number('42') 代替
parseInt('ff', 16);   // Good: 非十进制可以用

// Bad: 条件语句中不必要地用 !!
if (!!value) {}       // if (value) 就够了
while (!!items.length) {}  // while (items.length) 就够了
```

- 转 string：`String()`、模板字符串
- 转 boolean：`Boolean()`、`!!`
- 转 number：`Number()`（非十进制用 `parseInt` + `NaN` 检查）
- 禁用一元加号（`+`）做类型转换
- 条件语句中不要双重否定（`!!`）

## 7. 注释与文档

### JSDoc vs 普通注释
```typescript
/**
 * JSDoc 用于文档（面向使用者）
 * 支持 Markdown 格式
 * @param input - 输入数据
 * @returns 处理结果
 */
export function process(input: string): Result {
  // 普通注释用于实现说明（面向维护者）
  // 解释 Why 而非 What
}
```

- `/** */` 用于文档注释，`//` 用于实现注释
- 多行注释用多个 `//`，不用 `/* */`
- JSDoc 使用 Markdown 格式

### 必须文档化的内容
- **所有顶层导出**必须有 JSDoc
- 不明显的属性和方法应有文档
- 类/方法/函数注释包含足够的使用信息

### JSDoc 格式
```typescript
/**
 * 创建用户服务实例。
 *
 * @param config - 服务配置，包含数据库连接信息
 * @param options - 可选的行为参数
 * @returns 初始化完成的 UserService 实例
 *
 * @throws {ConfigError} 配置无效时抛出
 *
 * @example
 * ```ts
 * const service = createUserService(config, { timeout: 5000 });
 * ```
 */
export function createUserService(config: ServiceConfig, options?: Options): UserService {
  // ...
}
```

- 标签（`@param`, `@returns`）独占一行
- 换行的标签块缩进 4 空格
- 不要在 JSDoc 中重复 TypeScript 已有的类型信息
- `@param`/`@returns` 仅在能补充信息时才写
- JSDoc 放在装饰器之前

### 方法描述风格
- 使用第三人称动词短语：`Returns the user count` 而非 `Return the user count`

### 废弃标注
```typescript
/**
 * @deprecated 使用 `createUserService` 代替。此函数将在 v3 移除。
 */
export function createService(config: Config): Service {
  // ...
}
```

## 8. 禁用特性

以下特性**完全禁用**：

```typescript
// Bad: 原始类型的包装对象
new String('hello');
new Boolean(true);
new Number(42);

// Bad: 依赖自动分号插入（ASI）
const x = 1
const y = 2

// Bad: const enum
const enum Color { Red, Green, Blue }

// Bad: debugger 语句
debugger;

// Bad: with 语句
with (obj) {}

// Bad: eval 和动态代码执行
eval('console.log("x")');
new Function('return 1');

// Bad: 修改内置对象原型
Array.prototype.customMethod = function () {};

// Bad: @ts-ignore
// @ts-ignore
const x: string = 42;

// Good: 仅在测试中少量使用 @ts-expect-error
// @ts-expect-error — testing invalid input
const result = process(badInput);
```

- 禁用原始类型包装对象：`new String()`、`new Boolean()`、`new Number()`
- **始终使用分号**，不依赖 ASI
- 禁用 `const enum`、`debugger`、`with`、`eval`
- 禁用 `@ts-ignore`（测试中可用 `@ts-expect-error`）
- 禁止修改内置对象原型

## 9. 常见反模式

### 避免
- ❌ 使用 `var` 声明变量
- ❌ 使用 `any` 类型
- ❌ 使用 default export
- ❌ 使用 `namespace` 和 `<reference>`
- ❌ 使用 `#private` 字段（用 TypeScript `private`）
- ❌ 类型断言代替运行时检查
- ❌ `for...in` 遍历数组（用 `for...of`）
- ❌ 未过滤的 `for...in`（用 `Object.keys()`）
- ❌ `==` 代替 `===`（`== null` 除外）
- ❌ 在 JSDoc 中重复 TypeScript 类型
- ❌ 使用包装类型：`String`、`Number`、`Boolean`、`Object` 作为类型
- ❌ 用 `Object()` / `Array()` 构造函数
- ❌ 函数表达式中重新绑定 `this`（用箭头函数）
- ❌ 魔法数字（硬编码常量）

### 推荐
- ✅ 用 `const`/`let`，禁 `var`
- ✅ 用 `interface` 定义对象形状
- ✅ 用 named export/import
- ✅ 用 `import type` 导入纯类型
- ✅ 用 `readonly` 标记不可变属性
- ✅ 用运行时检查（类型守卫）代替类型断言
- ✅ 用 `for...of` 遍历数组，`Object.keys()`/`values()`/`entries()` 遍历对象
- ✅ 用 `Map`/`Set` 代替 object-as-dict
- ✅ 用箭头函数处理 `this` 绑定
- ✅ 解构赋值简化代码

## 10. JavaScript 文件补充说明

本 skill 所有规则对 `.js`/`.mjs`/`.cjs` 文件同样生效。以下是 JS 文件的差异点。

### JSDoc 类型注释代替 TypeScript 语法

JS 文件没有编译时类型检查，用 JSDoc 提供类型信息：

```javascript
/**
 * @typedef {Object} User
 * @property {string} name
 * @property {number} age
 */

/** @type {User[]} */
const users = [];

/**
 * @param {string} input
 * @returns {Result}
 */
function process(input) {
  // ...
}
```

### 不适用的 TS-only 规则

以下规则在 `.js` 文件中不适用，跳过即可：

| 规则 | 原因 |
|------|------|
| 禁用 `any` | JS 没有 `any` 关键字 |
| interface 优先于 type | JS 无 interface/type 语法 |
| 泛型 | JS 无泛型语法 |
| `import type` / `export type` | JS 无 type-only import |
| `readonly` | JS 无 readonly 修饰符 |
| 类型断言（`as`） | JS 无类型断言 |
| `const enum` | JS 无 enum |
| 参数属性 | JS 无 TypeScript 参数属性 |
| `tsc --noEmit` | JS 无需 TypeScript 编译 |

### JS 文件 Lint 检查

```bash
# ESLint 对 JS 文件同样有效
npx eslint <file.js>

# JS 文件不需要 tsc，ESLint 足够
# 如果项目同时有 tsconfig.json，JS 文件通常被 exclude
```

## 11. 代码质量验证

**写完或修改 TypeScript 代码后，必须依次执行以下验证步骤。**

### 第一步：格式化

```bash
# 格式化当前修改的文件
npx prettier --write <file.ts>

# 格式化整个项目
npx prettier --write "src/**/*.ts"
```

如果项目没有配置 Prettier，跳过此步。遵循项目已有配置。

### 第二步：Lint 检查

```bash
# 对修改的文件运行 ESLint
npx eslint <file.ts>

# 对整个项目运行 ESLint
npx eslint "src/**/*.ts"

# 自动修复可修复的问题
npx eslint --fix <file.ts>
```

### 第三步：类型检查

```bash
# 运行 TypeScript 编译器类型检查
npx tsc --noEmit

# 仅检查特定文件
npx tsc --noEmit <file.ts>
```

### 第四步：修复与迭代

1. 如果 ESLint 报错，**先修复所有问题再继续**
2. 修复后重新跑 `prettier --write`（修复过程可能改变格式）
3. 重新跑 `eslint` 和 `tsc --noEmit` 确认通过
4. 循环直到全部通过

### ESLint 配置

如果项目没有 ESLint 配置，使用以下最小配置（使用 flat config，ESLint 9+）：

```javascript
// eslint.config.js
import tseslint from 'typescript-eslint';

export default tseslint.config(
  ...tseslint.configs.recommended,
  {
    rules: {
      // 命名规范
      '@typescript-eslint/naming-convention': [
        'error',
        { selector: 'default', format: ['camelCase'] },
        { selector: 'variable', modifiers: ['const', 'global'], format: ['UPPER_CASE'] },
        { selector: 'variable', modifiers: ['const'], format: ['camelCase', 'UPPER_CASE'] },
        { selector: 'typeLike', format: ['PascalCase'] },
        { selector: 'enumMember', format: ['UPPER_CASE'] },
      ],

      // 类型安全
      '@typescript-eslint/no-explicit-any': 'error',
      '@typescript-eslint/no-unsafe-assignment': 'error',
      '@typescript-eslint/no-unsafe-call': 'error',
      '@typescript-eslint/no-unsafe-member-access': 'error',
      '@typescript-eslint/no-unsafe-return': 'error',
      'no-type-assertion/no-type-assertion': 'off', // 用 warn 如果项目可用

      // 代码质量
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
      '@typescript-eslint/explicit-function-return-type': 'off',
      '@typescript-eslint/no-non-null-assertion': 'warn',
      '@typescript-eslint/consistent-type-imports': ['error', { prefer: 'type-imports' }],
      '@typescript-eslint/consistent-type-exports': 'error',

      // 禁用特性
      'no-var': 'error',
      'no-eval': 'error',
      'no-implied-eval': 'error',
      '@typescript-eslint/no-namespace': 'error',
      'no-console': 'warn',
    },
  },
  {
    // 测试文件放宽规则
    files: ['**/*.test.ts', '**/*.spec.ts', '**/test/**/*.ts'],
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-non-null-assertion': 'off',
    },
  },
);
```

**依赖安装**（如果项目没有）：

```bash
npm install -D eslint typescript-eslint @typescript-eslint/parser @typescript-eslint/eslint-plugin
```

### 关键 ESLint 规则说明

| 规则 | 级别 | 对应本 skill 章节 |
|------|------|-----------------|
| `no-var` | error | 4. 语言特性 - 变量声明 |
| `@typescript-eslint/no-explicit-any` | error | 3. 类型系统 - 禁用 any |
| `@typescript-eslint/naming-convention` | error | 2. 命名规范 |
| `@typescript-eslint/consistent-type-imports` | error | 1. 文件结构 - Import 规则 |
| `@typescript-eslint/consistent-type-exports` | error | 1. 文件结构 - Export 规则 |
| `@typescript-eslint/no-namespace` | error | 1. 文件结构 - 禁用 namespace |
| `@typescript-eslint/no-unused-vars` | error | 4. 语言特性 - 变量声明 |
| `@typescript-eslint/no-non-null-assertion` | warn | 3. 类型系统 - 类型断言 |
| `no-eval` | error | 8. 禁用特性 |
| `no-console` | warn | 生产代码不应有 console |
| `eqeqeq` | error | 5. 控制流 - 相等比较 |
| `@typescript-eslint/switch-exhaustiveness-check` | error | 5. 控制流 - switch 语句 |

### 进阶配置（推荐，需类型信息）

在项目稳定后，可升级到需要类型检查的规则集：

```javascript
// 将 recommended 替换为 strict
export default tseslint.config(
  ...tseslint.configs.strict,
  // ... 其余规则同上
);
```

`strict` 额外包含：
- `no-floating-promises`：确保 Promise 被处理
- `no-misused-promises`：防止 Promise 被误用
- `await-thenable`：`await` 只用于 thenable
- `no-unnecessary-type-assertion`：移除冗余断言

### 项目级配置优先

如果项目根目录已有 `eslint.config.*` 或 `.eslintrc.*`，**遵循项目配置**。不要覆盖已有配置。仅在全新项目中使用上面提供的最小配置。

### 何时跳过

- 只修改注释或文档时，可跳过 lint（但类型检查仍建议执行）
- ESLint 未安装时，至少执行 `npx tsc --noEmit` 作为最小检查
