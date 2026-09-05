package log

import (
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Store provides SQLite-backed persistence for log data.
type Store struct {
	db *gorm.DB
}

// NewStore opens (or creates) the SQLite database at dbPath and auto-migrates
// the RequestLog and ProviderCall tables.
func NewStore(dbPath string) (*Store, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&RequestLog{}, &ProviderCall{}); err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

// Close releases the underlying database connection.
func (s *Store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// CreateRequestLog inserts a new request log entry.
func (s *Store) CreateRequestLog(log *RequestLog) error {
	return s.db.Create(log).Error
}

// UpdateRequestLog persists changes to an existing request log entry.
func (s *Store) UpdateRequestLog(log *RequestLog) error {
	return s.db.Save(log).Error
}

// CreateProviderCall inserts a new provider call entry.
func (s *Store) CreateProviderCall(call *ProviderCall) error {
	return s.db.Create(call).Error
}

// UpdateProviderCall persists changes to an existing provider call entry.
func (s *Store) UpdateProviderCall(call *ProviderCall) error {
	return s.db.Save(call).Error
}

// GetRequestLogs returns a paginated, filtered list of request logs and the
// total count matching the filters.
// Supported filter keys: start_time, end_time, status, model, provider_id.
func (s *Store) GetRequestLogs(page, pageSize int, filters map[string]interface{}) ([]RequestLog, int64, error) {
	var logs []RequestLog
	var total int64

	query := s.db.Model(&RequestLog{})

	if startTime, ok := filters["start_time"]; ok {
		query = query.Where("timestamp >= ?", startTime)
	}
	if endTime, ok := filters["end_time"]; ok {
		query = query.Where("timestamp <= ?", endTime)
	}
	if status, ok := filters["status"]; ok {
		query = query.Where("status = ?", status)
	}
	if model, ok := filters["model"]; ok {
		query = query.Where("model LIKE ?", "%"+model.(string)+"%")
	}
	if providerID, ok := filters["provider_id"]; ok {
		query = query.Where("id IN (SELECT request_log_id FROM provider_calls WHERE provider_id = ?)", providerID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Order("timestamp DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetRequestLogByID retrieves a single request log by its primary key.
func (s *Store) GetRequestLogByID(id uint) (*RequestLog, error) {
	var log RequestLog
	if err := s.db.First(&log, id).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

// GetProviderCallsByRequestLogID returns all provider calls associated with
// the given request log, ordered by creation time.
func (s *Store) GetProviderCallsByRequestLogID(requestLogID uint) ([]ProviderCall, error) {
	var calls []ProviderCall
	if err := s.db.Where("request_log_id = ?", requestLogID).Order("created_at ASC").Find(&calls).Error; err != nil {
		return nil, err
	}
	return calls, nil
}

// DeleteBefore removes all provider calls and request logs created before the
// given cutoff time. Returns the number of request logs deleted.
func (s *Store) DeleteBefore(cutoff time.Time) (int64, error) {
	result := s.db.Where("created_at < ?", cutoff).Delete(&ProviderCall{})
	if result.Error != nil {
		return 0, result.Error
	}

	result = s.db.Where("created_at < ?", cutoff).Delete(&RequestLog{})
	return result.RowsAffected, result.Error
}
