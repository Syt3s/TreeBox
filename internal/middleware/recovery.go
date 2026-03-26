package middleware

import (
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	appcontext "github.com/syt3s/TreeBox/internal/context"
	"github.com/syt3s/TreeBox/internal/logging"
)

func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logging.FromContext(c.Request.Context()).With(
			zap.String("component", "http.recovery"),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
		).Error("panic recovered",
			zap.Any("panic", recovered),
			zap.ByteString("stack", debug.Stack()),
		)

		if c.Writer.Written() {
			c.Abort()
			return
		}

		writeInternalError(c, appcontext.FromGin(c))
		c.Abort()
	})
}
