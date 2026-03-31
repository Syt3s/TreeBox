package api

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/syt3s/TreeBox/internal/config"
	"github.com/syt3s/TreeBox/internal/dbutil"
	"github.com/syt3s/TreeBox/internal/http/appctx"
	"github.com/syt3s/TreeBox/internal/http/request"
	"github.com/syt3s/TreeBox/internal/logging"
	"github.com/syt3s/TreeBox/internal/model"
	"github.com/syt3s/TreeBox/internal/repository"
	"github.com/syt3s/TreeBox/internal/security"
)

type LoginRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	Recaptcha string `json:"recaptcha"`
}

type LoginResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	User    *model.User `json:"user,omitempty"`
	Token   string      `json:"token,omitempty"`
}

func Login(ctx appctx.Context) error {
	var req LoginRequest
	if err := request.BindJSON(ctx, &req); err != nil {
		return err
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.login"),
	)

	if err := verifyRecaptchaIfNeeded(ctx, logger, req.Recaptcha); err != nil {
		return err
	}

	user, err := repository.Users.Authenticate(ctx.Request().Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, repository.ErrBadCredential) {
			return ctx.JSONError(40100, "邮箱或密码错误")
		}
		logger.Error("failed to authenticate user", zap.Error(err))
		return ctx.JSONError(50000, "登录失败")
	}

	if _, err := repository.Users.EnsureTenantBootstrap(ctx.Request().Context(), user.ID); err != nil {
		logger.Error("failed to ensure tenant bootstrap after login", zap.Error(err), zap.Uint("user_id", user.ID))
		return ctx.JSONError(50000, "登录失败，请稍后重试")
	}

	token, err := security.GenerateToken(user.ID, 30*24*time.Hour)
	if err != nil {
		logger.Error("failed to generate auth token", zap.Error(err), zap.Uint("user_id", user.ID))
		return ctx.JSONError(50000, "登录失败，请稍后重试")
	}

	security.SetAuthTokenCookie(ctx.ResponseWriter(), token, 30*24*time.Hour)
	return ctx.JSON(LoginResponse{
		Success: true,
		Message: "登录成功",
		User:    user,
		Token:   token,
	})
}

type RegisterRequest struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Domain    string `json:"domain"`
	Recaptcha string `json:"recaptcha"`
}

type RegisterResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	User    *model.User `json:"user,omitempty"`
	Token   string      `json:"token,omitempty"`
}

type AdminResetPasswordRequest struct {
	Email       string `json:"email"`
	NewPassword string `json:"new_password"`
}

func Register(ctx appctx.Context) error {
	var req RegisterRequest
	if err := request.BindJSON(ctx, &req); err != nil {
		return err
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.register"),
	)

	if err := verifyRecaptchaIfNeeded(ctx, logger, req.Recaptcha); err != nil {
		return err
	}

	result, err := repository.Users.Register(ctx.Request().Context(), repository.RegisterUserOptions{
		CreateUserOptions: repository.CreateUserOptions{
			Name:       req.Name,
			Email:      req.Email,
			Password:   req.Password,
			Domain:     req.Domain,
			Avatar:     config.Upload.DefaultAvatar,
			Background: config.Upload.DefaultBackground,
		},
		TenantName:    req.Name + " team",
		WorkspaceName: "Default workspace",
	})
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateEmail) {
			return ctx.JSONError(40900, "这个邮箱已经注册过账号了")
		}
		if errors.Is(err, repository.ErrDuplicateDomain) {
			return ctx.JSONError(40900, "个性域名已被占用，请换一个")
		}
		logger.Error("failed to create user", zap.Error(err))
		return ctx.JSONError(50000, "注册失败")
	}

	user := result.User
	if user == nil {
		logger.Error("failed to load user after registration", zap.Error(err))
		return ctx.JSONError(50000, "注册成功，但获取用户信息失败")
	}

	token, err := security.GenerateToken(user.ID, 30*24*time.Hour)
	if err != nil {
		logger.Error("failed to generate auth token", zap.Error(err), zap.Uint("user_id", user.ID))
		return ctx.JSONError(50000, "注册成功，但登录状态创建失败")
	}

	security.SetAuthTokenCookie(ctx.ResponseWriter(), token, 30*24*time.Hour)
	return ctx.JSON(RegisterResponse{
		Success: true,
		Message: "注册成功",
		User:    user,
		Token:   token,
	})
}

