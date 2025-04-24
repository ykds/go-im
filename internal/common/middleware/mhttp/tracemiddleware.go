package mhttp

import (
	"go-im/internal/pkg/mtrace"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

func Trace() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		propagator := otel.GetTextMapPropagator()
		name := ctx.Request.URL.Path
		ctx2 := propagator.Extract(ctx.Request.Context(), propagation.HeaderCarrier(ctx.Request.Header))
		ctx2, span := mtrace.StartSpan(ctx2, name, trace.WithSpanKind(trace.SpanKindServer), trace.WithAttributes(semconv.HTTPServerAttributesFromHTTPRequest(mtrace.TraceName, name, ctx.Request)...))
		defer mtrace.EndSpan(span)
		ctx.Request = ctx.Request.WithContext(ctx2)
		ctx.Next()
	}
}
