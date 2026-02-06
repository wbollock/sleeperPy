package otel

import (
	"context"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// Init initializes OpenTelemetry and returns cleanup function
func Init(ctx context.Context) func() {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("sleeperpy"),
			semconv.ServiceVersion("1.0.0"),
			semconv.DeploymentEnvironment(getEnv()),
		),
	)
	if err != nil {
		log.Fatalf("failed to create resource: %v", err)
	}

	// Setup trace provider
	tracerProvider := initTracer(ctx, res)
	otel.SetTracerProvider(tracerProvider)

	// Setup meter provider
	meterProvider := initMeter(ctx, res)
	otel.SetMeterProvider(meterProvider)

	// Return cleanup function
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := tracerProvider.Shutdown(ctx); err != nil {
			log.Printf("error shutting down tracer: %v", err)
		}
		if err := meterProvider.Shutdown(ctx); err != nil {
			log.Printf("error shutting down meter: %v", err)
		}
	}
}

func initTracer(ctx context.Context, res *resource.Resource) *trace.TracerProvider {
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(getOTELEndpoint()),
		otlptracegrpc.WithInsecure(), // Use TLS in production
	)
	if err != nil {
		log.Fatalf("failed to create trace exporter: %v", err)
	}

	return trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
		trace.WithSampler(getSampler()),
	)
}

func initMeter(ctx context.Context, res *resource.Resource) *metric.MeterProvider {
	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(getOTELEndpoint()),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("failed to create metric exporter: %v", err)
	}

	return metric.NewMeterProvider(
		metric.WithReader(
			metric.NewPeriodicReader(exporter,
				metric.WithInterval(10*time.Second),
			),
		),
		metric.WithResource(res),
	)
}

func getOTELEndpoint() string {
	if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); endpoint != "" {
		return endpoint
	}
	return "localhost:4317" // Default
}

func getEnv() string {
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		return env
	}
	return "development"
}

func getSampler() trace.Sampler {
	// Sample 100% in dev, 10% in prod
	if getEnv() == "production" {
		return trace.TraceIDRatioBased(0.1)
	}
	return trace.AlwaysSample()
}
