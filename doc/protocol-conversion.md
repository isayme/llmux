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



## 详细文档

各协议对的双向转换细节请参阅独立文档：

- **[OpenAI Chat ↔ Anthropic Messages](protocol-conversion-openai-anthropic.md)** — 请求/响应字段映射、SSE 流式转换、透传字段、SSE 基础设施
- **[OpenAI Chat ↔ OpenAI Responses](protocol-conversion-openai-responses.md)** — 请求/响应字段映射、SSE 流式转换
- **[Anthropic Messages ↔ OpenAI Responses](protocol-conversion-anthropic-responses.md)** — 请求/响应字段映射、SSE 流式转换
