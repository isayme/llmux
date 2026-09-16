package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"llmux/internal/config"
	llmuxlog "llmux/internal/log"

	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
)

// --- Test Helpers ---

func setupProxyTestConfig(t *testing.T, providers map[string]*config.ProviderConfig, aliases map[string]*config.ModelAlias, apiKeys []*config.ApiKeyConfig) func() {
	t.Helper()

	origDir, _ := os.Getwd()

	tmpDir, err := os.MkdirTemp("", "llmux-proxy-test")
	if err != nil {
		t.Fatal(err)
	}

	// Copy test config file to temp directory
	src, err := os.ReadFile("../../config/test.yaml")
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to read test config: %v", err)
	}
	if err := os.WriteFile(tmpDir+"/config.yaml", src, 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatal(err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatal(err)
	}

	config.LoadConfig()

	if providers != nil {
		config.Get().Providers = providers
	}
	if aliases != nil {
		config.Get().Aliases = aliases
	}
	if apiKeys != nil {
		config.Get().APIKeys = apiKeys
	} else {
		config.Get().APIKeys = []*config.ApiKeyConfig{
			{Name: "test-key", Key: "sk-test-key", Enabled: true},
		}
	}

	return func() {
		os.Chdir(origDir)
		os.RemoveAll(tmpDir)
	}
}

func newTestProxyHandler(t *testing.T) *ProxyHandler {
	t.Helper()

	store, err := llmuxlog.NewStore(config.LoggingConfig{Type: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatalf("Failed to create log store: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	logService := llmuxlog.NewService(store)
	return NewProxyHandler(logService)
}

func fakeOpenAIHandler(t *testing.T, responseBody map[string]interface{}) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseBody)
	}
}

func fakeOpenAIStreamHandler(t *testing.T, chunks []string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("response writer does not support flushing")
		}

		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}
}

func fakeAnthropicHandler(t *testing.T, responseBody map[string]interface{}) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseBody)
	}
}

func fakeAnthropicStreamHandler(t *testing.T, events []map[string]interface{}) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("response writer does not support flushing")
		}

		for _, event := range events {
			data, _ := json.Marshal(event)
			eventType, _ := event["type"].(string)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(data))
			flusher.Flush()
		}
	}
}

func fakeErrorHandler(t *testing.T, statusCode int, errorBody map[string]interface{}) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(errorBody)
	}
}

func fakeRetryHandler(t *testing.T, failStatus int, successBody map[string]interface{}) *retryServer {
	t.Helper()
	return &retryServer{
		t:            t,
		failStatus:   failStatus,
		successBody:  successBody,
		callCount:    0,
		mu:           sync.Mutex{},
	}
}

type retryServer struct {
	t           *testing.T
	failStatus  int
	successBody map[string]interface{}
	callCount   int
	mu          sync.Mutex
}

func (s *retryServer) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.callCount++
		count := s.callCount
		s.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if count == 1 {
			w.WriteHeader(s.failStatus)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"message": "rate limited",
					"type":    "rate_limit_error",
				},
			})
			return
		}
		json.NewEncoder(w).Encode(s.successBody)
	}
}

func (s *retryServer) CallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.callCount
}

func createTestContext(handler http.HandlerFunc) (*httptest.Server, *gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	ts := httptest.NewServer(handler)
	c.Request, _ = http.NewRequest("POST", "/v1/chat/completions", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Authorization", "Bearer sk-test-key")

	return ts, c, w
}

func setAPIKeyContext(c *gin.Context) {
	c.Set(apiKeyCtxKey, "sk-test-key")
}

func parseResponseBody(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response body: %v\nBody: %s", err, w.Body.String())
	}
	return resp
}

// --- OpenAI Non-Streaming Tests ---

func TestHandleProxy_OpenAI_NonStreaming_Success(t *testing.T) {
	cleanup := setupProxyTestConfig(t, nil, nil, nil)
	defer cleanup()

	h := newTestProxyHandler(t)

	openAIResp := map[string]interface{}{
		"id":      "chatcmpl-test-123",
		"object":  "chat.completion",
		"model":   "gpt-4",
		"choices": []interface{}{},
	}

	ts, c, w := createTestContext(fakeOpenAIHandler(t, openAIResp))
	defer ts.Close()

	config.Get().Providers = map[string]*config.ProviderConfig{
		"openai": {
			ID:      "openai",
			APIKey:  "sk-test",
			BaseURL: ts.URL,
			Type:    "openai",
			Enabled: true,
		},
	}

	body := map[string]interface{}{
		"model":  "openai/gpt-4",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello"},
		},
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	setAPIKeyContext(c)

	h.handleProxy(c)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	resp := parseResponseBody(t, w)
	if resp["id"] != "chatcmpl-test-123" {
		t.Errorf("Expected id chatcmpl-test-123, got %v", resp["id"])
	}
}

