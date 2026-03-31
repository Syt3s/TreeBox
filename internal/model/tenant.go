package model

import (
	"github.com/rs/xid"
	"gorm.io/gorm"

	"github.com/syt3s/TreeBox/internal/dbutil"
)

type Tenant struct {
	dbutil.Model
	UID         string     `gorm:"size:32;uniqueIndex;not null" json:"uid"`
	Name        string     `gorm:"size:128;not null" json:"name"`
	Plan        TenantPlan `gorm:"size:32;not null;default:free" json:"plan"`
	OwnerUserID uint       `gorm:"index;not null" json:"owner_user_id"`
	IsPersonal  bool       `gorm:"not null;default:false" json:"is_personal"`
}

func (t *Tenant) BeforeCreate(_ *gorm.DB) error {
	t.UID = xid.New().String()
	return nil
}

type TenantPlan string

const (
	TenantPlanFree       TenantPlan = "free"
	TenantPlanPro        TenantPlan = "pro"
	TenantPlanEnterprise TenantPlan = "enterprise"
)

type TenantRole string

const (
	TenantRoleOwner  TenantRole = "owner"
	TenantRoleAdmin  TenantRole = "admin"
	TenantRoleMember TenantRole = "member"
	TenantRoleViewer TenantRole = "viewer"
)

func (r TenantRole) CanManageWorkspace() bool {
	return r == TenantRoleOwner || r == TenantRoleAdmin
}

func (r TenantRole) CanViewAuditLogs() bool {
	return r == TenantRoleOwner || r == TenantRoleAdmin
}

func (r TenantRole) CanManageMembers() bool {
	return r == TenantRoleOwner || r == TenantRoleAdmin
}

func (r TenantRole) CanManageQuestions() bool {
	return r == TenantRoleOwner || r == TenantRoleAdmin || r == TenantRoleMember
}

type TenantMember struct {
	dbutil.Model
	TenantID uint       `gorm:"index;uniqueIndex:idx_tenant_members_tenant_user,priority:1;not null" json:"tenant_id"`
	UserID   uint       `gorm:"index;uniqueIndex:idx_tenant_members_tenant_user,priority:2;not null" json:"user_id"`
	Role     TenantRole `gorm:"size:32;not null;default:member" json:"role"`
}
