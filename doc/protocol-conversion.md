# OpenAI ↔ Anthropic 协议转换文档

LLMux 支持双向协议转换：OpenAI 客户端可以访问 Anthropic Provider，Anthropic 客户端也可以访问 OpenAI Provider。

转换只处理有差异的字段，其余字段透传。

代码所在包：`server/internal/handler/convert/`

## 转换方向

```
OpenAI 客户端 ──── OpenAI 协议 ──── LLMux ──── Anthropic 协议 ──── Anthropic Provider
                    openaiToAnthropicConverter

Anthropic 客户端 ── Anthropic 协议 ── LLMux ──── OpenAI 协议 ──── OpenAI Provider
                    anthropicToOpenAIConverter
```

---

## 一、请求转换

### 1.1 OpenAI 请求 → Anthropic 请求

#### 字段对照表

| OpenAI 字段 | Anthropic 字段 | 转换说明 |
|---|---|---|
| `messages[].role: "system"` | `system` (top-level) | 从 messages 中提取 system 角色，移到顶层 system 字段。多个 system 消息用 `\n\n` 拼接 |
| `stop` (string \| string[]) | `stop_sequences` (string[]) | 重命名。单字符串包裹为数组，保持透传 |
| `max_tokens` (可选) | `max_tokens` (必填) | 不存在时默认填充 `4096`。Anthropic 要求此字段，OpenAI 不要求 |
| 其他所有字段 | 同名透传 | `model`, `temperature`, `top_p`, `stream` 等原样传递 |

#### 样例

**OpenAI 请求**：
```json
{
  "model": "gpt-4",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello"}
  ],
  "temperature": 0.7,
  "stream": true,
  "stop": "\n\nHuman:"
}
```

**转换后的 Anthropic 请求**：
```json
{
  "model": "gpt-4",
  "messages": [
    {"role": "user", "content": "Hello"}
  ],
  "system": "You are a helpful assistant.",
  "temperature": 0.7,
  "stream": true,
  "stop_sequences": ["\n\nHuman:"],
  "max_tokens": 4096
}
```

#### 转换细节

**System 消息提取** (`extractSystemFromMessages`)：
- OpenAI 将 system prompt 放在 `messages` 数组中，`role` 为 `"system"`
- Anthropic 将 system prompt 放在顶层 `"system"` 字段，可以是 string 或 content block 数组
- 多个 system 消息用 `\n\n` 拼接为一个字符串
- 同时支持 `content` 为纯字符串和 content block 数组两种格式

**Stop 字段重命名** (`convertStopToStopSequences`)：
- OpenAI 允许 `stop: "foo"` 或 `stop: ["foo", "bar"]`
- Anthropic 只接受 `stop_sequences: ["foo", "bar"]`（数组）
- 单字符串自动包裹为单元素数组

**max_tokens 补全** (`ensureMaxTokens`)：
- `max_tokens` 在 Anthropic API 中是必填字段
- OpenAI API 中为可选，模型有默认最大 token 数
- 转换时如果请求中未提供，默认设为 `4096`

---

### 1.2 Anthropic 请求 → OpenAI 请求

#### 字段对照表

| Anthropic 字段 | OpenAI 字段 | 转换说明 |
|---|---|---|
| `system` (top-level, string \| block[]) | `messages[0].role: "system"` | 移到 messages 数组头部，作为 system 角色消息 |
| `stop_sequences` (string[]) | `stop` (string[]) | 重命名。不包裹单元素 |
| 其他所有字段 | 同名透传 | `model`, `temperature`, `top_p`, `max_tokens`, `stream` 等原样传递 |

#### 样例

**Anthropic 请求**：
```json
{
  "model": "claude-sonnet-4-20250514",
  "system": "You are a helpful assistant.",
  "messages": [
    {"role": "user", "content": "Hello"}
  ],
  "max_tokens": 1024,
  "temperature": 0.7,
  "stream": true,
  "stop_sequences": ["\n\nHuman:"]
}
```

