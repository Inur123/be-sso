package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"sso.pelajarnumagetan.or.id/internal/domain"
)

type AppAccessRepository interface {
	CheckAccess(userID, appID uuid.UUID) (bool, error)
	GetAssignedUsers(appID uuid.UUID) ([]domain.User, error)
	AssignUsers(appID uuid.UUID, userIDs []uuid.UUID) error
}

type appAccessRepository struct {
	db *gorm.DB
}

func NewAppAccessRepository(db *gorm.DB) AppAccessRepository {
	return &appAccessRepository{db: db}
}

func (r *appAccessRepository) CheckAccess(userID, appID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&domain.AppAccess{}).Where("user_id = ? AND app_id = ?", userID, appID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *appAccessRepository) GetAssignedUsers(appID uuid.UUID) ([]domain.User, error) {
	var users []domain.User
	err := r.db.Table("users").
		Select("users.*").
		Joins("join app_accesses on app_accesses.user_id = users.id").
		Where("app_accesses.app_id = ? AND users.deleted_at IS NULL", appID).
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (r *appAccessRepository) AssignUsers(appID uuid.UUID, userIDs []uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Hapus semua akses lama untuk app ini
		if err := tx.Where("app_id = ?", appID).Delete(&domain.AppAccess{}).Error; err != nil {
			return err
		}

		// 2. Insert akses baru
		if len(userIDs) > 0 {
			accesses := make([]domain.AppAccess, len(userIDs))
			for i, uID := range userIDs {
				accesses[i] = domain.AppAccess{
					UserID: uID,
					AppID:  appID,
				}
			}
			if err := tx.Create(&accesses).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
