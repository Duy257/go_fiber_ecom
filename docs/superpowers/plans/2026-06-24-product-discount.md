# Product Discount Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add product-level percent and fixed amount discounts that appear in product responses and are snapshotted into orders at checkout.

**Architecture:** Discounts are stored directly on `Product`, calculated in the service layer, and exposed through product response DTOs. Order creation uses the same discount calculation helper and stores original price plus discount audit fields on each `OrderItem` while keeping `OrderItem.Price` as the final discounted unit price.

**Tech Stack:** Go 1.26.4, Fiber v2, GORM v1.31.1, PostgreSQL in production, `glebarez/sqlite` for tests, `go test ./...` for verification.

## Global Constraints

- Discount scope is product-level only; variants do not store independent discount fields.
- Discount types are `percent` and `fixed_amount` only.
- No discount time windows, campaigns, coupons, stacking rules, usage limits, customer-specific discounts, or discount history tables.
- Fixed discounts larger than the selected product or variant price are allowed and clamp the final unit price to `0`.
- Product responses must include raw discount fields and computed `discounted_price` plus `discount_amount` fields.
- Computed discount fields must be response DTO fields, not persisted model columns.
- Order creation must use discounted prices for subtotal, total amount, and payment amount.
- Order items must snapshot original price, discount type, discount value, and per-unit discount amount.
- Do not backfill existing order audit fields; historical order totals remain as stored.
- Follow existing error response conventions: HTTP `400`, error code `VALIDATION_ERROR` for invalid discount input.

---

## File Structure

- Modify `internal/models/product.go`: add product discount constants and product discount columns.
- Modify `internal/models/order.go`: add order item audit columns.
- Create `internal/services/product_discount.go`: shared discount validation and calculation helpers used by product and order services.
- Create `internal/services/product_discount_test.go`: unit tests for helper validation and calculation rules.
- Modify `internal/services/product_service.go`: add discount fields to inputs, add product response DTOs, validate discount inputs, and map models to computed responses.
- Create `internal/services/product_service_discount_test.go`: service tests for create, update, validation, and computed product/variant responses.
- Modify `internal/services/order_service.go`: apply product discounts during order creation and fill order item audit fields.
- Create `internal/services/order_discount_test.go`: checkout tests for product and variant discounts, clamp behavior, audit snapshots, and discounted payment amount.
- Modify `docs/api.md`: document product discount request and response fields.

---

### Task 1: Add Discount Model Fields

**Files:**
- Modify: `internal/models/product.go`
- Modify: `internal/models/order.go`
- Create: `internal/models/product_test.go`

**Interfaces:**
- Produces: `models.ProductDiscountTypePercent string`, `models.ProductDiscountTypeFixedAmount string`.
- Produces: `models.Product.DiscountType string`, `models.Product.DiscountValue float64`.
- Produces: `models.OrderItem.OriginalPrice float64`, `DiscountType string`, `DiscountValue float64`, `DiscountAmount float64`.

- [ ] **Step 1: Write failing model tests**

Create `internal/models/product_test.go`:

```go
package models

import "testing"

func TestProductDiscountTypeConstants(t *testing.T) {
	if ProductDiscountTypePercent != "percent" {
		t.Fatalf("ProductDiscountTypePercent = %q, want percent", ProductDiscountTypePercent)
	}

	if ProductDiscountTypeFixedAmount != "fixed_amount" {
		t.Fatalf("ProductDiscountTypeFixedAmount = %q, want fixed_amount", ProductDiscountTypeFixedAmount)
	}
}

func TestProductDiscountFields(t *testing.T) {
	product := Product{
		DiscountType:  ProductDiscountTypePercent,
		DiscountValue: 15,
	}

	if product.DiscountType != ProductDiscountTypePercent {
		t.Fatalf("DiscountType = %q, want %q", product.DiscountType, ProductDiscountTypePercent)
	}

	if product.DiscountValue != 15 {
		t.Fatalf("DiscountValue = %v, want 15", product.DiscountValue)
	}
}
```

- [ ] **Step 2: Run model tests and verify failure**

Run:

```bash
go test ./internal/models
```

Expected: FAIL because `ProductDiscountTypePercent`, `ProductDiscountTypeFixedAmount`, `Product.DiscountType`, and `Product.DiscountValue` do not exist yet.

- [ ] **Step 3: Add product discount constants and fields**

Modify `internal/models/product.go` after the imports:

```go
const (
	ProductDiscountTypePercent     = "percent"
	ProductDiscountTypeFixedAmount = "fixed_amount"
)
```

Modify `Product` in `internal/models/product.go` by adding these fields after `Price`:

```go
	DiscountType  string  `gorm:"type:varchar(20)" json:"discount_type,omitempty"`
	DiscountValue float64 `gorm:"type:decimal(12,2);default:0" json:"discount_value"`
```

- [ ] **Step 4: Add order item audit fields**

Modify `OrderItem` in `internal/models/order.go` by replacing the price/quantity/total block with:

