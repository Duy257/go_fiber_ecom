package services

import (
	"errors"
	"fmt"

	"go-fiber/internal/models"
	"go-fiber/internal/repositories"
	"go-fiber/internal/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ShopService struct {
	repo     *repositories.ShopRepository
	userRepo *repositories.UserRepository
}

func NewShopService(repo *repositories.ShopRepository, userRepo *repositories.UserRepository) *ShopService {
	return &ShopService{repo: repo, userRepo: userRepo}
}

type CreateShopInput struct {
	UserID      string `json:"user_id" validate:"required"`
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	Logo        string `json:"logo"`
	Address     string `json:"address"`
	Phone       string `json:"phone"`
}

type UpdateShopInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Logo        *string `json:"logo"`
	Address     *string `json:"address"`
	Phone       *string `json:"phone"`
	Status      *string `json:"status"`
}

func generateUniqueSlug(slug string, existingCheck func(string) bool) string {
	if !existingCheck(slug) {
		return slug
	}
	for i := 1; i <= 100; i++ {
		candidate := fmt.Sprintf("%s-%d", slug, i)
		if !existingCheck(candidate) {
			return candidate
		}
	}
	return fmt.Sprintf("%s-%d", slug, 101)
}

func (s *ShopService) Create(input CreateShopInput) (*models.Shop, error) {
	userID, err := uuid.Parse(input.UserID)
	if err != nil {
		return nil, errors.New("invalid user_id")
	}

	_, err = s.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	_, err = s.repo.FindByUserID(userID)
	if err == nil {
		return nil, errors.New("user already has a shop")
	}

	slug := generateUniqueSlug(utils.GenerateSlug(input.Name), func(candidate string) bool {
		_, err := s.repo.FindBySlug(candidate)
		return err == nil
	})

	shop := &models.Shop{
		UserID:      userID,
		Name:        input.Name,
		Slug:        slug,
		Description: input.Description,
		Logo:        input.Logo,
		Address:     input.Address,
		Phone:       input.Phone,
		Status:      "active",
	}

	if err := s.repo.Create(shop); err != nil {
		return nil, err
	}
	return shop, nil
}

func (s *ShopService) GetByID(id uuid.UUID) (*models.Shop, error) {
	shop, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("shop not found")
		}
		return nil, err
	}
	return shop, nil
}

func (s *ShopService) GetByUserID(userID uuid.UUID) (*models.Shop, error) {
	shop, err := s.repo.FindByUserID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("shop not found")
		}
		return nil, err
	}
	return shop, nil
}

func (s *ShopService) GetAll(page, limit int) ([]models.Shop, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	return s.repo.FindAll(page, limit)
}

func (s *ShopService) Update(id uuid.UUID, input UpdateShopInput) (*models.Shop, error) {
	shop, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("shop not found")
		}
		return nil, err
	}

	if input.Name != nil {
		shop.Name = *input.Name
		shop.Slug = generateUniqueSlug(utils.GenerateSlug(*input.Name), func(candidate string) bool {
			_, err := s.repo.FindBySlugIncludingDeleted(candidate)
			return err == nil
		})
	}
	if input.Description != nil {
		shop.Description = *input.Description
	}
	if input.Logo != nil {
		shop.Logo = *input.Logo
	}
	if input.Address != nil {
		shop.Address = *input.Address
	}
	if input.Phone != nil {
		shop.Phone = *input.Phone
	}
	if input.Status != nil {
		shop.Status = *input.Status
	}

	if err := s.repo.Update(shop); err != nil {
		return nil, err
	}
	return shop, nil
}

func (s *ShopService) Delete(id uuid.UUID) error {
	_, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("shop not found")
		}
		return err
	}
	return s.repo.Delete(id)
}
