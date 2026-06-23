# Order Auto-Completion Cron Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Tự động chuyển đơn hàng từ `delivered` sang `completed` sau 7 ngày nếu đơn không bị khiếu nại.

**Architecture:** Thêm dữ liệu trạng thái vào `Order`, xử lý nghiệp vụ trong `OrderService`, query/update trong `OrderRepository`, và setup robfig/cron ở startup. Cron chỉ gọi service; service dùng transaction riêng cho từng đơn và ghi `OrderStatusHistory`.

**Tech Stack:** Go 1.26, Go Fiber, GORM, PostgreSQL, robfig/cron v3, SQLite in-memory cho unit/integration tests nhẹ.

---

## File Map

- Modify: `go.mod` and `go.sum`
  - Add `github.com/robfig/cron/v3` for cron scheduling.
  - Add `gorm.io/driver/sqlite` for in-memory tests.
- Modify: `internal/models/order.go`
  - Fix duplicated `OrderStatusDelivered` constant.
  - Add `OrderStatusCompleted`.
  - Add `DeliveredAt` and `HasComplaint` fields to `Order`.
- Create: `internal/models/order_test.go`
  - Verify new constants and new fields are usable.
- Modify: `internal/services/order_service.go`
  - Set `delivered_at` when status changes to `delivered`.
  - Add `AutoCompleteDeliveredOrders()` and `AutoCompleteDeliveredOrdersBefore(cutoff time.Time)`.
- Create: `internal/services/order_service_test.go`
  - Test `delivered_at` behavior and auto-completion business rules.
- Modify: `internal/repositories/order_repo.go`
  - Add query method for eligible delivered orders.
  - Add conditional completion update method for one order inside a transaction.
- Create: `cmd/server/cron.go`
  - Encapsulate robfig/cron setup for order auto-completion.
- Create: `cmd/server/cron_test.go`
  - Verify cron expression represents daily `02:00` and one job is registered.
- Modify: `cmd/server/main.go`
  - Start the order completion cron after `OrderService` is created.

## Notes Before Starting

- The worktree currently has an uncommitted edit in `internal/models/order.go` with duplicate constant names:

```go
OrderStatusDelivered = "delivered"
OrderStatusDelivered = "completed"
```

- Treat that as part of this feature and fix it to:

```go
OrderStatusDelivered = "delivered"
OrderStatusCompleted = "completed"
```

- robfig/cron v3 `Start()` does not return an error. Startup failure for this feature means `AddFunc` returns an error while registering the fixed expression `0 2 * * *`.

---

### Task 1: Add Order Model Fields And Constants

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`
- Modify: `internal/models/order.go`
- Create: `internal/models/order_test.go`

- [ ] **Step 1: Add test dependencies and cron dependency**

Run:

```bash
go get github.com/robfig/cron/v3 gorm.io/driver/sqlite
```

Expected: `go.mod` includes `github.com/robfig/cron/v3` and `gorm.io/driver/sqlite`; `go.sum` is updated.

- [ ] **Step 2: Write the failing model test**

Create `internal/models/order_test.go`:

```go
package models

import (
	"testing"
	"time"
)

func TestOrderStatusCompletedConstant(t *testing.T) {
	if OrderStatusDelivered != "delivered" {
		t.Fatalf("OrderStatusDelivered = %q, want delivered", OrderStatusDelivered)
	}

	if OrderStatusCompleted != "completed" {
		t.Fatalf("OrderStatusCompleted = %q, want completed", OrderStatusCompleted)
	}
}

func TestOrderCompletionFields(t *testing.T) {
	deliveredAt := time.Now().UTC()
	order := Order{
		DeliveredAt:  &deliveredAt,
		HasComplaint: true,
	}

	if order.DeliveredAt == nil {
		t.Fatal("DeliveredAt is nil, want timestamp pointer")
	}

	if !order.DeliveredAt.Equal(deliveredAt) {
		t.Fatalf("DeliveredAt = %v, want %v", order.DeliveredAt, deliveredAt)
	}

	if !order.HasComplaint {
		t.Fatal("HasComplaint = false, want true")
	}
}
```

- [ ] **Step 3: Run the model test to verify it fails**

Run:

```bash
go test ./internal/models -run 'TestOrderStatusCompletedConstant|TestOrderCompletionFields' -v
```

Expected: FAIL because `OrderStatusCompleted`, `DeliveredAt`, and `HasComplaint` do not exist yet, or because `OrderStatusDelivered` is duplicated.

- [ ] **Step 4: Update `internal/models/order.go` minimally**

Change the constants near the top of `internal/models/order.go` to:

```go
const (
	OrderStatusPending   = "pending"
	OrderStatusConfirmed = "confirmed"
	OrderStatusShipping  = "shipping"
	OrderStatusDelivered = "delivered"
	OrderStatusCompleted = "completed"
	OrderStatusCancelled = "cancelled"
)
```

Add the new fields to `Order` after `Status`:

```go
	Status          string     `gorm:"type:varchar(20);default:pending" json:"status"`
	DeliveredAt     *time.Time `gorm:"type:timestamptz" json:"delivered_at"`
	HasComplaint    bool       `gorm:"default:false" json:"has_complaint"`
