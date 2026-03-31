package api

import (
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/syt3s/TreeBox/internal/http/appctx"
	"github.com/syt3s/TreeBox/internal/http/request"
	"github.com/syt3s/TreeBox/internal/logging"
	"github.com/syt3s/TreeBox/internal/model"
	"github.com/syt3s/TreeBox/internal/repository"
)

type WorkspaceQuestionStatsResponse struct {
	Success   bool                              `json:"success"`
	Workspace WorkspaceSummary                  `json:"workspace"`
	Stats     repository.WorkspaceQuestionStats `json:"stats"`
}

type WorkspaceQuestionMutationResponse struct {
	Success  bool            `json:"success"`
	Message  string          `json:"message,omitempty"`
	Question *model.Question `json:"question,omitempty"`
}

type UpdateWorkspaceQuestionStatusRequest struct {
	Status string `json:"status"`
}

type UpdateWorkspaceQuestionAssigneeRequest struct {
	AssignedToUserID *uint `json:"assigned_to_user_id"`
}

type UpdateWorkspaceQuestionInternalNoteRequest struct {
	InternalNote string `json:"internal_note"`
}

type UpdateWorkspaceQuestionPrivacyRequest struct {
	IsPrivate bool `json:"is_private"`
}

type workspaceQuestionAccess struct {
	Workspace  *model.Workspace
	Tenant     *model.Tenant
	Membership *model.TenantMember
	Question   *model.Question
}

func GetWorkspaceQuestionStats(ctx appctx.Context) error {
	access, err := loadManagedWorkspaceAccess(ctx, false)
	if err != nil {
		return err
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.get_workspace_question_stats"),
		zap.Uint("workspace_id", access.Workspace.ID),
		zap.Uint("user_id", ctx.User.ID),
	)

	stats, err := repository.Questions.GetWorkspaceStats(ctx.Request().Context(), access.Workspace.ID, repository.GetWorkspaceQuestionStatsOptions{
		ShowPrivate: true,
	})
	if err != nil {
		logger.Error("failed to load workspace stats", zap.Error(err))
		return ctx.JSONError(50000, "获取工作区统计失败")
	}

	return ctx.JSON(WorkspaceQuestionStatsResponse{
		Success:   true,
		Workspace: buildWorkspaceSummary(access.Workspace, access.Tenant, access.Membership.Role),
		Stats:     *stats,
	})
}

func UpdateWorkspaceQuestionStatus(ctx appctx.Context) error {
	var req UpdateWorkspaceQuestionStatusRequest
	if err := request.BindJSON(ctx, &req); err != nil {
		return err
	}

	status := model.QuestionStatus(strings.TrimSpace(req.Status))
	if !status.IsValid() {
		return ctx.JSONError(40000, "问题状态无效")
	}

	access, err := loadManagedWorkspaceAccess(ctx, true)
	if err != nil {
		return err
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.update_workspace_question_status"),
		zap.Uint("workspace_id", access.Workspace.ID),
		zap.Uint("question_id", access.Question.ID),
		zap.Uint("user_id", ctx.User.ID),
	)

	question, err := repository.Questions.UpdateStatus(ctx.Request().Context(), access.Question.ID, status)
	if err != nil {
		logger.Error("failed to update workspace question status", zap.Error(err))
		return ctx.JSONError(50000, "更新工作区问题状态失败")
	}

	recordQuestionAudit(ctx.Request().Context(), logger, question, nil, ctx.User.ID, "question.status_changed", map[string]interface{}{
		"status": status,
	})

	return ctx.JSON(WorkspaceQuestionMutationResponse{
		Success:  true,
		Message:  "问题状态已更新",
		Question: question,
	})
}

