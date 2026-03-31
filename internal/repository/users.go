// Copyright 2022 E99p1ant. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repository

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/syt3s/TreeBox/internal/model"
)

var Users UserRepository

var _ UserRepository = (*usersRepository)(nil)

type UserRepository interface {
	Create(ctx context.Context, opts CreateUserOptions) error
	Register(ctx context.Context, opts RegisterUserOptions) (*RegisterUserResult, error)
	EnsureTenantBootstrap(ctx context.Context, userID uint) (*TenantBootstrapResult, error)
	GetByID(ctx context.Context, id uint) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByDomain(ctx context.Context, domain string) (*model.User, error)
	Update(ctx context.Context, id uint, opts UpdateUserOptions) error
	UpdateRoutingWorkspace(ctx context.Context, id, workspaceID uint) (*model.User, error)
	UpdateHarassmentSetting(ctx context.Context, id uint, options HarassmentSettingOptions) error
	Authenticate(ctx context.Context, email, password string) (*model.User, error)
	ChangePassword(ctx context.Context, id uint, oldPassword, newPassword string) error
	UpdatePassword(ctx context.Context, id uint, newPassword string) error
	Deactivate(ctx context.Context, id uint) error
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &usersRepository{db}
}

type usersRepository struct {
	*gorm.DB
}

type CreateUserOptions struct {
	Name       string
	Password   string
	Email      string
	Avatar     string
	Domain     string
	Background string
	Intro      string
}

type RegisterUserOptions struct {
	CreateUserOptions
	TenantName    string
	WorkspaceName string
}

type RegisterUserResult struct {
	User       *model.User
	Tenant     *model.Tenant
	Membership *model.TenantMember
	Workspace  *model.Workspace
}

type TenantBootstrapResult struct {
	User       *model.User
	Tenant     *model.Tenant
	Membership *model.TenantMember
	Workspace  *model.Workspace
}

var (
	ErrUserNotExists   = errors.New("账号不存在")
	ErrBadCredential   = errors.New("邮箱或密码错误")
	ErrDuplicateEmail  = errors.New("这个邮箱已经注册过账号了")
	ErrDuplicateDomain = errors.New("个性域名重复了，换一个吧~")
)

func (db *usersRepository) Create(ctx context.Context, opts CreateUserOptions) error {
	if err := db.validate(ctx, opts); err != nil {
		return err
	}

	newUser := buildUser(opts)

	if err := db.WithContext(ctx).Create(newUser).Error; err != nil {
		return errors.Wrap(err, "create user")
	}
	return nil
}

func (db *usersRepository) Register(ctx context.Context, opts RegisterUserOptions) (*RegisterUserResult, error) {
	result := &RegisterUserResult{}

	if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := &usersRepository{tx}
		if err := repo.validate(ctx, opts.CreateUserOptions); err != nil {
			return err
		}

		user := buildUser(opts.CreateUserOptions)
		if err := tx.WithContext(ctx).Create(user).Error; err != nil {
			return errors.Wrap(err, "create user")
		}

		bootstrapResult, err := ensureTenantBootstrapTx(ctx, tx, user, tenantBootstrapOptions{
			TenantName:    opts.TenantName,
			WorkspaceName: opts.WorkspaceName,
			Action:        "tenant.bootstrap",
		})
		if err != nil {
			return err
		}

		result.User = user
		result.Tenant = bootstrapResult.Tenant
		result.Membership = bootstrapResult.Membership
		result.Workspace = bootstrapResult.Workspace
		return nil
	}); err != nil {
		return nil, err
	}

	return result, nil
}

func (db *usersRepository) EnsureTenantBootstrap(ctx context.Context, userID uint) (*TenantBootstrapResult, error) {
	var result *TenantBootstrapResult
	if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user, err := (&usersRepository{tx}).GetByID(ctx, userID)
		if err != nil {
			return errors.Wrap(err, "get user for tenant bootstrap")
		}

		result, err = ensureTenantBootstrapTx(ctx, tx, user, tenantBootstrapOptions{
			TenantName:    user.Name + " team",
			WorkspaceName: "Default workspace",
			Action:        "tenant.bootstrap.repaired",
		})
		return err
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (db *usersRepository) getBy(ctx context.Context, where string, args ...interface{}) (*model.User, error) {
	var user model.User
	if err := db.WithContext(ctx).Where(where, args...).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotExists
		}
		return nil, errors.Wrap(err, "get user")
	}
	return &user, nil
}

func (db *usersRepository) GetByID(ctx context.Context, id uint) (*model.User, error) {
	return db.getBy(ctx, "id = ?", id)
}

