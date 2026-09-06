package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"llmux/internal/config"
	llmuxlog "llmux/internal/log"
	"llmux/internal/strategy"
	"llmux/internal/trace"
	"llmux/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
)

var restHttpClient = &http.Client{
	Transport: otelhttp.NewTransport(http.DefaultTransport),
	Timeout:   60 * time.Second,
}

var sseHttpClient = &http.Client{
	Transport: otelhttp.NewTransport(http.DefaultTransport),
	Timeout:   0,
}

var anthropicVersion = "2023-06-01"

// ProxyHandler handles LLM proxy requests. It caches model selectors per alias
// so that stateful strategies (round-robin) work correctly across requests.
// Cached selectors expire after 10 minutes of inactivity to prevent unbounded
// memory growth when aliases or API keys are removed.
type ProxyHandler struct {
	selectorMu    sync.Mutex
	selectorCache *cache.Cache
	logService    *llmuxlog.Service
}

func NewProxyHandler(logService *llmuxlog.Service) *ProxyHandler {
	return &ProxyHandler{
		selectorCache: cache.New(10*time.Minute, 5*time.Minute),
		logService:    logService,
	}
}

func (h *ProxyHandler) ChatCompletionsHandler(c *gin.Context) {
	h.handleProxy(c)
}

func (h *ProxyHandler) AnthropicMessagesHandler(c *gin.Context) {
	h.handleProxy(c)
}

func (h *ProxyHandler) ResponsesHandler(c *gin.Context) {
	h.handleProxy(c)
}

func (h *ProxyHandler) ListModelsHandler(c *gin.Context) {
	modelInfos := make([]modelInfo, 0, len(config.Get().Aliases))

	resp := listModelsResp{
		Object: "list",
	}

	for model, aliasConfig := range config.Get().Aliases {
		if !aliasConfig.Enabled {
			continue
		}
		modelInfos = append(modelInfos, modelInfo{
			ID:      model,
			Object:  "model",
			OwnedBy: "llmux",
			Created: time.Now().Unix(),
		})
	}
	resp.Data = modelInfos

	c.JSON(http.StatusOK, resp)
}

