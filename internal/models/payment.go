package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	PaymentStatusPending  = "pending"
	PaymentStatusPaid     = "paid"
	PaymentStatusFailed   = "failed"
	PaymentStatusRefunded = "refunded"
)

type Payment struct {
	ID            uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Type          string         `gorm:"type:varchar(50);not null;default:order" json:"type"`
	Status        string         `gorm:"type:varchar(20);default:pending" json:"status"`
	Method        string         `gorm:"type:varchar(50);not null" json:"method"`
	Amount        float64        `gorm:"type:decimal(12,2);not null" json:"amount"`
	TransactionID string         `gorm:"type:varchar(255)" json:"transaction_id,omitempty"`
	PaidAt        *time.Time     `json:"paid_at,omitempty"`

	OrderID       *uuid.UUID     `gorm:"type:uuid;uniqueIndex" json:"order_id,omitempty"`
	Order         *Order         `gorm:"foreignKey:OrderID" json:"-"`

	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}
