package trace

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace/noop"
)

var Tracer = noop.NewTracerProvider().Tracer("llmux")

// configurableSampler implements sdktrace.Sampler with a probability ratio.
type configurableSampler struct {
	ratio float64
}

func (s configurableSampler) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	if rand.Float64() < s.ratio {
		return sdktrace.SamplingResult{Decision: sdktrace.RecordAndSample}
	}
	return sdktrace.SamplingResult{Decision: sdktrace.Drop}
}

func (s configurableSampler) Description() string {
	return fmt.Sprintf("ConfigurableSampler{ratio=%.2f}", s.ratio)
}

// Init creates a TracerProvider based on config. Returns nil when disabled.
func Init(enabled bool, exporter string, endpoint string, samplingRatio float64) (*sdktrace.TracerProvider, error) {
	if !enabled {
		slog.Info("trace disabled, using noop tracer")
		return nil, nil
	}

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("llmux"),
	)

	var exp sdktrace.SpanExporter
	var err error
	switch exporter {
	case "otlp":
		opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(endpoint)}
		if endpoint == "" {
			opts = append(opts, otlptracehttp.WithEndpoint("localhost:4318"))
		}
		exp, err = otlptracehttp.New(context.Background(), opts...)
		if err != nil {
			return nil, fmt.Errorf("otlp exporter: %w", err)
		}
		slog.Info("trace exporter", "type", "otlp", "endpoint", endpoint)
	default:
		exp, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, fmt.Errorf("stdout exporter: %w", err)
		}
		slog.Info("trace exporter", "type", "stdout")
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(configurableSampler{ratio: samplingRatio}),
	)

	otel.SetTracerProvider(tp)
	Tracer = tp.Tracer("llmux")
	return tp, nil
}

// Shutdown gracefully flushes pending spans.
func Shutdown(ctx context.Context, tp *sdktrace.TracerProvider) {
	if tp == nil {
		return
	}
	if err := tp.Shutdown(ctx); err != nil {
		slog.Error("trace shutdown failed", "err", err)
	}
}
