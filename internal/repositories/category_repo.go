package repositories

import (
	"go-fiber/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) Create(category *models.Category) error {
	return r.db.Create(category).Error
}

func (r *CategoryRepository) FindByID(id uuid.UUID) (*models.Category, error) {
	var category models.Category
	err := r.db.Preload("Children").First(&category, "id = ?", id).Error
	return &category, err
}

func (r *CategoryRepository) FindBySlug(slug string) (*models.Category, error) {
	var category models.Category
	err := r.db.Preload("Children").Where("slug = ?", slug).First(&category).Error
	return &category, err
}

func (r *CategoryRepository) FindAll(parentID *uuid.UUID, page, limit int) ([]models.Category, int64, error) {
	var categories []models.Category
	var total int64

	query := r.db.Model(&models.Category{})
	if parentID != nil {
		query = query.Where("parent_id = ?", parentID)
	} else {
		query = query.Where("parent_id IS NULL")
	}

	query.Count(&total)
	err := query.Preload("Children").Offset((page - 1) * limit).Limit(limit).Order("sort_order ASC, created_at DESC").Find(&categories).Error
	return categories, total, err
}

func (r *CategoryRepository) Update(category *models.Category) error {
	return r.db.Save(category).Error
}

func (r *CategoryRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Category{}, "id = ?", id).Error
}

func (r *CategoryRepository) FindBySlugIncludingDeleted(slug string) (*models.Category, error) {
	var category models.Category
	err := r.db.Unscoped().Where("slug = ?", slug).First(&category).Error
	return &category, err
}

func (r *CategoryRepository) HasProducts(id uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&models.ProductCategory{}).Where("category_id = ?", id).Count(&count).Error
	return count > 0, err
}
