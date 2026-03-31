// Copyright 2022 E99p1ant. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/thanhpk/randstr"
	"gorm.io/gorm"

	"github.com/syt3s/TreeBox/internal/dbutil"
	"github.com/syt3s/TreeBox/internal/model"
)

var Questions QuestionRepository

type QuestionRepository interface {
	Create(ctx context.Context, opts CreateQuestionOptions) (*model.Question, error)
	GetByID(ctx context.Context, id uint) (*model.Question, error)
	GetByUserID(ctx context.Context, userID uint, opts GetQuestionsByUserIDOptions) ([]*model.Question, error)
	GetByWorkspaceID(ctx context.Context, workspaceID uint, opts GetQuestionsByWorkspaceIDOptions) ([]*model.Question, error)
	GetByAskUserID(ctx context.Context, userID uint, opts GetQuestionsByAskUserIDOptions) ([]*model.Question, error)
	AnswerByID(ctx context.Context, id uint, answer string) error
	UpdateStatus(ctx context.Context, id uint, status model.QuestionStatus) (*model.Question, error)
	UpdateAssignment(ctx context.Context, id uint, assignedToUserID *uint) (*model.Question, error)
	UpdateInternalNote(ctx context.Context, id uint, internalNote string) (*model.Question, error)
	MarkViewed(ctx context.Context, id uint, viewedAt time.Time) error
	MarkAllViewed(ctx context.Context, userID uint, viewedAt time.Time) (int64, error)
	DeleteByID(ctx context.Context, id uint) error
	Count(ctx context.Context, userID uint, opts GetQuestionsCountOptions) (int64, error)
	CountUnread(ctx context.Context, userID uint, showPrivate bool) (int64, error)
	GetWorkspaceStats(ctx context.Context, workspaceID uint, opts GetWorkspaceQuestionStatsOptions) (*WorkspaceQuestionStats, error)
	SetPrivate(ctx context.Context, id uint) error
	SetPublic(ctx context.Context, id uint) error
}

func NewQuestionRepository(db *gorm.DB) QuestionRepository {
	return &questionsRepository{db}
}

type questionsRepository struct {
	*gorm.DB
}

type CreateQuestionOptions struct {
	FromIP            string
	TenantID          uint
	WorkspaceID       uint
	UserID            uint
	Content           string
	ReceiveReplyEmail string
	AskerUserID       uint
	IsPrivate         bool
}

func (db *questionsRepository) Create(ctx context.Context, opts CreateQuestionOptions) (*model.Question, error) {
	question := model.Question{
		FromIP:            opts.FromIP,
		TenantID:          opts.TenantID,
		WorkspaceID:       opts.WorkspaceID,
		UserID:            opts.UserID,
		Token:             randstr.String(6),
		Content:           opts.Content,
		ReceiveReplyEmail: opts.ReceiveReplyEmail,
		AskerUserID:       opts.AskerUserID,
		Status:            model.QuestionStatusNew,
		IsPrivate:         opts.IsPrivate,
	}
	return &question, db.WithContext(ctx).Create(&question).Error
}

var ErrQuestionNotExist = errors.New("question does not exist")

func (db *questionsRepository) GetByID(ctx context.Context, id uint) (*model.Question, error) {
	var question model.Question
	if err := db.WithContext(ctx).First(&question, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuestionNotExist
		}
		return nil, errors.Wrap(err, "get question by ID")
	}
	return &question, nil
}

func (db *questionsRepository) getBy(ctx context.Context, cursor *dbutil.Cursor, baseQuery *gorm.DB) ([]*model.Question, error) {
	var questions []*model.Question
	q := baseQuery

	if cursor != nil {
		cursorID := cursor.Value
		if cursorID != nil && fmt.Sprintf("%v", cursorID) != "" {
			// We order by ID DESC, so the next page continues with smaller IDs.
			q = q.Where("id < ?", cursorID)
		}

		limit := cursor.Limit()
		q = q.Limit(limit)
	}

	q = q.Order("created_at DESC")
	if err := q.Find(&questions).Error; err != nil {
		return nil, errors.Wrap(err, "get questions")
	}
	return questions, nil
}

