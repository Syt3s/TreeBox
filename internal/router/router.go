package router

import (
	"github.com/gin-gonic/gin"

	"github.com/syt3s/TreeBox/internal/conf"
	"github.com/syt3s/TreeBox/internal/context"
	"github.com/syt3s/TreeBox/internal/middleware"
)

func New(middlewares ...gin.HandlerFunc) *gin.Engine {
	if conf.App.Production {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middlewares...)
	r.Use(middleware.Recovery(), middleware.CORS(), context.Contexter(), middleware.ErrorHandler())

	authRequired := context.Toggle(&context.ToggleOptions{UserSignInRequired: true})
	registerWebRoutes(r, authRequired)
	registerAPIRoutes(r, authRequired)

	return r
}