func UpdateWorkspaceQuestionAssignee(ctx appctx.Context) error {
	var req UpdateWorkspaceQuestionAssigneeRequest
	if err := request.BindJSON(ctx, &req); err != nil {
		return err
	}

	access, err := loadManagedWorkspaceAccess(ctx, true)
	if err != nil {
		return err
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.update_workspace_question_assignee"),
		zap.Uint("workspace_id", access.Workspace.ID),
		zap.Uint("question_id", access.Question.ID),
		zap.Uint("user_id", ctx.User.ID),
	)

	if req.AssignedToUserID != nil {
		assigneeMembership, err := repository.Tenants.GetMembership(ctx.Request().Context(), access.Tenant.ID, *req.AssignedToUserID)
		if err != nil {
			if errors.Is(err, repository.ErrTenantMembershipNotExists) {
				return ctx.JSONError(40400, "被指派成员不属于当前租户")
			}
			logger.Error("failed to load assignee membership", zap.Error(err), zap.Uint("assignee_user_id", *req.AssignedToUserID))
			return ctx.JSONError(50000, "更新工作区问题负责人失败")
		}
		if !assigneeMembership.Role.CanManageQuestions() {
			return ctx.JSONError(40000, "该成员当前角色不能处理问题")
		}
	}

	question, err := repository.Questions.UpdateAssignment(ctx.Request().Context(), access.Question.ID, req.AssignedToUserID)
	if err != nil {
		logger.Error("failed to update question assignee", zap.Error(err))
		return ctx.JSONError(50000, "更新工作区问题负责人失败")
	}

	metadata := map[string]interface{}{}
	if req.AssignedToUserID != nil {
		metadata["assigned_to_user_id"] = *req.AssignedToUserID
	} else {
		metadata["assigned_to_user_id"] = nil
	}
	recordQuestionAudit(ctx.Request().Context(), logger, question, nil, ctx.User.ID, "question.assignee_changed", metadata)

	return ctx.JSON(WorkspaceQuestionMutationResponse{
		Success:  true,
		Message:  "问题负责人已更新",
		Question: question,
	})
}

func UpdateWorkspaceQuestionInternalNote(ctx appctx.Context) error {
	var req UpdateWorkspaceQuestionInternalNoteRequest
	if err := request.BindJSON(ctx, &req); err != nil {
		return err
	}

	access, err := loadManagedWorkspaceAccess(ctx, true)
	if err != nil {
		return err
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.update_workspace_question_internal_note"),
		zap.Uint("workspace_id", access.Workspace.ID),
		zap.Uint("question_id", access.Question.ID),
		zap.Uint("user_id", ctx.User.ID),
	)

	question, err := repository.Questions.UpdateInternalNote(ctx.Request().Context(), access.Question.ID, req.InternalNote)
	if err != nil {
		logger.Error("failed to update internal note", zap.Error(err))
		return ctx.JSONError(50000, "更新内部备注失败")
	}

	recordQuestionAudit(ctx.Request().Context(), logger, question, nil, ctx.User.ID, "question.internal_note_changed", map[string]interface{}{
		"internal_note_length": len([]rune(question.InternalNote)),
	})

	return ctx.JSON(WorkspaceQuestionMutationResponse{
		Success:  true,
		Message:  "内部备注已更新",
		Question: question,
	})
}

func AnswerWorkspaceQuestion(ctx appctx.Context) error {
	var req AnswerQuestionRequest
	if err := request.BindJSON(ctx, &req); err != nil {
		return err
	}

	access, err := loadManagedWorkspaceAccess(ctx, true)
	if err != nil {
		return err
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.answer_workspace_question"),
		zap.Uint("workspace_id", access.Workspace.ID),
		zap.Uint("question_id", access.Question.ID),
		zap.Uint("user_id", ctx.User.ID),
	)

	if err := repository.Questions.AnswerByID(ctx.Request().Context(), access.Question.ID, req.Answer); err != nil {
		logger.Error("failed to answer workspace question", zap.Error(err))
		return ctx.JSONError(50000, "回复工作区问题失败")
	}

	question, err := repository.Questions.GetByID(ctx.Request().Context(), access.Question.ID)
	if err != nil {
		logger.Error("failed to reload answered question", zap.Error(err))
		return ctx.JSONError(50000, "回复工作区问题失败")
	}

	pageUser, err := repository.Users.GetByID(ctx.Request().Context(), question.UserID)
	if err != nil {
		logger.Error("failed to load page owner for workspace answer", zap.Error(err), zap.Uint("owner_user_id", question.UserID))
		return ctx.JSONError(50000, "回复工作区问题失败")
	}

	notifyQuestionAnswered(ctx.Request().Context(), logger, pageUser, access.Question, req.Answer)
	recordQuestionAudit(ctx.Request().Context(), logger, question, nil, ctx.User.ID, "question.answered", map[string]interface{}{
		"owner_user_id": pageUser.ID,
		"answer_length": len([]rune(strings.TrimSpace(req.Answer))),
		"status":        string(question.Status),
	})

	return ctx.JSON(WorkspaceQuestionMutationResponse{
		Success:  true,
		Message:  "工作区回复已发布",
		Question: question,
	})
}

