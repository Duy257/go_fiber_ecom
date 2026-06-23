# Shop Wallet Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add shop wallets, immutable wallet logs, withdrawal requests, order-completion wallet credit, 7-day pending release, and shop/admin APIs.

**Architecture:** Use `shop_wallets` as the current balance table, `shop_wallet_logs` as append-only audit history, and `shop_withdrawal_requests` as the withdrawal workflow table. Keep all balance mutations in `ShopWalletService` and run them inside database transactions with repository-level row locking for PostgreSQL. Integrate order completion by injecting `ShopWalletService` into `OrderService`, then add a cron job that releases pending funds 7 days after `orders.completed_at`.

**Tech Stack:** Go 1.26, Fiber v2, GORM, PostgreSQL, SQLite in-memory tests through `glebarez/sqlite`, robfig/cron, UUID primary keys.

---

## File Map

- Create `internal/models/shop_wallet.go`: wallet, log, withdrawal request models and constants.
- Modify `internal/models/order.go`: add `CompletedAt *time.Time`.
- Modify `internal/models/order_test.go`: cover `CompletedAt`.
- Create `internal/repositories/shop_wallet_repo.go`: wallet creation, row locking, log queries, withdrawal queries, transaction helper.
- Create `internal/repositories/shop_wallet_repo_test.go`: repository coverage using SQLite.
- Modify `internal/repositories/order_repo.go`: set `completed_at` during completion and query release-eligible completed orders.
- Modify `internal/repositories/order_repo_test.go`: completed-at and release-eligible order coverage.
- Create `internal/services/shop_wallet_service.go`: business logic for credit, release, withdrawal create, approve, reject, and shop-owner reads.
- Create `internal/services/shop_wallet_service_test.go`: service-level wallet lifecycle tests.
- Modify `internal/services/order_service.go`: inject `ShopWalletService` and credit pending balance when orders become `completed`.
- Modify `internal/services/order_service_test.go`: order completion wallet credit coverage and constructor updates.
- Modify `internal/cron/cron.go`: register order completion and wallet release jobs at 02:00 daily.
- Modify `internal/cron/cron_test.go`: verify both cron specs and registered job count.
- Create `internal/handlers/shop_wallet_handler.go`: shop owner and admin HTTP handlers.
- Modify `cmd/server/main.go`: wire repo/service/handler/routes and seed wallet permissions idempotently.
- Modify `internal/database/database.go`: AutoMigrate new models and add `completed_at` backfill SQL.

## Implementation Notes

- Use `float64` for money fields to stay consistent with existing `Order`, `OrderItem`, and `Payment` models, while retaining `gorm:"type:decimal(12,2)"` tags.
- Use `map[string]interface{}` with `gorm:"type:jsonb;serializer:json"` for `bank_info` and `metadata`.
- Use PostgreSQL row locking only when `tx.Dialector.Name() != "sqlite"`, because SQLite tests do not support `FOR UPDATE` syntax.
- Keep wallet log writes append-only. Do not add update or delete methods for logs.
- Preserve existing behavior when `OrderService` is constructed with a nil wallet service in older tests; wallet credit should no-op only when the service is nil.

---

### Task 1: Models And Migration

**Files:**
- Create: `internal/models/shop_wallet.go`
- Modify: `internal/models/order.go`
- Modify: `internal/models/order_test.go`
- Modify: `internal/database/database.go`

- [ ] **Step 1: Write failing model tests**

Create `internal/models/shop_wallet_test.go`:

```go
package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestShopWalletLogConstants(t *testing.T) {
	if ShopWalletLogTypeOrderCompletedPending != "order_completed_pending" {
		t.Fatalf("ShopWalletLogTypeOrderCompletedPending = %q", ShopWalletLogTypeOrderCompletedPending)
	}
	if ShopWalletLogTypePendingReleased != "pending_released" {
		t.Fatalf("ShopWalletLogTypePendingReleased = %q", ShopWalletLogTypePendingReleased)
	}
	if ShopWalletLogTypeWithdrawalHold != "withdrawal_hold" {
		t.Fatalf("ShopWalletLogTypeWithdrawalHold = %q", ShopWalletLogTypeWithdrawalHold)
	}
	if ShopWalletLogTypeWithdrawalApproved != "withdrawal_approved" {
		t.Fatalf("ShopWalletLogTypeWithdrawalApproved = %q", ShopWalletLogTypeWithdrawalApproved)
	}
	if ShopWalletLogTypeWithdrawalRejected != "withdrawal_rejected" {
		t.Fatalf("ShopWalletLogTypeWithdrawalRejected = %q", ShopWalletLogTypeWithdrawalRejected)
	}
}

func TestShopWithdrawalStatusConstants(t *testing.T) {
	if ShopWithdrawalStatusPending != "pending" {
		t.Fatalf("ShopWithdrawalStatusPending = %q", ShopWithdrawalStatusPending)
	}
	if ShopWithdrawalStatusApproved != "approved" {
		t.Fatalf("ShopWithdrawalStatusApproved = %q", ShopWithdrawalStatusApproved)
	}
	if ShopWithdrawalStatusRejected != "rejected" {
		t.Fatalf("ShopWithdrawalStatusRejected = %q", ShopWithdrawalStatusRejected)
	}
}

func TestShopWalletFields(t *testing.T) {
	shopID := uuid.New()
	wallet := ShopWallet{
		ID:               uuid.New(),
		ShopID:           shopID,
		PendingBalance:   100000,
		AvailableBalance: 50000,
		WithdrawnBalance: 25000,
	}

	if wallet.ShopID != shopID {
		t.Fatalf("ShopID = %s, want %s", wallet.ShopID, shopID)
	}
	if wallet.PendingBalance != 100000 {
		t.Fatalf("PendingBalance = %v, want 100000", wallet.PendingBalance)
	}
	if wallet.AvailableBalance != 50000 {
		t.Fatalf("AvailableBalance = %v, want 50000", wallet.AvailableBalance)
	}
	if wallet.WithdrawnBalance != 25000 {
		t.Fatalf("WithdrawnBalance = %v, want 25000", wallet.WithdrawnBalance)
	}
}
```

Extend `internal/models/order_test.go` with:

```go
func TestOrderCompletedAtField(t *testing.T) {
	completedAt := time.Now().UTC()
	order := Order{CompletedAt: &completedAt}

	if order.CompletedAt == nil {
		t.Fatal("CompletedAt is nil, want timestamp pointer")
	}
	if !order.CompletedAt.Equal(completedAt) {
		t.Fatalf("CompletedAt = %v, want %v", order.CompletedAt, completedAt)
	}
}
```

- [ ] **Step 2: Run model tests to verify they fail**

Run: `go test ./internal/models/...`

Expected: FAIL with undefined symbols such as `ShopWallet`, `ShopWalletLogTypeOrderCompletedPending`, and `Order.CompletedAt`.

- [ ] **Step 3: Add wallet models and order completion timestamp**

Create `internal/models/shop_wallet.go`:

```go
package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	ShopWalletLogTypeOrderCompletedPending = "order_completed_pending"
	ShopWalletLogTypePendingReleased        = "pending_released"
	ShopWalletLogTypeWithdrawalHold         = "withdrawal_hold"
	ShopWalletLogTypeWithdrawalApproved     = "withdrawal_approved"
	ShopWalletLogTypeWithdrawalRejected     = "withdrawal_rejected"
)

const (
	ShopWithdrawalStatusPending  = "pending"
	ShopWithdrawalStatusApproved = "approved"
	ShopWithdrawalStatusRejected = "rejected"
)

type ShopWallet struct {
	ID               uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ShopID           uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"shop_id"`
	Shop             Shop      `gorm:"foreignKey:ShopID" json:"shop,omitempty"`
	PendingBalance   float64   `gorm:"type:decimal(12,2);not null;default:0" json:"pending_balance"`
	AvailableBalance float64   `gorm:"type:decimal(12,2);not null;default:0" json:"available_balance"`
	WithdrawnBalance float64   `gorm:"type:decimal(12,2);not null;default:0" json:"withdrawn_balance"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type ShopWalletLog struct {
	ID                  uuid.UUID              `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	WalletID            uuid.UUID              `gorm:"type:uuid;index;not null" json:"wallet_id"`
	Wallet              ShopWallet             `gorm:"foreignKey:WalletID" json:"-"`
	ShopID              uuid.UUID              `gorm:"type:uuid;index;not null" json:"shop_id"`
	Shop                Shop                    `gorm:"foreignKey:ShopID" json:"shop,omitempty"`
	OrderID             *uuid.UUID             `gorm:"type:uuid;index" json:"order_id,omitempty"`
	Order               *Order                  `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	WithdrawalRequestID *uuid.UUID             `gorm:"type:uuid;index" json:"withdrawal_request_id,omitempty"`
	WithdrawalRequest   *ShopWithdrawalRequest `gorm:"foreignKey:WithdrawalRequestID" json:"withdrawal_request,omitempty"`
	Type                string                 `gorm:"type:varchar(50);index;not null" json:"type"`
	Amount              float64                `gorm:"type:decimal(12,2);not null" json:"amount"`
	AvailableBefore     float64                `gorm:"type:decimal(12,2);not null" json:"available_before"`
	AvailableAfter      float64                `gorm:"type:decimal(12,2);not null" json:"available_after"`
	PendingBefore       float64                `gorm:"type:decimal(12,2);not null" json:"pending_before"`
	PendingAfter        float64                `gorm:"type:decimal(12,2);not null" json:"pending_after"`
	WithdrawnBefore     float64                `gorm:"type:decimal(12,2);not null" json:"withdrawn_before"`
	WithdrawnAfter      float64                `gorm:"type:decimal(12,2);not null" json:"withdrawn_after"`
	Status              string                 `gorm:"type:varchar(20);not null" json:"status"`
	Description         string                 `gorm:"type:text" json:"description,omitempty"`
	Metadata            map[string]interface{} `gorm:"type:jsonb;serializer:json" json:"metadata,omitempty"`
	CreatedAt           time.Time              `json:"created_at"`
}

type ShopWithdrawalRequest struct {
	ID         uuid.UUID              `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ShopID     uuid.UUID              `gorm:"type:uuid;index;not null" json:"shop_id"`
	Shop       Shop                   `gorm:"foreignKey:ShopID" json:"shop,omitempty"`
	Amount     float64                `gorm:"type:decimal(12,2);not null" json:"amount"`
	Status     string                 `gorm:"type:varchar(20);index;not null" json:"status"`
	BankInfo   map[string]interface{} `gorm:"type:jsonb;serializer:json;not null" json:"bank_info"`
	Note       string                 `gorm:"type:text" json:"note,omitempty"`
	AdminNote  string                 `gorm:"type:text" json:"admin_note,omitempty"`
	ReviewedBy *uuid.UUID             `gorm:"type:uuid;index" json:"reviewed_by,omitempty"`
	ReviewedAt *time.Time             `gorm:"type:timestamptz" json:"reviewed_at,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}
```

