package trace

import (
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/gin-gonic/gin"
)

// MiddlewareSamplingConfig holds runtime sampling settings for the middleware.
type MiddlewareSamplingConfig struct {
	Ratio          float64
	ForceOnError   bool
	ForceOnLatency time.Duration
}

var (
	samplingCfg MiddlewareSamplingConfig
	cfgMu       sync.RWMutex
)

// SetSamplingConfig updates the sampling configuration used by the middleware.
func SetSamplingConfig(cfg MiddlewareSamplingConfig) {
	cfgMu.Lock()
	defer cfgMu.Unlock()
	samplingCfg = cfg
}

func getSamplingConfig() MiddlewareSamplingConfig {
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	return samplingCfg
}

// Middleware creates a root span for each Gin request.
// Implements W3C Trace Context propagation.
func Middleware() gin.HandlerFunc {
	propagator := propagation.TraceContext{}

	return func(c *gin.Context) {
		// 1. Extract incoming W3C trace context from request headers
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// 2. Create root span
		spanName := fmt.Sprintf("%s %s", c.Request.Method, c.FullPath())
		startTime := time.Now()

		ctx, span := Tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPRequestMethodKey.String(c.Request.Method),
				semconv.URLPath(c.Request.URL.Path),
			),
		)

		// Store trace_id in Gin context for future logging integration
		if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
			c.Set("trace_id", sc.TraceID().String())
		}

		// 3. Propagate updated context to downstream handlers
		c.Request = c.Request.WithContext(ctx)

		// 4. Execute downstream handler chain
		c.Next()

		// 5. Record response attributes
		status := c.Writer.Status()
		latency := time.Since(startTime)
		span.SetAttributes(
			semconv.HTTPResponseStatusCode(status),
			attribute.Int64("http.latency_ms", latency.Milliseconds()),
		)

		if status >= 400 {
			span.SetStatus(2, fmt.Sprintf("HTTP %d", status))
		}

		// 6. Apply sampling overrides
		cfg := getSamplingConfig()
		shouldForce := false
		if cfg.ForceOnError && status >= 400 {
			shouldForce = true
		}
		if cfg.ForceOnLatency > 0 && latency > cfg.ForceOnLatency {
			shouldForce = true
		}
		if shouldForce {
			span.SetAttributes(attribute.Bool("sampling.forced", true))
		}

		// 7. Inject trace context into response headers
		propagator.Inject(ctx, propagation.HeaderCarrier(c.Writer.Header()))

		span.End()
	}
}

// GetTraceID extracts the trace_id from the Gin context.
func GetTraceID(c *gin.Context) string {
	if id, ok := c.Get("trace_id"); ok {
		if s, ok := id.(string); ok {
			return s
		}
	}
	return ""
}
