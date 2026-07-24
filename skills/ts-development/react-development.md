---
name: react-development
description: MUST use when writing, editing, or reviewing React code (JSX, components, hooks, .tsx/.jsx files containing React components). This is a COMPANION to ts-development — always apply ts-development first, then add React-specific rules from this skill. Triggers on React component definitions, JSX syntax, hooks usage, and any file importing from 'react'.
---

# React 编码最佳实践

基于 [React 官方文档](https://react.dev) 提炼，适用于 AI 辅助 React 开发。

本 skill 聚焦 **纯 React**，不涉及 Next.js、React Native 等框架。

## 与 ts-development 的关系

**本 skill 是 `ts-development` 的补充，不是替代。** 写 React 代码时两个 skill 都适用：

- **`ts-development`**（[SKILL.md](SKILL.md)）— 所有 TS/JS 基础规则（命名、类型系统、控制流、ESLint 基础配置）
- **`react-development`**（本文件）— React 专用规则（组件设计、Hooks、Effect、状态管理、性能）

TypeScript 基础问题（`any`、`interface`、泛型等）参考 ts-development，不再重复。

## 核心原则

优先级从高到低：**正确性 > 可读性 > 性能 > 简洁性**

- **Correctness**：遵循 React 规则（hooks 规则、纯函数），避免隐藏 bug
- **Readability**：组件职责清晰，数据流一目了然
- **Performance**：不过早优化，只在测量后优化
- **Conciseness**：消除冗余状态和 Effect，用最简单的方案

## 1. 组件设计

### Thinking in React 五步法

1. **拆分 UI 为组件层级** — 按单一职责原则划分子组件，UI 结构与数据模型对应
2. **先构建静态版本** — 只用 props，不用 state，自顶向下渲染
3. **找到最小完整 state** — 用三问排除非 state：不变？来自父组件？可从现有数据计算？
4. **确定 state 归属** — 放在用到它的所有组件的最近公共父组件中
5. **添加反向数据流** — 子组件通过回调 props 更新父组件 state

### 组件文件组织

```
components/
├── Button/
│   └── Button.tsx        # 一个文件一个组件
├── SearchBar/
│   └── SearchBar.tsx
└── ProductTable/
    └── ProductTable.tsx
```

- 一个文件导出一个主组件，内部辅助组件不导出
- 文件名与组件名一致：`SearchBar.tsx` 导出 `SearchBar`
- 组件被多处复用或文件过长时，提取到独立文件

### 组件拆分时机
- 组件超过 50-100 行时考虑拆分
- 出现重复的 JSX 块时提取子组件
- 一个组件承担多个职责时按职责拆分

## 2. 命名规范

| 标识符类型 | 风格 | 示例 |
|-----------|------|------|
| 组件名 | PascalCase | `ProductTable`, `SearchBar` |
| 组件文件名 | PascalCase.tsx | `SearchBar.tsx` |
| Hooks（自定义） | `use` 前缀 | `useOnlineStatus`, `useWindowSize` |
| state 变量 | camelCase 配对 | `[filterText, setFilterText]` |
| 事件处理 props | `on` 前缀 | `onChange`, `onSubmit`, `onItemClick` |
| 事件处理函数 | `handle` 前缀 | `handleChange`, `handleSubmit` |
| 渲染函数 | render 前缀 | `renderItem`, `renderHeader` |
| 布尔 props | `is`/`has`/`should` 前缀 | `isVisible`, `hasError`, `shouldDisable` |

```tsx
// Good: 清晰的命名配对
function SearchBar({ filterText, onFilterTextChange }: SearchBarProps) {
  const [query, setQuery] = useState('');

  function handleSubmit() {
    onFilterTextChange(query);
  }

  return <form onSubmit={handleSubmit}>...</form>;
}
```

### Props 命名
```tsx
// Good: 回调 props 用 on 前缀
<Button onClick={handleClick} onSubmit={handleSubmit} />

// Good: 布尔 props 用 is/has/should
<Modal isVisible hasOverlay shouldCloseOnEsc />

// Good: children 用于插槽
<Card header="Title">content</Card>

// Good: render props / 函数子组件
<List renderItem={(item) => <Item key={item.id} {...item} />} />
```

## 3. Hooks 规则

**这些是规则，不是建议。违反会导致 bug。**

### 规则一：只在顶层调用 Hooks
```tsx
// Bad: 条件中调用 Hook
if (isActive) {
  const [count, setCount] = useState(0);
}

// Bad: 循环中调用 Hook
items.forEach((item) => {
  useEffect(() => { /* ... */ }, [item]);
});

// Bad: 嵌套函数中调用 Hook
function handleClick() {
  useState(0); // 不是组件顶层
}

// Good: 始终在组件函数顶层调用
function Counter({ isActive }: { isActive: boolean }) {
  const [count, setCount] = useState(0);
  useEffect(() => { /* ... */ }, []);
  // ...
}
```

### 规则二：只在 React 函数中调用 Hooks
```tsx
// Good: 组件中调用
function UserProfile() {
  const [user, setUser] = useState(null);
  // ...
}

// Good: 自定义 Hook 中调用
function useUser(userId: string) {
  const [user, setUser] = useState(null);
  // ...
  return user;
}

// Bad: 普通 JS 函数中调用
function formatDate(date: string) {
  const [format, setFormat] = useState('YYYY-MM-DD'); // 错误
  // ...
}
```

### 常用 Hooks 使用要点

| Hook | 核心要点 |
|------|---------|
| `useState` | 状态是不可变快照，不要直接修改。更新函数用函数式：`setCount(c => c + 1)` |
| `useEffect` | 仅用于同步外部系统。清理函数防止泄漏 |
| `useRef` | 不触发重渲染。DOM 引用或可变值的容器 |
| `useMemo` | 仅在计算耗时 1ms+ 时使用。不是缓存万能药 |
| `useCallback` | 仅在传递给已优化的子组件时有用。React Compiler 可自动处理 |
| `useContext` | 避免过度使用导致所有消费者重渲染。考虑拆分 context |
| `useReducer` | 多个关联 state 或下一个 state 依赖前一个时使用 |

## 4. 组件纯函数规则

### 组件必须是纯函数
```tsx
// Good: 相同输入，相同输出
function UserCard({ name, avatar }: UserCardProps) {
  return (
    <div>
      <img src={avatar} alt={name} />
      <h2>{name}</h2>
    </div>
  );
}

// Bad: 渲染中产生副作用
let count = 0;
function Counter() {
  count++; // 副作用！React 可能多次渲染
  return <p>{count}</p>;
}

// Bad: 渲染中修改 props/state
function UserCard({ user }: { user: User }) {
  user.name = user.name.toUpperCase(); // 修改 props！
  return <p>{user.name}</p>;
}

// Good: 创建新值
function UserCard({ user }: { user: User }) {
  const displayName = user.name.toUpperCase();
  return <p>{displayName}</p>;
}
```

### 不可变性
- **Props 是只读的** — 永远不要修改传入的 props
- **State 是快照** — 调用 setter 后，旧值不变，新值在下次渲染生效
- **不要修改传给 JSX 的值** — 所有修改在 JSX 创建之前完成

### Strict Mode
- 开发环境使用 `<StrictMode>` 检测不纯组件
- Strict Mode 会双重调用组件函数来暴露副作用
- 不要为了消除双重调用而关闭 Strict Mode

## 5. Effect 使用指南

### 核心原则

> Effect 是 React 范式的逃生舱。如果没有涉及外部系统，你不需要 Effect。

### 什么时候用 Effect

```tsx
// Good: 同步外部系统
useEffect(() => {
  const conn = createConnection(url);
  conn.connect();
  return () => conn.disconnect(); // 清理
}, [url]);

// Good: 数据请求（带竞态处理）
useEffect(() => {
  let ignore = false;
  fetchResults(query, page).then((data) => {
    if (!ignore) setResults(data);
  });
  return () => { ignore = true; };
}, [query, page]);
```

### 什么时候不用 Effect

```tsx
// BAD: 派生数据存 state + Effect
const [fullName, setFullName] = useState('');
useEffect(() => {
  setFullName(firstName + ' ' + lastName);
}, [firstName, lastName]);

// Good: 渲染时直接计算
const fullName = firstName + ' ' + lastName;
```

```tsx
// BAD: 重置 state 用 Effect
useEffect(() => {
  setComment('');
}, [userId]);

// Good: 用 key 重置整个组件
<Profile userId={userId} key={userId} />
```

```tsx
// BAD: 通知父组件用 Effect
useEffect(() => {
  onChange(isOn);
}, [isOn, onChange]);

// Good: 在事件处理中同时更新
function updateToggle(nextIsOn: boolean) {
  setIsOn(nextIsOn);
  onChange(nextIsOn);
}
```

```tsx
// BAD: 处理用户事件用 Effect
useEffect(() => {
  if (product.isInCart) {
    showNotification(`Added ${product.name}`);
  }
}, [product]);

// Good: 在事件处理中处理
function handleBuyClick() {
  addToCart(product);
  showNotification(`Added ${product.name}`);
}
```

### 决策框架

```
这段代码为什么需要运行？
├── 由特定交互触发（点击、提交）→ 事件处理函数
├── 由组件显示在屏幕上触发 → Effect
└── 可以从 props/state 计算 → 渲染时计算（什么都不用）
```

### Effect 清理
- 订阅、定时器、网络请求等必须在清理函数中取消
- 不要假设 Effect 只运行一次 — React 可能多次触发
- 使用 `ignore` 标志防止竞态

## 6. 状态管理

### State 最小化原则
```tsx
// Bad: 冗余状态
const [firstName, setFirstName] = useState('');
const [lastName, setLastName] = useState('');
const [fullName, setFullName] = useState(''); // 可从上面算出

// Good: 只存必要 state，其余计算
const [firstName, setFirstName] = useState('');
const [lastName, setLastName] = useState('');
const fullName = `${firstName} ${lastName}`;
```

### State 提升与下放
```tsx
// State 放在用到它的所有组件的最近公共父组件
function App() {
  const [filterText, setFilterText] = useState('');
  return (
    <>
      <SearchBar filterText={filterText} onFilterTextChange={setFilterText} />
      <ProductTable filterText={filterText} />
    </>
  );
}
```

- **提升**：多个兄弟组件需要共享 state → 提升到父组件
- **下放**：只有子组件用的 state → 下放到子组件，减少父组件重渲染范围
- **组合**：不需要 state 的组件用 `children` props 传入，避免不必要的重渲染

### State 更新
```tsx
// Good: 基于前一个 state 更新时用函数式
setCount(prev => prev + 1);
setItems(prev => [...prev, newItem]);
setUsers(prev => prev.filter(u => u.id !== removedId));

// Good: 批量更新（React 18+ 自动批处理）
function handleClick() {
  setCount(c => c + 1);    // 不会触发两次渲染
  setFlag(f => !f);        // React 自动批处理
}
```

### 选择状态管理工具

| 场景 | 推荐 |
|------|------|
| 组件内部状态 | `useState` / `useReducer` |
| 跨组件共享 | `useContext` + props |
| 复杂状态逻辑 | `useReducer` |
| 服务端状态 | TanStack Query / SWR |
| 全局应用状态 | Zustand / Jotai（按需引入） |

不引入全局状态库，除非 `useState` + `useContext` 已无法满足。

## 7. 性能优化

### 不要过早优化

- **先写正确的代码，再优化慢的部分**
- 用 React DevTools Profiler 测量，不要猜测
- React Compiler（自动批处理 + 自动 memo）能解决很多问题

### 何时用 useMemo
```tsx
// Good: 耗时计算（1ms+）
const sortedItems = useMemo(
  () => [...items].sort((a, b) => a.name.localeCompare(b.name)),
  [items],
);

// Bad: 不必要的 memo
const name = useMemo(() => user.name, [user.name]); // 简单取值，不值得
const displayItems = useMemo(() => items.slice(0, 10), [items]); // 极快操作
```

### 何时用 useCallback
```tsx
// Good: 传递给 memo 化的子组件
const MemoChild = React.memo(Child);

function Parent() {
  const handleClick = useCallback((id: string) => {
    setItems(prev => prev.filter(i => i.id !== id));
  }, []);

  return <MemoChild onClick={handleClick} />;
}

// Bad: 不传递给子组件时不需要
function Simple() {
  const handleClick = useCallback(() => { /* ... */ }, []); // 没意义
  return <button onClick={handleClick}>Click</button>;
}
```

### key 的正确使用
```tsx
// Good: 用稳定的唯一标识
{items.map(item => <Item key={item.id} {...item} />)}

// Bad: 用索引作为 key（列表会增删时）
{items.map((item, index) => <Item key={index} {...item} />)}

// Good: 用 key 重置组件
<Profile key={userId} userId={userId} />
```

### 避免不必要的重渲染
```tsx
// Good: 组件组合模式 — 避免传 state 给中间组件
function App() {
  const [theme, setTheme] = useState('dark');
  return (
    <Layout theme={theme}>
      <Panel>
        <ThemeButton theme={theme} onClick={() => setTheme(t => t === 'dark' ? 'light' : 'dark')} />
      </Panel>
    </Layout>
  );
}

// 更好: 如果 Layout 不需要 theme，用 children 传入
function Layout({ children }: { children: React.ReactNode }) {
  return <div className="layout">{children}</div>;
}
```

## 8. 常见反模式

### 避免
- ❌ 在渲染阶段产生副作用
- ❌ 直接修改 props 或 state
- ❌ 用 Effect 做数据转换（渲染时计算）
- ❌ 用 Effect 处理用户事件（用事件处理函数）
- ❌ 用 Effect 通知父组件（用事件处理函数同时更新）
- ❌ 用 Effect 重置 state（用 `key` prop）
- ❌ Hooks 放在条件/循环/嵌套函数中
- ❌ 在普通函数中调用 Hooks
- ❌ 用数组索引做 key（列表会变化时）
- ❌ `useMemo` / `useCallback` 滥用（过早优化）
- ❌ 匿名默认导出组件（`export default () => {}`）
- ❌ 多个 state 组成的链式 Effect（渲染时计算）
- ❌ 不清理 Effect 中的订阅/定时器/网络请求

### 推荐
- ✅ 组件保持纯函数 — 相同输入，相同输出
- ✅ 派生数据直接计算，不存 state
- ✅ Effect 仅用于同步外部系统
- ✅ 事件处理函数处理用户交互
- ✅ 用 `key` 重置组件 state
- ✅ 遵循 Hooks 规则（顶层 + React 函数内）
- ✅ 提取自定义 Hook 复用逻辑
- ✅ 用 `useMemo` 仅处理耗时计算
- ✅ state 最小化，能算出来的不存
- ✅ Strict Mode 开发环境必开
- ✅ 组件名始终用 PascalCase + 具名导出
- ✅ Effect 必须有清理函数（订阅/定时器/请求）

## 9. JSX 规范

### JSX 格式
```tsx
// Good: 多属性换行对齐
<Button
  type="submit"
  disabled={isSubmitting}
  onClick={handleSubmit}
>
  Submit
</Button>

// Good: 短属性可单行
<Button type="submit">Submit</Button>

// Good: 自闭合标签
<input type="text" value={name} onChange={handleChange} />

// Good: 条件渲染
{isLoggedIn && <UserPanel />}
{error ? <ErrorMessage /> : <Content />}
```

### 列表渲染
```tsx
// Good: map 返回 JSX，用唯一 id 做 key
function ProductList({ products }: { products: Product[] }) {
  return (
    <ul>
      {products.map((product) => (
        <li key={product.id}>
          <ProductCard product={product} />
        </li>
      ))}
    </ul>
  );
}
```

### Props 传递
```tsx
// Good: 展开传递多个 props
function EnhancedButton(props: ButtonProps) {
  return <Button {...props} className={cn(props.className, styles.enhanced)} />;
}

// Good: 显式传递关键 props
function UserCard({ user, onSelect }: UserCardProps) {
  return <Card onClick={() => onSelect(user.id)}>{user.name}</Card>;
}
```

### Children 类型
```tsx
// Good: 明确 children 类型
interface LayoutProps {
  children: React.ReactNode;
  header?: React.ReactNode;
}

// Good: 不需要 children 时省略
interface ButtonProps {
  onClick: () => void;
  disabled?: boolean;
}
```

## 10. TypeScript + React

TypeScript 基础规则（命名、类型系统、控制流等）参考 `ts-development`，这里只列出 React 特有的类型模式。

### 组件类型定义
```tsx
// Good: props 用 interface
interface ButtonProps {
  variant: 'primary' | 'secondary';
  size?: 'sm' | 'md' | 'lg';
  children: React.ReactNode;
  onClick: (e: React.MouseEvent<HTMLButtonElement>) => void;
}

function Button({ variant, size = 'md', children, onClick }: ButtonProps) {
  return <button onClick={onClick}>{children}</button>;
}
```

### 常用 React 类型速查
```tsx
// 事件类型
React.ChangeEvent<HTMLInputElement>
React.MouseEvent<HTMLButtonElement>
React.FormEvent<HTMLFormElement>
React.KeyboardEvent<HTMLInputElement>

// 子组件类型
React.ReactNode              // 任何可渲染内容
React.ReactElement           // React 元素
React.ComponentProps<typeof Component>  // 获取组件 props 类型

// Ref 类型
const inputRef = useRef<HTMLInputElement>(null);
const divRef = useRef<HTMLDivElement>(null);
```

### 泛型组件
```tsx
// Good: 泛型列表组件
interface ListProps<T> {
  items: T[];
  renderItem: (item: T) => React.ReactNode;
  keyExtractor: (item: T) => string;
}

function List<T>({ items, renderItem, keyExtractor }: ListProps<T>) {
  return (
    <ul>
      {items.map((item) => (
        <li key={keyExtractor(item)}>{renderItem(item)}</li>
      ))}
    </ul>
  );
}
```

### Hook 返回类型
```tsx
// Good: 自定义 Hook 返回明确类型
interface UseFetchResult<T> {
  data: T | null;
  loading: boolean;
  error: Error | null;
}

function useFetch<T>(url: string): UseFetchResult<T> {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  // ...
  return { data, loading, error };
}
```

## 11. 代码质量验证

基础验证流程（ESLint + Prettier + tsc）参考 `ts-development`。React 项目额外需要以下配置。

### React ESLint 插件

在 `ts-development` 的 ESLint 配置基础上，增加 React 专用插件：

```javascript
// 在 ts-development 的 eslint.config.js 基础上追加
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';

// 在 plugins 和 rules 中追加：
{
  plugins: {
    'react-hooks': reactHooks,
    'react-refresh': reactRefresh,
  },
  rules: {
    // React Hooks 规则（必须）
    'react-hooks/rules-of-hooks': 'error',
    'react-hooks/exhaustive-deps': 'warn',

    // React Refresh（开发体验）
    'react-refresh/only-export-components': [
      'warn',
      { allowConstantExport: true },
    ],
  },
}
```

**依赖安装**：

```bash
npm install -D eslint-plugin-react-hooks eslint-plugin-react-refresh
```

### React 关键 ESLint 规则

| 规则 | 级别 | 对应本 skill 章节 |
|------|------|-----------------|
| `react-hooks/rules-of-hooks` | error | 3. Hooks 规则 |
| `react-hooks/exhaustive-deps` | warn | 5. Effect 使用指南 |
| `react-refresh/only-export-components` | warn | 1. 组件文件组织 |

### 何时跳过

- 只修改样式时，可跳过类型检查（但 ESLint 仍建议执行）
- 没有 ESLint 时，至少手动确认 Hooks 规则和 Effect 依赖