Modify `internal/models/order.go` by adding `CompletedAt` after `DeliveredAt`:

```go
	DeliveredAt        *time.Time             `gorm:"type:timestamptz" json:"delivered_at"`
	CompletedAt        *time.Time             `gorm:"type:timestamptz" json:"completed_at"`
```

- [ ] **Step 4: Add migration wiring and completed-at backfill**

Modify `internal/database/database.go` so `Migrate` includes the backfill after `AutoMigrate`:

```go
	err := db.AutoMigrate(
		&models.Role{},
		&models.Permission{},
		&models.User{},
		&models.Customer{},
		&models.Category{},
		&models.ProductCategory{},
		&models.Shop{},
		&models.ShopWallet{},
		&models.Product{},
		&models.ProductVariant{},
		&models.ProductImage{},
		&models.Order{},
		&models.OrderItem{},
		&models.OrderStatusHistory{},
		&models.Payment{},
		&models.ShopWithdrawalRequest{},
		&models.ShopWalletLog{},
		&models.ShippingConfig{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	db.Exec(`UPDATE orders SET completed_at = updated_at WHERE status = 'completed' AND completed_at IS NULL`)
```

- [ ] **Step 5: Format and run model tests**

Run: `gofmt -w internal/models/shop_wallet.go internal/models/order.go internal/models/order_test.go internal/database/database.go`

Run: `go test ./internal/models/...`

Expected: PASS.

- [ ] **Step 6: Commit models and migration**

Run:

```bash
git add internal/models/shop_wallet.go internal/models/order.go internal/models/order_test.go internal/database/database.go
git commit -m "feat: add shop wallet models"
```

---

### Task 2: Wallet Repository

**Files:**
- Create: `internal/repositories/shop_wallet_repo.go`
- Create: `internal/repositories/shop_wallet_repo_test.go`
- Modify: `internal/repositories/order_repo.go`
- Modify: `internal/repositories/order_repo_test.go`

- [ ] **Step 1: Write failing wallet repository tests**

Create `internal/repositories/shop_wallet_repo_test.go`:

```go
package repositories

import (
	"os"
	"testing"
	"time"

	"go-fiber/internal/models"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func newShopWalletRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := uuid.NewString() + ".db"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
		os.Remove(dsn)
	})
	if err := db.AutoMigrate(&models.ShopWallet{}, &models.ShopWalletLog{}, &models.ShopWithdrawalRequest{}, &models.Order{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return db
}

func TestFindOrCreateWalletByShopIDCreatesZeroBalanceWallet(t *testing.T) {
	db := newShopWalletRepoTestDB(t)
	repo := NewShopWalletRepository(db)
	shopID := uuid.New()

	wallet, err := repo.FindOrCreateByShopID(db, shopID)
	if err != nil {
		t.Fatalf("FindOrCreateByShopID returned error: %v", err)
	}
	if wallet.ShopID != shopID {
		t.Fatalf("ShopID = %s, want %s", wallet.ShopID, shopID)
	}
	if wallet.PendingBalance != 0 || wallet.AvailableBalance != 0 || wallet.WithdrawnBalance != 0 {
		t.Fatalf("wallet balances = pending %v available %v withdrawn %v, want zero", wallet.PendingBalance, wallet.AvailableBalance, wallet.WithdrawnBalance)
	}

	second, err := repo.FindOrCreateByShopID(db, shopID)
	if err != nil {
		t.Fatalf("FindOrCreateByShopID second call returned error: %v", err)
	}
	if second.ID != wallet.ID {
		t.Fatalf("second wallet ID = %s, want %s", second.ID, wallet.ID)
	}
}

func TestFindWalletLogsFiltersByType(t *testing.T) {
	db := newShopWalletRepoTestDB(t)
	repo := NewShopWalletRepository(db)
	shopID := uuid.New()
	wallet, err := repo.FindOrCreateByShopID(db, shopID)
	if err != nil {
		t.Fatalf("create wallet: %v", err)
	}
	logs := []models.ShopWalletLog{
		{ID: uuid.New(), WalletID: wallet.ID, ShopID: shopID, Type: models.ShopWalletLogTypeOrderCompletedPending, Amount: 100, Status: models.OrderStatusCompleted},
		{ID: uuid.New(), WalletID: wallet.ID, ShopID: shopID, Type: models.ShopWalletLogTypePendingReleased, Amount: 100, Status: models.OrderStatusCompleted},
	}
	if err := db.Create(&logs).Error; err != nil {
		t.Fatalf("create logs: %v", err)
	}

	got, total, err := repo.FindLogs(shopID, models.ShopWalletLogTypePendingReleased, 1, 10)
	if err != nil {
		t.Fatalf("FindLogs returned error: %v", err)
	}
	if total != 1 || len(got) != 1 {
		t.Fatalf("total=%d len=%d, want 1 and 1", total, len(got))
	}
	if got[0].Type != models.ShopWalletLogTypePendingReleased {
		t.Fatalf("log type = %q, want pending_released", got[0].Type)
	}
}

func TestFindWithdrawalRequestsFiltersByShopAndStatus(t *testing.T) {
	db := newShopWalletRepoTestDB(t)
	repo := NewShopWalletRepository(db)
	shopID := uuid.New()
	otherShopID := uuid.New()
	requests := []models.ShopWithdrawalRequest{
		{ID: uuid.New(), ShopID: shopID, Amount: 100, Status: models.ShopWithdrawalStatusPending, BankInfo: map[string]interface{}{"bank_name": "A", "account_number": "1", "account_holder": "Owner"}},
		{ID: uuid.New(), ShopID: shopID, Amount: 100, Status: models.ShopWithdrawalStatusApproved, BankInfo: map[string]interface{}{"bank_name": "A", "account_number": "1", "account_holder": "Owner"}},
		{ID: uuid.New(), ShopID: otherShopID, Amount: 100, Status: models.ShopWithdrawalStatusPending, BankInfo: map[string]interface{}{"bank_name": "A", "account_number": "1", "account_holder": "Owner"}},
	}
	if err := db.Create(&requests).Error; err != nil {
		t.Fatalf("create requests: %v", err)
	}

	got, total, err := repo.FindWithdrawalRequests(&shopID, models.ShopWithdrawalStatusPending, 1, 10)
	if err != nil {
		t.Fatalf("FindWithdrawalRequests returned error: %v", err)
	}
	if total != 1 || len(got) != 1 {
		t.Fatalf("total=%d len=%d, want 1 and 1", total, len(got))
	}
	if got[0].ShopID != shopID || got[0].Status != models.ShopWithdrawalStatusPending {
		t.Fatalf("request shop/status = %s/%s, want %s/pending", got[0].ShopID, got[0].Status, shopID)
	}
}

func TestHasLogForOrderAndType(t *testing.T) {
	db := newShopWalletRepoTestDB(t)
	repo := NewShopWalletRepository(db)
	shopID := uuid.New()
	wallet, err := repo.FindOrCreateByShopID(db, shopID)
	if err != nil {
		t.Fatalf("create wallet: %v", err)
	}
	orderID := uuid.New()
	log := models.ShopWalletLog{ID: uuid.New(), WalletID: wallet.ID, ShopID: shopID, OrderID: &orderID, Type: models.ShopWalletLogTypeOrderCompletedPending, Amount: 100, Status: models.OrderStatusCompleted}
	if err := db.Create(&log).Error; err != nil {
		t.Fatalf("create log: %v", err)
	}

	exists, err := repo.HasLogForOrderAndType(db, orderID, models.ShopWalletLogTypeOrderCompletedPending)
	if err != nil {
		t.Fatalf("HasLogForOrderAndType returned error: %v", err)
	}
	if !exists {
		t.Fatal("exists = false, want true")
	}
}

func TestFindReleaseEligibleCompletedOrders(t *testing.T) {
	db := newShopWalletRepoTestDB(t)
	orderRepo := NewOrderRepository(db)
	walletRepo := NewShopWalletRepository(db)
	shopID := uuid.New()
	wallet, err := walletRepo.FindOrCreateByShopID(db, shopID)
	if err != nil {
		t.Fatalf("create wallet: %v", err)
	}
	oldCompletedAt := time.Now().UTC().Add(-8 * 24 * time.Hour)
	recentCompletedAt := time.Now().UTC().Add(-6 * 24 * time.Hour)
	eligible := createWalletRepoOrder(t, db, shopID, models.OrderStatusCompleted, &oldCompletedAt)
	recent := createWalletRepoOrder(t, db, shopID, models.OrderStatusCompleted, &recentCompletedAt)
	withoutCreditLog := createWalletRepoOrder(t, db, shopID, models.OrderStatusCompleted, &oldCompletedAt)
	alreadyReleased := createWalletRepoOrder(t, db, shopID, models.OrderStatusCompleted, &oldCompletedAt)

	createWalletRepoLog(t, db, wallet.ID, shopID, eligible.ID, models.ShopWalletLogTypeOrderCompletedPending)
	createWalletRepoLog(t, db, wallet.ID, shopID, recent.ID, models.ShopWalletLogTypeOrderCompletedPending)
	createWalletRepoLog(t, db, wallet.ID, shopID, alreadyReleased.ID, models.ShopWalletLogTypeOrderCompletedPending)
	createWalletRepoLog(t, db, wallet.ID, shopID, alreadyReleased.ID, models.ShopWalletLogTypePendingReleased)

	orders, err := orderRepo.FindWalletReleaseEligibleCompleted(time.Now().UTC().Add(-7 * 24 * time.Hour))
	if err != nil {
		t.Fatalf("FindWalletReleaseEligibleCompleted returned error: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("len(orders) = %d, want 1; recent=%s withoutCreditLog=%s", len(orders), recent.ID, withoutCreditLog.ID)
	}
	if orders[0].ID != eligible.ID {
		t.Fatalf("eligible ID = %s, want %s", orders[0].ID, eligible.ID)
	}
}

func createWalletRepoOrder(t *testing.T, db *gorm.DB, shopID uuid.UUID, status string, completedAt *time.Time) models.Order {
	t.Helper()
	order := models.Order{
		ID:              uuid.New(),
		CustomerID:      uuid.New(),
		ShopID:          shopID,
		OrderNumber:     "ORD-WALLET-" + uuid.NewString()[:8],
		Status:          status,
		CompletedAt:     completedAt,
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

func createWalletRepoLog(t *testing.T, db *gorm.DB, walletID, shopID, orderID uuid.UUID, logType string) {
	t.Helper()
	log := models.ShopWalletLog{ID: uuid.New(), WalletID: walletID, ShopID: shopID, OrderID: &orderID, Type: logType, Amount: 85000, Status: models.OrderStatusCompleted}
	if err := db.Create(&log).Error; err != nil {
		t.Fatalf("create log: %v", err)
	}
}
```

