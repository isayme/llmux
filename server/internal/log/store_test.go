package log

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Verify tables were created
	if !store.db.Migrator().HasTable(&RequestLog{}) {
		t.Error("RequestLog table not created")
	}
	if !store.db.Migrator().HasTable(&ProviderCall{}) {
		t.Error("ProviderCall table not created")
	}
}

func TestStoreCreateAndReadRequestLog(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	log := &RequestLog{
		RequestID:   "test-uuid-123",
		Timestamp:   time.Now(),
		Model:       "gpt-4",
		Method:      "POST",
		Path:        "/v1/chat/completions",
		RequestBody: []byte(`{"model":"gpt-4"}`),
		Status:      "success",
	}

	if err := store.CreateRequestLog(log); err != nil {
		t.Fatalf("CreateRequestLog failed: %v", err)
	}

	if log.ID == 0 {
		t.Error("Expected ID to be set after create")
	}

	readLog, err := store.GetRequestLogByID(log.ID)
	if err != nil {
		t.Fatalf("GetRequestLogByID failed: %v", err)
	}

	if readLog.RequestID != log.RequestID {
		t.Errorf("Expected RequestID %s, got %s", log.RequestID, readLog.RequestID)
	}
}
