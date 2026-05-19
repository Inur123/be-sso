package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"sso.pelajarnumagetan.or.id/internal/config"
	"sso.pelajarnumagetan.or.id/internal/domain"
	"sso.pelajarnumagetan.or.id/internal/utils"
)

type UserRepository interface {
	Create(user *domain.User) error
	FindByID(id uuid.UUID) (*domain.User, error)
	FindByEmail(email string) (*domain.User, error)
	FindByPhone(phone string) (*domain.User, error)
	FindByEmailUnscoped(email string) (*domain.User, error)
	FindByPhoneUnscoped(phone string) (*domain.User, error)
	Update(user *domain.User) error
	Delete(id uuid.UUID) error
	FindAll(page, perPage int) ([]domain.User, int64, error)
	UpdateRole(id uuid.UUID, role domain.UserRole) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *domain.User) error {
	return r.db.Create(user).Error
}

func (r *userRepository) FindByID(id uuid.UUID) (*domain.User, error) {
	var user domain.User
	if err := r.db.Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByEmail(email string) (*domain.User, error) {
	var user domain.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByPhone(phone string) (*domain.User, error) {
	if phone == "" {
		return nil, gorm.ErrRecordNotFound
	}

	cfg := config.Get()
	encPhone, err := utils.EncryptField(phone, cfg.EncryptionKey)
	if err != nil {
		return nil, err
	}

	var user domain.User
	// Cari yang single-encrypted
	if err := r.db.Where("phone = ?", encPhone).First(&user).Error; err == nil {
		return &user, nil
	}

	// Cari yang double-encrypted (untuk data legacy)
	encDouble, err := utils.EncryptField(encPhone, cfg.EncryptionKey)
	if err == nil {
		if err := r.db.Where("phone = ?", encDouble).First(&user).Error; err == nil {
			return &user, nil
		}
	}

	return nil, gorm.ErrRecordNotFound
}

func (r *userRepository) FindByEmailUnscoped(email string) (*domain.User, error) {
	var user domain.User
	if err := r.db.Unscoped().Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByPhoneUnscoped(phone string) (*domain.User, error) {
	if phone == "" {
		return nil, gorm.ErrRecordNotFound
	}

	cfg := config.Get()
	encPhone, err := utils.EncryptField(phone, cfg.EncryptionKey)
	if err != nil {
		return nil, err
	}

	var user domain.User
	// Cari yang single-encrypted (termasuk soft-delete)
	if err := r.db.Unscoped().Where("phone = ?", encPhone).First(&user).Error; err == nil {
		return &user, nil
	}

	// Cari yang double-encrypted (untuk data legacy, termasuk soft-delete)
	encDouble, err := utils.EncryptField(encPhone, cfg.EncryptionKey)
	if err == nil {
		if err := r.db.Unscoped().Where("phone = ?", encDouble).First(&user).Error; err == nil {
			return &user, nil
		}
	}

	return nil, gorm.ErrRecordNotFound
}

func (r *userRepository) Update(user *domain.User) error {
	return r.db.Save(user).Error
}

func (r *userRepository) Delete(id uuid.UUID) error {
	return r.db.Unscoped().Delete(&domain.User{}, "id = ?", id).Error
}

func (r *userRepository) FindAll(page, perPage int) ([]domain.User, int64, error) {
	var users []domain.User
	var total int64

	offset := (page - 1) * perPage

	if err := r.db.Model(&domain.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.Offset(offset).Limit(perPage).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *userRepository) UpdateRole(id uuid.UUID, role domain.UserRole) error {
	return r.db.Model(&domain.User{}).Where("id = ?", id).Update("role", role).Error
}
