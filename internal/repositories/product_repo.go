package repositories

import (
	"go-fiber/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) Create(product *models.Product) error {
	return r.db.Create(product).Error
}

func (r *ProductRepository) FindByID(id uuid.UUID) (*models.Product, error) {
	var product models.Product
	err := r.db.Preload("Shop").Preload("Variants").Preload("Images").Preload("Categories").First(&product, "id = ?", id).Error
	return &product, err
}

func (r *ProductRepository) FindBySlug(slug string) (*models.Product, error) {
	var product models.Product
	err := r.db.Preload("Shop").Preload("Variants").Preload("Images").Preload("Categories").Where("slug = ?", slug).First(&product).Error
	return &product, err
}

func (r *ProductRepository) FindBySlugIncludingDeleted(slug string) (*models.Product, error) {
	var product models.Product
	err := r.db.Unscoped().Preload("Shop").Preload("Variants").Preload("Images").Preload("Categories").Where("slug = ?", slug).First(&product).Error
	return &product, err
}

func (r *ProductRepository) FindAll(shopID *uuid.UUID, categoryID *uuid.UUID, page, limit int) ([]models.Product, int64, error) {
	var products []models.Product
	var total int64

	query := r.db.Model(&models.Product{})
	if shopID != nil {
		query = query.Where("shop_id = ?", shopID)
	}
	if categoryID != nil {
		query = query.Joins("JOIN product_categories ON product_categories.product_id = products.id").
			Where("product_categories.category_id = ?", categoryID)
	}

	query.Count(&total)
	err := query.Preload("Shop").Preload("Variants").Preload("Images").
		Offset((page - 1) * limit).Limit(limit).Order("created_at DESC").Find(&products).Error
	return products, total, err
}

func (r *ProductRepository) Update(product *models.Product) error {
	return r.db.Save(product).Error
}

func (r *ProductRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Product{}, "id = ?", id).Error
}

func (r *ProductRepository) ReplaceVariants(productID uuid.UUID, variants []models.ProductVariant) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("product_id = ?", productID).Delete(&models.ProductVariant{}).Error; err != nil {
			return err
		}
		for i := range variants {
			variants[i].ProductID = productID
			if err := tx.Create(&variants[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *ProductRepository) ReplaceImages(productID uuid.UUID, images []models.ProductImage) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("product_id = ?", productID).Delete(&models.ProductImage{}).Error; err != nil {
			return err
		}
		for i := range images {
			images[i].ProductID = productID
			if err := tx.Create(&images[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *ProductRepository) UpdateStock(variantID uuid.UUID, quantity int) error {
	return r.db.Model(&models.ProductVariant{}).Where("id = ?", variantID).
		Update("stock", gorm.Expr("stock - ?", quantity)).Error
}

func (r *ProductRepository) RestoreStock(variantID uuid.UUID, quantity int) error {
	return r.db.Model(&models.ProductVariant{}).Where("id = ?", variantID).
		Update("stock", gorm.Expr("stock + ?", quantity)).Error
}
