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

type CategoryService struct {
	repo *repositories.CategoryRepository
}

func NewCategoryService(repo *repositories.CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

type CreateCategoryInput struct {
	Name        string  `json:"name" validate:"required"`
	Description string  `json:"description"`
	Image       string  `json:"image"`
	ParentID    *string `json:"parent_id"`
	SortOrder   int     `json:"sort_order"`
}

type UpdateCategoryInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Image       *string `json:"image"`
	ParentID    *string `json:"parent_id"`
	SortOrder   *int    `json:"sort_order"`
	Status      *string `json:"status"`
}

func (s *CategoryService) Create(input CreateCategoryInput) (*models.Category, error) {
	slug := utils.GenerateSlug(input.Name)

	_, err := s.repo.FindBySlugIncludingDeleted(slug)
	if err == nil {
		return nil, errors.New("category with this name already exists")
	}

	category := &models.Category{
		Name:        input.Name,
		Slug:        slug,
		Description: input.Description,
		Image:       input.Image,
		SortOrder:   input.SortOrder,
		Status:      "active",
	}

	if input.ParentID != nil {
		parentID, err := uuid.Parse(*input.ParentID)
		if err != nil {
			return nil, errors.New("invalid parent_id")
		}
		parent, err := s.repo.FindByID(parentID)
		if err != nil {
			return nil, errors.New("parent category not found")
		}
		if parent.ParentID != nil {
			return nil, errors.New("cannot nest more than 2 levels")
		}
		category.ParentID = &parentID
	}

	if err := s.repo.Create(category); err != nil {
		return nil, err
	}
	return category, nil
}

func (s *CategoryService) GetByID(id uuid.UUID) (*models.Category, error) {
	category, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("category not found")
		}
		return nil, err
	}
	return category, nil
}

func (s *CategoryService) GetAll(parentID *string, page, limit int) ([]models.Category, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	var pid *uuid.UUID
	if parentID != nil {
		id, err := uuid.Parse(*parentID)
		if err != nil {
			return nil, 0, errors.New("invalid parent_id")
		}
		pid = &id
	}

	return s.repo.FindAll(pid, page, limit)
}

func (s *CategoryService) Update(id uuid.UUID, input UpdateCategoryInput) (*models.Category, error) {
	category, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("category not found")
		}
		return nil, err
	}

	if input.Name != nil {
		category.Name = *input.Name
		newSlug := utils.GenerateSlug(*input.Name)
		if newSlug != category.Slug {
			_, err := s.repo.FindBySlugIncludingDeleted(newSlug)
			if err == nil {
				return nil, errors.New("category with this name already exists")
			}
			category.Slug = newSlug
		} else {
			category.Slug = newSlug
		}
	}
	if input.Description != nil {
		category.Description = *input.Description
	}
	if input.Image != nil {
		category.Image = *input.Image
	}
	if input.SortOrder != nil {
		category.SortOrder = *input.SortOrder
	}
	if input.Status != nil {
		category.Status = *input.Status
	}
	if input.ParentID != nil {
		if *input.ParentID == "" {
			category.ParentID = nil
		} else {
			parentID, err := uuid.Parse(*input.ParentID)
			if err != nil {
				return nil, errors.New("invalid parent_id")
			}
			if parentID == id {
				return nil, errors.New("category cannot be its own parent")
			}
			parent, err := s.repo.FindByID(parentID)
			if err != nil {
				return nil, errors.New("parent category not found")
			}
			if parent.ParentID != nil {
				return nil, errors.New("cannot nest more than 2 levels")
			}
			category.ParentID = &parentID
		}
	}

	if err := s.repo.Update(category); err != nil {
		return nil, err
	}
	return category, nil
}

func (s *CategoryService) Delete(id uuid.UUID) error {
	hasProducts, err := s.repo.HasProducts(id)
	if err != nil {
		return err
	}
	if hasProducts {
		return fmt.Errorf("cannot delete category with products")
	}
	return s.repo.Delete(id)
}
