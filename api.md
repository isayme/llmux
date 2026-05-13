# API 文档

## 认证

所有管理接口需要通过 `Authorization` Header 传递 Master Key：

```
Authorization: Bearer <master_key>
```

LLM 代理接口需要通过 `Authorization` Header 传递 API Key：

```
Authorization: Bearer <api_key>
```

---

## 管理接口

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
      "provider": "sensenova",
      "model": "deepseek-v4-flash",
      "enabled": true
    }
  }
}
```

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
```

**请求体：**

```json
{
  "model": "provider_id/model_name",
  "messages": [
    {"role": "user", "content": "Hello"}
  ],
  "stream": false
}
```

**说明：**
- `model` 格式为 `provider_id/model_name`
- Provider 类型必须为 `anthropic`
- 支持 stream 模式

---

## 错误响应

错误时返回标准 HTTP 状态码和错误信息：

```json
{
  "error": {
    "message": "error message",
    "type": "invalid_request_error",
    "code": 400
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