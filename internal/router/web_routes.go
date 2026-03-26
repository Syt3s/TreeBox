package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/syt3s/TreeBox/handler"
	"github.com/syt3s/TreeBox/handler/pixel"
	"github.com/syt3s/TreeBox/handler/service"
	"github.com/syt3s/TreeBox/internal/context"
)

func registerWebRoutes(r *gin.Engine, authRequired gin.HandlerFunc) {
	r.GET("/", context.Wrap(handler.Home))
	r.Any("/pixel/*path", authRequired, context.Wrap(pixel.Proxy))
	r.GET("/robots.txt", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte("User-agent: *\nDisallow: /_/"))
	})
	r.Any("/service/*path", context.Wrap(service.Proxy))
	r.NoRoute(func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/")
	})
}