type GetQuestionsByUserIDOptions struct {
	*dbutil.Cursor
	FilterAnswered bool
	ShowPrivate    bool
}

func (db *questionsRepository) GetByUserID(ctx context.Context, userID uint, opts GetQuestionsByUserIDOptions) ([]*model.Question, error) {
	q := db.WithContext(ctx).Model(&model.Question{}).Where("user_id = ?", userID)
	if opts.FilterAnswered {
		q = q.Where("answer != ''")
	}
	if !opts.ShowPrivate {
		q = q.Where("is_private = ?", false)
	}

	questions, err := db.getBy(ctx, opts.Cursor, q)
	if err != nil {
		return nil, errors.Wrap(err, "get by user")
	}
	return questions, nil
}

type GetQuestionsByWorkspaceIDOptions struct {
	*dbutil.Cursor
	FilterAnswered   bool
	ShowPrivate      bool
	Status           model.QuestionStatus
	AssignedToUserID *uint
	OnlyAssigned     bool
	OnlyUnassigned   bool
}

func (db *questionsRepository) GetByWorkspaceID(ctx context.Context, workspaceID uint, opts GetQuestionsByWorkspaceIDOptions) ([]*model.Question, error) {
	q := db.WithContext(ctx).Model(&model.Question{}).Where("workspace_id = ?", workspaceID)
	if opts.FilterAnswered {
		q = q.Where("answer != ''")
	}
	if !opts.ShowPrivate {
		q = q.Where("is_private = ?", false)
	}
	if opts.Status != "" {
		q = q.Where("status = ?", opts.Status)
	}
	if opts.AssignedToUserID != nil {
		q = q.Where("assigned_to_user_id = ?", *opts.AssignedToUserID)
	}
	if opts.OnlyAssigned {
		q = q.Where("assigned_to_user_id IS NOT NULL")
	}
	if opts.OnlyUnassigned {
		q = q.Where("assigned_to_user_id IS NULL")
	}

	questions, err := db.getBy(ctx, opts.Cursor, q)
	if err != nil {
		return nil, errors.Wrap(err, "get by workspace")
	}
	return questions, nil
}

type GetQuestionsByAskUserIDOptions struct {
	*dbutil.Cursor
	FilterAnswered bool
	ShowPrivate    bool
}

func (db *questionsRepository) GetByAskUserID(ctx context.Context, userID uint, opts GetQuestionsByAskUserIDOptions) ([]*model.Question, error) {
	q := db.WithContext(ctx).Model(&model.Question{}).Where("asker_user_id = ?", userID)
	if opts.FilterAnswered {
		q = q.Where("answer != ''")
	}
	if !opts.ShowPrivate {
		q = q.Where("is_private = ?", false)
	}

	questions, err := db.getBy(ctx, opts.Cursor, q)
	if err != nil {
		return nil, errors.Wrap(err, "get by asker")
	}
	return questions, nil
}

func (db *questionsRepository) AnswerByID(ctx context.Context, id uint, answer string) error {
	question, err := db.GetByID(ctx, id)
	if err != nil {
		return err
	}

	answer = strings.TrimSpace(answer)
	update := map[string]interface{}{
		"answer": answer,
	}

	if answer == "" {
		update["status"] = model.QuestionStatusInProgress
		update["resolved_at"] = nil
	} else {
		update["status"] = model.QuestionStatusAnswered
		update["resolved_at"] = time.Now()
	}

	if err := db.WithContext(ctx).
		Model(&model.Question{}).
		Where("id = ?", question.ID).
		Updates(update).
		Error; err != nil {
		return errors.Wrap(err, "update question answer")
	}
	return nil
}

