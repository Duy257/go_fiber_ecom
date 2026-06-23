package services

import (
	"errors"
	"time"

	"go-fiber/internal/models"
	"go-fiber/internal/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrInvalidPaymentType   = errors.New("invalid payment type")
	ErrInvalidPaymentMethod = errors.New("invalid payment method")
	ErrInvalidPaymentAmount = errors.New("payment amount must be greater than 0")
	ErrOrderIDRequired      = errors.New("order_id is required for order payment type")
	ErrOrderIDNotAllowed    = errors.New("order_id must be empty for non-order payment type")
	ErrPaymentNotFound      = errors.New("payment not found")
)

type CreatePaymentInput struct {
	Type    string
	Method  string
	Amount  float64
	OrderID *uuid.UUID
}

type PaymentService struct {
	paymentRepo *repositories.PaymentRepository
}

func NewPaymentService(paymentRepo *repositories.PaymentRepository) *PaymentService {
	return &PaymentService{paymentRepo: paymentRepo}
}

func (s *PaymentService) CreatePayment(tx *gorm.DB, input CreatePaymentInput) (*models.Payment, error) {
	validTypes := map[string]bool{"order": true, "top_up": true, "membership": true}
	if !validTypes[input.Type] {
		return nil, ErrInvalidPaymentType
	}

	validMethods := map[string]bool{"cod": true, "bank_transfer": true, "e_wallet": true}
	if !validMethods[input.Method] {
		return nil, ErrInvalidPaymentMethod
	}

	if input.Amount <= 0 {
		return nil, ErrInvalidPaymentAmount
	}

	if input.Type == "order" && input.OrderID == nil {
		return nil, ErrOrderIDRequired
	}
	if input.Type != "order" && input.OrderID != nil {
		return nil, ErrOrderIDNotAllowed
	}

	payment := &models.Payment{
		Type:    input.Type,
		Method:  input.Method,
		Status:  models.PaymentStatusPending,
		Amount:  input.Amount,
		OrderID: input.OrderID,
	}
	if err := tx.Create(payment).Error; err != nil {
		return nil, err
	}
	return payment, nil
}

func (s *PaymentService) MarkAsPaid(tx *gorm.DB, orderID uuid.UUID) error {
	var payment models.Payment
	if err := tx.Where("order_id = ?", orderID).First(&payment).Error; err != nil {
		return err
	}
	if payment.Method == "cod" && payment.Status == models.PaymentStatusPending {
		now := time.Now()
		if err := tx.Model(&payment).Updates(map[string]interface{}{
			"status":  models.PaymentStatusPaid,
			"paid_at": &now,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *PaymentService) CancelPayment(tx *gorm.DB, paymentID uuid.UUID) error {
	var payment models.Payment
	if err := tx.First(&payment, "id = ?", paymentID).Error; err != nil {
		return err
	}
	newStatus := models.PaymentStatusFailed
	if payment.Status == models.PaymentStatusPaid {
		newStatus = models.PaymentStatusRefunded
	}
	if err := tx.Model(&payment).Update("status", newStatus).Error; err != nil {
		return err
	}
	return nil
}

func (s *PaymentService) FindByOrderID(tx *gorm.DB, orderID uuid.UUID) (*models.Payment, error) {
	var payment models.Payment
	if err := tx.Where("order_id = ?", orderID).First(&payment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPaymentNotFound
		}
		return nil, err
	}
	return &payment, nil
}
