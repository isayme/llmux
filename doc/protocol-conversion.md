# 协议转换文档

LLMux 支持三种协议之间的全量双向转换：OpenAI Chat、OpenAI Responses、Anthropic Messages。

转换只处理有差异的字段，其余字段透传。

代码所在包：`server/internal/handler/convert/`

## 转换方向

共 6 个转换器，覆盖 3 种协议之间的所有交叉访问（同协议直通，不转换）：

```
OpenAI Chat ──────── Chat 协议 ──────── LLMux ──── Anthropic 协议 ──── Anthropic Provider
                     openaiToAnthropicConverter

Anthropic ────────── Anthropic 协议 ─── LLMux ──── Chat 协议 ───────── OpenAI Chat Provider
                     anthropicToOpenAIConverter

OpenAI Responses ─── Responses 协议 ─── LLMux ──── Chat 协议 ───────── OpenAI Chat Provider
                     responsesToOpenAIConverter

OpenAI Chat ──────── Chat 协议 ──────── LLMux ──── Responses 协议 ───── Responses Provider
                     openaiToResponsesConverter

OpenAI Responses ─── Responses 协议 ─── LLMux ──── Anthropic 协议 ───── Anthropic Provider
                     responsesToAnthropicConverter

Anthropic ────────── Anthropic 协议 ─── LLMux ──── Responses 协议 ───── Responses Provider
                     anthropicToResponsesConverter
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

以下字段不参与转换，在两种协议间原样传递。每个转换器通过 `copyMap` 跳过需要处理的字段（如 `messages`、`input`、`max_tokens` 等），其余全部原样从原始 map 复制到目标 map。

**请求透传**：`model`, `temperature`, `top_p`, `top_k`, `stream`, `presence_penalty`, `frequency_penalty`, `n`, `seed`, `user`, `tools`, `tool_choice`, `response_format`, `logit_bias`, `metadata` 等

**Responses 专属请求字段（透传，不做转换）**：`previous_response_id`, `truncation`, `store`, `parallel_tool_calls`, `reasoning`, `text`, `include`, `top_logprobs` 等 — 当 Responses 客户端访问 Responses Provider 时直通，当访问其他协议 Provider 时这些字段会透传到上游（上游可能忽略不认识的字段）

**响应透传**：`id`, `model`, `created`, `stop_sequence` (Anthropic), `status` (Responses) 等

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

---

## 六、OpenAI Responses ↔ OpenAI Chat 转换

### 6.1 Responses 请求 → Chat 请求

#### 字段对照表

| Responses 字段 | Chat 字段 | 转换说明 |
|---|---|---|
| `input` (string \| input item[]) | `messages` | 转换为 messages 数组。数组元素过滤 `type: "message"` 的项，提取 `role` 和 `content`。string 类型时包装为单条 `user` 消息。非 message 类型的 input item 被丢弃 |
| `instructions` (string) | `messages[0].role: "system"` | 作为 system 角色消息插入 messages 数组头部。空字符串不插入 |
| `max_output_tokens` | `max_tokens` | 重命名。不存在时不填充默认值（无 max_tokens 要求） |
| 其他所有字段 | 同名透传 | `model`, `temperature`, `top_p`, `stream` 等原样传递 |

#### 样例

**Responses 请求**：
```json
{
  "model": "gpt-4",
  "instructions": "You are a helpful assistant.",
  "input": [{"type": "message", "role": "user", "content": "Hello"}],
  "max_output_tokens": 1024,
  "temperature": 0.7
}
```

**转换后的 Chat 请求**：
```json
{
  "model": "gpt-4",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello"}
  ],
  "max_tokens": 1024,
  "temperature": 0.7
}
```

#### 转换细节

**Input 转换**：
- Responses 的 `input` 支持两种格式：纯字符串（视为单条 user 消息）和 input item 数组
- 数组格式时，遍历所有 item，只取 `type: "message"` 的项，提取其 `role` 和 `content` 组成 Chat 的 messages
- 非 message 类型的 item（如 `function_call`、`function_call_output`）在转换中被丢弃
- 纯字符串 `"Hello"` 等价于 `[{"type": "message", "role": "user", "content": "Hello"}]`

**Instructions 注入**：
- `instructions` 为字符串且非空时，作为 `{"role": "system", "content": "..."}` 插入 messages 数组头部
- 空字符串不产生 system 消息

**max_output_tokens 重命名**：
- Responses 使用 `max_output_tokens`，Chat 使用 `max_tokens`
- 直接重命名，不存在时不填充默认值（与 OpenAI → Anthropic 不同，后者需要补全 4096）

---

### 6.2 Chat 请求 → Responses 请求

#### 字段对照表

| Chat 字段 | Responses 字段 | 转换说明 |
|---|---|---|
| `messages[]` | `input[]` | 提取非 system 消息，每项包装为 `{type: "message", role, content}`。system 角色消息不进入 input |
| `messages[].role: "system"` | `instructions` | 从 messages 中提取所有 system 角色的 content，多个用 `\n\n` 拼接为一个字符串 |
| `max_tokens` | `max_output_tokens` | 重命名 |
| 其他所有字段 | 同名透传 | `model`, `temperature`, `top_p`, `stream` 等原样传递 |

#### 样例

**Chat 请求**：
```json
{
  "model": "gpt-4",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello"}
  ],
  "max_tokens": 1024
}
```

**转换后的 Responses 请求**：
```json
{
  "model": "gpt-4",
  "instructions": "You are a helpful assistant.",
  "input": [{"type": "message", "role": "user", "content": "Hello"}],
  "max_output_tokens": 1024
}
```

#### 转换细节

**Messages 拆分**：
- 遍历 Chat 的 `messages` 数组，按 role 分流：
  - `role: "system"` → 提取 content，收集到 `instructions` 列表中
  - 其他 role（user、assistant、tool 等）→ 包装为 `{type: "message", role, content}` 放入 `input` 数组
- 多个 system 消息的 content 用 `"\n\n"` 拼接为一个 instructions 字符串
- 没有 system 消息时，`instructions` 字段不出现在输出中

**max_tokens 重命名**：
- 直接重命名为 `max_output_tokens`

---

### 6.3 Responses 响应 → Chat 响应

#### 字段对照表

| Responses 字段 | Chat 字段 | 转换说明 |
|---|---|---|
| `object: "response"` | (移除) | → `object: "chat.completion"` |
| (无) | `choices[0].index` | 硬编码 `0` |
| (无) | `choices[0].message.role` | 硬编码 `"assistant"` |
| `output[0].content[0].text` | `choices[0].message.content` | 通过 `extractResponsesContent` 提取文本 |
| (无) | `choices[0].finish_reason` | 硬编码 `"stop"`（Responses API 无 stop_reason 字段） |
| `created_at` | `created` | 重命名 |
| `usage.input_tokens` | `usage.prompt_tokens` | 重命名 |
| `usage.output_tokens` | `usage.completion_tokens` | 重命名 |
| — | `usage.total_tokens` | 计算：`input_tokens + output_tokens` |
| `id`, `model` 等 | 同名透传 | |

#### extractResponsesContent 提取路径

```
resp["output"] → []interface{}
  → output[0] → map[string]interface{} (type: "message")
    → ["content"] → []interface{}
      → content[0] → map[string]interface{} (type: "output_text")
        → ["text"] → string