func (db *questionsRepository) UpdateStatus(ctx context.Context, id uint, status model.QuestionStatus) (*model.Question, error) {
	if !status.IsValid() {
		return nil, errors.Errorf("unexpected question status: %q", status)
	}

	question, err := db.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	update := map[string]interface{}{
		"status": status,
	}
	if status.IsResolved() {
		update["resolved_at"] = time.Now()
	} else {
		update["resolved_at"] = nil
	}

	if err := db.WithContext(ctx).
		Model(&model.Question{}).
		Where("id = ?", question.ID).
		Updates(update).
		Error; err != nil {
		return nil, errors.Wrap(err, "update question status")
	}

	question.Status = status
	if status.IsResolved() {
		resolvedAt := time.Now()
		question.ResolvedAt = &resolvedAt
	} else {
		question.ResolvedAt = nil
	}
	return question, nil
}

func (db *questionsRepository) UpdateAssignment(ctx context.Context, id uint, assignedToUserID *uint) (*model.Question, error) {
	question, err := db.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	update := map[string]interface{}{
		"assigned_to_user_id": assignedToUserID,
	}
	if err := db.WithContext(ctx).
		Model(&model.Question{}).
		Where("id = ?", question.ID).
		Updates(update).
		Error; err != nil {
		return nil, errors.Wrap(err, "update question assignment")
	}

	question.AssignedToUserID = assignedToUserID
	return question, nil
}

func (db *questionsRepository) UpdateInternalNote(ctx context.Context, id uint, internalNote string) (*model.Question, error) {
	question, err := db.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	internalNote = strings.TrimSpace(internalNote)
	if err := db.WithContext(ctx).
		Model(&model.Question{}).
		Where("id = ?", question.ID).
		Update("internal_note", internalNote).
		Error; err != nil {
		return nil, errors.Wrap(err, "update question internal note")
	}

	question.InternalNote = internalNote
	return question, nil
}

func (db *questionsRepository) MarkViewed(ctx context.Context, id uint, viewedAt time.Time) error {
	if err := db.WithContext(ctx).
		Model(&model.Question{}).
		Where("id = ? AND viewed_at IS NULL", id).
		Update("viewed_at", viewedAt).
		Error; err != nil {
		return errors.Wrap(err, "mark question viewed")
	}
	return nil
}

func (db *questionsRepository) MarkAllViewed(ctx context.Context, userID uint, viewedAt time.Time) (int64, error) {
	result := db.WithContext(ctx).
		Model(&model.Question{}).
		Where("user_id = ? AND viewed_at IS NULL", userID).
		Update("viewed_at", viewedAt)
	if result.Error != nil {
		return 0, errors.Wrap(result.Error, "mark all questions viewed")
	}
	return result.RowsAffected, nil
}

func (db *questionsRepository) DeleteByID(ctx context.Context, id uint) error {
	question, err := db.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := db.WithContext(ctx).Delete(&model.Question{}, question.ID).Error; err != nil {
		return errors.Wrap(err, "delete question")
	}
	return nil
}

type GetQuestionsCountOptions struct {
	FilterAnswered bool
	ShowPrivate    bool
}

func (db *questionsRepository) Count(ctx context.Context, userID uint, opts GetQuestionsCountOptions) (int64, error) {
	q := db.WithContext(ctx).Model(&model.Question{}).Where("user_id = ?", userID)
	if opts.FilterAnswered {
		q = q.Where("answer != ''")
	}
	if !opts.ShowPrivate {
		q = q.Where("is_private = ?", false)
	}

	var count int64
	return count, q.Count(&count).Error
}

func (db *questionsRepository) CountUnread(ctx context.Context, userID uint, showPrivate bool) (int64, error) {
	q := db.WithContext(ctx).
		Model(&model.Question{}).
		Where("user_id = ? AND viewed_at IS NULL AND answer = ''", userID)
	if !showPrivate {
		q = q.Where("is_private = ?", false)
	}

	var count int64
	return count, q.Count(&count).Error
}

type GetWorkspaceQuestionStatsOptions struct {
	ShowPrivate bool
}

