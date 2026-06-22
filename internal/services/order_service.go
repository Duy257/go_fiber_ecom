package services

import (
	"errors"
	"fmt"

	"go-fiber/internal/models"
	"go-fiber/internal/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderService struct {
	repo         *repositories.OrderRepository
	paymentSvc   *PaymentService
	customerRepo *repositories.CustomerRepository
	productRepo  *repositories.ProductRepository
}

func NewOrderService(
	repo *repositories.OrderRepository,
	paymentSvc *PaymentService,
	customerRepo *repositories.CustomerRepository,
	productRepo *repositories.ProductRepository,
) *OrderService {
	return &OrderService{
		repo:         repo,
		paymentSvc:   paymentSvc,
		customerRepo: customerRepo,
		productRepo:  productRepo,
	}
}

type CreateOrderInput struct {
	CustomerID      string                 `json:"customer_id" validate:"required"`
	ShopID          string                 `json:"shop_id" validate:"required"`
	Items           []CreateOrderItemInput `json:"items" validate:"required,min=1"`
	ShippingFee     float64                `json:"shipping_fee"`
	ShippingAddress map[string]interface{} `json:"shipping_address" validate:"required"`
	Note            string                 `json:"note"`
	PaymentMethod   string                 `json:"payment_method" validate:"required,oneof=cod bank_transfer e_wallet"`
}

type CreateOrderItemInput struct {
	ProductID string `json:"product_id" validate:"required"`
	VariantID string `json:"variant_id"`
	Quantity  int    `json:"quantity" validate:"required,gt=0"`
}

type UpdateOrderStatusInput struct {
	Status string `json:"status" validate:"required,oneof=confirmed shipping delivered cancelled"`
	Note   string `json:"note"`
}

func (s *OrderService) Create(input CreateOrderInput) (*models.Order, error) {
	customerID, err := uuid.Parse(input.CustomerID)
	if err != nil {
		return nil, errors.New("invalid customer_id")
	}

	_, err = s.customerRepo.FindByID(customerID)
	if err != nil {
		return nil, errors.New("customer not found")
	}

	shopID, err := uuid.Parse(input.ShopID)
	if err != nil {
		return nil, errors.New("invalid shop_id")
	}

	orderNumber := s.repo.GenerateOrderNumber()

	order := &models.Order{
		CustomerID:      customerID,
		ShopID:          shopID,
		OrderNumber:     orderNumber,
		Status:          "pending",
		ShippingFee:     input.ShippingFee,
		ShippingAddress: input.ShippingAddress,
		Note:            input.Note,
	}

	var subTotal float64

	for _, item := range input.Items {
		productID, err := uuid.Parse(item.ProductID)
		if err != nil {
			return nil, errors.New("invalid product_id")
		}

		product, err := s.productRepo.FindByID(productID)
		if err != nil {
			return nil, errors.New("product not found")
		}

		orderItem := models.OrderItem{
			ProductID:   productID,
			ProductName: product.Name,
			Quantity:    item.Quantity,
		}

		if item.VariantID != "" {
			variantID, err := uuid.Parse(item.VariantID)
			if err != nil {
				return nil, errors.New("invalid variant_id")
			}

			var variant *models.ProductVariant
			for _, v := range product.Variants {
				if v.ID == variantID {
					variant = &v
					break
				}
			}
			if variant == nil {
				return nil, errors.New("variant not found")
			}

			orderItem.VariantID = &variantID
			orderItem.VariantName = variant.Name
			orderItem.Price = variant.Price
		} else {
			orderItem.Price = product.Price
		}

		orderItem.Total = orderItem.Price * float64(item.Quantity)
		subTotal += orderItem.Total

		order.Items = append(order.Items, orderItem)
	}

	order.SubTotal = subTotal
	order.TotalAmount = subTotal + input.ShippingFee

	err = s.repo.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(order).Error; err != nil {
			return err
		}

		for _, item := range order.Items {
			if item.VariantID != nil {
				result := tx.Model(&models.ProductVariant{}).
					Where("id = ? AND stock >= ?", *item.VariantID, item.Quantity).
					Update("stock", gorm.Expr("stock - ?", item.Quantity))
				if result.Error != nil {
					return result.Error
				}
				if result.RowsAffected == 0 {
					return fmt.Errorf("insufficient stock for variant %s", *item.VariantID)
				}
			}
		}

		orderID := order.ID
		_, err = s.paymentSvc.CreatePayment(tx, CreatePaymentInput{
			Type:    "order",
			Method:  input.PaymentMethod,
			Amount:  order.TotalAmount,
			OrderID: &orderID,
		})
		if err != nil {
			return err
		}

		history := &models.OrderStatusHistory{
			OrderID: order.ID,
			Status:  "pending",
			Note:    "Order created",
		}
		if err := tx.Create(history).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return order, nil
}

func (s *OrderService) GetByID(id uuid.UUID) (*models.Order, error) {
	order, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("order not found")
		}
		return nil, err
	}
	return order, nil
}

func (s *OrderService) GetByCustomerID(customerID uuid.UUID, page, limit int) ([]models.Order, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	return s.repo.FindByCustomerID(customerID, page, limit)
}

func (s *OrderService) GetByShopID(shopID uuid.UUID, status *string, page, limit int) ([]models.Order, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	return s.repo.FindByShopID(shopID, status, page, limit)
}

func (s *OrderService) UpdateStatus(id uuid.UUID, input UpdateOrderStatusInput) (*models.Order, error) {
	order, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("order not found")
		}
		return nil, err
	}

	validTransitions := map[string][]string{
		"pending":   {"confirmed", "cancelled"},
		"confirmed": {"shipping", "cancelled"},
		"shipping":  {"delivered"},
	}

	allowed, ok := validTransitions[order.Status]
	if !ok {
		return nil, errors.New("cannot change status from current state")
	}

	valid := false
	for _, s := range allowed {
		if s == input.Status {
			valid = true
			break
		}
	}
	if !valid {
		return nil, errors.New("invalid status transition")
	}

	err = s.repo.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Order{}).Where("id = ?", order.ID).Update("status", input.Status).Error; err != nil {
			return err
		}

		history := &models.OrderStatusHistory{
			OrderID: order.ID,
			Status:  input.Status,
			Note:    input.Note,
		}
		if err := tx.Create(history).Error; err != nil {
			return err
		}

		if input.Status == "delivered" {
			if err := s.paymentSvc.MarkAsPaid(tx, order.ID); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.repo.FindByID(order.ID)
}

func (s *OrderService) Cancel(id uuid.UUID, note string) (*models.Order, error) {
	order, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("order not found")
		}
		return nil, err
	}

	if order.Status != "pending" && order.Status != "confirmed" {
		return nil, errors.New("only pending or confirmed orders can be cancelled")
	}

	err = s.repo.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Order{}).Where("id = ?", order.ID).Update("status", "cancelled").Error; err != nil {
			return err
		}

		for _, item := range order.Items {
			if item.VariantID != nil {
				if err := tx.Model(&models.ProductVariant{}).Where("id = ?", *item.VariantID).
					Update("stock", gorm.Expr("stock + ?", item.Quantity)).Error; err != nil {
					return err
				}
			}
		}

		payment, err := s.paymentSvc.FindByOrderID(tx, order.ID)
		if err == nil {
			if err := s.paymentSvc.CancelPayment(tx, payment.ID); err != nil {
				return err
			}
		}

		history := &models.OrderStatusHistory{
			OrderID: order.ID,
			Status:  "cancelled",
			Note:    note,
		}
		if err := tx.Create(history).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.repo.FindByID(order.ID)
}