```

各级取不到时返回空字符串 `""`。

#### 样例

**Responses 响应**：
```json
{
  "id": "resp_abc123",
  "object": "response",
  "model": "gpt-4o",
  "created_at": 1734567890,
  "output": [
    {
      "type": "message",
      "id": "msg_1",
      "role": "assistant",
      "content": [
        {"type": "output_text", "text": "Hello! How can I help you today?"}
      ]
    }
  ],
  "usage": {
    "input_tokens": 25,
    "output_tokens": 40,
    "total_tokens": 65
  }
}
```

**转换后的 Chat 响应**：
```json
{
  "id": "resp_abc123",
  "object": "chat.completion",
  "model": "gpt-4o",
  "created": 1734567890,
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

### 6.4 Chat 响应 → Responses 响应

#### 字段对照表

| Chat 字段 | Responses 字段 | 转换说明 |
|---|---|---|
| `object: "chat.completion"` | (移除) | → `object: "response"` |
| (无) | `output[0].type` | 硬编码 `"message"` |
| `id` | `output[0].id` | 透传 |
| `choices[0].message.role` | `output[0].role` | 透传 |
| `choices[0].message.content` | `output[0].content[0].text` | 字符串包装为 `[{type: "output_text", text: "..."}]` |
| `created` | `created_at` | 重命名 |
| `usage.prompt_tokens` | `usage.input_tokens` | 重命名 |
| `usage.completion_tokens` | `usage.output_tokens` | 重命名 |
| `usage.total_tokens` | `usage.total_tokens` | 同名透传 |

#### 样例

**Chat 响应**：
```json
{
  "id": "chatcmpl-ABC123",
  "object": "chat.completion",
  "model": "gpt-4",
  "created": 1734567890,
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

**转换后的 Responses 响应**：
```json
{
  "id": "chatcmpl-ABC123",
  "object": "response",
  "model": "gpt-4",
  "created_at": 1734567890,
  "output": [
    {
      "type": "message",
      "id": "chatcmpl-ABC123",
      "role": "assistant",
      "content": [
        {"type": "output_text", "text": "Hello! How can I help you today?"}
      ]
    }
  ],
  "usage": {
    "input_tokens": 15,
    "output_tokens": 30,
    "total_tokens": 45
  }
}
```

---

### 6.5 SSE 流式转换：Responses → Chat

Responses 流式响应包含 `response.*` 类型事件，需要解析并重组成 OpenAI Chat 的 SSE 格式。

#### 事件转换流程

```
Responses SSE                              OpenAI Chat SSE
─────────────────────────────────────────  ────────────────────────────────────
event: response.created                    data: {"choices":[{"delta":{"role":"assistant"}}]}
data: {response: {id, model, created_at}}

event: response.text.delta                 data: {"choices":[{"delta":{"content":"Hello"}}]}
data: {delta: "Hello"}

event: response.text.delta                 data: {"choices":[{"delta":{"content":" world"}}]}
data: {delta: " world"}

event: response.done                       data: {"choices":[{"delta":{},"finish_reason":"stop"}]}
data: {response: {usage: {...}}}           data: [DONE]

event: error                               data: {"error":{"message":"..."}}
data: "error message"
```

#### 关键规则

- `response.created` 提取 `response.id`、`response.model`、`response.created_at`，发送首个 chunk，delta 仅含 `role: "assistant"`（无内容）
- `response.text.delta` 提取 `delta` 字段（纯字符串），发送 content chunk
- `response.done` 提取 usage 信息，发送 `finish_reason: "stop"` 的收尾 chunk，然后发送 `data: [DONE]`
- `error` 事件发送 `{"error": {"message": "..."}}` chunk
- 未识别的事件类型被忽略

#### 样例：Responses SSE 输入

```
event: response.created
data: {"type":"response.created","response":{"id":"resp_1","model":"gpt-4o","created_at":1734567890}}

event: response.text.delta
data: {"type":"response.text.delta","delta":"Hello"}

event: response.text.delta
data: {"type":"response.text.delta","delta":" world"}

event: response.done
data: {"type":"response.done","response":{"id":"resp_1","model":"gpt-4o","usage":{"input_tokens":10,"output_tokens":5}}}
```

#### 样例：转换后的 Chat SSE 输出

```
data: {"id":"resp_1","object":"chat.completion.chunk","created":1734567890,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant"}}]}

data: {"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"Hello"}}]}

data: {"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":" world"}}]}

