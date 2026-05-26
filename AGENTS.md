# AGENTS.md - 项目开发指南

## 项目概述

LLMux 是一个 LLM 代理服务，主要功能：
1. 代理多个 LLM Provider 的 API 请求
2. 提供 Web 管理界面用于配置管理
3. 支持模型别名映射
4. 支持多 API Key 访问控制

## 技术架构

### 后端 (Go + Gin)
- **入口**：`server/main.go`
- **配置**：`server/config/config.yaml`，使用 Viper 加载，支持热更新
- **路由**：`server/internal/router.go`
- **核心处理器**：
  - `handler/config.go` - 配置管理接口
  - `handler/llm_proxy.go` - LLM 代理转发

## 关键实现细节

### 认证机制
- Master Key：用于访问 Web 管理和 API 配置接口
- API Key：用于访问 LLM 代理接口
- Header 格式：`Authorization: Bearer <key>`

### 模型别名解析
- 格式：`alias_name` → `provider_id/model_name`
- 请求时自动解析别名并转发到对应 Provider

### Provider 类型
- `openai`：转发到 `/v1/chat/completions`
- `anthropic`：转发到 `/anthropic/v1/messages`

### 协议转换 (convert 包)
`server/internal/handler/convert/` 负责三个协议间的互转：**OpenAI Chat**、**OpenAI Responses**、**Anthropic**。

#### 转换器接口
- `ProtocolConverter` 定义三个方法：`ConvertRequest`、`ConvertResponse`、`ConvertSSE`
- 每个转换方向一个文件，命名规则：`{源协议}_to_{目标协议}.go`
- 工厂函数 `NewConverter` 在 `convert.go` 中，根据源/目标协议类型返回对应转换器

#### 类型定义约定
- 类型名 = 协议前缀(`OpenAI`/`Anthropic`) + SDK 接口名（如 `OpenAIChatCompletionRequest`）
- 每个字段必须有 `//` 注释说明用途，已废弃字段标注 `// Deprecated:`
- JSON tag 保持与 wire format 一致（snake_case）
- 动态类型字段（如 `messages[].content`）使用 `interface{}`
- 旧类型名通过 type alias 向后兼容（如 `OpenAIChatRequest` = `OpenAIChatCompletionRequest`）

#### 类型定义来源
- OpenAI Chat + Responses 类型：基于 `npm:openai` SDK 的 TypeScript 类型定义（`ChatCompletion*`、`Response*` 接口）
- Anthropic 类型：基于 `npm:@anthropic-ai/sdk` 的 TypeScript 类型定义（`Message*` 接口）
- 字段名保持与 SDK 一致（如 `MaxCompletionTokens` 而非 `MaxTokens`），Go 中使用协议前缀避免命名冲突
- 从 SDK 获取最新类型的方法：访问 unpkg/jsdelivr 查看对应 npm 包的类型声明文件

#### 转换函数中未映射字段的注释要求
- 每个转换函数（`ConvertRequest`/`ConvertResponse`/`ConvertSSE`）中，所有**有等价物但未转换**的字段必须通过注释说明
- 注释格式：在函数 return 前统一添加注释块，分为两段：
  - "有等价物但未实现"——列出字段名及对应的目标字段
  - "无等价物"——列出字段名
- 无等价物的字段也需注释说明，避免后续开发者反复确认

#### 当前转换覆盖情况
- 基础字段（model/messages/temperature/top_p/stream/stop）已覆盖
- **以下字段已有等价物但尚未实现转换**（标注在转换函数的注释中）：
  - Tools/ToolChoice：三个协议都支持，结构不同
  - Metadata：三个协议都支持，类型不同
  - ReasoningEffort ↔ Thinking：Chat ↔ Anthropic
  - ResponseFormat ↔ OutputConfig：Chat ↔ Anthropic
  - ResponseFormat ↔ Text：Chat ↔ Responses
  - Reasonning ↔ Reasoning：Responses ↔ Chat
  - 以及 Temperature/Stream/Store/ServiceTier/PresencePenalty/FrequencyPenalty/TopLogprobs/PromptCache 等公共参数
- 响应转换仅处理首个 text content block，tool_use/thinking 等复杂 block 未转换
- **MCP / Function Calling**：目前三个协议都定义了完整的 Tool 类型结构，但转换函数中尚未实现 Tools/ToolChoice 字段的跨协议映射。MCP Server 的 tool 类型定义在各协议的 Types struct 中已有对应字段，等待转换逻辑实现

#### 测试
```bash
cd server && go test ./internal/handler/convert/... -count=1
```
- 覆盖 ConvertRequest/ConvertResponse/ConvertSSE 三个方向
- 测试数据直接构造 struct 或使用 JSON body

## 开发注意事项

### 后端修改
- 配置变更会自动热更新
- Provider ID 使用 map 的 key，如果为空则自动设置为 key 值
- Alias Name 使用 map 的 key，如果为空则自动设置为 key 值

### 常见问题
- API Key 页面无数据：检查 `/api/api-keys` 返回格式是否为 `{ api_keys: [] }`
- 主题切换无效：确认使用了 `useState` 管理主题状态
- Provider 名称为空：显示 Provider ID

## 常用命令

```bash
# 启动后端
cd server && go run main.go

# 启动前端开发
cd web && pnpm dev

# 构建前端
cd web && pnpm build

# 运行测试（如有）
cd server && go test ./...
```