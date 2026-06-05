package domain

import (
	"time"

	"github.com/google/uuid"
)

type AppAccess struct {
	UserID    uuid.UUID    `gorm:"type:uuid;primaryKey;autoIncrement:false" json:"user_id"`
	User      *User        `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	AppID     uuid.UUID    `gorm:"type:uuid;primaryKey;autoIncrement:false" json:"app_id"`
	App       *Application `gorm:"foreignKey:AppID;constraint:OnDelete:CASCADE" json:"app,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}
