package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Product struct {
	ID          uuid.UUID        `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ShopID      uuid.UUID        `gorm:"type:uuid;index;not null" json:"shop_id"`
	Shop        Shop             `gorm:"foreignKey:ShopID" json:"shop,omitempty"`
	Name        string           `gorm:"type:varchar(255);not null" json:"name"`
	Slug        string           `gorm:"type:varchar(255);uniqueIndex;not null" json:"slug"`
	Description string           `gorm:"type:text" json:"description,omitempty"`
	Images      []ProductImage   `gorm:"foreignKey:ProductID" json:"images,omitempty"`
	Variants    []ProductVariant `gorm:"foreignKey:ProductID" json:"variants,omitempty"`
	Categories  []Category       `gorm:"many2many:product_categories;" json:"categories,omitempty"`
	Price       float64          `gorm:"type:decimal(12,2);not null" json:"price"`
	Status      string           `gorm:"type:varchar(20);default:active" json:"status"`
	DeletedAt   gorm.DeletedAt   `gorm:"index" json:"-"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

type ProductVariant struct {
	ID         uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ProductID  uuid.UUID      `gorm:"type:uuid;index;not null" json:"product_id"`
	Product    Product        `gorm:"foreignKey:ProductID" json:"-"`
	Name       string         `gorm:"type:varchar(255);not null" json:"name"`
	SKU        *string        `gorm:"type:varchar(100);uniqueIndex" json:"sku,omitempty"`
	Price      float64        `gorm:"type:decimal(12,2);not null" json:"price"`
	Stock      int            `gorm:"not null;default:0" json:"stock"`
	Attributes map[string]interface{} `gorm:"type:jsonb;serializer:json" json:"attributes,omitempty"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type ProductImage struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ProductID uuid.UUID `gorm:"type:uuid;index;not null" json:"product_id"`
	URL       string    `gorm:"type:varchar(500);not null" json:"url"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
