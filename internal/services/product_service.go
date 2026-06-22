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

type ProductService struct {
	repo         *repositories.ProductRepository
	shopRepo     *repositories.ShopRepository
	categoryRepo *repositories.CategoryRepository
}

func NewProductService(repo *repositories.ProductRepository, shopRepo *repositories.ShopRepository, categoryRepo *repositories.CategoryRepository) *ProductService {
	return &ProductService{repo: repo, shopRepo: shopRepo, categoryRepo: categoryRepo}
}

type CreateProductInput struct {
	ShopID      string               `json:"shop_id" validate:"required"`
	Name        string               `json:"name" validate:"required"`
	Description string               `json:"description"`
	Price       float64              `json:"price" validate:"required,gt=0"`
	CategoryIDs []string             `json:"category_ids"`
	Variants    []CreateVariantInput `json:"variants" validate:"required,min=1"`
	Images      []CreateImageInput   `json:"images"`
}

type CreateVariantInput struct {
	Name       string                 `json:"name" validate:"required"`
	SKU        *string                `json:"sku"`
	Price      float64                `json:"price" validate:"required,gt=0"`
	Stock      int                    `json:"stock" validate:"min=0"`
	Attributes map[string]interface{} `json:"attributes"`
}

type CreateImageInput struct {
	URL       string `json:"url" validate:"required"`
	SortOrder int    `json:"sort_order"`
}

type UpdateProductInput struct {
	Name        *string             `json:"name"`
	Description *string             `json:"description"`
	Price       *float64            `json:"price"`
	Status      *string             `json:"status"`
	CategoryIDs []string            `json:"category_ids"`
	Variants    []CreateVariantInput `json:"variants"`
	Images      []CreateImageInput   `json:"images"`
}

func (s *ProductService) Create(input CreateProductInput) (*models.Product, error) {
	shopID, err := uuid.Parse(input.ShopID)
	if err != nil {
		return nil, errors.New("invalid shop_id")
	}

	_, err = s.shopRepo.FindByID(shopID)
	if err != nil {
		return nil, errors.New("shop not found")
	}

	if len(input.CategoryIDs) > 0 {
		for _, catIDStr := range input.CategoryIDs {
			catID, err := uuid.Parse(catIDStr)
			if err != nil {
				return nil, errors.New("invalid category_id")
			}
			_, err = s.categoryRepo.FindByID(catID)
			if err != nil {
				return nil, fmt.Errorf("category %s not found", catIDStr)
			}
		}
	}

	slug := utils.GenerateSlug(input.Name)

	if _, err := s.repo.FindBySlugIncludingDeleted(slug); err == nil {
		return nil, errors.New("product with this name already exists")
	}

	product := &models.Product{
		ShopID:      shopID,
		Name:        input.Name,
		Slug:        slug,
		Description: input.Description,
		Price:       input.Price,
		Status:      "active",
	}

	for _, v := range input.Variants {
		variant := models.ProductVariant{
			Name:       v.Name,
			SKU:        v.SKU,
			Price:      v.Price,
			Stock:      v.Stock,
			Attributes: v.Attributes,
		}
		product.Variants = append(product.Variants, variant)
	}

	for _, img := range input.Images {
		image := models.ProductImage{
			URL:       img.URL,
			SortOrder: img.SortOrder,
		}
		product.Images = append(product.Images, image)
	}

	if len(input.CategoryIDs) > 0 {
		for _, catIDStr := range input.CategoryIDs {
			catID, err := uuid.Parse(catIDStr)
			if err != nil {
				return nil, errors.New("invalid category_id")
			}
			product.Categories = append(product.Categories, models.Category{ID: catID})
		}
	}

	if err := s.repo.Create(product); err != nil {
		return nil, err
	}
	return product, nil
}

func (s *ProductService) GetByID(id uuid.UUID) (*models.Product, error) {
	product, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("product not found")
		}
		return nil, err
	}
	return product, nil
}

func (s *ProductService) GetAll(shopID, categoryID *string, page, limit int) ([]models.Product, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	var sid, cid *uuid.UUID
	if shopID != nil {
		id, err := uuid.Parse(*shopID)
		if err != nil {
			return nil, 0, errors.New("invalid shop_id")
		}
		sid = &id
	}
	if categoryID != nil {
		id, err := uuid.Parse(*categoryID)
		if err != nil {
			return nil, 0, errors.New("invalid category_id")
		}
		cid = &id
	}

	return s.repo.FindAll(sid, cid, page, limit)
}

func (s *ProductService) Update(id uuid.UUID, input UpdateProductInput) (*models.Product, error) {
	product, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("product not found")
		}
		return nil, err
	}

	if input.CategoryIDs != nil && len(input.CategoryIDs) > 0 {
		for _, catIDStr := range input.CategoryIDs {
			catID, err := uuid.Parse(catIDStr)
			if err != nil {
				return nil, errors.New("invalid category_id")
			}
			_, err = s.categoryRepo.FindByID(catID)
			if err != nil {
				return nil, fmt.Errorf("category %s not found", catIDStr)
			}
		}
	}

	if input.Name != nil {
		product.Name = *input.Name
		newSlug := utils.GenerateSlug(*input.Name)
		if newSlug != product.Slug {
			if _, err := s.repo.FindBySlugIncludingDeleted(newSlug); err == nil {
				return nil, errors.New("product with this name already exists")
			}
		}
		product.Slug = newSlug
	}
	if input.Description != nil {
		product.Description = *input.Description
	}
	if input.Price != nil {
		product.Price = *input.Price
	}
	if input.Status != nil {
		product.Status = *input.Status
	}

	if input.CategoryIDs != nil {
		var categories []models.Category
		for _, catIDStr := range input.CategoryIDs {
			catID, err := uuid.Parse(catIDStr)
			if err != nil {
				return nil, errors.New("invalid category_id")
			}
			categories = append(categories, models.Category{ID: catID})
		}
		product.Categories = categories
	}

	if err := s.repo.Update(product); err != nil {
		return nil, err
	}

	// Replace variants if provided
	if input.Variants != nil {
		variants := make([]models.ProductVariant, len(input.Variants))
		for i, v := range input.Variants {
			variants[i] = models.ProductVariant{
				Name:       v.Name,
				SKU:        v.SKU,
				Price:      v.Price,
				Stock:      v.Stock,
				Attributes: v.Attributes,
			}
		}
		if err := s.repo.ReplaceVariants(product.ID, variants); err != nil {
			return nil, err
		}
	}

	// Replace images if provided
	if input.Images != nil {
		images := make([]models.ProductImage, len(input.Images))
		for i, img := range input.Images {
			images[i] = models.ProductImage{
				URL:       img.URL,
				SortOrder: img.SortOrder,
			}
		}
		if err := s.repo.ReplaceImages(product.ID, images); err != nil {
			return nil, err
		}
	}

	return s.repo.FindByID(product.ID)
}

func (s *ProductService) Delete(id uuid.UUID) error {
	_, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("product not found")
		}
		return err
	}
	return s.repo.Delete(id)
}
