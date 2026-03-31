package model

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strings"

	"github.com/rs/xid"
	"github.com/wuhan005/gadget"
	"golang.org/x/crypto/argon2"
	"gorm.io/gorm"

	"github.com/syt3s/TreeBox/internal/config"
)

type User struct {
	gorm.Model         `json:"-"`
	UID                string                `json:"-"`
	Name               string                `json:"name"`
	Password           string                `json:"-"`
	Email              string                `json:"email"`
	Avatar             string                `json:"avatar"`
	Domain             string                `json:"domain"`
	Background         string                `json:"background"`
	Intro              string                `json:"intro"`
	Notify             NotifyType            `json:"notify"`
	HarassmentSetting  HarassmentSettingType `json:"harassment_setting"`
	BlockWords         string                `json:"block_words"`
	RoutingWorkspaceID uint                  `gorm:"index" json:"routing_workspace_id,omitempty"`
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
	u.Password = hashPassword(u.Password, u.Email, u.UID)
}

func (u *User) Authenticate(password string) bool {
	return verifyPassword(u.Password, password, u.Email, u.UID)
}

func (u *User) NeedsPasswordUpgrade() bool {
	return !strings.HasPrefix(strings.TrimSpace(u.Password), passwordHashPrefix)
}

const passwordHashPrefix = "argon2id$"

func hashPassword(password, email, uid string) string {
	salt := passwordSalt(email, uid)
	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 2, 32)
	return passwordHashPrefix + base64.RawStdEncoding.EncodeToString(hash)
}

func verifyPassword(encodedHash, password, email, uid string) bool {
	encodedHash = strings.TrimSpace(encodedHash)
	if strings.HasPrefix(encodedHash, passwordHashPrefix) {
		expected := hashPassword(password, email, uid)
		return hmac.Equal([]byte(encodedHash), []byte(expected))
	}

	legacyPassword := gadget.HmacSha1(password, config.Server.Salt)
	return hmac.Equal([]byte(encodedHash), []byte(legacyPassword))
}

func passwordSalt(email, uid string) []byte {
	key := strings.ToLower(strings.TrimSpace(email))
	if key == "" {
		key = strings.TrimSpace(uid)
	}
	if key == "" {
		key = "treebox"
	}

	mac := hmac.New(sha256.New, []byte(config.Server.Salt))
	_, _ = mac.Write([]byte(key))

	full := mac.Sum(nil)
	return full[:16]
}
