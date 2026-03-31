package model

import (
	"time"

	"github.com/syt3s/TreeBox/internal/dbutil"
)

type Question struct {
	dbutil.Model
	FromIP            string         `json:"-"`
	TenantID          uint           `gorm:"index:idx_question_tenant_id" json:"tenant_id,omitempty"`
	WorkspaceID       uint           `gorm:"index:idx_question_workspace_id" json:"workspace_id,omitempty"`
	UserID            uint           `gorm:"index:idx_question_user_id" json:"-"`
	Content           string         `json:"content"`
	Token             string         `json:"-"`
	Answer            string         `json:"answer"`
	ReceiveReplyEmail string         `json:"-"`
	AskerUserID       uint           `json:"-"`
	Status            QuestionStatus `gorm:"size:32;not null;default:new;index" json:"status"`
	AssignedToUserID  *uint          `gorm:"index" json:"assigned_to_user_id,omitempty"`
	InternalNote      string         `gorm:"type:text" json:"internal_note,omitempty"`
	IsPrivate         bool           `gorm:"default: FALSE; NOT NULL" json:"is_private"`
	ViewedAt          *time.Time     `gorm:"index" json:"viewed_at,omitempty"`
	ResolvedAt        *time.Time     `gorm:"index" json:"resolved_at,omitempty"`
}

type QuestionStatus string

const (
	QuestionStatusNew        QuestionStatus = "new"
	QuestionStatusInProgress QuestionStatus = "in_progress"
	QuestionStatusAnswered   QuestionStatus = "answered"
	QuestionStatusClosed     QuestionStatus = "closed"
)

func (s QuestionStatus) IsValid() bool {
	switch s {
	case QuestionStatusNew, QuestionStatusInProgress, QuestionStatusAnswered, QuestionStatusClosed:
		return true
	default:
		return false
	}
}

func (s QuestionStatus) IsResolved() bool {
	return s == QuestionStatusAnswered || s == QuestionStatusClosed
}
