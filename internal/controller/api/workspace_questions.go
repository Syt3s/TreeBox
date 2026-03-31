package api

import (
	"fmt"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/syt3s/TreeBox/internal/dbutil"
	"github.com/syt3s/TreeBox/internal/http/appctx"
	"github.com/syt3s/TreeBox/internal/logging"
	"github.com/syt3s/TreeBox/internal/model"
	"github.com/syt3s/TreeBox/internal/repository"
)

type ListWorkspaceQuestionsResponse struct {
	Success    bool              `json:"success"`
	Workspace  WorkspaceSummary  `json:"workspace"`
	Questions  []*model.Question `json:"questions"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

func ListWorkspaceQuestions(ctx appctx.Context) error {
	access, err := loadManagedWorkspaceAccess(ctx, false)
	if err != nil {
		return err
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.list_workspace_questions"),
		zap.String("workspace_uid", access.Workspace.UID),
		zap.Uint("user_id", ctx.User.ID),
	)

	pageSize := ctx.QueryInt("page_size")
	cursor := ctx.Query("cursor")
	filterAnswered := ctx.Query("filter_answered") == "true"
	showPrivate := ctx.Query("show_private", "true") != "false"
	status := model.QuestionStatus(strings.TrimSpace(ctx.Query("status")))
	if status != "" && !status.IsValid() {
		return ctx.JSONError(40000, "问题状态无效")
	}

	var assignedToUserID *uint
	if rawAssignedUserID := strings.TrimSpace(ctx.Query("assigned_to_user_id")); rawAssignedUserID != "" {
		value, parseErr := strconv.ParseUint(rawAssignedUserID, 10, 64)
		if parseErr != nil {
			return ctx.JSONError(40000, "指派成员编号无效")
		}
		assignedUserID := uint(value)
		assignedToUserID = &assignedUserID
	}

	questions, err := repository.Questions.GetByWorkspaceID(ctx.Request().Context(), access.Workspace.ID, repository.GetQuestionsByWorkspaceIDOptions{
		Cursor: &dbutil.Cursor{
			Value:    cursor,
			PageSize: pageSize,
		},
		FilterAnswered:   filterAnswered,
		ShowPrivate:      showPrivate,
		Status:           status,
		AssignedToUserID: assignedToUserID,
		OnlyAssigned:     ctx.Query("only_assigned") == "true",
		OnlyUnassigned:   ctx.Query("only_unassigned") == "true",
	})
	if err != nil {
		logger.Error("failed to list workspace questions", zap.Error(err), zap.Uint("workspace_id", access.Workspace.ID))
		return ctx.JSONError(50000, "获取工作区问题失败")
	}

	nextCursor := ""
	if len(questions) > 0 {
		nextCursor = fmt.Sprintf("%d", questions[len(questions)-1].ID)
	}

	return ctx.JSON(ListWorkspaceQuestionsResponse{
		Success:    true,
		Workspace:  buildWorkspaceSummary(access.Workspace, access.Tenant, access.Membership.Role),
		Questions:  questions,
		NextCursor: nextCursor,
	})
}
