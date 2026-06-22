# Payment Module Refactor

## Overview

Tách Payment ra khỏi Order module thành module độc lập với model, repository, service riêng.
Thêm trường `type` để phân loại mục đích thanh toán, dùng strategy B (nullable FK per type)
cho các reference entity. Cho phép các module khác (top-up, membership...) dùng chung
PaymentService mà không cần tạo lại logic payment.

## Scope

- Tách Payment model thành file riêng, thêm `Type` + nullable FKs
- Tách Payment repository + service khỏi order module
- Sửa OrderService để gọi PaymentService thay vì tự xử lý payment
- Giữ nguyên handler layer — không public API payment riêng (option B)
- Các module sau này chỉ cần gọi `PaymentService.CreatePayment()` với `Type` khác

## Kiến trúc

```
┌──────────────────┐     ┌──────────────────┐     ┌──────────────────┐
│  OrderHandler    │     │  TopUpHandler    │     │  MembershipH     │
│  (giữ nguyên)    │     │  (sau này)       │     │  (sau này)       │
└──────┬───────────┘     └──────┬───────────┘     └──────┬───────────┘
       │                        │                        │
       ▼                        ▼                        ▼
┌──────────────────────────────────────────────────────────────┐
│  OrderService  │  TopUpService  │  MembershipService          │
│  (gọi Payment) │  (sau này)     │  (sau này)                  │
└──────┬───────────────────────────────────────────────────────┘
       │
       ▼
┌──────────────────┐
│  PaymentService  │
│  (chung)         │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│  PaymentRepo     │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│  payments table  │
└──────────────────┘
```

## 1. Payment Model

File: `internal/models/payment.go`

```go
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

    // Nullable FK (strategy B), uniqueIndex cho phép multiple NULL
    OrderID       *uuid.UUID     `gorm:"type:uuid;uniqueIndex" json:"order_id,omitempty"`
    Order         *Order         `gorm:"foreignKey:OrderID" json:"-"`

    DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
    CreatedAt     time.Time      `json:"created_at"`
    UpdatedAt     time.Time      `json:"updated_at"`
}
```

- `Type`: `"order"`, `"top_up"`, `"membership"`, ...
- `OrderID`: `*uuid.UUID` (nullable) — chỉ có giá trị khi `Type = "order"`
- `Method`: `"cod"`, `"bank_transfer"`, `"e_wallet"`, ...
- `Status`: `"pending"`, `"paid"`, `"failed"`, `"refunded"`

## 2. PaymentRepository

File: `internal/repositories/payment_repo.go`

```go
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

func NewPaymentRepository(db *gorm.DB) *PaymentRepository

// CRUD
func (r *PaymentRepository) FindByID(id uuid.UUID) (*models.Payment, error)
func (r *PaymentRepository) FindByOrderID(orderID uuid.UUID) (*models.Payment, error)
func (r *PaymentRepository) FindAll(filter PaymentFilter) ([]models.Payment, int64, error)
// - Khi Page < 1 → mặc định 1; Limit < 1 hoặc > 100 → mặc định 10
// - Filter chỉ áp dụng khi field != "" (zero value) để tránh lọc nhầm
func (r *PaymentRepository) Create(payment *models.Payment) error
func (r *PaymentRepository) Update(payment *models.Payment) error
```

- `FindByOrderID` giữ lại cho Order service khi auto-pay COD
- Xóa các hàm payment CRUD khỏi `order_repo.go`

## 3. PaymentService

File: `internal/services/payment_service.go`

```go
type CreatePaymentInput struct {
    Type    string
    Method  string
    Amount  float64
    OrderID *uuid.UUID  // nil when type != "order"
}

type PaymentService struct {
    paymentRepo *repositories.PaymentRepository
}

func NewPaymentService(paymentRepo *repositories.PaymentRepository) *PaymentService

// 1. Tạo payment record trong transaction
func (s *PaymentService) CreatePayment(tx *gorm.DB, input CreatePaymentInput) (*models.Payment, error)

// 2. Auto-pay COD khi order delivered (chỉ tác động nếu method == "cod" && status == "pending")
func (s *PaymentService) MarkAsPaid(tx *gorm.DB, orderID uuid.UUID) error

// 3. Cancel payment (failed nếu pending, refunded nếu đã paid)
func (s *PaymentService) CancelPayment(tx *gorm.DB, paymentID uuid.UUID) error

// 4. Helper — tìm payment theo order (dùng bởi OrderService)
func (s *PaymentService) FindByOrderID(tx *gorm.DB, orderID uuid.UUID) (*models.Payment, error)
```

### Validation trong `CreatePayment`

- `Type`: bắt buộc, phải thuộc `["order", "top_up", "membership"]`
- `OrderID`: **non-nil** khi `Type == "order"`, **nil** khi `Type != "order"` — nếu sai, trả về lỗi
- `Method`: bắt buộc, phải thuộc `["cod", "bank_transfer", "e_wallet"]`
- `Amount`: phải `> 0`
- Trả về `error` cụ thể để caller (OrderService, sau này là TopUpService) xử lý

## 4. OrderService — thay đổi

File: `internal/services/order_service.go`

- Inject `PaymentService` vào `OrderService`
- **Create order:** thay `tx.Create(payment)` bằng `paymentSvc.CreatePayment(tx, ...)`
- **Update status → delivered:** thay logic tự làm bằng `paymentSvc.MarkAsPaid(tx, order.ID)`
- **Cancel order:** `paymentSvc.FindByOrderID(tx, order.ID)` → `paymentSvc.CancelPayment(tx, payment.ID)`

## 5. Files thay đổi

| File | Action |
|------|--------|
| `internal/models/payment.go` | **Tạo mới** — Payment struct (từ order.go) |
| `internal/models/order.go` | **Sửa** — xóa Payment struct, giữ `Payment *Payment` |
| `internal/repositories/payment_repo.go` | **Tạo mới** — 5 hàm CRUD |
| `internal/repositories/order_repo.go` | **Sửa** — xóa 3 hàm payment |
| `internal/services/payment_service.go` | **Tạo mới** — 4 hàm business logic |
| `internal/services/order_service.go` | **Sửa** — gọi PaymentService |
| `cmd/server/main.go` | **Sửa** — wiring Payment module |

## 6. Không thay đổi

- Handlers (giữ nguyên)
- `internal/database/database.go` — `AutoMigrate(&models.Payment{})` đã có

## 7. Schema migration

Model cũ: `OrderID` NOT NULL, `Method` varchar(20), không có `Type`.
Model mới: `OrderID` nullable, `Method` varchar(50), có `Type` default `"order"`.

GORM AutoMigrate sẽ không tự xử lý được hoàn toàn. Cần thêm migration thủ công
vào `database.go` trước `AutoMigrate`:

```go
// Fix existing payments table
db.Exec(`ALTER TABLE payments ALTER COLUMN order_id DROP NOT NULL`)
db.Exec(`ALTER TABLE payments ADD COLUMN IF NOT EXISTS type varchar(50) NOT NULL DEFAULT 'order'`)
```

Sau đó `AutoMigrate` sẽ chạy và đồng bộ các index/tags còn lại. Dữ liệu cũ được
gán `type = 'order'` và không bị ảnh hưởng.