func TestHandleProxy_OpenAI_NonStreaming_WithModelReplacement(t *testing.T) {
	cleanup := setupProxyTestConfig(t, nil, nil, nil)
	defer cleanup()

	h := newTestProxyHandler(t)

	var receivedBody map[string]interface{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     "chatcmpl-test",
			"object": "chat.completion",
		})
	}))
	defer ts.Close()

	config.Get().Providers = map[string]*config.ProviderConfig{
		"openai": {
			ID:      "openai",
			APIKey:  "sk-test",
			BaseURL: ts.URL,
			Type:    "openai",
			Enabled: true,
		},
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/v1/chat/completions", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Authorization", "Bearer sk-test-key")

	body := map[string]interface{}{
		"model":  "openai/gpt-4",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello"},
		},
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	setAPIKeyContext(c)

	h.handleProxy(c)

	if receivedBody["model"] != "gpt-4" {
		t.Errorf("Expected model to be replaced to 'gpt-4', got %v", receivedBody["model"])
	}
}

// --- OpenAI Streaming Tests ---

func TestHandleProxy_OpenAI_Streaming_Success(t *testing.T) {
	cleanup := setupProxyTestConfig(t, nil, nil, nil)
	defer cleanup()

	h := newTestProxyHandler(t)

	chunks := []string{
		`{"id":"chatcmpl-test","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
		`{"id":"chatcmpl-test","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}`,
		`{"id":"chatcmpl-test","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
	}

	ts, c, w := createTestContext(fakeOpenAIStreamHandler(t, chunks))
	defer ts.Close()

	config.Get().Providers = map[string]*config.ProviderConfig{
		"openai": {
			ID:      "openai",
			APIKey:  "sk-test",
			BaseURL: ts.URL,
			Type:    "openai",
			Enabled: true,
		},
	}

	body := map[string]interface{}{
		"model":  "openai/gpt-4",
		"stream": true,
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello"},
		},
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	setAPIKeyContext(c)

	h.handleProxy(c)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	bodyStr := w.Body.String()
	if !strings.Contains(bodyStr, "data:") {
		t.Error("Expected SSE data in response")
	}
	if !strings.Contains(bodyStr, "[DONE]") {
		t.Error("Expected [DONE] in response")
	}
}

func TestHandleProxy_OpenAI_Streaming_ContentType(t *testing.T) {
	cleanup := setupProxyTestConfig(t, nil, nil, nil)
	defer cleanup()

	h := newTestProxyHandler(t)

	chunks := []string{
		`{"id":"chatcmpl-test","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"Hi"},"finish_reason":null}]}`,
	}

	ts, c, w := createTestContext(fakeOpenAIStreamHandler(t, chunks))
	defer ts.Close()

	config.Get().Providers = map[string]*config.ProviderConfig{
		"openai": {
			ID:      "openai",
			APIKey:  "sk-test",
			BaseURL: ts.URL,
			Type:    "openai",
			Enabled: true,
		},
	}

	body := map[string]interface{}{
		"model":  "openai/gpt-4",
		"stream": true,
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello"},
		},
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	setAPIKeyContext(c)

	h.handleProxy(c)

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/event-stream") {
		t.Errorf("Expected Content-Type to contain text/event-stream, got %s", contentType)
	}
}

// --- Anthropic Non-Streaming Tests ---

func TestHandleProxy_Anthropic_NonStreaming_Success(t *testing.T) {
	cleanup := setupProxyTestConfig(t, nil, nil, nil)
	defer cleanup()

	h := newTestProxyHandler(t)

	anthropicResp := map[string]interface{}{
		"id":      "msg-test-123",
		"type":    "message",
		"role":    "assistant",
		"content": []interface{}{},
		"model":   "claude-3-opus-20240229",
	}

	ts, c, w := createTestContext(fakeAnthropicHandler(t, anthropicResp))
	defer ts.Close()

	config.Get().Providers = map[string]*config.ProviderConfig{
		"anthropic": {
			ID:      "anthropic",
			APIKey:  "sk-ant-test",
			BaseURL: ts.URL,
			Type:    "anthropic",
			Enabled: true,
		},
	}

	body := map[string]interface{}{
		"model":      "anthropic/claude-3-opus-20240229",
		"max_tokens": 1024,
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello"},
		},
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
	c.Request.Header.Set("anthropic-version", "2023-06-01")

	setAPIKeyContext(c)

	h.handleProxy(c)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	resp := parseResponseBody(t, w)
	if resp["id"] != "msg-test-123" {
		t.Errorf("Expected id msg-test-123, got %v", resp["id"])
	}
}

func TestHandleProxy_Anthropic_NonStreaming_PathRewrite(t *testing.T) {
	cleanup := setupProxyTestConfig(t, nil, nil, nil)
	defer cleanup()

	h := newTestProxyHandler(t)

	var receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      "msg-test",
			"type":    "message",
			"content": []interface{}{},
		})
	}))
	defer ts.Close()

	config.Get().Providers = map[string]*config.ProviderConfig{
		"anthropic": {
			ID:      "anthropic",
			APIKey:  "sk-ant-test",
			BaseURL: ts.URL,
			Type:    "anthropic",
			Enabled: true,
		},
	}

	body := map[string]interface{}{
		"model":      "anthropic/claude-3-opus-20240229",
		"max_tokens": 1024,
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/anthropic/v1/messages", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Authorization", "Bearer sk-test-key")
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	setAPIKeyContext(c)

	h.handleProxy(c)

	if receivedPath != "/v1/messages" {
		t.Errorf("Expected path /v1/messages, got %s", receivedPath)
	}
}