- [ ] **Step 2: Run repository tests to verify they fail**

Run: `go test ./internal/repositories/...`

Expected: FAIL with undefined `NewShopWalletRepository`, `FindOrCreateByShopID`, and `FindWalletReleaseEligibleCompleted`.

- [ ] **Step 3: Implement wallet repository**

Create `internal/repositories/shop_wallet_repo.go`:

```go
package repositories

import (
	"errors"

	"go-fiber/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ShopWalletRepository struct {
	db *gorm.DB
}

func NewShopWalletRepository(db *gorm.DB) *ShopWalletRepository {
	return &ShopWalletRepository{db: db}
}

func (r *ShopWalletRepository) Transaction(fn func(tx *gorm.DB) error) error {
	return r.db.Transaction(fn)
}

func (r *ShopWalletRepository) FindOrCreateByShopID(tx *gorm.DB, shopID uuid.UUID) (*models.ShopWallet, error) {
	if tx == nil {
		tx = r.db
	}
	var wallet models.ShopWallet
	err := tx.Where("shop_id = ?", shopID).First(&wallet).Error
	if err == nil {
		return &wallet, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	wallet = models.ShopWallet{ShopID: shopID}
	if err := tx.Create(&wallet).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			err = tx.Where("shop_id = ?", shopID).First(&wallet).Error
			return &wallet, err
		}
		return nil, err
	}
	return &wallet, nil
}

func (r *ShopWalletRepository) LockByID(tx *gorm.DB, id uuid.UUID) (*models.ShopWallet, error) {
	query := tx
	if tx.Dialector.Name() != "sqlite" {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var wallet models.ShopWallet
	err := query.First(&wallet, "id = ?", id).Error
	return &wallet, err
}

func (r *ShopWalletRepository) LockWithdrawalRequest(tx *gorm.DB, id uuid.UUID) (*models.ShopWithdrawalRequest, error) {
	query := tx
	if tx.Dialector.Name() != "sqlite" {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var request models.ShopWithdrawalRequest
	err := query.First(&request, "id = ?", id).Error
	return &request, err
}

func (r *ShopWalletRepository) CreateLog(tx *gorm.DB, log *models.ShopWalletLog) error {
	return tx.Create(log).Error
}

func (r *ShopWalletRepository) CreateWithdrawalRequest(tx *gorm.DB, request *models.ShopWithdrawalRequest) error {
	return tx.Create(request).Error
}

func (r *ShopWalletRepository) UpdateWallet(tx *gorm.DB, wallet *models.ShopWallet) error {
	return tx.Save(wallet).Error
}

func (r *ShopWalletRepository) UpdateWithdrawalRequest(tx *gorm.DB, request *models.ShopWithdrawalRequest) error {
	return tx.Save(request).Error
}

func (r *ShopWalletRepository) FindLogs(shopID uuid.UUID, logType string, page, limit int) ([]models.ShopWalletLog, int64, error) {
	var logs []models.ShopWalletLog
	var total int64
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	query := r.db.Model(&models.ShopWalletLog{}).Where("shop_id = ?", shopID)
	if logType != "" {
		query = query.Where("type = ?", logType)
	}
	query.Count(&total)
	err := query.Offset((page - 1) * limit).Limit(limit).Order("created_at DESC").Find(&logs).Error
	return logs, total, err
}

func (r *ShopWalletRepository) FindWithdrawalRequests(shopID *uuid.UUID, status string, page, limit int) ([]models.ShopWithdrawalRequest, int64, error) {
	var requests []models.ShopWithdrawalRequest
	var total int64
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	query := r.db.Model(&models.ShopWithdrawalRequest{})
	if shopID != nil {
		query = query.Where("shop_id = ?", *shopID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	query.Count(&total)
	err := query.Offset((page - 1) * limit).Limit(limit).Order("created_at DESC").Find(&requests).Error
	return requests, total, err
}

func (r *ShopWalletRepository) FindWithdrawalRequestByID(id uuid.UUID) (*models.ShopWithdrawalRequest, error) {
	var request models.ShopWithdrawalRequest
	err := r.db.First(&request, "id = ?", id).Error
	return &request, err
}

func (r *ShopWalletRepository) HasLogForOrderAndType(tx *gorm.DB, orderID uuid.UUID, logType string) (bool, error) {
	if tx == nil {
		tx = r.db
	}
	var count int64
	err := tx.Model(&models.ShopWalletLog{}).Where("order_id = ? AND type = ?", orderID, logType).Count(&count).Error
	return count > 0, err
}

func (r *ShopWalletRepository) FindOrderCreditLog(tx *gorm.DB, orderID uuid.UUID) (*models.ShopWalletLog, error) {
	if tx == nil {
		tx = r.db
	}
	var log models.ShopWalletLog
	err := tx.Where("order_id = ? AND type = ?", orderID, models.ShopWalletLogTypeOrderCompletedPending).First(&log).Error
	return &log, err
}
```

- [ ] **Step 4: Implement order repository additions**

Modify `internal/repositories/order_repo.go`:

```go
func (r *OrderRepository) CompleteDeliveredOrder(tx *gorm.DB, orderID uuid.UUID) (int64, error) {
	now := time.Now()
	result := tx.Model(&models.Order{}).
		Where("id = ? AND status = ?", orderID, models.OrderStatusDelivered).
		Updates(map[string]interface{}{
			"status":       models.OrderStatusCompleted,
			"completed_at": &now,
		})
	return result.RowsAffected, result.Error
}

func (r *OrderRepository) FindWalletReleaseEligibleCompleted(cutoff time.Time) ([]models.Order, error) {
	var orders []models.Order
	err := r.db.
		Where("status = ?", models.OrderStatusCompleted).
		Where("completed_at IS NOT NULL AND completed_at <= ?", cutoff).
		Where("EXISTS (?)", r.db.Model(&models.ShopWalletLog{}).
			Select("1").
			Where("shop_wallet_logs.order_id = orders.id").
			Where("shop_wallet_logs.type = ?", models.ShopWalletLogTypeOrderCompletedPending)).
		Where("NOT EXISTS (?)", r.db.Model(&models.ShopWalletLog{}).
			Select("1").
			Where("shop_wallet_logs.order_id = orders.id").
			Where("shop_wallet_logs.type = ?", models.ShopWalletLogTypePendingReleased)).
		Order("completed_at ASC").
		Find(&orders).Error
	return orders, err
}
```

Update the `orders` test table in `internal/repositories/order_repo_test.go` to include:

```sql
"completed_at" timestamp,
```

Update `createRepoTestOrder` to set `CompletedAt` when needed later:

```go
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
```

Extend `TestCompleteDeliveredOrderOnlyUpdatesDeliveredRows` with:

```go
if reloadedDelivered.CompletedAt == nil {
	t.Fatal("CompletedAt is nil, want timestamp")
}
if time.Since(*reloadedDelivered.CompletedAt) > time.Minute {
	t.Fatalf("CompletedAt = %v, want recent timestamp", reloadedDelivered.CompletedAt)
}
```

- [ ] **Step 5: Format and run repository tests**

Run: `gofmt -w internal/repositories/shop_wallet_repo.go internal/repositories/shop_wallet_repo_test.go internal/repositories/order_repo.go internal/repositories/order_repo_test.go`

