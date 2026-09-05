package log

import (
	"path/filepath"
	"testing"
)

func TestLogServiceStartRequest(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	svc := NewService(store)

	reqLog := svc.StartRequest("test-uuid", "gpt-4", "gpt-4", "POST", "/v1/chat/completions", []byte(`{}`), "127.0.0.1", "key-123")

	if reqLog == nil {
		t.Fatal("Expected non-nil RequestLog")
	}
	if reqLog.RequestID != "test-uuid" {
		t.Errorf("Expected RequestID test-uuid, got %s", reqLog.RequestID)
	}
	if reqLog.Status != "pending" {
		t.Errorf("Expected Status pending, got %s", reqLog.Status)
	}
}