func (h *ProxyHandler) handleProxy(c *gin.Context) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Error(BadRequest.WithMessage("read req body failed", err))
		return
	}

	var rawMap map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &rawMap); err != nil {
		c.Error(BadRequest.WithMessage("json parse req body failed", err))
		return
	}

	_, parseSpan := trace.StartSpan(c.Request.Context(), "parse request")
	rawModel, err := getModel(rawMap)
	if err != nil {
		trace.SetError(parseSpan, err)
		parseSpan.End()
		c.Error(BadRequest.WithMessage("get model failed", err))
		return
	}
	trace.SetAttributes(parseSpan, attribute.String("model", rawModel))
	parseSpan.End()

	// Start logging
	requestID := uuid.New().String()
	apiKeyID := GetAPIKey(c)
	requestLog, logErr := h.logService.StartRequest(requestID, rawModel, rawModel, c.Request.Method, c.Request.URL.Path, bodyBytes, c.ClientIP(), apiKeyID)
	if logErr != nil {
		// Log error but don't fail the request
		slog.Error("Failed to create request log", "error", logErr)
	}
	var finalResponseBody []byte
	defer func() {
		if requestLog != nil {
			status := "success"
			if c.Request.Context().Err() == context.Canceled {
				status = "canceled"
			} else if c.Writer.Status() >= 400 {
				status = "failed"
			}
			if err := h.logService.CompleteRequest(requestLog, status, finalResponseBody); err != nil {
				slog.Error("Failed to complete request log", "error", err)
			}
		}
	}()

	_, resolveSpan := trace.StartSpan(c.Request.Context(), "resolve alias")
	selector, err := h.resolveModelSelector(c, rawModel)
	if err != nil {
		trace.SetError(resolveSpan, err)
		resolveSpan.End()
		c.Error(BadRequest.WithMessage("resolve alias failed", err))
		return
	}
	trace.SetAttributes(resolveSpan, attribute.String("alias", rawModel))
	resolveSpan.End()

	isStream, err := isStream(rawMap)
	attemptCount := 0
	if err != nil {
		c.Error(BadRequest.WithMessage("get stream failed", err))
		return
	}

	for {
		_, selectSpan := trace.StartSpan(c.Request.Context(), "select provider")
		providerId, model, canRetry := selector.Next()
		trace.SetAttributes(selectSpan,
			attribute.String("provider_id", providerId),
			attribute.String("model_name", model),
			attribute.Bool("retryable", canRetry),
		)
		selectSpan.End()
		if providerId == "" {
			c.Error(NotFound.WithMessage("no available model"))
			return
		}

		provider, err := findProvider(providerId)
		if err != nil {
			c.Error(NotFound.WithMessage("provider not found", err))
			return
		}

		if !provider.Enabled {
			if canRetry {
				continue
			}
			c.Error(Forbidden.WithMessage(fmt.Sprintf("provider %s disabled", providerId)))
			return
		}

		rawMap["model"] = model

		forwardBytes, err := json.Marshal(rawMap)
		if err != nil {
			c.Error(InternalServerError.WithMessage("marshal forward body failed", err))
			return
		}

		forwardPath := getProviderPath(provider.Type)

		// Log provider call start
		var providerCall *llmuxlog.ProviderCall
		if requestLog != nil {
			attemptCount++
			providerCall, logErr = h.logService.LogProviderCallStart(requestLog.ID, providerId, provider.Type, model, forwardBytes, attemptCount > 1)
			if logErr != nil {
				slog.Error("Failed to log provider call start", "error", logErr)
			}
		}
		callStart := time.Now()

		resp, err := forwardRequest(c.Request.Context(), getHttpClient(isStream), provider, c.Request.Method, forwardPath, c.Request.Header, forwardBytes)
		callDuration := time.Since(callStart).Milliseconds()

		// Log provider call end
		if providerCall != nil {
			if err != nil {
				if logErr := h.logService.LogProviderCallEnd(providerCall, 0, nil, nil, callDuration, err); logErr != nil {
					slog.Error("Failed to log provider call end", "error", logErr)
				}
			} else {
				// Read response body for logging (but don't consume it for streaming)
				var respBody []byte
				if !isStream && resp.Body != nil {
					var readErr error
					respBody, readErr = io.ReadAll(resp.Body)
					if readErr != nil {
						slog.Warn("Failed to read response body for logging", "error", readErr)
					}
					resp.Body.Close()
					// Re-create reader for downstream use
					resp.Body = io.NopCloser(bytes.NewReader(respBody))
				}

				// Convert response headers to JSON
				headerJSON, marshalErr := json.Marshal(resp.Header)
				if marshalErr != nil {
					slog.Warn("Failed to marshal response headers for logging", "error", marshalErr)
				}

				if logErr := h.logService.LogProviderCallEnd(providerCall, resp.StatusCode, headerJSON, respBody, callDuration, nil); logErr != nil {
					slog.Error("Failed to log provider call end", "error", logErr)
				}
			}
		}

		if err != nil {
			if canRetry {
				continue
			}
			c.Error(InternalServerError.WithMessage("forward request failed", err))
			return
		}

		if resp.StatusCode >= 400 {
			if shouldRetry(resp.StatusCode, provider, canRetry) {
				// 429: respect retry-after header
				if resp.StatusCode == 429 {
					delay := parseRetryAfter(resp.Header)
					time.Sleep(delay)
				}
				resp.Body.Close()
				continue
			}
			copyResponseHeaders(c, resp.Header)
			c.Status(resp.StatusCode)

			bs, err := io.ReadAll(resp.Body)
			if err != nil {
				c.Error(InternalServerError.WithMessage("read response body failed", err))
				return
			}
			resp.Body.Close()
			
			// Log provider call end with error response
			if providerCall != nil {
				headerJSON, marshalErr := json.Marshal(resp.Header)
				if marshalErr != nil {
					slog.Warn("Failed to marshal response headers for logging", "error", marshalErr)
				}
				if logErr := h.logService.LogProviderCallEnd(providerCall, resp.StatusCode, headerJSON, bs, callDuration, nil); logErr != nil {
					slog.Error("Failed to log provider call end", "error", logErr)
				}
			}
			finalResponseBody = bs
			
			c.Writer.Write(bs)
			return
		}

		defer resp.Body.Close()

		if isStream {
			_, sseSpan := trace.StartSpan(c.Request.Context(), "sse stream")
			copyResponseHeaders(c, resp.Header)
			c.Status(resp.StatusCode)
			respBody := proxySSE(c, resp.Body)
			sseSpan.End()
			
			// Log provider call end with stream response
			if providerCall != nil {
				headerJSON, marshalErr := json.Marshal(resp.Header)
				if marshalErr != nil {
					slog.Warn("Failed to marshal response headers for logging", "error", marshalErr)
				}
				if logErr := h.logService.LogProviderCallEnd(providerCall, resp.StatusCode, headerJSON, respBody, callDuration, nil); logErr != nil {
					slog.Error("Failed to log provider call end", "error", logErr)
				}
			}
			finalResponseBody = respBody
			
			return
		}

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			c.Error(InternalServerError.WithMessage("read response body failed", err))
			return
		}

		copyResponseHeaders(c, resp.Header)
		c.Status(resp.StatusCode)
		c.Writer.Write(respBody)
		finalResponseBody = respBody
		return
	}
}

