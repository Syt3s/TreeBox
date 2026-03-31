package api

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/wuhan005/govalid"
	"go.uber.org/zap"

	"github.com/syt3s/TreeBox/internal/dbutil"
	"github.com/syt3s/TreeBox/internal/http/appctx"
	"github.com/syt3s/TreeBox/internal/http/request"
	"github.com/syt3s/TreeBox/internal/logging"
	"github.com/syt3s/TreeBox/internal/model"
	"github.com/syt3s/TreeBox/internal/repository"
)

type CreateQuestionRequest struct {
	Content           string `json:"content"`
	IsPrivate         bool   `json:"is_private"`
	ReceiveReplyEmail string `json:"receive_reply_email"`
	Recaptcha         string `json:"recaptcha"`
}

type CreateQuestionResponse struct {
	Success  bool            `json:"success"`
	Message  string          `json:"message,omitempty"`
	Question *model.Question `json:"question,omitempty"`
}

func parseQuestionRouteParams(ctx appctx.Context) (string, uint, error) {
	domain := strings.TrimSpace(ctx.Param("domain"))
	if domain == "" {
		return "", 0, ctx.JSONError(40000, "用户标识不能为空")
	}

	questionIDParam := strings.TrimSpace(ctx.Param("questionID"))
	if questionIDParam == "" {
		return domain, 0, nil
	}

	questionID, err := strconv.ParseUint(questionIDParam, 10, 64)
	if err != nil {
		return "", 0, ctx.JSONError(40000, "问题编号无效")
	}
	return domain, uint(questionID), nil
}

func CreateQuestion(ctx appctx.Context) error {
	var req CreateQuestionRequest
	if err := request.BindJSON(ctx, &req); err != nil {
		return err
	}

	domain, _, err := parseQuestionRouteParams(ctx)
	if err != nil {
		return err
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.create_question"),
		zap.String("domain", domain),
	)

	pageUser, err := repository.Users.GetByDomain(ctx.Request().Context(), domain)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotExists) {
			return ctx.JSONError(40400, "用户不存在")
		}
		return ctx.JSONError(50000, "获取用户信息失败")
	}

	bootstrap, err := repository.Users.EnsureTenantBootstrap(ctx.Request().Context(), pageUser.ID)
	if err != nil {
		logger.Error("failed to ensure tenant bootstrap for page user", zap.Error(err), zap.Uint("page_user_id", pageUser.ID))
		return ctx.JSONError(50000, "创建问题失败")
	}

	workspace, err := resolveQuestionWorkspace(ctx.Request().Context(), pageUser, bootstrap)
	if err != nil {
		logger.Error("failed to resolve routed workspace", zap.Error(err), zap.Uint("page_user_id", pageUser.ID))
		return ctx.JSONError(50000, "创建问题失败")
	}
	if workspace == nil {
		logger.Error("question routing workspace is unavailable", zap.Uint("page_user_id", pageUser.ID))
		return ctx.JSONError(50000, "创建问题失败")
	}

	if !ctx.IsLogged && pageUser.HarassmentSetting == model.HarassmentSettingTypeRegisterOnly {
		return ctx.JSONError(40100, "该提问箱仅允许注册用户提问，请先登录")
	}

	var receiveReplyEmail string
	if req.ReceiveReplyEmail != "" {
		if errs, ok := govalid.Check(struct {
			Email string `valid:"required;email" label:"邮箱地址"`
		}{
			Email: req.ReceiveReplyEmail,
		}); !ok {
			return ctx.JSONError(40000, errs[0].Error())
		}
		receiveReplyEmail = req.ReceiveReplyEmail
	}

	if err := verifyRecaptchaIfNeeded(ctx, logger, req.Recaptcha); err != nil {
		return err
	}

	content := req.Content
	if len(pageUser.BlockWords) > 0 {
		blockWords := strings.Split(pageUser.BlockWords, ",")
		for _, word := range blockWords {
			if strings.Contains(content, word) {
				return ctx.JSONError(40000, "提问内容包含提问箱主人设置的屏蔽词，发送失败")
			}
		}
	}

	fromIP := ctx.Request().Header.Get("Ali-CDN-Real-IP")
	if fromIP == "" {
		fromIP = ctx.Request().Header.Get("CF-Connecting-IP")
	}
	if fromIP == "" {
		fromIP = ctx.Request().Header.Get("X-Real-IP")
	}

	var askerUserID uint
	if ctx.IsLogged {
		askerUserID = ctx.User.ID
	}

	question, err := repository.Questions.Create(ctx.Request().Context(), repository.CreateQuestionOptions{
		FromIP:            fromIP,
		TenantID:          workspace.TenantID,
		WorkspaceID:       workspace.ID,
		UserID:            pageUser.ID,
		Content:           content,
		ReceiveReplyEmail: receiveReplyEmail,
		AskerUserID:       askerUserID,
		IsPrivate:         req.IsPrivate,
	})
	if err != nil {
		logger.Error("failed to create question", zap.Error(err), zap.Uint("page_user_id", pageUser.ID))
		return ctx.JSONError(50000, "创建问题失败")
	}

	recordQuestionAudit(ctx.Request().Context(), logger, question, bootstrap, askerUserID, "question.created", map[string]interface{}{
		"owner_user_id":  pageUser.ID,
		"is_private":     question.IsPrivate,
		"asker_user_id":  askerUserID,
		"content_length": len([]rune(question.Content)),
		"receive_reply":  receiveReplyEmail != "",
	})

	return ctx.JSON(CreateQuestionResponse{
		Success:  true,
		Message:  "提问成功",
		Question: question,
	})
}