// --- Anthropic Streaming Tests ---

func TestHandleProxy_Anthropic_Streaming_Success(t *testing.T) {
	cleanup := setupProxyTestConfig(t, nil, nil, nil)
	defer cleanup()

	h := newTestProxyHandler(t)

	events := []map[string]interface{}{
		{"type": "message_start", "message": map[string]interface{}{"id": "msg-test"}},
		{"type": "content_block_start", "index": 0, "content_block": map[string]interface{}{"type": "text", "text": ""}},
		{"type": "content_block_delta", "index": 0, "delta": map[string]interface{}{"type": "text_delta", "text": "Hello"}},
		{"type": "content_block_delta", "index": 0, "delta": map[string]interface{}{"type": "text_delta", "text": " world"}},
		{"type": "content_block_stop", "index": 0},
		{"type": "message_delta", "delta": map[string]interface{}{"stop_reason": "end_turn"}, "usage": map[string]interface{}{"output_tokens": 10}},
		{"type": "message_stop"},
	}

	ts, c, w := createTestContext(fakeAnthropicStreamHandler(t, events))
	defer ts.Close()

	config.Get().Providers = map[string]*config.ProviderConfig{
		"anthropic": {
			ID:      "anthropic",
			APIKey:  "sk-ant-test",
			BaseURL: ts.URL,
			Type:    "anthropic",
			Enabled: true,
		},
	}

	body := map[string]interface{}{
		"model":      "anthropic/claude-3-opus-20240229",
		"max_tokens": 1024,
		"stream":     true,
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello"},
		},
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	setAPIKeyContext(c)

	h.handleProxy(c)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	bodyStr := w.Body.String()
	if !strings.Contains(bodyStr, "message_start") {
		t.Error("Expected message_start event in response")
	}
	if !strings.Contains(bodyStr, "content_block_delta") {
		t.Error("Expected content_block_delta event in response")
	}
}

// --- Retry Tests ---

func TestHandleProxy_Retry_On429(t *testing.T) {
	cleanup := setupProxyTestConfig(t, nil, nil, nil)
	defer cleanup()

	h := newTestProxyHandler(t)

	successBody := map[string]interface{}{
		"id":     "chatcmpl-test-retry",
		"object": "chat.completion",
	}

	retrySrv := fakeRetryHandler(t, 429, successBody)

	ts := httptest.NewServer(retrySrv.Handler())
	defer ts.Close()

	config.Get().Providers = map[string]*config.ProviderConfig{
		"openai": {
			ID:      "openai",
			APIKey:  "sk-test",
			BaseURL: ts.URL,
			Type:    "openai",
			Enabled: true,
			Retry: &config.RetryConfig{
				On429: true,
			},
		},
	}

	// Use fallback strategy with same provider repeated to allow retries
	config.Get().Aliases = map[string]*config.ModelAlias{
		"retry-test": {
			Name:     "retry-test",
			Strategy: "fallback",
			Models: []*config.ModelAliasItemConfig{
				{Provider: "openai", Model: "gpt-4", Weight: 1},
				{Provider: "openai", Model: "gpt-4", Weight: 1},
			},
			Enabled: true,
		},
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/v1/chat/completions", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Authorization", "Bearer sk-test-key")

	body := map[string]interface{}{
		"model": "retry-test",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello"},
		},
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	setAPIKeyContext(c)

	h.handleProxy(c)

	if w.Code != 200 {
		t.Errorf("Expected status 200 after retry, got %d. Body: %s", w.Code, w.Body.String())
	}

	if retrySrv.CallCount() != 2 {
		t.Errorf("Expected 2 calls (1 fail + 1 retry), got %d", retrySrv.CallCount())
	}
}

