package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Category struct {
	ID          uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name        string         `gorm:"type:varchar(255);not null" json:"name"`
	Slug        string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"slug"`
	Description string         `gorm:"type:text" json:"description,omitempty"`
	Image       string         `gorm:"type:varchar(500)" json:"image,omitempty"`
	ParentID    *uuid.UUID     `gorm:"type:uuid;index" json:"parent_id,omitempty"`
	Parent      *Category      `gorm:"foreignKey:ParentID;OnDelete:SET NULL" json:"parent,omitempty"`
	Children    []Category     `gorm:"foreignKey:ParentID;OnDelete:SET NULL" json:"children,omitempty"`
	SortOrder   int            `gorm:"default:0" json:"sort_order"`
	Status      string         `gorm:"type:varchar(20);default:active" json:"status"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type ProductCategory struct {
	ProductID  uuid.UUID `gorm:"type:uuid;primaryKey"`
	CategoryID uuid.UUID `gorm:"type:uuid;primaryKey"`
}