func Logout(ctx appctx.Context) error {
	security.ClearAuthTokenCookie(ctx.ResponseWriter())
	return ctx.JSON(map[string]bool{
		"success": true,
	})
}

func GetCurrentUser(ctx appctx.Context) error {
	if !ctx.IsLogged {
		return ctx.JSONError(40100, "请先登录")
	}
	return ctx.JSON(ctx.User)
}

func AdminResetPassword(ctx appctx.Context) error {
	var req AdminResetPasswordRequest
	if err := request.BindJSON(ctx, &req); err != nil {
		return err
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.admin_reset_password"),
	)

	if config.App.Production {
		return ctx.JSONError(40300, "生产环境不可用")
	}

	u, err := repository.Users.GetByEmail(ctx.Request().Context(), req.Email)
	if err != nil {
		return ctx.JSONError(40400, "用户不存在")
	}

	if err := repository.Users.UpdatePassword(ctx.Request().Context(), u.ID, req.NewPassword); err != nil {
		logger.Error("failed to reset password", zap.Error(err), zap.Uint("user_id", u.ID))
		return ctx.JSONError(50000, "重置失败")
	}

	return ctx.JSON(map[string]bool{"success": true})
}

type UpdateProfileRequest struct {
	Name        string `json:"name"`
	Intro       string `json:"intro"`
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
	NotifyEmail bool   `json:"notify_email"`
}

type UpdateProfileResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	User    *model.User `json:"user,omitempty"`
}

func UpdateProfile(ctx appctx.Context) error {
	var req UpdateProfileRequest
	if err := request.BindJSON(ctx, &req); err != nil {
		return err
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.update_profile"),
		zap.Uint("user_id", ctx.User.ID),
	)

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return ctx.JSONError(40000, "昵称不能为空")
	}

	if strings.TrimSpace(req.NewPassword) != "" {
		if strings.TrimSpace(req.OldPassword) == "" {
			return ctx.JSONError(40000, "请输入当前密码")
		}
		if err := repository.Users.ChangePassword(ctx.Request().Context(), ctx.User.ID, req.OldPassword, req.NewPassword); err != nil {
			if errors.Is(err, repository.ErrBadCredential) {
				return ctx.JSONError(40100, "当前密码错误")
			}
			logger.Error("failed to change password", zap.Error(err))
			return ctx.JSONError(50000, "修改密码失败")
		}
	}

	notify := model.NotifyTypeNone
	if req.NotifyEmail {
		notify = model.NotifyTypeEmail
	}

	if err := repository.Users.Update(ctx.Request().Context(), ctx.User.ID, repository.UpdateUserOptions{
		Name:       name,
		Avatar:     ctx.User.Avatar,
		Background: ctx.User.Background,
		Intro:      req.Intro,
		Notify:     notify,
	}); err != nil {
		logger.Error("failed to update user profile", zap.Error(err))
		return ctx.JSONError(50000, "更新失败")
	}

	u, err := repository.Users.GetByID(ctx.Request().Context(), ctx.User.ID)
	if err != nil {
		return ctx.JSONError(50000, "获取用户信息失败")
	}

	return ctx.JSON(UpdateProfileResponse{
		Success: true,
		Message: "更新成功",
		User:    u,
	})
}

type UpdateHarassmentRequest struct {
	RegisterOnly bool   `json:"register_only"`
	BlockWords   string `json:"block_words"`
}

type UpdateHarassmentResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	User    *model.User `json:"user,omitempty"`
}

