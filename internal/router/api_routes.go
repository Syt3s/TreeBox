package router

import (
	"github.com/gin-gonic/gin"

	"github.com/syt3s/TreeBox/handler/api"
	"github.com/syt3s/TreeBox/internal/context"
)

func registerAPIRoutes(r *gin.Engine, authRequired gin.HandlerFunc) {
	apiV2 := r.Group("/api/v2")
	apiV2.Use(context.APIEndpoint())
	apiV2.POST("/auth/login", context.Wrap(api.Login))
	apiV2.POST("/auth/register", context.Wrap(api.Register))
	apiV2.POST("/auth/logout", context.Wrap(api.Logout))
	apiV2.GET("/auth/me", authRequired, context.Wrap(api.GetCurrentUser))
	apiV2.POST("/auth/reset-password-dev", context.Wrap(api.AdminResetPassword))
	apiV2.GET("/users/:domain", context.Wrap(api.GetUser))

	userGroup := apiV2.Group("/user")
	userGroup.GET("/questions", authRequired, context.Wrap(api.GetUserQuestions))
	userGroup.POST("/profile", authRequired, context.Wrap(api.UpdateProfile))
	userGroup.POST("/harassment", authRequired, context.Wrap(api.UpdateHarassment))
	userGroup.GET("/export", authRequired, context.Wrap(api.ExportData))
	userGroup.POST("/deactivate", authRequired, context.Wrap(api.Deactivate))

	questionGroup := apiV2.Group("/questions")
	questionGroup.POST("/:domain", context.Wrap(api.CreateQuestion))
	questionGroup.GET("/:domain", context.Wrap(api.GetQuestions))
	questionGroup.GET("/:domain/:questionID", context.Wrap(api.GetQuestion))
	questionGroup.POST("/:domain/:questionID/answer", authRequired, context.Wrap(api.AnswerQuestion))
	questionGroup.POST("/:domain/:questionID/delete", authRequired, context.Wrap(api.DeleteQuestion))
	questionGroup.POST("/:domain/:questionID/private", authRequired, context.Wrap(api.SetQuestionPrivate))
	questionGroup.POST("/:domain/:questionID/public", authRequired, context.Wrap(api.SetQuestionPublic))
}
