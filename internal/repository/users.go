// Copyright 2022 E99p1ant. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repository

import (
	"context"

	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/syt3s/TreeBox/internal/model"
)

var Users UserRepository

var _ UserRepository = (*usersRepository)(nil)

type UserRepository interface {
	Create(ctx context.Context, opts CreateUserOptions) error
	GetByID(ctx context.Context, id uint) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByDomain(ctx context.Context, domain string) (*model.User, error)
	Update(ctx context.Context, id uint, opts UpdateUserOptions) error
	UpdateHarassmentSetting(ctx context.Context, id uint, options HarassmentSettingOptions) error
	Authenticate(ctx context.Context, email, password string) (*model.User, error)
	ChangePassword(ctx context.Context, id uint, oldPassword, newPassword string) error
	UpdatePassword(ctx context.Context, id uint, newPassword string) error
	Deactivate(ctx context.Context, id uint) error
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &usersRepository{db}
}

type usersRepository struct {
	*gorm.DB
}

type CreateUserOptions struct {
	Name       string
	Password   string
	Email      string
	Avatar     string
	Domain     string
	Background string
	Intro      string
}

var (
	ErrUserNotExists   = errors.New("账号不存在")
	ErrBadCredential   = errors.New("邮箱或密码错误")
	ErrDuplicateEmail  = errors.New("这个邮箱已经注册过账号了")
	ErrDuplicateDomain = errors.New("个性域名重复了，换一个吧~")
)

func (db *usersRepository) Create(ctx context.Context, opts CreateUserOptions) error {
	if err := db.validate(ctx, opts); err != nil {
		return err
	}

	newUser := &model.User{
		Name:       opts.Name,
		Password:   opts.Password,
		Email:      opts.Email,
		Avatar:     opts.Avatar,
		Domain:     opts.Domain,
		Background: opts.Background,
		Intro:      opts.Intro,
		Notify:     model.NotifyTypeEmail,
	}
	newUser.EncodePassword()

	if err := db.WithContext(ctx).Create(newUser).Error; err != nil {
		return errors.Wrap(err, "create user")
	}
	return nil
}

func (db *usersRepository) getBy(ctx context.Context, where string, args ...interface{}) (*model.User, error) {
	var user model.User
	if err := db.WithContext(ctx).Where(where, args...).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotExists
		}
		return nil, errors.Wrap(err, "get user")
	}
	return &user, nil
}

func (db *usersRepository) GetByID(ctx context.Context, id uint) (*model.User, error) {
	return db.getBy(ctx, "id = ?", id)
}

func (db *usersRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	return db.getBy(ctx, "email = ?", email)
}

func (db *usersRepository) GetByDomain(ctx context.Context, domain string) (*model.User, error) {
	return db.getBy(ctx, "domain = ?", domain)
}

type UpdateUserOptions struct {
	Name       string
	Avatar     string
	Background string
	Intro      string
	Notify     model.NotifyType
}

func (db *usersRepository) Update(ctx context.Context, id uint, opts UpdateUserOptions) error {
	_, err := db.GetByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "get user by id")
	}

	switch opts.Notify {
	case model.NotifyTypeEmail, model.NotifyTypeNone:
	default:
		return errors.Errorf("unexpected notify type: %q", opts.Notify)
	}

	if err := db.WithContext(ctx).Where("id = ?", id).Updates(&model.User{
		Name:       opts.Name,
		Avatar:     opts.Avatar,
		Background: opts.Background,
		Intro:      opts.Intro,
		Notify:     opts.Notify,
	}).Error; err != nil {
		return errors.Wrap(err, "update user")
	}
	return nil
}

type HarassmentSettingOptions struct {
	Type       model.HarassmentSettingType
	BlockWords string
}

func (db *usersRepository) UpdateHarassmentSetting(ctx context.Context, id uint, options HarassmentSettingOptions) error {
	typ := options.Type

	switch typ {
	case model.HarassmentSettingNone, model.HarassmentSettingTypeRegisterOnly:
	default:
		return errors.Errorf("unexpected harassment setting type: %q", typ)
	}

	if err := db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"HarassmentSetting": typ,
		"BlockWords":        options.BlockWords,
	}).Error; err != nil {
		return errors.Wrap(err, "update user")
	}
	return nil
}

func (db *usersRepository) Authenticate(ctx context.Context, email, password string) (*model.User, error) {
	u, err := db.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrBadCredential
	}

	if !u.Authenticate(password) {
		return nil, ErrBadCredential
	}

	return u, nil
}

func (db *usersRepository) ChangePassword(ctx context.Context, id uint, oldPassword, newPassword string) error {
	u, err := db.GetByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "get user by id")
	}

	if !u.Authenticate(oldPassword) {
		return ErrBadCredential
	}

	u.Password = newPassword
	u.EncodePassword()

	if err := db.WithContext(ctx).Model(&model.User{}).Where("id = ?", u.ID).Update("password", u.Password).Error; err != nil {
		return errors.Wrap(err, "change password")
	}
	return nil
}

func (db *usersRepository) UpdatePassword(ctx context.Context, id uint, newPassword string) error {
	u, err := db.GetByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "get user by id")
	}

	u.Password = newPassword
	u.EncodePassword()

	if err := db.WithContext(ctx).Model(&model.User{}).Where("id = ?", u.ID).Update("password", u.Password).Error; err != nil {
		return errors.Wrap(err, "change password")
	}
	return nil
}

func (db *usersRepository) Deactivate(ctx context.Context, id uint) error {
	u, err := db.GetByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "get user by id")
	}

	if err := db.WithContext(ctx).Where("id = ?", u.ID).Delete(&model.User{}).Error; err != nil {
		return errors.Wrap(err, "delete user")
	}
	return nil
}

func (db *usersRepository) validate(ctx context.Context, opts CreateUserOptions) error {
	if err := db.WithContext(ctx).Model(&model.User{}).Where("email = ?", opts.Email).First(&model.User{}).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return errors.Wrap(err, "validate email")
		}
	} else {
		return ErrDuplicateEmail
	}

	if err := db.WithContext(ctx).Model(&model.User{}).Where("domain = ?", opts.Domain).First(&model.User{}).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return errors.Wrap(err, "validate name")
		}
	} else {
		return ErrDuplicateDomain
	}

	return nil
}
