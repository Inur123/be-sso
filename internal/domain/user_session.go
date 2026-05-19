package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserSession struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID       uuid.UUID      `gorm:"type:uuid;not null" json:"user_id"`
	User         *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	AppID        *uuid.UUID     `gorm:"type:uuid" json:"app_id"`          // nullable — nil = login dashboard SSO
	App          *Application   `gorm:"foreignKey:AppID" json:"app,omitempty"`
	RefreshToken string         `gorm:"size:500;uniqueIndex;not null" json:"refresh_token"` // shown truncated in FE sessions page
	Scope        string         `gorm:"size:500" json:"scope"`
	UserAgent    string         `gorm:"size:500" json:"user_agent"`
	IPAddress    string         `gorm:"size:50" json:"ip_address"`
	ExpiresAt    time.Time      `json:"expires_at"`
	CreatedAt    time.Time      `json:"created_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (s *UserSession) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

func (s *UserSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}
