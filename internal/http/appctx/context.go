package appctx

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/syt3s/TreeBox/internal/branding"
	"github.com/syt3s/TreeBox/internal/config"
	"github.com/syt3s/TreeBox/internal/model"
)

type EndpointType string

const (
	EndpointAPI EndpointType = "api"
	EndpointWeb EndpointType = "web"
)

const contextKey = "treebox.context"

var ErrHandled = errors.New("request handled")

func (e EndpointType) IsAPI() bool {
	return e == EndpointAPI
}

func (e EndpointType) IsWeb() bool {
	return e == EndpointWeb
}

type Context struct {
	*gin.Context

	User     *model.User
	IsLogged bool
	endpoint EndpointType
}

func FromGin(c *gin.Context) *Context {
	if value, ok := c.Get(contextKey); ok {
		if ctx, ok := value.(*Context); ok {
			return ctx
		}
	}

	ctx := &Context{
		Context:  c,
		endpoint: EndpointWeb,
	}
	c.Set(contextKey, ctx)
	return ctx
}

func Wrap(handler func(Context) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := FromGin(c)
		if err := handler(*ctx); err != nil && !errors.Is(err, ErrHandled) {
			_ = c.Error(err)
			c.Abort()
		}
	}
}

func APIEndpoint() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := FromGin(c)
		ctx.endpoint = EndpointAPI
		c.Next()
	}
}

func (c *Context) Request() *http.Request {
	return c.Context.Request
}

func (c *Context) ResponseWriter() gin.ResponseWriter {
	return c.Context.Writer
}

func (c *Context) IsAPIRequest() bool {
	return c.endpoint.IsAPI() || strings.HasPrefix(c.Request().URL.Path, "/api/")
}

func (c *Context) Param(name string) string {
	if name == "**" {
		return strings.TrimPrefix(c.Context.Param("path"), "/")
	}
	return c.Context.Param(name)
}

func (c *Context) Query(name string, defaultVal ...string) string {
	value := strings.TrimSpace(c.Context.Query(name))
	if value != "" {
		return value
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return ""
}

func (c *Context) QueryInt(name string, defaultVal ...int) int {
	value := c.Query(name)
	if value == "" && len(defaultVal) > 0 {
		return defaultVal[0]
	}

	number, _ := strconv.Atoi(value)
	return number
}

func (c *Context) Redirect(location string, status ...int) {
	code := http.StatusFound
	if len(status) == 1 {
		code = status[0]
	}
	c.Context.Redirect(code, location)
}

func (c *Context) BindJSON(target interface{}) error {
	return c.Context.ShouldBindJSON(target)
}

func (c *Context) JSON(data interface{}) error {
	c.Context.JSON(http.StatusOK, gin.H{
		"code":    0,
		"data":    data,
		"message": "success",
	})
	return ErrHandled
}

func (c *Context) ServerError() error {
	return c.JSONError(50000, "服务器内部错误")
}

func (c *Context) JSONError(errorCode int, message string) error {
	span := trace.SpanFromContext(c.Request().Context())

	statusCode := errorCode / 100
	if statusCode < 100 || statusCode > 999 {
		statusCode = http.StatusInternalServerError
	}

	if !c.Writer.Written() {
		c.Context.JSON(statusCode, gin.H{
			"code":     errorCode,
			"message":  message,
			"trace_id": span.SpanContext().TraceID().String(),
		})
	}

	return ErrHandled
}

func Contexter() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := FromGin(c)
		ctx.endpoint = EndpointWeb
		ctx.User = authenticatedUser(c)
		ctx.IsLogged = ctx.User != nil

		var userID uint
		if ctx.User != nil {
			userID = ctx.User.ID
		}

		span := trace.SpanFromContext(c.Request.Context())
		if span.IsRecording() {
			span.SetAttributes(
				attribute.Bool(branding.TelemetryNamespace+".user.is-login", ctx.IsLogged),
				attribute.Int(branding.TelemetryNamespace+".user.id", int(userID)),
			)
		}

		c.Writer.Header().Set("Trace-ID", span.SpanContext().TraceID().String())
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "DENY")

		if config.App.MaintenanceMode {
			c.Writer.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			if strings.HasPrefix(c.Request.URL.Path, "/api/") {
				_ = ctx.JSONError(http.StatusServiceUnavailable, "服务维护中")
			} else {
				c.Data(http.StatusServiceUnavailable, "text/plain; charset=utf-8", []byte(branding.ProductName+" is under maintenance"))
			}
			c.Abort()
			return
		}

		c.Next()
	}
}
