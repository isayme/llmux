package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"llmux/internal/config"

	"github.com/gin-gonic/gin"
)

func setupTestConfig(t *testing.T, apiKeys []*config.ApiKeyConfig) func() {
	t.Helper()

	origDir, _ := os.Getwd()

	tmpDir, err := os.MkdirTemp("", "llmux-test")
	if err != nil {
		t.Fatal(err)
	}

	cfgContent := "server:\n  port: 8080\n  master_key: test\n"
	if err := os.WriteFile(tmpDir+"/config.yaml", []byte(cfgContent), 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatal(err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatal(err)
	}

	config.LoadConfig()

	if apiKeys != nil {
		config.Get().APIKeys = apiKeys
	} else {
		config.Get().APIKeys = nil
	}

	return func() {
		os.Chdir(origDir)
		os.RemoveAll(tmpDir)
	}
}

func TestListAPIKeys_Empty(t *testing.T) {
	cleanup := setupTestConfig(t, nil)
	defer cleanup()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/api-keys", nil)

	ListAPIKeys(c)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}

	keys, ok := resp["api_keys"]
	if !ok {
		t.Fatal("expected api_keys field in response")
	}

	keysArr, ok := keys.([]interface{})
	if !ok {
		t.Fatalf("expected api_keys to be an array, got %T", keys)
	}
	if len(keysArr) != 0 {
		t.Errorf("expected empty array, got %v", keysArr)
	}

	bodyStr := string(w.Body.Bytes())
	if bodyStr != `{"api_keys":[]}` {
		t.Errorf("expected exact JSON `{\"api_keys\":[]}`, got %s", bodyStr)
	}
}

func TestListAPIKeys_WithItems(t *testing.T) {
	cleanup := setupTestConfig(t, nil)
	defer cleanup()

	config.Get().APIKeys = []*config.ApiKeyConfig{
		{Name: "key1", Key: "sk-xxx", Enabled: true},
		{Name: "key2", Key: "sk-yyy", Enabled: false},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/api-keys", nil)

	ListAPIKeys(c)

	var resp struct {
		APIKeys []config.ApiKeyConfig `json:"api_keys"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}

	if len(resp.APIKeys) != 2 {
		t.Fatalf("expected 2 api keys, got %d", len(resp.APIKeys))
	}
	if resp.APIKeys[0].Name != "key1" {
		t.Errorf("expected first key name=key1, got %q", resp.APIKeys[0].Name)
	}
	if resp.APIKeys[1].Name != "key2" {
		t.Errorf("expected second key name=key2, got %q", resp.APIKeys[1].Name)
	}
}

func TestListAPIKeys_ResponseIsArrayNotNil(t *testing.T) {
	cleanup := setupTestConfig(t, nil)
	defer cleanup()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/api-keys", nil)

	ListAPIKeys(c)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}

	keys := resp["api_keys"]
	bodyStr := string(w.Body.Bytes())
	switch keys.(type) {
	case nil:
		t.Errorf("api_keys should not be null, got body: %s", bodyStr)
	case []interface{}:
		// ok
	default:
		t.Errorf("api_keys should be an array, got %T, body: %s", keys, bodyStr)
	}
}