Run: `go test ./internal/repositories/...`

Expected: PASS.

- [ ] **Step 6: Commit repository work**

Run:

```bash
git add internal/repositories/shop_wallet_repo.go internal/repositories/shop_wallet_repo_test.go internal/repositories/order_repo.go internal/repositories/order_repo_test.go
git commit -m "feat: add shop wallet repository"
```

---

### Task 3: Wallet Service Lifecycle

**Files:**
- Create: `internal/services/shop_wallet_service.go`
- Create: `internal/services/shop_wallet_service_test.go`

- [ ] **Step 1: Write failing service tests**

Create `internal/services/shop_wallet_service_test.go` with these tests:

```go
package services

import (
	"errors"
	"os"
	"testing"
	"time"

	"go-fiber/internal/models"
	"go-fiber/internal/repositories"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func newShopWalletServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := uuid.NewString() + ".db"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
		os.Remove(dsn)
	})
	if err := db.AutoMigrate(&models.Shop{}, &models.ShopWallet{}, &models.ShopWalletLog{}, &models.ShopWithdrawalRequest{}, &models.Order{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return db
}

func newShopWalletServiceForTest(db *gorm.DB) *ShopWalletService {
	walletRepo := repositories.NewShopWalletRepository(db)
	shopRepo := repositories.NewShopRepository(db)
	orderRepo := repositories.NewOrderRepository(db)
	return NewShopWalletService(walletRepo, shopRepo, orderRepo)
}

func TestCreditOrderPendingCreditsRevenueOnce(t *testing.T) {
	db := newShopWalletServiceTestDB(t)
	service := newShopWalletServiceForTest(db)
	order := createWalletServiceOrder(t, db, models.OrderStatusCompleted, time.Now().UTC().Add(-time.Hour))

	if err := db.Transaction(func(tx *gorm.DB) error { return service.CreditOrderPending(tx, order) }); err != nil {
		t.Fatalf("CreditOrderPending returned error: %v", err)
	}
	if err := db.Transaction(func(tx *gorm.DB) error { return service.CreditOrderPending(tx, order) }); err != nil {
		t.Fatalf("CreditOrderPending second call returned error: %v", err)
	}

	wallet := findWalletByShopID(t, db, order.ShopID)
	if wallet.PendingBalance != 85000 {
		t.Fatalf("PendingBalance = %v, want 85000", wallet.PendingBalance)
	}
	var count int64
	db.Model(&models.ShopWalletLog{}).Where("order_id = ? AND type = ?", order.ID, models.ShopWalletLogTypeOrderCompletedPending).Count(&count)
	if count != 1 {
		t.Fatalf("credit log count = %d, want 1", count)
	}
}

func TestReleasePendingForCompletedOrdersBeforeMovesEligibleFunds(t *testing.T) {
	db := newShopWalletServiceTestDB(t)
	service := newShopWalletServiceForTest(db)
	completedAt := time.Now().UTC().Add(-8 * 24 * time.Hour)
	order := createWalletServiceOrder(t, db, models.OrderStatusCompleted, completedAt)
	if err := db.Transaction(func(tx *gorm.DB) error { return service.CreditOrderPending(tx, order) }); err != nil {
		t.Fatalf("credit pending: %v", err)
	}

	released, err := service.ReleasePendingForCompletedOrdersBefore(time.Now().UTC().Add(-7 * 24 * time.Hour))
	if err != nil {
		t.Fatalf("ReleasePendingForCompletedOrdersBefore returned error: %v", err)
	}
	if released != 1 {
		t.Fatalf("released = %d, want 1", released)
	}
	wallet := findWalletByShopID(t, db, order.ShopID)
	if wallet.PendingBalance != 0 || wallet.AvailableBalance != 85000 {
		t.Fatalf("balances = pending %v available %v, want 0 and 85000", wallet.PendingBalance, wallet.AvailableBalance)
	}

	releasedAgain, err := service.ReleasePendingForCompletedOrdersBefore(time.Now().UTC().Add(-7 * 24 * time.Hour))
	if err != nil {
		t.Fatalf("release second call: %v", err)
	}
	if releasedAgain != 0 {
		t.Fatalf("releasedAgain = %d, want 0", releasedAgain)
	}
}

func TestReleasePendingForCompletedOrdersBeforeSkipsRecentOrders(t *testing.T) {
	db := newShopWalletServiceTestDB(t)
	service := newShopWalletServiceForTest(db)
	completedAt := time.Now().UTC().Add(-6 * 24 * time.Hour)
	order := createWalletServiceOrder(t, db, models.OrderStatusCompleted, completedAt)
	if err := db.Transaction(func(tx *gorm.DB) error { return service.CreditOrderPending(tx, order) }); err != nil {
		t.Fatalf("credit pending: %v", err)
	}

	released, err := service.ReleasePendingForCompletedOrdersBefore(time.Now().UTC().Add(-7 * 24 * time.Hour))
	if err != nil {
		t.Fatalf("ReleasePendingForCompletedOrdersBefore returned error: %v", err)
	}
	if released != 0 {
		t.Fatalf("released = %d, want 0", released)
	}
	wallet := findWalletByShopID(t, db, order.ShopID)
	if wallet.PendingBalance != 85000 || wallet.AvailableBalance != 0 {
		t.Fatalf("balances = pending %v available %v, want 85000 and 0", wallet.PendingBalance, wallet.AvailableBalance)
	}
}

func TestCreateWithdrawalHoldsAvailableBalance(t *testing.T) {
	db := newShopWalletServiceTestDB(t)
	service := newShopWalletServiceForTest(db)
	shopID, userID := createWalletServiceShop(t, db)
	seedWallet(t, db, shopID, 0, 100000, 0)

	request, err := service.CreateWithdrawal(userID, CreateWithdrawalInput{
		Amount: 40000,
		BankInfo: map[string]interface{}{
			"bank_name":      "Test Bank",
			"account_number": "123456",
			"account_holder": "Shop Owner",
		},
		Note: "withdraw revenue",
	})
	if err != nil {
		t.Fatalf("CreateWithdrawal returned error: %v", err)
	}
	if request.Status != models.ShopWithdrawalStatusPending {
		t.Fatalf("request status = %q, want pending", request.Status)
	}
	wallet := findWalletByShopID(t, db, shopID)
	if wallet.AvailableBalance != 60000 {
		t.Fatalf("AvailableBalance = %v, want 60000", wallet.AvailableBalance)
	}
}

func TestCreateWithdrawalValidatesBankInfoAndFunds(t *testing.T) {
	db := newShopWalletServiceTestDB(t)
	service := newShopWalletServiceForTest(db)
	shopID, userID := createWalletServiceShop(t, db)
	seedWallet(t, db, shopID, 0, 10000, 0)

	_, err := service.CreateWithdrawal(userID, CreateWithdrawalInput{Amount: 40000, BankInfo: map[string]interface{}{"bank_name": "Test Bank", "account_number": "123456", "account_holder": "Shop Owner"}})
	if !errors.Is(err, ErrInsufficientAvailableBalance) {
		t.Fatalf("err = %v, want ErrInsufficientAvailableBalance", err)
	}
	_, err = service.CreateWithdrawal(userID, CreateWithdrawalInput{Amount: 1000, BankInfo: map[string]interface{}{"bank_name": "Test Bank"}})
	if !errors.Is(err, ErrInvalidBankInfo) {
		t.Fatalf("err = %v, want ErrInvalidBankInfo", err)
	}
}

func TestApproveWithdrawalIncrementsWithdrawnBalance(t *testing.T) {
	db := newShopWalletServiceTestDB(t)
	service := newShopWalletServiceForTest(db)
	shopID, userID := createWalletServiceShop(t, db)
	seedWallet(t, db, shopID, 0, 100000, 0)
	request, err := service.CreateWithdrawal(userID, CreateWithdrawalInput{Amount: 40000, BankInfo: validBankInfoForTest()})
	if err != nil {
		t.Fatalf("CreateWithdrawal returned error: %v", err)
	}
	adminID := uuid.New()

	approved, err := service.ApproveWithdrawal(request.ID, adminID)
	if err != nil {
		t.Fatalf("ApproveWithdrawal returned error: %v", err)
	}
	if approved.Status != models.ShopWithdrawalStatusApproved {
		t.Fatalf("status = %q, want approved", approved.Status)
	}
	wallet := findWalletByShopID(t, db, shopID)
	if wallet.AvailableBalance != 60000 || wallet.WithdrawnBalance != 40000 {
		t.Fatalf("balances = available %v withdrawn %v, want 60000 and 40000", wallet.AvailableBalance, wallet.WithdrawnBalance)
	}
	var log models.ShopWalletLog
	if err := db.Where("withdrawal_request_id = ? AND type = ?", request.ID, models.ShopWalletLogTypeWithdrawalApproved).First(&log).Error; err != nil {
		t.Fatalf("find approved log: %v", err)
	}
	if log.WithdrawnBefore != 0 || log.WithdrawnAfter != 40000 {
		t.Fatalf("withdrawn before/after = %v/%v, want 0/40000", log.WithdrawnBefore, log.WithdrawnAfter)
	}
}

func TestRejectWithdrawalReturnsAvailableBalance(t *testing.T) {
	db := newShopWalletServiceTestDB(t)
	service := newShopWalletServiceForTest(db)
	shopID, userID := createWalletServiceShop(t, db)
	seedWallet(t, db, shopID, 0, 100000, 0)
	request, err := service.CreateWithdrawal(userID, CreateWithdrawalInput{Amount: 40000, BankInfo: validBankInfoForTest()})
	if err != nil {
		t.Fatalf("CreateWithdrawal returned error: %v", err)
	}

	rejected, err := service.RejectWithdrawal(request.ID, uuid.New(), "invalid bank account")
	if err != nil {
		t.Fatalf("RejectWithdrawal returned error: %v", err)
	}
	if rejected.Status != models.ShopWithdrawalStatusRejected {
		t.Fatalf("status = %q, want rejected", rejected.Status)
	}
	wallet := findWalletByShopID(t, db, shopID)
	if wallet.AvailableBalance != 100000 || wallet.WithdrawnBalance != 0 {
		t.Fatalf("balances = available %v withdrawn %v, want 100000 and 0", wallet.AvailableBalance, wallet.WithdrawnBalance)
	}
}
```

