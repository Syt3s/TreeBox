// Copyright 2023 E99p1ant. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/syt3s/TreeBox/internal/dbutil"
	"github.com/syt3s/TreeBox/internal/model"
)

func TestQuestions(t *testing.T) {
	t.Parallel()

	db, cleanup := newTestDB(t)
	ctx := context.Background()

	questionsStore := NewQuestionRepository(db)

	for _, tc := range []struct {
		name string
		test func(t *testing.T, ctx context.Context, db *questionsRepository)
	}{
		{"Create", testQuestionsCreate},
		{"GetByID", testQuestionsGetByID},
		{"GetByUserID", testQuestionsGetByUserID},
		{"GetByAskUserID", testQuestionsGetByAskUserID},
		{"AnswerByID", testQuestionsAnswerByID},
		{"DeleteByID", testQuestionsDeleteByID},
		{"Count", testQuestionsCount},
		{"CountUnread", testQuestionsCountUnread},
		{"SetPrivate", testQuestionsSetPrivate},
		{"SetPublic", testQuestionsSetPublic},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() {
				if err := cleanup("questions"); err != nil {
					t.Fatal(err)
				}
			})
			tc.test(t, ctx, questionsStore.(*questionsRepository))
		})
	}
}

func testQuestionsCreate(t *testing.T, ctx context.Context, db *questionsRepository) {
	t.Run("normal", func(t *testing.T) {
		got, err := db.Create(ctx, CreateQuestionOptions{
			FromIP:            "114.5.1.4",
			UserID:            1,
			Content:           "Hello, world!",
			ReceiveReplyEmail: "i@github.red",
			AskerUserID:       0,
		})
		require.Nil(t, err)

		got.Token = ""
		got.CreatedAt = time.Time{}
		got.UpdatedAt = time.Time{}

		want := &model.Question{
			Model: dbutil.Model{
				ID: 1,
			},
			FromIP:            "114.5.1.4",
			UserID:            1,
			Content:           "Hello, world!",
			ReceiveReplyEmail: "i@github.red",
		}
		require.Equal(t, got, want)
	})
}

func testQuestionsGetByID(t *testing.T, ctx context.Context, db *questionsRepository) {
	_, err := db.Create(ctx, CreateQuestionOptions{
		FromIP:            "114.5.1.4",
		UserID:            1,
		Content:           "Hello, world!",
		ReceiveReplyEmail: "i@github.red",
		AskerUserID:       0,
	})
	require.Nil(t, err)

	t.Run("normal", func(t *testing.T) {
		got, err := db.GetByID(ctx, 1)
		require.Nil(t, err)

		got.Token = ""
		got.CreatedAt = time.Time{}
		got.UpdatedAt = time.Time{}

		want := &model.Question{
			Model: dbutil.Model{
				ID: 1,
			},
			FromIP:            "114.5.1.4",
			UserID:            1,
			Content:           "Hello, world!",
			ReceiveReplyEmail: "i@github.red",
		}
		require.Equal(t, got, want)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := db.GetByID(ctx, 404)
		require.Equal(t, ErrQuestionNotExist, err)
	})
}

func testQuestionsGetByUserID(t *testing.T, ctx context.Context, db *questionsRepository) {
	_, err := db.Create(ctx, CreateQuestionOptions{
		FromIP:            "114.5.1.4",
		UserID:            1,
		Content:           "Content - 1",
		ReceiveReplyEmail: "i@github.red",
		AskerUserID:       0,
	})
	require.Nil(t, err)

	_, err = db.Create(ctx, CreateQuestionOptions{
		FromIP:            "114.5.1.4",
		UserID:            2,
		Content:           "Content - 2",
		ReceiveReplyEmail: "i@github.red",
		AskerUserID:       0,
	})
	require.Nil(t, err)

	_, err = db.Create(ctx, CreateQuestionOptions{
		FromIP:            "114.5.1.4",
		UserID:            1,
		Content:           "Content - 3",
		ReceiveReplyEmail: "i@github.red",
		AskerUserID:       0,
	})
	require.Nil(t, err)

	t.Run("all", func(t *testing.T) {
		got, err := db.GetByUserID(ctx, 1, GetQuestionsByUserIDOptions{})
		require.Nil(t, err)

		for _, g := range got {
			g.Token = ""
			g.CreatedAt = time.Time{}
			g.UpdatedAt = time.Time{}
		}

		want := []*model.Question{
			{
				Model: dbutil.Model{
					ID: 3,
				},
				FromIP:            "114.5.1.4",
				UserID:            1,
				Content:           "Content - 3",
				ReceiveReplyEmail: "i@github.red",
			},
			{
				Model: dbutil.Model{
					ID: 1,
				},
				FromIP:            "114.5.1.4",
				UserID:            1,
				Content:           "Content - 1",
				ReceiveReplyEmail: "i@github.red",
			},
		}
		require.Equal(t, want, got)
	})

	t.Run("cursor", func(t *testing.T) {
		got, err := db.GetByUserID(ctx, 1, GetQuestionsByUserIDOptions{
			Cursor: &dbutil.Cursor{
				Value:    3,
				PageSize: 10,
			},
		})
		require.Nil(t, err)

		for _, g := range got {
			g.Token = ""
			g.CreatedAt = time.Time{}
			g.UpdatedAt = time.Time{}
		}

		want := []*model.Question{
			{
				Model: dbutil.Model{
					ID: 1,
				},
				FromIP:            "114.5.1.4",
				UserID:            1,
				Content:           "Content - 1",
				ReceiveReplyEmail: "i@github.red",
			},
		}
		require.Equal(t, want, got)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := db.GetByID(ctx, 404)
		require.Equal(t, ErrQuestionNotExist, err)
	})
}

