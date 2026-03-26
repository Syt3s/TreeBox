package tracing

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/syt3s/TreeBox/internal/logging"
)

const instrumentationName = "github.com/syt3s/TreeBox/internal/tracing"

func Middleware(service string, opts ...Option) gin.HandlerFunc {
	cfg := newConfig(opts)
	tracer := cfg.TracerProvider.Tracer(
		instrumentationName,
		oteltrace.WithInstrumentationVersion("1.0.0"),
	)

	return func(c *gin.Context) {
		start := time.Now()
		baseCtx := c.Request.Context()

		ctx := cfg.Propagators.Extract(baseCtx, propagation.HeaderCarrier(c.Request.Header))
		spanOpts := []oteltrace.SpanStartOption{
			oteltrace.WithAttributes(semconv.NetAttributesFromHTTPRequest("tcp", c.Request)...),
			oteltrace.WithAttributes(semconv.EndUserAttributesFromHTTPRequest(c.Request)...),
			oteltrace.WithAttributes(semconv.HTTPServerAttributesFromHTTPRequest(service, c.FullPath(), c.Request)...),
			oteltrace.WithSpanKind(oteltrace.SpanKindServer),
		}

		spanName := c.FullPath()
		if spanName == "" {
			spanName = c.Request.URL.Path
		}
		if spanName == "" {
			spanName = fmt.Sprintf("HTTP %s route not found", c.Request.Method)
		}

		ctx, span := tracer.Start(ctx, spanName, spanOpts...)
		defer span.End()

		c.Request = c.Request.WithContext(ctx)
		c.Next()

		status := c.Writer.Status()
		attrs := semconv.HTTPAttributesFromHTTPStatusCode(status)
		spanStatus, spanMessage := semconv.SpanStatusFromHTTPStatusCodeAndSpanKind(status, oteltrace.SpanKindServer)
		span.SetAttributes(attrs...)
		span.SetStatus(spanStatus, spanMessage)

		logger := logging.FromContext(ctx).With(
			zap.String("component", "http.access"),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.Int("status", status),
			zap.Duration("latency", time.Since(start)),
			zap.String("remote_addr", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		)

		switch {
		case status >= http.StatusInternalServerError:
			logger.Error("http request completed")
		case status >= http.StatusBadRequest:
			logger.Warn("http request completed")
		default:
			logger.Info("http request completed")
		}
	}
}