Add helper functions in the same file:

```go
func createWalletServiceShop(t *testing.T, db *gorm.DB) (uuid.UUID, uuid.UUID) {
	t.Helper()
	shopID := uuid.New()
	userID := uuid.New()
	shop := models.Shop{ID: shopID, UserID: userID, Name: "Wallet Shop", Slug: "wallet-shop", Status: "active"}
	if err := db.Create(&shop).Error; err != nil {
		t.Fatalf("create shop: %v", err)
	}
	return shopID, userID
}

func createWalletServiceOrder(t *testing.T, db *gorm.DB, status string, completedAt time.Time) models.Order {
	t.Helper()
	shopID, _ := createWalletServiceShop(t, db)
	order := models.Order{ID: uuid.New(), CustomerID: uuid.New(), ShopID: shopID, OrderNumber: "ORD-SVC-" + uuid.NewString()[:8], Status: status, CompletedAt: &completedAt, SubTotal: 100000, ShippingFee: 15000, TotalAmount: 115000, ShippingAddress: map[string]interface{}{"address": "Test address"}}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}
	return order
}

func seedWallet(t *testing.T, db *gorm.DB, shopID uuid.UUID, pending, available, withdrawn float64) models.ShopWallet {
	t.Helper()
	wallet := models.ShopWallet{ID: uuid.New(), ShopID: shopID, PendingBalance: pending, AvailableBalance: available, WithdrawnBalance: withdrawn}
	if err := db.Create(&wallet).Error; err != nil {
		t.Fatalf("create wallet: %v", err)
	}
	return wallet
}

func findWalletByShopID(t *testing.T, db *gorm.DB, shopID uuid.UUID) models.ShopWallet {
	t.Helper()
	var wallet models.ShopWallet
	if err := db.First(&wallet, "shop_id = ?", shopID).Error; err != nil {
		t.Fatalf("find wallet: %v", err)
	}
	return wallet
}

func validBankInfoForTest() map[string]interface{} {
	return map[string]interface{}{"bank_name": "Test Bank", "account_number": "123456", "account_holder": "Shop Owner"}
}
```

- [ ] **Step 2: Run service tests to verify they fail**

Run: `go test ./internal/services/...`

Expected: FAIL with undefined `ShopWalletService`, `NewShopWalletService`, `CreateWithdrawalInput`, and wallet service errors.

- [ ] **Step 3: Implement wallet service**

Create `internal/services/shop_wallet_service.go`:

```go
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
	ErrInvalidAmount                = errors.New("amount must be greater than 0")
	ErrInvalidBankInfo              = errors.New("bank_info must include bank_name, account_number, and account_holder")
	ErrInsufficientAvailableBalance = errors.New("insufficient available balance")
	ErrInsufficientPendingBalance   = errors.New("insufficient pending balance")
	ErrWithdrawalNotFound           = errors.New("withdrawal request not found")
	ErrWithdrawalAlreadyProcessed   = errors.New("withdrawal request already processed")
)

type CreateWithdrawalInput struct {
	Amount   float64                `json:"amount" validate:"required,gt=0"`
	BankInfo map[string]interface{} `json:"bank_info" validate:"required"`
	Note     string                 `json:"note"`
}

type RejectWithdrawalInput struct {
	AdminNote string `json:"admin_note"`
}

type ShopWalletService struct {
	walletRepo *repositories.ShopWalletRepository
	shopRepo   *repositories.ShopRepository
	orderRepo  *repositories.OrderRepository
}

func NewShopWalletService(walletRepo *repositories.ShopWalletRepository, shopRepo *repositories.ShopRepository, orderRepo *repositories.OrderRepository) *ShopWalletService {
	return &ShopWalletService{walletRepo: walletRepo, shopRepo: shopRepo, orderRepo: orderRepo}
}

func (s *ShopWalletService) GetWalletByUserID(userID uuid.UUID) (*models.ShopWallet, error) {
	shop, err := s.shopRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}
	return s.walletRepo.FindOrCreateByShopID(nil, shop.ID)
}

func (s *ShopWalletService) CreditOrderPending(tx *gorm.DB, order models.Order) error {
	if order.ID == uuid.Nil || order.ShopID == uuid.Nil {
		return nil
	}
	exists, err := s.walletRepo.HasLogForOrderAndType(tx, order.ID, models.ShopWalletLogTypeOrderCompletedPending)
	if err != nil || exists {
		return err
	}
	wallet, err := s.walletRepo.FindOrCreateByShopID(tx, order.ShopID)
	if err != nil {
		return err
	}
	wallet, err = s.walletRepo.LockByID(tx, wallet.ID)
	if err != nil {
		return err
	}
	amount := order.TotalAmount - order.ShippingFee
	if amount <= 0 {
		return ErrInvalidAmount
	}
	before := *wallet
	wallet.PendingBalance += amount
	if err := s.walletRepo.UpdateWallet(tx, wallet); err != nil {
		return err
	}
	return s.walletRepo.CreateLog(tx, &models.ShopWalletLog{WalletID: wallet.ID, ShopID: wallet.ShopID, OrderID: &order.ID, Type: models.ShopWalletLogTypeOrderCompletedPending, Amount: amount, AvailableBefore: before.AvailableBalance, AvailableAfter: wallet.AvailableBalance, PendingBefore: before.PendingBalance, PendingAfter: wallet.PendingBalance, WithdrawnBefore: before.WithdrawnBalance, WithdrawnAfter: wallet.WithdrawnBalance, Status: models.OrderStatusCompleted, Description: "Order completed; revenue moved to pending balance"})
}

func (s *ShopWalletService) ReleasePendingForCompletedOrdersBefore(cutoff time.Time) (int, error) {
	orders, err := s.orderRepo.FindWalletReleaseEligibleCompleted(cutoff)
	if err != nil {
		return 0, err
	}
	released := 0
	for _, order := range orders {
		err := s.walletRepo.Transaction(func(tx *gorm.DB) error {
			alreadyReleased, err := s.walletRepo.HasLogForOrderAndType(tx, order.ID, models.ShopWalletLogTypePendingReleased)
			if err != nil || alreadyReleased {
				return err
			}
			creditLog, err := s.walletRepo.FindOrderCreditLog(tx, order.ID)
			if err != nil {
				return err
			}
			wallet, err := s.walletRepo.FindOrCreateByShopID(tx, order.ShopID)
			if err != nil {
				return err
			}
			wallet, err = s.walletRepo.LockByID(tx, wallet.ID)
			if err != nil {
				return err
			}
			if wallet.PendingBalance < creditLog.Amount {
				return ErrInsufficientPendingBalance
			}
			before := *wallet
			wallet.PendingBalance -= creditLog.Amount
			wallet.AvailableBalance += creditLog.Amount
			if err := s.walletRepo.UpdateWallet(tx, wallet); err != nil {
				return err
			}
			return s.walletRepo.CreateLog(tx, &models.ShopWalletLog{WalletID: wallet.ID, ShopID: wallet.ShopID, OrderID: &order.ID, Type: models.ShopWalletLogTypePendingReleased, Amount: creditLog.Amount, AvailableBefore: before.AvailableBalance, AvailableAfter: wallet.AvailableBalance, PendingBefore: before.PendingBalance, PendingAfter: wallet.PendingBalance, WithdrawnBefore: before.WithdrawnBalance, WithdrawnAfter: wallet.WithdrawnBalance, Status: models.OrderStatusCompleted, Description: "Pending balance released after 7 days"})
		})
		if err != nil {
			return released, err
		}
		released++
	}
	return released, nil
}

func (s *ShopWalletService) CreateWithdrawal(userID uuid.UUID, input CreateWithdrawalInput) (*models.ShopWithdrawalRequest, error) {
	if input.Amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if !hasRequiredBankInfo(input.BankInfo) {
		return nil, ErrInvalidBankInfo
	}
	shop, err := s.shopRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}
	var request *models.ShopWithdrawalRequest
	err = s.walletRepo.Transaction(func(tx *gorm.DB) error {
		wallet, err := s.walletRepo.FindOrCreateByShopID(tx, shop.ID)
		if err != nil {
			return err
		}
		wallet, err = s.walletRepo.LockByID(tx, wallet.ID)
		if err != nil {
			return err
		}
		if wallet.AvailableBalance < input.Amount {
			return ErrInsufficientAvailableBalance
		}
		request = &models.ShopWithdrawalRequest{ShopID: shop.ID, Amount: input.Amount, Status: models.ShopWithdrawalStatusPending, BankInfo: input.BankInfo, Note: input.Note}
		if err := s.walletRepo.CreateWithdrawalRequest(tx, request); err != nil {
			return err
		}
		before := *wallet
		wallet.AvailableBalance -= input.Amount
		if err := s.walletRepo.UpdateWallet(tx, wallet); err != nil {
			return err
		}
		return s.walletRepo.CreateLog(tx, &models.ShopWalletLog{WalletID: wallet.ID, ShopID: wallet.ShopID, WithdrawalRequestID: &request.ID, Type: models.ShopWalletLogTypeWithdrawalHold, Amount: input.Amount, AvailableBefore: before.AvailableBalance, AvailableAfter: wallet.AvailableBalance, PendingBefore: before.PendingBalance, PendingAfter: wallet.PendingBalance, WithdrawnBefore: before.WithdrawnBalance, WithdrawnAfter: wallet.WithdrawnBalance, Status: models.ShopWithdrawalStatusPending, Description: "Withdrawal request created; available balance held"})
	})
	return request, err
}

func (s *ShopWalletService) ApproveWithdrawal(id, adminID uuid.UUID) (*models.ShopWithdrawalRequest, error) {
	return s.reviewWithdrawal(id, adminID, models.ShopWithdrawalStatusApproved, "")
}

func (s *ShopWalletService) RejectWithdrawal(id, adminID uuid.UUID, adminNote string) (*models.ShopWithdrawalRequest, error) {
	return s.reviewWithdrawal(id, adminID, models.ShopWithdrawalStatusRejected, adminNote)
}

func (s *ShopWalletService) reviewWithdrawal(id, adminID uuid.UUID, status, adminNote string) (*models.ShopWithdrawalRequest, error) {
	var request *models.ShopWithdrawalRequest
	err := s.walletRepo.Transaction(func(tx *gorm.DB) error {
		lockedRequest, err := s.walletRepo.LockWithdrawalRequest(tx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrWithdrawalNotFound
			}
			return err
		}
		if lockedRequest.Status != models.ShopWithdrawalStatusPending {
			return ErrWithdrawalAlreadyProcessed
		}
		wallet, err := s.walletRepo.FindOrCreateByShopID(tx, lockedRequest.ShopID)
		if err != nil {
			return err
		}
		wallet, err = s.walletRepo.LockByID(tx, wallet.ID)
		if err != nil {
			return err
		}
		before := *wallet
		now := time.Now()
		lockedRequest.Status = status
		lockedRequest.ReviewedBy = &adminID
		lockedRequest.ReviewedAt = &now
		lockedRequest.AdminNote = adminNote
		logType := models.ShopWalletLogTypeWithdrawalApproved
		description := "Withdrawal request approved"
		if status == models.ShopWithdrawalStatusApproved {
			wallet.WithdrawnBalance += lockedRequest.Amount
		} else {
			wallet.AvailableBalance += lockedRequest.Amount
			logType = models.ShopWalletLogTypeWithdrawalRejected
			description = "Withdrawal request rejected; held balance returned"
		}
		if err := s.walletRepo.UpdateWithdrawalRequest(tx, lockedRequest); err != nil {
			return err
		}
		if err := s.walletRepo.UpdateWallet(tx, wallet); err != nil {
			return err
		}
		if err := s.walletRepo.CreateLog(tx, &models.ShopWalletLog{WalletID: wallet.ID, ShopID: wallet.ShopID, WithdrawalRequestID: &lockedRequest.ID, Type: logType, Amount: lockedRequest.Amount, AvailableBefore: before.AvailableBalance, AvailableAfter: wallet.AvailableBalance, PendingBefore: before.PendingBalance, PendingAfter: wallet.PendingBalance, WithdrawnBefore: before.WithdrawnBalance, WithdrawnAfter: wallet.WithdrawnBalance, Status: status, Description: description}); err != nil {
			return err
		}
		request = lockedRequest
		return nil
	})
	return request, err
}

func hasRequiredBankInfo(bankInfo map[string]interface{}) bool {
	for _, key := range []string{"bank_name", "account_number", "account_holder"} {
		value, ok := bankInfo[key]
		if !ok {
			return false
		}
		text, ok := value.(string)
		if !ok || text == "" {
			return false
		}
	}
	return true
}
```

