package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"sso.pelajarnumagetan.or.id/internal/domain"
)

type SessionRepository interface {
	Create(session *domain.UserSession) error
	FindByRefreshToken(token string) (*domain.UserSession, error)
	FindByUserAndApp(userID, appID uuid.UUID) ([]domain.UserSession, error)
	FindByUserID(userID uuid.UUID) ([]domain.UserSession, error)
	FindOAuthByUserID(userID uuid.UUID) ([]domain.UserSession, error) // only app sessions
	DeleteByRefreshToken(token string) error
	DeleteByUserAndApp(userID, appID uuid.UUID) error
	DeleteExpired() error
}

type sessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) Create(session *domain.UserSession) error {
	return r.db.Create(session).Error
}

func (r *sessionRepository) FindByRefreshToken(token string) (*domain.UserSession, error) {
	var session domain.UserSession
	if err := r.db.Preload("User").Preload("App").
		Where("refresh_token = ?", token).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *sessionRepository) FindByUserAndApp(userID, appID uuid.UUID) ([]domain.UserSession, error) {
	var sessions []domain.UserSession
	if err := r.db.Where("user_id = ? AND app_id = ?", userID, appID).Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *sessionRepository) FindByUserID(userID uuid.UUID) ([]domain.UserSession, error) {
	var sessions []domain.UserSession
	if err := r.db.Preload("App").Where("user_id = ?", userID).Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

// FindOAuthByUserID returns only sessions linked to a third-party app (app_id IS NOT NULL)
func (r *sessionRepository) FindOAuthByUserID(userID uuid.UUID) ([]domain.UserSession, error) {
	var sessions []domain.UserSession
	if err := r.db.Preload("App").
		Where("user_id = ? AND app_id IS NOT NULL", userID).
		Order("expires_at DESC").
		Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *sessionRepository) DeleteByRefreshToken(token string) error {
	return r.db.Where("refresh_token = ?", token).Delete(&domain.UserSession{}).Error
}

func (r *sessionRepository) DeleteByUserAndApp(userID, appID uuid.UUID) error {
	return r.db.Where("user_id = ? AND app_id = ?", userID, appID).Delete(&domain.UserSession{}).Error
}

func (r *sessionRepository) DeleteExpired() error {
	return r.db.Where("expires_at < ?", time.Now()).Delete(&domain.UserSession{}).Error
}
