package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"llmux/internal/config"
	"llmux/internal/constant"
	"llmux/internal/util"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var restHttpClient = &http.Client{
	Timeout: 60 * 1000,
}

var sseHttpClient = &http.Client{
	Timeout: 0,
}

func ChatCompletionsHandler(c *gin.Context) {
	var req map[string]interface{}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Error(BadRequest.WithMessage("json parse req body failed", err))
		return
	}

	model, err := getModel(req)
	if err != nil {
		c.Error(BadRequest.WithMessage("get model failed", err))
		return
	}

	providerId, model, err := parseModel(model)
	if err != nil {
		c.Error(BadRequest.WithMessage("parse model failed", err))
		return
	}
	// set model for provider
	req["model"] = model

	provider, err := findProvider(providerId)
	if err != nil {
		c.Error(NotFound.WithMessage("provider not found", err))
		return
	}

	if !provider.Enabled {
		c.Error(Forbidden.WithMessage(fmt.Sprintf("provider %s disabled", providerId)))
		return
	}

	if provider.Type != constant.ProviderTypeOpenAI {
		c.Error(BadRequest.WithMessage(fmt.Sprintf("provider it not '%s' type", constant.ProviderTypeOpenAI)))
		return
	}

	isStream, err := isStream(req)
	if err != nil {
		c.Error(BadRequest.WithMessage("get stream failed", err))
		return
	}

	resp, err := forwardRequest(c.Request.Context(), getHttpClient(isStream), provider, c.Request.Method, "/v1/chat/completions", c.Request.Header, req)
	if err != nil {
		c.Error(InternalServerError.WithMessage("forward request failed", err))
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}
	c.Status(resp.StatusCode)

	if isStream {
		proxySse(c, resp)
		return
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.Error(InternalServerError.WithMessage("read response body failed", err))
		return
	}

	c.Writer.Write(respBody)
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

func getAliasedModel(model string) string {
	for name, modelAlias := range config.Get().Aliases {
		if name == model {
			return fmt.Sprintf("%s/%s", modelAlias.Provider, modelAlias.Model)
		}
	}

	return model
}

func parseModel(model string) (string, string, error) {
	model = getAliasedModel(model)

	parts := strings.SplitN(model, "/", 2)
	if len(parts) != 2 {
		return "", "", errors.New("invalid model format")
	}

	return parts[0], parts[1], nil
}

func findProvider(providerId string) (*config.ProviderConfig, error) {
	for id, provider := range config.Get().Providers {
		if id == providerId {
			return provider, nil
		}
	}

	return nil, errors.New("provider not found")
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
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	req.Header.Set("Authorization", bearerPrefix+provider.APIKey)

	return httpClient.Do(req)
}

func getModel(reqBody map[string]interface{}) (string, error) {
	return util.GetString(reqBody, "model")
}

func isStream(reqBody map[string]interface{}) (bool, error) {
	return util.GetBool(reqBody, "stream")
}

func AnthropicMessagesHandler(c *gin.Context) {
	var req map[string]interface{}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Error(BadRequest.WithMessage("json parse req body failed", err))
		return
	}

	model, err := getModel(req)
	if err != nil {
		c.Error(BadRequest.WithMessage("get model failed", err))
		return
	}

	providerId, model, err := parseModel(model)
	if err != nil {
		c.Error(BadRequest.WithMessage("parse model failed", err))
		return
	}
	req["model"] = model

	provider, err := findProvider(providerId)
	if err != nil {
		c.Error(NotFound.WithMessage("provider not found", err))
		return
	}

	if !provider.Enabled {
		c.Error(Forbidden.WithMessage(fmt.Sprintf("provider %s disabled", providerId)))
		return
	}

	if provider.Type != constant.ProviderTypeAnthropic {
		c.Error(BadRequest.WithMessage(fmt.Sprintf("provider it not '%s' type", constant.ProviderTypeAnthropic)))
		return
	}

	isStream, err := isStream(req)
	if err != nil {
		c.Error(BadRequest.WithMessage("get stream failed", err))
		return
	}

	resp, err := forwardRequest(c.Request.Context(), getHttpClient(isStream), provider, c.Request.Method, "/v1/messages", c.Request.Header, req)
	if err != nil {
		c.Error(InternalServerError.WithMessage("forward request failed", err))
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}
	c.Status(resp.StatusCode)

	if isStream {
		proxySse(c, resp)
		return
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.Error(InternalServerError.WithMessage("read response body failed", err))
		return
	}

	c.Writer.Write(respBody)
}

func getHttpClient(isStream bool) *http.Client {
	if isStream {
		return sseHttpClient
	}

	return restHttpClient
}

func proxySse(c *gin.Context, resp *http.Response) {
	ctx := c.Request.Context()

	flusher, _ := c.Writer.(http.Flusher)
	reader := resp.Body
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
