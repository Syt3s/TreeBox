package service

import (
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/syt3s/TreeBox/internal/branding"
	"github.com/syt3s/TreeBox/internal/conf"
	"github.com/syt3s/TreeBox/internal/context"
	"github.com/syt3s/TreeBox/internal/logging"
)

func Proxy(ctx context.Context) error {
	span := trace.SpanFromContext(ctx.Request().Context())

	var userID uint
	if ctx.IsLogged {
		userID = ctx.User.ID
	}

	if span.SpanContext().IsValid() {
		span.SetAttributes(
			attribute.Int(branding.TelemetryNamespace+".service.user-id", int(userID)),
		)
	}

	uri := ctx.Param("**")
	basePath := strings.Split(uri, "/")[0]
	forwardPath := strings.TrimPrefix(uri, basePath)
	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "service.proxy"),
		zap.String("base_path", basePath),
		zap.String("forward_path", forwardPath),
		zap.Uint("user_id", userID),
	)

	var forwardURLStr string
	for _, backend := range conf.Service.Backends {
		if backend.Prefix == basePath {
			forwardURLStr = backend.ForwardURL
			break
		}
	}
	if len(forwardURLStr) == 0 {
		return ctx.JSONError(http.StatusNotFound, "页面不存在")
	}

	forwardURL, err := url.Parse(forwardURLStr)
	if err != nil {
		logger.Error("failed to parse forward url", zap.Error(err), zap.String("forward_url", forwardURLStr))
		return ctx.JSONError(http.StatusInternalServerError, "服务网关内部错误")
	}

	reverseProxy := httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL = forwardURL
			req.URL.Path = strings.TrimRight(req.URL.Path, "/") + forwardPath
			req.Host = forwardURL.Host

			traceHeaders := http.Header{}
			otel.GetTextMapPropagator().Inject(ctx.Request().Context(), propagation.HeaderCarrier(traceHeaders))
			for key := range traceHeaders {
				req.Header.Set(key, traceHeaders.Get(key))
			}

			req.Header.Set(branding.GatewayHeaderFrom, branding.GatewayName)
			req.Header.Set(branding.LegacyGatewayHeaderFrom, branding.LegacyGatewayName)
			req.Header.Set(branding.GatewayHeaderUserID, strconv.Itoa(int(userID)))
			req.Header.Set(branding.LegacyGatewayHeaderUserID, strconv.Itoa(int(userID)))
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		ErrorHandler: func(writer http.ResponseWriter, request *http.Request, err error) {
			logger.Error("failed to proxy service request", zap.Error(err))
			_ = ctx.JSONError(http.StatusInternalServerError, "服务网关内部错误")
		},
	}

	reverseProxy.ServeHTTP(ctx.ResponseWriter(), ctx.Request())
	return nil
}