- [ ] **Step 4: Add read/query service methods for handlers**

Append these methods to `internal/services/shop_wallet_service.go`:

```go
func (s *ShopWalletService) GetLogsByUserID(userID uuid.UUID, logType string, page, limit int) ([]models.ShopWalletLog, int64, error) {
	shop, err := s.shopRepo.FindByUserID(userID)
	if err != nil {
		return nil, 0, err
	}
	return s.walletRepo.FindLogs(shop.ID, logType, page, limit)
}

func (s *ShopWalletService) GetWithdrawalsByUserID(userID uuid.UUID, status string, page, limit int) ([]models.ShopWithdrawalRequest, int64, error) {
	shop, err := s.shopRepo.FindByUserID(userID)
	if err != nil {
		return nil, 0, err
	}
	return s.walletRepo.FindWithdrawalRequests(&shop.ID, status, page, limit)
}

func (s *ShopWalletService) GetWithdrawalByUserID(userID, requestID uuid.UUID) (*models.ShopWithdrawalRequest, error) {
	shop, err := s.shopRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}
	request, err := s.walletRepo.FindWithdrawalRequestByID(requestID)
	if err != nil {
		return nil, err
	}
	if request.ShopID != shop.ID {
		return nil, ErrWithdrawalNotFound
	}
	return request, nil
}

func (s *ShopWalletService) GetWallets(shopID *uuid.UUID, page, limit int) ([]models.ShopWallet, int64, error) {
	return s.walletRepo.FindWallets(shopID, page, limit)
}

func (s *ShopWalletService) GetLogsByShopID(shopID uuid.UUID, logType string, page, limit int) ([]models.ShopWalletLog, int64, error) {
	return s.walletRepo.FindLogs(shopID, logType, page, limit)
}

func (s *ShopWalletService) GetWithdrawalRequests(shopID *uuid.UUID, status string, page, limit int) ([]models.ShopWithdrawalRequest, int64, error) {
	return s.walletRepo.FindWithdrawalRequests(shopID, status, page, limit)
}

func (s *ShopWalletService) GetWithdrawalRequestByID(id uuid.UUID) (*models.ShopWithdrawalRequest, error) {
	return s.walletRepo.FindWithdrawalRequestByID(id)
}
```

Also add this repository method to `internal/repositories/shop_wallet_repo.go`:

```go
func (r *ShopWalletRepository) FindWallets(shopID *uuid.UUID, page, limit int) ([]models.ShopWallet, int64, error) {
	var wallets []models.ShopWallet
	var total int64
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	query := r.db.Model(&models.ShopWallet{})
	if shopID != nil {
		query = query.Where("shop_id = ?", *shopID)
	}
	query.Count(&total)
	err := query.Offset((page - 1) * limit).Limit(limit).Order("created_at DESC").Find(&wallets).Error
	return wallets, total, err
}
```

- [ ] **Step 5: Format and run service tests**

Run: `gofmt -w internal/services/shop_wallet_service.go internal/services/shop_wallet_service_test.go internal/repositories/shop_wallet_repo.go`

Run: `go test ./internal/services/...`

Expected: PASS for the new service tests; existing order service tests may fail until Task 4 updates constructors and order test tables.

- [ ] **Step 6: Commit wallet service**

Run:

```bash
git add internal/services/shop_wallet_service.go internal/services/shop_wallet_service_test.go internal/repositories/shop_wallet_repo.go
git commit -m "feat: add shop wallet service"
```

---

### Task 4: Order Completion Integration

**Files:**
- Modify: `internal/services/order_service.go`
- Modify: `internal/services/order_service_test.go`

- [ ] **Step 1: Update failing order service tests**

Modify `internal/services/order_service_test.go` order table SQL to add:

```sql
"completed_at" timestamp,
```

Add wallet tables to `createTableSQL`:

```go
`CREATE TABLE "shop_wallets" (
	"id" text PRIMARY KEY,
	"shop_id" text NOT NULL UNIQUE,
	"pending_balance" real NOT NULL DEFAULT 0,
	"available_balance" real NOT NULL DEFAULT 0,
	"withdrawn_balance" real NOT NULL DEFAULT 0,
	"created_at" timestamp,
	"updated_at" timestamp
)`,
`CREATE TABLE "shop_wallet_logs" (
	"id" text PRIMARY KEY,
	"wallet_id" text NOT NULL,
	"shop_id" text NOT NULL,
	"order_id" text,
	"withdrawal_request_id" text,
	"type" text NOT NULL,
	"amount" real NOT NULL,
	"available_before" real NOT NULL DEFAULT 0,
	"available_after" real NOT NULL DEFAULT 0,
	"pending_before" real NOT NULL DEFAULT 0,
	"pending_after" real NOT NULL DEFAULT 0,
	"withdrawn_before" real NOT NULL DEFAULT 0,
	"withdrawn_after" real NOT NULL DEFAULT 0,
	"status" text NOT NULL,
	"description" text,
	"metadata" text,
	"created_at" timestamp
)`,
`CREATE TABLE "shop_withdrawal_requests" (
	"id" text PRIMARY KEY,
	"shop_id" text NOT NULL,
	"amount" real NOT NULL,
	"status" text NOT NULL,
	"bank_info" text NOT NULL,
	"note" text,
	"admin_note" text,
	"reviewed_by" text,
	"reviewed_at" timestamp,
	"created_at" timestamp,
	"updated_at" timestamp
)`,
```

Add this helper:

```go
func newOrderServiceWalletService(db *gorm.DB) *ShopWalletService {
	walletRepo := repositories.NewShopWalletRepository(db)
	shopRepo := repositories.NewShopRepository(db)
	orderRepo := repositories.NewOrderRepository(db)
	return NewShopWalletService(walletRepo, shopRepo, orderRepo)
}
```

Add this test:

