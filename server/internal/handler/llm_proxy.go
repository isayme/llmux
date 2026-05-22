package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"llmux/internal/config"
	"llmux/internal/handler/convert"
	"llmux/internal/strategy"
	"llmux/internal/trace"
	"llmux/internal/util"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"go.opentelemetry.io/otel/attribute"
)

var restHttpClient = &http.Client{
	Timeout: 60 * time.Second,
}

var sseHttpClient = &http.Client{
	Timeout: 0,
}

var anthropicVersion = "2023-06-01"

// ProxyHandler handles LLM proxy requests. It caches model selectors per alias
// so that stateful strategies (round-robin) work correctly across requests.
// Cached selectors expire after 10 minutes of inactivity to prevent unbounded
// memory growth when aliases or API keys are removed.
type ProxyHandler struct {
	selectorMu    sync.Mutex
	selectorCache *cache.Cache
}

func NewProxyHandler() *ProxyHandler {
	return &ProxyHandler{
		selectorCache: cache.New(10*time.Minute, 5*time.Minute),
	}
}

func (h *ProxyHandler) ChatCompletionsHandler(c *gin.Context) {
	h.handleProxy(c, convert.ProtocolOpenAI)
}

func (h *ProxyHandler) AnthropicMessagesHandler(c *gin.Context) {
	h.handleProxy(c, convert.ProtocolAnthropic)
}

func (h *ProxyHandler) ResponsesHandler(c *gin.Context) {
	h.handleProxy(c, convert.ProtocolOpenAIResponses)
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

func (h *ProxyHandler) handleProxy(c *gin.Context, usedProtocol convert.UsedAIProtocol) {
	var req map[string]interface{}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Error(BadRequest.WithMessage("json parse req body failed", err))
		return
	}

	_, parseSpan := trace.StartSpan(c.Request.Context(), "parse request")
	rawModel, err := getModel(req)
	if err != nil {
		trace.SetError(parseSpan, err)
		parseSpan.End()
		c.Error(BadRequest.WithMessage("get model failed", err))
		return
	}
	trace.SetAttributes(parseSpan, attribute.String("model", rawModel))
	parseSpan.End()

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

	isStream, err := isStream(req)
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

		req["model"] = model

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

		converter := convert.GetConverter(usedProtocol, provider.Type)

		_, convertReqSpan := trace.StartSpan(c.Request.Context(), "convert request")
		trace.SetAttributes(convertReqSpan,
			attribute.String("converter", fmt.Sprintf("%T", converter)),
			attribute.String("from_protocol", usedProtocol.String()),
			attribute.String("to_provider_type", provider.Type),
		)
		forwardBody := converter.ConvertRequest(req)
		convertReqSpan.End()

		forwardPath := getProviderPath(provider.Type)

		resp, err := forwardRequest(c.Request.Context(), getHttpClient(isStream), provider, c.Request.Method, forwardPath, c.Request.Header, forwardBody)
		if err != nil {
			if canRetry {
				continue
			}
			c.Error(InternalServerError.WithMessage("forward request failed", err))
			return
		}

		if resp.StatusCode >= 400 {
			if canRetry {
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
			c.Writer.Write(bs)
			return
		}

		defer resp.Body.Close()

		if isStream {
			_, sseSpan := trace.StartSpan(c.Request.Context(), "sse stream")
			reader := converter.ConvertSSE(resp.Body)
			copyResponseHeaders(c, resp.Header)
			c.Status(resp.StatusCode)
			proxySSE(c, reader)
			sseSpan.End()
			return
		}

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			c.Error(InternalServerError.WithMessage("read response body failed", err))
			return
		}

		_, convertRespSpan := trace.StartSpan(c.Request.Context(), "convert response")
		respBody, err = converter.ConvertResponse(respBody)
		if err != nil {
			trace.SetError(convertRespSpan, err)
			convertRespSpan.End()
			c.Error(InternalServerError.WithMessage("convert response body failed", err))
			return
		}
		convertRespSpan.End()

		copyResponseHeaders(c, resp.Header)
		c.Status(resp.StatusCode)
		c.Writer.Write(respBody)
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

func forwardRequest(ctx context.Context, httpClient *http.Client, provider *config.ProviderConfig, method, path string, header http.Header, body map[string]interface{}) (*http.Response, error) {
	url := strings.TrimSuffix(provider.BaseURL, "/") + path

	var bodyReader io.Reader
	if body != nil {
		reqBytes, _ := json.Marshal(body)
		bodyReader = strings.NewReader(string(reqBytes))
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
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

	return httpClient.Do(req)
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

func proxySSE(c *gin.Context, reader io.Reader) {
	ctx := c.Request.Context()

	flusher, _ := c.Writer.(http.Flusher)
	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, err := reader.Read(buf)
		if n > 0 {
			c.Writer.Write(buf[:n])
			flusher.Flush()
			c.Writer.Flush()
		}
		if err != nil {
			break
		}
	}
}
