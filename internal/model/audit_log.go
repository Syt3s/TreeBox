package model

import "github.com/syt3s/TreeBox/internal/dbutil"

type AuditLog struct {
	dbutil.Model
	TenantID     uint   `gorm:"index;not null" json:"tenant_id"`
	WorkspaceID  *uint  `gorm:"index" json:"workspace_id,omitempty"`
	ActorUserID  uint   `gorm:"index;not null" json:"actor_user_id"`
	Action       string `gorm:"size:64;index;not null" json:"action"`
	ResourceType string `gorm:"size:64;index;not null" json:"resource_type"`
	ResourceID   string `gorm:"size:64;index;not null" json:"resource_id"`
	Metadata     string `gorm:"type:text" json:"metadata,omitempty"`
}
