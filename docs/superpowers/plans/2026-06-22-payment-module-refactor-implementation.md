# Payment Module Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Tách Payment ra khỏi Order module thành module độc lập, thêm trường `type`, dùng strategy B (nullable FK per type). Cho phép các module khác (top-up, membership...) dùng chung PaymentService.

**Architecture:** Giữ nguyên clean architecture: Handler → Service → Repository → Model. Payment có 3 layer riêng (model, repository, service), không public API riêng (các module gọi PaymentService trực tiếp).

**Tech Stack:** Go, Fiber v2, GORM, PostgreSQL, google/uuid

---

## File Structure

```
internal/
├── models/
│   ├── payment.go          # Tạo mới — Payment struct + status/type constants
│   └── order.go            # Sửa — xóa Payment struct, giữ Payment *Payment relation
├── repositories/
│   ├── payment_repo.go     # Tạo mới — 5 hàm CRUD
│   └── order_repo.go       # Sửa — xóa 3 hàm payment CRUD
├── services/
│   ├── payment_service.go  # Tạo mới — 4 hàm business logic
│   └── order_service.go    # Sửa — inject PaymentService, replace payment logic
├── database/
│   └── database.go         # Sửa — thêm migration SQL cho existing data
└── cmd/server/
    └── main.go             # Sửa — wiring PaymentRepo + PaymentService
```

---

### Task 1: Payment Model

**Files:**
- Create: `internal/models/payment.go`
- Modify: `internal/models/order.go` (xóa Payment struct)

- [ ] **Step 1: Tạo file internal/models/payment.go**

```go
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	PaymentStatusPending  = "pending"
	PaymentStatusPaid     = "paid"
	PaymentStatusFailed   = "failed"
	PaymentStatusRefunded = "refunded"
)

type Payment struct {
	ID            uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Type          string         `gorm:"type:varchar(50);not null;default:order" json:"type"`
	Status        string         `gorm:"type:varchar(20);default:pending" json:"status"`
	Method        string         `gorm:"type:varchar(50);not null" json:"method"`
	Amount        float64        `gorm:"type:decimal(12,2);not null" json:"amount"`
	TransactionID string         `gorm:"type:varchar(255)" json:"transaction_id,omitempty"`
	PaidAt        *time.Time     `json:"paid_at,omitempty"`

	OrderID       *uuid.UUID     `gorm:"type:uuid;uniqueIndex" json:"order_id,omitempty"`
	Order         *Order         `gorm:"foreignKey:OrderID" json:"-"`

	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}
```

- [ ] **Step 2: Xóa Payment struct khỏi internal/models/order.go**

Xóa block `type Payment struct { ... }` (dòng 54-66). Giữ nguyên `Payment *Payment` ở dòng 25 của Order struct.

```go
// Trong Order struct, giữ nguyên dòng:
	Payment         *Payment       `gorm:"foreignKey:OrderID" json:"payment,omitempty"`
```

- [ ] **Step 3: Build thử**

Run: `go build ./...`
Expected: Build thành công (không còn Payment struct duplicate)

- [ ] **Step 4: Commit**

```bash
git add internal/models/payment.go internal/models/order.go
git commit -m "feat(payment): create Payment model, remove from order.go"
```

---

### Task 2: Payment Repository

**Files:**
- Create: `internal/repositories/payment_repo.go`
- Modify: `internal/repositories/order_repo.go` (xóa 3 hàm payment)

- [ ] **Step 1: Tạo file internal/repositories/payment_repo.go**

```go
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
```

- [ ] **Step 2: Xóa 3 hàm payment khỏi internal/repositories/order_repo.go**

Xóa 3 hàm:
- `CreatePayment` (dòng 72-74)
- `UpdatePayment` (dòng 76-78)
- `FindPaymentByOrderID` (dòng 80-84)

Không sửa gì khác.

- [ ] **Step 3: Build thử**

Run: `go build ./...`
Expected: Build thành công

- [ ] **Step 4: Commit**

```bash
git add internal/repositories/payment_repo.go internal/repositories/order_repo.go
git commit -m "feat(payment): create PaymentRepository, remove from order_repo"
```

---

### Task 3: Payment Service

**Files:**
- Create: `internal/services/payment_service.go`

