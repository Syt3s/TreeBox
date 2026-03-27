// Copyright 2022 E99p1ant. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repository

import (
	"context"
	"fmt"
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
	GetByAskUserID(ctx context.Context, userID uint, opts GetQuestionsByAskUserIDOptions) ([]*model.Question, error)
	AnswerByID(ctx context.Context, id uint, answer string) error
	MarkViewed(ctx context.Context, id uint, viewedAt time.Time) error
	DeleteByID(ctx context.Context, id uint) error
	Count(ctx context.Context, userID uint, opts GetQuestionsCountOptions) (int64, error)
	CountUnread(ctx context.Context, userID uint, showPrivate bool) (int64, error)
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
	UserID            uint
	Content           string
	ReceiveReplyEmail string
	AskerUserID       uint
	IsPrivate         bool
}

func (db *questionsRepository) Create(ctx context.Context, opts CreateQuestionOptions) (*model.Question, error) {
	question := model.Question{
		FromIP:            opts.FromIP,
		UserID:            opts.UserID,
		Token:             randstr.String(6),
		Content:           opts.Content,
		ReceiveReplyEmail: opts.ReceiveReplyEmail,
		AskerUserID:       opts.AskerUserID,
		IsPrivate:         opts.IsPrivate,
	}
	return &question, db.WithContext(ctx).Create(&question).Error
}

var ErrQuestionNotExist = errors.New("提问不存在")

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

func (db *questionsRepository) getBy(ctx context.Context, cursor *dbutil.Cursor, whereQuery string, args ...interface{}) ([]*model.Question, error) {
	var questions []*model.Question
	q := db.WithContext(ctx).Where(whereQuery, args...)

	if cursor != nil {
		cursorID := cursor.Value
		if cursorID != nil && fmt.Sprintf("%v", cursorID) != "" {
			// For we ordered by ID DESC, so we need to use `>` instead of `<`.
			q = q.Where(`id < ?`, cursorID)
		}

		limit := cursor.Limit()
		q = q.Limit(limit)
	}

	q = q.Order("created_at DESC")
	if err := q.Find(&questions).Error; err != nil {
		return nil, errors.Wrap(err, "get questions by page ID")
	}
	return questions, nil
}

type GetQuestionsByUserIDOptions struct {
	*dbutil.Cursor
	FilterAnswered bool
	ShowPrivate    bool
}

func (db *questionsRepository) GetByUserID(ctx context.Context, userID uint, opts GetQuestionsByUserIDOptions) ([]*model.Question, error) {
	where := `user_id = ?`
	args := userID

	if opts.FilterAnswered {
		where = `user_id = ? AND answer != ''`
	}
	if !opts.ShowPrivate {
		where += ` AND is_private = false`
	}

	questions, err := db.getBy(ctx, opts.Cursor, where, args)
	if err != nil {
		return nil, errors.Wrap(err, "get by")
	}
	return questions, nil
}

type GetQuestionsByAskUserIDOptions struct {
	*dbutil.Cursor
	FilterAnswered bool
	ShowPrivate    bool
}

func (db *questionsRepository) GetByAskUserID(ctx context.Context, userID uint, opts GetQuestionsByAskUserIDOptions) ([]*model.Question, error) {
	where := `asker_user_id = ?`
	args := userID

	if opts.FilterAnswered {
		where = `asker_user_id = ? AND answer != ''`
	}
	if !opts.ShowPrivate {
		where += ` AND is_private = false`
	}

	questions, err := db.getBy(ctx, opts.Cursor, where, args)
	if err != nil {
		return nil, errors.Wrap(err, "get by")
	}
	return questions, nil
}

func (db *questionsRepository) AnswerByID(ctx context.Context, id uint, answer string) error {
	var question model.Question
	if err := db.WithContext(ctx).First(&question, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrQuestionNotExist
		}
		return errors.Wrap(err, "get question by ID")
	}

	if err := db.WithContext(ctx).Model(&question).Where("id = ?", id).Update("answer", answer).Error; err != nil {
		return errors.Wrap(err, "update question answer")
	}
	return nil
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

func (db *questionsRepository) DeleteByID(ctx context.Context, id uint) error {
	var question model.Question
	if err := db.WithContext(ctx).First(&question, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrQuestionNotExist
		}
		return errors.Wrap(err, "get question by ID")
	}

	if err := db.WithContext(ctx).Delete(&model.Question{}, id).Error; err != nil {
		return errors.Wrap(err, "delete question")
	}
	return nil
}

type GetQuestionsCountOptions struct {
	FilterAnswered bool
	ShowPrivate    bool
}

func (db *questionsRepository) Count(ctx context.Context, userID uint, opts GetQuestionsCountOptions) (int64, error) {
	q := db.WithContext(ctx).Model(&model.Question{})
	if opts.FilterAnswered {
		q = q.Where(`user_id = ? AND answer != ''`, userID)
	} else {
		q = q.Where(`user_id = ?`, userID)
	}
	if !opts.ShowPrivate {
		q = q.Where(`is_private = ?`, false)
	}

	var count int64
	return count, q.Count(&count).Error
}

func (db *questionsRepository) CountUnread(ctx context.Context, userID uint, showPrivate bool) (int64, error) {
	q := db.WithContext(ctx).Model(&model.Question{}).Where(`user_id = ? AND viewed_at IS NULL`, userID)
	if !showPrivate {
		q = q.Where(`is_private = ?`, false)
	}

	var count int64
	return count, q.Count(&count).Error
}

func (db *questionsRepository) SetPrivate(ctx context.Context, id uint) error {
	return db.WithContext(ctx).Model(&model.Question{}).Where("id = ?", id).Update("is_private", true).Error
}

func (db *questionsRepository) SetPublic(ctx context.Context, id uint) error {
	return db.WithContext(ctx).Model(&model.Question{}).Where("id = ?", id).Update("is_private", false).Error
}
