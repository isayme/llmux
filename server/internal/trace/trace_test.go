package trace

import (
	"context"
	"fmt"
	"testing"

	"go.opentelemetry.io/otel/attribute"
)

func TestConfigurableSampler_RatioOne(t *testing.T) {
	s := configurableSampler{ratio: 1.0}
	desc := s.Description()
	if desc == "" {
		t.Error("expected non-empty description")
	}
}

func TestConfigurableSampler_RatioZero(t *testing.T) {
	s := configurableSampler{ratio: 0.0}
	desc := s.Description()
	if desc == "" {
		t.Error("expected non-empty description")
	}
}

func TestInit_Disabled(t *testing.T) {
	tp, err := Init(false, "stdout", "", 1.0)
	if err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	if tp != nil {
		t.Error("expected nil TracerProvider when disabled")
	}
}

func TestStartSpan(t *testing.T) {
	ctx := context.Background()
	ctx2, span := StartSpan(ctx, "test-span")
	if span == nil {
		t.Error("expected non-nil span")
	}
	span.End()
	_ = ctx2
}

func TestSetError(t *testing.T) {
	ctx := context.Background()
	_, span := StartSpan(ctx, "test-span")
	SetError(span, fmt.Errorf("test error"))
	span.End()
}

func TestSetAttributes(t *testing.T) {
	ctx := context.Background()
	_, span := StartSpan(ctx, "test-span")
	SetAttributes(span, attribute.String("key", "value"))
	span.End()
}

func TestAddEvent(t *testing.T) {
	ctx := context.Background()
	_, span := StartSpan(ctx, "test-span")
	AddEvent(span, "test-event", attribute.String("event-key", "event-value"))
	span.End()
}

func TestSetSamplingConfig(t *testing.T) {
	cfg := MiddlewareSamplingConfig{
		Ratio:          0.5,
		ForceOnError:   true,
		ForceOnLatency: 0,
	}
	SetSamplingConfig(cfg)
	got := getSamplingConfig()
	if got.Ratio != 0.5 {
		t.Errorf("expected ratio 0.5, got %f", got.Ratio)
	}
	if !got.ForceOnError {
		t.Error("expected ForceOnError true")
	}
}
