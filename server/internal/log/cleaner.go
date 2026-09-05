package log

import (
	"log"
	"time"
)

// Cleaner periodically deletes expired log entries based on the configured retention period.
type Cleaner struct {
	service       *Service
	retentionDays int
	ticker        *time.Ticker
	stopCh        chan struct{}
}

// NewCleaner creates a new Cleaner that will use the given Service to delete logs
// older than retentionDays.
func NewCleaner(service *Service, retentionDays int) *Cleaner {
	return &Cleaner{
		service:       service,
		retentionDays: retentionDays,
		stopCh:        make(chan struct{}),
	}
}

// Start begins the periodic cleanup at the given interval.
func (c *Cleaner) Start(interval time.Duration) {
	c.ticker = time.NewTicker(interval)

	go func() {
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
}

// cleanExpired deletes log entries older than the configured retention period.
func (c *Cleaner) cleanExpired() {
	cutoff := time.Now().AddDate(0, 0, -c.retentionDays)
	deleted, err := c.service.DeleteLogs(cutoff)
	if err != nil {
		log.Printf("Failed to clean expired logs: %v", err)
		return
	}
	if deleted > 0 {
		log.Printf("Cleaned %d expired log entries", deleted)
	}
}
