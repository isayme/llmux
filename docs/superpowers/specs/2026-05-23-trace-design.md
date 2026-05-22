# LLMux Trace 设计

## 概述

为 LLMux 增加 OpenTelemetry 标准的分布式链路追踪，覆盖每个代理请求的完整生命周期（解析 → 别名解析 → Provider 选择 → 协议转换 → 上游调用 → 响应转换），支持 W3C Trace Context 传播到上游 Provider，导出到 stdout 或 OTLP 兼容后端。

## 架构

```
Client → Gin → TraceMiddleware (root span + W3C propagation)
  → APIKeyMiddleware (span attrs: identity)
  → handleProxy
    ├── "parse request"         (model extraction)
    ├── "resolve alias"         (alias → selector)
    ├── "select provider"       (strategy.Next)
    ├── "convert request"       (protocol conversion)
    ├── "upstream call"         (HTTP forward + traceparent injection)
    ├── "convert response"      (protocol conversion)
    └── "sse stream"            (streaming duration)
  → Response (traceparent header)
```

## 依赖

- `go.opentelemetry.io/otel` — 核心 API
- `go.opentelemetry.io/otel/sdk/trace` — SDK TracerProvider
- `go.opentelemetry.io/otel/exporters/stdout/stdouttrace` — stdout 导出器
- `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp` — OTLP HTTP 导出器
- `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` — 上游 HTTP 调用的自动 instrumentation

## 配置

```yaml
trace:
  enabled: true                # 关闭时零开销
  exporter: stdout             # stdout | otlp
  endpoint: ""                 # OTLP collector endpoint (http://host:4318/v1/traces)
  sampling:
    ratio: 0.1                 # 概率采样，1.0 = 100%
    force_on_error: true       # HTTP >=400 或内部错误强制采样
    force_on_latency: 5s       # 超过此延迟强制采样
```

## 文件结构

```
server/internal/trace/
  trace.go          # TracerProvider 初始化、导出器选择、shutdown
  middleware.go      # Gin middleware: root span 创建 + W3C 传播
  span.go           # Span 辅助: 子 span 创建、属性设置、错误记录
```

### trace.go

- `Init(cfg TraceConfig) (*sdktrace.TracerProvider, error)` — 根据配置创建 TracerProvider
  - `enabled=false` → 返回 NoopTracerProvider
  - `exporter=stdout` → stdouttrace exporter
  - `exporter=otlp` → otlptracehttp exporter
- `Shutdown(ctx)` — graceful shutdown，flush 所有 pending span
- 全局 `Tracer` 变量，包内复用

### middleware.go

- `Middleware() gin.HandlerFunc`
  1. 从请求 header 提取 W3C `traceparent`、`tracestate`
  2. 传播或创建新 trace context
  3. 创建 root span，span name = `"POST /v1/chat/completions"`（method + route pattern）
  4. 采样决策：先由 Sampler 决定，然后检查 `force_on_error` / `force_on_latency` 条件
  5. 将 span context 存入 `c.Request.Context()`
  6. 写入 `traceresponse` header 到 HTTP 响应
  7. `c.Next()` 后设置 status code attribute，结束 root span

### span.go

- `StartSpan(ctx, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span)` — 创建子 span
- `SetError(span, err)` — 记录错误到 span
- `SetAttributes(span, attrs ...)` — 批量设置属性
- `AddEvent(span, name, attrs ...)` — 添加事件

## Span 清单

| Span Name | 位置 | 关键属性 |
|---|---|---|
| `POST /v1/chat/completions` (root) | middleware | method, path, status_code, trace_id, api_key_id |
| `parse request` | handleProxy | model (提取的原始模型名) |
| `resolve alias` | resolveModelSelector | alias, selector_type (alias/direct) |
| `select provider` | handleProxy loop | provider_id, model_name, retryable |
| `convert request` | handleProxy | converter (转换器类名), from_protocol, to_protocol |
| `upstream call` | forwardRequest | http.method, http.url, http.status_code, provider |
| `convert response` | handleProxy | converter, status_code |
| `sse stream` | proxySSE | duration_ms, bytes_transferred |

## 上游 Trace Context 传播

在 `forwardRequest` 中，通过 `otelhttp` 的 transport 自动将 `traceparent` header 注入到上游 HTTP 请求。

对于 SSE 流式请求，`otelhttp` 同样适用（`http.Client.Do` 在创建请求时注入 header）。

## W3C Trace Context

- 接受 header：`traceparent`（必须）、`tracestate`（可选）
- 响应 header：`traceresponse`（标准尚未统一，使用 `traceparent` 回传）
- 格式：`traceparent: 00-{trace_id}-{span_id}-{trace_flags}`

## 与 slog 集成

- 每个请求的 root span 的 `trace_id` 写入 Gin context
- slog 日志可通过 `slog.String("trace_id", ...)` 关联到 trace
- 后续按需在 `handleProxy` 的关键分支记录 slog 日志（含 trace_id）

## 采样逻辑

```go
func shouldSample(cfg SamplingConfig, statusCode int, latency time.Duration) bool {
    if cfg.ForceOnError && statusCode >= 400 {
        return true
    }
    if cfg.ForceOnLatency > 0 && latency > cfg.ForceOnLatency {
        return true
    }
    if rand.Float64() < cfg.Ratio {
        return true
    }
    return false
}
```

采样在 root span 创建时决策（statusCode 和 latency 在请求结束后判断，若命中强制条件则更新 span 的 Sampled flag）。

## 零开销关闭

```go
if !cfg.Enabled {
    global.SetTracerProvider(noop.NewTracerProvider())
    return noop.NewTracerProvider(), nil
}
```

关闭时：不创建 span、不注入 middleware（Gin 路由不添加），不额外分配内存。

## 验证

1. `trace.enabled=false` → 启动服务，检查无 span 输出，go test 确认无额外延迟
2. `exporter=stdout` → 发起请求，检查 stdout 输出完整的 span 树
3. `exporter=otlp` → 启动 Jaeger（docker），发起请求，在 Jaeger UI 查看 trace
4. W3C 传播 → 客户端带 `traceparent` header，检查 LLMux 的 root span 的 parent 是客户端 trace
5. 上游传播 → 检查上游 Provider 收到 `traceparent` header
6. 采样 → `ratio=0` 时无 span 输出；`force_on_error=true` 时 4xx/5xx 必定输出