func TestHandleProxy_NoRetry_On400(t *testing.T) {
	cleanup := setupProxyTestConfig(t, nil, nil, nil)
	defer cleanup()

	h := newTestProxyHandler(t)

	var callCount int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"message": "invalid request",
				"type":    "invalid_request_error",
			},
		})
	}))
	defer ts.Close()

	config.Get().Providers = map[string]*config.ProviderConfig{
		"openai": {
			ID:      "openai",
			APIKey:  "sk-test",
			BaseURL: ts.URL,
			Type:    "openai",
			Enabled: true,
			Retry: &config.RetryConfig{
				On429: true,
				On4xx: true,
			},
		},
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/v1/chat/completions", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Authorization", "Bearer sk-test-key")

	body := map[string]interface{}{
		"model": "openai/gpt-4",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello"},
		},
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	setAPIKeyContext(c)

	h.handleProxy(c)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call (no retry for 400), got %d", callCount)
	}
}

// --- Fallback Tests ---

func TestHandleProxy_Fallback_PrimaryFails_SecondarySucceeds(t *testing.T) {
	cleanup := setupProxyTestConfig(t, nil, nil, nil)
	defer cleanup()

	h := newTestProxyHandler(t)

	// Primary provider fails
	primaryTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{"message": "internal error"},
		})
	}))
	defer primaryTS.Close()

	// Secondary provider succeeds
	secondaryTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     "chatcmpl-fallback",
			"object": "chat.completion",
		})
	}))
	defer secondaryTS.Close()

	config.Get().Providers = map[string]*config.ProviderConfig{
		"openai-primary": {
			ID:      "openai-primary",
			APIKey:  "sk-test",
			BaseURL: primaryTS.URL,
			Type:    "openai",
			Enabled: true,
			Retry: &config.RetryConfig{
				On5xx: true,
			},
		},
		"openai-secondary": {
			ID:      "openai-secondary",
			APIKey:  "sk-test",
			BaseURL: secondaryTS.URL,
			Type:    "openai",
			Enabled: true,
		},
	}

	config.Get().Aliases = map[string]*config.ModelAlias{
		"gpt-4-fallback": {
			Name:     "gpt-4-fallback",
			Strategy: "fallback",
			Models: []*config.ModelAliasItemConfig{
				{Provider: "openai-primary", Model: "gpt-4", Weight: 1},
				{Provider: "openai-secondary", Model: "gpt-4", Weight: 1},
			},
			Enabled: true,
		},
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/v1/chat/completions", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Authorization", "Bearer sk-test-key")

	body := map[string]interface{}{
		"model": "gpt-4-fallback",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello"},
		},
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	setAPIKeyContext(c)

	h.handleProxy(c)

	if w.Code != 200 {
		t.Errorf("Expected status 200 from fallback, got %d. Body: %s", w.Code, w.Body.String())
	}

	resp := parseResponseBody(t, w)
	if resp["id"] != "chatcmpl-fallback" {
		t.Errorf("Expected id chatcmpl-fallback, got %v", resp["id"])
	}
}

// --- Error Tests ---

func TestHandleProxy_InvalidModel(t *testing.T) {
	cleanup := setupProxyTestConfig(t, nil, nil, nil)
	defer cleanup()

	h := newTestProxyHandler(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/v1/chat/completions", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Authorization", "Bearer sk-test-key")

	body := map[string]interface{}{
		"model": "invalid-model-format",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello"},
		},
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	setAPIKeyContext(c)

	h.handleProxy(c)

	// Check that an error was added to the context
	if len(c.Errors) == 0 {
		t.Fatal("Expected errors in context, got none")
	}

	err := c.Errors.Last().Err
	httpErr, ok := err.(HttpError)
	if !ok {
		t.Fatalf("Expected HttpError, got %T: %v", err, err)
	}
	if httpErr.StatusCode != 400 {
		t.Errorf("Expected status 400, got %d", httpErr.StatusCode)
	}
}

func TestHandleProxy_MissingModel(t *testing.T) {
	cleanup := setupProxyTestConfig(t, nil, nil, nil)
	defer cleanup()

	h := newTestProxyHandler(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/v1/chat/completions", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Authorization", "Bearer sk-test-key")

	body := map[string]interface{}{
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello"},
		},
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	setAPIKeyContext(c)

	h.handleProxy(c)

	// Check that an error was added to the context
	if len(c.Errors) == 0 {
		t.Fatal("Expected errors in context, got none")
	}

	err := c.Errors.Last().Err
	httpErr, ok := err.(HttpError)
	if !ok {
		t.Fatalf("Expected HttpError, got %T: %v", err, err)
	}
	if httpErr.StatusCode != 400 {
		t.Errorf("Expected status 400, got %d", httpErr.StatusCode)
	}
}

