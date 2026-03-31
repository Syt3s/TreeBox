package repository

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/syt3s/TreeBox/internal/model"
)

var Tenants TenantRepository

type TenantRepository interface {
	GetByID(ctx context.Context, id uint) (*model.Tenant, error)
	GetByUID(ctx context.Context, uid string) (*model.Tenant, error)
	GetMembership(ctx context.Context, tenantID, userID uint) (*model.TenantMember, error)
	ListByUserID(ctx context.Context, userID uint) ([]*TenantMembership, error)
	ListMembers(ctx context.Context, tenantID uint) ([]*TenantMemberAccess, error)
	AddMember(ctx context.Context, opts AddTenantMemberOptions) (*TenantMemberAccess, error)
	UpdateMemberRole(ctx context.Context, opts UpdateTenantMemberRoleOptions) (*TenantMemberAccess, error)
	RemoveMember(ctx context.Context, opts RemoveTenantMemberOptions) error
}

type TenantMembership struct {
	Tenant *model.Tenant
	Role   model.TenantRole
}

type TenantMemberAccess struct {
	Membership *model.TenantMember
	User       *model.User
}

type AddTenantMemberOptions struct {
	TenantID      uint
	ActorUserID   uint
	MemberUserID  uint
	Role          model.TenantRole
}

type UpdateTenantMemberRoleOptions struct {
	TenantID      uint
	ActorUserID   uint
	MemberUserID  uint
	Role          model.TenantRole
}

type RemoveTenantMemberOptions struct {
	TenantID     uint
	ActorUserID  uint
	MemberUserID uint
}

var (
	ErrTenantNotExist            = errors.New("tenant does not exist")
	ErrTenantMembershipNotExists = errors.New("tenant membership does not exist")
	ErrTenantAccessDenied        = errors.New("tenant access denied")
	ErrTenantMemberAlreadyExists = errors.New("tenant member already exists")
	ErrTenantRoleInvalid         = errors.New("tenant role is invalid")
	ErrTenantOwnerImmutable      = errors.New("tenant owner membership cannot be modified")
)

func NewTenantRepository(db *gorm.DB) TenantRepository {
	return &tenantsRepository{db}
}

type tenantsRepository struct {
	*gorm.DB
}

func (db *tenantsRepository) getBy(ctx context.Context, where string, args ...interface{}) (*model.Tenant, error) {
	var tenant model.Tenant
	if err := db.WithContext(ctx).Where(where, args...).First(&tenant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTenantNotExist
		}
		return nil, errors.Wrap(err, "get tenant")
	}
	return &tenant, nil
}

func (db *tenantsRepository) GetByID(ctx context.Context, id uint) (*model.Tenant, error) {
	return db.getBy(ctx, "id = ?", id)
}

func (db *tenantsRepository) GetByUID(ctx context.Context, uid string) (*model.Tenant, error) {
	return db.getBy(ctx, "uid = ?", uid)
}

func (db *tenantsRepository) GetMembership(ctx context.Context, tenantID, userID uint) (*model.TenantMember, error) {
	var membership model.TenantMember
	if err := db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		First(&membership).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTenantMembershipNotExists
		}
		return nil, errors.Wrap(err, "get tenant membership")
	}
	return &membership, nil
}

