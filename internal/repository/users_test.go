// Copyright 2023 E99p1ant. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/wuhan005/gadget"
	"gorm.io/gorm"

	"github.com/syt3s/TreeBox/internal/config"
	"github.com/syt3s/TreeBox/internal/model"
)

func TestUsers(t *testing.T) {
	t.Parallel()

	db, cleanup := newTestDB(t)
	ctx := context.Background()

	usersStore := NewUserRepository(db)

	for _, tc := range []struct {
		name string
		test func(t *testing.T, ctx context.Context, db *usersRepository)
	}{
		{"Create", testUsersCreate},
		{"Register", testUsersRegister},
		{"GetByID", testUsersGetByID},
		{"GetByEmail", testUsersGetByEmail},
		{"GetByDomain", testUsersGetByDomain},
		{"Update", testUsersUpdate},
		{"UpdateHarassmentSetting", testUsersUpdateHarassmentSetting},
		{"Authenticate", testUsersAuthenticate},
		{"AuthenticateLegacyHashUpgrade", testUsersAuthenticateLegacyHashUpgrade},
		{"ChangePassword", testUsersChangePassword},
		{"UpdatePassword", testUsersUpdatePassword},
		{"Deactivate", testUsersDeactivate},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() {
				if err := cleanup("users"); err != nil {
					t.Fatal(err)
				}
			})
			tc.test(t, ctx, usersStore.(*usersRepository))
		})
	}
}

func testUsersCreate(t *testing.T, ctx context.Context, db *usersRepository) {
	t.Run("normal", func(t *testing.T) {
		err := db.Create(ctx, CreateUserOptions{
			Name:       "E99p1ant",
			Password:   "super_secret",
			Email:      "i@github.red",
			Avatar:     "avater.png",
			Domain:     "e99",
			Background: "background.png",
			Intro:      "Be cool, but also be warm.",
		})
		require.Nil(t, err)
	})

	t.Run("repeat email", func(t *testing.T) {
		err := db.Create(ctx, CreateUserOptions{
			Name:       "E99p1ant",
			Password:   "super_secret",
			Email:      "i@github.red",
			Avatar:     "avater.png",
			Domain:     "e99p1ant",
			Background: "background.png",
			Intro:      "Be cool, but also be warm.",
		})
		require.Equal(t, ErrDuplicateEmail, err)
	})

	t.Run("repeat domain", func(t *testing.T) {
		err := db.Create(ctx, CreateUserOptions{
			Name:       "E99p1ant",
			Password:   "super_secret",
			Email:      "e99@github.red",
			Avatar:     "avater.png",
			Domain:     "e99",
			Background: "background.png",
			Intro:      "Be cool, but also be warm.",
		})
		require.Equal(t, ErrDuplicateDomain, err)
	})
}

