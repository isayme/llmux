# LLMux

一个简单的 LLM 代理服务，支持配置多个 Provider 和 API Key，并提供 Web 管理界面。

## 功能特性

- **多 Provider 支持**：支持配置多个 LLM 服务提供商
- **API Key 管理**：支持多个 API Key，用于访问控制
- **模型别名**：支持为模型设置别名，方便使用
- **统一接口**：提供 OpenAI 兼容和 Anthropic 兼容的 API
- **Web 管理界面**：直观的 Web 页面管理配置
- **主题切换**：支持亮色/暗色模式

## 快速开始

### 配置

在 `server/config/config.yaml` 中配置：

```yaml
server:
  master_key: "your-master-key"  # Web 管理界面访问密钥
  port: 8080

api_keys:
  - key: "your-api-key-1"
    enabled: true
  - key: "your-api-key-2"
    enabled: true

providers:
  provider1:
    type: openai          # openai 或 anthropic
    name: "Provider 1"
    base_url: "https://api.openai.com"
    api_key: "sk-xxx"
    enabled: true

aliases:
  my-model:
    provider: provider1
    model: gpt-4
    enabled: true
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
4. 在 Aliases 页面管理模型别名

## API 接口

### OpenAI 兼容接口

```bash
# Chat Completions
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"model": "my-model", "messages": [{"role": "user", "content": "Hello"}]}'

# List Models
curl http://localhost:8080/v1/models \
  -H "Authorization: Bearer your-api-key"
```

### Anthropic 兼容接口

```bash
# Messages
curl http://localhost:8080/anthropic/v1/messages \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"model": "provider1/claude-3", "messages": [{"role": "user", "content": "Hello"}]}'
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

## 项目结构

```
llmux/
├── server/                 # Go 后端服务
│   ├── main.go            # 入口文件
│   ├── config/            # 配置文件
│   │   └── config.yaml
│   └── internal/          # 内部包
│       ├── handler/       # HTTP 处理器
│       ├── config/        # 配置加载
│       ├── model/         # 数据模型
│       └── router.go      # 路由配置
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
- **配置**：Viper (YAML)