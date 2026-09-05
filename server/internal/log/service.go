package log

import (
	"time"
)

// Service provides a higher-level API for logging requests and provider calls.
// It wraps the Store and adds business logic like calculating request duration.
type Service struct {
	store *Store
}

// NewService creates a new logging Service backed by the given Store.
func NewService(store *Store) *Service {
	return &Service{store: store}
}

// StartRequest creates a new RequestLog entry with status "pending" and
// persists it via the store. Returns the created RequestLog.
func (s *Service) StartRequest(requestID, model, alias, method, path string, body []byte, clientIP, apiKeyID string) (*RequestLog, error) {
	log := &RequestLog{
		RequestID:   requestID,
		Timestamp:   time.Now(),
		Model:       model,
		Alias:       alias,
		Method:      method,
		Path:        path,
		RequestBody: body,
		ClientIP:    clientIP,
		APIKeyID:    apiKeyID,
		Status:      "pending",
	}

	if err := s.store.CreateRequestLog(log); err != nil {
		return nil, err
	}
	return log, nil
}

// LogProviderCallStart creates a new ProviderCall entry and persists it.
// Returns the created ProviderCall.
func (s *Service) LogProviderCallStart(requestLogID uint, providerID, providerType, model string, requestBody []byte, isRetry bool) (*ProviderCall, error) {
	call := &ProviderCall{
		RequestLogID: requestLogID,
		ProviderID:   providerID,
		ProviderType: providerType,
		Model:        model,
		RequestBody:  requestBody,
		IsRetry:      isRetry,
	}

	if err := s.store.CreateProviderCall(call); err != nil {
		return nil, err
	}
	return call, nil
}

// LogProviderCallEnd updates an existing ProviderCall with the response details.
func (s *Service) LogProviderCallEnd(call *ProviderCall, responseCode int, responseHeader, responseBody []byte, duration int64, err error) error {
	call.ResponseCode = responseCode
	call.ResponseHeader = responseHeader
	call.ResponseBody = responseBody
	call.Duration = duration
	if err != nil {
		call.Error = err.Error()
	}

	return s.store.UpdateProviderCall(call)
}

// CompleteRequest marks a RequestLog as completed with the given status
// and calculates the total duration from when the request was started.
func (s *Service) CompleteRequest(log *RequestLog, status string) error {
	log.Status = status
	log.Duration = time.Since(log.Timestamp).Milliseconds()
	return s.store.UpdateRequestLog(log)
}

// GetRequestLogs returns a paginated, filtered list of request logs and total count.
func (s *Service) GetRequestLogs(page, pageSize int, filters map[string]interface{}) ([]RequestLog, int64, error) {
	return s.store.GetRequestLogs(page, pageSize, filters)
}

// GetRequestLogByID retrieves a single request log by its primary key.
func (s *Service) GetRequestLogByID(id uint) (*RequestLog, error) {
	return s.store.GetRequestLogByID(id)
}

// GetProviderCallsByRequestLogID returns all provider calls for a given request log.
func (s *Service) GetProviderCallsByRequestLogID(requestLogID uint) ([]ProviderCall, error) {
	return s.store.GetProviderCallsByRequestLogID(requestLogID)
}

// DeleteLogs removes all request logs and provider calls created before the
// given cutoff time. Returns the number of request logs deleted.
func (s *Service) DeleteLogs(cutoff time.Time) (int64, error) {
	return s.store.DeleteBefore(cutoff)
}