data: {"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}

data: [DONE]
```

---

### 6.6 SSE 流式转换：Chat → Responses

OpenAI Chat 的 `data:` 流需要推断并重构为 Responses 的完整事件序列。

#### 事件转换流程

```
OpenAI Chat SSE                            Responses SSE
─────────────────────────────────────────  ────────────────────────────────────
data: {"choices":[{"delta":                event: response.created
  {"role":"assistant"}}]}                  data: {response: {id, model}}
                                           event: response.output_item.added
                                           data: {item: {type:"message",role:"assistant"}}
                                           event: response.content_part.added
                                           data: {part: {type:"text"}}

data: {"choices":[{"delta":                event: response.text.delta
  {"content":"Hello"}}]}                   data: {delta: "Hello"}

data: {"choices":[{"delta":                event: response.text.delta
  {"content":" world"}}]}                  data: {delta: " world"}

data: {"choices":[{"delta":{},             event: response.text.done
  "finish_reason":"stop"}]}                data: {text: ""}
                                           event: response.output_item.done
                                           event: response.done
                                           data: {response: {id, model}}
```

#### 关键规则

- 第一条有效 chunk（含 `choices[0].delta`）触发连续三个事件：`response.created` → `response.output_item.added` → `response.content_part.added`
- `response_id` 和 `model` 从第一个 chunk 提取后缓存
- 每条有非空 `delta.content` 的 chunk 发送 `response.text.delta`，同时标记 `hadContent = true`
- 遇到非空 `finish_reason` 时：如果曾发送过内容则先发 `response.text.done`，然后发 `response.output_item.done`，最后发 `response.done`
- `data: [DONE]`：如果曾发送过内容，补发 `response.text.done` + `response.output_item.done`，最后发 `response.done`（含 response id 和 model）
- 空 data、choices 为空、choices[0] 不是 map 的 chunk 被静默跳过
- `finish_reason` 不参与映射，Responses API 的结束语义由事件序列表达

#### 样例：Chat SSE 输入

```
data: {"id":"chatcmpl-123","model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant"}}]}

