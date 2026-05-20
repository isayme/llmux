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
	"llmux/internal/util"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var restHttpClient = &http.Client{
	Timeout: 60 * time.Second,
}

var sseHttpClient = &http.Client{
	Timeout: 0,
}

var anthropicVersion = "2023-06-01"

func ChatCompletionsHandler(c *gin.Context) {
	handleProxy(c, convert.ProtocolOpenAI)
}

func AnthropicMessagesHandler(c *gin.Context) {
	handleProxy(c, convert.ProtocolAnthropic)
}

func handleProxy(c *gin.Context, usedProtocol convert.UsedAIProtocol) {
	var req map[string]interface{}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Error(BadRequest.WithMessage("json parse req body failed", err))
		return
	}

	rawModel, err := getModel(req)
	if err != nil {
		c.Error(BadRequest.WithMessage("get model failed", err))
		return
	}

	selector, err := resolveModelSelector(rawModel)
	if err != nil {
		c.Error(BadRequest.WithMessage("resolve alias failed", err))
		return
	}

	isStream, err := isStream(req)
	if err != nil {
		c.Error(BadRequest.WithMessage("get stream failed", err))
		return
	}

	for {
		providerId, model, canRetry := selector.Next()
		// slog.Info("using", "provider", providerId, "model", model, "canRetry", canRetry)
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

		forwardBody := converter.ConvertRequest(req)

		forwardPath := getProviderPath(provider.Type)

		resp, err := forwardRequest(c.Request.Context(), getHttpClient(isStream), provider, c.Request.Method, forwardPath, c.Request.Header, forwardBody)
		if err != nil {
			if canRetry {
				// slog.Info("forwardRequest Fail", "err", err)
				continue
			}
			c.Error(InternalServerError.WithMessage("forward request failed", err))
			return
		}

		if resp.StatusCode >= 400 {
			if canRetry {
				// slog.Info("forwardRequest Fail", "StatusCode", resp.StatusCode)
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
			reader := converter.ConvertSSE(resp.Body)
			copyResponseHeaders(c, resp.Header)
			c.Status(resp.StatusCode)
			proxySseReader(c, reader)
			return
		}

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			c.Error(InternalServerError.WithMessage("read response body failed", err))
			return
		}

		respBody, err = converter.ConvertResponse(respBody)
		if err != nil {
			c.Error(InternalServerError.WithMessage("convert response body failed", err))
			return
		}

		copyResponseHeaders(c, resp.Header)
		c.Status(resp.StatusCode)
		c.Writer.Write(respBody)
		return
	}
}

// copyResponseHeaders copies upstream response headers to the client.
// When converted is true, headers that would conflict with the transformed
// body (Content-Length, Content-Encoding) are stripped.
func copyResponseHeaders(c *gin.Context, upstream http.Header) {
	for key, values := range upstream {
		switch key {
		case "Content-Length":
			// Body size changed after protocol conversion.
			continue
		case "Content-Encoding":
			// Body was decompressed by the HTTP client, then re-encoded
			// as raw JSON after conversion.
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

func ListModelsHandler(c *gin.Context) {
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

func resolveModelSelector(modelOrAlias string) (selector strategy.ModelSelector, err error) {
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

	if len(aliasConfig.Models) > 0 {
		selector = strategy.NewSelector(aliasConfig.Strategy, aliasConfig.Models)
		return selector, nil
	}

	return nil, errors.New("alias has no models configured")
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
	if providerType == "anthropic" {
		return "/v1/messages"
	}
	return "/chat/completions"
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
		if key == "Content-Length" {
			continue
		}
		if key == "Accept-Encoding" {
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

func proxySse(c *gin.Context, resp *http.Response) {
	proxySseReader(c, resp.Body)
}

func proxySseReader(c *gin.Context, reader io.Reader) {
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