func testQuestionsGetByAskUserID(t *testing.T, ctx context.Context, db *questionsRepository) {
	_, err := db.Create(ctx, CreateQuestionOptions{
		FromIP:            "114.5.1.4",
		UserID:            1,
		Content:           "Content - 1",
		ReceiveReplyEmail: "i@github.red",
		AskerUserID:       1,
	})
	require.Nil(t, err)

	_, err = db.Create(ctx, CreateQuestionOptions{
		FromIP:            "114.5.1.4",
		UserID:            2,
		Content:           "Content - 2",
		ReceiveReplyEmail: "i@github.red",
		AskerUserID:       2,
	})
	require.Nil(t, err)

	_, err = db.Create(ctx, CreateQuestionOptions{
		FromIP:            "114.5.1.4",
		UserID:            2,
		Content:           "Content - 2",
		ReceiveReplyEmail: "i@github.red",
		AskerUserID:       2,
	})
	require.Nil(t, err)

	t.Run("all", func(t *testing.T) {
		got, err := db.GetByAskUserID(ctx, 1, GetQuestionsByAskUserIDOptions{})
		require.Nil(t, err)

		for _, g := range got {
			g.Token = ""
			g.CreatedAt = time.Time{}
			g.UpdatedAt = time.Time{}
		}

		want := []*model.Question{
			{
				Model: dbutil.Model{
					ID: 1,
				},
				FromIP:            "114.5.1.4",
				UserID:            1,
				Content:           "Content - 1",
				ReceiveReplyEmail: "i@github.red",
				AskerUserID:       1,
			},
		}
		require.Equal(t, want, got)
	})

	t.Run("cursor", func(t *testing.T) {
		got, err := db.GetByAskUserID(ctx, 2, GetQuestionsByAskUserIDOptions{
			Cursor: &dbutil.Cursor{
				Value:    3,
				PageSize: 10,
			},
		})
		require.Nil(t, err)

		for _, g := range got {
			g.Token = ""
			g.CreatedAt = time.Time{}
			g.UpdatedAt = time.Time{}
		}

		want := []*model.Question{
			{
				Model: dbutil.Model{
					ID: 2,
				},
				FromIP:            "114.5.1.4",
				UserID:            2,
				Content:           "Content - 2",
				ReceiveReplyEmail: "i@github.red",
				AskerUserID:       2,
			},
		}
		require.Equal(t, want, got)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := db.GetByID(ctx, 404)
		require.Equal(t, ErrQuestionNotExist, err)
	})
}

func testQuestionsAnswerByID(t *testing.T, ctx context.Context, db *questionsRepository) {
	t.Run("normal", func(t *testing.T) {
		_, err := db.Create(ctx, CreateQuestionOptions{
			FromIP:            "114.5.1.4",
			UserID:            1,
			Content:           "Content - 1",
			ReceiveReplyEmail: "i@github.red",
			AskerUserID:       1,
		})
		require.Nil(t, err)

		err = db.AnswerByID(ctx, 1, "Answer - 1")
		require.Nil(t, err)

		got, err := db.GetByID(ctx, 1)
		require.Nil(t, err)
		require.Equal(t, "Answer - 1", got.Answer)
	})

	t.Run("not found", func(t *testing.T) {
		err := db.AnswerByID(ctx, 404, "Answer - 1")
		require.Equal(t, ErrQuestionNotExist, err)
	})
}

func testQuestionsDeleteByID(t *testing.T, ctx context.Context, db *questionsRepository) {
	t.Run("normal", func(t *testing.T) {
		_, err := db.Create(ctx, CreateQuestionOptions{
			FromIP:            "114.5.1.4",
			UserID:            1,
			Content:           "Content - 1",
			ReceiveReplyEmail: "i@github.red",
			AskerUserID:       1,
		})
		require.Nil(t, err)

		err = db.DeleteByID(ctx, 1)
		require.Nil(t, err)

		_, err = db.GetByID(ctx, 1)
		require.Equal(t, ErrQuestionNotExist, err)
	})

	t.Run("not found", func(t *testing.T) {
		err := db.DeleteByID(ctx, 404)
		require.Equal(t, ErrQuestionNotExist, err)
	})
}