func (db *usersRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	return db.getBy(ctx, "email = ?", email)
}

func (db *usersRepository) GetByDomain(ctx context.Context, domain string) (*model.User, error) {
	return db.getBy(ctx, "domain = ?", domain)
}

type UpdateUserOptions struct {
	Name       string
	Avatar     string
	Background string
	Intro      string
	Notify     model.NotifyType
}

func (db *usersRepository) Update(ctx context.Context, id uint, opts UpdateUserOptions) error {
	_, err := db.GetByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "get user by id")
	}

	switch opts.Notify {
	case model.NotifyTypeEmail, model.NotifyTypeNone:
	default:
		return errors.Errorf("unexpected notify type: %q", opts.Notify)
	}

	if err := db.WithContext(ctx).Where("id = ?", id).Updates(&model.User{
		Name:       opts.Name,
		Avatar:     opts.Avatar,
		Background: opts.Background,
		Intro:      opts.Intro,
		Notify:     opts.Notify,
	}).Error; err != nil {
		return errors.Wrap(err, "update user")
	}
	return nil
}

func (db *usersRepository) UpdateRoutingWorkspace(ctx context.Context, id, workspaceID uint) (*model.User, error) {
	if _, err := db.GetByID(ctx, id); err != nil {
		return nil, errors.Wrap(err, "get user by id")
	}

	if err := db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", id).
		Update("routing_workspace_id", workspaceID).
		Error; err != nil {
		return nil, errors.Wrap(err, "update routing workspace")
	}

	updatedUser, err := db.GetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "get updated user")
	}
	return updatedUser, nil
}

type HarassmentSettingOptions struct {
	Type       model.HarassmentSettingType
	BlockWords string
}

func (db *usersRepository) UpdateHarassmentSetting(ctx context.Context, id uint, options HarassmentSettingOptions) error {
	typ := options.Type

	switch typ {
	case model.HarassmentSettingNone, model.HarassmentSettingTypeRegisterOnly:
	default:
		return errors.Errorf("unexpected harassment setting type: %q", typ)
	}

	if err := db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"HarassmentSetting": typ,
		"BlockWords":        options.BlockWords,
	}).Error; err != nil {
		return errors.Wrap(err, "update user")
	}
	return nil
}

func (db *usersRepository) Authenticate(ctx context.Context, email, password string) (*model.User, error) {
	u, err := db.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrBadCredential
	}

	if !u.Authenticate(password) {
		return nil, ErrBadCredential
	}

	if u.NeedsPasswordUpgrade() {
		upgradedPassword := password
		u.Password = upgradedPassword
		u.EncodePassword()
		if err := db.WithContext(ctx).Model(&model.User{}).Where("id = ?", u.ID).Update("password", u.Password).Error; err != nil {
			return nil, errors.Wrap(err, "upgrade password hash")
		}
	}

	return u, nil
}

func (db *usersRepository) ChangePassword(ctx context.Context, id uint, oldPassword, newPassword string) error {
	u, err := db.GetByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "get user by id")
	}

	if !u.Authenticate(oldPassword) {
		return ErrBadCredential
	}

	u.Password = newPassword
	u.EncodePassword()

	if err := db.WithContext(ctx).Model(&model.User{}).Where("id = ?", u.ID).Update("password", u.Password).Error; err != nil {
		return errors.Wrap(err, "change password")
	}
	return nil
}

func (db *usersRepository) UpdatePassword(ctx context.Context, id uint, newPassword string) error {
	u, err := db.GetByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "get user by id")
	}

	u.Password = newPassword
	u.EncodePassword()

	if err := db.WithContext(ctx).Model(&model.User{}).Where("id = ?", u.ID).Update("password", u.Password).Error; err != nil {
		return errors.Wrap(err, "change password")
	}
	return nil
}

func (db *usersRepository) Deactivate(ctx context.Context, id uint) error {
	u, err := db.GetByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "get user by id")
	}

	if err := db.WithContext(ctx).Where("id = ?", u.ID).Delete(&model.User{}).Error; err != nil {
		return errors.Wrap(err, "delete user")
	}
	return nil
}

func (db *usersRepository) validate(ctx context.Context, opts CreateUserOptions) error {
	if err := db.WithContext(ctx).Model(&model.User{}).Where("email = ?", opts.Email).First(&model.User{}).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return errors.Wrap(err, "validate email")
		}
	} else {
		return ErrDuplicateEmail
	}

	if err := db.WithContext(ctx).Model(&model.User{}).Where("domain = ?", opts.Domain).First(&model.User{}).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return errors.Wrap(err, "validate name")
		}
	} else {
		return ErrDuplicateDomain
	}

	return nil
}

