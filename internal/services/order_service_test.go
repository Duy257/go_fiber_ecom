package services

import (
	"os"
	"testing"
	"time"

	"go-fiber/internal/models"
	"go-fiber/internal/repositories"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func newOrderServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := uuid.NewString() + ".db"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
		os.Remove(dsn)
	})

	createTableSQL := []string{
		`CREATE TABLE "users" (
			"id" text PRIMARY KEY,
			"email" text UNIQUE,
			"phone_number" text UNIQUE,
			"password" text NOT NULL,
			"name" text,
			"role_id" text,
			"status" text DEFAULT 'active',
			"created_at" timestamp,
			"updated_at" timestamp
		)`,
		`CREATE TABLE "customers" (
			"id" text PRIMARY KEY,
			"email" text UNIQUE,
			"phone_number" text UNIQUE,
			"password" text NOT NULL,
			"name" text,
			"status" text DEFAULT 'active',
			"created_at" timestamp,
			"updated_at" timestamp
		)`,
		`CREATE TABLE "shops" (
			"id" text PRIMARY KEY,
			"user_id" text NOT NULL UNIQUE,
			"name" text NOT NULL,
			"slug" text NOT NULL UNIQUE,
			"description" text,
			"logo" text,
			"address" text,
			"latitude" real,
			"longitude" real,
			"phone" text,
			"status" text DEFAULT 'active',
			"deleted_at" timestamp,
			"created_at" timestamp,
			"updated_at" timestamp
		)`,
		`CREATE TABLE "orders" (
			"id" text PRIMARY KEY,
			"customer_id" text NOT NULL,
			"shop_id" text NOT NULL,
			"order_number" text NOT NULL UNIQUE,
			"status" text DEFAULT 'pending',
			"delivered_at" timestamp,
			"has_complaint" integer DEFAULT 0,
			"sub_total" real NOT NULL,
			"shipping_fee" real DEFAULT 0,
			"total_amount" real NOT NULL,
			"shipping_address" text NOT NULL,
			"shipping_latitude" real,
			"shipping_longitude" real,
			"shipping_distance_km" real,
			"note" text,
			"deleted_at" timestamp,
			"created_at" timestamp,
			"updated_at" timestamp
		)`,
		`CREATE TABLE "order_items" (
			"id" text PRIMARY KEY,
			"order_id" text NOT NULL,
			"product_id" text NOT NULL,
			"variant_id" text,
			"product_name" text NOT NULL,
			"variant_name" text,
			"original_price" real NOT NULL DEFAULT 0,
			"price" real NOT NULL,
			"discount_type" text,
			"discount_value" real DEFAULT 0,
			"discount_amount" real DEFAULT 0,
			"quantity" integer NOT NULL,
			"total" real NOT NULL
		)`,
		`CREATE TABLE "order_status_histories" (
			"id" text PRIMARY KEY,
			"order_id" text NOT NULL,
			"status" text NOT NULL,
			"note" text,
			"created_at" timestamp
		)`,
		`CREATE TABLE "payments" (
			"id" text PRIMARY KEY,
			"type" text NOT NULL DEFAULT 'order',
			"status" text DEFAULT 'pending',
			"method" text NOT NULL,
			"amount" real NOT NULL,
			"transaction_id" text,
			"paid_at" timestamp,
			"order_id" text UNIQUE,
			"note" text,
			"deleted_at" timestamp,
			"created_at" timestamp,
			"updated_at" timestamp
		)`,
		`CREATE TABLE "products" (
			"id" text PRIMARY KEY,
			"shop_id" text NOT NULL,
			"name" text NOT NULL,
			"slug" text NOT NULL UNIQUE,
			"description" text,
			"price" real NOT NULL,
			"discount_type" text,
			"discount_value" real DEFAULT 0,
			"status" text DEFAULT 'active',
			"deleted_at" timestamp,
			"created_at" timestamp,
			"updated_at" timestamp
		)`,
		`CREATE TABLE "product_variants" (
			"id" text PRIMARY KEY,
			"product_id" text NOT NULL,
			"name" text NOT NULL,
			"sku" text UNIQUE,
			"price" real NOT NULL,
			"stock" integer NOT NULL DEFAULT 0,
			"attributes" text,
			"deleted_at" timestamp,
			"created_at" timestamp,
			"updated_at" timestamp
		)`,
		`CREATE TABLE "product_categories" (
			"product_id" text NOT NULL,
			"category_id" text NOT NULL,
			PRIMARY KEY ("product_id", "category_id")
		)`,
		`CREATE TABLE "product_images" (
			"id" text PRIMARY KEY,
			"product_id" text NOT NULL,
			"url" text NOT NULL,
			"sort_order" integer DEFAULT 0,
			"created_at" timestamp,
			"updated_at" timestamp
		)`,
		`CREATE TABLE "shipping_configs" (
			"id" text PRIMARY KEY,
			"base_fee" real DEFAULT 10000,
			"per_km_rate" real DEFAULT 3000,
			"max_distance_km" real DEFAULT 30,
			"deleted_at" timestamp,
			"created_at" timestamp,
			"updated_at" timestamp
		)`,
	}
	for _, sql := range createTableSQL {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatalf("create table: %v", err)
		}
	}

	return db
}

func createServiceTestOrder(t *testing.T, db *gorm.DB, status string, deliveredAt *time.Time, hasComplaint bool) models.Order {
	t.Helper()

	order := models.Order{
		ID:              uuid.New(),
		CustomerID:      uuid.New(),
		ShopID:          uuid.New(),
		OrderNumber:     "ORD-TEST-" + uuid.New().String()[:8],
		Status:          status,
		DeliveredAt:     deliveredAt,
		HasComplaint:    hasComplaint,
		SubTotal:        100000,
		ShippingFee:     15000,
		TotalAmount:     115000,
		ShippingAddress: map[string]interface{}{"address": "Test address"},
	}

	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	return order
}

func createDiscountTestProduct(t *testing.T, db *gorm.DB, shopID uuid.UUID, price float64, discountType string, discountValue float64) (models.Product, models.ProductVariant) {
	t.Helper()

	product := models.Product{
		ID:            uuid.New(),
		ShopID:        shopID,
		Name:          "Discount Test Product " + uuid.New().String()[:8],
		Slug:          "discount-test-" + uuid.New().String()[:8],
		Price:         price,
		DiscountType:  discountType,
		DiscountValue: discountValue,
		Status:        "active",
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}

	variant := models.ProductVariant{
		ID:        uuid.New(),
		ProductID: product.ID,
		Name:      "Default",
		Price:     price * 1.25,
		Stock:     100,
	}
	if err := db.Create(&variant).Error; err != nil {
		t.Fatalf("create variant: %v", err)
	}

	return product, variant
}

func TestUpdateStatusToDeliveredSetsDeliveredAt(t *testing.T) {
	db := newOrderServiceTestDB(t)
	orderRepo := repositories.NewOrderRepository(db)
	paymentRepo := repositories.NewPaymentRepository(db)
	paymentSvc := NewPaymentService(paymentRepo)
	orderSvc := NewOrderService(orderRepo, paymentSvc, nil, nil, nil)

	order := createServiceTestOrder(t, db, models.OrderStatusShipping, nil, false)
	payment := models.Payment{
		ID:      uuid.New(),
		Type:    "order",
		Status:  models.PaymentStatusPending,
		Method:  "cod",
		Amount:  order.TotalAmount,
		OrderID: &order.ID,
	}
	if err := db.Create(&payment).Error; err != nil {
		t.Fatalf("create payment: %v", err)
	}

	updated, err := orderSvc.UpdateStatus(order.ID, UpdateOrderStatusInput{
		Status: models.OrderStatusDelivered,
		Note:   "Delivered by courier",
	})
	if err != nil {
		t.Fatalf("UpdateStatus returned error: %v", err)
	}

	if updated.Status != models.OrderStatusDelivered {
		t.Fatalf("status = %q, want %q", updated.Status, models.OrderStatusDelivered)
	}

	if updated.DeliveredAt == nil {
		t.Fatal("DeliveredAt is nil, want timestamp")
	}

	if time.Since(*updated.DeliveredAt) > time.Minute {
		t.Fatalf("DeliveredAt = %v, want recent timestamp", updated.DeliveredAt)
	}

	var history models.OrderStatusHistory
	if err := db.Where("order_id = ? AND status = ?", order.ID, models.OrderStatusDelivered).First(&history).Error; err != nil {
		t.Fatalf("find delivered history: %v", err)
	}
}

func TestAutoCompleteDeliveredOrdersBeforeCompletesOnlyEligibleOrders(t *testing.T) {
	db := newOrderServiceTestDB(t)
	orderRepo := repositories.NewOrderRepository(db)
	orderSvc := NewOrderService(orderRepo, nil, nil, nil, nil)
	now := time.Now().UTC()
	oldDeliveredAt := now.Add(-8 * 24 * time.Hour)
	recentDeliveredAt := now.Add(-6 * 24 * time.Hour)

	eligible := createServiceTestOrder(t, db, models.OrderStatusDelivered, &oldDeliveredAt, false)
	recent := createServiceTestOrder(t, db, models.OrderStatusDelivered, &recentDeliveredAt, false)
	complained := createServiceTestOrder(t, db, models.OrderStatusDelivered, &oldDeliveredAt, true)
	alreadyCompleted := createServiceTestOrder(t, db, models.OrderStatusCompleted, &oldDeliveredAt, false)

	completedCount, err := orderSvc.AutoCompleteDeliveredOrdersBefore(now.Add(-7 * 24 * time.Hour))
	if err != nil {
		t.Fatalf("AutoCompleteDeliveredOrdersBefore returned error: %v", err)
	}

	if completedCount != 1 {
		t.Fatalf("completedCount = %d, want 1", completedCount)
	}

	assertOrderStatus(t, db, eligible.ID, models.OrderStatusCompleted)
	assertOrderStatus(t, db, recent.ID, models.OrderStatusDelivered)
	assertOrderStatus(t, db, complained.ID, models.OrderStatusDelivered)
	assertOrderStatus(t, db, alreadyCompleted.ID, models.OrderStatusCompleted)
}

func TestAutoCompleteDeliveredOrdersBeforeWritesHistory(t *testing.T) {
	db := newOrderServiceTestDB(t)
	orderRepo := repositories.NewOrderRepository(db)
	orderSvc := NewOrderService(orderRepo, nil, nil, nil, nil)
	now := time.Now().UTC()
	oldDeliveredAt := now.Add(-8 * 24 * time.Hour)
	order := createServiceTestOrder(t, db, models.OrderStatusDelivered, &oldDeliveredAt, false)

	completedCount, err := orderSvc.AutoCompleteDeliveredOrdersBefore(now.Add(-7 * 24 * time.Hour))
	if err != nil {
		t.Fatalf("AutoCompleteDeliveredOrdersBefore returned error: %v", err)
	}

	if completedCount != 1 {
		t.Fatalf("completedCount = %d, want 1", completedCount)
	}

	var history models.OrderStatusHistory
	if err := db.Where("order_id = ? AND status = ?", order.ID, models.OrderStatusCompleted).First(&history).Error; err != nil {
		t.Fatalf("find completed history: %v", err)
	}

	if history.Note != "Auto-completed after 7 days without complaint" {
		t.Fatalf("history note = %q, want system auto-complete note", history.Note)
	}
}

func setupOrderDiscountTest(t *testing.T, db *gorm.DB) (uuid.UUID, uuid.UUID, *repositories.ShopRepository) {
	email := "test@test.com"
	customerID := uuid.New()
	userID := uuid.New()
	shopID := uuid.New()

	if err := db.Create(&models.User{ID: userID, Name: "Test User", Password: "hash"}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&models.Customer{ID: customerID, Name: "Test", Email: &email, Password: "hash"}).Error; err != nil {
		t.Fatalf("create customer: %v", err)
	}
	shop := models.Shop{
		ID:        shopID,
		UserID:    userID,
		Name:      "Test Shop",
		Slug:      "test-shop-" + uuid.New().String()[:8],
		Latitude:  10.8231,
		Longitude: 106.6297,
		Status:    "active",
	}
	if err := db.Create(&shop).Error; err != nil {
		t.Fatalf("create shop: %v", err)
	}
	if err := db.Create(&models.ShippingConfig{BaseFee: 10000, PerKmRate: 3000, MaxDistanceKm: 30}).Error; err != nil {
		t.Fatalf("create shipping config: %v", err)
	}

	shopRepo := repositories.NewShopRepository(db)
	return customerID, shopID, shopRepo
}

// --- Discount order creation tests ---

func TestCreateOrderWithPercentDiscountOnProduct(t *testing.T) {
	db := newOrderServiceTestDB(t)
	customerID, shopID, shopRepo := setupOrderDiscountTest(t, db)

	product, _ := createDiscountTestProduct(t, db, shopID, 200000, "percent", 10)

	orderRepo := repositories.NewOrderRepository(db)
	paymentRepo := repositories.NewPaymentRepository(db)
	paymentSvc := NewPaymentService(paymentRepo)
	customerRepo := repositories.NewCustomerRepository(db)
	productRepo := repositories.NewProductRepository(db)
	shippingConfigRepo := repositories.NewShippingConfigRepository(db)
	shippingSvc := NewShippingService(shippingConfigRepo, shopRepo)
	orderSvc := NewOrderService(orderRepo, paymentSvc, customerRepo, productRepo, shippingSvc)

	input := CreateOrderInput{
		CustomerID:        customerID.String(),
		ShopID:            shopID.String(),
		ShippingAddress:   map[string]interface{}{"address": "123 Test St"},
		ShippingLatitude:  10.8231,
		ShippingLongitude: 106.6297,
		PaymentMethod:     "cod",
		Items: []CreateOrderItemInput{
			{ProductID: product.ID.String(), Quantity: 2},
		},
	}

	order, err := orderSvc.Create(input)
	if err != nil {
		t.Fatalf("Create order returned error: %v", err)
	}

	if len(order.Items) != 1 {
		t.Fatalf("order items = %d, want 1", len(order.Items))
	}

	item := order.Items[0]
	if item.OriginalPrice != 200000 {
		t.Fatalf("OriginalPrice = %v, want 200000", item.OriginalPrice)
	}
	if item.Price != 180000 {
		t.Fatalf("Price = %v, want 180000", item.Price)
	}
	if item.DiscountType != "percent" {
		t.Fatalf("DiscountType = %q, want %q", item.DiscountType, "percent")
	}
	if item.DiscountValue != 10 {
		t.Fatalf("DiscountValue = %v, want 10", item.DiscountValue)
	}
	if item.DiscountAmount != 20000 {
		t.Fatalf("DiscountAmount = %v, want 20000", item.DiscountAmount)
	}
	if item.Total != 360000 {
		t.Fatalf("Total = %v, want 360000", item.Total)
	}

	if order.SubTotal != 360000 {
		t.Fatalf("SubTotal = %v, want 360000", order.SubTotal)
	}
	if order.TotalAmount != 360000+10000 {
		t.Fatalf("TotalAmount = %v, want %v", order.TotalAmount, 360000+10000)
	}

	// Verify payment uses discounted amount
	var payment models.Payment
	if err := db.Where("order_id = ?", order.ID).First(&payment).Error; err != nil {
		t.Fatalf("find payment: %v", err)
	}
	if payment.Amount != order.TotalAmount {
		t.Fatalf("payment amount = %v, want %v", payment.Amount, order.TotalAmount)
	}
}

func TestCreateOrderWithFixedDiscountOnVariant(t *testing.T) {
	db := newOrderServiceTestDB(t)
	customerID, shopID, shopRepo := setupOrderDiscountTest(t, db)

	product, variant := createDiscountTestProduct(t, db, shopID, 200000, "fixed_amount", 50000)

	orderRepo := repositories.NewOrderRepository(db)
	paymentRepo := repositories.NewPaymentRepository(db)
	paymentSvc := NewPaymentService(paymentRepo)
	customerRepo := repositories.NewCustomerRepository(db)
	productRepo := repositories.NewProductRepository(db)
	shippingConfigRepo := repositories.NewShippingConfigRepository(db)
	shippingSvc := NewShippingService(shippingConfigRepo, shopRepo)
	orderSvc := NewOrderService(orderRepo, paymentSvc, customerRepo, productRepo, shippingSvc)

	input := CreateOrderInput{
		CustomerID:        customerID.String(),
		ShopID:            shopID.String(),
		ShippingAddress:   map[string]interface{}{"address": "123 Test St"},
		ShippingLatitude:  10.8231,
		ShippingLongitude: 106.6297,
		PaymentMethod:     "cod",
		Items: []CreateOrderItemInput{
			{ProductID: product.ID.String(), VariantID: variant.ID.String(), Quantity: 1},
		},
	}

	order, err := orderSvc.Create(input)
	if err != nil {
		t.Fatalf("Create order returned error: %v", err)
	}

	if len(order.Items) != 1 {
		t.Fatalf("order items = %d, want 1", len(order.Items))
	}

	item := order.Items[0]
	// Variant price = 200000 * 1.25 = 250000
	if item.OriginalPrice != 250000 {
		t.Fatalf("OriginalPrice = %v, want 250000", item.OriginalPrice)
	}
	if item.Price != 200000 {
		t.Fatalf("Price = %v, want 200000", item.Price)
	}
	if item.DiscountType != "fixed_amount" {
		t.Fatalf("DiscountType = %q, want %q", item.DiscountType, "fixed_amount")
	}
	if item.DiscountValue != 50000 {
		t.Fatalf("DiscountValue = %v, want 50000", item.DiscountValue)
	}
	if item.DiscountAmount != 50000 {
		t.Fatalf("DiscountAmount = %v, want 50000", item.DiscountAmount)
	}
	if item.Total != 200000 {
		t.Fatalf("Total = %v, want 200000", item.Total)
	}

	if order.SubTotal != 200000 {
		t.Fatalf("SubTotal = %v, want 200000", order.SubTotal)
	}
}

func TestCreateOrderClampsDiscountToZero(t *testing.T) {
	db := newOrderServiceTestDB(t)
	customerID, shopID, shopRepo := setupOrderDiscountTest(t, db)

	// Product with fixed amount discount larger than price
	product := models.Product{
		ID:            uuid.New(),
		ShopID:        shopID,
		Name:          "Zero Price Product",
		Slug:          "zero-price-" + uuid.New().String()[:8],
		Price:         50000,
		DiscountType:  "fixed_amount",
		DiscountValue: 100000,
		Status:        "active",
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}

	orderRepo := repositories.NewOrderRepository(db)
	paymentRepo := repositories.NewPaymentRepository(db)
	paymentSvc := NewPaymentService(paymentRepo)
	customerRepo := repositories.NewCustomerRepository(db)
	productRepo := repositories.NewProductRepository(db)
	shippingConfigRepo := repositories.NewShippingConfigRepository(db)
	shippingSvc := NewShippingService(shippingConfigRepo, shopRepo)
	orderSvc := NewOrderService(orderRepo, paymentSvc, customerRepo, productRepo, shippingSvc)

	input := CreateOrderInput{
		CustomerID:        customerID.String(),
		ShopID:            shopID.String(),
		ShippingAddress:   map[string]interface{}{"address": "123 Test St"},
		ShippingLatitude:  10.8231,
		ShippingLongitude: 106.6297,
		PaymentMethod:     "cod",
		Items: []CreateOrderItemInput{
			{ProductID: product.ID.String(), Quantity: 1},
		},
	}

	order, err := orderSvc.Create(input)
	if err != nil {
		t.Fatalf("Create order returned error: %v", err)
	}

	item := order.Items[0]
	if item.Price != 0 {
		t.Fatalf("Price = %v, want 0 (clamped)", item.Price)
	}
	if item.DiscountAmount != 50000 {
		t.Fatalf("DiscountAmount = %v, want 50000", item.DiscountAmount)
	}
	if item.OriginalPrice != 50000 {
		t.Fatalf("OriginalPrice = %v, want 50000", item.OriginalPrice)
	}
}

func TestCreateOrderNoDiscountHasConsistentAuditFields(t *testing.T) {
	db := newOrderServiceTestDB(t)
	customerID, shopID, shopRepo := setupOrderDiscountTest(t, db)

	product, _ := createDiscountTestProduct(t, db, shopID, 200000, "", 0)

	orderRepo := repositories.NewOrderRepository(db)
	paymentRepo := repositories.NewPaymentRepository(db)
	paymentSvc := NewPaymentService(paymentRepo)
	customerRepo := repositories.NewCustomerRepository(db)
	productRepo := repositories.NewProductRepository(db)
	shippingConfigRepo := repositories.NewShippingConfigRepository(db)
	shippingSvc := NewShippingService(shippingConfigRepo, shopRepo)
	orderSvc := NewOrderService(orderRepo, paymentSvc, customerRepo, productRepo, shippingSvc)

	input := CreateOrderInput{
		CustomerID:        customerID.String(),
		ShopID:            shopID.String(),
		ShippingAddress:   map[string]interface{}{"address": "123 Test St"},
		ShippingLatitude:  10.8231,
		ShippingLongitude: 106.6297,
		PaymentMethod:     "cod",
		Items: []CreateOrderItemInput{
			{ProductID: product.ID.String(), Quantity: 1},
		},
	}

	order, err := orderSvc.Create(input)
	if err != nil {
		t.Fatalf("Create order returned error: %v", err)
	}

	item := order.Items[0]
	if item.OriginalPrice != 200000 {
		t.Fatalf("OriginalPrice = %v, want 200000 (always set for new orders)", item.OriginalPrice)
	}
	if item.Price != 200000 {
		t.Fatalf("Price = %v, want 200000 (equal to original price when no discount)", item.Price)
	}
	if item.DiscountAmount != 0 {
		t.Fatalf("DiscountAmount = %v, want 0", item.DiscountAmount)
	}
	if item.DiscountType != "" {
		t.Fatalf("DiscountType = %q, want empty", item.DiscountType)
	}
	if item.Total != 200000 {
		t.Fatalf("Total = %v, want 200000", item.Total)
	}
}

func assertOrderStatus(t *testing.T, db *gorm.DB, orderID uuid.UUID, want string) {
	t.Helper()

	var order models.Order
	if err := db.First(&order, "id = ?", orderID).Error; err != nil {
		t.Fatalf("reload order %s: %v", orderID, err)
	}

	if order.Status != want {
		t.Fatalf("order %s status = %q, want %q", orderID, order.Status, want)
	}
}
