package services

import (
	"errors"

	"go-fiber/internal/models"
	"go-fiber/internal/repositories"
	"go-fiber/internal/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CustomerService struct {
	repo *repositories.CustomerRepository
}

func NewCustomerService(repo *repositories.CustomerRepository) *CustomerService {
	return &CustomerService{repo: repo}
}

type CreateCustomerInput struct {
	Email       string `json:"email" validate:"omitempty,email"`
	PhoneNumber string `json:"phone_number" validate:"omitempty"`
	Password    string `json:"password" validate:"required,min=6"`
	Name        string `json:"name" validate:"required"`
}

type UpdateCustomerInput struct {
	Email       *string `json:"email"`
	PhoneNumber *string `json:"phone_number"`
	Name        *string `json:"name"`
	Status      *string `json:"status"`
}

func (s *CustomerService) GetAll(page, limit int) ([]models.Customer, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	return s.repo.FindAll(page, limit)
}

func (s *CustomerService) GetByID(id uuid.UUID) (*models.Customer, error) {
	return s.repo.FindByID(id)
}

func (s *CustomerService) Create(input CreateCustomerInput) (*models.Customer, error) {
	if input.Email == "" && input.PhoneNumber == "" {
		return nil, errors.New("email or phone_number is required")
	}

	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	customer := &models.Customer{
		Password: hashedPassword,
		Name:     input.Name,
		Status:   "active",
	}

	if input.Email != "" {
		customer.Email = &input.Email
	}
	if input.PhoneNumber != "" {
		customer.PhoneNumber = &input.PhoneNumber
	}

	if err := s.repo.Create(customer); err != nil {
		return nil, err
	}
	return customer, nil
}

func (s *CustomerService) Update(id uuid.UUID, input UpdateCustomerInput) (*models.Customer, error) {
	customer, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("customer not found")
		}
		return nil, err
	}

	if input.Email != nil {
		customer.Email = input.Email
	}
	if input.PhoneNumber != nil {
		customer.PhoneNumber = input.PhoneNumber
	}
	if input.Name != nil {
		customer.Name = *input.Name
	}
	if input.Status != nil {
		customer.Status = *input.Status
	}

	if err := s.repo.Update(customer); err != nil {
		return nil, err
	}
	return customer, nil
}

func (s *CustomerService) Delete(id uuid.UUID) error {
	_, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("customer not found")
		}
		return err
	}
	return s.repo.Delete(id)
}
