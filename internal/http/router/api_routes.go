package router

import (
	"github.com/gin-gonic/gin"

	"github.com/syt3s/TreeBox/internal/controller/api"
	"github.com/syt3s/TreeBox/internal/http/appctx"
)

func registerAPIRoutes(r *gin.Engine, authRequired gin.HandlerFunc) {
	apiV2 := r.Group("/api/v2")
	apiV2.Use(appctx.APIEndpoint())
	apiV2.POST("/auth/login", appctx.Wrap(api.Login))
	apiV2.POST("/auth/register", appctx.Wrap(api.Register))
	apiV2.POST("/auth/logout", appctx.Wrap(api.Logout))
	apiV2.GET("/auth/me", authRequired, appctx.Wrap(api.GetCurrentUser))
	apiV2.POST("/auth/reset-password-dev", appctx.Wrap(api.AdminResetPassword))
	apiV2.GET("/users/:domain", appctx.Wrap(api.GetUser))

	userGroup := apiV2.Group("/user")
	userGroup.GET("/questions", authRequired, appctx.Wrap(api.GetUserQuestions))
	userGroup.GET("/questions/stats", authRequired, appctx.Wrap(api.GetUserQuestionStats))
	userGroup.POST("/questions/:questionID/viewed", authRequired, appctx.Wrap(api.MarkUserQuestionViewed))
	userGroup.POST("/profile", authRequired, appctx.Wrap(api.UpdateProfile))
	userGroup.POST("/avatar", authRequired, appctx.Wrap(api.UploadAvatar))
	userGroup.POST("/background", authRequired, appctx.Wrap(api.UploadBackground))
	userGroup.POST("/harassment", authRequired, appctx.Wrap(api.UpdateHarassment))
	userGroup.GET("/export", authRequired, appctx.Wrap(api.ExportData))
	userGroup.POST("/deactivate", authRequired, appctx.Wrap(api.Deactivate))

	questionGroup := apiV2.Group("/questions")
	questionGroup.POST("/:domain", appctx.Wrap(api.CreateQuestion))
	questionGroup.GET("/:domain", appctx.Wrap(api.GetQuestions))
	questionGroup.GET("/:domain/:questionID", appctx.Wrap(api.GetQuestion))
	questionGroup.POST("/:domain/:questionID/answer", authRequired, appctx.Wrap(api.AnswerQuestion))
	questionGroup.POST("/:domain/:questionID/delete", authRequired, appctx.Wrap(api.DeleteQuestion))
	questionGroup.POST("/:domain/:questionID/private", authRequired, appctx.Wrap(api.SetQuestionPrivate))
	questionGroup.POST("/:domain/:questionID/public", authRequired, appctx.Wrap(api.SetQuestionPublic))
}