func TestHandleProxy_DisabledProvider(t *testing.T) {
	cleanup := setupProxyTestConfig(t, nil, nil, nil)
	defer cleanup()

	h := newTestProxyHandler(t)

	config.Get().Providers = map[string]*config.ProviderConfig{
		"openai": {
			ID:      "openai",
			APIKey:  "sk-test",
			BaseURL: "http://unused",
			Type:    "openai",
			Enabled: false,
		},
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/v1/chat/completions", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Authorization", "Bearer sk-test-key")

	body := map[string]interface{}{
		"model": "openai/gpt-4",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello"},
		},
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	setAPIKeyContext(c)

	h.handleProxy(c)

	// Check that an error was added to the context
	if len(c.Errors) == 0 {
		t.Fatal("Expected errors in context, got none")
	}

	err := c.Errors.Last().Err
	httpErr, ok := err.(HttpError)
	if !ok {
		t.Fatalf("Expected HttpError, got %T: %v", err, err)
	}
	if httpErr.StatusCode != 403 {
		t.Errorf("Expected status 403, got %d", httpErr.StatusCode)
	}
}

func TestHandleProxy_ProviderForwardError(t *testing.T) {
	cleanup := setupProxyTestConfig(t, nil, nil, nil)
	defer cleanup()

	h := newTestProxyHandler(t)

	// Server that immediately closes connection
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			return
		}
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer ts.Close()

	config.Get().Providers = map[string]*config.ProviderConfig{
		"openai": {
			ID:      "openai",
			APIKey:  "sk-test",
			BaseURL: ts.URL,
			Type:    "openai",
			Enabled: true,
		},
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/v1/chat/completions", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Authorization", "Bearer sk-test-key")

	body := map[string]interface{}{
		"model": "openai/gpt-4",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello"},
		},
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	setAPIKeyContext(c)

	h.handleProxy(c)

	// Check that an error was added to the context
	if len(c.Errors) == 0 {
		t.Fatal("Expected errors in context, got none")
	}

	err := c.Errors.Last().Err
	httpErr, ok := err.(HttpError)
	if !ok {
		t.Fatalf("Expected HttpError, got %T: %v", err, err)
	}
	if httpErr.StatusCode != 500 {
		t.Errorf("Expected status 500, got %d", httpErr.StatusCode)
	}
}

// --- Request Logging Tests ---

