package log

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestListRequestLogs(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	svc := NewService(store)
	handler := NewHandler(svc)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/logs/requests", handler.ListRequestLogs)

	req := httptest.NewRequest("GET", "/logs/requests?page=1&page_size=10", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if _, ok := response["data"]; !ok {
		t.Error("Expected data field in response")
	}
}