```go
	OriginalPrice  float64 `gorm:"type:decimal(12,2);not null;default:0" json:"original_price"`
	Price          float64 `gorm:"type:decimal(12,2);not null" json:"price"`
	DiscountType   string  `gorm:"type:varchar(20)" json:"discount_type,omitempty"`
	DiscountValue  float64 `gorm:"type:decimal(12,2);default:0" json:"discount_value"`
	DiscountAmount float64 `gorm:"type:decimal(12,2);default:0" json:"discount_amount"`
	Quantity       int     `gorm:"not null" json:"quantity"`
	Total          float64 `gorm:"type:decimal(12,2);not null" json:"total"`
```

- [ ] **Step 5: Run model tests and verify pass**

Run:

```bash
go test ./internal/models
```

Expected: PASS.

- [ ] **Step 6: Commit Task 1**

Run:

```bash
git add internal/models/product.go internal/models/order.go internal/models/product_test.go
git commit -m "feat: add product discount model fields"
```

---

### Task 2: Add Shared Discount Calculation Helpers

**Files:**
- Create: `internal/services/product_discount.go`
- Create: `internal/services/product_discount_test.go`

**Interfaces:**
- Consumes: `models.ProductDiscountTypePercent`, `models.ProductDiscountTypeFixedAmount` from Task 1.
- Produces: `type DiscountCalculation struct { OriginalPrice float64; DiscountedPrice float64; DiscountAmount float64 }`.
- Produces: `func validateProductDiscount(discountType string, discountValue float64) error`.
- Produces: `func calculateProductDiscount(originalPrice float64, discountType string, discountValue float64) DiscountCalculation`.

- [ ] **Step 1: Write failing helper tests**

Create `internal/services/product_discount_test.go`:

```go
package services

import (
	"testing"

	"go-fiber/internal/models"
)

func TestValidateProductDiscountRejectsUnknownType(t *testing.T) {
	err := validateProductDiscount("bogus", 10)
	if err == nil {
		t.Fatal("validateProductDiscount returned nil, want error")
	}
}

func TestValidateProductDiscountRejectsPercentOver100(t *testing.T) {
	err := validateProductDiscount(models.ProductDiscountTypePercent, 101)
	if err == nil {
		t.Fatal("validateProductDiscount returned nil, want error")
	}
}

func TestValidateProductDiscountRejectsValueWithoutType(t *testing.T) {
	err := validateProductDiscount("", 10)
	if err == nil {
		t.Fatal("validateProductDiscount returned nil, want error")
	}
}

func TestCalculateProductDiscountPercent(t *testing.T) {
	result := calculateProductDiscount(200000, models.ProductDiscountTypePercent, 10)

	if result.OriginalPrice != 200000 {
		t.Fatalf("OriginalPrice = %v, want 200000", result.OriginalPrice)
	}
	if result.DiscountAmount != 20000 {
		t.Fatalf("DiscountAmount = %v, want 20000", result.DiscountAmount)
	}
	if result.DiscountedPrice != 180000 {
		t.Fatalf("DiscountedPrice = %v, want 180000", result.DiscountedPrice)
	}
}

func TestCalculateProductDiscountFixedAmountClampsToZero(t *testing.T) {
	result := calculateProductDiscount(50000, models.ProductDiscountTypeFixedAmount, 80000)

	if result.DiscountAmount != 50000 {
		t.Fatalf("DiscountAmount = %v, want 50000", result.DiscountAmount)
	}
	if result.DiscountedPrice != 0 {
		t.Fatalf("DiscountedPrice = %v, want 0", result.DiscountedPrice)
	}
}

func TestCalculateProductDiscountWithoutDiscount(t *testing.T) {
	result := calculateProductDiscount(120000, "", 0)

	if result.DiscountAmount != 0 {
		t.Fatalf("DiscountAmount = %v, want 0", result.DiscountAmount)
	}
	if result.DiscountedPrice != 120000 {
		t.Fatalf("DiscountedPrice = %v, want 120000", result.DiscountedPrice)
	}
}
```

- [ ] **Step 2: Run helper tests and verify failure**

Run:

```bash
go test ./internal/services -run 'TestValidateProductDiscount|TestCalculateProductDiscount'
```

Expected: FAIL because `validateProductDiscount`, `calculateProductDiscount`, and `DiscountCalculation` do not exist yet.

- [ ] **Step 3: Implement discount helpers**

Create `internal/services/product_discount.go`:

```go
package services

import (
	"errors"

	"go-fiber/internal/models"
)

type DiscountCalculation struct {
	OriginalPrice    float64
	DiscountedPrice  float64
	DiscountAmount   float64
}

func validateProductDiscount(discountType string, discountValue float64) error {
	if discountType == "" {
		if discountValue != 0 {
			return errors.New("discount_type is required when setting discount_value")
		}
		return nil
	}

	if discountValue < 0 {
		return errors.New("discount_value must be greater than or equal to 0")
	}

	switch discountType {
	case models.ProductDiscountTypePercent:
		if discountValue > 100 {
			return errors.New("discount_value must be between 0 and 100 for percent discount")
		}
	case models.ProductDiscountTypeFixedAmount:
		return nil
	default:
		return errors.New("invalid discount_type")
	}

	return nil
}

func calculateProductDiscount(originalPrice float64, discountType string, discountValue float64) DiscountCalculation {
	result := DiscountCalculation{
		OriginalPrice:   originalPrice,
		DiscountedPrice: originalPrice,
		DiscountAmount:  0,
	}

	if originalPrice <= 0 || discountType == "" || discountValue <= 0 {
		if originalPrice < 0 {
			result.OriginalPrice = 0
			result.DiscountedPrice = 0
		}
		return result
	}

	var rawDiscount float64
	switch discountType {
	case models.ProductDiscountTypePercent:
		rawDiscount = originalPrice * discountValue / 100
	case models.ProductDiscountTypeFixedAmount:
		rawDiscount = discountValue
	default:
		return result
	}

	if rawDiscount > originalPrice {
		rawDiscount = originalPrice
	}
	if rawDiscount < 0 {
		rawDiscount = 0
	}

	result.DiscountAmount = rawDiscount
	result.DiscountedPrice = originalPrice - rawDiscount
	return result
}
```

- [ ] **Step 4: Run helper tests and verify pass**

Run:

```bash
go test ./internal/services -run 'TestValidateProductDiscount|TestCalculateProductDiscount'
```

Expected: PASS.

- [ ] **Step 5: Commit Task 2**

Run:

```bash
git add internal/services/product_discount.go internal/services/product_discount_test.go
git commit -m "feat: add product discount calculation"
```

---

### Task 3: Return Product Discount DTOs From Product Service

**Files:**
- Modify: `internal/services/product_service.go`
- Create: `internal/services/product_service_discount_test.go`
- Modify: `internal/handlers/product_handler.go` only if compilation shows handler type assumptions after service signatures change.

**Interfaces:**
- Consumes: `validateProductDiscount` and `calculateProductDiscount` from Task 2.
- Produces: `type ProductResponse` with raw product fields, computed `DiscountedPrice`, `DiscountAmount`, and variant response data.
- Produces: `type ProductVariantResponse` with raw variant fields, computed `DiscountedPrice`, and computed `DiscountAmount`.
- Changes: `ProductService.Create` returns `(*ProductResponse, error)`.
- Changes: `ProductService.GetByID` returns `(*ProductResponse, error)`.
- Changes: `ProductService.GetAll` returns `([]ProductResponse, int64, error)`.
- Changes: `ProductService.Update` returns `(*ProductResponse, error)`.

- [ ] **Step 1: Write failing product service discount tests**

Create `internal/services/product_service_discount_test.go`:

