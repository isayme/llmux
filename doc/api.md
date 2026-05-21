# API 文档

## 认证

管理接口通过 session cookie 认证（先通过 `/api/login` 登录）或 `Authorization` Header 传递 Master Key：

```
Authorization: Bearer <master_key>
```

LLM 代理接口需要通过 `Authorization` Header 传递 API Key：

```
Authorization: Bearer <api_key>
```

---

## 管理接口

### 登录

```http
POST /api/login
```

**请求体：**

```json
{
  "master_key": "your-master-key"
}
```

### 获取 Providers 列表

```http
GET /api/providers
```

**响应：**

```json
{
  "providers": [
    {
      "id": "provider1",
      "name": "Provider 1",
      "type": "openai",
      "base_url": "https://api.openai.com",
      "enabled": true
    }
  ]
}
```

---

### 获取 API Keys 列表

```http
GET /api/api-keys
```

**响应：**

```json
{
  "api_keys": [
    {
      "name": "key1",
      "key": "913c61bcfd11ae7be4976e1680d8086b30548494619b",
      "enabled": true
    }
  ]
}
```

---

### 获取 Aliases 列表

```http
POST /api/aliases
```

**响应：**

```json
{
  "aliases": {
    "deepseek-v4-flash": {
      "name": "deepseek-v4-flash",
      "strategy": "random",
      "models": [
        {"provider": "deepseek", "model": "deepseek-v4-flash", "weight": 0},
        {"provider": "openrouter", "model": "deepseek/deepseek-v4-flash:free", "weight": 0}
      ],
      "enabled": true
    }
  }
}
```

**字段说明：**
- `strategy` — 策略：`"random"`（随机）、`"round_robin"`（轮询）、`"fallback"`（故障转移），空字符串默认使用 round_robin
- `models` — 模型列表，每项包含 `provider`（提供商标识）、`model`（模型名）、`weight`（权重，目前 fallback 下无影响）

---

### 获取版本信息

```http
GET /version
```

**响应：**

```json
{
  "version": "0.1.0"
}
```

---

## LLM 代理接口

### 协议转换说明

LLMux 支持 **双向协议转换**：可以用 OpenAI 协议访问 Anthropic 类型的 Provider，也可以用 Anthropic 协议访问 OpenAI 类型的 Provider。服务端会自动完成请求体和响应体的格式转换（包括 SSE 流式传输）。

### OpenAI Chat Completions

```http
POST /v1/chat/completions
```

**请求头：**

```
Authorization: Bearer <api_key>
Content-Type: application/json
```

**请求体：**

```json
{
  "model": "alias-name",
  "messages": [
    {"role": "user", "content": "Hello"}
  ],
  "stream": false
}
```

**说明：**
- `model` 可以是别名名称，也可以是 `provider_id/model_name` 格式
- 支持 stream 模式
- 可以访问任意类型的 Provider（OpenAI 或 Anthropic），协议自动转换

---

### OpenAI List Models

```http
GET /v1/models
```

**请求头：**

```
Authorization: Bearer <api_key>
```

**响应：**

```json
{
  "object": "list",
  "data": [
    {
      "id": "alias-name",
      "object": "model",
      "owned_by": "llmux",
      "created": 1234567890
    }
  ]
}
```

---

### Anthropic Messages

```http
POST /anthropic/v1/messages
```

**请求头：**

```
Authorization: Bearer <api_key>
Content-Type: application/json
anthropic-version: 2023-06-01
```

**请求体：**

```json
{
  "model": "alias-name",
  "messages": [
    {"role": "user", "content": "Hello"}
  ],
  "max_tokens": 1024,
  "stream": false
}
```

**说明：**
- `model` 可以是别名名称，也可以是 `provider_id/model_name` 格式
- 支持 stream 模式
- 可以访问任意类型的 Provider（OpenAI 或 Anthropic），协议自动转换
- 如果请求头未包含 `anthropic-version`，代理会自动添加 `2023-06-01`

---

## 错误响应

### OpenAI 接口错误格式

```json
{
  "error": {
    "message": "error message",
    "type": "BadRequest",
    "code": "BadRequest"
  }
}
```

### Anthropic 接口错误格式

```json
{
  "type": "error",
  "error": {
    "message": "error message",
    "type": "BadRequest"
  }
}
```

### 常见状态码

| 状态码 | 说明 |
|--------|------|
| 400 | 请求参数错误 |
| 401 | 认证失败 |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |
