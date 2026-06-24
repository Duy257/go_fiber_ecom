package services

import (
	"errors"
	"fmt"
	"time"

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
	shippingSvc  *ShippingService
}

func NewOrderService(
	repo *repositories.OrderRepository,
	paymentSvc *PaymentService,
	customerRepo *repositories.CustomerRepository,
	productRepo *repositories.ProductRepository,
	shippingSvc *ShippingService,
) *OrderService {
	return &OrderService{
		repo:         repo,
		paymentSvc:   paymentSvc,
		customerRepo: customerRepo,
		productRepo:  productRepo,
		shippingSvc:  shippingSvc,
	}
}

type CreateOrderInput struct {
	CustomerID        string                 `json:"customer_id" validate:"required"`
	ShopID            string                 `json:"shop_id" validate:"required"`
	Items             []CreateOrderItemInput `json:"items" validate:"required,min=1"`
	ShippingFee       *float64               `json:"shipping_fee"`
	ShippingAddress   map[string]interface{} `json:"shipping_address" validate:"required"`
	ShippingLatitude  float64                `json:"shipping_latitude" validate:"required,min=-90,max=90"`
	ShippingLongitude float64                `json:"shipping_longitude" validate:"required,min=-180,max=180"`
	Note              string                 `json:"note"`
	PaymentMethod     string                 `json:"payment_method" validate:"required,oneof=cod bank_transfer e_wallet"`
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

	// Calculate shipping fee
	shippingResult, err := s.shippingSvc.Calculate(shopID, input.ShippingLatitude, input.ShippingLongitude)
	if err != nil {
		return nil, err
	}

	finalShippingFee := shippingResult.TotalFee
	if input.ShippingFee != nil {
		if *input.ShippingFee < finalShippingFee {
			return nil, errors.New("SHIPPING_FEE_OVERRIDE_TOO_LOW")
		}
		finalShippingFee = *input.ShippingFee
	}

	orderNumber := s.repo.GenerateOrderNumber()

	order := &models.Order{
		CustomerID:         customerID,
		ShopID:             shopID,
		OrderNumber:        orderNumber,
		Status:             "pending",
		ShippingFee:        finalShippingFee,
		ShippingAddress:    input.ShippingAddress,
		ShippingLatitude:   input.ShippingLatitude,
		ShippingLongitude:  input.ShippingLongitude,
		ShippingDistanceKm: shippingResult.DistanceKm,
		Note:               input.Note,
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
			ProductID:     productID,
			ProductName:   product.Name,
			Quantity:      item.Quantity,
			DiscountType:  product.DiscountType,
			DiscountValue: product.DiscountValue,
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
			orderItem.OriginalPrice = variant.Price
		} else {
			orderItem.OriginalPrice = product.Price
		}

		// Apply product-level discount
		discountedPrice, discountAmount := CalculateDiscount(orderItem.OriginalPrice, orderItem.DiscountType, orderItem.DiscountValue)
		orderItem.Price = discountedPrice
		orderItem.DiscountAmount = discountAmount

		orderItem.Total = orderItem.Price * float64(item.Quantity)
		subTotal += orderItem.Total

		order.Items = append(order.Items, orderItem)
	}

	order.SubTotal = subTotal
	order.TotalAmount = subTotal + finalShippingFee

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
		models.OrderStatusPending:   {models.OrderStatusConfirmed, models.OrderStatusCancelled},
		models.OrderStatusConfirmed: {models.OrderStatusShipping, models.OrderStatusCancelled},
		models.OrderStatusShipping:  {models.OrderStatusDelivered},
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
		updates := map[string]interface{}{
			"status": input.Status,
		}
		if input.Status == models.OrderStatusDelivered {
			updates["delivered_at"] = time.Now()
		}

		if err := tx.Model(&models.Order{}).Where("id = ?", order.ID).Updates(updates).Error; err != nil {
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

		if input.Status == models.OrderStatusDelivered {
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

func (s *OrderService) AutoCompleteDeliveredOrders() (int, error) {
	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	return s.AutoCompleteDeliveredOrdersBefore(cutoff)
}

func (s *OrderService) AutoCompleteDeliveredOrdersBefore(cutoff time.Time) (int, error) {
	orders, err := s.repo.FindAutoCompletableDelivered(cutoff)
	if err != nil {
		return 0, err
	}

	completedCount := 0
	for _, order := range orders {
		err := s.repo.Transaction(func(tx *gorm.DB) error {
			rowsAffected, err := s.repo.CompleteDeliveredOrder(tx, order.ID)
			if err != nil {
				return err
			}
			if rowsAffected == 0 {
				return nil
			}

			history := &models.OrderStatusHistory{
				OrderID: order.ID,
				Status:  models.OrderStatusCompleted,
				Note:    "Auto-completed after 7 days without complaint",
			}
			if err := tx.Create(history).Error; err != nil {
				return err
			}

			completedCount++
			return nil
		})
		if err != nil {
			return completedCount, err
		}
	}

	return completedCount, nil
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
