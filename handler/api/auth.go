package api

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/syt3s/TreeBox/internal/conf"
	"github.com/syt3s/TreeBox/internal/context"
	"github.com/syt3s/TreeBox/internal/db"
	"github.com/syt3s/TreeBox/internal/dbutil"
	"github.com/syt3s/TreeBox/internal/form"
	"github.com/syt3s/TreeBox/internal/logging"
	"github.com/syt3s/TreeBox/internal/security"
)

type LoginRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	Recaptcha string `json:"recaptcha"`
}

type LoginResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message,omitempty"`
	User    *db.User `json:"user,omitempty"`
	Token   string   `json:"token,omitempty"`
}

func Login(ctx context.Context) error {
	var req LoginRequest
	if err := form.BindJSON(ctx, &req); err != nil {
		return err
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.login"),
	)

	if conf.App.Production {
		resp, err := security.VerifyRecaptcha(ctx.Request().Context(), req.Recaptcha, ctx.Request().RemoteAddr)
		if err != nil {
			logger.Error("failed to verify recaptcha", zap.Error(err))
			return ctx.JSONError(50000, "验证码校验失败")
		}
		if !resp.Success {
			return ctx.JSONError(40000, "验证码错误")
		}
	}

	user, err := db.Users.Authenticate(ctx.Request().Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, db.ErrBadCredential) {
			return ctx.JSONError(40100, "邮箱或密码错误")
		}
		logger.Error("failed to authenticate user", zap.Error(err))
		return ctx.JSONError(50000, "登录失败")
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
	Success bool     `json:"success"`
	Message string   `json:"message,omitempty"`
	User    *db.User `json:"user,omitempty"`
	Token   string   `json:"token,omitempty"`
}

type AdminResetPasswordRequest struct {
	Email       string `json:"email"`
	NewPassword string `json:"new_password"`
}

