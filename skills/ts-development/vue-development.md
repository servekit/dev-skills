---
name: vue-development
description: MUST use when writing, editing, or reviewing Vue code (.vue files, composables, Vue components). This is a COMPANION to ts-development — always apply ts-development first, then add Vue-specific rules from this skill. Triggers on .vue files, defineComponent, defineProps, defineEmits, ref/reactive/computed from 'vue', and Vue template syntax.
---

# Vue 编码最佳实践

基于 [Vue.js 官方风格指南](https://vuejs.org/style-guide/) 提炼，适用于 AI 辅助 Vue 开发。

本 skill 聚焦 **纯 Vue 3**（Composition API 优先），不涉及 Nuxt 等框架。

## 与 ts-development 的关系

**本 skill 是 `ts-development` 的补充，不是替代。** 写 Vue 代码时两个 skill 都适用：

- **`ts-development`**（[SKILL.md](SKILL.md)）— 所有 TS/JS 基础规则（命名、类型系统、控制流、ESLint 基础配置）
- **`vue-development`**（本文件）— Vue 专用规则（组件设计、模板、响应式、样式、Composition API）

TypeScript 基础问题参考 ts-development，不再重复。

## 核心原则

优先级对标 Vue 官方风格指南的四级分类：

| 级别 | 含义 | 要求 |
|------|------|------|
| **A: Essential** | 错误预防 | 必须遵守，几乎无例外 |
| **B: Strongly Recommended** | 可读性改善 | 强烈建议，违反需充分理由 |
| **C: Recommended** | 一致性选择 | 选一种，全项目统一 |
| **D: Use with Caution** | 潜在风险 | 了解风险后谨慎使用 |

## 1. 组件设计（Priority A — 必须遵守）

### 组件名必须多词
```vue
<!-- Bad: 单词组件名可能和 HTML 元素冲突 -->
<Item />

<!-- Good: 多词组件名 -->
<TodoItem />
```

### Props 定义必须详细
```vue
<script setup lang="ts">
// Bad: 只传字符串数组（原型阶段可以，提交代码不行）
const props = defineProps(['status']);

// Good: 至少标注类型
const props = defineProps({
  status: String,
});

// Best: 完整定义
const props = defineProps({
  status: {
    type: String as PropType<'syncing' | 'synced' | 'error'>,
    required: true,
    validator: (value: string) => ['syncing', 'synced', 'error'].includes(value),
  },
});
</script>
```

### v-for 必须带 key
```vue
<!-- Bad -->
<li v-for="todo in todos">{{ todo.text }}</li>

<!-- Good: 用唯一 id，不用索引 -->
<li v-for="todo in todos" :key="todo.id">{{ todo.text }}</li>
```

### 禁止 v-if 和 v-for 同元素
```vue
<!-- Bad: v-if 和 v-for 在同一元素 -->
<li v-for="user in users" v-if="user.isActive" :key="user.id">
  {{ user.name }}
</li>

<!-- Good: 用 computed 过滤 -->
<script setup lang="ts">
const activeUsers = computed(() => users.filter((u) => u.isActive));
</script>
<li v-for="user in activeUsers" :key="user.id">{{ user.name }}</li>

<!-- Good: 外层 template 控制 -->
<template v-for="user in users" :key="user.id">
  <li v-if="user.isActive">{{ user.name }}</li>
</template>
```

### 组件样式必须 scoped
```vue
<!-- Bad: 全局样式污染 -->
<style>
.btn-close { background-color: red; }
</style>

<!-- Good: scoped -->
<style scoped>
.btn-close { background-color: red; }
</style>

<!-- Good: CSS Modules -->
<style module>
.btnClose { background-color: red; }
</style>
```

## 2. 组件命名（Priority B — 强烈建议）

### 文件命名
```
components/
├── BaseButton.vue          # 基础组件用 Base/App/V 前缀
├── BaseIcon.vue
├── TodoList.vue            # PascalCase
├── TodoListItem.vue        # 紧耦合子组件用父名前缀
├── TodoListItemButton.vue
├── SearchButtonClear.vue   # 高层级词在前，修饰词在后
└── SearchButtonRun.vue
```

### 命名规则

| 场景 | 风格 | 示例 |
|------|------|------|
| 组件名 | PascalCase，多词 | `TodoList`, `SearchButton` |
| 组件文件名 | PascalCase | `TodoList.vue` |
| 基础组件 | Base/App/V 前缀 | `BaseButton.vue` |
| 子组件 | 父名前缀 | `TodoListItem.vue` |
| 组件内词序 | 高层级词在前 | `SearchButtonClear` |
| Props 声明 | camelCase | `greetingText` |
| Props 模板使用 | kebab-case（DOM）/ camelCase（SFC） | `greeting-text` / `greetingText` |
| 事件名 | kebab-case | `@update-value`, `@item-click` |
| Composable | use 前缀 | `useCounter`, `useFetch` |

### SFC 中组件使用 PascalCase
```vue
<!-- Good: SFC 模板中用 PascalCase -->
<MyComponent />
<TodoList />

<!-- Bad: SFC 模板中用 kebab-case（除非项目统一约定） -->
<my-component />
```

## 3. 模板规范（Priority B — 强烈建议）

### 模板表达式保持简单
```vue
<!-- Bad: 复杂表达式 -->
{{ fullName.split(' ').map(w => w[0].toUpperCase() + w.slice(1)).join(' ') }}

<!-- Good: 用 computed -->
{{ normalizedFullName }}
```

### 多属性换行
```vue
<!-- Bad -->
<img src="/logo.png" alt="Logo">
<MyComponent foo="a" bar="b" baz="c" />

<!-- Good -->
<img
  src="/logo.png"
  alt="Logo"
>
<MyComponent
  foo="a"
  bar="b"
  baz="c"
/>
```

### 指令简写一致性
```vue
<!-- Good: 全用简写（或全用全写，二选一，统一即可） -->
<input :value="text" @input="onInput">
<template #header>...</template>

<!-- Bad: 混用 -->
<input v-bind:value="text" :placeholder="hint">
```

### 自闭合组件
```vue
<!-- Good: SFC 中无内容组件自闭合 -->
<MyComponent />

<!-- Bad: SFC 中无内容却没自闭合 -->
<MyComponent></MyComponent>
```

### 属性引号
```vue
<!-- Bad -->
<input type=text>

<!-- Good: 始终加引号 -->
<input type="text">
```

## 4. SFC 结构（Priority C — 推荐统一）

### 顶层元素顺序
```vue
<!-- 推荐: <script> → <template> → <style> -->
<script setup lang="ts">
// ...
</script>

<template>
  <!-- ... -->
</template>

<style scoped>
/* ... */
</style>
```

- `<style>` 始终在最后
- `<script>` 和 `<template>` 顺序二选一，**全项目统一**
- 推荐用 `<script setup>` 语法糖

### 元素属性顺序
```
1. v-for
2. v-if / v-else-if / v-else / v-show
3. id
4. ref / key
5. v-model
6. 其他属性
7. v-on (@)
8. v-html / v-text
```

## 5. Composition API

### 响应式
```typescript
// ref: 基本类型和需要 .value 的场景
const count = ref(0);
const name = ref('');

// reactive: 对象/数组，不需要 .value
const state = reactive({
  users: [] as User[],
  loading: false,
});

// computed: 派生数据
const fullName = computed(() => `${firstName.value} ${lastName.value}`);

// Good: 简单 computed 拆分为多个
const basePrice = computed(() => cost.value / (1 - margin.value));
const discount = computed(() => basePrice.value * (discountPercent.value || 0));
const finalPrice = computed(() => basePrice.value - discount.value);
```

### Props 与 Emits
```typescript
// Props: 详细定义 + 类型
interface Props {
  title: string;
  count?: number;
  items: Item[];
}

const props = withDefaults(defineProps<Props>(), {
  count: 0,
});

// Emits: 显式声明
const emit = defineEmits<{
  update: [value: string];
  delete: [id: number];
}>();
```

### Composable 设计
```typescript
// use 前缀，返回 ref/computed
function useCounter(initial = 0) {
  const count = ref(initial);
  const doubled = computed(() => count.value * 2);

  function increment() {
    count.value++;
  }
  function reset() {
    count.value = initial;
  }

  return { count, doubled, increment, reset };
}
```

### 生命周期
```typescript
// 推荐顺序
onMounted(() => { /* 初始化 */ });
onUpdated(() => { /* 更新后 */ });
onUnmounted(() => { /* 清理：取消订阅、定时器等 */ });
```

## 6. 组件通信（Priority D — 谨慎使用）

### Props Down, Events Up
```vue
<!-- Good: 父传 props，子 emit 事件 -->
<!-- Parent -->
<ChildComponent
  :items="items"
  @update="handleUpdate"
  @delete="handleDelete"
/>

<!-- Child -->
<script setup lang="ts">
const props = defineProps<{ items: Item[] }>();
const emit = defineEmits<{
  update: [value: string];
  delete: [id: number];
}>();
</script>
```

### 禁止的模式
```typescript
// Bad: 直接修改 props
props.items.push(newItem);

// Bad: 用 $parent 访问父组件
const parent = getCurrentInstance()?.parent;

// Bad: v-model 直接绑定 prop（单向数据流破坏）
<input v-model="props.title">
```

### 跨层级通信选择

| 场景 | 方案 |
|------|------|
| 父子 | Props + Emits |
| 跨多层 | Provide / Inject |
| 全局状态 | Pinia |
| 兄弟组件 | 共同父组件中转 / Pinia |

## 7. 常见反模式

### 避免
- ❌ `v-if` 和 `v-for` 同一元素
- ❌ `v-for` 不带 `key`
- ❌ 单词组件名（`Item`, `List`）
- ❌ Props 只传字符串数组（`['status']`）
- ❌ 直接修改 props
- ❌ 使用 `$parent` 通信
- ❌ 模板中写复杂表达式
- ❌ 组件样式不 scoped
- ❌ `scoped` 样式中用元素选择器（用 class）
- ❌ 混用指令简写和全写
- ❌ 响应式对象解构丢失响应性
- ❌ 在 `v-for` 中用索引做 key（列表会变化时）

### 推荐
- ✅ 组件名多词、PascalCase
- ✅ Props 详细定义（类型、required、validator）
- ✅ 复杂逻辑提取到 computed 或 composable
- ✅ Props down, Events up
- ✅ 所有组件样式 scoped
- ✅ Composable 用 `use` 前缀
- ✅ `<script setup>` + TypeScript
- ✅ SFC 顶层元素顺序统一
- ✅ 基础组件用 `Base`/`App`/`V` 前缀
- ✅ 紧耦合子组件用父名前缀

## 8. 代码质量验证

基础验证流程（ESLint + Prettier + tsc）参考 `ts-development`。Vue 项目额外需要以下配置。

### Vue ESLint 插件

在 `ts-development` 的 ESLint 配置基础上，增加 Vue 专用插件：

```javascript
// 在 ts-development 的 eslint.config.js 基础上追加
import pluginVue from 'eslint-plugin-vue';

export default [
  // Vue 推荐规则（SFC 特有的模板/脚本检查）
  ...pluginVue.configs['flat/recommended'],
  {
    files: ['*.vue', '**/*.vue'],
    rules: {
      // Vue 特有规则
      'vue/multi-word-component-names': 'error',
      'vue/require-v-for-key': 'error',
      'vue/no-v-for-template-key-on-child': 'error',
      'vue/no-mutating-props': 'error',
      'vue/require-prop-types': 'error',
      'vue/html-self-closing': ['error', {
        html: { void: 'always', normal: 'always', component: 'always' },
      }],
      // 模板规范
      'vue/max-attributes-per-line': ['warn', { singleline: { max: 3 }, multiline: { max: 1 } }],
      'vue/html-indent': ['warn', { attribute: 1, baseIndent: 1 }],
    },
  },
];
```

**依赖安装**：

```bash
npm install -D eslint-plugin-vue vue-eslint-parser
```

### Vue 关键 ESLint 规则

| 规则 | 级别 | 对应本 skill 章节 |
|------|------|-----------------|
| `vue/multi-word-component-names` | error | 1. 组件设计 |
| `vue/require-v-for-key` | error | 1. 组件设计 |
| `vue/no-mutating-props` | error | 6. 组件通信 |
| `vue/require-prop-types` | error | 1. 组件设计 |
| `vue/html-self-closing` | error | 3. 模板规范 |
| `vue/max-attributes-per-line` | warn | 3. 模板规范 |

### 何时跳过

- 只修改 `<style>` 部分时，可跳过类型检查
- 没有 ESLint 时，至少手动确认 v-for key、v-if/v-for 分离、Props 类型定义
