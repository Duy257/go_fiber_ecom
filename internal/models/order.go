package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Order struct {
	ID              uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	CustomerID      uuid.UUID      `gorm:"type:uuid;index;not null" json:"customer_id"`
	Customer        Customer       `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	ShopID          uuid.UUID      `gorm:"type:uuid;index;not null" json:"shop_id"`
	Shop            Shop           `gorm:"foreignKey:ShopID" json:"shop,omitempty"`
	OrderNumber     string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"order_number"`
	Status          string         `gorm:"type:varchar(20);default:pending" json:"status"`
	SubTotal        float64        `gorm:"type:decimal(12,2);not null" json:"sub_total"`
	ShippingFee     float64        `gorm:"type:decimal(12,2);default:0" json:"shipping_fee"`
	TotalAmount     float64        `gorm:"type:decimal(12,2);not null" json:"total_amount"`
	ShippingAddress    map[string]interface{} `gorm:"type:jsonb;serializer:json;not null" json:"shipping_address"`
	ShippingLatitude   float64                `gorm:"type:decimal(10,7)" json:"shipping_latitude,omitempty"`
	ShippingLongitude  float64                `gorm:"type:decimal(10,7)" json:"shipping_longitude,omitempty"`
	ShippingDistanceKm float64                `gorm:"type:decimal(8,2)" json:"shipping_distance_km,omitempty"`
	Note               string                 `gorm:"type:text" json:"note,omitempty"`
	Items           []OrderItem    `gorm:"foreignKey:OrderID" json:"items,omitempty"`
	StatusHistory   []OrderStatusHistory `gorm:"foreignKey:OrderID" json:"status_history,omitempty"`
	Payment         *Payment       `gorm:"foreignKey:OrderID" json:"payment,omitempty"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type OrderItem struct {
	ID          uuid.UUID       `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrderID     uuid.UUID       `gorm:"type:uuid;index;not null" json:"order_id"`
	Order       Order           `gorm:"foreignKey:OrderID" json:"-"`
	ProductID   uuid.UUID       `gorm:"type:uuid;not null" json:"product_id"`
	Product     Product         `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	VariantID   *uuid.UUID      `gorm:"type:uuid" json:"variant_id,omitempty"`
	Variant     *ProductVariant `gorm:"foreignKey:VariantID" json:"variant,omitempty"`
	ProductName string          `gorm:"type:varchar(255);not null" json:"product_name"`
	VariantName string          `gorm:"type:varchar(255)" json:"variant_name,omitempty"`
	Price       float64         `gorm:"type:decimal(12,2);not null" json:"price"`
	Quantity    int             `gorm:"not null" json:"quantity"`
	Total       float64         `gorm:"type:decimal(12,2);not null" json:"total"`
}

type OrderStatusHistory struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrderID   uuid.UUID `gorm:"type:uuid;index;not null" json:"order_id"`
	Status    string    `gorm:"type:varchar(20);not null" json:"status"`
	Note      string    `gorm:"type:text" json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