func testQuestionsCount(t *testing.T, ctx context.Context, db *questionsRepository) {
	_, err := db.Create(ctx, CreateQuestionOptions{
		FromIP:            "114.5.1.4",
		UserID:            1,
		Content:           "Content - 1",
		ReceiveReplyEmail: "i@github.red",
		AskerUserID:       1,
	})
	require.Nil(t, err)

	_, err = db.Create(ctx, CreateQuestionOptions{
		FromIP:            "114.5.1.4",
		UserID:            2,
		Content:           "Content - 2",
		ReceiveReplyEmail: "i@github.red",
		AskerUserID:       0,
	})
	require.Nil(t, err)

	_, err = db.Create(ctx, CreateQuestionOptions{
		FromIP:            "114.5.1.4",
		UserID:            1,
		Content:           "Content - 3",
		ReceiveReplyEmail: "i@github.red",
		AskerUserID:       0,
	})
	require.Nil(t, err)

	err = db.AnswerByID(ctx, 3, "Answer - 3")
	require.Nil(t, err)

	t.Run("normal", func(t *testing.T) {
		got, err := db.Count(ctx, 1, GetQuestionsCountOptions{})
		require.Nil(t, err)

		want := int64(2)
		require.Equal(t, want, got)
	})

	t.Run("filter", func(t *testing.T) {
		got, err := db.Count(ctx, 1, GetQuestionsCountOptions{FilterAnswered: true})
		require.Nil(t, err)

		want := int64(1)
		require.Equal(t, want, got)
	})

	t.Run("not found", func(t *testing.T) {
		got, err := db.Count(ctx, 404, GetQuestionsCountOptions{})
		require.Nil(t, err)

		want := int64(0)
		require.Equal(t, want, got)
	})
}

func testQuestionsCountUnread(t *testing.T, ctx context.Context, db *questionsRepository) {
	_, err := db.Create(ctx, CreateQuestionOptions{
		FromIP:            "114.5.1.4",
		UserID:            1,
		Content:           "Content - 1",
		ReceiveReplyEmail: "i@github.red",
		AskerUserID:       1,
	})
	require.Nil(t, err)

	_, err = db.Create(ctx, CreateQuestionOptions{
		FromIP:            "114.5.1.4",
		UserID:            1,
		Content:           "Content - 2",
		ReceiveReplyEmail: "i@github.red",
		AskerUserID:       1,
	})
	require.Nil(t, err)

	_, err = db.Create(ctx, CreateQuestionOptions{
		FromIP:            "114.5.1.4",
		UserID:            1,
		Content:           "Content - 3",
		ReceiveReplyEmail: "i@github.red",
		AskerUserID:       1,
		IsPrivate:         true,
	})
	require.Nil(t, err)

	_, err = db.Create(ctx, CreateQuestionOptions{
		FromIP:            "114.5.1.4",
		UserID:            2,
		Content:           "Content - 4",
		ReceiveReplyEmail: "i@github.red",
		AskerUserID:       1,
	})
	require.Nil(t, err)

	err = db.AnswerByID(ctx, 2, "Answer - 2")
	require.Nil(t, err)

	t.Run("show private", func(t *testing.T) {
		got, err := db.CountUnread(ctx, 1, true)
		require.Nil(t, err)
		require.Equal(t, int64(2), got)
	})

	t.Run("hide private", func(t *testing.T) {
		got, err := db.CountUnread(ctx, 1, false)
		require.Nil(t, err)
		require.Equal(t, int64(1), got)
	})

	t.Run("not found", func(t *testing.T) {
		got, err := db.CountUnread(ctx, 404, true)
		require.Nil(t, err)
		require.Equal(t, int64(0), got)
	})
}

func testQuestionsSetPrivate(t *testing.T, ctx context.Context, db *questionsRepository) {
	_, err := db.Create(ctx, CreateQuestionOptions{
		FromIP:            "114.5.1.4",
		UserID:            1,
		Content:           "Content - 1",
		ReceiveReplyEmail: "i@github.red",
		AskerUserID:       1,
	})
	require.Nil(t, err)

	t.Run("normal", func(t *testing.T) {
		err = db.SetPrivate(ctx, 1)
		require.Nil(t, err)

		got, err := db.GetByID(ctx, 1)
		require.Nil(t, err)
		require.True(t, got.IsPrivate)
	})
}

func testQuestionsSetPublic(t *testing.T, ctx context.Context, db *questionsRepository) {
	_, err := db.Create(ctx, CreateQuestionOptions{
		FromIP:            "114.5.1.4",
		UserID:            1,
		Content:           "Content - 1",
		ReceiveReplyEmail: "i@github.red",
		AskerUserID:       1,
	})
	require.Nil(t, err)

	t.Run("normal", func(t *testing.T) {
		err = db.SetPublic(ctx, 1)
		require.Nil(t, err)

		got, err := db.GetByID(ctx, 1)
		require.Nil(t, err)
		require.False(t, got.IsPrivate)
	})
}