func Register(ctx context.Context) error {
	var req RegisterRequest
	if err := form.BindJSON(ctx, &req); err != nil {
		return err
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.register"),
	)

	if conf.App.Production {
		resp, err := security.VerifyRecaptcha(ctx.Request().Context(), req.Recaptcha, ctx.Request().RemoteAddr)
		if err != nil {
			logger.Error("failed to verify recaptcha", zap.Error(err))
			return ctx.JSONError(50000, "验证码校验失败")
		}
		if !resp.Success {
			return ctx.JSONError(40000, "验证码错误")
		}
	}

	if err := db.Users.Create(ctx.Request().Context(), db.CreateUserOptions{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Domain:   req.Domain,
	}); err != nil {
		if errors.Is(err, db.ErrDuplicateEmail) {
			return ctx.JSONError(40900, "这个邮箱已经注册过账号了")
		}
		if errors.Is(err, db.ErrDuplicateDomain) {
			return ctx.JSONError(40900, "个性域名已被占用，请换一个")
		}
		logger.Error("failed to create user", zap.Error(err))
		return ctx.JSONError(50000, "注册失败")
	}

	user, err := db.Users.GetByEmail(ctx.Request().Context(), req.Email)
	if err != nil {
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

func Logout(ctx context.Context) error {
	security.ClearAuthTokenCookie(ctx.ResponseWriter())
	return ctx.JSON(map[string]bool{
		"success": true,
	})
}

func GetCurrentUser(ctx context.Context) error {
	if !ctx.IsLogged {
		return ctx.JSONError(40100, "请先登录")
	}
	return ctx.JSON(ctx.User)
}

func AdminResetPassword(ctx context.Context) error {
	var req AdminResetPasswordRequest
	if err := form.BindJSON(ctx, &req); err != nil {
		return err
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.admin_reset_password"),
	)

	if conf.App.Production {
		return ctx.JSONError(40300, "生产环境不可用")
	}

	u, err := db.Users.GetByEmail(ctx.Request().Context(), req.Email)
	if err != nil {
		return ctx.JSONError(40400, "用户不存在")
	}

	if err := db.Users.UpdatePassword(ctx.Request().Context(), u.ID, req.NewPassword); err != nil {
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
	Success bool     `json:"success"`
	Message string   `json:"message,omitempty"`
	User    *db.User `json:"user,omitempty"`
}

func UpdateProfile(ctx context.Context) error {
	var req UpdateProfileRequest
	if err := form.BindJSON(ctx, &req); err != nil {
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
		if err := db.Users.ChangePassword(ctx.Request().Context(), ctx.User.ID, req.OldPassword, req.NewPassword); err != nil {
			if errors.Is(err, db.ErrBadCredential) {
				return ctx.JSONError(40100, "当前密码错误")
			}
			logger.Error("failed to change password", zap.Error(err))
			return ctx.JSONError(50000, "修改密码失败")
		}
	}

	notify := db.NotifyTypeNone
	if req.NotifyEmail {
		notify = db.NotifyTypeEmail
	}

	if err := db.Users.Update(ctx.Request().Context(), ctx.User.ID, db.UpdateUserOptions{
		Name:       name,
		Avatar:     ctx.User.Avatar,
		Background: ctx.User.Background,
		Intro:      req.Intro,
		Notify:     notify,
	}); err != nil {
		logger.Error("failed to update user profile", zap.Error(err))
		return ctx.JSONError(50000, "更新失败")
	}

	u, err := db.Users.GetByID(ctx.Request().Context(), ctx.User.ID)
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
	Success bool     `json:"success"`
	Message string   `json:"message,omitempty"`
	User    *db.User `json:"user,omitempty"`
}

func UpdateHarassment(ctx context.Context) error {
	var req UpdateHarassmentRequest
	if err := form.BindJSON(ctx, &req); err != nil {
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

	setting := db.HarassmentSettingNone
	if req.RegisterOnly {
		setting = db.HarassmentSettingTypeRegisterOnly
	}

	if err := db.Users.UpdateHarassmentSetting(ctx.Request().Context(), ctx.User.ID, db.HarassmentSettingOptions{
		Type:       setting,
		BlockWords: blockWords,
	}); err != nil {
		logger.Error("failed to update harassment setting", zap.Error(err))
		return ctx.JSONError(50000, "更新失败")
	}

	u, err := db.Users.GetByID(ctx.Request().Context(), ctx.User.ID)
	if err != nil {
		return ctx.JSONError(50000, "获取用户信息失败")
	}

	return ctx.JSON(UpdateHarassmentResponse{
		Success: true,
		Message: "更新成功",
		User:    u,
	})
}

func GetUserQuestions(ctx context.Context) error {
	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.get_user_questions"),
		zap.Uint("user_id", ctx.User.ID),
	)

	pageSize, _ := strconv.Atoi(ctx.Query("page_size"))
	cursor := ctx.Query("cursor")

	questions, err := db.Questions.GetByUserID(ctx.Request().Context(), ctx.User.ID, db.GetQuestionsByUserIDOptions{
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

type ExportDataResponse struct {
	Success   bool           `json:"success"`
	User      *db.User       `json:"user,omitempty"`
	Questions []*db.Question `json:"questions,omitempty"`
}

func ExportData(ctx context.Context) error {
	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.export_data"),
		zap.Uint("user_id", ctx.User.ID),
	)

	questions, err := db.Questions.GetByUserID(ctx.Request().Context(), ctx.User.ID, db.GetQuestionsByUserIDOptions{
		Cursor:         nil,
		FilterAnswered: false,
		ShowPrivate:    true,
	})
	if err != nil {
		logger.Error("failed to export questions", zap.Error(err))
		return ctx.JSONError(50000, "导出失败")
	}

	u, err := db.Users.GetByID(ctx.Request().Context(), ctx.User.ID)
	if err != nil {
		return ctx.JSONError(50000, "导出失败")
	}

	return ctx.JSON(ExportDataResponse{
		Success:   true,
		User:      u,
		Questions: questions,
	})
}

func Deactivate(ctx context.Context) error {
	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.deactivate"),
		zap.Uint("user_id", ctx.User.ID),
	)

	if err := db.Users.Deactivate(ctx.Request().Context(), ctx.User.ID); err != nil {
		logger.Error("failed to deactivate user", zap.Error(err))
		return ctx.JSONError(50000, "停用失败")
	}

	security.ClearAuthTokenCookie(ctx.ResponseWriter())
	return ctx.JSON(map[string]bool{"success": true})
}