```go
package services

import (
	"os"
	"testing"

	"go-fiber/internal/models"
	"go-fiber/internal/repositories"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func newProductDiscountServiceTestDB(t *testing.T) (*gorm.DB, *ProductService) {
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

	if err := db.AutoMigrate(
		&models.Shop{},
		&models.Category{},
		&models.ProductCategory{},
		&models.Product{},
		&models.ProductVariant{},
		&models.ProductImage{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	productRepo := repositories.NewProductRepository(db)
	shopRepo := repositories.NewShopRepository(db)
	categoryRepo := repositories.NewCategoryRepository(db)
	return db, NewProductService(productRepo, shopRepo, categoryRepo)
}

func createProductDiscountTestShop(t *testing.T, db *gorm.DB) models.Shop {
	t.Helper()

	shop := models.Shop{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Name:   "Discount Shop",
		Slug:   "discount-shop-" + uuid.NewString()[:8],
		Status: "active",
	}
	if err := db.Create(&shop).Error; err != nil {
		t.Fatalf("create shop: %v", err)
	}
	return shop
}

func TestCreateProductWithPercentDiscountReturnsComputedPrices(t *testing.T) {
	db, svc := newProductDiscountServiceTestDB(t)
	shop := createProductDiscountTestShop(t, db)

	product, err := svc.Create(CreateProductInput{
		ShopID:        shop.ID.String(),
		Name:          "Percent Discount Product",
		Price:         200000,
		DiscountType:  models.ProductDiscountTypePercent,
		DiscountValue: 10,
		Variants: []CreateVariantInput{{
			Name:  "Large",
			Price: 250000,
			Stock: 3,
		}},
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if product.DiscountedPrice != 180000 {
		t.Fatalf("DiscountedPrice = %v, want 180000", product.DiscountedPrice)
	}
	if product.DiscountAmount != 20000 {
		t.Fatalf("DiscountAmount = %v, want 20000", product.DiscountAmount)
	}
	if len(product.Variants) != 1 {
		t.Fatalf("len(Variants) = %d, want 1", len(product.Variants))
	}
	if product.Variants[0].DiscountedPrice != 225000 {
		t.Fatalf("variant DiscountedPrice = %v, want 225000", product.Variants[0].DiscountedPrice)
	}
	if product.Variants[0].DiscountAmount != 25000 {
		t.Fatalf("variant DiscountAmount = %v, want 25000", product.Variants[0].DiscountAmount)
	}
}

func TestCreateProductRejectsInvalidDiscount(t *testing.T) {
	db, svc := newProductDiscountServiceTestDB(t)
	shop := createProductDiscountTestShop(t, db)

	_, err := svc.Create(CreateProductInput{
		ShopID:        shop.ID.String(),
		Name:          "Invalid Discount Product",
		Price:         200000,
		DiscountType:  models.ProductDiscountTypePercent,
		DiscountValue: 101,
		Variants: []CreateVariantInput{{
			Name:  "Default",
			Price: 200000,
			Stock: 1,
		}},
	})
	if err == nil {
		t.Fatal("Create returned nil error, want validation error")
	}
}

func TestUpdateProductRejectsDiscountValueWithoutExistingType(t *testing.T) {
	db, svc := newProductDiscountServiceTestDB(t)
	shop := createProductDiscountTestShop(t, db)

	product, err := svc.Create(CreateProductInput{
		ShopID: shop.ID.String(),
		Name:   "No Discount Product",
		Price:  100000,
		Variants: []CreateVariantInput{{
			Name:  "Default",
			Price: 100000,
			Stock: 1,
		}},
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	value := 10.0
	_, err = svc.Update(product.ID, UpdateProductInput{DiscountValue: &value})
	if err == nil {
		t.Fatal("Update returned nil error, want validation error")
	}
}

func TestUpdateProductClearsDiscountWithExplicitZero(t *testing.T) {
	db, svc := newProductDiscountServiceTestDB(t)
	shop := createProductDiscountTestShop(t, db)

	product, err := svc.Create(CreateProductInput{
		ShopID:        shop.ID.String(),
		Name:          "Clear Discount Product",
		Price:         100000,
		DiscountType:  models.ProductDiscountTypeFixedAmount,
		DiscountValue: 25000,
		Variants: []CreateVariantInput{{
			Name:  "Default",
			Price: 100000,
			Stock: 1,
		}},
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	emptyType := ""
	zero := 0.0
	updated, err := svc.Update(product.ID, UpdateProductInput{
		DiscountType:  &emptyType,
		DiscountValue: &zero,
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	if updated.DiscountType != "" {
		t.Fatalf("DiscountType = %q, want empty", updated.DiscountType)
	}
	if updated.DiscountValue != 0 {
		t.Fatalf("DiscountValue = %v, want 0", updated.DiscountValue)
	}
	if updated.DiscountedPrice != 100000 {
		t.Fatalf("DiscountedPrice = %v, want 100000", updated.DiscountedPrice)
	}
	if updated.DiscountAmount != 0 {
		t.Fatalf("DiscountAmount = %v, want 0", updated.DiscountAmount)
	}
}
```

- [ ] **Step 2: Run product service discount tests and verify failure**

Run:

```bash
go test ./internal/services -run 'TestCreateProductWithPercentDiscountReturnsComputedPrices|TestCreateProductRejectsInvalidDiscount|TestUpdateProductRejectsDiscountValueWithoutExistingType|TestUpdateProductClearsDiscountWithExplicitZero'
```

Expected: FAIL because product input discount fields and product response DTO fields do not exist yet.

- [ ] **Step 3: Add discount fields to product service inputs**

Modify `CreateProductInput` in `internal/services/product_service.go`:

```go
type CreateProductInput struct {
	ShopID        string               `json:"shop_id" validate:"required"`
	Name          string               `json:"name" validate:"required"`
	Description   string               `json:"description"`
	Price         float64              `json:"price" validate:"required,gt=0"`
	DiscountType  string               `json:"discount_type"`
	DiscountValue float64              `json:"discount_value"`
	CategoryIDs   []string             `json:"category_ids"`
	Variants      []CreateVariantInput `json:"variants" validate:"required,min=1"`
	Images        []CreateImageInput   `json:"images"`
}
```

Modify `UpdateProductInput` in `internal/services/product_service.go`:

```go
type UpdateProductInput struct {
	Name          *string              `json:"name"`
	Description   *string              `json:"description"`
	Price         *float64             `json:"price"`
	DiscountType  *string              `json:"discount_type"`
	DiscountValue *float64             `json:"discount_value"`
	Status        *string              `json:"status"`
	CategoryIDs   []string             `json:"category_ids"`
	Variants      []CreateVariantInput `json:"variants"`
	Images        []CreateImageInput   `json:"images"`
}
```

- [ ] **Step 4: Add product response DTOs and mapper**

Add these types and methods in `internal/services/product_service.go` after `CreateImageInput`:

```go
type ProductResponse struct {
	ID              uuid.UUID                `json:"id"`
	ShopID          uuid.UUID                `json:"shop_id"`
	Shop            models.Shop              `json:"shop,omitempty"`
	Name            string                   `json:"name"`
	Slug            string                   `json:"slug"`
	Description     string                   `json:"description,omitempty"`
	Images          []models.ProductImage    `json:"images,omitempty"`
	Variants        []ProductVariantResponse `json:"variants,omitempty"`
	Categories      []models.Category        `json:"categories,omitempty"`
	Price           float64                  `json:"price"`
	DiscountType    string                   `json:"discount_type,omitempty"`
	DiscountValue   float64                  `json:"discount_value"`
	DiscountedPrice float64                  `json:"discounted_price"`
	DiscountAmount  float64                  `json:"discount_amount"`
	Status          string                   `json:"status"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
}

