package log

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
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

	reqLog, err := svc.StartRequest("test-uuid", "gpt-4", "gpt-4", "POST", "/v1/chat/completions", []byte(`{}`), "127.0.0.1", "key-123")
	if err != nil {
		t.Fatalf("StartRequest failed: %v", err)
	}

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

func TestLogServiceCompleteRequest(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	svc := NewService(store)

	reqLog, err := svc.StartRequest("test-uuid", "gpt-4", "gpt-4", "POST", "/v1/chat/completions", []byte(`{}`), "127.0.0.1", "key-123")
	if err != nil {
		t.Fatalf("StartRequest failed: %v", err)
	}

	// Wait a bit to ensure duration > 0
	time.Sleep(10 * time.Millisecond)

	if err := svc.CompleteRequest(reqLog, "success"); err != nil {
		t.Fatalf("CompleteRequest failed: %v", err)
	}

	if reqLog.Status != "success" {
		t.Errorf("Expected Status success, got %s", reqLog.Status)
	}
	if reqLog.Duration <= 0 {
		t.Errorf("Expected Duration > 0, got %d", reqLog.Duration)
	}
}

func TestLogServiceLogProviderCallEnd(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	svc := NewService(store)

	reqLog, err := svc.StartRequest("test-uuid", "gpt-4", "gpt-4", "POST", "/v1/chat/completions", []byte(`{}`), "127.0.0.1", "key-123")
	if err != nil {
		t.Fatalf("StartRequest failed: %v", err)
	}

	call, err := svc.LogProviderCallStart(reqLog.ID, "openai", "openai", "gpt-4", []byte(`{"model":"gpt-4"}`), false)
	if err != nil {
		t.Fatalf("LogProviderCallStart failed: %v", err)
	}

	responseHeader := []byte(`{"content-type":"application/json"}`)
	responseBody := []byte(`{"choices":[{"message":{"content":"hello"}}]}`)

	if err := svc.LogProviderCallEnd(call, 200, responseHeader, responseBody, 150, nil); err != nil {
		t.Fatalf("LogProviderCallEnd failed: %v", err)
	}

	if call.ResponseCode != 200 {
		t.Errorf("Expected ResponseCode 200, got %d", call.ResponseCode)
	}
	if call.Duration != 150 {
		t.Errorf("Expected Duration 150, got %d", call.Duration)
	}
	if call.Error != "" {
		t.Errorf("Expected empty Error, got %s", call.Error)
	}
	if string(call.ResponseHeader) != string(responseHeader) {
		t.Errorf("Expected ResponseHeader %s, got %s", responseHeader, call.ResponseHeader)
	}
	if string(call.ResponseBody) != string(responseBody) {
		t.Errorf("Expected ResponseBody %s, got %s", responseBody, call.ResponseBody)
	}
}

func TestLogServiceLogProviderCallEndWithError(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	svc := NewService(store)

	reqLog, err := svc.StartRequest("test-uuid", "gpt-4", "gpt-4", "POST", "/v1/chat/completions", []byte(`{}`), "127.0.0.1", "key-123")
	if err != nil {
		t.Fatalf("StartRequest failed: %v", err)
	}

	call, err := svc.LogProviderCallStart(reqLog.ID, "openai", "openai", "gpt-4", []byte(`{"model":"gpt-4"}`), true)
	if err != nil {
		t.Fatalf("LogProviderCallStart failed: %v", err)
	}

	testErr := fmt.Errorf("connection refused")
	if err := svc.LogProviderCallEnd(call, 500, nil, nil, 50, testErr); err != nil {
		t.Fatalf("LogProviderCallEnd failed: %v", err)
	}

	if call.ResponseCode != 500 {
		t.Errorf("Expected ResponseCode 500, got %d", call.ResponseCode)
	}
	if call.Error != "connection refused" {
		t.Errorf("Expected Error 'connection refused', got %s", call.Error)
	}
	if !call.IsRetry {
		t.Error("Expected IsRetry true")
	}
}
