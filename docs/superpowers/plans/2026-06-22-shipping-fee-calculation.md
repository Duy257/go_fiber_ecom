# Shipping Fee Calculation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add lat/long to Shop and auto-calculate shipping fee based on distance using Haversine formula with global config.

**Architecture:** Pure Go Haversine implementation for distance calculation, global ShippingConfig model for fee parameters, new ShippingService for fee calculation logic, integrated into order creation flow with optional client override.

**Tech Stack:** Go, Fiber v2, GORM, PostgreSQL, go-playground/validator

---

## File Structure

```
internal/
├── models/
│   ├── shop.go                    ← modify: add Latitude, Longitude
│   ├── order.go                   ← modify: add ShippingLatitude, ShippingLongitude, ShippingDistanceKm
│   └── shipping_config.go         ← create: ShippingConfig model
├── repositories/
│   ├── shop_repo.go               ← modify: no changes needed (GORM handles new fields)
│   └── shipping_config_repo.go    ← create: CRUD for ShippingConfig
├── services/
│   ├── shop_service.go            ← modify: add Latitude/Longitude to inputs
│   ├── order_service.go           ← modify: inject ShippingService, auto-calculate fee
│   └── shipping_service.go        ← create: fee calculation logic
├── handlers/
│   ├── shop_handler.go            ← no changes needed (input binding via service)
│   └── shipping_handler.go        ← create: shipping estimate endpoint + admin config endpoints
├── utils/
│   ├── haversine.go               ← create: HaversineDistance function
│   └── math.go                    ← create: CeilToNearest function
cmd/server/
└── main.go                        ← modify: register routes, seed data, wire dependencies
internal/database/
└── database.go                    ← modify: add ShippingConfig to AutoMigrate
```

---

### Task 1: Utility Functions (haversine.go + math.go)

**Files:**
- Create: `internal/utils/haversine.go`
- Create: `internal/utils/math.go`

- [ ] **Step 1: Create haversine.go**

```go
// internal/utils/haversine.go
package utils

import "math"

const earthRadiusKm = 6371

// HaversineDistance calculates the distance in kilometers between two points
// using the Haversine formula. Coordinates are in decimal degrees.
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	dLat := toRadians(lat2 - lat1)
	dLon := toRadians(lon2 - lon1)

	lat1Rad := toRadians(lat1)
	lat2Rad := toRadians(lat2)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

func toRadians(deg float64) float64 {
	return deg * math.Pi / 180
}
```

- [ ] **Step 2: Create math.go**

```go
// internal/utils/math.go
package utils

import "math"

// CeilToNearest rounds value up to the nearest multiple of nearest.
// Example: CeilToNearest(25300, 1000) → 26000
func CeilToNearest(value, nearest float64) float64 {
	return math.Ceil(value/nearest) * nearest
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/utils/...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add internal/utils/haversine.go internal/utils/math.go
git commit -m "feat: add HaversineDistance and CeilToNearest utilities"
```

---

### Task 2: Model Changes (shop.go, order.go, shipping_config.go)

**Files:**
- Modify: `internal/models/shop.go`
- Modify: `internal/models/order.go`
- Create: `internal/models/shipping_config.go`

- [ ] **Step 1: Add Latitude/Longitude to Shop model**

In `internal/models/shop.go`, add two fields after `Address`:

```go
type Shop struct {
	ID          uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID      uuid.UUID      `gorm:"type:uuid;uniqueIndex;not null" json:"user_id"`
	User        User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Name        string         `gorm:"type:varchar(255);not null" json:"name"`
	Slug        string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"slug"`
	Description string         `gorm:"type:text" json:"description,omitempty"`
	Logo        string         `gorm:"type:varchar(500)" json:"logo,omitempty"`
	Address     string         `gorm:"type:varchar(500)" json:"address,omitempty"`
	Latitude    float64        `gorm:"type:decimal(10,7)" json:"latitude,omitempty"`
	Longitude   float64        `gorm:"type:decimal(10,7)" json:"longitude,omitempty"`
	Phone       string         `gorm:"type:varchar(20)" json:"phone,omitempty"`
	Status      string         `gorm:"type:varchar(20);default:active" json:"status"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}
```

- [ ] **Step 2: Add shipping coordinate fields to Order model**

In `internal/models/order.go`, add three fields after `ShippingAddress`:

