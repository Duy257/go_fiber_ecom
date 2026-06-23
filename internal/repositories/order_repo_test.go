package repositories

import (
	"testing"
	"time"

	"go-fiber/internal/models"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func newOrderRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := uuid.NewString() + ".db"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	createTableSQL := []string{
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
		`CREATE TABLE "order_status_histories" (
			"id" text PRIMARY KEY,
			"order_id" text NOT NULL,
			"status" text NOT NULL,
			"note" text,
			"created_at" timestamp
		)`,
	}
	for _, sql := range createTableSQL {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatalf("create table: %v", err)
		}
	}

	return db
}

func createRepoTestOrder(t *testing.T, db *gorm.DB, status string, deliveredAt *time.Time, hasComplaint bool) models.Order {
	t.Helper()

	order := models.Order{
		ID:              uuid.New(),
		CustomerID:      uuid.New(),
		ShopID:          uuid.New(),
		OrderNumber:     "ORD-REPO-" + uuid.New().String()[:8],
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

func TestFindAutoCompletableDelivered(t *testing.T) {
	db := newOrderRepoTestDB(t)
	repo := NewOrderRepository(db)
	now := time.Now().UTC()
	oldDeliveredAt := now.Add(-8 * 24 * time.Hour)
	recentDeliveredAt := now.Add(-6 * 24 * time.Hour)

	eligible := createRepoTestOrder(t, db, models.OrderStatusDelivered, &oldDeliveredAt, false)
	createRepoTestOrder(t, db, models.OrderStatusDelivered, &recentDeliveredAt, false)
	createRepoTestOrder(t, db, models.OrderStatusDelivered, &oldDeliveredAt, true)
	createRepoTestOrder(t, db, models.OrderStatusCompleted, &oldDeliveredAt, false)
	createRepoTestOrder(t, db, models.OrderStatusShipping, nil, false)

	orders, err := repo.FindAutoCompletableDelivered(now.Add(-7 * 24 * time.Hour))
	if err != nil {
		t.Fatalf("FindAutoCompletableDelivered returned error: %v", err)
	}

	if len(orders) != 1 {
		t.Fatalf("len(orders) = %d, want 1", len(orders))
	}

	if orders[0].ID != eligible.ID {
		t.Fatalf("eligible order ID = %s, want %s", orders[0].ID, eligible.ID)
	}
}

func TestCompleteDeliveredOrderOnlyUpdatesDeliveredRows(t *testing.T) {
	db := newOrderRepoTestDB(t)
	repo := NewOrderRepository(db)
	now := time.Now().UTC()

	delivered := createRepoTestOrder(t, db, models.OrderStatusDelivered, &now, false)
	completed := createRepoTestOrder(t, db, models.OrderStatusCompleted, &now, false)

	var deliveredRows int64
	if err := repo.Transaction(func(tx *gorm.DB) error {
		rows, err := repo.CompleteDeliveredOrder(tx, delivered.ID)
		deliveredRows = rows
		return err
	}); err != nil {
		t.Fatalf("CompleteDeliveredOrder delivered: %v", err)
	}

	if deliveredRows != 1 {
		t.Fatalf("delivered rows affected = %d, want 1", deliveredRows)
	}

	var reloadedDelivered models.Order
	if err := db.First(&reloadedDelivered, "id = ?", delivered.ID).Error; err != nil {
		t.Fatalf("reload delivered order: %v", err)
	}
	if reloadedDelivered.Status != models.OrderStatusCompleted {
		t.Fatalf("delivered status = %q, want %q", reloadedDelivered.Status, models.OrderStatusCompleted)
	}

	var completedRows int64
	if err := repo.Transaction(func(tx *gorm.DB) error {
		rows, err := repo.CompleteDeliveredOrder(tx, completed.ID)
		completedRows = rows
		return err
	}); err != nil {
		t.Fatalf("CompleteDeliveredOrder completed: %v", err)
	}

	if completedRows != 0 {
		t.Fatalf("completed rows affected = %d, want 0", completedRows)
	}
}
