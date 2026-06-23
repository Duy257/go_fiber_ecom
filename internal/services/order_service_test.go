package services

import (
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

	createTableSQL := []string{
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
			"price" real NOT NULL,
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