func buildUser(opts CreateUserOptions) *model.User {
	newUser := &model.User{
		Name:       opts.Name,
		Password:   opts.Password,
		Email:      opts.Email,
		Avatar:     opts.Avatar,
		Domain:     opts.Domain,
		Background: opts.Background,
		Intro:      opts.Intro,
		Notify:     model.NotifyTypeEmail,
	}
	newUser.EncodePassword()
	return newUser
}

func normalizeTenantName(tenantName, userName string) string {
	tenantName = strings.TrimSpace(tenantName)
	if tenantName != "" {
		return tenantName
	}

	userName = strings.TrimSpace(userName)
	if userName == "" {
		return "Personal tenant"
	}
	return userName + " team"
}

func normalizeWorkspaceName(workspaceName string) string {
	workspaceName = strings.TrimSpace(workspaceName)
	if workspaceName != "" {
		return workspaceName
	}
	return "Default workspace"
}

type tenantBootstrapOptions struct {
	TenantName    string
	WorkspaceName string
	Action        string
}

func ensureTenantBootstrapTx(ctx context.Context, tx *gorm.DB, user *model.User, opts tenantBootstrapOptions) (*TenantBootstrapResult, error) {
	result := &TenantBootstrapResult{User: user}

	var (
		tenant            model.Tenant
		membership        model.TenantMember
		workspace         model.Workspace
		createdTenant     bool
		createdMembership bool
		createdWorkspace  bool
		updatedMembership bool
	)

	if err := tx.WithContext(ctx).
		Where("owner_user_id = ? AND is_personal = ?", user.ID, true).
		First(&tenant).
		Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrap(err, "find personal tenant")
		}

		tenant = model.Tenant{
			Name:        normalizeTenantName(opts.TenantName, user.Name),
			Plan:        model.TenantPlanFree,
			OwnerUserID: user.ID,
			IsPersonal:  true,
		}
		if err := tx.WithContext(ctx).Create(&tenant).Error; err != nil {
			return nil, errors.Wrap(err, "create personal tenant")
		}
		createdTenant = true
	}

	if err := tx.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ?", tenant.ID, user.ID).
		First(&membership).
		Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrap(err, "find tenant membership")
		}

		membership = model.TenantMember{
			TenantID: tenant.ID,
			UserID:   user.ID,
			Role:     model.TenantRoleOwner,
		}
		if err := tx.WithContext(ctx).Create(&membership).Error; err != nil {
			return nil, errors.Wrap(err, "create tenant membership")
		}
		createdMembership = true
	} else if membership.Role != model.TenantRoleOwner {
		if err := tx.WithContext(ctx).
			Model(&model.TenantMember{}).
			Where("id = ?", membership.ID).
			Update("role", model.TenantRoleOwner).
			Error; err != nil {
			return nil, errors.Wrap(err, "upgrade tenant owner membership")
		}
		membership.Role = model.TenantRoleOwner
		updatedMembership = true
	}

	if err := tx.WithContext(ctx).
		Where("tenant_id = ? AND is_default = ?", tenant.ID, true).
		Order("created_at ASC").
		First(&workspace).
		Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrap(err, "find default workspace")
		}

		workspace = model.Workspace{
			TenantID:        tenant.ID,
			Name:            normalizeWorkspaceName(opts.WorkspaceName),
			CreatedByUserID: user.ID,
			IsDefault:       true,
		}
		if err := tx.WithContext(ctx).Create(&workspace).Error; err != nil {
			return nil, errors.Wrap(err, "create default workspace")
		}
		createdWorkspace = true
	}

	if createdTenant || createdMembership || updatedMembership || createdWorkspace {
		if err := recordAuditLog(ctx, tx, RecordAuditLogOptions{
			TenantID:     tenant.ID,
			WorkspaceID:  &workspace.ID,
			ActorUserID:  user.ID,
			Action:       strings.TrimSpace(opts.Action),
			ResourceType: "tenant",
			ResourceID:   tenant.UID,
			Metadata: map[string]interface{}{
				"created_tenant":      createdTenant,
				"created_membership":  createdMembership,
				"updated_membership":  updatedMembership,
				"created_workspace":   createdWorkspace,
				"workspace_uid":       workspace.UID,
				"workspace_name":      workspace.Name,
				"is_personal":         tenant.IsPersonal,
				"bootstrap_owner_uid": user.UID,
			},
		}); err != nil {
			return nil, err
		}
	}

	result.Tenant = &tenant
	result.Membership = &membership
	result.Workspace = &workspace
	return result, nil
}
