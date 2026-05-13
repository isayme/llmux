# AGENTS.md - 项目开发指南

## 项目概述

LLMux 是一个 LLM 代理服务，主要功能：
1. 代理多个 LLM Provider 的 API 请求
2. 提供 Web 管理界面用于配置管理
3. 支持模型别名映射
4. 支持多 API Key 访问控制

## 技术架构

### 后端 (Go + Gin)
- **入口**：`server/main.go`
- **配置**：`server/config/config.yaml`，使用 Viper 加载，支持热更新
- **路由**：`server/internal/router.go`
- **核心处理器**：
  - `handler/config.go` - 配置管理接口
  - `handler/llm_proxy.go` - LLM 代理转发

### 前端 (React + TailwindCSS)
- **入口**：`web/src/main.tsx`
- **路由**：`web/src/App.tsx`，基于 `/admin/` 路径
- **页面**：
  - `pages/Login.tsx` - 登录页
  - `pages/Providers.tsx` - Provider 列表
  - `pages/ApiKeys.tsx` - API Key 列表
  - `pages/Aliases.tsx` - 别名列表
- **组件**：`components/Layout.tsx` - 公共布局

## 关键实现细节

### 认证机制
- Master Key：用于访问 Web 管理和 API 配置接口
- API Key：用于访问 LLM 代理接口
- Header 格式：`Authorization: Bearer <key>`

### 模型别名解析
- 格式：`alias_name` → `provider_id/model_name`
- 请求时自动解析别名并转发到对应 Provider

### Provider 类型
- `openai`：转发到 `/v1/chat/completions`
- `anthropic`：转发到 `/anthropic/v1/messages`

## 开发注意事项

### 后端修改
- 配置变更会自动热更新
- Provider ID 使用 map 的 key，如果为空则自动设置为 key 值
- Alias Name 使用 map 的 key，如果为空则自动设置为 key 值

### 前端修改
- TailwindCSS v4 使用 `@import "tailwindcss"` 引入
- 主题通过 CSS 变量实现，存于 `index.css`
- 主题切换：设置 `data-theme` 属性（`light` 或 `dark`）
- 构建后需复制到 server 的 dist 目录或通过静态文件服务

### 常见问题
- API Key 页面无数据：检查 `/api/api-keys` 返回格式是否为 `{ api_keys: [] }`
- 主题切换无效：确认使用了 `useState` 管理主题状态
- Provider 名称为空：显示 Provider ID

## 常用命令

```bash
# 启动后端
cd server && go run main.go

# 启动前端开发
cd web && pnpm dev

# 构建前端
cd web && pnpm build

# 运行测试（如有）
cd server && go test ./...
```