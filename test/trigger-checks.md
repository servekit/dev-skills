# dev-skills 触发测试

> 在**新开的 ZCode 会话**里逐条贴入下面每条 prompt，观察 agent 是否调用了对应的 Skill 工具（看回复里有没有 `Skill(golang-development)` 之类）。每个 skill 一条最小触发信号。

## 测试前确认

新会话开后，先确认 skill 真的加载了：

- 看输入框 `/` 菜单或 Settings → Skills 里有没有 `golang-development`、`proto-development` 等 9 个 skill
- 如果一个都没有 → dev-skills 插件没启用或没重启 ZCode，先解决再测

## 触发用例（每条对应一个 skill）

### golang-development（Go 基线风格）
```
帮我写个 Go 函数 readConfig(path string)，从文件读 JSON 配置返回 *Config，要求错误处理规范、有 doc comment。
```
预期：调用 Skill → **golang-development**

### golang-service-development（go-common 服务架构）
```
帮我用 go-common 脚手架新建一个 Go gRPC 服务 pay-service，先告诉我目录结构和 scaffold 步骤。
```
预期：调用 Skill → **golang-service-development**（LEAD）

### gorm-cli-development（GORM DAL）
```
我的 Go 项目用 GORM，帮我给 User model 写 store/dal 层的类型安全 CRUD（gorm gen 风格）。
```
预期：调用 Skill → **gorm-cli-development**

### proto-development（Protobuf/Buf）
```
帮我写个 user.proto，message User 有 id/name/email，带 protovalidate 字段校验规则，再给我 buf.yaml 配置。
```
预期：调用 Skill → **proto-development**

### opa-development（OPA/Rego）
```
帮我写一条 Rego 策略：只允许 role=admin 的用户访问 admin 资源，deny 其它。
```
预期：调用 Skill → **opa-development**

### rust-development
```
帮我写个 Rust 函数读文件内容到 String，要求正确处理 io::Error，用 ? 传播错误。
```
预期：调用 Skill → **rust-development**

### ts-development（含 React/Vue）
```
帮我写个 React 函数组件 UserCard，接收 props {name, email}，返回带样式的卡片。
```
预期：调用 Skill → **ts-development**（可能再路由到 react-development 子文档）

### golang-service-docker（grpcx 服务打包）
```
我有个现成的 grpcx Go 服务，帮我写多阶段 Dockerfile 打成 distroless 镜像，外加 docker-compose.yaml 带 postgres。
```
预期：调用 Skill → **golang-service-docker**

### using-dev-skills（meta，自动注入，通常不用手测）
SessionStart 时自动注入。若想验证：新会话开头问 "你装了哪些 dev-skills skill？分别什么场景用？"，agent 能列出完整清单即说明注入成功。

---

## 判定标准

- ✅ **通过**：回复里出现 `Skill(<skill-name>)` 工具调用，且内容符合该 skill 规范
- ⚠️ **漏触发**：agent 直接写代码没调 Skill —— 可能 description 触发词不够强，或 skill 没加载
- ❌ **错路由**：调了错的 skill（如 gRPC 服务却调了 golang-development 而非 golang-service-development）

## 排查

如果 skill 一个都没触发：
1. 当前会话是否在装/更新插件**之后**新开的？（旧会话不会热加载）
2. Settings → Plugin Management 里 dev-skills 是否 Enabled？
3. 重启 ZCode 客户端再开新会话