```go
type Order struct {
	ID                 uuid.UUID              `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	CustomerID         uuid.UUID              `gorm:"type:uuid;index;not null" json:"customer_id"`
	Customer           Customer               `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	ShopID             uuid.UUID              `gorm:"type:uuid;index;not null" json:"shop_id"`
	Shop               Shop                   `gorm:"foreignKey:ShopID" json:"shop,omitempty"`
	OrderNumber        string                 `gorm:"type:varchar(50);uniqueIndex;not null" json:"order_number"`
	Status             string                 `gorm:"type:varchar(20);default:pending" json:"status"`
	SubTotal           float64                `gorm:"type:decimal(12,2);not null" json:"sub_total"`
	ShippingFee        float64                `gorm:"type:decimal(12,2);default:0" json:"shipping_fee"`
	TotalAmount        float64                `gorm:"type:decimal(12,2);not null" json:"total_amount"`
	ShippingAddress    map[string]interface{} `gorm:"type:jsonb;serializer:json;not null" json:"shipping_address"`
	ShippingLatitude   float64                `gorm:"type:decimal(10,7)" json:"shipping_latitude,omitempty"`
	ShippingLongitude  float64                `gorm:"type:decimal(10,7)" json:"shipping_longitude,omitempty"`
	ShippingDistanceKm float64                `gorm:"type:decimal(8,2)" json:"shipping_distance_km,omitempty"`
	Note               string                 `gorm:"type:text" json:"note,omitempty"`
	Items              []OrderItem            `gorm:"foreignKey:OrderID" json:"items,omitempty"`
	StatusHistory      []OrderStatusHistory   `gorm:"foreignKey:OrderID" json:"status_history,omitempty"`
	Payment            *Payment               `gorm:"foreignKey:OrderID" json:"payment,omitempty"`
	DeletedAt          gorm.DeletedAt         `gorm:"index" json:"-"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}
```

- [ ] **Step 3: Create ShippingConfig model**

```go
// internal/models/shipping_config.go
package models

import "time"

type ShippingConfig struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	BaseFee       float64   `gorm:"type:decimal(12,2);not null" json:"base_fee"`
	PerKmRate     float64   `gorm:"type:decimal(12,2);not null" json:"per_km_rate"`
	MaxDistanceKm float64   `gorm:"type:decimal(8,2);not null" json:"max_distance_km"`
	UpdatedAt     time.Time `json:"updated_at"`
}
```

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/models/...`
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add internal/models/shop.go internal/models/order.go internal/models/shipping_config.go
git commit -m "feat: add Latitude/Longitude to Shop, shipping coords to Order, ShippingConfig model"
```

---

### Task 3: ShippingConfig Repository

**Files:**
- Create: `internal/repositories/shipping_config_repo.go`

- [ ] **Step 1: Create repository**

```go
// internal/repositories/shipping_config_repo.go
package repositories

import (
	"go-fiber/internal/models"

	"gorm.io/gorm"
)

type ShippingConfigRepository struct {
	db *gorm.DB
}

func NewShippingConfigRepository(db *gorm.DB) *ShippingConfigRepository {
	return &ShippingConfigRepository{db: db}
}

func (r *ShippingConfigRepository) Get() (*models.ShippingConfig, error) {
	var config models.ShippingConfig
	err := r.db.First(&config).Error
	return &config, err
}

