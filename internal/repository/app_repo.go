package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"sso.pelajarnumagetan.or.id/internal/domain"
)

type AppRepository interface {
	Create(app *domain.Application) error
	FindByID(id uuid.UUID) (*domain.Application, error)
	FindByClientID(clientID string) (*domain.Application, error)
	FindByOwner(ownerID uuid.UUID) ([]domain.Application, error)
	FindAll(page, perPage int, status string) ([]domain.Application, int64, error)
	FindPending() ([]domain.Application, error)
	Update(app *domain.Application) error
	UpdateStatus(id uuid.UUID, status domain.AppStatus) error
	Delete(id uuid.UUID) error
}

type appRepository struct {
	db *gorm.DB
}

func NewAppRepository(db *gorm.DB) AppRepository {
	return &appRepository{db: db}
}

func (r *appRepository) Create(app *domain.Application) error {
	return r.db.Create(app).Error
}

func (r *appRepository) FindByID(id uuid.UUID) (*domain.Application, error) {
	var app domain.Application
	if err := r.db.Preload("Owner").Where("id = ?", id).First(&app).Error; err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *appRepository) FindByClientID(clientID string) (*domain.Application, error) {
	var app domain.Application
	if err := r.db.Where("client_id = ? AND is_active = true", clientID).First(&app).Error; err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *appRepository) FindByOwner(ownerID uuid.UUID) ([]domain.Application, error) {
	var apps []domain.Application
	if err := r.db.Where("owner_id = ?", ownerID).Find(&apps).Error; err != nil {
		return nil, err
	}
	return apps, nil
}

func (r *appRepository) FindAll(page, perPage int, status string) ([]domain.Application, int64, error) {
	var apps []domain.Application
	var total int64

	query := r.db.Model(&domain.Application{}).Preload("Owner")
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	if err := query.Offset(offset).Limit(perPage).Find(&apps).Error; err != nil {
		return nil, 0, err
	}

	return apps, total, nil
}

func (r *appRepository) FindPending() ([]domain.Application, error) {
	var apps []domain.Application
	if err := r.db.Preload("Owner").Where("status = ?", domain.StatusPending).Find(&apps).Error; err != nil {
		return nil, err
	}
	return apps, nil
}

func (r *appRepository) Update(app *domain.Application) error {
	return r.db.Save(app).Error
}

func (r *appRepository) UpdateStatus(id uuid.UUID, status domain.AppStatus) error {
	return r.db.Model(&domain.Application{}).Where("id = ?", id).Update("status", status).Error
}

func (r *appRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&domain.Application{}, "id = ?", id).Error
}
