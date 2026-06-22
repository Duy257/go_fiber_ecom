package repositories

import (
	"go-fiber/internal/models"

	"gorm.io/gorm"
)

type ShippingConfigRepository struct {
	db *gorm.DB
}

func NewShippingConfigRepository(db *gorm.DB) *ShippingConfigRepository {
	return &ShippingConfigRepository{db: db}
}

func (r *ShippingConfigRepository) Get() (*models.ShippingConfig, error) {
	var config models.ShippingConfig
	err := r.db.First(&config).Error
	return &config, err
}

func (r *ShippingConfigRepository) Update(config *models.ShippingConfig) error {
	return r.db.Save(config).Error
}
