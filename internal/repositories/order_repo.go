package repositories

import (
	"fmt"
	"time"

	"go-fiber/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(order *models.Order) error {
	return r.db.Create(order).Error
}

func (r *OrderRepository) FindByID(id uuid.UUID) (*models.Order, error) {
	var order models.Order
	err := r.db.Preload("Customer").Preload("Shop").Preload("Items").Preload("StatusHistory").Preload("Payment").
		First(&order, "id = ?", id).Error
	return &order, err
}

func (r *OrderRepository) FindByOrderNumber(orderNumber string) (*models.Order, error) {
	var order models.Order
	err := r.db.Preload("Customer").Preload("Shop").Preload("Items").Preload("StatusHistory").Preload("Payment").
		Where("order_number = ?", orderNumber).First(&order).Error
	return &order, err
}

func (r *OrderRepository) FindByCustomerID(customerID uuid.UUID, page, limit int) ([]models.Order, int64, error) {
	var orders []models.Order
	var total int64

	query := r.db.Model(&models.Order{}).Where("customer_id = ?", customerID)
	query.Count(&total)
	err := query.Preload("Shop").Preload("Items").
		Offset((page - 1) * limit).Limit(limit).Order("created_at DESC").Find(&orders).Error
	return orders, total, err
}

func (r *OrderRepository) FindByShopID(shopID uuid.UUID, status *string, page, limit int) ([]models.Order, int64, error) {
	var orders []models.Order
	var total int64

	query := r.db.Model(&models.Order{}).Where("shop_id = ?", shopID)
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	query.Count(&total)
	err := query.Preload("Customer").Preload("Items").
		Offset((page - 1) * limit).Limit(limit).Order("created_at DESC").Find(&orders).Error
	return orders, total, err
}

func (r *OrderRepository) Update(order *models.Order) error {
	return r.db.Save(order).Error
}

func (r *OrderRepository) CreateStatusHistory(history *models.OrderStatusHistory) error {
	return r.db.Create(history).Error
}

func (r *OrderRepository) GenerateOrderNumber() string {
	return fmt.Sprintf("ORD-%s-%s", time.Now().Format("20060102"), uuid.New().String()[:8])
}

func (r *OrderRepository) Transaction(fn func(tx *gorm.DB) error) error {
	return r.db.Transaction(fn)
}
