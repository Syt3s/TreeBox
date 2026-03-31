package api

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/syt3s/TreeBox/internal/http/appctx"
	"github.com/syt3s/TreeBox/internal/http/request"
	"github.com/syt3s/TreeBox/internal/logging"
	"github.com/syt3s/TreeBox/internal/repository"
)

type TenantSummary struct {
	UID        string `json:"uid"`
	Name       string `json:"name"`
	Plan       string `json:"plan"`
	Role       string `json:"role"`
	IsPersonal bool   `json:"is_personal"`
	CreatedAt  string `json:"created_at"`
}

type ListTenantsResponse struct {
	Success bool            `json:"success"`
	Tenants []TenantSummary `json:"tenants"`
}

func ListTenants(ctx appctx.Context) error {
	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.list_tenants"),
		zap.Uint("user_id", ctx.User.ID),
	)

	memberships, err := repository.Tenants.ListByUserID(ctx.Request().Context(), ctx.User.ID)
	if err != nil {
		logger.Error("failed to list tenants", zap.Error(err))
		return ctx.JSONError(50000, "获取租户列表失败")
	}

	tenants := make([]TenantSummary, 0, len(memberships))
	for _, membership := range memberships {
		tenants = append(tenants, TenantSummary{
			UID:        membership.Tenant.UID,
			Name:       membership.Tenant.Name,
			Plan:       string(membership.Tenant.Plan),
			Role:       string(membership.Role),
			IsPersonal: membership.Tenant.IsPersonal,
			CreatedAt:  membership.Tenant.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return ctx.JSON(ListTenantsResponse{
		Success: true,
		Tenants: tenants,
	})
}

type WorkspaceSummary struct {
	ID          uint          `json:"id"`
	UID         string        `json:"uid"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	IsDefault   bool          `json:"is_default"`
	CreatedAt   string        `json:"created_at"`
	Tenant      TenantSummary `json:"tenant"`
}

type ListWorkspacesResponse struct {
	Success    bool               `json:"success"`
	Workspaces []WorkspaceSummary `json:"workspaces"`
}

func ListWorkspaces(ctx appctx.Context) error {
	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.list_workspaces"),
		zap.Uint("user_id", ctx.User.ID),
	)

	accessList, err := repository.Workspaces.ListByUserID(ctx.Request().Context(), ctx.User.ID)
	if err != nil {
		logger.Error("failed to list workspaces", zap.Error(err))
		return ctx.JSONError(50000, "获取工作区列表失败")
	}

	workspaces := make([]WorkspaceSummary, 0, len(accessList))
	for _, access := range accessList {
		workspaces = append(workspaces, WorkspaceSummary{
			ID:          access.Workspace.ID,
			UID:         access.Workspace.UID,
			Name:        access.Workspace.Name,
			Description: access.Workspace.Description,
			IsDefault:   access.Workspace.IsDefault,
			CreatedAt:   access.Workspace.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Tenant: TenantSummary{
				UID:        access.Tenant.UID,
				Name:       access.Tenant.Name,
				Plan:       string(access.Tenant.Plan),
				Role:       string(access.Role),
				IsPersonal: access.Tenant.IsPersonal,
				CreatedAt:  access.Tenant.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			},
		})
	}

	return ctx.JSON(ListWorkspacesResponse{
		Success:    true,
		Workspaces: workspaces,
	})
}

type CreateWorkspaceRequest struct {
	TenantUID   string `json:"tenant_uid"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CreateWorkspaceResponse struct {
	Success   bool             `json:"success"`
	Message   string           `json:"message,omitempty"`
	Workspace WorkspaceSummary `json:"workspace"`
}

func CreateWorkspace(ctx appctx.Context) error {
	var req CreateWorkspaceRequest
	if err := request.BindJSON(ctx, &req); err != nil {
		return err
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.create_workspace"),
		zap.Uint("user_id", ctx.User.ID),
	)

	tenantUID := strings.TrimSpace(req.TenantUID)
	if tenantUID == "" {
		return ctx.JSONError(40000, "租户标识不能为空")
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return ctx.JSONError(40000, "工作区名称不能为空")
	}

	tenant, err := repository.Tenants.GetByUID(ctx.Request().Context(), tenantUID)
	if err != nil {
		if errors.Is(err, repository.ErrTenantNotExist) {
			return ctx.JSONError(40400, "租户不存在")
		}
		logger.Error("failed to load tenant", zap.Error(err), zap.String("tenant_uid", tenantUID))
		return ctx.JSONError(50000, "获取租户失败")
	}

	workspace, err := repository.Workspaces.CreateForTenantMember(ctx.Request().Context(), repository.CreateWorkspaceOptions{
		TenantID:    tenant.ID,
		ActorUserID: ctx.User.ID,
		Name:        name,
		Description: strings.TrimSpace(req.Description),
	})
	if err != nil {
		if errors.Is(err, repository.ErrTenantAccessDenied) {
			return ctx.JSONError(40300, "无权创建该租户下的工作区")
		}
		logger.Error("failed to create workspace", zap.Error(err), zap.String("tenant_uid", tenantUID))
		return ctx.JSONError(50000, "创建工作区失败")
	}

	membership, err := repository.Tenants.GetMembership(ctx.Request().Context(), tenant.ID, ctx.User.ID)
	if err != nil {
		logger.Error("failed to load tenant membership after workspace creation", zap.Error(err))
		return ctx.JSONError(50000, "创建工作区成功，但获取成员信息失败")
	}

	return ctx.JSON(CreateWorkspaceResponse{
		Success: true,
		Message: "创建工作区成功",
		Workspace: WorkspaceSummary{
			ID:          workspace.ID,
			UID:         workspace.UID,
			Name:        workspace.Name,
			Description: workspace.Description,
			IsDefault:   workspace.IsDefault,
			CreatedAt:   workspace.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Tenant: TenantSummary{
				UID:        tenant.UID,
				Name:       tenant.Name,
				Plan:       string(tenant.Plan),
				Role:       string(membership.Role),
				IsPersonal: tenant.IsPersonal,
				CreatedAt:  tenant.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			},
		},
	})
}

type AuditLogEntry struct {
	Action       string                 `json:"action"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	ActorUserID  uint                   `json:"actor_user_id"`
	CreatedAt    string                 `json:"created_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type ListAuditLogsResponse struct {
	Success bool            `json:"success"`
	Tenant  TenantSummary   `json:"tenant"`
	Logs    []AuditLogEntry `json:"logs"`
}

func ListTenantAuditLogs(ctx appctx.Context) error {
	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.list_tenant_audit_logs"),
		zap.Uint("user_id", ctx.User.ID),
	)

	tenantUID := strings.TrimSpace(ctx.Param("tenantUID"))
	if tenantUID == "" {
		return ctx.JSONError(40000, "租户标识不能为空")
	}

	tenant, err := repository.Tenants.GetByUID(ctx.Request().Context(), tenantUID)
	if err != nil {
		if errors.Is(err, repository.ErrTenantNotExist) {
			return ctx.JSONError(40400, "租户不存在")
		}
		logger.Error("failed to load tenant for audit logs", zap.Error(err), zap.String("tenant_uid", tenantUID))
		return ctx.JSONError(50000, "获取租户失败")
	}

	membership, err := repository.Tenants.GetMembership(ctx.Request().Context(), tenant.ID, ctx.User.ID)
	if err != nil {
		if errors.Is(err, repository.ErrTenantMembershipNotExists) {
			return ctx.JSONError(40300, "无权查看该租户的审计日志")
		}
		logger.Error("failed to load tenant membership for audit logs", zap.Error(err), zap.String("tenant_uid", tenantUID))
		return ctx.JSONError(50000, "获取租户成员信息失败")
	}
	if !membership.Role.CanViewAuditLogs() {
		return ctx.JSONError(40300, "当前角色无权查看审计日志")
	}

	limit := ctx.QueryInt("limit", 50)
	logs, err := repository.AuditLogs.ListByTenantID(ctx.Request().Context(), tenant.ID, limit)
	if err != nil {
		logger.Error("failed to list tenant audit logs", zap.Error(err), zap.String("tenant_uid", tenantUID))
		return ctx.JSONError(50000, "获取审计日志失败")
	}

	entries := make([]AuditLogEntry, 0, len(logs))
	for _, auditLog := range logs {
		entries = append(entries, AuditLogEntry{
			Action:       auditLog.Action,
			ResourceType: auditLog.ResourceType,
			ResourceID:   auditLog.ResourceID,
			ActorUserID:  auditLog.ActorUserID,
			CreatedAt:    auditLog.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Metadata:     parseAuditMetadata(auditLog.Metadata),
		})
	}

	return ctx.JSON(ListAuditLogsResponse{
		Success: true,
		Tenant: TenantSummary{
			UID:        tenant.UID,
			Name:       tenant.Name,
			Plan:       string(tenant.Plan),
			Role:       string(membership.Role),
			IsPersonal: tenant.IsPersonal,
			CreatedAt:  tenant.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
		Logs: entries,
	})
}

func parseAuditMetadata(raw string) map[string]interface{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
		return map[string]interface{}{
			"raw": raw,
		}
	}
	return metadata
}
