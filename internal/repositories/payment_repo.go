package repositories

import (
	"go-fiber/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentFilter struct {
	Type   string
	Status string
	Method string
	Page   int
	Limit  int
}

type PaymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) FindByID(id uuid.UUID) (*models.Payment, error) {
	var payment models.Payment
	err := r.db.First(&payment, "id = ?", id).Error
	return &payment, err
}

func (r *PaymentRepository) FindByOrderID(orderID uuid.UUID) (*models.Payment, error) {
	var payment models.Payment
	err := r.db.Where("order_id = ?", orderID).First(&payment).Error
	return &payment, err
}

func (r *PaymentRepository) FindAll(filter PaymentFilter) ([]models.Payment, int64, error) {
	var payments []models.Payment
	var total int64

	query := r.db.Model(&models.Payment{})
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Method != "" {
		query = query.Where("method = ?", filter.Method)
	}
	query.Count(&total)

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 10
	}
	err := query.Offset((filter.Page - 1) * filter.Limit).Limit(filter.Limit).
		Order("created_at DESC").Find(&payments).Error
	return payments, total, err
}

func (r *PaymentRepository) Create(payment *models.Payment) error {
	return r.db.Create(payment).Error
}

func (r *PaymentRepository) Update(payment *models.Payment) error {
	return r.db.Save(payment).Error
}
