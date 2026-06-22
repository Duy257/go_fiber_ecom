package models

import (
	"time"

	"github.com/google/uuid"
)

type Customer struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Email       *string   `gorm:"type:varchar(255);uniqueIndex" json:"email,omitempty"`
	PhoneNumber *string   `gorm:"type:varchar(20);uniqueIndex" json:"phone_number,omitempty"`
	Password    string    `gorm:"type:varchar(255);not null" json:"-"`
	Name        string    `gorm:"type:varchar(255)" json:"name"`
	Status      string    `gorm:"type:varchar(20);default:active" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
