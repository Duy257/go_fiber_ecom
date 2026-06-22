package models

import "time"

type ShippingConfig struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	BaseFee       float64   `gorm:"type:decimal(12,2);not null" json:"base_fee"`
	PerKmRate     float64   `gorm:"type:decimal(12,2);not null" json:"per_km_rate"`
	MaxDistanceKm float64   `gorm:"type:decimal(8,2);not null" json:"max_distance_km"`
	UpdatedAt     time.Time `json:"updated_at"`
}
