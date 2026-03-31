package repository

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/syt3s/TreeBox/internal/model"
)

var Workspaces WorkspaceRepository

type WorkspaceRepository interface {
	GetByID(ctx context.Context, id uint) (*model.Workspace, error)
	GetByUID(ctx context.Context, uid string) (*model.Workspace, error)
	ListByUserID(ctx context.Context, userID uint) ([]*WorkspaceAccess, error)
	CreateForTenantMember(ctx context.Context, opts CreateWorkspaceOptions) (*model.Workspace, error)
}

type WorkspaceAccess struct {
	Workspace *model.Workspace
	Tenant    *model.Tenant
	Role      model.TenantRole
}

type CreateWorkspaceOptions struct {
	TenantID        uint
	ActorUserID     uint
	Name            string
	Description     string
	IsDefault       bool
	AuditActionName string
}

var ErrWorkspaceNotExist = errors.New("workspace does not exist")

func NewWorkspaceRepository(db *gorm.DB) WorkspaceRepository {
	return &workspacesRepository{db}
}

type workspacesRepository struct {
	*gorm.DB
}

func (db *workspacesRepository) getBy(ctx context.Context, where string, args ...interface{}) (*model.Workspace, error) {
	var workspace model.Workspace
	if err := db.WithContext(ctx).Where(where, args...).First(&workspace).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWorkspaceNotExist
		}
		return nil, errors.Wrap(err, "get workspace")
	}
	return &workspace, nil
}

func (db *workspacesRepository) GetByID(ctx context.Context, id uint) (*model.Workspace, error) {
	return db.getBy(ctx, "id = ?", id)
}

func (db *workspacesRepository) GetByUID(ctx context.Context, uid string) (*model.Workspace, error) {
	return db.getBy(ctx, "uid = ?", uid)
}

func (db *workspacesRepository) ListByUserID(ctx context.Context, userID uint) ([]*WorkspaceAccess, error) {
	memberships, err := listTenantMembersByUserID(ctx, db.DB, userID)
	if err != nil {
		return nil, err
	}
	if len(memberships) == 0 {
		return []*WorkspaceAccess{}, nil
	}

	tenantIDs := make([]uint, 0, len(memberships))
	roleByTenantID := make(map[uint]model.TenantRole, len(memberships))
	for _, membership := range memberships {
		tenantIDs = append(tenantIDs, membership.TenantID)
		roleByTenantID[membership.TenantID] = membership.Role
	}

	var tenants []*model.Tenant
	if err := db.WithContext(ctx).
		Where("id IN ?", tenantIDs).
		Find(&tenants).
		Error; err != nil {
		return nil, errors.Wrap(err, "list workspace tenants")
	}
	tenantByID := make(map[uint]*model.Tenant, len(tenants))
	for _, tenant := range tenants {
		tenantByID[tenant.ID] = tenant
	}

	var workspaces []*model.Workspace
	if err := db.WithContext(ctx).
		Where("tenant_id IN ?", tenantIDs).
		Order("tenant_id ASC, is_default DESC, created_at ASC").
		Find(&workspaces).
		Error; err != nil {
		return nil, errors.Wrap(err, "list workspaces by user id")
	}

	results := make([]*WorkspaceAccess, 0, len(workspaces))
	for _, workspace := range workspaces {
		tenant := tenantByID[workspace.TenantID]
		if tenant == nil {
			continue
		}
		results = append(results, &WorkspaceAccess{
			Workspace: workspace,
			Tenant:    tenant,
			Role:      roleByTenantID[workspace.TenantID],
		})
	}

	return results, nil
}

func (db *workspacesRepository) CreateForTenantMember(ctx context.Context, opts CreateWorkspaceOptions) (*model.Workspace, error) {
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		return nil, errors.New("workspace name is required")
	}

	var created *model.Workspace
	if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tenantRepo := &tenantsRepository{tx}
		membership, err := tenantRepo.GetMembership(ctx, opts.TenantID, opts.ActorUserID)
		if err != nil {
			if errors.Is(err, ErrTenantMembershipNotExists) {
				return ErrTenantAccessDenied
			}
			return err
		}
		if !membership.Role.CanManageWorkspace() {
			return ErrTenantAccessDenied
		}

		workspace := &model.Workspace{
			TenantID:        opts.TenantID,
			Name:            name,
			Description:     strings.TrimSpace(opts.Description),
			CreatedByUserID: opts.ActorUserID,
			IsDefault:       opts.IsDefault,
		}
		if err := tx.WithContext(ctx).Create(workspace).Error; err != nil {
			return errors.Wrap(err, "create workspace")
		}

		actionName := strings.TrimSpace(opts.AuditActionName)
		if actionName == "" {
			actionName = "workspace.created"
		}
		if err := recordAuditLog(ctx, tx, RecordAuditLogOptions{
			TenantID:     opts.TenantID,
			WorkspaceID:  &workspace.ID,
			ActorUserID:  opts.ActorUserID,
			Action:       actionName,
			ResourceType: "workspace",
			ResourceID:   workspace.UID,
			Metadata: map[string]interface{}{
				"name":        workspace.Name,
				"description": workspace.Description,
				"is_default":  workspace.IsDefault,
			},
		}); err != nil {
			return err
		}

		created = workspace
		return nil
	}); err != nil {
		return nil, err
	}

	return created, nil
}
