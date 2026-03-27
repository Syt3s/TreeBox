package model

import (
	"github.com/rs/xid"
	"github.com/wuhan005/gadget"
	"gorm.io/gorm"

	"github.com/syt3s/TreeBox/internal/config"
)

type User struct {
	gorm.Model        `json:"-"`
	UID               string                `json:"-"`
	Name              string                `json:"name"`
	Password          string                `json:"-"`
	Email             string                `json:"email"`
	Avatar            string                `json:"avatar"`
	Domain            string                `json:"domain"`
	Background        string                `json:"background"`
	Intro             string                `json:"intro"`
	Notify            NotifyType            `json:"notify"`
	HarassmentSetting HarassmentSettingType `json:"harassment_setting"`
	BlockWords        string                `json:"block_words"`
}

func (u *User) BeforeCreate(_ *gorm.DB) error {
	u.UID = xid.New().String()
	return nil
}

type NotifyType string

const (
	NotifyTypeEmail NotifyType = "email"
	NotifyTypeNone  NotifyType = "none"
)

type HarassmentSettingType string

const (
	HarassmentSettingNone             HarassmentSettingType = "none"
	HarassmentSettingTypeRegisterOnly HarassmentSettingType = "register_only"
)

func (u *User) EncodePassword() {
	u.Password = gadget.HmacSha1(u.Password, config.Server.Salt)
}

func (u *User) Authenticate(password string) bool {
	password = gadget.HmacSha1(password, config.Server.Salt)
	return u.Password == password
}
