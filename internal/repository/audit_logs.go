package repository

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/syt3s/TreeBox/internal/model"
)

var AuditLogs AuditLogRepository

type AuditLogRepository interface {
	Record(ctx context.Context, opts RecordAuditLogOptions) (*model.AuditLog, error)
	ListByTenantID(ctx context.Context, tenantID uint, limit int) ([]*model.AuditLog, error)
}

type RecordAuditLogOptions struct {
	TenantID     uint
	WorkspaceID  *uint
	ActorUserID  uint
	Action       string
	ResourceType string
	ResourceID   string
	Metadata     map[string]interface{}
}

func NewAuditLogRepository(db *gorm.DB) AuditLogRepository {
	return &auditLogsRepository{db}
}

type auditLogsRepository struct {
	*gorm.DB
}

func (db *auditLogsRepository) Record(ctx context.Context, opts RecordAuditLogOptions) (*model.AuditLog, error) {
	entry := &model.AuditLog{}
	if err := recordAuditLog(ctx, db.DB, opts, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

func (db *auditLogsRepository) ListByTenantID(ctx context.Context, tenantID uint, limit int) ([]*model.AuditLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var auditLogs []*model.AuditLog
	if err := db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("created_at DESC").
		Limit(limit).
		Find(&auditLogs).
		Error; err != nil {
		return nil, errors.Wrap(err, "list audit logs by tenant id")
	}
	return auditLogs, nil
}

func recordAuditLog(ctx context.Context, db *gorm.DB, opts RecordAuditLogOptions, targets ...*model.AuditLog) error {
	action := strings.TrimSpace(opts.Action)
	if action == "" {
		return errors.New("audit log action is required")
	}

	resourceType := strings.TrimSpace(opts.ResourceType)
	if resourceType == "" {
		return errors.New("audit log resource type is required")
	}

	resourceID := strings.TrimSpace(opts.ResourceID)
	if resourceID == "" {
		return errors.New("audit log resource id is required")
	}

	var metadata string
	if len(opts.Metadata) > 0 {
		raw, err := json.Marshal(opts.Metadata)
		if err != nil {
			return errors.Wrap(err, "marshal audit log metadata")
		}
		metadata = string(raw)
	}

	entry := &model.AuditLog{
		TenantID:     opts.TenantID,
		WorkspaceID:  opts.WorkspaceID,
		ActorUserID:  opts.ActorUserID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Metadata:     metadata,
	}
	if err := db.WithContext(ctx).Create(entry).Error; err != nil {
		return errors.Wrap(err, "record audit log")
	}

	for _, target := range targets {
		if target != nil {
			*target = *entry
		}
	}

	return nil
}