data: {"choices":[{"delta":{"content":"Hello"}}]}

data: {"choices":[{"delta":{"content":" world"}}]}

data: {"choices":[{"delta":{},"finish_reason":"stop"}]}

data: [DONE]
```

#### 样例：转换后的 Responses SSE 输出

```
event: response.created
data: {"type":"response.created","response":{"id":"chatcmpl-123","model":"gpt-4o"}}

event: response.output_item.added
data: {"type":"response.output_item.added","item":{"type":"message","role":"assistant"}}

event: response.content_part.added
data: {"type":"response.content_part.added","part":{"type":"text"}}

event: response.text.delta
data: {"type":"response.text.delta","delta":"Hello"}

event: response.text.delta
data: {"type":"response.text.delta","delta":" world"}

event: response.text.done
data: {"type":"response.text.done","text":""}

event: response.output_item.done
data: {"type":"response.output_item.done"}

event: response.done
data: {"type":"response.done","response":{"id":"chatcmpl-123","model":"gpt-4o"}}
```

---

## 七、OpenAI Responses ↔ Anthropic 转换

### 7.1 Responses 请求 → Anthropic 请求

#### 字段对照表

| Responses 字段 | Anthropic 字段 | 转换说明 |
|---|---|---|
| `input` (string \| input item[]) | `messages` | 转换为 messages 数组。数组元素过滤 `type: "message"` 的项，提取 `role` 和 `content`。string 类型时包装为单条 `user` 消息 |
| `instructions` (string) | `system` (top-level, string) | 移到顶层 system 字段。空字符串不设置 |
| `max_output_tokens` | `max_tokens` | 重命名 |
| 其他所有字段 | 同名透传 | `model`, `temperature`, `top_p`, `stream` 等原样传递 |

#### 样例

**Responses 请求**：
```json
{
  "model": "claude-sonnet-4-20250514",
  "instructions": "You are a helpful assistant.",
  "input": [{"type": "message", "role": "user", "content": "Hello"}],
  "max_output_tokens": 1024,
  "temperature": 0.7
}
```

**转换后的 Anthropic 请求**：
```json
{
  "model": "claude-sonnet-4-20250514",
  "system": "You are a helpful assistant.",
  "messages": [{"role": "user", "content": "Hello"}],
  "max_tokens": 1024,
  "temperature": 0.7
}
```

#### 转换细节

**Input 转换**：
- 与 6.1 Responses → Chat 的 input 转换逻辑相同
- string 类型 input 包装为 `[{"role": "user", "content": "..."}]`
- 数组类型时，遍历 item，只取 `type: "message"` 的项，提取 role 和 content 组成 messages

**Instructions → System**：
- `instructions` 为字符串且非空时，直接设为顶层 `system` 字段
- 空字符串不产生 `system` 字段

**max_output_tokens 重命名**：
- 直接重命名为 `max_tokens`。Anthropic 要求此字段必填，但此处不做补全（信任上游 Responses API 返回的默认行为或后续层处理）

---

### 7.2 Anthropic 请求 → Responses 请求

#### 字段对照表

| Anthropic 字段 | Responses 字段 | 转换说明 |
|---|---|---|
| `messages[]` | `input[]` | 每项包装为 `{type: "message", role, content}` |
| `system` (string \| text block[]) | `instructions` | 移到 instructions 字段。string 直接赋值；content block 数组时提取 `text` 字段，多个用 `\n\n` 拼接 |
| `max_tokens` | `max_output_tokens` | 重命名 |
| 其他所有字段 | 同名透传 | `model`, `temperature`, `top_p`, `stream` 等原样传递 |

#### 样例

**Anthropic 请求**：
```json
{
  "model": "claude-sonnet-4-20250514",
  "system": "You are a helpful assistant.",
  "messages": [{"role": "user", "content": "Hello"}],
  "max_tokens": 1024
}
```

**转换后的 Responses 请求**：
```json
{
  "model": "claude-sonnet-4-20250514",
  "instructions": "You are a helpful assistant.",
  "input": [{"type": "message", "role": "user", "content": "Hello"}],
  "max_output_tokens": 1024
}
```

#### 转换细节

**Messages → Input**：
- 遍历 Anthropic 的 `messages` 数组，每项包装为 `{type: "message", role, content}` 放入 `input`
- 不做 role 过滤，所有消息（user、assistant）都进入 input

**System → Instructions**：
- Anthropic 的 `system` 支持两种格式：纯字符串和 text content block 数组 `[{type: "text", text: "..."}]`
- 字符串：直接赋值给 `instructions`
- Content block 数组：遍历提取每个 block 的 `text` 字段，非空文本用 `"\n\n"` 拼接
- 空字符串不产生 `instructions` 字段

**max_tokens 重命名**：
- 直接重命名为 `max_output_tokens`

---

### 7.3 Responses 响应 → Anthropic 响应

#### 字段对照表

| Responses 字段 | Anthropic 字段 | 转换说明 |
|---|---|---|
| — | `type: "message"` | 硬编码 |
| — | `role: "assistant"` | 硬编码 |
| `object` | (移除) | Responses 的 `object: "response"` 被丢弃 |
| `output[0].content[0].text` | `content[0].text` | 通过 `extractResponsesContent` 提取文本，包装为 content block |
| `usage.input_tokens` | `usage.input_tokens` | 同名透传 |
| `usage.output_tokens` | `usage.output_tokens` | 同名透传 |
| `id`, `model` 等 | 同名透传 | |

#### 样例

**Responses 响应**：
```json
{
  "id": "resp_abc123",
  "object": "response",
  "model": "gpt-4o",
  "output": [
    {
      "type": "message",
      "role": "assistant",
      "content": [
        {"type": "output_text", "text": "Hello! How can I help you today?"}
      ]
    }
  ],
  "usage": {
    "input_tokens": 25,
    "output_tokens": 40
  }
}
```

**转换后的 Anthropic 响应**：
```json
{
  "id": "resp_abc123",
  "type": "message",
  "role": "assistant",
  "model": "gpt-4o",
  "content": [
    {"type": "text", "text": "Hello! How can I help you today?"}
  ],
  "usage": {
    "input_tokens": 25,
    "output_tokens": 40
  }
}
```

---

### 7.4 Anthropic 响应 → Responses 响应

#### 字段对照表

| Anthropic 字段 | Responses 字段 | 转换说明 |
|---|---|---|
| — | `object: "response"` | 硬编码 |
| `type` | (移除) | Anthropic 的 `type: "message"` 被丢弃 |
| `role` | `output[0].role` | 透传 |
| `content[0].text` | `output[0].content[0].text` | 文本包装为 output 结构：`[{type: "output_text", text: "..."}]` |
| `id` | `output[0].id` | 透传 |
| `usage.input_tokens` | `usage.input_tokens` | 同名透传 |
| `usage.output_tokens` | `usage.output_tokens` | 同名透传 |
| — | `usage.total_tokens` | 计算：`input_tokens + output_tokens` |

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
  "usage": {
    "input_tokens": 25,
    "output_tokens": 40
  }
}
```