// resolveModelSelector resolves a model name or alias to a ModelSelector.
// Selectors for aliases are cached on the ProxyHandler, keyed by API key + alias
// so different API keys get independent selector state (e.g. round-robin position).
func (h *ProxyHandler) resolveModelSelector(c *gin.Context, modelOrAlias string) (strategy.ModelSelector, error) {
	aliasConfig, found := config.Get().Aliases[modelOrAlias]
	if !found {
		parts := strings.SplitN(modelOrAlias, "/", 2)
		if len(parts) != 2 {
			return nil, errors.New("invalid model format, expected alias_name or provider_id/model_name")
		}
		models := []*config.ModelAliasItemConfig{
			{
				Provider: parts[0],
				Model:    parts[1],
				Weight:   1,
			},
		}
		return strategy.NewSelector("round_robin", models), nil
	}

	if !aliasConfig.Enabled {
		return nil, errors.New("alias disabled")
	}

	if len(aliasConfig.Models) == 0 {
		return nil, errors.New("alias has no models configured")
	}

	cacheKey := GetAPIKey(c) + ":" + modelOrAlias

	h.selectorMu.Lock()
	defer h.selectorMu.Unlock()

	if s, found := h.selectorCache.Get(cacheKey); found {
		return s.(strategy.ModelSelector), nil
	}

	sel := strategy.NewSelector(aliasConfig.Strategy, aliasConfig.Models)
	h.selectorCache.Set(cacheKey, sel, cache.DefaultExpiration)
	return sel, nil
}

// copyResponseHeaders copies upstream response headers to the client.
// Content-Length and Content-Encoding are stripped because the response body
// may have been decompressed or converted, invalidating the original values.
func copyResponseHeaders(c *gin.Context, upstream http.Header) {
	for key, values := range upstream {
		switch key {
		case "Content-Length", "Content-Encoding":
			continue
		}
		for _, value := range values {
			c.Header(key, value)
		}
	}
}

type modelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	OwnedBy string `json:"owned_by"`
	Created int64  `json:"created"`
}

type listModelsResp struct {
	Object string      `json:"object"`
	Data   []modelInfo `json:"data"`
}

func findProvider(providerId string) (*config.ProviderConfig, error) {
	for id, provider := range config.Get().Providers {
		if id == providerId {
			return provider, nil
		}
	}

	return nil, errors.New("provider not found")
}

func getProviderPath(providerType string) string {
	switch providerType {
	case "anthropic":
		return "/v1/messages"
	case "openai_responses":
		return "/v1/responses"
	default:
		return "/chat/completions"
	}
}

func forwardRequest(ctx context.Context, httpClient *http.Client, provider *config.ProviderConfig, method, path string, header http.Header, body []byte) (*http.Response, error) {
	ctx, upstreamSpan := trace.StartSpan(ctx, "upstream call",
		attribute.String("provider", provider.ID),
		attribute.String("provider_type", provider.Type),
	)
	defer upstreamSpan.End()

	url := strings.TrimSuffix(provider.BaseURL, "/") + path

	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		trace.SetError(upstreamSpan, err)
		return nil, err
	}

	for key, values := range header {
		if key == "Content-Length" || key == "Accept-Encoding" {
			continue
		}
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	req.Header.Set("Authorization", bearerPrefix+provider.APIKey)

	if provider.Type == "anthropic" {
		if req.Header.Get("anthropic-version") == "" {
			req.Header.Set("anthropic-version", anthropicVersion)
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		trace.SetError(upstreamSpan, err)
		return nil, err
	}

	upstreamSpan.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
	return resp, nil
}

// shouldRetry determines if the request should be retried based on status code and provider config.
// Returns true if the request should be retried.
func shouldRetry(statusCode int, provider *config.ProviderConfig, canRetry bool) bool {
	if !canRetry {
		return false
	}

	// 400/401/403 never retry
	if statusCode == 400 || statusCode == 401 || statusCode == 403 {
		return false
	}

	retryConfig := provider.Retry
	if retryConfig == nil {
		return false
	}

	switch {
	case statusCode == 429:
		return retryConfig.On429
	case statusCode >= 400 && statusCode < 500:
		return retryConfig.On4xx
	case statusCode >= 500:
		return retryConfig.On5xx
	default:
		return false
	}
}

// parseRetryAfter parses the Retry-After header and returns the delay duration.
// If the header is missing or invalid, returns 1 second.
func parseRetryAfter(header http.Header) time.Duration {
	retryAfter := header.Get("Retry-After")
	if retryAfter == "" {
		return 1 * time.Second
	}

	// Try to parse as seconds
	if seconds, err := strconv.Atoi(retryAfter); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try to parse as HTTP date
	if t, err := http.ParseTime(retryAfter); err == nil {
		delay := time.Until(t)
		if delay > 0 {
			return delay
		}
	}

	return 1 * time.Second
}

func getModel(reqBody map[string]interface{}) (string, error) {
	return util.GetString(reqBody, "model")
}

func isStream(reqBody map[string]interface{}) (bool, error) {
	return util.GetBool(reqBody, "stream")
}

func getHttpClient(isStream bool) *http.Client {
	if isStream {
		return sseHttpClient
	}
	return restHttpClient
}

func proxySSE(c *gin.Context, reader io.Reader) []byte {
	ctx := c.Request.Context()

	flusher, _ := c.Writer.(http.Flusher)
	buf := make([]byte, 4096)
	var captured []byte
	for {
		select {
		case <-ctx.Done():
			return captured
		default:
		}

		n, err := reader.Read(buf)
		if n > 0 {
			captured = append(captured, buf[:n]...)
			c.Writer.Write(buf[:n])
			flusher.Flush()
			c.Writer.Flush()
		}
		if err != nil {
			break
		}
	}
	return captured
}