func testUsersRegister(t *testing.T, ctx context.Context, db *usersRepository) {
	t.Run("normal", func(t *testing.T) {
		result, err := db.Register(ctx, RegisterUserOptions{
			CreateUserOptions: CreateUserOptions{
				Name:       "Acme Owner",
				Password:   "super_secret",
				Email:      "owner@acme.test",
				Avatar:     "avatar.png",
				Domain:     "acme-owner",
				Background: "background.png",
				Intro:      "Personal tenant owner",
			},
			TenantName:    "Acme",
			WorkspaceName: "Default workspace",
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.User)
		require.NotNil(t, result.Tenant)
		require.NotNil(t, result.Membership)
		require.NotNil(t, result.Workspace)

		require.Equal(t, result.User.ID, result.Tenant.OwnerUserID)
		require.Equal(t, result.Tenant.ID, result.Membership.TenantID)
		require.Equal(t, result.User.ID, result.Membership.UserID)
		require.Equal(t, model.TenantRoleOwner, result.Membership.Role)
		require.Equal(t, result.Tenant.ID, result.Workspace.TenantID)
		require.Equal(t, result.User.ID, result.Workspace.CreatedByUserID)
		require.True(t, result.Workspace.IsDefault)

		var auditLogCount int64
		err = db.WithContext(ctx).
			Model(&model.AuditLog{}).
			Where("tenant_id = ? AND action = ?", result.Tenant.ID, "tenant.bootstrap").
			Count(&auditLogCount).
			Error
		require.NoError(t, err)
		require.Equal(t, int64(1), auditLogCount)
	})
}

func testUsersGetByID(t *testing.T, ctx context.Context, db *usersRepository) {
	err := db.Create(ctx, CreateUserOptions{
		Name:       "E99p1ant",
		Password:   "super_secret",
		Email:      "i@github.red",
		Avatar:     "avater.png",
		Domain:     "e99",
		Background: "background.png",
		Intro:      "Be cool, but also be warm.",
	})
	require.Nil(t, err)

	t.Run("normal", func(t *testing.T) {
		got, err := db.GetByID(ctx, 1)
		require.Nil(t, err)

		got.CreatedAt = time.Time{}
		got.UpdatedAt = time.Time{}

		want := &model.User{
			Model: gorm.Model{
				ID: 1,
			},
			UID:        got.UID,
			Name:       "E99p1ant",
			Password:   "super_secret",
			Email:      "i@github.red",
			Avatar:     "avater.png",
			Domain:     "e99",
			Background: "background.png",
			Intro:      "Be cool, but also be warm.",
			Notify:     model.NotifyTypeEmail,
		}
		want.EncodePassword()
		require.Equal(t, want, got)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := db.GetByID(ctx, 404)
		require.Equal(t, ErrUserNotExists, err)
	})
}

func testUsersGetByEmail(t *testing.T, ctx context.Context, db *usersRepository) {
	err := db.Create(ctx, CreateUserOptions{
		Name:       "E99p1ant",
		Password:   "super_secret",
		Email:      "i@github.red",
		Avatar:     "avater.png",
		Domain:     "e99",
		Background: "background.png",
		Intro:      "Be cool, but also be warm.",
	})
	require.Nil(t, err)

	t.Run("normal", func(t *testing.T) {
		got, err := db.GetByEmail(ctx, "i@github.red")
		require.Nil(t, err)

		got.CreatedAt = time.Time{}
		got.UpdatedAt = time.Time{}

		want := &model.User{
			Model: gorm.Model{
				ID: 1,
			},
			UID:        got.UID,
			Name:       "E99p1ant",
			Password:   "super_secret",
			Email:      "i@github.red",
			Avatar:     "avater.png",
			Domain:     "e99",
			Background: "background.png",
			Intro:      "Be cool, but also be warm.",
			Notify:     model.NotifyTypeEmail,
		}
		want.EncodePassword()
		require.Equal(t, want, got)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := db.GetByEmail(ctx, "404")
		require.Equal(t, ErrUserNotExists, err)
	})
}

func testUsersGetByDomain(t *testing.T, ctx context.Context, db *usersRepository) {
	err := db.Create(ctx, CreateUserOptions{
		Name:       "E99p1ant",
		Password:   "super_secret",
		Email:      "i@github.red",
		Avatar:     "avater.png",
		Domain:     "e99",
		Background: "background.png",
		Intro:      "Be cool, but also be warm.",
	})
	require.Nil(t, err)

	t.Run("normal", func(t *testing.T) {
		got, err := db.GetByDomain(ctx, "e99")
		require.Nil(t, err)

		got.CreatedAt = time.Time{}
		got.UpdatedAt = time.Time{}

		want := &model.User{
			Model: gorm.Model{
				ID: 1,
			},
			UID:        got.UID,
			Name:       "E99p1ant",
			Password:   "super_secret",
			Email:      "i@github.red",
			Avatar:     "avater.png",
			Domain:     "e99",
			Background: "background.png",
			Intro:      "Be cool, but also be warm.",
			Notify:     model.NotifyTypeEmail,
		}
		want.EncodePassword()
		require.Equal(t, want, got)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := db.GetByDomain(ctx, "404")
		require.Equal(t, ErrUserNotExists, err)
	})
}

func testUsersUpdate(t *testing.T, ctx context.Context, db *usersRepository) {
	err := db.Create(ctx, CreateUserOptions{
		Name:       "E99p1ant",
		Password:   "super_secret",
		Email:      "i@github.red",
		Avatar:     "avater.png",
		Domain:     "e99",
		Background: "background.png",
		Intro:      "Be cool, but also be warm.",
	})
	require.Nil(t, err)

	t.Run("normal", func(t *testing.T) {
		err := db.Update(ctx, 1, UpdateUserOptions{
			Name:       "e99",
			Avatar:     "new_avatar.png",
			Background: "new_background.png",
			Intro:      "Be cool, but also be warm!!",
			Notify:     model.NotifyTypeNone,
		})
		require.Nil(t, err)

		got, err := db.GetByID(ctx, 1)
		require.Nil(t, err)

		got.CreatedAt = time.Time{}
		got.UpdatedAt = time.Time{}

		want := &model.User{
			Model: gorm.Model{
				ID: 1,
			},
			UID:        got.UID,
			Name:       "e99",
			Password:   "super_secret",
			Email:      "i@github.red",
			Avatar:     "new_avatar.png",
			Domain:     "e99",
			Background: "new_background.png",
			Intro:      "Be cool, but also be warm!!",
			Notify:     model.NotifyTypeNone,
		}
		want.EncodePassword()
		require.Equal(t, want, got)
	})
}

func testUsersUpdateHarassmentSetting(t *testing.T, ctx context.Context, db *usersRepository) {
	err := db.Create(ctx, CreateUserOptions{
		Name:       "E99p1ant",
		Password:   "super_secret",
		Email:      "i@github.red",
		Avatar:     "avater.png",
		Domain:     "e99",
		Background: "background.png",
		Intro:      "Be cool, but also be warm.",
	})
	require.Nil(t, err)

	t.Run("normal", func(t *testing.T) {
		err := db.UpdateHarassmentSetting(ctx, 1, HarassmentSettingOptions{
			Type: model.HarassmentSettingNone,
		})
		require.Nil(t, err)
	})

	t.Run("unexpected harassment setting", func(t *testing.T) {
		err := db.UpdateHarassmentSetting(ctx, 1, HarassmentSettingOptions{
			Type: "not found",
		})
		require.NotNil(t, err)
	})
}

func testUsersAuthenticate(t *testing.T, ctx context.Context, db *usersRepository) {
	err := db.Create(ctx, CreateUserOptions{
		Name:       "E99p1ant",
		Password:   "super_secret",
		Email:      "i@github.red",
		Avatar:     "avater.png",
		Domain:     "e99",
		Background: "background.png",
		Intro:      "Be cool, but also be warm.",
	})
	require.Nil(t, err)

	got, err := db.Authenticate(ctx, "i@github.red", "super_secret")
	require.Nil(t, err)

	got.CreatedAt = time.Time{}
	got.UpdatedAt = time.Time{}

	want := &model.User{
		Model: gorm.Model{
			ID: 1,
		},
		UID:        got.UID,
		Name:       "E99p1ant",
		Password:   "super_secret",
		Email:      "i@github.red",
		Avatar:     "avater.png",
		Domain:     "e99",
		Background: "background.png",
		Intro:      "Be cool, but also be warm.",
		Notify:     model.NotifyTypeEmail,
	}
	want.EncodePassword()
	require.Equal(t, want, got)
}

func testUsersAuthenticateLegacyHashUpgrade(t *testing.T, ctx context.Context, db *usersRepository) {
	err := db.Create(ctx, CreateUserOptions{
		Name:       "Legacy",
		Password:   "super_secret",
		Email:      "legacy@github.red",
		Avatar:     "avatar.png",
		Domain:     "legacy",
		Background: "background.png",
		Intro:      "Legacy hash user",
	})
	require.Nil(t, err)

	legacyHash := gadget.HmacSha1("super_secret", config.Server.Salt)
	err = db.WithContext(ctx).Model(&model.User{}).Where("email = ?", "legacy@github.red").Update("password", legacyHash).Error
	require.NoError(t, err)

	got, err := db.Authenticate(ctx, "legacy@github.red", "super_secret")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.False(t, got.NeedsPasswordUpgrade())
	require.True(t, strings.HasPrefix(got.Password, "argon2id$"))
}

func testUsersChangePassword(t *testing.T, ctx context.Context, db *usersRepository) {
	err := db.Create(ctx, CreateUserOptions{
		Name:       "E99p1ant",
		Password:   "super_secret",
		Email:      "i@github.red",
		Avatar:     "avater.png",
		Domain:     "e99",
		Background: "background.png",
		Intro:      "Be cool, but also be warm.",
	})
	require.Nil(t, err)

	t.Run("normal", func(t *testing.T) {
		err := db.ChangePassword(ctx, 1, "super_secret", "new_password")
		require.Nil(t, err)
	})

	t.Run("wrong password", func(t *testing.T) {
		err := db.ChangePassword(ctx, 1, "wrong_password", "new_password")
		require.Equal(t, ErrBadCredential, err)
	})
}

func testUsersUpdatePassword(t *testing.T, ctx context.Context, db *usersRepository) {
	err := db.Create(ctx, CreateUserOptions{
		Name:       "E99p1ant",
		Password:   "super_secret",
		Email:      "i@github.red",
		Avatar:     "avater.png",
		Domain:     "e99",
		Background: "background.png",
		Intro:      "Be cool, but also be warm.",
	})
	require.Nil(t, err)

	t.Run("normal", func(t *testing.T) {
		err := db.UpdatePassword(ctx, 1, "new_password")
		require.Nil(t, err)
	})
}

func testUsersDeactivate(t *testing.T, ctx context.Context, db *usersRepository) {
	err := db.Create(ctx, CreateUserOptions{
		Name:       "E99p1ant",
		Password:   "super_secret",
		Email:      "i@github.red",
		Avatar:     "avater.png",
		Domain:     "e99",
		Background: "background.png",
		Intro:      "Be cool, but also be warm.",
	})
	require.Nil(t, err)

	t.Run("normal", func(t *testing.T) {
		err := db.Deactivate(ctx, 1)
		require.Nil(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		err := db.Deactivate(ctx, 404)
		require.NotNil(t, err)
	})
}