**转换后的 Responses 响应**：
```json
{
  "id": "msg_01ABC123",
  "object": "response",
  "model": "claude-sonnet-4-20250514",
  "output": [
    {
      "type": "message",
      "id": "msg_01ABC123",
      "role": "assistant",
      "content": [
        {"type": "output_text", "text": "Hello! How can I help you today?"}
      ]
    }
  ],
  "usage": {
    "input_tokens": 25,
    "output_tokens": 40,
    "total_tokens": 65
  }
}
```

---

### 7.5 SSE 流式转换：Responses → Anthropic

Responses 流式事件需要解析并重组成 Anthropic 的事件序列。

#### 事件转换流程

```
Responses SSE                              Anthropic SSE
─────────────────────────────────────────  ────────────────────────────────────
event: response.created                    event: message_start
data: {response: {id}}                     data: {message: {id, type:"message",role:"assistant",content:[]}}
                                           event: content_block_start
                                           data: {index:0, content_block:{type:"text",text:""}}

event: response.text.delta                 event: content_block_delta
data: {delta: "Hello"}                     data: {index:0, delta:{type:"text_delta",text:"Hello"}}

event: response.done                       event: content_block_stop
data: {}                                   data: {index:0}
                                           event: message_delta
                                           data: {delta:{stop_reason:"end_turn"}}
                                           event: message_stop
                                           data: {type:"message_stop"}
```