func UpdateHarassment(ctx appctx.Context) error {
	var req UpdateHarassmentRequest
	if err := request.BindJSON(ctx, &req); err != nil {
		return err
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.update_harassment"),
		zap.Uint("user_id", ctx.User.ID),
	)

	words := []string{}
	for _, w := range strings.Split(req.BlockWords, ",") {
		w = strings.TrimSpace(w)
		if w == "" {
			continue
		}
		if len([]rune(w)) > 10 {
			return ctx.JSONError(40000, "屏蔽词长度不能超过 10 个字符")
		}
		words = append(words, w)
	}
	if len(words) > 10 {
		return ctx.JSONError(40000, "最多支持 10 个屏蔽词")
	}
	blockWords := strings.Join(words, ",")

	setting := model.HarassmentSettingNone
	if req.RegisterOnly {
		setting = model.HarassmentSettingTypeRegisterOnly
	}

	if err := repository.Users.UpdateHarassmentSetting(ctx.Request().Context(), ctx.User.ID, repository.HarassmentSettingOptions{
		Type:       setting,
		BlockWords: blockWords,
	}); err != nil {
		logger.Error("failed to update harassment setting", zap.Error(err))
		return ctx.JSONError(50000, "更新失败")
	}

	u, err := repository.Users.GetByID(ctx.Request().Context(), ctx.User.ID)
	if err != nil {
		return ctx.JSONError(50000, "获取用户信息失败")
	}

	return ctx.JSON(UpdateHarassmentResponse{
		Success: true,
		Message: "更新成功",
		User:    u,
	})
}

func GetUserQuestions(ctx appctx.Context) error {
	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.get_user_questions"),
		zap.Uint("user_id", ctx.User.ID),
	)

	pageSize, _ := strconv.Atoi(ctx.Query("page_size"))
	cursor := ctx.Query("cursor")

	questions, err := repository.Questions.GetByUserID(ctx.Request().Context(), ctx.User.ID, repository.GetQuestionsByUserIDOptions{
		Cursor: &dbutil.Cursor{
			Value:    cursor,
			PageSize: pageSize,
		},
		FilterAnswered: false,
		ShowPrivate:    true,
	})
	if err != nil {
		logger.Error("failed to get user questions", zap.Error(err))
		return ctx.JSONError(50000, "获取问题列表失败")
	}

	nextCursor := ""
	if len(questions) > 0 {
		nextCursor = fmt.Sprintf("%d", questions[len(questions)-1].ID)
	}

	return ctx.JSON(GetQuestionsResponse{
		Success:    true,
		Questions:  questions,
		NextCursor: nextCursor,
	})
}

type GetUserQuestionStatsResponse struct {
	Success       bool  `json:"success"`
	TotalCount    int64 `json:"total_count"`
	AnsweredCount int64 `json:"answered_count"`
	UnreadCount   int64 `json:"unread_count"`
	PendingCount  int64 `json:"pending_count"`
}

func GetUserQuestionStats(ctx appctx.Context) error {
	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.get_user_question_stats"),
		zap.Uint("user_id", ctx.User.ID),
	)

	totalCount, err := repository.Questions.Count(ctx.Request().Context(), ctx.User.ID, repository.GetQuestionsCountOptions{
		ShowPrivate: true,
	})
	if err != nil {
		logger.Error("failed to count total questions", zap.Error(err))
		return ctx.JSONError(50000, "获取问题统计失败")
	}

	answeredCount, err := repository.Questions.Count(ctx.Request().Context(), ctx.User.ID, repository.GetQuestionsCountOptions{
		FilterAnswered: true,
		ShowPrivate:    true,
	})
	if err != nil {
		logger.Error("failed to count answered questions", zap.Error(err))
		return ctx.JSONError(50000, "获取问题统计失败")
	}

	unreadCount, err := repository.Questions.CountUnread(ctx.Request().Context(), ctx.User.ID, true)
	if err != nil {
		logger.Error("failed to count unread questions", zap.Error(err))
		return ctx.JSONError(50000, "获取问题统计失败")
	}

	return ctx.JSON(GetUserQuestionStatsResponse{
		Success:       true,
		TotalCount:    totalCount,
		AnsweredCount: answeredCount,
		UnreadCount:   unreadCount,
		PendingCount:  unreadCount,
	})
}

