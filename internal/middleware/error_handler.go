package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	appcontext "github.com/syt3s/TreeBox/internal/context"
	"github.com/syt3s/TreeBox/internal/logging"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err
		logger := logging.FromContext(c.Request.Context()).With(
			zap.String("component", "http.error"),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Int("error_count", len(c.Errors)),
			zap.Bool("response_written", c.Writer.Written()),
		)

		if c.Writer.Written() {
			logger.Error("request failed after response started", zap.Error(err))
			return
		}

		logger.Error("request failed", zap.Error(err))
		writeInternalError(c, appcontext.FromGin(c))
	}
}

func writeInternalError(c *gin.Context, ctx *appcontext.Context) {
	if ctx.IsAPIRequest() {
		_ = ctx.JSONError(50000, "服务器内部错误")
		return
	}

	c.Data(http.StatusInternalServerError, "text/plain; charset=utf-8", []byte("internal server error"))
}