```

- [ ] **Step 5: Format and run the model test**

Run:

```bash
gofmt -w internal/models/order.go internal/models/order_test.go
go test ./internal/models -run 'TestOrderStatusCompletedConstant|TestOrderCompletionFields' -v
```

Expected: PASS.

- [ ] **Step 6: Run package compile check**

Run:

```bash
go test ./internal/models ./internal/database -v
```

Expected: PASS. This confirms GORM model definitions compile and `AutoMigrate` still references valid models.

- [ ] **Step 7: Commit Task 1**

Run:

```bash
git add go.mod go.sum internal/models/order.go internal/models/order_test.go
git commit -m "feat: add order completion fields"
```

Expected: commit succeeds.

---

### Task 2: Set DeliveredAt When Order Becomes Delivered

**Files:**
- Modify: `internal/services/order_service.go`
- Create or modify: `internal/services/order_service_test.go`

- [ ] **Step 1: Write the failing service test**

Create `internal/services/order_service_test.go` with this initial content:

```go
package services

import (
	"testing"
	"time"

	"go-fiber/internal/models"
	"go-fiber/internal/repositories"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newOrderServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.AutoMigrate(&models.Order{}, &models.OrderStatusHistory{}, &models.Payment{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
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
```

- [ ] **Step 2: Run the service test to verify it fails**

Run:

```bash
go test ./internal/services -run TestUpdateStatusToDeliveredSetsDeliveredAt -v
```

Expected: FAIL because `UpdateStatus` updates only `status` and does not set `delivered_at`.

- [ ] **Step 3: Update `UpdateStatus` to set `delivered_at`**

In `internal/services/order_service.go`, add `time` to imports:

```go
import (
	"errors"
	"fmt"
	"time"

	"go-fiber/internal/models"
	"go-fiber/internal/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)
```

Replace the status update block inside `UpdateStatus` transaction with:

```go
		updates := map[string]interface{}{
			"status": input.Status,
		}
		if input.Status == models.OrderStatusDelivered {
			updates["delivered_at"] = time.Now()
		}

		if err := tx.Model(&models.Order{}).Where("id = ?", order.ID).Updates(updates).Error; err != nil {
			return err
		}
```

Keep the existing status history creation and payment handling after this block. Do not add `completed` to `validTransitions`.

- [ ] **Step 4: Replace status string literals in touched code with constants**

In `UpdateStatus`, change the transition map to:

```go
	validTransitions := map[string][]string{
		models.OrderStatusPending:   {models.OrderStatusConfirmed, models.OrderStatusCancelled},
		models.OrderStatusConfirmed: {models.OrderStatusShipping, models.OrderStatusCancelled},
		models.OrderStatusShipping:  {models.OrderStatusDelivered},
	}
```

Change the delivered payment block to:

```go
		if input.Status == models.OrderStatusDelivered {
			if err := s.paymentSvc.MarkAsPaid(tx, order.ID); err != nil {
				return err
			}
		}
```

- [ ] **Step 5: Format and run the service test**

Run:

```bash
gofmt -w internal/services/order_service.go internal/services/order_service_test.go
go test ./internal/services -run TestUpdateStatusToDeliveredSetsDeliveredAt -v
```

Expected: PASS.

- [ ] **Step 6: Run related package tests**

Run:

```bash
go test ./internal/models ./internal/repositories ./internal/services -v
```

Expected: PASS.

- [ ] **Step 7: Commit Task 2**

Run:

```bash
git add internal/services/order_service.go internal/services/order_service_test.go
git commit -m "feat: set delivered timestamp on order delivery"
```

Expected: commit succeeds.

---

### Task 3: Add Repository Methods For Auto-Completion

**Files:**
- Modify: `internal/repositories/order_repo.go`
- Create: `internal/repositories/order_repo_test.go`

- [ ] **Step 1: Write failing repository tests**

Create `internal/repositories/order_repo_test.go`:

```go
package repositories

import (
	"testing"
	"time"

	"go-fiber/internal/models"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newOrderRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.AutoMigrate(&models.Order{}, &models.OrderStatusHistory{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
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
```

- [ ] **Step 2: Run repository tests to verify they fail**

Run:

```bash
go test ./internal/repositories -run 'TestFindAutoCompletableDelivered|TestCompleteDeliveredOrderOnlyUpdatesDeliveredRows' -v
```

Expected: FAIL because `FindAutoCompletableDelivered` and `CompleteDeliveredOrder` do not exist.

- [ ] **Step 3: Implement repository methods**

Add these methods to `internal/repositories/order_repo.go` after `FindByShopID`:

```go
func (r *OrderRepository) FindAutoCompletableDelivered(cutoff time.Time) ([]models.Order, error) {
	var orders []models.Order
	err := r.db.
		Where("status = ?", models.OrderStatusDelivered).
		Where("delivered_at IS NOT NULL AND delivered_at <= ?", cutoff).
		Where("has_complaint = ?", false).
		Order("delivered_at ASC").
		Find(&orders).Error
	return orders, err
}

func (r *OrderRepository) CompleteDeliveredOrder(tx *gorm.DB, orderID uuid.UUID) (int64, error) {
	result := tx.Model(&models.Order{}).
		Where("id = ? AND status = ?", orderID, models.OrderStatusDelivered).
		Update("status", models.OrderStatusCompleted)
	return result.RowsAffected, result.Error
}
```

The file already imports `time`, `uuid`, `models`, and `gorm`, so no new import is needed.

- [ ] **Step 4: Format and run repository tests**

Run:

```bash
gofmt -w internal/repositories/order_repo.go internal/repositories/order_repo_test.go
go test ./internal/repositories -run 'TestFindAutoCompletableDelivered|TestCompleteDeliveredOrderOnlyUpdatesDeliveredRows' -v
```

Expected: PASS.

- [ ] **Step 5: Run related package tests**

Run:

```bash
go test ./internal/repositories ./internal/services -v
```

Expected: PASS.

- [ ] **Step 6: Commit Task 3**

Run:

```bash
git add internal/repositories/order_repo.go internal/repositories/order_repo_test.go
git commit -m "feat: add order auto-completion repository queries"
```

Expected: commit succeeds.

---

### Task 4: Add OrderService Auto-Completion Logic

**Files:**
- Modify: `internal/services/order_service.go`
- Modify: `internal/services/order_service_test.go`

- [ ] **Step 1: Add failing auto-completion tests**

Append these tests to `internal/services/order_service_test.go`:

```go
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
```

- [ ] **Step 2: Run auto-completion tests to verify they fail**

Run:

```bash
go test ./internal/services -run 'TestAutoCompleteDeliveredOrdersBefore' -v
```

Expected: FAIL because `AutoCompleteDeliveredOrdersBefore` does not exist.

- [ ] **Step 3: Implement service methods**

Add these methods to `internal/services/order_service.go` after `UpdateStatus` and before `Cancel`:

```go
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
```

- [ ] **Step 4: Format and run service auto-completion tests**

Run:

```bash
gofmt -w internal/services/order_service.go internal/services/order_service_test.go
go test ./internal/services -run 'TestAutoCompleteDeliveredOrdersBefore|TestUpdateStatusToDeliveredSetsDeliveredAt' -v
```

Expected: PASS.

- [ ] **Step 5: Run related package tests**

Run:

```bash
go test ./internal/repositories ./internal/services -v
```

Expected: PASS.

- [ ] **Step 6: Commit Task 4**

Run:

```bash
git add internal/services/order_service.go internal/services/order_service_test.go
git commit -m "feat: auto-complete delivered orders"
```

Expected: commit succeeds.

---

### Task 5: Add Cron Setup And Wire It Into Server Startup

**Files:**
- Create: `cmd/server/cron.go`
- Create: `cmd/server/cron_test.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Write failing cron tests**

Create `cmd/server/cron_test.go`:

```go
package main

import (
	"testing"
	"time"

	"github.com/robfig/cron/v3"
)

func TestOrderCompletionCronSpecRunsAtTwoAM(t *testing.T) {
	schedule, err := cron.ParseStandard(orderCompletionCronSpec)
	if err != nil {
		t.Fatalf("ParseStandard returned error: %v", err)
	}

	next := schedule.Next(time.Date(2026, 6, 23, 0, 0, 0, 0, time.Local))
	if next.Hour() != 2 || next.Minute() != 0 {
		t.Fatalf("next run = %v, want 02:00", next)
	}
}

func TestStartOrderCompletionCronRegistersOneJob(t *testing.T) {
	cronRunner, err := startOrderCompletionCron(nil)
	if err != nil {
		t.Fatalf("startOrderCompletionCron returned error: %v", err)
	}
	defer cronRunner.Stop()

	entries := cronRunner.Entries()
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
}
```

- [ ] **Step 2: Run cron tests to verify they fail**

Run:

```bash
go test ./cmd/server -run 'TestOrderCompletionCron' -v
```

Expected: FAIL because `orderCompletionCronSpec` and `startOrderCompletionCron` do not exist.

- [ ] **Step 3: Implement cron setup helper**

Create `cmd/server/cron.go`:

```go
package main

import (
	"log"

	"go-fiber/internal/services"

	"github.com/robfig/cron/v3"
)

const orderCompletionCronSpec = "0 2 * * *"

func startOrderCompletionCron(orderService *services.OrderService) (*cron.Cron, error) {
	cronRunner := cron.New()

	_, err := cronRunner.AddFunc(orderCompletionCronSpec, func() {
		if orderService == nil {
			log.Printf("order completion cron skipped: order service is nil")
			return
		}

		completedCount, err := orderService.AutoCompleteDeliveredOrders()
		if err != nil {
			log.Printf("order completion cron failed: %v", err)
			return
		}

		log.Printf("order completion cron completed %d orders", completedCount)
	})
	if err != nil {
		return nil, err
	}

	cronRunner.Start()
	return cronRunner, nil
}
```

- [ ] **Step 4: Wire cron into `main.go`**

In `cmd/server/main.go`, after this line:

```go
	orderService := services.NewOrderService(orderRepo, paymentSvc, customerRepo, productRepo, shippingSvc)
```

Add:

```go
	orderCompletionCron, err := startOrderCompletionCron(orderService)
	if err != nil {
		log.Fatalf("Failed to start order completion cron: %v", err)
	}
	defer orderCompletionCron.Stop()
```

This makes `AddFunc` registration failure fatal. `cron.Start()` itself does not return an error in robfig/cron v3.

- [ ] **Step 5: Format and run cron tests**

Run:

```bash
gofmt -w cmd/server/cron.go cmd/server/cron_test.go cmd/server/main.go
go test ./cmd/server -run 'TestOrderCompletionCron' -v
```

Expected: PASS.

- [ ] **Step 6: Run all tests**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 7: Commit Task 5**

Run:

```bash
git add cmd/server/cron.go cmd/server/cron_test.go cmd/server/main.go
git commit -m "feat: schedule order auto-completion cron"
```

Expected: commit succeeds.

---

### Task 6: Final Verification And Documentation Check

**Files:**
- Modify only if verification reveals a real issue.

- [ ] **Step 1: Run formatting across touched Go files**

Run:

```bash
gofmt -w internal/models/order.go internal/models/order_test.go internal/repositories/order_repo.go internal/repositories/order_repo_test.go internal/services/order_service.go internal/services/order_service_test.go cmd/server/cron.go cmd/server/cron_test.go cmd/server/main.go
```

Expected: command exits successfully.

- [ ] **Step 2: Run full test suite**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 3: Run build check**

Run:

```bash
go build ./...
```

Expected: PASS.

- [ ] **Step 4: Inspect git diff**

Run:

```bash
git diff --stat
git diff -- go.mod go.sum internal/models/order.go internal/repositories/order_repo.go internal/services/order_service.go cmd/server/cron.go cmd/server/main.go
```

Expected: diff only contains order completion cron changes described in this plan.

- [ ] **Step 5: Commit final fixes if any**

If Step 1-4 required code changes, run:

```bash
git add go.mod go.sum internal/models/order.go internal/models/order_test.go internal/repositories/order_repo.go internal/repositories/order_repo_test.go internal/services/order_service.go internal/services/order_service_test.go cmd/server/cron.go cmd/server/cron_test.go cmd/server/main.go
git commit -m "test: verify order auto-completion cron"
```

Expected: commit succeeds only if there were verification fixes. If there were no changes, skip this commit.

---

## Completion Criteria

- `github.com/robfig/cron/v3` is installed and used for the order completion job.
- Cron schedule is fixed at `0 2 * * *`.
- `Order` has `DeliveredAt` and `HasComplaint` fields.
- `OrderStatusCompleted` exists and the duplicate `OrderStatusDelivered` constant is fixed.
- `UpdateStatus` sets `delivered_at` when moving to `delivered`.
- Cron-completion only processes `delivered` orders older than 7 days with `has_complaint=false`.
- Auto-completion writes `OrderStatusHistory`.
- `completed` is not added to manual `UpdateStatus` transitions.
- `go test ./...` passes.
- `go build ./...` passes.