```go
func TestAutoCompleteDeliveredOrdersBeforeCreditsWalletPending(t *testing.T) {
	db := newOrderServiceTestDB(t)
	orderRepo := repositories.NewOrderRepository(db)
	walletSvc := newOrderServiceWalletService(db)
	orderSvc := NewOrderService(orderRepo, nil, nil, nil, nil, walletSvc)
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

	var wallet models.ShopWallet
	if err := db.First(&wallet, "shop_id = ?", order.ShopID).Error; err != nil {
		t.Fatalf("find wallet: %v", err)
	}
	if wallet.PendingBalance != 100000 {
		t.Fatalf("PendingBalance = %v, want 100000", wallet.PendingBalance)
	}
	var completed models.Order
	if err := db.First(&completed, "id = ?", order.ID).Error; err != nil {
		t.Fatalf("find order: %v", err)
	}
	if completed.CompletedAt == nil {
		t.Fatal("CompletedAt is nil, want timestamp")
	}
}
```

- [ ] **Step 2: Run order service tests to verify they fail**

Run: `go test ./internal/services/...`

Expected: FAIL with constructor arity mismatch or missing wallet credit behavior.

- [ ] **Step 3: Update OrderService constructor and completion flow**

Modify `internal/services/order_service.go` struct and constructor:

```go
type OrderService struct {
	repo        *repositories.OrderRepository
	paymentSvc  *PaymentService
	customerRepo *repositories.CustomerRepository
	productRepo  *repositories.ProductRepository
	shippingSvc  *ShippingService
	walletSvc    *ShopWalletService
}

func NewOrderService(
	repo *repositories.OrderRepository,
	paymentSvc *PaymentService,
	customerRepo *repositories.CustomerRepository,
	productRepo *repositories.ProductRepository,
	shippingSvc *ShippingService,
	walletSvc *ShopWalletService,
) *OrderService {
	return &OrderService{repo: repo, paymentSvc: paymentSvc, customerRepo: customerRepo, productRepo: productRepo, shippingSvc: shippingSvc, walletSvc: walletSvc}
}
```

In `AutoCompleteDeliveredOrdersBefore`, after `rowsAffected == 0` guard and before incrementing `completedCount`, add:

```go
			if s.walletSvc != nil {
				completedOrder := order
				completedOrder.Status = models.OrderStatusCompleted
				if err := s.walletSvc.CreditOrderPending(tx, completedOrder); err != nil {
					return err
				}
			}
```

Update all existing `NewOrderService` calls in tests and `cmd/server/main.go` later to pass a sixth argument.

- [ ] **Step 4: Update order service tests to pass nil wallet service when not testing wallet**

Replace existing constructor calls in `internal/services/order_service_test.go`:

```go
orderSvc := NewOrderService(orderRepo, paymentSvc, nil, nil, nil, nil)
```

and:

```go
orderSvc := NewOrderService(orderRepo, nil, nil, nil, nil, nil)
```

- [ ] **Step 5: Format and run service tests**

Run: `gofmt -w internal/services/order_service.go internal/services/order_service_test.go`

Run: `go test ./internal/services/...`

Expected: PASS.

- [ ] **Step 6: Commit order integration**

Run:

```bash
git add internal/services/order_service.go internal/services/order_service_test.go
git commit -m "feat: credit wallet on order completion"
```

---

### Task 5: Wallet Release Cron

**Files:**
- Modify: `internal/cron/cron.go`
- Modify: `internal/cron/cron_test.go`

- [ ] **Step 1: Update failing cron tests**

Modify `internal/cron/cron_test.go`:

```go
func TestWalletReleaseCronSpecRunsAtTwoAM(t *testing.T) {
	schedule, err := cron.ParseStandard(walletReleaseCronSpec)
	if err != nil {
		t.Fatalf("ParseStandard returned error: %v", err)
	}
	next := schedule.Next(time.Date(2026, 6, 23, 0, 0, 0, 0, time.Local))
	if next.Hour() != 2 || next.Minute() != 0 {
		t.Fatalf("next run = %v, want 02:00", next)
	}
}

func TestManagerStartRegistersTwoJobs(t *testing.T) {
	manager := NewManager(nil, nil)
	err := manager.Start()
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer manager.Stop()

	entries := manager.cronRunner.Entries()
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}
}
```

Remove the old `TestManagerStartRegistersOneJob`.

- [ ] **Step 2: Run cron tests to verify they fail**

Run: `go test ./internal/cron/...`

Expected: FAIL with undefined `walletReleaseCronSpec` and constructor arity mismatch.

- [ ] **Step 3: Implement wallet release cron registration**

Modify `internal/cron/cron.go`:

```go
const orderCompletionCronSpec = "0 2 * * *"
const walletReleaseCronSpec = "0 2 * * *"

type Manager struct {
	orderService  *services.OrderService
	walletService *services.ShopWalletService
	cronRunner    *cron.Cron
}

func NewManager(orderService *services.OrderService, walletService *services.ShopWalletService) *Manager {
	return &Manager{orderService: orderService, walletService: walletService, cronRunner: cron.New()}
}
```

In `Start`, keep the existing order job and add:

```go
	_, err = m.cronRunner.AddFunc(walletReleaseCronSpec, func() {
		if m.walletService == nil {
			log.Printf("wallet release cron skipped: wallet service is nil")
			return
		}
		releasedCount, err := m.walletService.ReleasePendingForCompletedOrdersBefore(time.Now().Add(-7 * 24 * time.Hour))
		if err != nil {
			log.Printf("wallet release cron failed: %v", err)
			return
		}
		log.Printf("wallet release cron released %d orders", releasedCount)
	})
	if err != nil {
		return err
	}
```

Add `time` to imports.

- [ ] **Step 4: Format and run cron tests**

Run: `gofmt -w internal/cron/cron.go internal/cron/cron_test.go`

Run: `go test ./internal/cron/...`

Expected: PASS.

- [ ] **Step 5: Commit cron work**

Run:

```bash
git add internal/cron/cron.go internal/cron/cron_test.go
git commit -m "feat: schedule wallet release cron"
```

---

### Task 6: HTTP Handlers, Routes, And Permissions

**Files:**
- Create: `internal/handlers/shop_wallet_handler.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Implement shop wallet handler**

Create `internal/handlers/shop_wallet_handler.go`:

```go
package handlers

import (
	"errors"
	"strconv"

	"go-fiber/internal/services"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ShopWalletHandler struct {
	service *services.ShopWalletService
}

func NewShopWalletHandler(service *services.ShopWalletService) *ShopWalletHandler {
	return &ShopWalletHandler{service: service}
}

func (h *ShopWalletHandler) GetMyWallet(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return utils.Error(c, 401, "UNAUTHORIZED", "Invalid user ID in token")
	}
	wallet, err := h.service.GetWalletByUserID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.Error(c, 404, "NOT_FOUND", "Shop not found")
		}
		return utils.Error(c, 500, "INTERNAL_ERROR", "Failed to fetch wallet")
	}
	return utils.Success(c, wallet, "")
}

func (h *ShopWalletHandler) GetMyWalletLogs(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return utils.Error(c, 401, "UNAUTHORIZED", "Invalid user ID in token")
	}
	page, limit := pagination(c)
	logs, total, err := h.service.GetLogsByUserID(userID, c.Query("type"), page, limit)
	if err != nil {
		return utils.Error(c, 500, "INTERNAL_ERROR", "Failed to fetch wallet logs")
	}
	return utils.SuccessWithPagination(c, logs, page, limit, total)
}

func (h *ShopWalletHandler) CreateWithdrawal(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return utils.Error(c, 401, "UNAUTHORIZED", "Invalid user ID in token")
	}
	var input services.CreateWithdrawalInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}
	request, err := h.service.CreateWithdrawal(userID, input)
	if err != nil {
		return walletServiceError(c, err)
	}
	return utils.Success(c, request, "Withdrawal request created")
}

func (h *ShopWalletHandler) GetMyWithdrawals(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return utils.Error(c, 401, "UNAUTHORIZED", "Invalid user ID in token")
	}
	page, limit := pagination(c)
	requests, total, err := h.service.GetWithdrawalsByUserID(userID, c.Query("status"), page, limit)
	if err != nil {
		return utils.Error(c, 500, "INTERNAL_ERROR", "Failed to fetch withdrawals")
	}
	return utils.SuccessWithPagination(c, requests, page, limit, total)
}

func (h *ShopWalletHandler) GetMyWithdrawal(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return utils.Error(c, 401, "UNAUTHORIZED", "Invalid user ID in token")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}
	request, err := h.service.GetWithdrawalByUserID(userID, id)
	if err != nil {
		return utils.Error(c, 404, "NOT_FOUND", "Withdrawal request not found")
	}
	return utils.Success(c, request, "")
}

func (h *ShopWalletHandler) GetAdminWallets(c *fiber.Ctx) error {
	page, limit := pagination(c)
	var shopID *uuid.UUID
	if raw := c.Query("shop_id"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid shop ID")
		}
		shopID = &parsed
	}
	wallets, total, err := h.service.GetWallets(shopID, page, limit)
	if err != nil {
		return utils.Error(c, 500, "INTERNAL_ERROR", "Failed to fetch wallets")
	}
	return utils.SuccessWithPagination(c, wallets, page, limit, total)
}

func (h *ShopWalletHandler) GetAdminWalletLogs(c *fiber.Ctx) error {
	shopID, err := uuid.Parse(c.Params("shop_id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid shop ID")
	}
	page, limit := pagination(c)
	logs, total, err := h.service.GetLogsByShopID(shopID, c.Query("type"), page, limit)
	if err != nil {
		return utils.Error(c, 500, "INTERNAL_ERROR", "Failed to fetch wallet logs")
	}
	return utils.SuccessWithPagination(c, logs, page, limit, total)
}

