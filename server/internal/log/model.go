package log

import (
	"time"
)

// RequestLog stores information about an incoming LLM API request
type RequestLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	RequestID   string    `gorm:"uniqueIndex;size:36" json:"request_id"`
	Timestamp   time.Time `gorm:"index" json:"timestamp"`
	Model       string    `gorm:"size:100;index" json:"model"`
	Alias       string    `gorm:"size:100;index" json:"alias"`
	Method      string    `gorm:"size:10" json:"method"`
	Path        string    `gorm:"size:100" json:"path"`
	RequestBody []byte    `json:"request_body"`
	ClientIP    string    `gorm:"size:50" json:"client_ip"`
	APIKeyID    string    `gorm:"size:50;index" json:"api_key_id"`
	Duration    int64     `json:"duration"`
	Status      string    `gorm:"size:20;index" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// ProviderCall stores information about a single provider call
type ProviderCall struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	RequestLogID   uint      `gorm:"index" json:"request_log_id"`
	ProviderID     string    `gorm:"size:50;index" json:"provider_id"`
	ProviderType   string    `gorm:"size:50" json:"provider_type"`
	Model          string    `gorm:"size:100" json:"model"`
	RequestBody    []byte    `json:"request_body"`
	ResponseCode   int       `json:"response_code"`
	ResponseHeader []byte    `json:"response_header"`
	ResponseBody   []byte    `json:"response_body"`
	Duration       int64     `json:"duration"`
	IsRetry        bool      `json:"is_retry"`
	Error          string    `gorm:"size:500" json:"error"`
	CreatedAt      time.Time `json:"created_at"`
}
