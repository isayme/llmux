# OpenAI Chat ↔ OpenAI Responses 协议转换文档

LLMux 支持 OpenAI Chat 与 OpenAI Responses 两种协议之间的双向转换。

转换只处理有差异的字段，其余字段透传。

代码所在包：`server/internal/handler/convert/`

涉及转换器：`responsesToOpenAIConverter`、`openaiToResponsesConverter`

---


## 一、OpenAI Responses ↔ OpenAI Chat 转换

### 1.1 Responses 请求 → Chat 请求

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

### 1.2 Chat 请求 → Responses 请求

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

### 1.3 Responses 响应 → Chat 响应

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

### 1.4 Chat 响应 → Responses 响应

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

### 1.5 SSE 流式转换：Responses → Chat

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

### 1.6 SSE 流式转换：Chat → Responses

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
  "finish_reason":"stop"}]}                data: {text: ""}        (if hadContent)
                                           event: response.output_item.done
                                           event: response.done
                                           data: {response: {id, model}}

data: [DONE]                               event: response.text.done      (if hadContent)
                                           event: response.output_item.done (if hadContent)
                                           event: response.done
                                           data: {response: {id, model}}
```

#### 关键规则

- 第一条有效 chunk（含 `choices[0].delta`）触发连续三个事件：`response.created` → `response.output_item.added` → `response.content_part.added`
- `response_id` 和 `model` 从第一个 chunk 提取后缓存，后续事件中的 response.done 复用缓存值
- 每条有非空 `delta.content` 的 chunk 发送 `response.text.delta`，同时标记 `hadContent = true`
- 遇到非空 `finish_reason` 时：若 `hadContent` 为 true 则先发 `response.text.done`；然后无条件发送 `response.output_item.done` 和 `response.done`（含缓存的 id 和 model）
- `data: [DONE]`：无条件发送 `response.done`（含缓存的 id 和 model）；若 `hadContent` 为 true 则额外在之前发送 `response.text.done` + `response.output_item.done`
- 空 data、choices 为空、choices[0] 不是 map 的 chunk 被静默跳过
- `finish_reason` 的值不参与映射（不转换为 stop_reason），Responses API 的结束语义由事件序列表达

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

event: response.text.done
data: {"type":"response.text.done","text":""}

event: response.output_item.done
data: {"type":"response.output_item.done"}

event: response.done
data: {"type":"response.done","response":{"id":"chatcmpl-123","model":"gpt-4o"}}
```

---
