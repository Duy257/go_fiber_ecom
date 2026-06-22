package repositories

import (
	"go-fiber/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ShopRepository struct {
	db *gorm.DB
}

func NewShopRepository(db *gorm.DB) *ShopRepository {
	return &ShopRepository{db: db}
}

func (r *ShopRepository) Create(shop *models.Shop) error {
	return r.db.Create(shop).Error
}

func (r *ShopRepository) FindByID(id uuid.UUID) (*models.Shop, error) {
	var shop models.Shop
	err := r.db.Preload("User").First(&shop, "id = ?", id).Error
	return &shop, err
}

func (r *ShopRepository) FindByUserID(userID uuid.UUID) (*models.Shop, error) {
	var shop models.Shop
	err := r.db.Where("user_id = ?", userID).First(&shop).Error
	return &shop, err
}

func (r *ShopRepository) FindBySlug(slug string) (*models.Shop, error) {
	var shop models.Shop
	err := r.db.Where("slug = ?", slug).First(&shop).Error
	return &shop, err
}

func (r *ShopRepository) FindBySlugIncludingDeleted(slug string) (*models.Shop, error) {
	var shop models.Shop
	err := r.db.Unscoped().Where("slug = ?", slug).First(&shop).Error
	return &shop, err
}

func (r *ShopRepository) FindAll(page, limit int) ([]models.Shop, int64, error) {
	var shops []models.Shop
	var total int64

	r.db.Model(&models.Shop{}).Count(&total)
	err := r.db.Preload("User").Offset((page - 1) * limit).Limit(limit).Order("created_at DESC").Find(&shops).Error
	return shops, total, err
}

func (r *ShopRepository) Update(shop *models.Shop) error {
	return r.db.Save(shop).Error
}

func (r *ShopRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Shop{}, "id = ?", id).Error
}