func TestHandleProxy_Logging_NonStreaming(t *testing.T) {
	cleanup := setupProxyTestConfig(t, nil, nil, nil)
	defer cleanup()

	store, err := llmuxlog.NewStore(config.LoggingConfig{Type: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatalf("Failed to create log store: %v", err)
	}
	defer store.Close()

	logService := llmuxlog.NewService(store)
	h := NewProxyHandler(logService)

	ts, c, w := createTestContext(fakeOpenAIHandler(t, map[string]interface{}{
		"id":     "chatcmpl-test",
		"object": "chat.completion",
	}))
	defer ts.Close()

	config.Get().Providers = map[string]*config.ProviderConfig{
		"openai": {
			ID:      "openai",
			APIKey:  "sk-test",
			BaseURL: ts.URL,
			Type:    "openai",
			Enabled: true,
		},
	}

	body := map[string]interface{}{
		"model": "openai/gpt-4",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello"},
		},
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	setAPIKeyContext(c)

	h.handleProxy(c)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Verify request log was created
	logs, total, err := logService.GetRequestLogs(1, 10, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to get request logs: %v", err)
	}
	if total != 1 {
		t.Fatalf("Expected 1 request log, got %d", total)
	}

	log := logs[0]
	if log.Model != "openai/gpt-4" {
		t.Errorf("Expected model openai/gpt-4, got %s", log.Model)
	}
	if log.Method != "POST" {
		t.Errorf("Expected method POST, got %s", log.Method)
	}
	if log.Status != "success" {
		t.Errorf("Expected status success, got %s", log.Status)
	}
	if log.Duration < 0 {
		t.Errorf("Expected non-negative duration, got %d", log.Duration)
	}
	if len(log.RequestBody) == 0 {
		t.Error("Expected non-empty request body")
	}
	if len(log.ResponseBody) == 0 {
		t.Error("Expected non-empty response body")
	}

	// Verify provider call was logged
	calls, err := logService.GetProviderCallsByRequestLogID(log.ID)
	if err != nil {
		t.Fatalf("Failed to get provider calls: %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("Expected 1 provider call, got %d", len(calls))
	}

	call := calls[0]
	if call.ProviderID != "openai" {
		t.Errorf("Expected provider openai, got %s", call.ProviderID)
	}
	if call.ResponseCode != 200 {
		t.Errorf("Expected response code 200, got %d", call.ResponseCode)
	}
	if call.Duration < 0 {
		t.Errorf("Expected non-negative duration, got %d", call.Duration)
	}
	if call.IsRetry {
		t.Error("Expected IsRetry false for first call")
	}
}

func TestHandleProxy_Logging_Streaming(t *testing.T) {
	cleanup := setupProxyTestConfig(t, nil, nil, nil)
	defer cleanup()

	store, err := llmuxlog.NewStore(config.LoggingConfig{Type: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatalf("Failed to create log store: %v", err)
	}
	defer store.Close()

	logService := llmuxlog.NewService(store)
	h := NewProxyHandler(logService)

	chunks := []string{
		`{"id":"chatcmpl-test","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"Hi"},"finish_reason":null}]}`,
	}

	ts, c, w := createTestContext(fakeOpenAIStreamHandler(t, chunks))
	defer ts.Close()

	config.Get().Providers = map[string]*config.ProviderConfig{
		"openai": {
			ID:      "openai",
			APIKey:  "sk-test",
			BaseURL: ts.URL,
			Type:    "openai",
			Enabled: true,
		},
	}

	body := map[string]interface{}{
		"model":  "openai/gpt-4",
		"stream": true,
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello"},
		},
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	setAPIKeyContext(c)

	h.handleProxy(c)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Verify request log
	logs, total, err := logService.GetRequestLogs(1, 10, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to get request logs: %v", err)
	}
	if total != 1 {
		t.Fatalf("Expected 1 request log, got %d", total)
	}

	log := logs[0]
	if log.Status != "success" {
		t.Errorf("Expected status success, got %s", log.Status)
	}

	// Verify streaming response body was captured
	if len(log.ResponseBody) == 0 {
		t.Error("Expected non-empty response body for stream")
	}

	// Verify provider call
	calls, err := logService.GetProviderCallsByRequestLogID(log.ID)
	if err != nil {
		t.Fatalf("Failed to get provider calls: %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("Expected 1 provider call, got %d", len(calls))
	}

	call := calls[0]
	if call.ResponseCode != 200 {
		t.Errorf("Expected response code 200, got %d", call.ResponseCode)
	}
}

func TestHandleProxy_Logging_Retry(t *testing.T) {
	cleanup := setupProxyTestConfig(t, nil, nil, nil)
	defer cleanup()

	store, err := llmuxlog.NewStore(config.LoggingConfig{Type: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatalf("Failed to create log store: %v", err)
	}
	defer store.Close()

	logService := llmuxlog.NewService(store)
	h := NewProxyHandler(logService)

	successBody := map[string]interface{}{
		"id":     "chatcmpl-retry",
		"object": "chat.completion",
	}

	retrySrv := fakeRetryHandler(t, 429, successBody)
	ts := httptest.NewServer(retrySrv.Handler())
	defer ts.Close()

	config.Get().Providers = map[string]*config.ProviderConfig{
		"openai": {
			ID:      "openai",
			APIKey:  "sk-test",
			BaseURL: ts.URL,
			Type:    "openai",
			Enabled: true,
			Retry: &config.RetryConfig{
				On429: true,
			},
		},
	}

	// Use fallback strategy with same provider repeated to allow retries
	config.Get().Aliases = map[string]*config.ModelAlias{
		"retry-logging-test": {
			Name:     "retry-logging-test",
			Strategy: "fallback",
			Models: []*config.ModelAliasItemConfig{
				{Provider: "openai", Model: "gpt-4", Weight: 1},
				{Provider: "openai", Model: "gpt-4", Weight: 1},
			},
			Enabled: true,
		},
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/v1/chat/completions", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Authorization", "Bearer sk-test-key")

	body := map[string]interface{}{
		"model": "retry-logging-test",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello"},
		},
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	setAPIKeyContext(c)

	h.handleProxy(c)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Verify request log
	logs, total, err := logService.GetRequestLogs(1, 10, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to get request logs: %v", err)
	}
	if total != 1 {
		t.Fatalf("Expected 1 request log, got %d", total)
	}

	// Verify provider calls (should have 2: 1 failed + 1 success)
	calls, err := logService.GetProviderCallsByRequestLogID(logs[0].ID)
	if err != nil {
		t.Fatalf("Failed to get provider calls: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("Expected 2 provider calls, got %d", len(calls))
	}

	// First call should be the failed one
	if calls[0].ResponseCode != 429 {
		t.Errorf("Expected first call response code 429, got %d", calls[0].ResponseCode)
	}
	if calls[0].IsRetry {
		t.Error("Expected first call IsRetry false")
	}

	// Second call should be the successful retry
	if calls[1].ResponseCode != 200 {
		t.Errorf("Expected second call response code 200, got %d", calls[1].ResponseCode)
	}
	if !calls[1].IsRetry {
		t.Error("Expected second call IsRetry true")
	}
}

func TestHandleProxy_Logging_ProviderError(t *testing.T) {
	cleanup := setupProxyTestConfig(t, nil, nil, nil)
	defer cleanup()

	store, err := llmuxlog.NewStore(config.LoggingConfig{Type: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatalf("Failed to create log store: %v", err)
	}
	defer store.Close()

	logService := llmuxlog.NewService(store)
	h := NewProxyHandler(logService)

	ts, c, w := createTestContext(fakeErrorHandler(t, 500, map[string]interface{}{
		"error": map[string]interface{}{
			"message": "internal server error",
			"type":    "server_error",
		},
	}))
	defer ts.Close()

	config.Get().Providers = map[string]*config.ProviderConfig{
		"openai": {
			ID:      "openai",
			APIKey:  "sk-test",
			BaseURL: ts.URL,
			Type:    "openai",
			Enabled: true,
		},
	}

	body := map[string]interface{}{
		"model": "openai/gpt-4",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello"},
		},
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	setAPIKeyContext(c)

	h.handleProxy(c)

	if w.Code != 500 {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	// Verify request log status is "failed"
	logs, total, err := logService.GetRequestLogs(1, 10, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to get request logs: %v", err)
	}
	if total != 1 {
		t.Fatalf("Expected 1 request log, got %d", total)
	}
	if logs[0].Status != "failed" {
		t.Errorf("Expected status failed, got %s", logs[0].Status)
	}

	// Verify provider call has error response body
	calls, err := logService.GetProviderCallsByRequestLogID(logs[0].ID)
	if err != nil {
		t.Fatalf("Failed to get provider calls: %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("Expected 1 provider call, got %d", len(calls))
	}
	if calls[0].ResponseCode != 500 {
		t.Errorf("Expected response code 500, got %d", calls[0].ResponseCode)
	}
	if len(calls[0].ResponseBody) == 0 {
		t.Error("Expected non-empty response body for error")
	}
}

// --- Alias Resolution Tests ---

func TestHandleProxy_Alias_Resolution(t *testing.T) {
	cleanup := setupProxyTestConfig(t, nil, nil, nil)
	defer cleanup()

	h := newTestProxyHandler(t)

	var receivedModel string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		receivedModel, _ = body["model"].(string)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     "chatcmpl-test",
			"object": "chat.completion",
		})
	}))
	defer ts.Close()

	config.Get().Providers = map[string]*config.ProviderConfig{
		"openai": {
			ID:      "openai",
			APIKey:  "sk-test",
			BaseURL: ts.URL,
			Type:    "openai",
			Enabled: true,
		},
	}

	config.Get().Aliases = map[string]*config.ModelAlias{
		"my-gpt4": {
			Name:     "my-gpt4",
			Strategy: "round_robin",
			Models: []*config.ModelAliasItemConfig{
				{Provider: "openai", Model: "gpt-4-turbo", Weight: 1},
			},
			Enabled: true,
		},
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/v1/chat/completions", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Authorization", "Bearer sk-test-key")

	body := map[string]interface{}{
		"model": "my-gpt4",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello"},
		},
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	setAPIKeyContext(c)

	h.handleProxy(c)

	if receivedModel != "gpt-4-turbo" {
		t.Errorf("Expected model to be resolved to gpt-4-turbo, got %s", receivedModel)
	}
}

// --- Path Routing Tests ---

func TestHandleProxy_ProviderPath_Routing(t *testing.T) {
	tests := []struct {
		name         string
		providerType string
		expectedPath string
	}{
		{"openai default", "openai", "/chat/completions"},
		{"anthropic", "anthropic", "/v1/messages"},
		{"openai_responses", "openai_responses", "/v1/responses"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedPath string
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":     "chatcmpl-test",
					"object": "chat.completion",
				})
			}))
			defer ts.Close()

			config.Get().Providers = map[string]*config.ProviderConfig{
				"test-provider": {
					ID:      "test-provider",
					APIKey:  "sk-test",
					BaseURL: ts.URL,
					Type:    tt.providerType,
					Enabled: true,
				},
			}

			// Create a real log service for this test
			store, err := llmuxlog.NewStore(config.LoggingConfig{Type: "sqlite", DSN: ":memory:"})
			if err != nil {
				t.Fatalf("Failed to create log store: %v", err)
			}
			defer store.Close()
			logService := llmuxlog.NewService(store)

			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("POST", "/v1/chat/completions", nil)
			c.Request.Header.Set("Content-Type", "application/json")
			c.Request.Header.Set("Authorization", "Bearer sk-test-key")

			body := map[string]interface{}{
				"model": "test-provider/gpt-4",
				"messages": []interface{}{
					map[string]interface{}{"role": "user", "content": "Hello"},
				},
			}
			bodyBytes, _ := json.Marshal(body)
			c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
			c.Set(apiKeyCtxKey, "sk-test-key")

			h := NewProxyHandler(logService)

			h.handleProxy(c)

			if receivedPath != tt.expectedPath {
				t.Errorf("Expected path %s, got %s", tt.expectedPath, receivedPath)
			}
		})
	}
}

