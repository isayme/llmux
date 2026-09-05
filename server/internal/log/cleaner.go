package log

import (
	"log/slog"
	"time"
)

// Cleaner periodically deletes expired log entries based on the configured retention period.
type Cleaner struct {
	service       *Service
	retentionDays int
	ticker        *time.Ticker
	stopCh        chan struct{}
	doneCh        chan struct{}
}

// NewCleaner creates a new Cleaner that will use the given Service to delete logs
// older than retentionDays.
func NewCleaner(service *Service, retentionDays int) *Cleaner {
	if retentionDays <= 0 {
		retentionDays = 7
	}
	return &Cleaner{
		service:       service,
		retentionDays: retentionDays,
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
	}
}

// Start begins the periodic cleanup at the given interval.
func (c *Cleaner) Start(interval time.Duration) {
	if interval <= 0 {
		interval = 1 * time.Hour
	}
	c.ticker = time.NewTicker(interval)

	go func() {
		defer close(c.doneCh)
		for {
			select {
			case <-c.ticker.C:
				c.cleanExpired()
			case <-c.stopCh:
				c.ticker.Stop()
				return
			}
		}
	}()
}

// Stop signals the cleaner to stop and waits for it to finish.
func (c *Cleaner) Stop() {
	close(c.stopCh)
	<-c.doneCh
}

// cleanExpired deletes log entries older than the configured retention period.
func (c *Cleaner) cleanExpired() {
	cutoff := time.Now().AddDate(0, 0, -c.retentionDays)
	deleted, err := c.service.DeleteLogs(cutoff)
	if err != nil {
		slog.Error("Failed to clean expired logs", "error", err)
		return
	}
	if deleted > 0 {
		slog.Info("Cleaned expired log entries", "count", deleted)
	}
}
