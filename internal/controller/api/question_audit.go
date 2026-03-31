package api

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/syt3s/TreeBox/internal/model"
	"github.com/syt3s/TreeBox/internal/repository"
)

func recordQuestionAudit(ctx context.Context, logger *zap.Logger, question *model.Question, bootstrap *repository.TenantBootstrapResult, actorUserID uint, action string, metadata map[string]interface{}) {
	if question == nil || repository.AuditLogs == nil {
		return
	}

	tenantID := question.TenantID
	if tenantID == 0 && bootstrap != nil && bootstrap.Tenant != nil {
		tenantID = bootstrap.Tenant.ID
	}
	if tenantID == 0 {
		return
	}

	workspaceID := question.WorkspaceID
	if workspaceID == 0 && bootstrap != nil && bootstrap.Workspace != nil {
		workspaceID = bootstrap.Workspace.ID
	}

	var workspaceIDPtr *uint
	if workspaceID != 0 {
		workspaceIDPtr = &workspaceID
	}

	if _, err := repository.AuditLogs.Record(ctx, repository.RecordAuditLogOptions{
		TenantID:     tenantID,
		WorkspaceID:  workspaceIDPtr,
		ActorUserID:  actorUserID,
		Action:       action,
		ResourceType: "question",
		ResourceID:   fmt.Sprintf("%d", question.ID),
		Metadata:     metadata,
	}); err != nil && logger != nil {
		logger.Warn("failed to record question audit log", zap.Error(err), zap.Uint("question_id", question.ID), zap.String("action", action))
	}
}
