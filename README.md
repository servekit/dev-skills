# dev-skills

一个 Claude Code 插件 —— 一套面向 AI 编码的领域开发 Skills(Go / Rust / TypeScript / OPA / Protobuf 等编码规范,外加 go-common 服务工具链)。

Skills 由 Claude Code 自动发现,SessionStart hook 注入路由引导文并强制使用(机制参考 [superpowers](https://github.com/obra/superpowers))。

## 作为插件安装

```
/plugin marketplace add https://github.com/servekit/dev-skills
/plugin install dev-skills@dev-skills
```

## 结构

```
dev-skills/
├── .claude-plugin/          插件清单(plugin.json)+ marketplace(marketplace.json)
├── hooks/
│   ├── hooks.json           SessionStart 注册
│   └── session-start        注入引导文 + 自动索引 skills
├── skills/                  每个 skill 一个目录,入口 SKILL.md
└── README.md
```

## Skills

`skills/` 下分两层:

**通用语言标准** —— 适用对应语言的任意项目:

| Skill | 覆盖 |
|-------|------|
| golang-development | Go 编码规范（命名/错误/并发/测试/gofmt/golangci-lint） |
| rust-development | Rust 编码规范（rustfmt/clippy/unsafe/async） |
| ts-development | TypeScript / JavaScript（含 React / Vue 子文档） |
| opa-development | OPA / Rego 策略 |
| proto-development | Protobuf / Buf / protovalidate |
| gorm-cli-development | GORM CLI（gorm gen / dal / 类型安全 CRUD） |

**go-common 服务工具链** —— 专属于 `github.com/servekit/go-common` 生态（grpcx / configx / lifecycle），**不要套用到非 go-common 的项目**:

| Skill | 覆盖 |
|-------|------|
| golang-service-development | go-common `-service` 架构（目录 / 分层 / thirdcall / lifecycle / 脚手架） |
| golang-service-docker | 把 grpcx Go 服务打包成 Docker 镜像 + compose 开发栈 |

`using-dev-skills` 是 meta-skill（操作指南：何时用哪个 skill），由 SessionStart hook 自动注入。

## 给 skill 作者

- 每个 skill 一个目录 `skills/<name>/`,入口必须是 `SKILL.md`,frontmatter 带 `name`(与目录名一致)+ `description`。
- `description` 是 Claude 决定「何时用这个 skill」的唯一依据 —— 写清楚触发条件。
- **加 skill 不用改 hook**:`session-start` 运行时自动扫描 `skills/*/SKILL.md` 生成索引。但 `using-dev-skills/SKILL.md` 里的路由表是人工维护的,加完 skill 记得更新路由。
- 子文档(如 `proto-development/proto-best-practices.md`)由入口按需加载,不进自动索引。
- 改完 hook 可本地验证:`./hooks/session-start` 输出应是合法 JSON。