func (db *tenantsRepository) ListByUserID(ctx context.Context, userID uint) ([]*TenantMembership, error) {
	memberships, err := listTenantMembersByUserID(ctx, db.DB, userID)
	if err != nil {
		return nil, err
	}
	if len(memberships) == 0 {
		return []*TenantMembership{}, nil
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
		Order("created_at ASC").
		Find(&tenants).
		Error; err != nil {
		return nil, errors.Wrap(err, "list tenants by user id")
	}

	results := make([]*TenantMembership, 0, len(tenants))
	for _, tenant := range tenants {
		results = append(results, &TenantMembership{
			Tenant: tenant,
			Role:   roleByTenantID[tenant.ID],
		})
	}

	return results, nil
}

func (db *tenantsRepository) ListMembers(ctx context.Context, tenantID uint) ([]*TenantMemberAccess, error) {
	var memberships []*model.TenantMember
	if err := db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("created_at ASC").
		Find(&memberships).
		Error; err != nil {
		return nil, errors.Wrap(err, "list tenant members")
	}
	if len(memberships) == 0 {
		return []*TenantMemberAccess{}, nil
	}

	userIDs := make([]uint, 0, len(memberships))
	for _, membership := range memberships {
		userIDs = append(userIDs, membership.UserID)
	}

	var users []*model.User
	if err := db.WithContext(ctx).
		Where("id IN ?", userIDs).
		Find(&users).
		Error; err != nil {
		return nil, errors.Wrap(err, "list tenant member users")
	}

	userByID := make(map[uint]*model.User, len(users))
	for _, user := range users {
		userByID[user.ID] = user
	}

	accessList := make([]*TenantMemberAccess, 0, len(memberships))
	for _, membership := range memberships {
		user := userByID[membership.UserID]
		if user == nil {
			continue
		}
		accessList = append(accessList, &TenantMemberAccess{
			Membership: membership,
			User:       user,
		})
	}

	return accessList, nil
}

func (db *tenantsRepository) AddMember(ctx context.Context, opts AddTenantMemberOptions) (*TenantMemberAccess, error) {
	if err := validateTenantMemberRole(opts.Role, false); err != nil {
		return nil, err
	}

	var access *TenantMemberAccess
	if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		actorMembership, err := ensureTenantActorCanManageMembers(ctx, tx, opts.TenantID, opts.ActorUserID)
		if err != nil {
			return err
		}
		if actorMembership.Role != model.TenantRoleOwner && opts.Role == model.TenantRoleAdmin {
			return ErrTenantAccessDenied
		}

		if _, err := (&tenantsRepository{tx}).GetMembership(ctx, opts.TenantID, opts.MemberUserID); err == nil {
			return ErrTenantMemberAlreadyExists
		} else if !errors.Is(err, ErrTenantMembershipNotExists) {
			return err
		}

		membership := &model.TenantMember{
			TenantID: opts.TenantID,
			UserID:   opts.MemberUserID,
			Role:     opts.Role,
		}
		if err := tx.WithContext(ctx).Create(membership).Error; err != nil {
			return errors.Wrap(err, "create tenant member")
		}

		user, err := (&usersRepository{tx}).GetByID(ctx, opts.MemberUserID)
		if err != nil {
			return err
		}

		if err := recordAuditLog(ctx, tx, RecordAuditLogOptions{
			TenantID:     opts.TenantID,
			ActorUserID:  opts.ActorUserID,
			Action:       "tenant.member.added",
			ResourceType: "tenant_member",
			ResourceID:   membershipRoleResourceID(opts.TenantID, opts.MemberUserID),
			Metadata: map[string]interface{}{
				"member_user_id": opts.MemberUserID,
				"member_email":   user.Email,
				"role":           string(opts.Role),
			},
		}); err != nil {
			return err
		}

		access = &TenantMemberAccess{
			Membership: membership,
			User:       user,
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return access, nil
}

func (db *tenantsRepository) UpdateMemberRole(ctx context.Context, opts UpdateTenantMemberRoleOptions) (*TenantMemberAccess, error) {
	if err := validateTenantMemberRole(opts.Role, false); err != nil {
		return nil, err
	}

	var access *TenantMemberAccess
	if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		actorMembership, err := ensureTenantActorCanManageMembers(ctx, tx, opts.TenantID, opts.ActorUserID)
		if err != nil {
			return err
		}

		targetMembership, err := (&tenantsRepository{tx}).GetMembership(ctx, opts.TenantID, opts.MemberUserID)
		if err != nil {
			return err
		}
		if targetMembership.Role == model.TenantRoleOwner {
			return ErrTenantOwnerImmutable
		}
		if actorMembership.Role != model.TenantRoleOwner && opts.Role == model.TenantRoleAdmin {
			return ErrTenantAccessDenied
		}

		if err := tx.WithContext(ctx).
			Model(&model.TenantMember{}).
			Where("id = ?", targetMembership.ID).
			Update("role", opts.Role).
			Error; err != nil {
			return errors.Wrap(err, "update tenant member role")
		}
		targetMembership.Role = opts.Role

		user, err := (&usersRepository{tx}).GetByID(ctx, opts.MemberUserID)
		if err != nil {
			return err
		}

		if err := recordAuditLog(ctx, tx, RecordAuditLogOptions{
			TenantID:     opts.TenantID,
			ActorUserID:  opts.ActorUserID,
			Action:       "tenant.member.role_updated",
			ResourceType: "tenant_member",
			ResourceID:   membershipRoleResourceID(opts.TenantID, opts.MemberUserID),
			Metadata: map[string]interface{}{
				"member_user_id": opts.MemberUserID,
				"member_email":   user.Email,
				"role":           string(opts.Role),
			},
		}); err != nil {
			return err
		}

		access = &TenantMemberAccess{
			Membership: targetMembership,
			User:       user,
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return access, nil
}

func (db *tenantsRepository) RemoveMember(ctx context.Context, opts RemoveTenantMemberOptions) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := ensureTenantActorCanManageMembers(ctx, tx, opts.TenantID, opts.ActorUserID); err != nil {
			return err
		}

		targetMembership, err := (&tenantsRepository{tx}).GetMembership(ctx, opts.TenantID, opts.MemberUserID)
		if err != nil {
			return err
		}
		if targetMembership.Role == model.TenantRoleOwner {
			return ErrTenantOwnerImmutable
		}

		user, err := (&usersRepository{tx}).GetByID(ctx, opts.MemberUserID)
		if err != nil {
			return err
		}

		if err := tx.WithContext(ctx).Delete(&model.TenantMember{}, targetMembership.ID).Error; err != nil {
			return errors.Wrap(err, "delete tenant member")
		}

		if err := recordAuditLog(ctx, tx, RecordAuditLogOptions{
			TenantID:     opts.TenantID,
			ActorUserID:  opts.ActorUserID,
			Action:       "tenant.member.removed",
			ResourceType: "tenant_member",
			ResourceID:   membershipRoleResourceID(opts.TenantID, opts.MemberUserID),
			Metadata: map[string]interface{}{
				"member_user_id": opts.MemberUserID,
				"member_email":   user.Email,
			},
		}); err != nil {
			return err
		}

		return nil
	})
}

