package api

import (
	"strings"

	"go.uber.org/zap"

	"github.com/syt3s/TreeBox/internal/http/appctx"
	"github.com/syt3s/TreeBox/internal/logging"
	"github.com/syt3s/TreeBox/internal/model"
	"github.com/syt3s/TreeBox/internal/repository"
)

type SetWorkspaceIntakeResponse struct {
	Success   bool             `json:"success"`
	Message   string           `json:"message,omitempty"`
	User      *model.User      `json:"user,omitempty"`
	Workspace WorkspaceSummary `json:"workspace"`
}

func SetWorkspaceIntake(ctx appctx.Context) error {
	access, err := loadManagedWorkspaceAccess(ctx, false)
	if err != nil {
		return err
	}
	if !access.Membership.Role.CanManageWorkspace() {
		return ctx.JSONError(40300, "当前角色无权设置工作区接收入口")
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.set_workspace_intake"),
		zap.Uint("workspace_id", access.Workspace.ID),
		zap.Uint("user_id", ctx.User.ID),
	)

	user, err := repository.Users.UpdateRoutingWorkspace(ctx.Request().Context(), ctx.User.ID, access.Workspace.ID)
	if err != nil {
		logger.Error("failed to update routing workspace", zap.Error(err))
		return ctx.JSONError(50000, "设置工作区接收入口失败")
	}

	if repository.AuditLogs != nil {
		workspaceID := access.Workspace.ID
		if _, err := repository.AuditLogs.Record(ctx.Request().Context(), repository.RecordAuditLogOptions{
			TenantID:     access.Tenant.ID,
			WorkspaceID:  &workspaceID,
			ActorUserID:  ctx.User.ID,
			Action:       "workspace.intake.bound",
			ResourceType: "workspace",
			ResourceID:   access.Workspace.UID,
			Metadata: map[string]interface{}{
				"page_user_id":         ctx.User.ID,
				"workspace_name":       strings.TrimSpace(access.Workspace.Name),
				"routing_workspace_id": access.Workspace.ID,
			},
		}); err != nil {
			logger.Warn("failed to record workspace intake audit log", zap.Error(err))
		}
	}

	return ctx.JSON(SetWorkspaceIntakeResponse{
		Success:   true,
		Message:   "工作区接收入口已更新",
		User:      user,
		Workspace: buildWorkspaceSummary(access.Workspace, access.Tenant, access.Membership.Role),
	})
}