- [ ] **Step 1: Tạo file internal/services/payment_service.go**

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
	ErrInvalidPaymentType    = errors.New("invalid payment type")
	ErrInvalidPaymentMethod  = errors.New("invalid payment method")
	ErrInvalidPaymentAmount  = errors.New("payment amount must be greater than 0")
	ErrOrderIDRequired       = errors.New("order_id is required for order payment type")
	ErrOrderIDNotAllowed     = errors.New("order_id must be empty for non-order payment type")
	ErrPaymentNotFound       = errors.New("payment not found")
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
```

- [ ] **Step 2: Build thử**

Run: `go build ./...`
Expected: Build thành công

- [ ] **Step 3: Commit**

```bash
git add internal/services/payment_service.go
git commit -m "feat(payment): create PaymentService with CreatePayment, MarkAsPaid, CancelPayment"
```

---

### Task 4: Update OrderService

**Files:**
- Modify: `internal/services/order_service.go`

- [ ] **Step 1: Inject PaymentService vào OrderService**

Sửa struct và constructor:

```go
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
```

- [ ] **Step 2: Sửa Create — thay tx.Create(payment) bằng paymentSvc.CreatePayment**

Trong `Create`, xóa block `payment := &models.Payment{...}` và `tx.Create(payment)` (dòng 153-161), thay bằng:

```go
orderID := order.ID
payment, err := s.paymentSvc.CreatePayment(tx, CreatePaymentInput{
	Type:    "order",
	Method:  input.PaymentMethod,
	Amount:  order.TotalAmount,
	OrderID: &orderID,
})
if err != nil {
	return err
}
```

- [ ] **Step 3: Sửa UpdateStatus (delivered) — thay bằng paymentSvc.MarkAsPaid**

Trong `UpdateStatus`, xóa block `if input.Status == "delivered" { ... }` (dòng 257-270), thay bằng:

```go
if input.Status == "delivered" {
	if err := s.paymentSvc.MarkAsPaid(tx, order.ID); err != nil {
		return err
	}
}
```

- [ ] **Step 4: Sửa Cancel — thay bằng paymentSvc.FindByOrderID + paymentSvc.CancelPayment**

Trong `Cancel`, xóa block `var payment models.Payment ...` (dòng 309-318), thay bằng:

```go
payment, err := s.paymentSvc.FindByOrderID(tx, order.ID)
if err == nil {
	if err := s.paymentSvc.CancelPayment(tx, payment.ID); err != nil {
		return err
	}
}
```

- [ ] **Step 5: Build thử**

Run: `go build ./...`
Expected: Build thành công

- [ ] **Step 6: Commit**

```bash
git add internal/services/order_service.go
git commit -m "refactor(order): use PaymentService instead of direct payment logic"
```

---

### Task 5: Update database.go with migration SQL

**Files:**
- Modify: `internal/database/database.go`

- [ ] **Step 1: Thêm migration SQL trước AutoMigrate**

Trong hàm `Migrate`, thêm vào trước `db.AutoMigrate(...)`:

```go
// Migration: sửa payments table cho Payment model mới
db.Exec(`ALTER TABLE payments ALTER COLUMN order_id DROP NOT NULL`)
db.Exec(`ALTER TABLE payments ADD COLUMN IF NOT EXISTS type varchar(50) NOT NULL DEFAULT 'order'`)
```

```go
func Migrate(db *gorm.DB) {
	db.Exec(`ALTER TABLE payments ALTER COLUMN order_id DROP NOT NULL`)
	db.Exec(`ALTER TABLE payments ADD COLUMN IF NOT EXISTS type varchar(50) NOT NULL DEFAULT 'order'`)

	err := db.AutoMigrate(
		// ... giữ nguyên danh sách hiện tại
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
}
```

- [ ] **Step 2: Build thử**

Run: `go build ./...`
Expected: Build thành công

- [ ] **Step 3: Commit**

```bash
git add internal/database/database.go
git commit -m "fix(database): add migration for payments table schema changes"
```

---

### Task 6: Update main.go wiring

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Thêm PaymentRepo + PaymentService wiring**

Sau dòng `orderRepo := repositories.NewOrderRepository(db)`, thêm:

```go
paymentRepo := repositories.NewPaymentRepository(db)
```

Sau dòng `productService := services.NewProductService(...)`, sửa dòng `orderService` thành:

```go
paymentSvc := services.NewPaymentService(paymentRepo)
orderService := services.NewOrderService(orderRepo, paymentSvc, customerRepo, productRepo)
```

- [ ] **Step 2: Build thử**

Run: `go build ./...`
Expected: Build thành công

- [ ] **Step 3: Chạy go vet**

Run: `go vet ./...`
Expected: Không có lỗi

- [ ] **Step 4: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat(payment): wire PaymentRepository and PaymentService in main.go"
```

---

### Task 7: Verify với full build

- [ ] **Step 1: Build toàn bộ**

Run: `go build ./...`
Expected: Thành công, không lỗi

- [ ] **Step 2: Vet toàn bộ**

Run: `go vet ./...`
Expected: Không vấn đề

- [ ] **Step 3: Git status check**

Run: `git status`
Expected: Working tree clean (all changes committed)