func (h *ShopWalletHandler) GetAdminWithdrawals(c *fiber.Ctx) error {
	page, limit := pagination(c)
	var shopID *uuid.UUID
	if raw := c.Query("shop_id"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid shop ID")
		}
		shopID = &parsed
	}
	requests, total, err := h.service.GetWithdrawalRequests(shopID, c.Query("status"), page, limit)
	if err != nil {
		return utils.Error(c, 500, "INTERNAL_ERROR", "Failed to fetch withdrawals")
	}
	return utils.SuccessWithPagination(c, requests, page, limit, total)
}

func (h *ShopWalletHandler) GetAdminWithdrawal(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}
	request, err := h.service.GetWithdrawalRequestByID(id)
	if err != nil {
		return utils.Error(c, 404, "NOT_FOUND", "Withdrawal request not found")
	}
	return utils.Success(c, request, "")
}

func (h *ShopWalletHandler) ApproveWithdrawal(c *fiber.Ctx) error {
	id, adminID, ok := adminRequestIDs(c)
	if !ok {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}
	request, err := h.service.ApproveWithdrawal(id, adminID)
	if err != nil {
		return walletServiceError(c, err)
	}
	return utils.Success(c, request, "Withdrawal approved")
}

func (h *ShopWalletHandler) RejectWithdrawal(c *fiber.Ctx) error {
	id, adminID, ok := adminRequestIDs(c)
	if !ok {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}
	var input services.RejectWithdrawalInput
	if err := c.BodyParser(&input); err != nil {
		input.AdminNote = ""
	}
	request, err := h.service.RejectWithdrawal(id, adminID, input.AdminNote)
	if err != nil {
		return walletServiceError(c, err)
	}
	return utils.Success(c, request, "Withdrawal rejected")
}

func currentUserID(c *fiber.Ctx) (uuid.UUID, error) {
	userID, ok := c.Locals("userID").(string)
	if !ok {
		return uuid.Nil, fiber.ErrUnauthorized
	}
	return uuid.Parse(userID)
}

func adminRequestIDs(c *fiber.Ctx) (uuid.UUID, uuid.UUID, bool) {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, false
	}
	adminID, err := currentUserID(c)
	if err != nil {
		return uuid.Nil, uuid.Nil, false
	}
	return id, adminID, true
}

func pagination(c *fiber.Ctx) (int, int) {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	return page, limit
}

func walletServiceError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, services.ErrInvalidAmount), errors.Is(err, services.ErrInvalidBankInfo), errors.Is(err, services.ErrInsufficientAvailableBalance), errors.Is(err, services.ErrInsufficientPendingBalance), errors.Is(err, services.ErrWithdrawalAlreadyProcessed):
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, services.ErrWithdrawalNotFound), errors.Is(err, gorm.ErrRecordNotFound):
		return utils.Error(c, 404, "NOT_FOUND", err.Error())
	default:
		return utils.Error(c, 500, "INTERNAL_ERROR", "Wallet operation failed")
	}
}
```

- [ ] **Step 2: Wire dependencies and routes**

Modify `cmd/server/main.go` repository wiring:

```go
walletRepo := repositories.NewShopWalletRepository(db)
```

Modify service wiring:

```go
walletService := services.NewShopWalletService(walletRepo, shopRepo, orderRepo)
orderService := services.NewOrderService(orderRepo, paymentSvc, customerRepo, productRepo, shippingSvc, walletService)
```

Modify cron wiring:

```go
cronManager := cron.NewManager(orderService, walletService)
```

Modify handler wiring:

```go
walletHandler := handlers.NewShopWalletHandler(walletService)
```

Add shop owner routes after customer order routes:

```go
shopOwner := api.Group("/shop", middleware.JWTAuth(cfg))
shopOwner.Get("/wallet", walletHandler.GetMyWallet)
shopOwner.Get("/wallet/logs", walletHandler.GetMyWalletLogs)
shopOwner.Post("/withdrawals", walletHandler.CreateWithdrawal)
shopOwner.Get("/withdrawals", walletHandler.GetMyWithdrawals)
shopOwner.Get("/withdrawals/:id", walletHandler.GetMyWithdrawal)
```

Add admin wallet routes near admin ecommerce routes:

```go
adminWallets := api.Group("/admin/wallets", middleware.JWTAuth(cfg))
adminWallets.Get("/", middleware.RequirePermission(userRepo, "wallet:read"), walletHandler.GetAdminWallets)
adminWallets.Get("/:shop_id/logs", middleware.RequirePermission(userRepo, "wallet:read"), walletHandler.GetAdminWalletLogs)

adminWithdrawals := api.Group("/admin/withdrawals", middleware.JWTAuth(cfg))
adminWithdrawals.Get("/", middleware.RequirePermission(userRepo, "withdrawal:read"), walletHandler.GetAdminWithdrawals)
adminWithdrawals.Get("/:id", middleware.RequirePermission(userRepo, "withdrawal:read"), walletHandler.GetAdminWithdrawal)
adminWithdrawals.Post("/:id/approve", middleware.RequirePermission(userRepo, "withdrawal:write"), walletHandler.ApproveWithdrawal)
adminWithdrawals.Post("/:id/reject", middleware.RequirePermission(userRepo, "withdrawal:write"), walletHandler.RejectWithdrawal)
```

Refactor permission seeding so new permissions are inserted even when roles already exist. Add this helper near `seedData`:

```go
func walletPermissions() []models.Permission {
	return []models.Permission{
		{Name: "wallet:read", Description: "View shop wallets"},
		{Name: "wallet:write", Description: "Manage shop wallets"},
		{Name: "withdrawal:read", Description: "View withdrawal requests"},
		{Name: "withdrawal:write", Description: "Approve/reject withdrawal requests"},
	}
}

func ensurePermissions(db *gorm.DB, permissions []models.Permission) []models.Permission {
	created := make([]models.Permission, 0, len(permissions))
	for _, permission := range permissions {
		var existing models.Permission
		err := db.Where("name = ?", permission.Name).First(&existing).Error
		if err == nil {
			created = append(created, existing)
			continue
		}
		permission.ID = uuid.New()
		db.Create(&permission)
		created = append(created, permission)
	}
	return created
}
```

At the top of `seedData`, before the existing early return, add:

```go
	ensurePermissions(db, walletPermissions())
```

Add wallet permissions to the initial `permissions` slice too, so fresh installs give `super_admin` all wallet permissions immediately:

```go
{Name: "wallet:read", Description: "View shop wallets"},
{Name: "wallet:write", Description: "Manage shop wallets"},
{Name: "withdrawal:read", Description: "View withdrawal requests"},
{Name: "withdrawal:write", Description: "Approve/reject withdrawal requests"},
```

Replace the initial permission creation loop with:

```go
	permissions = ensurePermissions(db, permissions)
```

- [ ] **Step 3: Format and compile routes**

Run: `gofmt -w internal/handlers/shop_wallet_handler.go cmd/server/main.go`

Run: `go test ./cmd/server ./internal/handlers/...`

Expected: PASS or no test files with successful compilation.

- [ ] **Step 4: Commit handlers and routes**

Run:

```bash
git add internal/handlers/shop_wallet_handler.go cmd/server/main.go
git commit -m "feat: add shop wallet APIs"
```

---

### Task 7: Full Verification And Cleanup

**Files:**
- Review all files changed in Tasks 1-6.

- [ ] **Step 1: Run full test suite**

Run: `go test ./...`

Expected: PASS.

- [ ] **Step 2: Run race detector**

Run: `go test -race ./...`

Expected: PASS.

- [ ] **Step 3: Build server**

Run: `go build -o server ./cmd/server`

Expected: PASS and produces `server` binary.

- [ ] **Step 4: Remove local build artifact**

Run: `rm -f server`

Expected: `server` binary removed.

- [ ] **Step 5: Inspect git diff**

Run: `git status --short`

Expected: only intentional feature files are modified or untracked. Existing unrelated `AGENTS.md` may remain untracked and must not be staged unless the user explicitly asks.

Run: `git diff --stat`

Expected: diff includes wallet models, repository, service, handler, cron, database migration, route wiring, and tests.

- [ ] **Step 6: Final commit if verification fixes were needed**

If Step 1-5 required fixes after the last feature commit, run:

```bash
git add internal cmd docs
git commit -m "test: verify shop wallet flow"
```

If no fixes were needed, do not create an empty commit.

---

## Acceptance Checklist

- [ ] `shop_wallets`, `shop_wallet_logs`, and `shop_withdrawal_requests` models migrate through `database.Migrate`.
- [ ] Wallets are lazily created on wallet read, order completion, or withdrawal creation.
- [ ] Order completion sets `orders.completed_at`.
- [ ] Order completion credits `TotalAmount - ShippingFee` into `pending_balance` once per order.
- [ ] Wallet release cron runs at 02:00 and releases pending funds 7 days after `completed_at`.
- [ ] Release uses the original `order_completed_pending` log amount.
- [ ] Withdrawal request validates `bank_name`, `account_number`, and `account_holder`.
- [ ] Withdrawal creation holds available balance immediately.
- [ ] Approval increments `withdrawn_balance` and does not subtract available balance a second time.
- [ ] Rejection returns held funds to `available_balance`.
- [ ] All wallet mutations write `shop_wallet_logs` with available, pending, and withdrawn before/after snapshots.
- [ ] Shop owner APIs live under `/api/v1/shop` and resolve shop by JWT `userID`.
- [ ] Admin APIs use `wallet:read`, `withdrawal:read`, and `withdrawal:write` permissions.
- [ ] `wallet:write` is seeded but unused by this iteration.
- [ ] `go test ./...`, `go test -race ./...`, and `go build -o server ./cmd/server` pass before completion is claimed.