type WorkspaceQuestionStats struct {
	TotalCount      int64 `json:"total_count"`
	NewCount        int64 `json:"new_count"`
	InProgressCount int64 `json:"in_progress_count"`
	AnsweredCount   int64 `json:"answered_count"`
	ClosedCount     int64 `json:"closed_count"`
	PrivateCount    int64 `json:"private_count"`
	AssignedCount   int64 `json:"assigned_count"`
	UnassignedCount int64 `json:"unassigned_count"`
	ResolvedCount   int64 `json:"resolved_count"`
	UnresolvedCount int64 `json:"unresolved_count"`
}

func (db *questionsRepository) GetWorkspaceStats(ctx context.Context, workspaceID uint, opts GetWorkspaceQuestionStatsOptions) (*WorkspaceQuestionStats, error) {
	base := db.WithContext(ctx).Model(&model.Question{}).Where("workspace_id = ?", workspaceID)
	if !opts.ShowPrivate {
		base = base.Where("is_private = ?", false)
	}

	stats := &WorkspaceQuestionStats{}

	countWithQuery := func(q *gorm.DB) (int64, error) {
		var count int64
		if err := q.Count(&count).Error; err != nil {
			return 0, err
		}
		return count, nil
	}

	var err error
	if stats.TotalCount, err = countWithQuery(base.Session(&gorm.Session{})); err != nil {
		return nil, errors.Wrap(err, "count workspace total questions")
	}
	if stats.NewCount, err = countWithQuery(base.Session(&gorm.Session{}).Where("status = ?", model.QuestionStatusNew)); err != nil {
		return nil, errors.Wrap(err, "count workspace new questions")
	}
	if stats.InProgressCount, err = countWithQuery(base.Session(&gorm.Session{}).Where("status = ?", model.QuestionStatusInProgress)); err != nil {
		return nil, errors.Wrap(err, "count workspace in progress questions")
	}
	if stats.AnsweredCount, err = countWithQuery(base.Session(&gorm.Session{}).Where("status = ?", model.QuestionStatusAnswered)); err != nil {
		return nil, errors.Wrap(err, "count workspace answered questions")
	}
	if stats.ClosedCount, err = countWithQuery(base.Session(&gorm.Session{}).Where("status = ?", model.QuestionStatusClosed)); err != nil {
		return nil, errors.Wrap(err, "count workspace closed questions")
	}
	if stats.PrivateCount, err = countWithQuery(base.Session(&gorm.Session{}).Where("is_private = ?", true)); err != nil {
		return nil, errors.Wrap(err, "count workspace private questions")
	}
	if stats.AssignedCount, err = countWithQuery(base.Session(&gorm.Session{}).Where("assigned_to_user_id IS NOT NULL")); err != nil {
		return nil, errors.Wrap(err, "count workspace assigned questions")
	}
	if stats.UnassignedCount, err = countWithQuery(base.Session(&gorm.Session{}).Where("assigned_to_user_id IS NULL")); err != nil {
		return nil, errors.Wrap(err, "count workspace unassigned questions")
	}
	if stats.ResolvedCount, err = countWithQuery(base.Session(&gorm.Session{}).Where("status IN ?", []model.QuestionStatus{model.QuestionStatusAnswered, model.QuestionStatusClosed})); err != nil {
		return nil, errors.Wrap(err, "count workspace resolved questions")
	}
	stats.UnresolvedCount = stats.TotalCount - stats.ResolvedCount

	return stats, nil
}

func (db *questionsRepository) SetPrivate(ctx context.Context, id uint) error {
	return db.WithContext(ctx).
		Model(&model.Question{}).
		Where("id = ?", id).
		Update("is_private", true).
		Error
}

func (db *questionsRepository) SetPublic(ctx context.Context, id uint) error {
	return db.WithContext(ctx).
		Model(&model.Question{}).
		Where("id = ?", id).
		Update("is_private", false).
		Error
}
