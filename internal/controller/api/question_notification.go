package api

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/syt3s/TreeBox/internal/config"
	"github.com/syt3s/TreeBox/internal/model"
	"github.com/syt3s/TreeBox/internal/notify"
)

func notifyQuestionAnswered(ctx context.Context, logger *zap.Logger, pageUser *model.User, question *model.Question, answer string) {
	if strings.TrimSpace(question.ReceiveReplyEmail) == "" {
		return
	}
	if strings.TrimSpace(question.Answer) != "" {
		return
	}
	if strings.TrimSpace(answer) == "" {
		return
	}

	questionURL := ""
	if !question.IsPrivate {
		questionURL = fmt.Sprintf("%s/box/%s/%d", config.App.ExternalURL, pageUser.Domain, question.ID)
	}

	err := notify.SendQuestionAnswered(ctx, notify.ReplyEmailInput{
		To:              question.ReceiveReplyEmail,
		PageName:        pageUser.Name,
		PageDomain:      pageUser.Domain,
		QuestionContent: question.Content,
		AnswerContent:   answer,
		QuestionURL:     questionURL,
		IsPrivate:       question.IsPrivate,
	})
	if err == nil {
		return
	}

	if errors.Is(err, notify.ErrMailNotConfigured) {
		logger.Warn("reply notification email skipped because mail is not configured",
			zap.Uint("question_id", question.ID),
			zap.String("receive_reply_email", question.ReceiveReplyEmail),
		)
		return
	}

	logger.Error("failed to send reply notification email",
		zap.Error(err),
		zap.Uint("question_id", question.ID),
		zap.String("receive_reply_email", question.ReceiveReplyEmail),
	)
}
