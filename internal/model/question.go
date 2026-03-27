package model

import (
	"time"

	"github.com/syt3s/TreeBox/internal/dbutil"
)

type Question struct {
	dbutil.Model
	FromIP            string     `json:"-"`
	UserID            uint       `gorm:"index:idx_question_user_id" json:"-"`
	Content           string     `json:"content"`
	Token             string     `json:"-"`
	Answer            string     `json:"answer"`
	ReceiveReplyEmail string     `json:"-"`
	AskerUserID       uint       `json:"-"`
	IsPrivate         bool       `gorm:"default: FALSE; NOT NULL" json:"is_private"`
	ViewedAt          *time.Time `gorm:"index" json:"viewed_at,omitempty"`
}
