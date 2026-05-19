package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"sso.pelajarnumagetan.or.id/internal/config"
	"sso.pelajarnumagetan.or.id/internal/utils"
)

type UserRole string

const (
	RoleSuperAdmin UserRole = "superadmin"
	RoleDeveloper  UserRole = "developer"
	RoleUser       UserRole = "user"
)

type User struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Name       string         `gorm:"type:text;not null" json:"name"`
	Email      string         `gorm:"size:255;uniqueIndex;not null" json:"email"`
	Phone      string         `gorm:"type:text" json:"phone"`
	Gender     string         `gorm:"type:text" json:"gender"`
	Password   string         `gorm:"size:255;not null" json:"-"`
	Image      string         `gorm:"type:text" json:"image"`
	Role       UserRole       `gorm:"size:20;default:user" json:"role"`
	IsActive   bool           `gorm:"default:true" json:"is_active"`
	IsVerified bool           `gorm:"default:false" json:"is_verified"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate sets UUID
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// BeforeSave enkripsi field PII sebelum disimpan ke DB
func (u *User) BeforeSave(tx *gorm.DB) error {
	return u.encryptFields()
}

// AfterFind dekripsi field PII setelah dibaca dari DB
func (u *User) AfterFind(tx *gorm.DB) error {
	return u.decryptFields()
}

func (u *User) encryptFields() error {
	key := config.Get().EncryptionKey

	if enc, err := utils.EncryptField(u.Name, key); err == nil {
		u.Name = enc
	}
	if enc, err := utils.EncryptField(u.Phone, key); err == nil {
		u.Phone = enc
	}
	if enc, err := utils.EncryptField(u.Gender, key); err == nil {
		u.Gender = enc
	}
	if enc, err := utils.EncryptField(u.Image, key); err == nil {
		u.Image = enc
	}
	return nil
}

func (u *User) decryptFields() error {
	key := config.Get().EncryptionKey

	println("DEBUG DECRYPT: u.Name before:", u.Name)
	decName, errName := utils.DecryptField(u.Name, key)
	if errName != nil {
		println("❌ DECRYPT NAME FAILED for email:", u.Email, "err:", errName.Error(), "val:", u.Name)
	} else {
		u.Name = decName
		println("DEBUG DECRYPT: u.Name after:", u.Name)
	}

	decPhone, errPhone := utils.DecryptField(u.Phone, key)
	if errPhone != nil {
		println("❌ DECRYPT PHONE FAILED:", errPhone.Error())
	} else {
		u.Phone = decPhone
	}

	decGender, errGender := utils.DecryptField(u.Gender, key)
	if errGender != nil {
		println("❌ DECRYPT GENDER FAILED:", errGender.Error())
	} else {
		u.Gender = decGender
	}

	decImage, errImage := utils.DecryptField(u.Image, key)
	if errImage != nil {
		println("❌ DECRYPT IMAGE FAILED:", errImage.Error())
	} else {
		u.Image = decImage
	}

	return nil
}