// --- Authorization Header Tests ---

func TestHandleProxy_AuthHeader_Forwarded(t *testing.T) {
	cleanup := setupProxyTestConfig(t, nil, nil, nil)
	defer cleanup()

	h := newTestProxyHandler(t)

	var receivedAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     "chatcmpl-test",
			"object": "chat.completion",
		})
	}))
	defer ts.Close()

	config.Get().Providers = map[string]*config.ProviderConfig{
		"openai": {
			ID:      "openai",
			APIKey:  "sk-real-key",
			BaseURL: ts.URL,
			Type:    "openai",
			Enabled: true,
		},
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/v1/chat/completions", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Authorization", "Bearer sk-test-key")

	body := map[string]interface{}{
		"model": "openai/gpt-4",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello"},
		},
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
	c.Set(apiKeyCtxKey, "sk-test-key")

	h.handleProxy(c)

	expectedAuth := "Bearer sk-real-key"
	if receivedAuth != expectedAuth {
		t.Errorf("Expected Authorization header %s, got %s", expectedAuth, receivedAuth)
	}
}

// --- ListModels Tests ---

func TestListModelsHandler(t *testing.T) {
	cleanup := setupProxyTestConfig(t, nil, nil, nil)
	defer cleanup()

	h := newTestProxyHandler(t)

	config.Get().Aliases = map[string]*config.ModelAlias{
		"gpt-4": {
			Name:    "gpt-4",
			Enabled: true,
		},
		"claude-3": {
			Name:    "claude-3",
			Enabled: true,
		},
		"disabled-model": {
			Name:    "disabled-model",
			Enabled: false,
		},
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/v1/models", nil)

	h.ListModelsHandler(c)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var resp struct {
		Object string      `json:"object"`
		Data   []modelInfo `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Object != "list" {
		t.Errorf("Expected object 'list', got %s", resp.Object)
	}

	// Should only include enabled models
	if len(resp.Data) != 2 {
		t.Errorf("Expected 2 models (enabled only), got %d", len(resp.Data))
	}

	modelIDs := make(map[string]bool)
	for _, m := range resp.Data {
		modelIDs[m.ID] = true
	}

	if !modelIDs["gpt-4"] {
		t.Error("Expected gpt-4 in response")
	}
	if !modelIDs["claude-3"] {
		t.Error("Expected claude-3 in response")
	}
	if modelIDs["disabled-model"] {
		t.Error("Expected disabled-model to be excluded")
	}
}

// --- CopyResponseHeaders Tests ---

func TestCopyResponseHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)

	upstream := http.Header{
		"Content-Type":    {"application/json"},
		"X-Custom-Header": {"custom-value"},
		"X-Single-Value":  {"only-one"},
		"Content-Length":   {"100"},
		"Content-Encoding": {"gzip"},
	}

	copyResponseHeaders(c, upstream)

	// Should be copied
	if c.Writer.Header().Get("Content-Type") != "application/json" {
		t.Error("Expected Content-Type to be copied")
	}
	if c.Writer.Header().Get("X-Custom-Header") != "custom-value" {
		t.Error("Expected X-Custom-Header to be copied")
	}
	if c.Writer.Header().Get("X-Single-Value") != "only-one" {
		t.Error("Expected X-Single-Value to be copied")
	}

	// Should be excluded
	if c.Writer.Header().Get("Content-Length") != "" {
		t.Error("Expected Content-Length to be excluded")
	}
	if c.Writer.Header().Get("Content-Encoding") != "" {
		t.Error("Expected Content-Encoding to be excluded")
	}
}

// --- Helper to create selector cache ---

func newSelectorCache() *cache.Cache {
	return cache.New(10*time.Minute, 5*time.Minute)
}
