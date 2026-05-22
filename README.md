# LLMux

一个简单的 LLM 代理服务，支持配置多个 Provider 和 API Key，并提供 Web 管理界面。

## 功能特性

- **多 Provider 支持**：支持配置多个 LLM 服务提供商（OpenAI / Anthropic / OpenAI Responses 协议）
- **协议转换**：自动双向转换 OpenAI Chat、OpenAI Responses、Anthropic Messages 三种协议之间的请求和响应（含 SSE 流式）
- **API Key 管理**：支持多个 API Key，用于访问控制
- **模型别名**：支持为模型设置别名，支持 random、round-robin、fallback 调度策略
- **统一接口**：提供 OpenAI 兼容和 Anthropic 兼容的 API
- **配置热更新**：修改配置文件无需重启服务
- **Web 管理界面**：直观的 Web 页面管理配置
- **主题切换**：支持亮色/暗色模式

## 快速开始

### 配置

在 `server/config/config.yaml` 中配置：

```yaml
server:
  master_key: "your-master-key"  # Web 管理界面访问密钥 (bcrypt hash)
  port: 8080

api_keys:
  - key: "your-api-key-1"
    enabled: true

providers:
  provider1:
    type: openai          # openai / anthropic / openai_responses
    name: "Provider 1"
    base_url: "https://api.openai.com/v1"
    api_key: "sk-xxx"
    enabled: true

aliases:
  my-model:
    enabled: true
    strategy: round_robin  # 策略: random / round_robin / fallback，默认 round_robin
    models:
      - provider: provider1
        model: gpt-4
      - provider: provider2
        model: deepseek-v4-flash
```

### 启动服务

```bash
cd server
go run main.go
```

服务启动后：
- Web 管理界面：`http://localhost:8080/admin/`
- API 接口：`http://localhost:8080/v1/`

### Web 界面

1. 访问 `/admin/` 输入 Master Key 登录
2. 在 Providers 页面管理 LLM 提供商
3. 在 API Keys 页面管理访问密钥
4. 在 Aliases 页面管理模型别名和调度策略

## API 接口

### OpenAI 兼容接口

```bash
# Chat Completions（支持访问任意类型的 Provider）
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"model": "my-model", "messages": [{"role": "user", "content": "Hello"}]}'

# List Models
curl http://localhost:8080/v1/models \
  -H "Authorization: Bearer your-api-key"
```

### OpenAI Responses 兼容接口

```bash
# Responses（支持访问任意类型的 Provider）
curl http://localhost:8080/v1/responses \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"model": "my-model", "input": "Hello"}'
```

### Anthropic 兼容接口

```bash
# Messages（支持访问任意类型的 Provider）
curl http://localhost:8080/anthropic/v1/messages \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"model": "my-model", "messages": [{"role": "user", "content": "Hello"}], "max_tokens": 1024}'
```

### 管理接口

```bash
# 获取 Providers
curl http://localhost:8080/api/providers \
  -H "Authorization: Bearer your-master-key"

# 获取 API Keys
curl http://localhost:8080/api/api-keys \
  -H "Authorization: Bearer your-master-key"

# 获取 Aliases
curl -X POST http://localhost:8080/api/aliases \
  -H "Authorization: Bearer your-master-key"
```

## 别名策略

别名支持四种调度策略：

| 策略 | 说明 |
|------|------|
| `round_robin` | 轮询，每次请求依次使用下一个模型（默认） |
| `random` | 随机选择一个模型 |
| `fallback` | 故障转移，按列表顺序尝试，失败后尝试下一个 |

`model` 参数也支持直接使用 `provider_id/model_name` 格式而不配置别名。

## 项目结构

```
llmux/
├── server/                 # Go 后端服务
│   ├── main.go            # 入口文件
│   ├── config/            # 配置文件
│   │   └── config.yaml
│   └── internal/          # 内部包
│       ├── handler/       # HTTP 处理器 + 协议转换
│       ├── config/        # 配置加载
│       ├── strategy/      # 模型调度策略
│       ├── constant/      # 常量定义
│       └── util/          # 工具函数
└── web/                   # React 前端
    ├── src/
    │   ├── pages/        # 页面组件
    │   ├── components/   # 公共组件
    │   └── services/     # API 服务
    └── dist/             # 构建产物
```

## 技术栈

- **后端**：Go + Gin
- **前端**：React + React Router + TailwindCSS
- **配置**：Viper (YAML, 支持热更新)
