package router

import (
	"github.com/gin-gonic/gin"

	"github.com/syt3s/TreeBox/internal/config"
	"github.com/syt3s/TreeBox/internal/http/appctx"
	"github.com/syt3s/TreeBox/internal/http/middleware"
)

func New(middlewares ...gin.HandlerFunc) *gin.Engine {
	if config.App.Production {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middlewares...)
	r.Use(middleware.Recovery(), middleware.CORS(), appctx.Contexter(), middleware.ErrorHandler())

	authRequired := appctx.Toggle(&appctx.ToggleOptions{UserSignInRequired: true})
	registerWebRoutes(r, authRequired)
	registerAPIRoutes(r, authRequired)

	return r
}