**转换后的 OpenAI 请求**：
```json
{
  "model": "claude-sonnet-4-20250514",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello"}
  ],
  "max_tokens": 1024,
  "temperature": 0.7,
  "stream": true,
  "stop": ["\n\nHuman:"]
}
```

#### 转换细节

**System 注入到 Messages** (`injectSystemIntoMessages`)：
- Anthropic 的 `system` 可以是 string 或 text content block 数组 `[{type: "text", text: "..."}]`
- 转换为 OpenAI 格式的单条 `{"role": "system", "content": "..."}`，插入 messages 数组头部
- 多个 content block 的 text 用 `\n\n` 拼接

**Stop_sequences 重命名** (`convertStopSequencesToStop`)：
- 直接重命名 `stop_sequences` → `stop`，值为 string 数组

---

## 二、响应转换（非流式）

### 2.1 Anthropic 响应 → OpenAI 响应

#### 字段对照表

| Anthropic 字段 | OpenAI 字段 | 转换说明 |
|---|---|---|
| `type: "message"` | (移除) | → `object: "chat.completion"` |
| `role: "assistant"` | (移除) | → `choices[0].message.role: "assistant"` |
| `content[0].text` | `choices[0].message.content` | content block 数组的第一个 text 块提取为字符串 |
| `stop_reason` | `choices[0].finish_reason` | 枚举值映射 |
| `usage.input_tokens` | `usage.prompt_tokens` | 重命名 |
| `usage.output_tokens` | `usage.completion_tokens` | 重命名 |
| — | `usage.total_tokens` | 计算：`input + output` |
| `id`, `model` 等 | 同名透传 | |

#### stop_reason → finish_reason 映射

| Anthropic | OpenAI |
|---|---|
| `end_turn` | `stop` |
| `max_tokens` | `length` |
| `stop_sequence` | `stop` |
| 其他 | `stop` |

#### 样例

**Anthropic 响应**：
```json
{
  "id": "msg_01ABC123",
  "type": "message",
  "role": "assistant",
  "model": "claude-sonnet-4-20250514",
  "content": [
    {"type": "text", "text": "Hello! How can I help you today?"}
  ],
  "stop_reason": "end_turn",
  "stop_sequence": null,
  "usage": {
    "input_tokens": 25,
    "output_tokens": 40
  }
}
```

**转换后的 OpenAI 响应**：
```json
{
  "id": "msg_01ABC123",
  "object": "chat.completion",
  "model": "claude-sonnet-4-20250514",
  "stop_sequence": null,
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! How can I help you today?"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 25,
    "completion_tokens": 40,
    "total_tokens": 65
  }
}
```

---

### 2.2 OpenAI 响应 → Anthropic 响应

#### 字段对照表

| OpenAI 字段 | Anthropic 字段 | 转换说明 |
|---|---|---|
| `object: "chat.completion"` | (移除) | → `type: "message"` |
| (无) | `role: "assistant"` | 硬编码 |
| `choices[0].message.content` | `content[0].text` | 字符串包裹为 `[{type: "text", text: "..."}]` |
| `choices[0].finish_reason` | `stop_reason` | 枚举值映射 |
| `usage.prompt_tokens` | `usage.input_tokens` | 重命名 |
| `usage.completion_tokens` | `usage.output_tokens` | 重命名 |
| `id`, `model` 等 | 同名透传 | |

#### finish_reason → stop_reason 映射

| OpenAI | Anthropic |
|---|---|
| `stop` | `end_turn` |
| `length` | `max_tokens` |
| 其他 | `end_turn` |

#### 样例

**OpenAI 响应**：
```json
{
  "id": "chatcmpl-ABC123",
  "object": "chat.completion",
  "model": "gpt-4",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! How can I help you today?"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 15,
    "completion_tokens": 30,
    "total_tokens": 45
  }
}
```

