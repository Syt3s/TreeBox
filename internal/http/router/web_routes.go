package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/syt3s/TreeBox/internal/controller"
	"github.com/syt3s/TreeBox/internal/controller/pixel"
	"github.com/syt3s/TreeBox/internal/controller/service"
	"github.com/syt3s/TreeBox/internal/http/appctx"
)

func registerWebRoutes(r *gin.Engine, authRequired gin.HandlerFunc) {
	r.GET("/", appctx.Wrap(controller.Home))
	r.Any("/pixel/*path", authRequired, appctx.Wrap(pixel.Proxy))
	r.GET("/robots.txt", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte("User-agent: *\nDisallow: /_/"))
	})
	r.Any("/service/*path", appctx.Wrap(service.Proxy))
	r.NoRoute(func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/")
	})
}
