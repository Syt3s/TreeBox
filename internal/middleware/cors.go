package middleware

import (
	"net/http"

	"github.com/flamego/flamego"
)

func CORS() flamego.Handler {
	return func(ctx flamego.Context) {
		origin := ctx.Request().Header.Get("Origin")
		if origin != "" {
			ctx.ResponseWriter().Header().Set("Access-Control-Allow-Origin", origin)
		}
		ctx.ResponseWriter().Header().Add("Vary", "Origin")
		ctx.ResponseWriter().Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		ctx.ResponseWriter().Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
		ctx.ResponseWriter().Header().Set("Access-Control-Allow-Credentials", "true")
		ctx.ResponseWriter().Header().Set("Access-Control-Max-Age", "86400")

		if ctx.Request().Method == http.MethodOptions {
			ctx.ResponseWriter().WriteHeader(http.StatusOK)
			return
		}

		ctx.Next()
	}
}
