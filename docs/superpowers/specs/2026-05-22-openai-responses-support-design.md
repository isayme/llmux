# OpenAI Responses API 支持设计

## 概述

为 LLMux 添加 OpenAI `/v1/responses` API 协议支持，实现 openai/chat-completions、openai/responses、anthropic/messages 三个协议之间的全量双向转换。

## 新的 Provider 类型

```go
const ProviderTypeOpenAIResponses = "openai_responses"
```

配置示例：
```yaml
providers:
  my-responses:
    type: openai_responses
    base_url: "https://api.openai.com/v1"
    api_key: "sk-..."
    enabled: true
```

## 新的客户端路由

`POST /v1/responses` — 与 `/v1/chat/completions` 同级，Bearer token 认证。

## 架构变更

### 新增常量
- `constant/provide.go`: `ProviderTypeOpenAIResponses`
- `handler/convert/convert.go`: `ProtocolOpenAIResponses`

### 新增路由 (`router.go`)
- `POST /v1/responses` → `proxyHandler.ResponsesHandler`
- 使用 `ResponsesErrorHandler` 中间件

### 新增 Handler (`llm_proxy.go`)
- `ResponsesHandler` → 调用 `handleProxy(c, ProtocolOpenAIResponses)`
- `getProviderPath()` 新增 `openai_responses` → `/v1/responses`

### 新增错误格式 (`errorhandler.go`)
- `ResponsesErrorHandler` 输出 `{"type": "error", "error": {"code":..., "message":...}}`

### GetConverter 更新
6 路转换 + 3 路透传（noop），完整覆盖所有 9 种组合。

### 新 Converter 文件（4 个）

1. **`responses_to_openai.go`** — Responses 请求/响应 → Chat 格式
   - 请求：`input`→`messages`, `instructions`→system message, `max_output_tokens`→`max_tokens`
   - 响应：`output[0].content[0].text`→`choices[0].message.content`, usage 字段映射
   - SSE：Responses 事件流 → Chat `data:` 流

2. **`openai_to_responses.go`** — Chat → Responses（逆映射）
   - 请求：`messages`→`input`, system→`instructions`, `max_tokens`→`max_output_tokens`
   - 响应：`choices[0].message.content`→`output[0].content[0].text`
   - SSE：Chat `data:` → Responses 事件流

3. **`responses_to_anthropic.go`** — Responses → Anthropic
   - 请求：`input`→`messages`, `instructions`→`system`, `max_output_tokens`→`max_tokens`
   - 响应：`output[0].content[0].text`→`content[0].text`, 字段映射
   - SSE：Responses 事件 → Anthropic 事件

4. **`anthropic_to_responses.go`** — Anthropic → Responses（逆映射）
   - 请求：`messages`→`input`, `system`→`instructions`, `max_tokens`→`max_output_tokens`
   - 响应：`content[0].text`→`output[0].content[0].text`
   - SSE：Anthropic 事件 → Responses 事件

### 字段映射原则
- 使用现有的 `copyMap()` 模式，排除/重命名特定字段
- 独有字段（`previous_response_id`, `store` 等）在转换中丢弃
- 未知字段自动透传

## 测试计划

- 每个转换器：请求/响应/SSE 三种方法全覆盖
- 边缘用例：空输入、未知字段透传、usage 缺失
- 路由验证：`/v1/responses` 正常分发
