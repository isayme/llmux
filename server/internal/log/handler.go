package log

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for the log API.
type Handler struct {
	service *Service
}

// NewHandler creates a new log Handler backed by the given Service.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ListRequestLogs returns a paginated list of request logs with optional filters.
func (h *Handler) ListRequestLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filters := make(map[string]interface{})

	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filters["start_time"] = t
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filters["end_time"] = t
		}
	}
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if model := c.Query("model"); model != "" {
		filters["model"] = model
	}
	if providerID := c.Query("provider_id"); providerID != "" {
		filters["provider_id"] = providerID
	}

	logs, total, err := h.service.GetRequestLogs(page, pageSize, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetRequestLog returns a single request log and its provider calls.
func (h *Handler) GetRequestLog(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	log, err := h.service.GetRequestLogByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	calls, err := h.service.GetProviderCallsByRequestLogID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  log,
		"calls": calls,
	})
}

// GetProviderCalls returns all provider calls for a given request log.
func (h *Handler) GetProviderCalls(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	calls, err := h.service.GetProviderCallsByRequestLogID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": calls})
}

// DeleteLogs removes old logs. The "days" query param controls how many days
// back to keep (default 7).
func (h *Handler) DeleteLogs(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	cutoff := time.Now().AddDate(0, 0, -days)

	deleted, err := h.service.DeleteLogs(cutoff)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": deleted})
}
