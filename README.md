# dev-skills

一套面向 AI 编码 agent 的领域开发 Skills —— Go / Rust / TypeScript / OPA / Protobuf 等编码规范,外加基于 [`github.com/servekit/go-common`](https://github.com/servekit/go-common) 的服务工具链。

每个 Skill 是一份带触发条件的规范文档,agent 自动发现并按场景加载。当前以 Claude Code 插件形式交付,后续会扩展到其它 agent(见 [Quickstart](#quickstart))。

## Quickstart

| Agent | 状态 |
|-------|------|
| Claude Code | ✅ 已支持 — [安装](#claude-code) |
| Codex · Cursor · Gemini CLI · 其它 | 🚧 规划中 |

## How it works

两层机制让 skills 被可靠使用:

1. **自动发现(Discovery)** —— agent 扫描 `skills/*/SKILL.md`,靠 frontmatter 的 `name` + `description` 注册到目录、按场景匹配加载。新增 skill 无需任何注册声明。
2. **强制使用(Forcing)** —— SessionStart hook 注入 `using-dev-skills` 路由引导文(何时用哪个 skill)+ 一份运行时生成的 skills 索引。索引从 frontmatter 自动生成,新增 skill 无需改 hook。

> 第 1 层(自动发现)与 agent 无关;第 2 层(注入 / 强制)是 agent 特定的适配 —— Claude Code 下由 `hooks/session-start` 实现,换 agent 只换这一层。

项目布局:

```
dev-skills/
├── .claude-plugin/        插件清单 + marketplace(Claude Code 交付形态)
├── hooks/
│   ├── hooks.json         SessionStart 注册
│   └── session-start      注入路由引导文 + 自动索引 skills
├── skills/<name>/         每个 skill 一个目录,入口 SKILL.md
└── README.md
```

## Installation

安装方式随 agent 而异,每个 agent 独立适配。

### Claude Code

```
/plugin marketplace add https://github.com/servekit/dev-skills
/plugin install dev-skills@dev-skills
```

新开会话后,SessionStart hook 自动注入路由引导文 + skills 索引。

> 更多 agent 的适配在路上。dev-skills 的 skills 是与 agent 无关的 Markdown —— 为一个新 agent 提供支持,只需补一层「自动发现 + 注入」的适配(参考 `hooks/`)。

## What's Inside

### Skills

`skills/` 下分两类:

| Skill | 覆盖 |
|-------|------|
| golang-development | Go 编码规范（命名/错误/并发/测试/gofmt/golangci-lint） |
| rust-development | Rust 编码规范（rustfmt/clippy/unsafe/async） |
| ts-development | TypeScript / JavaScript（含 React / Vue 子文档） |
| opa-development | OPA / Rego 策略 |
| proto-development | Protobuf / Buf / protovalidate |
| gorm-cli-development | GORM CLI（gorm gen / dal / 类型安全 CRUD） |
| golang-service-development | go-common `-service` 架构（目录 / 分层 / thirdcall / lifecycle / 脚手架） |
| golang-service-docker | 把 grpcx Go 服务打包成 Docker 镜像 + compose 开发栈 |

前 6 个是**通用语言标准**,适用对应语言的任意项目。后 2 个是 **go-common 服务工具链**,专属于 [`github.com/servekit/go-common`](https://github.com/servekit/go-common) —— servekit 内部的一个公共组件库 / 函数库(封装 grpcx / configx / lifecycle 等能力),**不要套用到非 go-common 的项目**。

`using-dev-skills` 是 meta-skill(操作指南:何时用哪个 skill),由 hook 自动注入。

## 给 skill 作者

- 每个 skill 一个目录 `skills/<name>/`,入口必须是 `SKILL.md`,frontmatter 带 `name`(与目录名一致)+ `description`。
- `description` 是 agent 决定「何时用这个 skill」的唯一依据 —— 写清楚触发条件。
- **新增 skill 无需改适配层**:发现机制运行时自动扫描 `skills/*/SKILL.md` 生成索引(Claude Code 下由 `session-start` hook 注入)。但 `using-dev-skills/SKILL.md` 里的路由表是人工维护的,加完 skill 记得更新路由。
- 子文档(如 `proto-development/proto-best-practices.md`)由入口按需加载,不进自动索引。
- (Claude Code)改完 hook 可本地验证:`./hooks/session-start` 输出应是合法 JSON。