func (r *ShippingConfigRepository) Update(config *models.ShippingConfig) error {
	return r.db.Save(config).Error
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/repositories/...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/repositories/shipping_config_repo.go
git commit -m "feat: add ShippingConfigRepository"
```

---

### Task 4: Shipping Service

**Files:**
- Create: `internal/services/shipping_service.go`

- [ ] **Step 1: Create ShippingService**

```go
// internal/services/shipping_service.go
package services

import (
	"errors"
	"math"

	"go-fiber/internal/models"
	"go-fiber/internal/repositories"
	"go-fiber/internal/utils"

	"github.com/google/uuid"
)

type ShippingService struct {
	configRepo *repositories.ShippingConfigRepository
	shopRepo   *repositories.ShopRepository
}

func NewShippingService(
	configRepo *repositories.ShippingConfigRepository,
	shopRepo *repositories.ShopRepository,
) *ShippingService {
	return &ShippingService{
		configRepo: configRepo,
		shopRepo:   shopRepo,
	}
}

type ShippingEstimateInput struct {
	ShopID            string  `json:"shop_id" validate:"required"`
	ShippingLatitude  float64 `json:"shipping_latitude" validate:"required,min=-90,max=90"`
	ShippingLongitude float64 `json:"shipping_longitude" validate:"required,min=-180,max=180"`
}

type ShippingEstimateResult struct {
	DistanceKm    float64 `json:"distance_km"`
	BaseFee       float64 `json:"base_fee"`
	KmBasedFee    float64 `json:"km_based_fee"`
	TotalFee      float64 `json:"total_fee"`
	MaxDistanceKm float64 `json:"max_distance_km"`
}

func (s *ShippingService) Calculate(shopID uuid.UUID, shippingLat, shippingLong float64) (*ShippingEstimateResult, error) {
	config, err := s.configRepo.Get()
	if err != nil {
		return nil, errors.New("SHIPPING_CONFIG_NOT_FOUND")
	}

	shop, err := s.shopRepo.FindByID(shopID)
	if err != nil {
		return nil, errors.New("shop not found")
	}

	if shop.Latitude == 0 && shop.Longitude == 0 {
		return nil, errors.New("SHOP_LOCATION_NOT_SET")
	}

	distance := utils.HaversineDistance(shop.Latitude, shop.Longitude, shippingLat, shippingLong)
	// Round distance to 2 decimal places
	distance = math.Round(distance*100) / 100

	if distance > config.MaxDistanceKm {
		return nil, errors.New("OUTSIDE_DELIVERY_RANGE")
	}

	baseFee := config.BaseFee
	kmBasedFee := config.PerKmRate * distance
	rawFee := baseFee + kmBasedFee
	totalFee := utils.CeilToNearest(rawFee, 1000)

	return &ShippingEstimateResult{
		DistanceKm:    distance,
		BaseFee:       baseFee,
		KmBasedFee:    kmBasedFee,
		TotalFee:      totalFee,
		MaxDistanceKm: config.MaxDistanceKm,
	}, nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/services/...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/services/shipping_service.go
git commit -m "feat: add ShippingService with distance-based fee calculation"
```

---

### Task 5: Shipping Handler

**Files:**
- Create: `internal/handlers/shipping_handler.go`

- [ ] **Step 1: Create ShippingHandler**

```go
// internal/handlers/shipping_handler.go
package handlers

import (
	"go-fiber/internal/services"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ShippingHandler struct {
	service *services.ShippingService
}

func NewShippingHandler(service *services.ShippingService) *ShippingHandler {
	return &ShippingHandler{service: service}
}

func (h *ShippingHandler) Estimate(c *fiber.Ctx) error {
	var input services.ShippingEstimateInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	if errs := utils.Validate(input); errs != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", errs)
	}

	shopID, err := uuid.Parse(input.ShopID)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid shop_id")
	}

	result, err := h.service.Calculate(shopID, input.ShippingLatitude, input.ShippingLongitude)
	if err != nil {
		code := "VALIDATION_ERROR"
		status := 400
		switch err.Error() {
		case "SHOP_LOCATION_NOT_SET":
			code = "SHOP_LOCATION_NOT_SET"
		case "OUTSIDE_DELIVERY_RANGE":
			code = "OUTSIDE_DELIVERY_RANGE"
		case "SHIPPING_CONFIG_NOT_FOUND":
			code = "SHIPPING_CONFIG_NOT_FOUND"
			status = 500
		}
		return utils.Error(c, status, code, err.Error())
	}

	return utils.Success(c, result, "")
}

func (h *ShippingHandler) GetConfig(c *fiber.Ctx) error {
	result, err := h.service.GetConfig()
	if err != nil {
		return utils.Error(c, 500, "SHIPPING_CONFIG_NOT_FOUND", "Shipping config not found")
	}
	return utils.Success(c, result, "")
}

func (h *ShippingHandler) UpdateConfig(c *fiber.Ctx) error {
	var input services.UpdateShippingConfigInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	if errs := utils.Validate(input); errs != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", errs)
	}

	result, err := h.service.UpdateConfig(input)
	if err != nil {
		return utils.Error(c, 500, "INTERNAL_ERROR", err.Error())
	}

	return utils.Success(c, result, "Shipping config updated")
}
```

Note: There's a typo in Estimate — `s.service` should be `h.service`. Fix it when implementing.

Also note: `GetConfig` and `UpdateConfig` methods need to be added to `ShippingService`. Add these to `shipping_service.go`:

```go
func (s *ShippingService) GetConfig() (*models.ShippingConfig, error) {
	return s.configRepo.Get()
}

type UpdateShippingConfigInput struct {
	BaseFee       float64 `json:"base_fee" validate:"required,min=0"`
	PerKmRate     float64 `json:"per_km_rate" validate:"required,min=0"`
	MaxDistanceKm float64 `json:"max_distance_km" validate:"required,gt=0"`
}