**转换后的 Anthropic 响应**：
```json
{
  "id": "chatcmpl-ABC123",
  "type": "message",
  "role": "assistant",
  "model": "gpt-4",
  "content": [
    {"type": "text", "text": "Hello! How can I help you today?"}
  ],
  "stop_reason": "end_turn",
  "usage": {
    "input_tokens": 15,
    "output_tokens": 30
  }
}
```

---

## 三、SSE 流式转换

### 3.1 Anthropic SSE → OpenAI SSE

Anthropic 流式响应包含多种事件类型，需要解析并重组成 OpenAI 的 SSE 格式。

#### 事件转换流程

```
Anthropic SSE                          OpenAI SSE
────────────────────────────────────   ──────────────────────────────
event: message_start                   data: {"choices":[{"delta":{"role":"assistant"}}]}
data: {message: {id, model, ...}}

event: content_block_delta              data: {"choices":[{"delta":{"content":"Hello"}}]}
data: {delta: {type: "text_delta",
       text: "Hello"}}

event: content_block_delta              data: {"choices":[{"delta":{"content":" world"}}]}
data: {delta: {type: "text_delta",
       text: " world"}}

event: message_delta                    data: {"choices":[{"delta":{},"finish_reason":"stop"}]}
data: {delta: {stop_reason: "end_turn"},
       usage: {output_tokens: 8}}

event: message_stop                     data: [DONE]
data: {type: "message_stop"}

event: content_block_start             (忽略, 无对应 OpenAI 事件)
event: content_block_stop              (忽略, 无对应 OpenAI 事件)
event: ping                            (忽略, Anthropic 心跳)
```

#### 关键规则

- `message_start` 提取 `message.id`、`message.model`，发送初始 chunk，delta 中仅含 `role: "assistant"`（无内容）
- `content_block_delta` 仅处理 `delta.type == "text_delta"` 的文本增量，忽略 thinking delta 等其他类型
- `message_delta` 提取 `delta.stop_reason`，映射为 `finish_reason`，发送收尾 chunk
- `message_stop` → 发送 `data: [DONE]`（注意：`[DONE]` 不是合法 JSON）
- `content_block_start`、`content_block_stop`、`ping` 事件忽略
- model 从 `message_start` 事件提取后缓存，后续 chunk 复用

#### 样例：Anthropic SSE 输入

```
event: message_start
data: {"type":"message_start","message":{"id":"msg_01","type":"message","role":"assistant","model":"claude-sonnet-4-20250514","usage":{"input_tokens":25}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" world"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":8}}

event: message_stop
data: {"type":"message_stop"}
```

#### 样例：转换后的 OpenAI SSE 输出

```
data: {"id":"msg_01","object":"chat.completion.chunk","created":1717800000,"model":"claude-sonnet-4-20250514","choices":[{"index":0,"delta":{"role":"assistant"}}]}

data: {"object":"chat.completion.chunk","created":1717800000,"model":"claude-sonnet-4-20250514","choices":[{"index":0,"delta":{"content":"Hello"}}]}

data: {"object":"chat.completion.chunk","created":1717800000,"model":"claude-sonnet-4-20250514","choices":[{"index":0,"delta":{"content":" world"}}]}

data: {"object":"chat.completion.chunk","created":1717800000,"model":"claude-sonnet-4-20250514","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]
```

---

### 3.2 OpenAI SSE → Anthropic SSE

OpenAI 的 SSE 结构更简单，每条都是 `data: {...}`（无 event 类型），需要推断 Anthropic 的事件序列。

#### 事件转换流程

