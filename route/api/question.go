package api

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/flamego/recaptcha"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/wuhan005/govalid"

	"github.com/syt3s/TreeBox/internal/context"
	"github.com/syt3s/TreeBox/internal/db"
	"github.com/syt3s/TreeBox/internal/dbutil"
)

type CreateQuestionRequest struct {
	Content           string `json:"content"`
	IsPrivate         bool   `json:"is_private"`
	ReceiveReplyEmail string `json:"receive_reply_email"`
	Recaptcha         string `json:"recaptcha"`
}

type CreateQuestionResponse struct {
	Success  bool         `json:"success"`
	Message  string       `json:"message,omitempty"`
	Question *db.Question `json:"question,omitempty"`
}

func parseQuestionRouteParams(ctx context.Context) (string, uint, error) {
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

func CreateQuestion(ctx context.Context, req CreateQuestionRequest, recaptcha recaptcha.RecaptchaV3) error {
	domain, _, err := parseQuestionRouteParams(ctx)
	if err != nil {
		return err
	}

	pageUser, err := db.Users.GetByDomain(ctx.Request().Context(), domain)
	if err != nil {
		if errors.Is(err, db.ErrUserNotExists) {
			return ctx.JSONError(40400, "用户不存在")
		}
		return ctx.JSONError(50000, "获取用户信息失败")
	}

	if !ctx.IsLogged && pageUser.HarassmentSetting == db.HarassmentSettingTypeRegisterOnly {
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

	resp, err := recaptcha.Verify(req.Recaptcha, ctx.Request().Request.RemoteAddr)
	if err != nil {
		logrus.WithContext(ctx.Request().Context()).WithError(err).Error("Failed to check recaptcha")
		return ctx.JSONError(50000, "验证码校验失败")
	}
	if !resp.Success {
		return ctx.JSONError(40000, "验证码错误")
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

	question, err := db.Questions.Create(ctx.Request().Context(), db.CreateQuestionOptions{
		FromIP:            fromIP,
		UserID:            pageUser.ID,
		Content:           content,
		ReceiveReplyEmail: receiveReplyEmail,
		AskerUserID:       askerUserID,
		IsPrivate:         req.IsPrivate,
	})
	if err != nil {
		logrus.WithContext(ctx.Request().Context()).WithError(err).Error("Failed to create question")
		return ctx.JSONError(50000, "创建问题失败")
	}

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
	Success    bool           `json:"success"`
	Questions  []*db.Question `json:"questions"`
	NextCursor string         `json:"next_cursor,omitempty"`
}

func GetQuestions(ctx context.Context) error {
	domain, _, err := parseQuestionRouteParams(ctx)
	if err != nil {
		return err
	}

	req := GetQuestionsRequest{
		PageSize: ctx.QueryInt("page_size"),
		Cursor:   ctx.Query("cursor"),
	}

	pageUser, err := db.Users.GetByDomain(ctx.Request().Context(), domain)
	if err != nil {
		if errors.Is(err, db.ErrUserNotExists) {
			return ctx.JSONError(40400, "用户不存在")
		}
		return ctx.JSONError(50000, "获取用户信息失败")
	}

	pageQuestions, err := db.Questions.GetByUserID(ctx.Request().Context(), pageUser.ID, db.GetQuestionsByUserIDOptions{
		Cursor: &dbutil.Cursor{
			Value:    req.Cursor,
			PageSize: req.PageSize,
		},
		FilterAnswered: true,
	})
	if err != nil {
		logrus.WithContext(ctx.Request().Context()).WithError(err).Error("Failed to get questions")
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
	Success   bool         `json:"success"`
	Question  *db.Question `json:"question,omitempty"`
	CanDelete bool         `json:"can_delete,omitempty"`
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

func GetUser(ctx context.Context) error {
	domain, _, err := parseQuestionRouteParams(ctx)
	if err != nil {
		return err
	}

	u, err := db.Users.GetByDomain(ctx.Request().Context(), domain)
	if err != nil {
		if errors.Is(err, db.ErrUserNotExists) {
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

func GetQuestion(ctx context.Context) error {
	domain, questionID, err := parseQuestionRouteParams(ctx)
	if err != nil {
		return err
	}

	pageUser, err := db.Users.GetByDomain(ctx.Request().Context(), domain)
	if err != nil {
		if errors.Is(err, db.ErrUserNotExists) {
			return ctx.JSONError(40400, "用户不存在")
		}
		return ctx.JSONError(50000, "获取用户信息失败")
	}

	question, err := db.Questions.GetByID(ctx.Request().Context(), questionID)
	if err != nil {
		if errors.Is(err, db.ErrQuestionNotExist) {
			return ctx.JSONError(40400, "问题不存在")
		}
		return ctx.JSONError(50000, "获取问题失败")
	}

	if question.UserID != pageUser.ID {
		return ctx.JSONError(40300, "无权访问该问题")
	}

	canDelete := ctx.IsLogged && ctx.User.ID == pageUser.ID
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

func AnswerQuestion(ctx context.Context, req AnswerQuestionRequest) error {
	domain, questionID, err := parseQuestionRouteParams(ctx)
	if err != nil {
		return err
	}

	pageUser, err := db.Users.GetByDomain(ctx.Request().Context(), domain)
	if err != nil {
		if errors.Is(err, db.ErrUserNotExists) {
			return ctx.JSONError(40400, "用户不存在")
		}
		return ctx.JSONError(50000, "获取用户信息失败")
	}

	if ctx.User.ID != pageUser.ID {
		return ctx.JSONError(40300, "无权回答该问题")
	}

	question, err := db.Questions.GetByID(ctx.Request().Context(), questionID)
	if err != nil {
		if errors.Is(err, db.ErrQuestionNotExist) {
			return ctx.JSONError(40400, "问题不存在")
		}
		return ctx.JSONError(50000, "获取问题失败")
	}

	if err := db.Questions.AnswerByID(ctx.Request().Context(), question.ID, req.Answer); err != nil {
		logrus.WithContext(ctx.Request().Context()).WithError(err).Error("Failed to answer question")
		return ctx.JSONError(50000, "回答问题失败")
	}

	return ctx.JSON(AnswerQuestionResponse{
		Success: true,
		Message: "回答发布成功",
	})
}

type DeleteQuestionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func DeleteQuestion(ctx context.Context) error {
	domain, questionID, err := parseQuestionRouteParams(ctx)
	if err != nil {
		return err
	}

	pageUser, err := db.Users.GetByDomain(ctx.Request().Context(), domain)
	if err != nil {
		if errors.Is(err, db.ErrUserNotExists) {
			return ctx.JSONError(40400, "用户不存在")
		}
		return ctx.JSONError(50000, "获取用户信息失败")
	}

	if ctx.User.ID != pageUser.ID {
		return ctx.JSONError(40300, "无权删除该问题")
	}

	if err := db.Questions.DeleteByID(ctx.Request().Context(), questionID); err != nil {
		logrus.WithContext(ctx.Request().Context()).WithError(err).Error("Failed to delete question")
		return ctx.JSONError(50000, "删除问题失败")
	}

	return ctx.JSON(DeleteQuestionResponse{
		Success: true,
		Message: "删除成功",
	})
}

type SetQuestionPrivateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func SetQuestionPrivate(ctx context.Context) error {
	domain, questionID, err := parseQuestionRouteParams(ctx)
	if err != nil {
		return err
	}

	pageUser, err := db.Users.GetByDomain(ctx.Request().Context(), domain)
	if err != nil {
		if errors.Is(err, db.ErrUserNotExists) {
			return ctx.JSONError(40400, "用户不存在")
		}
		return ctx.JSONError(50000, "获取用户信息失败")
	}

	if ctx.User.ID != pageUser.ID {
		return ctx.JSONError(40300, "无权操作该问题")
	}

	if err := db.Questions.SetPrivate(ctx.Request().Context(), questionID); err != nil {
		logrus.WithContext(ctx.Request().Context()).WithError(err).Error("Failed to set question private")
		return ctx.JSONError(50000, "设置问题私密失败")
	}

	return ctx.JSON(SetQuestionPrivateResponse{
		Success: true,
		Message: "设置成功",
	})
}

func SetQuestionPublic(ctx context.Context) error {
	domain, questionID, err := parseQuestionRouteParams(ctx)
	if err != nil {
		return err
	}

	pageUser, err := db.Users.GetByDomain(ctx.Request().Context(), domain)
	if err != nil {
		if errors.Is(err, db.ErrUserNotExists) {
			return ctx.JSONError(40400, "用户不存在")
		}
		return ctx.JSONError(50000, "获取用户信息失败")
	}

	if ctx.User.ID != pageUser.ID {
		return ctx.JSONError(40300, "无权操作该问题")
	}

	if err := db.Questions.SetPublic(ctx.Request().Context(), questionID); err != nil {
		logrus.WithContext(ctx.Request().Context()).WithError(err).Error("Failed to set question public")
		return ctx.JSONError(50000, "设置问题公开失败")
	}

	return ctx.JSON(SetQuestionPrivateResponse{
		Success: true,
		Message: "设置成功",
	})
}