type MarkUserQuestionViewedResponse struct {
	Success  bool       `json:"success"`
	ViewedAt *time.Time `json:"viewed_at,omitempty"`
}

type MarkAllUserQuestionsViewedResponse struct {
	Success     bool       `json:"success"`
	ViewedAt    *time.Time `json:"viewed_at,omitempty"`
	ViewedCount int64      `json:"viewed_count"`
}

func MarkUserQuestionViewed(ctx appctx.Context) error {
	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.mark_user_question_viewed"),
		zap.Uint("user_id", ctx.User.ID),
	)

	questionID, err := strconv.ParseUint(strings.TrimSpace(ctx.Param("questionID")), 10, 64)
	if err != nil {
		return ctx.JSONError(40000, "问题编号无效")
	}

	question, err := repository.Questions.GetByID(ctx.Request().Context(), uint(questionID))
	if err != nil {
		if errors.Is(err, repository.ErrQuestionNotExist) {
			return ctx.JSONError(40400, "问题不存在")
		}
		return ctx.JSONError(50000, "获取问题失败")
	}

	if question.UserID != ctx.User.ID {
		return ctx.JSONError(40300, "无权操作该问题")
	}

	if question.ViewedAt == nil {
		viewedAt := time.Now()
		if err := repository.Questions.MarkViewed(ctx.Request().Context(), question.ID, viewedAt); err != nil {
			logger.Error("failed to mark question viewed", zap.Error(err), zap.Uint("question_id", question.ID))
			return ctx.JSONError(50000, "标记已查看失败")
		}
		question.ViewedAt = &viewedAt
	}

	return ctx.JSON(MarkUserQuestionViewedResponse{
		Success:  true,
		ViewedAt: question.ViewedAt,
	})
}

func MarkAllUserQuestionsViewed(ctx appctx.Context) error {
	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.mark_all_user_questions_viewed"),
		zap.Uint("user_id", ctx.User.ID),
	)

	viewedAt := time.Now()
	viewedCount, err := repository.Questions.MarkAllViewed(ctx.Request().Context(), ctx.User.ID, viewedAt)
	if err != nil {
		logger.Error("failed to mark all questions viewed", zap.Error(err))
		return ctx.JSONError(50000, "标记全部问题已查看失败")
	}

	var responseViewedAt *time.Time
	if viewedCount > 0 {
		responseViewedAt = &viewedAt
	}

	return ctx.JSON(MarkAllUserQuestionsViewedResponse{
		Success:     true,
		ViewedAt:    responseViewedAt,
		ViewedCount: viewedCount,
	})
}

type ExportDataResponse struct {
	Success   bool              `json:"success"`
	User      *model.User       `json:"user,omitempty"`
	Questions []*model.Question `json:"questions,omitempty"`
}

func ExportData(ctx appctx.Context) error {
	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.export_data"),
		zap.Uint("user_id", ctx.User.ID),
	)

	questions, err := repository.Questions.GetByUserID(ctx.Request().Context(), ctx.User.ID, repository.GetQuestionsByUserIDOptions{
		Cursor:         nil,
		FilterAnswered: false,
		ShowPrivate:    true,
	})
	if err != nil {
		logger.Error("failed to export questions", zap.Error(err))
		return ctx.JSONError(50000, "导出失败")
	}

	u, err := repository.Users.GetByID(ctx.Request().Context(), ctx.User.ID)
	if err != nil {
		return ctx.JSONError(50000, "导出失败")
	}

	return ctx.JSON(ExportDataResponse{
		Success:   true,
		User:      u,
		Questions: questions,
	})
}

func Deactivate(ctx appctx.Context) error {
	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.deactivate"),
		zap.Uint("user_id", ctx.User.ID),
	)

	if err := repository.Users.Deactivate(ctx.Request().Context(), ctx.User.ID); err != nil {
		logger.Error("failed to deactivate user", zap.Error(err))
		return ctx.JSONError(50000, "停用失败")
	}

	security.ClearAuthTokenCookie(ctx.ResponseWriter())
	return ctx.JSON(map[string]bool{"success": true})
}