```
OpenAI SSE                              Anthropic SSE
─────────────────────────────────────   ────────────────────────────────────
data: {"choices":[{"delta":             event: message_start
  {"role":"assistant"}}]}               data: {message: {id, model, ...}}
                                        event: content_block_start
                                        data: {index:0, content_block:{type:"text",text:""}}

data: {"choices":[{"delta":             event: content_block_delta
  {"content":"Hello"}}]}                data: {delta: {type:"text_delta", text:"Hello"}}

data: {"choices":[{"delta":             event: content_block_delta
  {"content":" world"}}]}               data: {delta: {type:"text_delta", text:" world"}}

data: {"choices":[{"finish_reason":     event: content_block_stop
  "stop"}]}                             data: {index:0}
                                        event: message_delta
                                        data: {delta: {stop_reason:"end_turn"}, usage: {output_tokens:0}}
                                        event: message_stop
                                        data: {type:"message_stop"}

data: [DONE]                            event: message_stop (如果还未发送)
                                        data: {type:"message_stop"}
```

#### 关键规则

- 第一条非空 chunk 触发 `message_start` + `content_block_start`，message ID 从第一个 chunk 的 `id` 字段提取（不可用时用 `"msg_unknown"`）
- 每条有 `delta.content` 的 chunk 发送 `content_block_delta`
- 遇到非空 `finish_reason` 时连续发送 `content_block_stop` → `message_delta` → `message_stop`
- `finish_reason` 映射：`stop` → `end_turn`，`length` → `max_tokens`
- `data: [DONE]` 如果 message_stop 还没发送，补发一次
- `message_id` 和 `model` 从第一个 chunk 提取后缓存

#### 样例：OpenAI SSE 输入

```
data: {"id":"chatcmpl-ABC123","object":"chat.completion.chunk","created":1717800000,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant"}}]}

data: {"id":"chatcmpl-ABC123","object":"chat.completion.chunk","created":1717800000,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"Hello"}}]}

data: {"id":"chatcmpl-ABC123","object":"chat.completion.chunk","created":1717800000,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" world"}}]}

data: {"id":"chatcmpl-ABC123","object":"chat.completion.chunk","created":1717800000,"model":"gpt-4","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]
```

#### 样例：转换后的 Anthropic SSE 输出

```
event: message_start
data: {"type":"message_start","message":{"id":"chatcmpl-ABC123","type":"message","role":"assistant","model":"gpt-4","usage":{"input_tokens":0}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" world"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":0}}

event: message_stop
data: {"type":"message_stop"}
```

---

## 四、透传字段（不转换）

以下字段不参与转换，在两个协议间原样传递：

**请求透传**：`model`, `temperature`, `top_p`, `top_k`, `stream`, `presence_penalty`, `frequency_penalty`, `n`, `seed`, `user`, `tools`, `tool_choice`, `response_format`, `logit_bias`, `metadata` 等

**响应透传**：`id`, `model`, `created`, `stop_sequence` (Anthropic) 等

> 注意：转换使用 `copyMap` 跳过需要处理的字段，其余全部原样从原始 map 复制到目标 map。未来协议新增字段无需修改转换代码。

---

## 五、SSE 解析基础设施

`convert/sse.go` 提供协议无关的 SSE 解析和写入：

**ParseSSE(r io.Reader)** 返回两个 channel：
- `<-chan SSEEvent`：解析出的 SSE 事件，包含 `Event` (事件类型) 和 `Data` (JSON 负载)
- `<-chan error`：解析错误

**SSEEvent** 结构：
```go
type SSEEvent struct {
    Event string // event 类型（OpenAI SSE 无此字段，为空字符串）
    Data  string // JSON 负载字符串
}
```

**WriteSSE(w io.Writer, event SSEEvent)**：写入一条 SSE 事件到 writer
- 如果有 `Event` 字段，写入 `event: <type>\n`
- 写入 `data: <json>\n\n`（`[DONE]` 为特例，不编码为 JSON）

**writeSSEJSON(w io.Writer, eventType string, v interface{})**：内部便捷函数，将 v 序列化为 JSON 后调用 WriteSSE。