func listTenantMembersByUserID(ctx context.Context, db *gorm.DB, userID uint) ([]*model.TenantMember, error) {
	var memberships []*model.TenantMember
	if err := db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at ASC").
		Find(&memberships).
		Error; err != nil {
		return nil, errors.Wrap(err, "list tenant memberships by user id")
	}
	return memberships, nil
}

func ensureTenantActorCanManageMembers(ctx context.Context, tx *gorm.DB, tenantID, actorUserID uint) (*model.TenantMember, error) {
	actorMembership, err := (&tenantsRepository{tx}).GetMembership(ctx, tenantID, actorUserID)
	if err != nil {
		if errors.Is(err, ErrTenantMembershipNotExists) {
			return nil, ErrTenantAccessDenied
		}
		return nil, err
	}
	if !actorMembership.Role.CanManageMembers() {
		return nil, ErrTenantAccessDenied
	}
	return actorMembership, nil
}

func validateTenantMemberRole(role model.TenantRole, allowOwner bool) error {
	switch role {
	case model.TenantRoleAdmin, model.TenantRoleMember, model.TenantRoleViewer:
		return nil
	case model.TenantRoleOwner:
		if allowOwner {
			return nil
		}
	}
	return ErrTenantRoleInvalid
}

func membershipRoleResourceID(tenantID, userID uint) string {
	return fmt.Sprintf("%d:%d", tenantID, userID)
}