func (s *ShippingService) UpdateConfig(input UpdateShippingConfigInput) (*models.ShippingConfig, error) {
	config, err := s.configRepo.Get()
	if err != nil {
		return nil, errors.New("SHIPPING_CONFIG_NOT_FOUND")
	}

	config.BaseFee = input.BaseFee
	config.PerKmRate = input.PerKmRate
	config.MaxDistanceKm = input.MaxDistanceKm

	if err := s.configRepo.Update(config); err != nil {
		return nil, err
	}
	return config, nil
}
```

- [ ] **Step 2: Fix the typo and verify compilation**

Run: `go build ./internal/handlers/...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/handlers/shipping_handler.go internal/services/shipping_service.go
git commit -m "feat: add ShippingHandler with estimate and admin config endpoints"
```

---

### Task 6: Update Shop Service (add Latitude/Longitude to inputs)

**Files:**
- Modify: `internal/services/shop_service.go`

- [ ] **Step 1: Update CreateShopInput**

Add Latitude and Longitude fields:

```go
type CreateShopInput struct {
	UserID      string  `json:"user_id" validate:"required"`
	Name        string  `json:"name" validate:"required"`
	Description string  `json:"description"`
	Logo        string  `json:"logo"`
	Address     string  `json:"address"`
	Latitude    float64 `json:"latitude" validate:"min=-90,max=90"`
	Longitude   float64 `json:"longitude" validate:"min=-180,max=180"`
	Phone       string  `json:"phone"`
}
```

- [ ] **Step 2: Update UpdateShopInput**

Add Latitude and Longitude as pointer fields:

```go
type UpdateShopInput struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Logo        *string  `json:"logo"`
	Address     *string  `json:"address"`
	Latitude    *float64 `json:"latitude" validate:"omitempty,min=-90,max=90"`
	Longitude   *float64 `json:"longitude" validate:"omitempty,min=-180,max=180"`
	Phone       *string  `json:"phone"`
	Status      *string  `json:"status"`
}
```

- [ ] **Step 3: Update Create method**

In the `Create` method, add Latitude and Longitude to the shop struct initialization:

```go
shop := &models.Shop{
	UserID:      userID,
	Name:        input.Name,
	Slug:        slug,
	Description: input.Description,
	Logo:        input.Logo,
	Address:     input.Address,
	Latitude:    input.Latitude,
	Longitude:   input.Longitude,
	Phone:       input.Phone,
	Status:      "active",
}
```

- [ ] **Step 4: Update Update method**

Add handling for Latitude and Longitude updates:

```go
if input.Latitude != nil {
	shop.Latitude = *input.Latitude
}
if input.Longitude != nil {
	shop.Longitude = *input.Longitude
}
```

- [ ] **Step 5: Verify compilation**

Run: `go build ./internal/services/...`
Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add internal/services/shop_service.go
git commit -m "feat: add Latitude/Longitude to shop create/update inputs"
```

---

### Task 7: Update Order Service (auto-calculate shipping fee)

**Files:**
- Modify: `internal/services/order_service.go`

- [ ] **Step 1: Add ShippingService dependency to OrderService**

```go
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
```

- [ ] **Step 2: Update CreateOrderInput**

Add ShippingLatitude, ShippingLongitude, and make ShippingFee a pointer:

```go
type CreateOrderInput struct {
	CustomerID         string                 `json:"customer_id" validate:"required"`
	ShopID             string                 `json:"shop_id" validate:"required"`
	Items              []CreateOrderItemInput `json:"items" validate:"required,min=1"`
	ShippingFee        *float64               `json:"shipping_fee"`
	ShippingAddress    map[string]interface{} `json:"shipping_address" validate:"required"`
	ShippingLatitude   float64                `json:"shipping_latitude" validate:"required,min=-90,max=90"`
	ShippingLongitude  float64                `json:"shipping_longitude" validate:"required,min=-180,max=180"`
	Note               string                 `json:"note"`
	PaymentMethod      string                 `json:"payment_method" validate:"required,oneof=cod bank_transfer e_wallet"`
}
```

- [ ] **Step 3: Update Create method — add shipping calculation**

After parsing shopID (line ~67 in current code), add shipping calculation:

```go
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

	// ... rest of items processing remains the same ...

	order.SubTotal = subTotal
	order.TotalAmount = subTotal + finalShippingFee

	// ... rest of transaction remains the same ...
}
```

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/services/...`
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add internal/services/order_service.go
git commit -m "feat: integrate shipping fee auto-calculation into order creation"
```

