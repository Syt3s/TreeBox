// Copyright 2022 E99p1ant. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package route

import (
	"encoding/gob"
	"io"
	"net/http"
	"time"

	"github.com/flamego/cache"
	cacheRedis "github.com/flamego/cache/redis"
	"github.com/flamego/csrf"
	"github.com/flamego/flamego"
	"github.com/flamego/recaptcha"
	"github.com/flamego/session"
	"github.com/flamego/session/mysql"
	sessionRedis "github.com/flamego/session/redis"
	"github.com/flamego/template"
	"github.com/sirupsen/logrus"

	"github.com/syt3s/TreeBox/internal/conf"
	"github.com/syt3s/TreeBox/internal/context"
	"github.com/syt3s/TreeBox/internal/form"
	"github.com/syt3s/TreeBox/internal/middleware"
	templatepkg "github.com/syt3s/TreeBox/internal/template"
	"github.com/syt3s/TreeBox/route"
	"github.com/syt3s/TreeBox/route/api"
	"github.com/syt3s/TreeBox/route/pixel"
	"github.com/syt3s/TreeBox/route/service"
	"github.com/syt3s/TreeBox/static"
	"github.com/syt3s/TreeBox/templates"
)

func New() *flamego.Flame {
	f := flamego.Classic()
	if conf.App.Production {
		flamego.SetEnv(flamego.EnvTypeProd)
	}

	templateFS, err := template.EmbedFS(templates.FS, ".", []string{".html"})
	if err != nil {
		logrus.WithError(err).Fatal("Failed to embed templates file system")
	}

	// We prefer to save session into database,
	// if no database configuration, the session will be saved into memory instead.
	gob.Register(time.Time{})
	gob.Register(context.Flash{})
	var sessionStorage interface{}
	initer := session.MemoryIniter()
	if conf.Database.DSN != "" {
		initer = mysql.Initer()
		sessionStorage = mysql.Config{
			DSN:      conf.Database.DSN,
			Lifetime: 7 * 24 * time.Hour,
		}
	}
	if conf.Redis.Addr != "" {
		initer = sessionRedis.Initer()
		sessionStorage = sessionRedis.Config{
			Options: &cacheRedis.Options{
				Addr:     conf.Redis.Addr,
				Password: conf.Redis.Password,
				DB:       1,
			},
			Lifetime: 30 * 24 * time.Hour,
		}
	}
	sessioner := session.Sessioner(session.Options{
		Initer: initer,
		Config: sessionStorage,
	})

	f.Use(middleware.CORS())

	f.Use(flamego.Static(flamego.StaticOptions{
		FileSystem: http.FS(static.FS),
		Prefix:     "/static",
	}))

	reqUserSignIn := context.Toggle(&context.ToggleOptions{UserSignInRequired: true})

	f.Group("", func() {
		f.Get("/", route.Home)
		f.Get("/pixel", reqUserSignIn, pixel.Index)
		f.Any("/pixel/{**}", reqUserSignIn, pixel.Proxy)
		f.Get("/robots.txt", func(c context.Context) {
			_, _ = c.ResponseWriter().Write([]byte("User-agent: *\nDisallow: /_/"))
		})
		f.Get("/favicon.ico", func(c context.Context) {
			fs, _ := static.FS.Open("favicon.ico")
			defer func() { _ = fs.Close() }()
			c.ResponseWriter().Header().Set("Content-Type", "image/x-icon")
			_, _ = io.Copy(c.ResponseWriter(), fs)
		})

		f.Any("/service/{**}", service.Proxy)

		f.Group("/api/v2", func() {
			f.Post("/auth/login", form.JSONBind(api.LoginRequest{}), api.Login)
			f.Post("/auth/register", form.JSONBind(api.RegisterRequest{}), api.Register)
			f.Post("/auth/logout", api.Logout)
			f.Get("/auth/me", reqUserSignIn, api.GetCurrentUser)
			f.Post("/auth/reset-password-dev", form.JSONBind(api.AdminResetPasswordRequest{}), api.AdminResetPassword)

			f.Get("/users/{domain}", api.GetUser)

			f.Group("/user", func() {
				f.Get("/questions", reqUserSignIn, api.GetUserQuestions)
				f.Post("/profile", reqUserSignIn, form.JSONBind(api.UpdateProfileRequest{}), api.UpdateProfile)
				f.Post("/harassment", reqUserSignIn, form.JSONBind(api.UpdateHarassmentRequest{}), api.UpdateHarassment)
				f.Get("/export", reqUserSignIn, api.ExportData)
				f.Post("/deactivate", reqUserSignIn, api.Deactivate)
			})

			f.Group("/questions", func() {
				f.Post("/{domain}", form.JSONBind(api.CreateQuestionRequest{}), api.CreateQuestion)
				f.Get("/{domain}", api.GetQuestions)
				f.Get("/{domain}/{questionID}", api.GetQuestion)
				f.Post("/{domain}/{questionID}/answer", reqUserSignIn, form.JSONBind(api.AnswerQuestionRequest{}), api.AnswerQuestion)
				f.Post("/{domain}/{questionID}/delete", reqUserSignIn, api.DeleteQuestion)
				f.Post("/{domain}/{questionID}/private", reqUserSignIn, api.SetQuestionPrivate)
				f.Post("/{domain}/{questionID}/public", reqUserSignIn, api.SetQuestionPublic)
			})
		}, context.APIEndpoint)
	},
		cache.Cacher(cache.Options{
			Initer: cacheRedis.Initer(),
			Config: cacheRedis.Config{
				Options: &cacheRedis.Options{
					Addr:     conf.Redis.Addr,
					Password: conf.Redis.Password,
					DB:       0,
				},
			},
		}),
		recaptcha.V3(
			recaptcha.Options{
				Secret: conf.Recaptcha.ServerKey,
				VerifyURL: func() recaptcha.VerifyURL {
					if conf.Recaptcha.TurnstileStyle {
						// FYI: https://developers.cloudflare.com/turnstile/migration/migrating-from-recaptcha/
						return "https://challenges.cloudflare.com/turnstile/v0/siteverify"
					}
					return recaptcha.VerifyURLGlobal
				}(),
			},
		),
		sessioner,
		csrf.Csrfer(csrf.Options{
			Secret: conf.Server.XSRFKey,
			Header: "X-CSRF-Token",
		}),
		template.Templater(template.Options{
			FileSystem: templateFS,
			FuncMaps:   templatepkg.FuncMap(),
		}),
		context.Contexter(),
	)
	f.NotFound(func(ctx flamego.Context) {
		ctx.Redirect("/")
	})

	return f
}
