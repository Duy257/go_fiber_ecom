package services

import (
	"errors"

	"go-fiber/internal/models"
	"go-fiber/internal/repositories"
	"go-fiber/internal/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserService struct {
	repo     *repositories.UserRepository
	roleRepo *repositories.RoleRepository
}

func NewUserService(repo *repositories.UserRepository, roleRepo *repositories.RoleRepository) *UserService {
	return &UserService{repo: repo, roleRepo: roleRepo}
}

type CreateUserInput struct {
	Email       string `json:"email" validate:"omitempty,email"`
	PhoneNumber string `json:"phone_number" validate:"omitempty"`
	Password    string `json:"password" validate:"required,min=6"`
	Name        string `json:"name" validate:"required"`
	RoleID      string `json:"role_id" validate:"required"`
}

type UpdateUserInput struct {
	Email       *string `json:"email"`
	PhoneNumber *string `json:"phone_number"`
	Name        *string `json:"name"`
	RoleID      *string `json:"role_id"`
	Status      *string `json:"status"`
}

func (s *UserService) GetAll(page, limit int) ([]models.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	return s.repo.FindAll(page, limit)
}

func (s *UserService) GetByID(id uuid.UUID) (*models.User, error) {
	return s.repo.FindByID(id)
}

func (s *UserService) Create(input CreateUserInput) (*models.User, error) {
	if input.Email == "" && input.PhoneNumber == "" {
		return nil, errors.New("email or phone_number is required")
	}

	roleID, err := uuid.Parse(input.RoleID)
	if err != nil {
		return nil, errors.New("invalid role_id")
	}

	_, err = s.roleRepo.FindByID(roleID)
	if err != nil {
		return nil, errors.New("role not found")
	}

	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Password: hashedPassword,
		Name:     input.Name,
		RoleID:   roleID,
		Status:   "active",
	}

	if input.Email != "" {
		user.Email = &input.Email
	}
	if input.PhoneNumber != "" {
		user.PhoneNumber = &input.PhoneNumber
	}

	if err := s.repo.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) Update(id uuid.UUID, input UpdateUserInput) (*models.User, error) {
	user, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	if input.Email != nil {
		user.Email = input.Email
	}
	if input.PhoneNumber != nil {
		user.PhoneNumber = input.PhoneNumber
	}
	if input.Name != nil {
		user.Name = *input.Name
	}
	if input.Status != nil {
		user.Status = *input.Status
	}
	if input.RoleID != nil {
		roleID, err := uuid.Parse(*input.RoleID)
		if err != nil {
			return nil, errors.New("invalid role_id")
		}
		user.RoleID = roleID
	}

	if err := s.repo.Update(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) Delete(id uuid.UUID) error {
	_, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}
	return s.repo.Delete(id)
}