---

### Task 8: Update Database Migration and Seed Data

**Files:**
- Modify: `internal/database/database.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Add ShippingConfig to AutoMigrate**

In `internal/database/database.go`, add `&models.ShippingConfig{}` to the AutoMigrate call:

```go
err := db.AutoMigrate(
	&models.Role{},
	&models.Permission{},
	&models.User{},
	&models.Customer{},
	&models.Category{},
	&models.ProductCategory{},
	&models.Shop{},
	&models.Product{},
	&models.ProductVariant{},
	&models.ProductImage{},
	&models.Order{},
	&models.OrderItem{},
	&models.OrderStatusHistory{},
	&models.Payment{},
	&models.ShippingConfig{},
)
```

- [ ] **Step 2: Add shipping_config permissions to seed data**

In `cmd/server/main.go` `seedData()` function, add two permissions to the `permissions` slice:

```go
{Name: "shipping_config:read", Description: "View shipping config"},
{Name: "shipping_config:write", Description: "Update shipping config"},
```

- [ ] **Step 3: Add ShippingConfig seed data**

After the role/user seeding block (before `log.Println("Seed data created successfully")`), add:

```go
// Seed shipping config
var shippingConfigCount int64
db.Model(&models.ShippingConfig{}).Count(&shippingConfigCount)
if shippingConfigCount == 0 {
	db.Create(&models.ShippingConfig{
		BaseFee:       10000,
		PerKmRate:     3000,
		MaxDistanceKm: 30,
	})
}
```

- [ ] **Step 4: Verify compilation**

Run: `go build ./cmd/server/...`
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add internal/database/database.go cmd/server/main.go
git commit -m "feat: add ShippingConfig migration, permissions, and seed data"
```

---

### Task 9: Wire Dependencies and Register Routes

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Add ShippingConfigRepository**

After existing repository declarations (line ~55):

```go
shippingConfigRepo := repositories.NewShippingConfigRepository(db)
```

- [ ] **Step 2: Add ShippingService**

After existing service declarations (line ~67):

```go
shippingService := services.NewShippingService(shippingConfigRepo, shopRepo)
```

- [ ] **Step 3: Update OrderService constructor**

Add `shippingService` as the last argument:

```go
orderService := services.NewOrderService(orderRepo, paymentSvc, customerRepo, productRepo, shippingService)
```

- [ ] **Step 4: Add ShippingHandler**

After existing handler declarations (line ~79):

```go
shippingHandler := handlers.NewShippingHandler(shippingService)
```

- [ ] **Step 5: Register shipping routes**

After the shops route group (line ~140), add:

```go
// Public shipping estimate
shipping := api.Group("/shipping")
shipping.Post("/estimate", shippingHandler.Estimate)

// Admin shipping config
adminShipping := api.Group("/admin/shipping-config", middleware.JWTAuth(cfg))
adminShipping.Get("/", middleware.RequirePermission(userRepo, "shipping_config:read"), shippingHandler.GetConfig)
adminShipping.Put("/", middleware.RequirePermission(userRepo, "shipping_config:write"), shippingHandler.UpdateConfig)
```

- [ ] **Step 6: Verify compilation**

Run: `go build ./cmd/server/...`
Expected: no errors

- [ ] **Step 7: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: wire shipping dependencies and register routes"
```

---

### Task 10: Manual Verification

- [ ] **Step 1: Build the entire project**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 2: Run the server locally** (if DB available)

Run: `go run cmd/server/main.go`
Expected: server starts, AutoMigrate runs, seed data created

- [ ] **Step 3: Test shipping estimate endpoint**

```bash
curl -X POST http://localhost:3000/api/v1/shipping/estimate \
  -H "Content-Type: application/json" \
  -d '{"shop_id": "<shop-uuid>", "shipping_latitude": 10.7769, "shipping_longitude": 106.7009}'
```

Expected: JSON response with distance, base_fee, km_based_fee, total_fee, max_distance_km

- [ ] **Step 4: Test admin config endpoint**

```bash
# Login first to get token
curl -X PUT http://localhost:3000/api/v1/admin/shipping-config \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"base_fee": 15000, "per_km_rate": 4000, "max_distance_km": 50}'
```

Expected: updated config response

- [ ] **Step 5: Final commit with all changes**

```bash
git add -A
git status
git commit -m "feat: shipping fee calculation by distance - complete implementation"
```
