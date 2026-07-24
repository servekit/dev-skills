---
name: proto-development
description: "MUST load when writing, editing, reviewing, or creating any .proto file, or any Protobuf/Buf/protovalidate related task, or adding field validation/constraints to proto messages. This is the ONLY entry point for the proto-development skill set. Read this file first, then load sub-documents as needed. Trigger keywords: proto, protobuf, buf, protovalidate, buf.yaml, buf.gen.yaml, gRPC, .proto, protoc, compile proto, generate proto, field validation, validate, protovalidate rules, cel expression"
---

# Protobuf 项目开发指南

**这是 proto-development skill set 的唯一入口。Agent 必须先读本文件，再按需加载子文档。**

## 版本锁定

本 skill set 基于以下版本编写，Agent 应注意版本差异：

| 工具/规范 | 版本 |
|----------|------|
| Protobuf 语法 | proto3（proto2 仅在预定义规则的定义文件中使用） |
| Buf CLI | v1.68.4+ |
| buf.yaml / buf.gen.yaml | v2 格式 |
| Protovalidate | `buf.build/bufbuild/protovalidate`（BSR 最新版） |
| Go | 1.23+ |

如果项目使用不同版本，Agent 应先读取项目中的实际配置（buf.yaml、go.mod）再决定如何应用规范。

## 上下文传承（重要）

Agent 在应用本 skill 之前，**必须先检查项目现有状态**：

1. **如果项目已有 `buf.yaml`** → 先读取现有配置（modules、deps、lint、breaking），在此基础上调整，不要从零开始
2. **如果项目已有 `buf.gen.yaml`** → 先读取现有插件和 managed mode 配置
3. **如果项目已有 `.proto` 文件** → 先观察现有的 package 命名风格、目录结构、是否用了 protovalidate，保持一致性
4. **如果是全新项目** → 按本 skill set 的推荐结构创建

## 子文档路由

本 skill set 包含三个子文档，按需加载：

- **[proto-best-practices.md](proto-best-practices.md)** — Proto 编写最佳实践（主文档）
- **[buf-usage.md](buf-usage.md)** — Buf 工具使用指南
- **[protovalidate.md](protovalidate.md)** — 字段验证规则

## 何时阅读哪个文档

### 始终先读：proto-best-practices.md

以下场景**必须**先阅读此文档：

- 编写或修改 `.proto` 文件（消息、枚举、服务定义）
- 决定字段类型（int32 vs sint32、string vs bytes）
- 命名规范（message/field/enum/service 命名）
- 目录结构设计（package 与目录对应、1-1-1 规则）
- 理解兼容性（Wire-safe/Wire-unsafe、JSON 兼容）
- 选择 Well-Known Types（Timestamp/Duration/Empty/Any）

这是**默认文档**。任何 proto 相关工作都应先参考此文档中的规范。

### 当涉及 Buf 工具操作时：追加阅读 buf-usage.md

以下场景**追加**阅读此文档：

- 初始化 proto 项目（`buf config init`）
- 配置 buf.yaml（模块/工作区/依赖）
- 配置 buf.gen.yaml（代码生成、managed mode、插件）
- 运行 buf generate / buf lint / buf breaking / buf format
- 设置 CI（GitHub Actions proto 检查）
- 解决 buf 相关错误（go_package 不对、generate 输出为空）

**判断依据：** 如果任务涉及 `buf.yaml`、`buf.gen.yaml`、`buf` CLI 命令、代码生成配置、CI 配置，就需要此文档。

### 当需要字段验证时：追加阅读 protovalidate.md

以下场景**追加**阅读此文档：

- 给字段添加验证规则（邮箱、UUID、IP、长度、范围）
- 使用 `(buf.validate.field)` 标准规则
- 跨字段验证（开始时间 < 结束时间、密码确认匹配）
- 编写自定义 CEL 表达式
- 创建可复用的预定义规则
- Go 运行时集成（`protovalidate.Validate()`、gRPC 拦截器）

**判断依据：** 如果 proto 文件中出现了 `(buf.validate.field)` 或 `(buf.validate.message)` 注解，就需要此文档。

### 典型工作流

| 任务 | 需要的文档 |
|------|-----------|
| 新建 proto 项目 | buf-usage → proto-best-practices |
| 编写 .proto 文件 | proto-best-practices（+ protovalidate 如需验证） |
| 给字段加验证规则 | protovalidate |
| 配置代码生成 | buf-usage |
| 设置 proto CI | buf-usage |
| 重构/迁移 proto | proto-best-practices → buf-usage |
| 完整的 proto 服务开发 | 三篇都读 |

## 快速决策

```
用户要做什么？
├── 写/改 .proto 文件 → proto-best-practices.md
│   ├── 涉及 buf validate 注解？ → + protovalidate.md
│   └── 不涉及验证 → proto-best-practices.md 足够
├── 配置/运行 buf 命令 → buf-usage.md
│   ├── buf.yaml / buf.gen.yaml 配置问题 → buf-usage.md
│   ├── buf generate / lint / breaking 报错 → buf-usage.md
│   └── managed mode / 插件配置 → buf-usage.md
├── 加字段验证 → protovalidate.md
│   ├── 标准规则（email/uuid/ip/范围） → protovalidate.md 标准规则章节
│   ├── 跨字段验证 → protovalidate.md CEL 章节
│   └── Go 运行时集成 → protovalidate.md Go 章节节
└── 不确定 → 先读 proto-best-practices.md
```
