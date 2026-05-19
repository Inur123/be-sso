package service

import (
	"errors"

	"github.com/google/uuid"

	"sso.pelajarnumagetan.or.id/internal/domain"
	"sso.pelajarnumagetan.or.id/internal/repository"
	"sso.pelajarnumagetan.or.id/internal/utils"
)

type UserService interface {
	GetAll(page, perPage int) ([]domain.User, int64, error)
	GetByID(id uuid.UUID) (*domain.User, error)
	UpdateRole(id uuid.UUID, role string) error
	Deactivate(id uuid.UUID) error
	Activate(id uuid.UUID) error
	VerifyEmail(id uuid.UUID) error
	Delete(id uuid.UUID) error
}

type userService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{userRepo: userRepo}
}

func (s *userService) GetAll(page, perPage int) ([]domain.User, int64, error) {
	return s.userRepo.FindAll(page, perPage)
}

func (s *userService) GetByID(id uuid.UUID) (*domain.User, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("user tidak ditemukan")
	}
	return user, nil
}

func (s *userService) UpdateRole(id uuid.UUID, role string) error {
	validRoles := map[string]bool{
		string(domain.RoleSuperAdmin): true,
		string(domain.RoleDeveloper):  true,
		string(domain.RoleUser):       true,
	}
	if !validRoles[role] {
		return errors.New("role tidak valid. Pilih: superadmin, developer, atau user")
	}

	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return errors.New("user tidak ditemukan")
	}

	if user.Role == domain.RoleSuperAdmin {
		return errors.New("tidak dapat mengubah role seorang Superadmin")
	}

	return s.userRepo.UpdateRole(id, domain.UserRole(role))
}

func (s *userService) Deactivate(id uuid.UUID) error {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return errors.New("user tidak ditemukan")
	}
	if user.Role == domain.RoleSuperAdmin {
		return errors.New("tidak dapat menonaktifkan akun Superadmin")
	}
	user.IsActive = false
	return s.userRepo.Update(user)
}

func (s *userService) Activate(id uuid.UUID) error {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return errors.New("user tidak ditemukan")
	}
	user.IsActive = true
	return s.userRepo.Update(user)
}

func (s *userService) VerifyEmail(id uuid.UUID) error {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return errors.New("user tidak ditemukan")
	}
	user.IsVerified = true
	return s.userRepo.Update(user)
}

func (s *userService) Delete(id uuid.UUID) error {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return errors.New("user tidak ditemukan")
	}
	if user.Role == domain.RoleSuperAdmin {
		return errors.New("tidak dapat menghapus akun Superadmin")
	}
	return s.userRepo.Delete(id)
}

// SeedSuperAdmin — buat akun superadmin pertama jika belum ada
func SeedSuperAdmin(userRepo repository.UserRepository) error {
	existing, _ := userRepo.FindByEmail("superadmin@pelajarnumagetan.or.id")
	if existing != nil {
		return nil // sudah ada, skip
	}

	hashed, err := utils.HashPassword("superadmin123")
	if err != nil {
		return err
	}

	superadmin := &domain.User{
		Name:       "Super Admin SSO",
		Email:      "superadmin@pelajarnumagetan.or.id",
		Password:   hashed,
		Role:       domain.RoleSuperAdmin,
		IsActive:   true,
		IsVerified: true,
	}

	return userRepo.Create(superadmin)
}
