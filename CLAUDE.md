# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Backend (Go 1.26, Gin framework)
cd server && go run main.go              # Run server (port 8080, or $PORT)
cd server && go build -o ./dist/llmux main.go  # Build binary
cd server && go test ./...               # Run tests

# Frontend (React 19, Vite, pnpm 10)
cd web && pnpm dev                       # Dev server (proxies /api, /version to :8080)
cd web && pnpm build                     # TypeScript check + Vite build
cd web && pnpm lint                      # ESLint

# Combined
make dev                                 # Build frontend, copy to server/dist/, run server
make buildweb                            # Build frontend and copy into server/dist/
```

## Architecture

**LLMux is a config-driven LLM proxy** — there is no database at runtime. All state (providers, API keys, aliases) lives in `server/config/config.yaml`, loaded via Viper with hot-reload. The GORM models in `server/internal/model/` are scaffolding for a future DB-backed mode and are not wired in.

### Request flow for LLM proxy

```
client request → API key auth middleware → model extraction from body
→ alias resolution (resolveAlias) → strategy-based provider/model selection
→ protocol conversion if needed (OpenAI ↔ Anthropic)
→ forwardRequest (replaces Authorization header, adds anthropic-version if needed)
→ if stream: SSE proxy; if not: read body + optional response conversion
→ write response (headers stripped of Content-Length/Content-Encoding when converted)
```

### Model alias resolution

Aliases are defined in `config.yaml` under `aliases`. Each alias has:
- `strategy` — `"random"`, `"round_robin"`, `"fallback"`, or empty (defaults to round_robin)
- `models` — list of `{provider, model}` targets

The `/v1/models` endpoint lists all enabled alias names as available models. Clients can also use raw `provider_id/model_name` format without an alias.

The `server/internal/strategy/` package implements the `ModelSelector` interface:
- `Next() (provider, model string, retryable bool)` — returns the next target
- `NewSelector(strategy, models)` — factory dispatching to the right implementation

For `"fallback"` strategy, `retryable` is `true` and the proxy handler loops through models on failure (any HTTP error or network error triggers trying the next model).

### Bidirectional protocol conversion

Any provider can be accessed via either API format. The `server/internal/handler/convert/` package handles:

- **Request conversion** — field-level helpers in `convert/convert.go`:
  - OpenAI `messages[role=system]` ↔ Anthropic top-level `system` field
  - OpenAI `stop` ↔ Anthropic `stop_sequences`
  - Anthropic requires `max_tokens` (defaults to 4096 when converting from OpenAI)
- **Response conversion** — restructures response bodies between formats
- **SSE streaming conversion** — `convert/stream_convert.go` using `io.Pipe()` and `ParseSSE`/`WriteSSE` from `convert/sse.go`

Conversion happens automatically when the client's protocol differs from the provider's type. The `needsConversion` flag in `handleProxy` gates both body and header adjustments.

### Three error handler formats

Each route group uses a different error middleware matching the target API:

| Middleware | Routes | JSON format |
|---|---|---|
| `InternalErrorHandler` | `/api/*` (admin) | `{"code": "...", "message": "..."}` |
| `OpenaiErrorHandler` | `/v1/*` | `{"error": {"message": "...", "type": "...", "code": "..."}}` |
| `AnthropicErrorHandler` | `/anthropic/*` | `{"type": "error", "error": {"message": "...", "type": "..."}}` |

### Authentication (two-tier)

- **Master key** (bcrypt hash in config) → session cookie → protects `/api/*` admin routes
- **API keys** → `Authorization: Bearer <key>` header → protects `/v1/*` and `/anthropic/*` proxy routes

### Config hot-reload

Viper `WatchConfig()` + `OnConfigChange` re-unmarshals the YAML and atomically swaps the `globalConfig` pointer. Always read config via `config.Get()` rather than caching references. Auto-population: provider `ID` defaults to its map key, alias `Name` defaults to its map key.

### Frontend

React 19 SPA at `/admin/` with React Router 7, TailwindCSS 4, Lucide icons. In dev, Vite proxies `/api` and `/version` to `localhost:8080`. In production, the Go server serves `server/dist/` as static files with SPA fallback (`NoRoute` serves `index.html`). Auth state is session-based (cookie), checked on mount via `GET /api/check`. Theme (light/dark/system) uses CSS custom properties on `<html>`.
