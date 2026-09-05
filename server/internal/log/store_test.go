package log

import (
	"fmt"
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

func newTestStore(t *testing.T) *Store {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	return store
}

func TestGetRequestLogsFilterByStatus(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	now := time.Now()

	// Create logs with different statuses
	statuses := []string{"success", "error", "success", "timeout"}
	for i, status := range statuses {
		store.CreateRequestLog(&RequestLog{
			RequestID: fmt.Sprintf("req-%s-%d", status, i),
			Timestamp: now,
			Model:     "gpt-4",
			Status:    status,
		})
	}

	// Filter by "success"
	logs, total, err := store.GetRequestLogs(1, 10, map[string]interface{}{
		"status": "success",
	})
	if err != nil {
		t.Fatalf("GetRequestLogs failed: %v", err)
	}
	if total != 2 {
		t.Errorf("Expected total 2, got %d", total)
	}
	if len(logs) != 2 {
		t.Errorf("Expected 2 logs, got %d", len(logs))
	}
	for _, l := range logs {
		if l.Status != "success" {
			t.Errorf("Expected status 'success', got %s", l.Status)
		}
	}
}

func TestGetRequestLogsFilterByModel(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	now := time.Now()

	// Create logs with different models
	for _, model := range []string{"gpt-4", "claude-3-opus", "gpt-4-turbo", "claude-3-sonnet"} {
		store.CreateRequestLog(&RequestLog{
			RequestID: "req-" + model,
			Timestamp: now,
			Model:     model,
			Status:    "success",
		})
	}

	// Filter by "gpt" - should match gpt-4 and gpt-4-turbo
	logs, total, err := store.GetRequestLogs(1, 10, map[string]interface{}{
		"model": "gpt",
	})
	if err != nil {
		t.Fatalf("GetRequestLogs failed: %v", err)
	}
	if total != 2 {
		t.Errorf("Expected total 2, got %d", total)
	}
	if len(logs) != 2 {
		t.Errorf("Expected 2 logs, got %d", len(logs))
	}

	// Filter by non-string model value should be safely ignored
	logs, total, err = store.GetRequestLogs(1, 10, map[string]interface{}{
		"model": 42,
	})
	if err != nil {
		t.Fatalf("GetRequestLogs with non-string model failed: %v", err)
	}
	if total != 4 {
		t.Errorf("Expected total 4 (no filter applied), got %d", total)
	}
}

func TestGetRequestLogsFilterByTimeRange(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	t1 := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)
	t4 := time.Date(2025, 1, 25, 0, 0, 0, 0, time.UTC)

	timestamps := []time.Time{t1, t2, t3, t4}
	for i, ts := range timestamps {
		store.CreateRequestLog(&RequestLog{
			RequestID: "req-time-" + string(rune('a'+i)),
			Timestamp: ts,
			Model:     "gpt-4",
			Status:    "success",
		})
	}

	// Filter: between t2 and t3 inclusive
	logs, total, err := store.GetRequestLogs(1, 10, map[string]interface{}{
		"start_time": t2,
		"end_time":   t3,
	})
	if err != nil {
		t.Fatalf("GetRequestLogs failed: %v", err)
	}
	if total != 2 {
		t.Errorf("Expected total 2, got %d", total)
	}
	if len(logs) != 2 {
		t.Errorf("Expected 2 logs, got %d", len(logs))
	}
}

func TestGetRequestLogsPagination(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	now := time.Now()
	for i := 0; i < 10; i++ {
		store.CreateRequestLog(&RequestLog{
			RequestID: "req-page-" + string(rune('a'+i)),
			Timestamp: now.Add(time.Duration(i) * time.Minute),
			Model:     "gpt-4",
			Status:    "success",
		})
	}

	// Page 1, size 3
	logs, total, err := store.GetRequestLogs(1, 3, map[string]interface{}{})
	if err != nil {
		t.Fatalf("GetRequestLogs failed: %v", err)
	}
	if total != 10 {
		t.Errorf("Expected total 10, got %d", total)
	}
	if len(logs) != 3 {
		t.Errorf("Expected 3 logs on page 1, got %d", len(logs))
	}

	// Page 4 (last page), size 3 -> should have 1 item
	logs, total, err = store.GetRequestLogs(4, 3, map[string]interface{}{})
	if err != nil {
		t.Fatalf("GetRequestLogs failed: %v", err)
	}
	if total != 10 {
		t.Errorf("Expected total 10, got %d", total)
	}
	if len(logs) != 1 {
		t.Errorf("Expected 1 log on page 4, got %d", len(logs))
	}
}

func TestDeleteBeforeTransactional(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	old := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	recent := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	// Create old request log with associated provider call
	oldLog := &RequestLog{
		RequestID: "old-req",
		Timestamp: old,
		Model:     "gpt-4",
		Status:    "success",
		CreatedAt: old,
	}
	store.CreateRequestLog(oldLog)

	store.CreateProviderCall(&ProviderCall{
		RequestLogID: oldLog.ID,
		ProviderID:   "openai",
		ProviderType: "openai",
		Model:        "gpt-4",
		CreatedAt:    old,
	})

	// Create recent request log with associated provider call
	recentLog := &RequestLog{
		RequestID: "recent-req",
		Timestamp: recent,
		Model:     "gpt-4",
		Status:    "success",
		CreatedAt: recent,
	}
	store.CreateRequestLog(recentLog)

	store.CreateProviderCall(&ProviderCall{
		RequestLogID: recentLog.ID,
		ProviderID:   "openai",
		ProviderType: "openai",
		Model:        "gpt-4",
		CreatedAt:    recent,
	})

	// Delete everything before mid-2025
	cutoff := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	deleted, err := store.DeleteBefore(cutoff)
	if err != nil {
		t.Fatalf("DeleteBefore failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("Expected 1 request log deleted, got %d", deleted)
	}

	// Verify old provider call was deleted
	var oldCallCount int64
	store.db.Model(&ProviderCall{}).Where("request_log_id = ?", oldLog.ID).Count(&oldCallCount)
	if oldCallCount != 0 {
		t.Errorf("Expected old provider call to be deleted, but found %d", oldCallCount)
	}

	// Verify recent data remains
	var recentCallCount int64
	store.db.Model(&ProviderCall{}).Where("request_log_id = ?", recentLog.ID).Count(&recentCallCount)
	if recentCallCount != 1 {
		t.Errorf("Expected recent provider call to remain, found %d", recentCallCount)
	}

	var recentLogCount int64
	store.db.Model(&RequestLog{}).Where("id = ?", recentLog.ID).Count(&recentLogCount)
	if recentLogCount != 1 {
		t.Errorf("Expected recent request log to remain, found %d", recentLogCount)
	}
}

func TestDeleteBeforeNoData(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	// Delete from empty database should return 0 and no error
	deleted, err := store.DeleteBefore(time.Now())
	if err != nil {
		t.Fatalf("DeleteBefore on empty DB failed: %v", err)
	}
	if deleted != 0 {
		t.Errorf("Expected 0 deleted, got %d", deleted)
	}
}
