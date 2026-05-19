package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AppStatus string

const (
	StatusPending  AppStatus = "pending"
	StatusVerified AppStatus = "verified"
	StatusRejected AppStatus = "rejected"
)

type Application struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Name         string         `gorm:"size:255;not null" json:"name"`
	Description  string         `gorm:"type:text" json:"description"`
	ClientID     string         `gorm:"size:100;uniqueIndex;not null" json:"client_id"`
	ClientSecret string         `gorm:"size:255;not null" json:"-"`
	RedirectURIs []string       `gorm:"type:text;serializer:json" json:"redirect_uris"`
	LogoURL      string         `gorm:"size:500" json:"logo_url"`
	Status       AppStatus      `gorm:"size:20;default:pending" json:"status"`
	IsActive     bool           `gorm:"default:true" json:"is_active"`
	OwnerID      uuid.UUID      `gorm:"type:uuid;not null" json:"owner_id"`
	Owner        *User          `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (a *Application) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
