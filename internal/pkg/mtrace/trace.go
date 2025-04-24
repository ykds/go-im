package mtrace

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	SQLKey   = attribute.Key("exec.sql")
	SQLError = attribute.Key("exec.sql.error")

	RedisExecCmd   = attribute.Key("redis.cmd")
	RedisExecError = attribute.Key("redis.error")

	GRPCStatusCodeKey = attribute.Key("rpc.grpc.status_code")
	GrpcErrorCode     = attribute.Key("rpc.grpc.error.code")
	GrpcErrorMsg      = attribute.Key("rpc.grpc.error.msg")
)

var (
	TraceName string
	enable    bool
)

type Config struct {
	Name     string  `json:"name" yaml:"name"`
	Endpoint string  `json:"endpoint" yaml:"endpoint"`
	Sampler  float64 `json:"sampler" yaml:"sampler"`
	Enable   bool    `json:"enable" yaml:"enable"`
}

func InitTelemetry(cfg Config) {
	if !cfg.Enable {
		enable = cfg.Enable
		return
	}
	TraceName = cfg.Name
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceNameKey.String(cfg.Name))),
	}
	if strings.TrimSpace(cfg.Endpoint) != "" {
		exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(cfg.Endpoint)))
		if err != nil {
			panic(err)
		}
		opts = append(opts, sdktrace.WithBatcher(exporter))
	}
	if cfg.Sampler > 0 {
		opts = append(opts, sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.Sampler))))
	}
	tp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
}

func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if !enable {
		return ctx, nil
	}
	tr := otel.Tracer(TraceName)
	return tr.Start(ctx, name, opts...)
}

func EndSpan(span trace.Span) {
	if !enable {
		return
	}
	if span != nil {
		span.End()
	}
}