func UpdateWorkspaceQuestionPrivacy(ctx appctx.Context) error {
	var req UpdateWorkspaceQuestionPrivacyRequest
	if err := request.BindJSON(ctx, &req); err != nil {
		return err
	}

	access, err := loadManagedWorkspaceAccess(ctx, true)
	if err != nil {
		return err
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.update_workspace_question_privacy"),
		zap.Uint("workspace_id", access.Workspace.ID),
		zap.Uint("question_id", access.Question.ID),
		zap.Uint("user_id", ctx.User.ID),
	)

	if req.IsPrivate {
		if err := repository.Questions.SetPrivate(ctx.Request().Context(), access.Question.ID); err != nil {
			logger.Error("failed to set workspace question private", zap.Error(err))
			return ctx.JSONError(50000, "更新问题可见性失败")
		}
	} else {
		if err := repository.Questions.SetPublic(ctx.Request().Context(), access.Question.ID); err != nil {
			logger.Error("failed to set workspace question public", zap.Error(err))
			return ctx.JSONError(50000, "更新问题可见性失败")
		}
	}

	access.Question.IsPrivate = req.IsPrivate
	recordQuestionAudit(ctx.Request().Context(), logger, access.Question, nil, ctx.User.ID, "question.visibility_changed", map[string]interface{}{
		"is_private": req.IsPrivate,
	})

	return ctx.JSON(WorkspaceQuestionMutationResponse{
		Success:  true,
		Message:  "问题可见性已更新",
		Question: access.Question,
	})
}

func loadManagedWorkspaceAccess(ctx appctx.Context, includeQuestion bool) (*workspaceQuestionAccess, error) {
	workspaceUID := strings.TrimSpace(ctx.Param("workspaceUID"))
	if workspaceUID == "" {
		return nil, ctx.JSONError(40000, "工作区标识不能为空")
	}

	workspace, err := repository.Workspaces.GetByUID(ctx.Request().Context(), workspaceUID)
	if err != nil {
		if err == repository.ErrWorkspaceNotExist {
			return nil, ctx.JSONError(40400, "工作区不存在")
		}
		return nil, ctx.JSONError(50000, "获取工作区失败")
	}

	tenant, err := repository.Tenants.GetByID(ctx.Request().Context(), workspace.TenantID)
	if err != nil {
		return nil, ctx.JSONError(50000, "获取租户失败")
	}

	membership, err := repository.Tenants.GetMembership(ctx.Request().Context(), tenant.ID, ctx.User.ID)
	if err != nil {
		if errors.Is(err, repository.ErrTenantMembershipNotExists) {
			return nil, ctx.JSONError(40300, "无权访问该工作区")
		}
		return nil, ctx.JSONError(50000, "获取工作区成员信息失败")
	}
	if !membership.Role.CanManageQuestions() {
		return nil, ctx.JSONError(40300, "当前角色无权处理该工作区的问题")
	}

	access := &workspaceQuestionAccess{
		Workspace:  workspace,
		Tenant:     tenant,
		Membership: membership,
	}
	if !includeQuestion {
		return access, nil
	}

	questionID, err := strconv.ParseUint(strings.TrimSpace(ctx.Param("questionID")), 10, 64)
	if err != nil {
		return nil, ctx.JSONError(40000, "问题编号无效")
	}

	question, err := repository.Questions.GetByID(ctx.Request().Context(), uint(questionID))
	if err != nil {
		if errors.Is(err, repository.ErrQuestionNotExist) {
			return nil, ctx.JSONError(40400, "问题不存在")
		}
		return nil, ctx.JSONError(50000, "获取问题失败")
	}
	if question.WorkspaceID != workspace.ID {
		return nil, ctx.JSONError(40300, "该问题不属于当前工作区")
	}

	access.Question = question
	return access, nil
}

func buildWorkspaceSummary(workspace *model.Workspace, tenant *model.Tenant, role model.TenantRole) WorkspaceSummary {
	return WorkspaceSummary{
		ID:          workspace.ID,
		UID:         workspace.UID,
		Name:        workspace.Name,
		Description: workspace.Description,
		IsDefault:   workspace.IsDefault,
		CreatedAt:   workspace.CreatedAt.Format(time.RFC3339),
		Tenant:      buildTenantSummary(tenant, role),
	}
}