type GetQuestionsRequest struct {
	PageSize int    `json:"page_size"`
	Cursor   string `json:"cursor"`
}

type GetQuestionsResponse struct {
	Success    bool              `json:"success"`
	Questions  []*model.Question `json:"questions"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

func GetQuestions(ctx appctx.Context) error {
	domain, _, err := parseQuestionRouteParams(ctx)
	if err != nil {
		return err
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.get_questions"),
		zap.String("domain", domain),
	)

	req := GetQuestionsRequest{
		PageSize: ctx.QueryInt("page_size"),
		Cursor:   ctx.Query("cursor"),
	}

	pageUser, err := repository.Users.GetByDomain(ctx.Request().Context(), domain)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotExists) {
			return ctx.JSONError(40400, "用户不存在")
		}
		return ctx.JSONError(50000, "获取用户信息失败")
	}

	pageQuestions, err := repository.Questions.GetByUserID(ctx.Request().Context(), pageUser.ID, repository.GetQuestionsByUserIDOptions{
		Cursor: &dbutil.Cursor{
			Value:    req.Cursor,
			PageSize: req.PageSize,
		},
		FilterAnswered: true,
	})
	if err != nil {
		logger.Error("failed to get questions", zap.Error(err), zap.Uint("page_user_id", pageUser.ID))
		return ctx.JSONError(50000, "获取问题列表失败")
	}

	nextCursor := ""
	if len(pageQuestions) > 0 {
		nextCursor = fmt.Sprintf("%d", pageQuestions[len(pageQuestions)-1].ID)
	}

	return ctx.JSON(GetQuestionsResponse{
		Success:    true,
		Questions:  pageQuestions,
		NextCursor: nextCursor,
	})
}

type GetQuestionResponse struct {
	Success   bool            `json:"success"`
	Question  *model.Question `json:"question,omitempty"`
	CanDelete bool            `json:"can_delete,omitempty"`
}

type PublicUser struct {
	Name       string `json:"name"`
	Avatar     string `json:"avatar"`
	Domain     string `json:"domain"`
	Background string `json:"background"`
	Intro      string `json:"intro"`
}

type GetUserResponse struct {
	Success bool       `json:"success"`
	User    PublicUser `json:"user"`
}

func GetUser(ctx appctx.Context) error {
	domain, _, err := parseQuestionRouteParams(ctx)
	if err != nil {
		return err
	}

	u, err := repository.Users.GetByDomain(ctx.Request().Context(), domain)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotExists) {
			return ctx.JSONError(40400, "用户不存在")
		}
		return ctx.JSONError(50000, "获取用户信息失败")
	}

	return ctx.JSON(GetUserResponse{
		Success: true,
		User: PublicUser{
			Name:       u.Name,
			Avatar:     u.Avatar,
			Domain:     u.Domain,
			Background: u.Background,
			Intro:      u.Intro,
		},
	})
}

func GetQuestion(ctx appctx.Context) error {
	domain, questionID, err := parseQuestionRouteParams(ctx)
	if err != nil {
		return err
	}

	pageUser, err := repository.Users.GetByDomain(ctx.Request().Context(), domain)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotExists) {
			return ctx.JSONError(40400, "用户不存在")
		}
		return ctx.JSONError(50000, "获取用户信息失败")
	}

	question, err := repository.Questions.GetByID(ctx.Request().Context(), questionID)
	if err != nil {
		if errors.Is(err, repository.ErrQuestionNotExist) {
			return ctx.JSONError(40400, "问题不存在")
		}
		return ctx.JSONError(50000, "获取问题失败")
	}

	if question.UserID != pageUser.ID {
		return ctx.JSONError(40300, "无权访问该问题")
	}

	canDelete, _, err := canManageQuestion(ctx, pageUser, question)
	if err != nil {
		return ctx.JSONError(50000, "获取问题失败")
	}
	if !canDelete && (question.IsPrivate || strings.TrimSpace(question.Answer) == "") {
		return ctx.JSONError(40300, "无权访问该问题")
	}

	return ctx.JSON(GetQuestionResponse{
		Success:   true,
		Question:  question,
		CanDelete: canDelete,
	})
}

type AnswerQuestionRequest struct {
	Answer string `json:"answer"`
}

type AnswerQuestionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func AnswerQuestion(ctx appctx.Context) error {
	var req AnswerQuestionRequest
	if err := request.BindJSON(ctx, &req); err != nil {
		return err
	}

	domain, questionID, err := parseQuestionRouteParams(ctx)
	if err != nil {
		return err
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.answer_question"),
		zap.String("domain", domain),
		zap.Uint("question_id", questionID),
		zap.Uint("user_id", ctx.User.ID),
	)

	pageUser, err := repository.Users.GetByDomain(ctx.Request().Context(), domain)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotExists) {
			return ctx.JSONError(40400, "用户不存在")
		}
		return ctx.JSONError(50000, "获取用户信息失败")
	}

	question, err := repository.Questions.GetByID(ctx.Request().Context(), questionID)
	if err != nil {
		if errors.Is(err, repository.ErrQuestionNotExist) {
			return ctx.JSONError(40400, "问题不存在")
		}
		return ctx.JSONError(50000, "获取问题失败")
	}
	if !questionBelongsToPage(question, pageUser) {
		return ctx.JSONError(40300, "无权回答该问题")
	}

	canManage, bootstrap, err := canManageQuestion(ctx, pageUser, question)
	if err != nil {
		logger.Error("failed to resolve question permissions", zap.Error(err))
		return ctx.JSONError(50000, "回答问题失败")
	}
	if !canManage {
		return ctx.JSONError(40300, "无权回答该问题")
	}

	if err := repository.Questions.AnswerByID(ctx.Request().Context(), question.ID, req.Answer); err != nil {
		logger.Error("failed to answer question", zap.Error(err))
		return ctx.JSONError(50000, "回答问题失败")
	}

	notifyQuestionAnswered(ctx.Request().Context(), logger, pageUser, question, req.Answer)

	question.Answer = strings.TrimSpace(req.Answer)
	if question.Answer == "" {
		question.Status = model.QuestionStatusInProgress
		question.ResolvedAt = nil
	} else {
		question.Status = model.QuestionStatusAnswered
		now := time.Now()
		question.ResolvedAt = &now
	}

	recordQuestionAudit(ctx.Request().Context(), logger, question, bootstrap, ctx.User.ID, "question.answered", map[string]interface{}{
		"owner_user_id": pageUser.ID,
		"answer_length": len([]rune(req.Answer)),
		"status":        string(question.Status),
	})

	return ctx.JSON(AnswerQuestionResponse{
		Success: true,
		Message: "回答发布成功",
	})
}

type DeleteQuestionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func DeleteQuestion(ctx appctx.Context) error {
	domain, questionID, err := parseQuestionRouteParams(ctx)
	if err != nil {
		return err
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.delete_question"),
		zap.String("domain", domain),
		zap.Uint("question_id", questionID),
		zap.Uint("user_id", ctx.User.ID),
	)

	pageUser, err := repository.Users.GetByDomain(ctx.Request().Context(), domain)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotExists) {
			return ctx.JSONError(40400, "用户不存在")
		}
		return ctx.JSONError(50000, "获取用户信息失败")
	}

	question, err := repository.Questions.GetByID(ctx.Request().Context(), questionID)
	if err != nil {
		if errors.Is(err, repository.ErrQuestionNotExist) {
			return ctx.JSONError(40400, "问题不存在")
		}
		return ctx.JSONError(50000, "获取问题失败")
	}
	if !questionBelongsToPage(question, pageUser) {
		return ctx.JSONError(40300, "无权删除该问题")
	}

	canManage, bootstrap, err := canManageQuestion(ctx, pageUser, question)
	if err != nil {
		logger.Error("failed to resolve question permissions", zap.Error(err))
		return ctx.JSONError(50000, "删除问题失败")
	}
	if !canManage {
		return ctx.JSONError(40300, "无权删除该问题")
	}

	if err := repository.Questions.DeleteByID(ctx.Request().Context(), questionID); err != nil {
		logger.Error("failed to delete question", zap.Error(err))
		return ctx.JSONError(50000, "删除问题失败")
	}

	recordQuestionAudit(ctx.Request().Context(), logger, question, bootstrap, ctx.User.ID, "question.deleted", map[string]interface{}{
		"owner_user_id": pageUser.ID,
		"status":        string(question.Status),
		"is_private":    question.IsPrivate,
	})

	return ctx.JSON(DeleteQuestionResponse{
		Success: true,
		Message: "删除成功",
	})
}

type SetQuestionPrivateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func SetQuestionPrivate(ctx appctx.Context) error {
	domain, questionID, err := parseQuestionRouteParams(ctx)
	if err != nil {
		return err
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.set_question_private"),
		zap.String("domain", domain),
		zap.Uint("question_id", questionID),
		zap.Uint("user_id", ctx.User.ID),
	)

	pageUser, err := repository.Users.GetByDomain(ctx.Request().Context(), domain)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotExists) {
			return ctx.JSONError(40400, "用户不存在")
		}
		return ctx.JSONError(50000, "获取用户信息失败")
	}

	question, err := repository.Questions.GetByID(ctx.Request().Context(), questionID)
	if err != nil {
		if errors.Is(err, repository.ErrQuestionNotExist) {
			return ctx.JSONError(40400, "问题不存在")
		}
		return ctx.JSONError(50000, "获取问题失败")
	}
	if !questionBelongsToPage(question, pageUser) {
		return ctx.JSONError(40300, "无权操作该问题")
	}

	canManage, bootstrap, err := canManageQuestion(ctx, pageUser, question)
	if err != nil {
		logger.Error("failed to resolve question permissions", zap.Error(err))
		return ctx.JSONError(50000, "设置问题私密失败")
	}
	if !canManage {
		return ctx.JSONError(40300, "无权操作该问题")
	}

	if err := repository.Questions.SetPrivate(ctx.Request().Context(), questionID); err != nil {
		logger.Error("failed to set question private", zap.Error(err))
		return ctx.JSONError(50000, "设置问题私密失败")
	}

	question.IsPrivate = true
	recordQuestionAudit(ctx.Request().Context(), logger, question, bootstrap, ctx.User.ID, "question.visibility_changed", map[string]interface{}{
		"owner_user_id": pageUser.ID,
		"is_private":    true,
	})

	return ctx.JSON(SetQuestionPrivateResponse{
		Success: true,
		Message: "设置成功",
	})
}

func SetQuestionPublic(ctx appctx.Context) error {
	domain, questionID, err := parseQuestionRouteParams(ctx)
	if err != nil {
		return err
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.set_question_public"),
		zap.String("domain", domain),
		zap.Uint("question_id", questionID),
		zap.Uint("user_id", ctx.User.ID),
	)

	pageUser, err := repository.Users.GetByDomain(ctx.Request().Context(), domain)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotExists) {
			return ctx.JSONError(40400, "用户不存在")
		}
		return ctx.JSONError(50000, "获取用户信息失败")
	}

	question, err := repository.Questions.GetByID(ctx.Request().Context(), questionID)
	if err != nil {
		if errors.Is(err, repository.ErrQuestionNotExist) {
			return ctx.JSONError(40400, "问题不存在")
		}
		return ctx.JSONError(50000, "获取问题失败")
	}
	if !questionBelongsToPage(question, pageUser) {
		return ctx.JSONError(40300, "无权操作该问题")
	}

	canManage, bootstrap, err := canManageQuestion(ctx, pageUser, question)
	if err != nil {
		logger.Error("failed to resolve question permissions", zap.Error(err))
		return ctx.JSONError(50000, "设置问题公开失败")
	}
	if !canManage {
		return ctx.JSONError(40300, "无权操作该问题")
	}

	if err := repository.Questions.SetPublic(ctx.Request().Context(), questionID); err != nil {
		logger.Error("failed to set question public", zap.Error(err))
		return ctx.JSONError(50000, "设置问题公开失败")
	}

	question.IsPrivate = false
	recordQuestionAudit(ctx.Request().Context(), logger, question, bootstrap, ctx.User.ID, "question.visibility_changed", map[string]interface{}{
		"owner_user_id": pageUser.ID,
		"is_private":    false,
	})

	return ctx.JSON(SetQuestionPrivateResponse{
		Success: true,
		Message: "设置成功",
	})
}
