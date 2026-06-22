package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Shop struct {
	ID          uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID      uuid.UUID      `gorm:"type:uuid;uniqueIndex;not null" json:"user_id"`
	User        User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Name        string         `gorm:"type:varchar(255);not null" json:"name"`
	Slug        string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"slug"`
	Description string         `gorm:"type:text" json:"description,omitempty"`
	Logo        string         `gorm:"type:varchar(500)" json:"logo,omitempty"`
	Address     string         `gorm:"type:varchar(500)" json:"address,omitempty"`
	Latitude    float64        `gorm:"type:decimal(10,7)" json:"latitude,omitempty"`
	Longitude   float64        `gorm:"type:decimal(10,7)" json:"longitude,omitempty"`
	Phone       string         `gorm:"type:varchar(20)" json:"phone,omitempty"`
	Status      string         `gorm:"type:varchar(20);default:active" json:"status"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}
