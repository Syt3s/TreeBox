package context

import (
	"strings"

	"github.com/flamego/flamego"
	"github.com/flamego/session"

	"github.com/syt3s/TreeBox/internal/db"
	"github.com/syt3s/TreeBox/internal/security"
)

// authenticatedUser returns the user object of the authenticated user.
func authenticatedUser(ctx flamego.Context, sess session.Session) *db.User {
	authz := ctx.Request().Header.Get("Authorization")
	if strings.HasPrefix(authz, "Bearer ") {
		token := strings.TrimSpace(strings.TrimPrefix(authz, "Bearer "))
		if token != "" {
			if claims, err := security.ParseToken(token); err == nil {
				if user, err := db.Users.GetByID(ctx.Request().Context(), claims.UID); err == nil {
					return user
				}
			}
		}
	}

	if cookie, err := ctx.Request().Cookie(security.AuthTokenCookieName); err == nil {
		token := strings.TrimSpace(cookie.Value)
		if token != "" {
			if claims, err := security.ParseToken(token); err == nil {
				if user, err := db.Users.GetByID(ctx.Request().Context(), claims.UID); err == nil {
					return user
				}
			}
		}
	}

	uid, ok := sess.Get("uid").(uint)
	if !ok {
		return nil
	}

	user, _ := db.Users.GetByID(ctx.Request().Context(), uid)
	return user
}

type ToggleOptions struct {
	UserSignInRequired  bool
	UserSignOutRequired bool
}

func Toggle(options *ToggleOptions) flamego.Handler {
	return func(ctx Context, endpoint EndpointType) error {
		if endpoint.IsWeb() && strings.HasPrefix(ctx.Request().URL.Path, "/api/") {
			endpoint = EndpointAPI
		}

		if options.UserSignOutRequired && ctx.IsLogged {
			ctx.Redirect("/")
			return nil
		}

		if options.UserSignInRequired && !ctx.IsLogged {
			if endpoint.IsAPI() {
				return ctx.JSONError(40100, "请先登录")
			}
			ctx.SetErrorFlash("请先登录")
			ctx.Redirect("/login")
			return nil
		}
		return nil
	}
}