type ProductVariantResponse struct {
	ID              uuid.UUID              `json:"id"`
	ProductID       uuid.UUID              `json:"product_id"`
	Name            string                 `json:"name"`
	SKU             *string                `json:"sku,omitempty"`
	Price           float64                `json:"price"`
	DiscountedPrice float64                `json:"discounted_price"`
	DiscountAmount  float64                `json:"discount_amount"`
	Stock           int                    `json:"stock"`
	Attributes      map[string]interface{} `json:"attributes,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

func productToResponse(product *models.Product) *ProductResponse {
	calculation := calculateProductDiscount(product.Price, product.DiscountType, product.DiscountValue)
	response := &ProductResponse{
		ID:              product.ID,
		ShopID:          product.ShopID,
		Shop:            product.Shop,
		Name:            product.Name,
		Slug:            product.Slug,
		Description:     product.Description,
		Images:          product.Images,
		Categories:      product.Categories,
		Price:           product.Price,
		DiscountType:    product.DiscountType,
		DiscountValue:   product.DiscountValue,
		DiscountedPrice: calculation.DiscountedPrice,
		DiscountAmount:  calculation.DiscountAmount,
		Status:          product.Status,
		CreatedAt:       product.CreatedAt,
		UpdatedAt:       product.UpdatedAt,
	}

	for _, variant := range product.Variants {
		variantCalculation := calculateProductDiscount(variant.Price, product.DiscountType, product.DiscountValue)
		response.Variants = append(response.Variants, ProductVariantResponse{
			ID:              variant.ID,
			ProductID:       variant.ProductID,
			Name:            variant.Name,
			SKU:             variant.SKU,
			Price:           variant.Price,
			DiscountedPrice: variantCalculation.DiscountedPrice,
			DiscountAmount:  variantCalculation.DiscountAmount,
			Stock:           variant.Stock,
			Attributes:      variant.Attributes,
			CreatedAt:       variant.CreatedAt,
			UpdatedAt:       variant.UpdatedAt,
		})
	}

	return response
}

func productsToResponse(products []models.Product) []ProductResponse {
	responses := make([]ProductResponse, 0, len(products))
	for i := range products {
		responses = append(responses, *productToResponse(&products[i]))
	}
	return responses
}
```

Add `time` to the import block in `internal/services/product_service.go`.

- [ ] **Step 5: Validate discount on create and return DTOs**

In `ProductService.Create`, after slug duplicate checking and before constructing `product`, add:

```go
	if err := validateProductDiscount(input.DiscountType, input.DiscountValue); err != nil {
		return nil, err
	}
```

When constructing `models.Product`, include:

```go
		DiscountType:  input.DiscountType,
		DiscountValue: input.DiscountValue,
```

Change `ProductService.Create` signature to:

```go
func (s *ProductService) Create(input CreateProductInput) (*ProductResponse, error)
```

Replace the final return in `Create` with:

```go
	created, err := s.repo.FindByID(product.ID)
	if err != nil {
		return nil, err
	}
	return productToResponse(created), nil
```

- [ ] **Step 6: Return DTOs from read and list methods**

Change `GetByID` signature to:

```go
func (s *ProductService) GetByID(id uuid.UUID) (*ProductResponse, error)
```

Replace its success return with:

```go
	return productToResponse(product), nil
```

Change `GetAll` signature to:

```go
func (s *ProductService) GetAll(shopID, categoryID *string, page, limit int) ([]ProductResponse, int64, error)
```

Replace the final repository return in `GetAll` with:

```go
	products, total, err := s.repo.FindAll(sid, cid, page, limit)
	if err != nil {
		return nil, 0, err
	}
	return productsToResponse(products), total, nil
```

- [ ] **Step 7: Validate discount on update and return DTOs**

Change `Update` signature to:

```go
func (s *ProductService) Update(id uuid.UUID, input UpdateProductInput) (*ProductResponse, error)
```

After the existing `Price` update block and before the `Status` update block, add:

```go
	discountType := product.DiscountType
	discountValue := product.DiscountValue
	if input.DiscountType != nil {
		discountType = *input.DiscountType
	}
	if input.DiscountValue != nil {
		if product.DiscountType == "" && input.DiscountType == nil {
			return nil, errors.New("discount_type is required when setting discount_value")
		}
		discountValue = *input.DiscountValue
	}
	if err := validateProductDiscount(discountType, discountValue); err != nil {
		return nil, err
	}
	product.DiscountType = discountType
	product.DiscountValue = discountValue
```

Replace the final return in `Update` with:

```go
	updated, err := s.repo.FindByID(product.ID)
	if err != nil {
		return nil, err
	}
	return productToResponse(updated), nil
```

- [ ] **Step 8: Run product service discount tests and verify pass**

Run:

```bash
go test ./internal/services -run 'TestCreateProductWithPercentDiscountReturnsComputedPrices|TestCreateProductRejectsInvalidDiscount|TestUpdateProductRejectsDiscountValueWithoutExistingType|TestUpdateProductClearsDiscountWithExplicitZero'
```

Expected: PASS.

- [ ] **Step 9: Run product handler compilation through package tests**

Run:

```bash
go test ./internal/handlers ./internal/services
```

Expected: PASS. If `internal/handlers/product_handler.go` fails to compile because of service return types, keep handler logic unchanged and adjust only local variable types inferred through `:=`; no response behavior change is needed because `utils.Success` accepts any data type.

- [ ] **Step 10: Commit Task 3**

Run:

```bash
git add internal/services/product_service.go internal/services/product_service_discount_test.go internal/handlers/product_handler.go
git commit -m "feat: expose product discount responses"
```

---

### Task 4: Apply Discounts During Order Creation

**Files:**
- Modify: `internal/services/order_service.go`
- Create: `internal/services/order_discount_test.go`

**Interfaces:**
- Consumes: `calculateProductDiscount` from Task 2.
- Consumes: `models.Product.DiscountType` and `models.Product.DiscountValue` from Task 1.
- Produces: discounted `OrderItem.Price`, populated `OriginalPrice`, `DiscountType`, `DiscountValue`, and `DiscountAmount` for new order items.

- [ ] **Step 1: Write failing order discount tests**

Create `internal/services/order_discount_test.go`:

```go
package services

import (
	"os"
	"testing"

	"go-fiber/internal/models"
	"go-fiber/internal/repositories"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type orderDiscountFixture struct {
	db       *gorm.DB
	service  *OrderService
	customer models.Customer
	shop     models.Shop
}

func newOrderDiscountFixture(t *testing.T) orderDiscountFixture {
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

	if err := db.AutoMigrate(
		&models.Customer{},
		&models.Shop{},
		&models.Product{},
		&models.ProductVariant{},
		&models.ProductImage{},
		&models.Order{},
		&models.OrderItem{},
		&models.OrderStatusHistory{},
		&models.Payment{},
		&models.ShippingConfig{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	email := "discount-customer-" + uuid.NewString()[:8] + "@example.com"
	phone := "09" + uuid.NewString()[:8]
	customer := models.Customer{
		ID:          uuid.New(),
		Email:       &email,
		PhoneNumber: &phone,
		Password:    "hashed-password",
		Name:        "Discount Customer",
		Status:      "active",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatalf("create customer: %v", err)
	}

	shop := models.Shop{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Name:      "Discount Checkout Shop",
		Slug:      "discount-checkout-shop-" + uuid.NewString()[:8],
		Latitude:  10,
		Longitude: 20,
		Status:    "active",
	}
	if err := db.Create(&shop).Error; err != nil {
		t.Fatalf("create shop: %v", err)
	}

	config := models.ShippingConfig{BaseFee: 10000, PerKmRate: 0, MaxDistanceKm: 100}
	if err := db.Create(&config).Error; err != nil {
		t.Fatalf("create shipping config: %v", err)
	}

	orderRepo := repositories.NewOrderRepository(db)
	paymentRepo := repositories.NewPaymentRepository(db)
	paymentSvc := NewPaymentService(paymentRepo)
	customerRepo := repositories.NewCustomerRepository(db)
	productRepo := repositories.NewProductRepository(db)
	shopRepo := repositories.NewShopRepository(db)
	shippingConfigRepo := repositories.NewShippingConfigRepository(db)
	shippingSvc := NewShippingService(shippingConfigRepo, shopRepo)

	return orderDiscountFixture{
		db:       db,
		service:  NewOrderService(orderRepo, paymentSvc, customerRepo, productRepo, shippingSvc),
		customer: customer,
		shop:     shop,
	}
}

func createOrderDiscountProduct(t *testing.T, db *gorm.DB, shopID uuid.UUID, price float64, discountType string, discountValue float64, variantPrice float64) models.Product {
	t.Helper()

	product := models.Product{
		ID:            uuid.New(),
		ShopID:        shopID,
		Name:          "Discount Checkout Product " + uuid.NewString()[:8],
		Slug:          "discount-checkout-product-" + uuid.NewString()[:8],
		Price:         price,
		DiscountType:  discountType,
		DiscountValue: discountValue,
		Status:        "active",
		Variants: []models.ProductVariant{{
			ID:    uuid.New(),
			Name:  "Variant",
			Price: variantPrice,
			Stock: 5,
		}},
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}
	return product
}

func createDiscountOrderInput(fixture orderDiscountFixture, product models.Product, variantID string, quantity int) CreateOrderInput {
	return CreateOrderInput{
		CustomerID:        fixture.customer.ID.String(),
		ShopID:            fixture.shop.ID.String(),
		Items:             []CreateOrderItemInput{{ProductID: product.ID.String(), VariantID: variantID, Quantity: quantity}},
		ShippingAddress:   map[string]interface{}{"address": "Test address"},
		ShippingLatitude:  10,
		ShippingLongitude: 20,
		PaymentMethod:     "cod",
	}
}

func TestCreateOrderAppliesProductPercentDiscount(t *testing.T) {
	fixture := newOrderDiscountFixture(t)
	product := createOrderDiscountProduct(t, fixture.db, fixture.shop.ID, 200000, models.ProductDiscountTypePercent, 10, 250000)

	order, err := fixture.service.Create(createDiscountOrderInput(fixture, product, "", 2))
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if len(order.Items) != 1 {
		t.Fatalf("len(order.Items) = %d, want 1", len(order.Items))
	}
	item := order.Items[0]
	if item.OriginalPrice != 200000 {
		t.Fatalf("OriginalPrice = %v, want 200000", item.OriginalPrice)
	}
	if item.Price != 180000 {
		t.Fatalf("Price = %v, want 180000", item.Price)
	}
	if item.DiscountType != models.ProductDiscountTypePercent {
		t.Fatalf("DiscountType = %q, want %q", item.DiscountType, models.ProductDiscountTypePercent)
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
	if order.TotalAmount != 370000 {
		t.Fatalf("TotalAmount = %v, want 370000", order.TotalAmount)
	}

	var payment models.Payment
	if err := fixture.db.Where("order_id = ?", order.ID).First(&payment).Error; err != nil {
		t.Fatalf("find payment: %v", err)
	}
	if payment.Amount != 370000 {
		t.Fatalf("payment Amount = %v, want 370000", payment.Amount)
	}
}

func TestCreateOrderAppliesFixedDiscountToVariant(t *testing.T) {
	fixture := newOrderDiscountFixture(t)
	product := createOrderDiscountProduct(t, fixture.db, fixture.shop.ID, 200000, models.ProductDiscountTypeFixedAmount, 50000, 250000)
	variantID := product.Variants[0].ID.String()

	order, err := fixture.service.Create(createDiscountOrderInput(fixture, product, variantID, 1))
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	item := order.Items[0]
	if item.OriginalPrice != 250000 {
		t.Fatalf("OriginalPrice = %v, want 250000", item.OriginalPrice)
	}
	if item.Price != 200000 {
		t.Fatalf("Price = %v, want 200000", item.Price)
	}
	if item.DiscountAmount != 50000 {
		t.Fatalf("DiscountAmount = %v, want 50000", item.DiscountAmount)
	}
}

func TestCreateOrderClampsFixedDiscountToZero(t *testing.T) {
	fixture := newOrderDiscountFixture(t)
	product := createOrderDiscountProduct(t, fixture.db, fixture.shop.ID, 30000, models.ProductDiscountTypeFixedAmount, 50000, 30000)

	order, err := fixture.service.Create(createDiscountOrderInput(fixture, product, "", 1))
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	item := order.Items[0]
	if item.Price != 0 {
		t.Fatalf("Price = %v, want 0", item.Price)
	}
	if item.DiscountAmount != 30000 {
		t.Fatalf("DiscountAmount = %v, want 30000", item.DiscountAmount)
	}
	if order.SubTotal != 0 {
		t.Fatalf("SubTotal = %v, want 0", order.SubTotal)
	}
	if order.TotalAmount != 10000 {
		t.Fatalf("TotalAmount = %v, want 10000", order.TotalAmount)
	}
}

func TestCreateOrderWithoutDiscountStoresOriginalPrice(t *testing.T) {
	fixture := newOrderDiscountFixture(t)
	product := createOrderDiscountProduct(t, fixture.db, fixture.shop.ID, 120000, "", 0, 120000)

	order, err := fixture.service.Create(createDiscountOrderInput(fixture, product, "", 1))
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	item := order.Items[0]
	if item.OriginalPrice != 120000 {
		t.Fatalf("OriginalPrice = %v, want 120000", item.OriginalPrice)
	}
	if item.Price != 120000 {
		t.Fatalf("Price = %v, want 120000", item.Price)
	}
	if item.DiscountAmount != 0 {
		t.Fatalf("DiscountAmount = %v, want 0", item.DiscountAmount)
	}
}
```

- [ ] **Step 2: Run order discount tests and verify failure**

Run:

```bash
go test ./internal/services -run 'TestCreateOrderAppliesProductPercentDiscount|TestCreateOrderAppliesFixedDiscountToVariant|TestCreateOrderClampsFixedDiscountToZero|TestCreateOrderWithoutDiscountStoresOriginalPrice'
```

Expected: FAIL because `OrderService.Create` does not populate discount audit fields or discounted item prices yet.

- [ ] **Step 3: Apply discount calculation inside order creation**

Modify the item price block in `OrderService.Create` in `internal/services/order_service.go`.

Replace this logic:

```go
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
```

With this logic:

```go
		originalPrice := product.Price
		if item.VariantID != "" {
			variantID, err := uuid.Parse(item.VariantID)
			if err != nil {
				return nil, errors.New("invalid variant_id")
			}

			var variant *models.ProductVariant
			for i := range product.Variants {
				if product.Variants[i].ID == variantID {
					variant = &product.Variants[i]
					break
				}
			}
			if variant == nil {
				return nil, errors.New("variant not found")
			}

			orderItem.VariantID = &variantID
			orderItem.VariantName = variant.Name
			originalPrice = variant.Price
		}

		discount := calculateProductDiscount(originalPrice, product.DiscountType, product.DiscountValue)
		orderItem.OriginalPrice = discount.OriginalPrice
		orderItem.Price = discount.DiscountedPrice
		orderItem.DiscountType = product.DiscountType
		orderItem.DiscountValue = product.DiscountValue
		orderItem.DiscountAmount = discount.DiscountAmount
		orderItem.Total = orderItem.Price * float64(item.Quantity)
```

- [ ] **Step 4: Run order discount tests and verify pass**

Run:

```bash
go test ./internal/services -run 'TestCreateOrderAppliesProductPercentDiscount|TestCreateOrderAppliesFixedDiscountToVariant|TestCreateOrderClampsFixedDiscountToZero|TestCreateOrderWithoutDiscountStoresOriginalPrice'
```

Expected: PASS.

- [ ] **Step 5: Run service package tests**

Run:

```bash
go test ./internal/services
```

Expected: PASS.

- [ ] **Step 6: Commit Task 4**

Run:

```bash
git add internal/services/order_service.go internal/services/order_discount_test.go
git commit -m "feat: apply product discounts to orders"
```

---

### Task 5: Document API Fields And Verify Full Test Suite

**Files:**
- Modify: `docs/api.md`

**Interfaces:**
- Consumes: product create/update discount fields from Task 3.
- Consumes: product response computed fields from Task 3.
- Consumes: order item audit fields from Task 4.

- [ ] **Step 1: Update API documentation for product discount payloads**

Add this section to `docs/api.md` before `## Error Response Format`:

````markdown
---

## Product Discounts

Product discounts are product-level and apply to the product base price and every variant price.

### Admin product create/update discount fields

```json
{
  "discount_type": "percent",
  "discount_value": 10
}
```

Valid `discount_type` values:

- `percent`: `discount_value` must be between `0` and `100`.
- `fixed_amount`: `discount_value` must be greater than or equal to `0`.
- Empty `discount_type` with `discount_value: 0` means no discount.

Fixed amount discounts larger than a product or variant price are allowed. The final price is clamped to `0`.

### Product response discount fields

```json
{
  "price": 200000,
  "discount_type": "percent",
  "discount_value": 10,
  "discounted_price": 180000,
  "discount_amount": 20000,
  "variants": [
    {
      "price": 250000,
      "discounted_price": 225000,
      "discount_amount": 25000
    }
  ]
}
```

### Order item discount audit fields

New order items store:

- `original_price`: selected product or variant unit price before discount.
- `price`: final unit price after discount.
- `discount_type`: discount type snapshot at checkout.
- `discount_value`: discount value snapshot at checkout.
- `discount_amount`: per-unit discount amount applied at checkout.
````

- [ ] **Step 2: Run formatting**

Run:

```bash
gofmt -w internal/models/product.go internal/models/order.go internal/services/product_discount.go internal/services/product_discount_test.go internal/services/product_service.go internal/services/product_service_discount_test.go internal/services/order_service.go internal/services/order_discount_test.go
```

Expected: command exits with code `0`.

- [ ] **Step 3: Run full tests**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 4: Inspect final diff**

Run:

```bash
git diff --stat
git diff -- internal/models/product.go internal/models/order.go internal/services/product_discount.go internal/services/product_service.go internal/services/order_service.go docs/api.md
```

Expected: diff shows only product discount model fields, shared discount helpers, product DTO mapping, order discount application, tests, and API docs.

- [ ] **Step 5: Commit Task 5**

Run:

```bash
git add docs/api.md internal/models/product.go internal/models/order.go internal/services/product_discount.go internal/services/product_discount_test.go internal/services/product_service.go internal/services/product_service_discount_test.go internal/services/order_service.go internal/services/order_discount_test.go
git commit -m "docs: document product discounts"
```

---

## Final Verification

- [ ] Run `go test ./...` and confirm PASS.
- [ ] Run `git status --short` and confirm the working tree is clean.
- [ ] Review recent commits with `git log --oneline -5` and confirm the feature is split into focused commits.

## Self-Review Notes

- Spec coverage: product model fields, order audit fields, raw and computed product response fields, validation, clamp behavior, no backfill, and checkout payment totals are each covered by tasks.
- Type consistency: DTOs use `DiscountedPrice` and `DiscountAmount`; order audit fields use `OriginalPrice`, `DiscountType`, `DiscountValue`, and `DiscountAmount`; helper names are shared consistently.
- Scope check: plan stays within product-level discounts and does not add campaigns, coupons, time windows, or variant-specific discount storage.