#### 关键规则

- `response.created` 提取 `response.id`，发送 `message_start` + `content_block_start`，标记 `started = true`
- `response.text.delta` 仅在 `started` 后才处理，提取 `delta` 字符串字段，发送 `content_block_delta`（delta type 固定为 `"text_delta"`）
- `response.done` 仅在 `started` 后才处理，连续发送 `content_block_stop` → `message_delta`（stop_reason 固定为 `"end_turn"`）→ `message_stop`
- 未识别的 event 类型被忽略
- **注意**：如果 `response.text.delta` 或 `response.done` 出现在 `response.created` 之前，会被静默丢弃（`started` 为 false）

#### 样例：Responses SSE 输入

```
event: response.created
data: {"type":"response.created","response":{"id":"resp_1"}}

event: response.text.delta
data: {"type":"response.text.delta","delta":"Hello"}

event: response.text.delta
data: {"type":"response.text.delta","delta":" world"}

event: response.done
data: {"type":"response.done"}
```

#### 样例：转换后的 Anthropic SSE 输出

```
event: message_start
data: {"type":"message_start","message":{"id":"resp_1","type":"message","role":"assistant","content":[]}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" world"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"}}

event: message_stop
data: {"type":"message_stop"}
```

---

### 7.6 SSE 流式转换：Anthropic → Responses

Anthropic 的 SSE 事件需要推断并重构为 Responses 的事件序列。

#### 事件转换流程

```
Anthropic SSE                              Responses SSE
─────────────────────────────────────────  ────────────────────────────────────
event: message_start                       event: response.created
data: {message: {id}}                      data: {response: {id}}
                                           event: response.output_item.added
                                           data: {item: {type:"message",role:"assistant"}}
                                           event: response.content_part.added
                                           data: {part: {type:"text"}}

event: content_block_delta                 event: response.text.delta
data: {delta:{type:"text_delta",           data: {delta: "Hello"}
       text:"Hello"}}

event: message_delta                       (忽略, 无对应 Responses 事件)
event: content_block_stop                  (忽略, 无对应 Responses 事件)

event: message_stop                        event: response.text.done
data: {type:"message_stop"}                data: {text: ""}
                                           event: response.output_item.done
                                           event: response.done
                                           data: {response: {id}}
```

#### 关键规则

- `message_start` 提取 `message.id`，发送 `response.created` + `response.output_item.added` + `response.content_part.added`，标记 `started = true`
- `content_block_delta` 仅在 `started` 后才处理，且仅处理 `delta.type == "text_delta"` 的增量，提取 `delta.text` 发送 `response.text.delta`
- `message_stop` 仅在 `started` 后才处理，发送 `response.text.done` → `response.output_item.done` → `response.done`
- `message_delta`、`content_block_start`、`content_block_stop` 事件被忽略（无对应 Responses 事件）
- 空 data、无法解析为 JSON 的 data 被静默跳过

#### 样例：Anthropic SSE 输入

```
event: message_start
data: {"type":"message_start","message":{"id":"msg_1"}}

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

#### 样例：转换后的 Responses SSE 输出

```
event: response.created
data: {"type":"response.created","response":{"id":"msg_1"}}

event: response.output_item.added
data: {"type":"response.output_item.added","item":{"type":"message","role":"assistant"}}

event: response.content_part.added
data: {"type":"response.content_part.added","part":{"type":"text"}}

event: response.text.delta
data: {"type":"response.text.delta","delta":"Hello"}

event: response.text.delta
data: {"type":"response.text.delta","delta":" world"}

event: response.text.done
data: {"type":"response.text.done","text":""}

event: response.output_item.done
data: {"type":"response.output_item.done"}

event: response.done
data: {"type":"response.done","response":{"id":"msg_1"}}
```
