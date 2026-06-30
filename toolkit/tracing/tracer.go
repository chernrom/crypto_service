package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	active     bool
	tracerName string
)

type Span struct {
	otelSpan trace.Span
}

func (s *Span) SetError(err error) {
	if err == nil || s == nil {
		return
	}

	s.otelSpan.RecordError(err)
	s.otelSpan.SetStatus(codes.Error, err.Error())
}

func Start(ctx context.Context, name string) (context.Context, *Span, func()) {
	if name == "" {
		name = "unknown"
	}

	if active {
		ctx, otelSpan := otel.Tracer(tracerName).Start(ctx, name)
		return ctx, &Span{otelSpan: otelSpan}, func() {
			otelSpan.End()
		}
	}

	return ctx, &Span{otelSpan: trace.SpanFromContext(ctx)}, func() {}
}

func StartTracer(ctx context.Context, serviceName, serviceVersion, endpoint string) (func(context.Context) error, error) {
	tracerName = serviceName
	exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(endpoint), otlptracehttp.WithInsecure())
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx, resource.WithAttributes(semconv.ServiceName(serviceName), semconv.ServiceVersion(serviceVersion)))
	if err != nil {
		return nil, err
	}

	telemetryProvider := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter), sdktrace.WithResource(res))
	otel.SetTracerProvider(telemetryProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	
	return telemetryProvider.Shutdown, nil
}

func ActivateTracer() {
	active = true
}
