# Anthropic Messages ↔ OpenAI Responses 协议转换文档

LLMux 支持 Anthropic Messages 与 OpenAI Responses 两种协议之间的双向转换。

转换只处理有差异的字段，其余字段透传。

代码所在包：`server/internal/handler/convert/`

涉及转换器：`responsesToAnthropicConverter`、`anthropicToResponsesConverter`

---


## 一、OpenAI Responses ↔ Anthropic 转换

### 1.1 Responses 请求 → Anthropic 请求

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

### 1.2 Anthropic 请求 → Responses 请求

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

### 1.3 Responses 响应 → Anthropic 响应

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

### 1.4 Anthropic 响应 → Responses 响应

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

### 1.5 SSE 流式转换：Responses → Anthropic

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

### 1.6 SSE 流式转换：Anthropic → Responses

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
