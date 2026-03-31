package model

import (
	"github.com/rs/xid"
	"gorm.io/gorm"

	"github.com/syt3s/TreeBox/internal/dbutil"
)

type Workspace struct {
	dbutil.Model
	UID             string `gorm:"size:32;uniqueIndex;not null" json:"uid"`
	TenantID        uint   `gorm:"index;not null" json:"tenant_id"`
	Name            string `gorm:"size:128;not null" json:"name"`
	Description     string `gorm:"size:512" json:"description,omitempty"`
	CreatedByUserID uint   `gorm:"index;not null" json:"created_by_user_id"`
	IsDefault       bool   `gorm:"not null;default:false" json:"is_default"`
}

func (w *Workspace) BeforeCreate(_ *gorm.DB) error {
	w.UID = xid.New().String()
	return nil
}
