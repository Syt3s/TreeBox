package api

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/syt3s/TreeBox/internal/http/appctx"
	"github.com/syt3s/TreeBox/internal/http/request"
	"github.com/syt3s/TreeBox/internal/logging"
	"github.com/syt3s/TreeBox/internal/model"
	"github.com/syt3s/TreeBox/internal/repository"
)

type TenantMemberSummary struct {
	UserID   uint   `json:"user_id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Domain   string `json:"domain"`
	Role     string `json:"role"`
	JoinedAt string `json:"joined_at"`
}

type ListTenantMembersResponse struct {
	Success bool                  `json:"success"`
	Tenant  TenantSummary         `json:"tenant"`
	Members []TenantMemberSummary `json:"members"`
}

func ListTenantMembers(ctx appctx.Context) error {
	tenant, membership, err := loadTenantMembership(ctx)
	if err != nil {
		return err
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.list_tenant_members"),
		zap.Uint("tenant_id", tenant.ID),
		zap.Uint("user_id", ctx.User.ID),
	)

	members, err := repository.Tenants.ListMembers(ctx.Request().Context(), tenant.ID)
	if err != nil {
		logger.Error("failed to list tenant members", zap.Error(err))
		return ctx.JSONError(50000, "获取成员列表失败")
	}

	return ctx.JSON(ListTenantMembersResponse{
		Success: true,
		Tenant:  buildTenantSummary(tenant, membership.Role),
		Members: buildTenantMemberSummaries(members),
	})
}

type AddTenantMemberRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type AddTenantMemberResponse struct {
	Success bool                `json:"success"`
	Message string              `json:"message,omitempty"`
	Member  TenantMemberSummary `json:"member"`
}

func AddTenantMember(ctx appctx.Context) error {
	var req AddTenantMemberRequest
	if err := request.BindJSON(ctx, &req); err != nil {
		return err
	}

	tenant, membership, err := loadTenantMembership(ctx)
	if err != nil {
		return err
	}
	if !membership.Role.CanManageMembers() {
		return ctx.JSONError(40300, "当前角色无权管理成员")
	}

	role, err := parseMutableTenantRole(req.Role)
	if err != nil {
		return ctx.JSONError(40000, "成员角色无效")
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.add_tenant_member"),
		zap.Uint("tenant_id", tenant.ID),
		zap.Uint("user_id", ctx.User.ID),
	)

	email := strings.TrimSpace(req.Email)
	if email == "" {
		return ctx.JSONError(40000, "成员邮箱不能为空")
	}

	user, err := repository.Users.GetByEmail(ctx.Request().Context(), email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotExists) {
			return ctx.JSONError(40400, "该邮箱对应的用户不存在")
		}
		logger.Error("failed to load member user by email", zap.Error(err), zap.String("email", email))
		return ctx.JSONError(50000, "获取成员用户失败")
	}

	memberAccess, err := repository.Tenants.AddMember(ctx.Request().Context(), repository.AddTenantMemberOptions{
		TenantID:     tenant.ID,
		ActorUserID:  ctx.User.ID,
		MemberUserID: user.ID,
		Role:         role,
	})
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrTenantMemberAlreadyExists):
			return ctx.JSONError(40900, "该用户已经是租户成员")
		case errors.Is(err, repository.ErrTenantAccessDenied):
			return ctx.JSONError(40300, "当前角色无权管理成员")
		case errors.Is(err, repository.ErrTenantRoleInvalid):
			return ctx.JSONError(40000, "成员角色无效")
		default:
			logger.Error("failed to add tenant member", zap.Error(err), zap.String("email", email))
			return ctx.JSONError(50000, "添加成员失败")
		}
	}

	return ctx.JSON(AddTenantMemberResponse{
		Success: true,
		Message: "添加成员成功",
		Member:  buildTenantMemberSummary(memberAccess),
	})
}

type UpdateTenantMemberRoleRequest struct {
	Role string `json:"role"`
}

type UpdateTenantMemberRoleResponse struct {
	Success bool                `json:"success"`
	Message string              `json:"message,omitempty"`
	Member  TenantMemberSummary `json:"member"`
}

func UpdateTenantMemberRole(ctx appctx.Context) error {
	var req UpdateTenantMemberRoleRequest
	if err := request.BindJSON(ctx, &req); err != nil {
		return err
	}

	tenant, membership, err := loadTenantMembership(ctx)
	if err != nil {
		return err
	}
	if !membership.Role.CanManageMembers() {
		return ctx.JSONError(40300, "当前角色无权管理成员")
	}

	role, err := parseMutableTenantRole(req.Role)
	if err != nil {
		return ctx.JSONError(40000, "成员角色无效")
	}

	memberUserID, err := parseTenantMemberUserID(ctx)
	if err != nil {
		return err
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.update_tenant_member_role"),
		zap.Uint("tenant_id", tenant.ID),
		zap.Uint("user_id", ctx.User.ID),
		zap.Uint("member_user_id", memberUserID),
	)

	memberAccess, err := repository.Tenants.UpdateMemberRole(ctx.Request().Context(), repository.UpdateTenantMemberRoleOptions{
		TenantID:     tenant.ID,
		ActorUserID:  ctx.User.ID,
		MemberUserID: memberUserID,
		Role:         role,
	})
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrTenantAccessDenied):
			return ctx.JSONError(40300, "当前角色无权管理成员")
		case errors.Is(err, repository.ErrTenantRoleInvalid):
			return ctx.JSONError(40000, "成员角色无效")
		case errors.Is(err, repository.ErrTenantMembershipNotExists):
			return ctx.JSONError(40400, "成员不存在")
		case errors.Is(err, repository.ErrTenantOwnerImmutable):
			return ctx.JSONError(40000, "租户所有者角色不能修改")
		default:
			logger.Error("failed to update tenant member role", zap.Error(err))
			return ctx.JSONError(50000, "更新成员角色失败")
		}
	}

	return ctx.JSON(UpdateTenantMemberRoleResponse{
		Success: true,
		Message: "更新成员角色成功",
		Member:  buildTenantMemberSummary(memberAccess),
	})
}

type RemoveTenantMemberResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func RemoveTenantMember(ctx appctx.Context) error {
	tenant, membership, err := loadTenantMembership(ctx)
	if err != nil {
		return err
	}
	if !membership.Role.CanManageMembers() {
		return ctx.JSONError(40300, "当前角色无权管理成员")
	}

	memberUserID, err := parseTenantMemberUserID(ctx)
	if err != nil {
		return err
	}

	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.remove_tenant_member"),
		zap.Uint("tenant_id", tenant.ID),
		zap.Uint("user_id", ctx.User.ID),
		zap.Uint("member_user_id", memberUserID),
	)

	if err := repository.Tenants.RemoveMember(ctx.Request().Context(), repository.RemoveTenantMemberOptions{
		TenantID:     tenant.ID,
		ActorUserID:  ctx.User.ID,
		MemberUserID: memberUserID,
	}); err != nil {
		switch {
		case errors.Is(err, repository.ErrTenantAccessDenied):
			return ctx.JSONError(40300, "当前角色无权管理成员")
		case errors.Is(err, repository.ErrTenantMembershipNotExists):
			return ctx.JSONError(40400, "成员不存在")
		case errors.Is(err, repository.ErrTenantOwnerImmutable):
			return ctx.JSONError(40000, "租户所有者不能移除")
		default:
			logger.Error("failed to remove tenant member", zap.Error(err))
			return ctx.JSONError(50000, "移除成员失败")
		}
	}

	return ctx.JSON(RemoveTenantMemberResponse{
		Success: true,
		Message: "移除成员成功",
	})
}

func loadTenantMembership(ctx appctx.Context) (*model.Tenant, *model.TenantMember, error) {
	tenantUID := strings.TrimSpace(ctx.Param("tenantUID"))
	if tenantUID == "" {
		return nil, nil, ctx.JSONError(40000, "租户标识不能为空")
	}

	tenant, err := repository.Tenants.GetByUID(ctx.Request().Context(), tenantUID)
	if err != nil {
		if errors.Is(err, repository.ErrTenantNotExist) {
			return nil, nil, ctx.JSONError(40400, "租户不存在")
		}
		return nil, nil, ctx.JSONError(50000, "获取租户失败")
	}

	membership, err := repository.Tenants.GetMembership(ctx.Request().Context(), tenant.ID, ctx.User.ID)
	if err != nil {
		if errors.Is(err, repository.ErrTenantMembershipNotExists) {
			return nil, nil, ctx.JSONError(40300, "无权访问该租户")
		}
		return nil, nil, ctx.JSONError(50000, "获取租户成员信息失败")
	}

	return tenant, membership, nil
}

func parseMutableTenantRole(raw string) (model.TenantRole, error) {
	switch model.TenantRole(strings.TrimSpace(raw)) {
	case model.TenantRoleAdmin:
		return model.TenantRoleAdmin, nil
	case model.TenantRoleMember:
		return model.TenantRoleMember, nil
	case model.TenantRoleViewer:
		return model.TenantRoleViewer, nil
	default:
		return "", repository.ErrTenantRoleInvalid
	}
}

func parseTenantMemberUserID(ctx appctx.Context) (uint, error) {
	memberUserID, err := strconv.ParseUint(strings.TrimSpace(ctx.Param("memberUserID")), 10, 64)
	if err != nil {
		return 0, ctx.JSONError(40000, "成员编号无效")
	}
	return uint(memberUserID), nil
}

func buildTenantSummary(tenant *model.Tenant, role model.TenantRole) TenantSummary {
	return TenantSummary{
		UID:        tenant.UID,
		Name:       tenant.Name,
		Plan:       string(tenant.Plan),
		Role:       string(role),
		IsPersonal: tenant.IsPersonal,
		CreatedAt:  tenant.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func buildTenantMemberSummary(access *repository.TenantMemberAccess) TenantMemberSummary {
	return TenantMemberSummary{
		UserID:   access.User.ID,
		Name:     access.User.Name,
		Email:    access.User.Email,
		Domain:   access.User.Domain,
		Role:     string(access.Membership.Role),
		JoinedAt: access.Membership.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func buildTenantMemberSummaries(members []*repository.TenantMemberAccess) []TenantMemberSummary {
	summaries := make([]TenantMemberSummary, 0, len(members))
	for _, member := range members {
		summaries = append(summaries, buildTenantMemberSummary(member))
	}
	return summaries
}
