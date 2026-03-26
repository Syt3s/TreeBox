package context

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/syt3s/TreeBox/internal/branding"
	"github.com/syt3s/TreeBox/internal/conf"
	"github.com/syt3s/TreeBox/internal/db"
	"github.com/syt3s/TreeBox/internal/security"
)

func authenticatedUser(c *gin.Context) *db.User {
	authz := c.Request.Header.Get("Authorization")
	if strings.HasPrefix(authz, "Bearer ") {
		token := strings.TrimSpace(strings.TrimPrefix(authz, "Bearer "))
		if token != "" {
			if claims, err := security.ParseToken(token); err == nil {
				if user, err := db.Users.GetByID(c.Request.Context(), claims.UID); err == nil {
					return user
				}
			}
		}
	}

	for _, cookieName := range []string{security.AuthTokenCookieName, branding.LegacyAuthTokenCookieName} {
		if cookie, err := c.Request.Cookie(cookieName); err == nil {
			token := strings.TrimSpace(cookie.Value)
			if token != "" {
				if claims, err := security.ParseToken(token); err == nil {
					if user, err := db.Users.GetByID(c.Request.Context(), claims.UID); err == nil {
						return user
					}
				}
			}
		}
	}

	return nil
}

type ToggleOptions struct {
	UserSignInRequired  bool
	UserSignOutRequired bool
}

func Toggle(options *ToggleOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := FromGin(c)
		endpoint := ctx.endpoint
		if endpoint.IsWeb() && strings.HasPrefix(c.Request.URL.Path, "/api/") {
			endpoint = EndpointAPI
		}

		if options.UserSignOutRequired && ctx.IsLogged {
			ctx.Redirect("/")
			c.Abort()
			return
		}

		if options.UserSignInRequired && !ctx.IsLogged {
			if endpoint.IsAPI() {
				_ = ctx.JSONError(40100, "请先登录")
			} else {
				ctx.Redirect(conf.App.ExternalURL + "/login")
			}
			c.Abort()
			return
		}

		c.Next()
	}
}
