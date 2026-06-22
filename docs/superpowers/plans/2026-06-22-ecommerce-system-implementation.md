# E-Commerce System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Xây dựng hệ thống ecommerce hoàn chỉnh với Category, Shop, Product, và Order trên go-fiber backend.

**Architecture:** Tuân theo clean architecture hiện có: Handler → Service → Repository → Model. Mỗi module có 4 layer riêng biệt. Soft delete trên tất cả bảng chính.

**Tech Stack:** Go, Fiber v2, GORM, PostgreSQL, go-playground/validator, google/uuid

---

## File Structure

```
internal/
├── models/
│   ├── category.go          # Category, ProductCategory
│   ├── shop.go              # Shop
│   ├── product.go           # Product, ProductVariant, ProductImage
│   └── order.go             # Order, OrderItem, OrderStatusHistory, Payment
├── repositories/
│   ├── category_repo.go
│   ├── shop_repo.go
│   ├── product_repo.go
│   └── order_repo.go
├── services/
│   ├── category_service.go
│   ├── shop_service.go
│   ├── product_service.go
│   └── order_service.go
├── handlers/
│   ├── category_handler.go
│   ├── shop_handler.go
│   ├── product_handler.go
│   └── order_handler.go
├── database/
│   └── database.go          # Thêm models mới vào Migrate()
└── cmd/server/
    └── main.go              # Đăng ký routes mới
```

---

### Task 1: Category Model

**Files:**
- Create: `internal/models/category.go`
- Modify: `internal/database/database.go`

- [ ] **Step 1: Tạo Category model**

```go
// internal/models/category.go
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Category struct {
	ID          uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name        string         `gorm:"type:varchar(255);not null" json:"name"`
	Slug        string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"slug"`
	Description string         `gorm:"type:text" json:"description,omitempty"`
	Image       string         `gorm:"type:varchar(500)" json:"image,omitempty"`
	ParentID    *uuid.UUID     `gorm:"type:uuid;index" json:"parent_id,omitempty"`
	Parent      *Category      `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children    []Category     `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	SortOrder   int            `gorm:"default:0" json:"sort_order"`
	Status      string         `gorm:"type:varchar(20);default:active" json:"status"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type ProductCategory struct {
	ProductID  uuid.UUID `gorm:"type:uuid;primaryKey"`
	CategoryID uuid.UUID `gorm:"type:uuid;primaryKey"`
}
```

- [ ] **Step 2: Thêm vào database migration**

```go
// internal/database/database.go - thêm vào hàm Migrate()
func Migrate(db *gorm.DB) {
	err := db.AutoMigrate(
		&models.Role{},
		&models.Permission{},
		&models.User{},
		&models.Customer{},
		&models.Category{},        // thêm
		&models.ProductCategory{}, // thêm
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/models/category.go internal/database/database.go
git commit -m "feat: add Category model and migration"
```

---

### Task 2: Shop Model

**Files:**
- Create: `internal/models/shop.go`
- Modify: `internal/database/database.go`

- [ ] **Step 1: Tạo Shop model**

```go
// internal/models/shop.go
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Shop struct {
	ID          uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID      uuid.UUID      `gorm:"type:uuid;uniqueIndex;not null" json:"user_id"`
	User        User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Name        string         `gorm:"type:varchar(255);not null" json:"name"`
	Slug        string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"slug"`
	Description string         `gorm:"type:text" json:"description,omitempty"`
	Logo        string         `gorm:"type:varchar(500)" json:"logo,omitempty"`
	Address     string         `gorm:"type:varchar(500)" json:"address,omitempty"`
	Phone       string         `gorm:"type:varchar(20)" json:"phone,omitempty"`
	Status      string         `gorm:"type:varchar(20);default:active" json:"status"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}
```

- [ ] **Step 2: Thêm vào database migration**

```go
// internal/database/database.go - thêm &models.Shop{} vào AutoMigrate
```

- [ ] **Step 3: Commit**

```bash
git add internal/models/shop.go internal/database/database.go
git commit -m "feat: add Shop model and migration"
```

---

### Task 3: Product Model

**Files:**
- Create: `internal/models/product.go`
- Modify: `internal/database/database.go`

- [ ] **Step 1: Tạo Product, ProductVariant, ProductImage models**

```go
// internal/models/product.go
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Product struct {
	ID          uuid.UUID        `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ShopID      uuid.UUID        `gorm:"type:uuid;index;not null" json:"shop_id"`
	Shop        Shop             `gorm:"foreignKey:ShopID" json:"shop,omitempty"`
	Name        string           `gorm:"type:varchar(255);not null" json:"name"`
	Slug        string           `gorm:"type:varchar(255);uniqueIndex;not null" json:"slug"`
	Description string           `gorm:"type:text" json:"description,omitempty"`
	Images      []ProductImage   `gorm:"foreignKey:ProductID" json:"images,omitempty"`
	Variants    []ProductVariant `gorm:"foreignKey:ProductID" json:"variants,omitempty"`
	Categories  []Category       `gorm:"many2many:product_categories;" json:"categories,omitempty"`
	Price       float64          `gorm:"type:decimal(12,2);not null" json:"price"`
	Status      string           `gorm:"type:varchar(20);default:active" json:"status"`
	DeletedAt   gorm.DeletedAt   `gorm:"index" json:"-"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

type ProductVariant struct {
	ID         uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ProductID  uuid.UUID      `gorm:"type:uuid;index;not null" json:"product_id"`
	Product    Product        `gorm:"foreignKey:ProductID" json:"-"`
	Name       string         `gorm:"type:varchar(255);not null" json:"name"`
	SKU        string         `gorm:"type:varchar(100);uniqueIndex" json:"sku,omitempty"`
	Price      float64        `gorm:"type:decimal(12,2);not null" json:"price"`
	Stock      int            `gorm:"not null;default:0" json:"stock"`
	Attributes map[string]interface{} `gorm:"type:jsonb;serializer:json" json:"attributes,omitempty"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type ProductImage struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ProductID uuid.UUID `gorm:"type:uuid;index;not null" json:"product_id"`
	URL       string    `gorm:"type:varchar(500);not null" json:"url"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
}
```

- [ ] **Step 2: Thêm vào database migration**

```go
// internal/database/database.go - thêm vào AutoMigrate:
// &models.Product{},
// &models.ProductVariant{},
// &models.ProductImage{},
```

- [ ] **Step 3: Commit**

```bash
git add internal/models/product.go internal/database/database.go
git commit -m "feat: add Product, ProductVariant, ProductImage models"
```

---

### Task 4: Order Model

**Files:**
- Create: `internal/models/order.go`
- Modify: `internal/database/database.go`

- [ ] **Step 1: Tạo Order, OrderItem, OrderStatusHistory, Payment models**

```go
// internal/models/order.go
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Order struct {
	ID              uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	CustomerID      uuid.UUID      `gorm:"type:uuid;index;not null" json:"customer_id"`
	Customer        Customer       `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	ShopID          uuid.UUID      `gorm:"type:uuid;index;not null" json:"shop_id"`
	Shop            Shop           `gorm:"foreignKey:ShopID" json:"shop,omitempty"`
	OrderNumber     string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"order_number"`
	Status          string         `gorm:"type:varchar(20);default:pending" json:"status"`
	SubTotal        float64        `gorm:"type:decimal(12,2);not null" json:"sub_total"`
	ShippingFee     float64        `gorm:"type:decimal(12,2);default:0" json:"shipping_fee"`
	TotalAmount     float64        `gorm:"type:decimal(12,2);not null" json:"total_amount"`
	ShippingAddress map[string]interface{} `gorm:"type:jsonb;serializer:json;not null" json:"shipping_address"`
	Note            string         `gorm:"type:text" json:"note,omitempty"`
	Items           []OrderItem    `gorm:"foreignKey:OrderID" json:"items,omitempty"`
	StatusHistory   []OrderStatusHistory `gorm:"foreignKey:OrderID" json:"status_history,omitempty"`
	Payment         *Payment       `gorm:"foreignKey:OrderID" json:"payment,omitempty"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type OrderItem struct {
	ID          uuid.UUID       `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrderID     uuid.UUID       `gorm:"type:uuid;index;not null" json:"order_id"`
	Order       Order           `gorm:"foreignKey:OrderID" json:"-"`
	ProductID   uuid.UUID       `gorm:"type:uuid;not null" json:"product_id"`
	Product     Product         `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	VariantID   *uuid.UUID      `gorm:"type:uuid" json:"variant_id,omitempty"`
	Variant     *ProductVariant `gorm:"foreignKey:VariantID" json:"variant,omitempty"`
	ProductName string          `gorm:"type:varchar(255);not null" json:"product_name"`
	VariantName string          `gorm:"type:varchar(255)" json:"variant_name,omitempty"`
	Price       float64         `gorm:"type:decimal(12,2);not null" json:"price"`
	Quantity    int             `gorm:"not null" json:"quantity"`
	Total       float64         `gorm:"type:decimal(12,2);not null" json:"total"`
}

type OrderStatusHistory struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrderID   uuid.UUID `gorm:"type:uuid;index;not null" json:"order_id"`
	Status    string    `gorm:"type:varchar(20);not null" json:"status"`
	Note      string    `gorm:"type:text" json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Payment struct {
	ID            uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrderID       uuid.UUID      `gorm:"type:uuid;uniqueIndex;not null" json:"order_id"`
	Order         Order          `gorm:"foreignKey:OrderID" json:"-"`
	Method        string         `gorm:"type:varchar(50);not null" json:"method"`
	Status        string         `gorm:"type:varchar(20);default:pending" json:"status"`
	Amount        float64        `gorm:"type:decimal(12,2);not null" json:"amount"`
	TransactionID string         `gorm:"type:varchar(255)" json:"transaction_id,omitempty"`
	PaidAt        *time.Time     `json:"paid_at,omitempty"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}
```

- [ ] **Step 2: Thêm vào database migration**

```go
// internal/database/database.go - thêm vào AutoMigrate:
// &models.Order{},
// &models.OrderItem{},
// &models.OrderStatusHistory{},
// &models.Payment{},
```

- [ ] **Step 3: Commit**

```bash
git add internal/models/order.go internal/database/database.go
git commit -m "feat: add Order, OrderItem, OrderStatusHistory, Payment models"
```

---

### Task 5: Category Repository

**Files:**
- Create: `internal/repositories/category_repo.go`

- [ ] **Step 1: Tạo CategoryRepository**

```go
// internal/repositories/category_repo.go
package repositories

import (
	"go-fiber/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) Create(category *models.Category) error {
	return r.db.Create(category).Error
}

func (r *CategoryRepository) FindByID(id uuid.UUID) (*models.Category, error) {
	var category models.Category
	err := r.db.Preload("Children").First(&category, "id = ?", id).Error
	return &category, err
}

func (r *CategoryRepository) FindBySlug(slug string) (*models.Category, error) {
	var category models.Category
	err := r.db.Preload("Children").Where("slug = ?", slug).First(&category).Error
	return &category, err
}

func (r *CategoryRepository) FindAll(parentID *uuid.UUID, page, limit int) ([]models.Category, int64, error) {
	var categories []models.Category
	var total int64

	query := r.db.Model(&models.Category{})
	if parentID != nil {
		query = query.Where("parent_id = ?", parentID)
	} else {
		query = query.Where("parent_id IS NULL")
	}

	query.Count(&total)
	err := query.Preload("Children").Offset((page - 1) * limit).Limit(limit).Order("sort_order ASC, created_at DESC").Find(&categories).Error
	return categories, total, err
}

func (r *CategoryRepository) Update(category *models.Category) error {
	return r.db.Save(category).Error
}

func (r *CategoryRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Category{}, "id = ?", id).Error
}

func (r *CategoryRepository) HasProducts(id uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&models.ProductCategory{}).Where("category_id = ?", id).Count(&count).Error
	return count > 0, err
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/repositories/category_repo.go
git commit -m "feat: add CategoryRepository"
```

---

### Task 6: Category Service

**Files:**
- Create: `internal/services/category_service.go`

- [ ] **Step 1: Tạo CategoryService**

```go
// internal/services/category_service.go
package services

import (
	"errors"
	"fmt"
	"strings"

	"go-fiber/internal/models"
	"go-fiber/internal/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CategoryService struct {
	repo *repositories.CategoryRepository
}

func NewCategoryService(repo *repositories.CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

type CreateCategoryInput struct {
	Name        string  `json:"name" validate:"required"`
	Description string  `json:"description"`
	Image       string  `json:"image"`
	ParentID    *string `json:"parent_id"`
	SortOrder   int     `json:"sort_order"`
}

type UpdateCategoryInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Image       *string `json:"image"`
	ParentID    *string `json:"parent_id"`
	SortOrder   *int    `json:"sort_order"`
	Status      *string `json:"status"`
}

func generateSlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "--", "-")
	return slug
}

func (s *CategoryService) Create(input CreateCategoryInput) (*models.Category, error) {
	slug := generateSlug(input.Name)

	_, err := s.repo.FindBySlug(slug)
	if err == nil {
		return nil, errors.New("category with this name already exists")
	}

	category := &models.Category{
		Name:        input.Name,
		Slug:        slug,
		Description: input.Description,
		Image:       input.Image,
		SortOrder:   input.SortOrder,
		Status:      "active",
	}

	if input.ParentID != nil {
		parentID, err := uuid.Parse(*input.ParentID)
		if err != nil {
			return nil, errors.New("invalid parent_id")
		}
		parent, err := s.repo.FindByID(parentID)
		if err != nil {
			return nil, errors.New("parent category not found")
		}
		if parent.ParentID != nil {
			return nil, errors.New("cannot nest more than 2 levels")
		}
		category.ParentID = &parentID
	}

	if err := s.repo.Create(category); err != nil {
		return nil, err
	}
	return category, nil
}

func (s *CategoryService) GetByID(id uuid.UUID) (*models.Category, error) {
	category, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("category not found")
		}
		return nil, err
	}
	return category, nil
}

func (s *CategoryService) GetAll(parentID *string, page, limit int) ([]models.Category, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	var pid *uuid.UUID
	if parentID != nil {
		id, err := uuid.Parse(*parentID)
		if err != nil {
			return nil, 0, errors.New("invalid parent_id")
		}
		pid = &id
	}

	return s.repo.FindAll(pid, page, limit)
}

func (s *CategoryService) Update(id uuid.UUID, input UpdateCategoryInput) (*models.Category, error) {
	category, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("category not found")
		}
		return nil, err
	}

	if input.Name != nil {
		category.Name = *input.Name
		category.Slug = generateSlug(*input.Name)
	}
	if input.Description != nil {
		category.Description = *input.Description
	}
	if input.Image != nil {
		category.Image = *input.Image
	}
	if input.SortOrder != nil {
		category.SortOrder = *input.SortOrder
	}
	if input.Status != nil {
		category.Status = *input.Status
	}
	if input.ParentID != nil {
		if *input.ParentID == "" {
			category.ParentID = nil
		} else {
			parentID, err := uuid.Parse(*input.ParentID)
			if err != nil {
				return nil, errors.New("invalid parent_id")
			}
			if parentID == id {
				return nil, errors.New("category cannot be its own parent")
			}
			category.ParentID = &parentID
		}
	}

	if err := s.repo.Update(category); err != nil {
		return nil, err
	}
	return category, nil
}

func (s *CategoryService) Delete(id uuid.UUID) error {
	hasProducts, err := s.repo.HasProducts(id)
	if err != nil {
		return err
	}
	if hasProducts {
		return fmt.Errorf("cannot delete category with products")
	}
	return s.repo.Delete(id)
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/services/category_service.go
git commit -m "feat: add CategoryService"
```

---

### Task 7: Category Handler & Routes

**Files:**
- Create: `internal/handlers/category_handler.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Tạo CategoryHandler**

```go
// internal/handlers/category_handler.go
package handlers

import (
	"strconv"

	"go-fiber/internal/services"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type CategoryHandler struct {
	service *services.CategoryService
}

func NewCategoryHandler(service *services.CategoryService) *CategoryHandler {
	return &CategoryHandler{service: service}
}

func (h *CategoryHandler) Create(c *fiber.Ctx) error {
	var input services.CreateCategoryInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	if errs := utils.Validate(input); errs != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", errs)
	}

	category, err := h.service.Create(input)
	if err != nil {
		if utils.IsDuplicateEntry(err) {
			return utils.Error(c, 409, "DUPLICATE_ENTRY", err.Error())
		}
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, category, "Category created")
}

func (h *CategoryHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	category, err := h.service.GetByID(id)
	if err != nil {
		return utils.Error(c, 404, "NOT_FOUND", "Category not found")
	}

	return utils.Success(c, category, "")
}

func (h *CategoryHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	parentID := c.Query("parent_id")

	var pid *string
	if parentID != "" {
		pid = &parentID
	}

	categories, total, err := h.service.GetAll(pid, page, limit)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.SuccessWithPagination(c, categories, page, limit, total)
}

func (h *CategoryHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	var input services.UpdateCategoryInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	category, err := h.service.Update(id, input)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, category, "Category updated")
}

func (h *CategoryHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	if err := h.service.Delete(id); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, nil, "Category deleted")
}
```

- [ ] **Step 2: Đăng ký routes trong main.go**

```go
// cmd/server/main.go - thêm sau phần repositories
categoryRepo := repositories.NewCategoryRepository(db)

// thêm sau phần services
categoryService := services.NewCategoryService(categoryRepo)

// thêm sau phần handlers
categoryHandler := handlers.NewCategoryHandler(categoryService)

// thêm routes (public cho GET, admin cho CRUD)
categories := api.Group("/categories")
categories.Get("/", categoryHandler.GetAll)
categories.Get("/:id", categoryHandler.GetByID)

adminCategories := api.Group("/admin/categories", middleware.JWTAuth(cfg))
adminCategories.Post("/", middleware.RequirePermission(userRepo, "category:write"), categoryHandler.Create)
adminCategories.Put("/:id", middleware.RequirePermission(userRepo, "category:write"), categoryHandler.Update)
adminCategories.Delete("/:id", middleware.RequirePermission(userRepo, "category:delete"), categoryHandler.Delete)
```

- [ ] **Step 3: Thêm permissions vào seed data**

```go
// cmd/server/main.go - thêm vào seedData()
{Name: "category:read", Description: "View categories"},
{Name: "category:write", Description: "Create/update categories"},
{Name: "category:delete", Description: "Delete categories"},
```

- [ ] **Step 4: Commit**

```bash
git add internal/handlers/category_handler.go cmd/server/main.go
git commit -m "feat: add CategoryHandler and routes"
```

---

### Task 8: Shop Repository, Service, Handler & Routes

**Files:**
- Create: `internal/repositories/shop_repo.go`
- Create: `internal/services/shop_service.go`
- Create: `internal/handlers/shop_handler.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Tạo ShopRepository**

```go
// internal/repositories/shop_repo.go
package repositories

import (
	"go-fiber/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ShopRepository struct {
	db *gorm.DB
}

func NewShopRepository(db *gorm.DB) *ShopRepository {
	return &ShopRepository{db: db}
}

func (r *ShopRepository) Create(shop *models.Shop) error {
	return r.db.Create(shop).Error
}

func (r *ShopRepository) FindByID(id uuid.UUID) (*models.Shop, error) {
	var shop models.Shop
	err := r.db.Preload("User").First(&shop, "id = ?", id).Error
	return &shop, err
}

func (r *ShopRepository) FindByUserID(userID uuid.UUID) (*models.Shop, error) {
	var shop models.Shop
	err := r.db.Where("user_id = ?", userID).First(&shop).Error
	return &shop, err
}

func (r *ShopRepository) FindBySlug(slug string) (*models.Shop, error) {
	var shop models.Shop
	err := r.db.Where("slug = ?", slug).First(&shop).Error
	return &shop, err
}

func (r *ShopRepository) FindAll(page, limit int) ([]models.Shop, int64, error) {
	var shops []models.Shop
	var total int64

	r.db.Model(&models.Shop{}).Count(&total)
	err := r.db.Preload("User").Offset((page - 1) * limit).Limit(limit).Order("created_at DESC").Find(&shops).Error
	return shops, total, err
}

func (r *ShopRepository) Update(shop *models.Shop) error {
	return r.db.Save(shop).Error
}

func (r *ShopRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Shop{}, "id = ?", id).Error
}
```

- [ ] **Step 2: Tạo ShopService**

```go
// internal/services/shop_service.go
package services

import (
	"errors"
	"strings"

	"go-fiber/internal/models"
	"go-fiber/internal/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ShopService struct {
	repo     *repositories.ShopRepository
	userRepo *repositories.UserRepository
}

func NewShopService(repo *repositories.ShopRepository, userRepo *repositories.UserRepository) *ShopService {
	return &ShopService{repo: repo, userRepo: userRepo}
}

type CreateShopInput struct {
	UserID      string `json:"user_id" validate:"required"`
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	Logo        string `json:"logo"`
	Address     string `json:"address"`
	Phone       string `json:"phone"`
}

type UpdateShopInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Logo        *string `json:"logo"`
	Address     *string `json:"address"`
	Phone       *string `json:"phone"`
	Status      *string `json:"status"`
}

func (s *ShopService) Create(input CreateShopInput) (*models.Shop, error) {
	userID, err := uuid.Parse(input.UserID)
	if err != nil {
		return nil, errors.New("invalid user_id")
	}

	_, err = s.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	_, err = s.repo.FindByUserID(userID)
	if err == nil {
		return nil, errors.New("user already has a shop")
	}

	slug := strings.ToLower(strings.TrimSpace(input.Name))
	slug = strings.ReplaceAll(slug, " ", "-")

	shop := &models.Shop{
		UserID:      userID,
		Name:        input.Name,
		Slug:        slug,
		Description: input.Description,
		Logo:        input.Logo,
		Address:     input.Address,
		Phone:       input.Phone,
		Status:      "active",
	}

	if err := s.repo.Create(shop); err != nil {
		return nil, err
	}
	return shop, nil
}

func (s *ShopService) GetByID(id uuid.UUID) (*models.Shop, error) {
	shop, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("shop not found")
		}
		return nil, err
	}
	return shop, nil
}

func (s *ShopService) GetByUserID(userID uuid.UUID) (*models.Shop, error) {
	shop, err := s.repo.FindByUserID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("shop not found")
		}
		return nil, err
	}
	return shop, nil
}

func (s *ShopService) GetAll(page, limit int) ([]models.Shop, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	return s.repo.FindAll(page, limit)
}

func (s *ShopService) Update(id uuid.UUID, input UpdateShopInput) (*models.Shop, error) {
	shop, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("shop not found")
		}
		return nil, err
	}

	if input.Name != nil {
		shop.Name = *input.Name
		shop.Slug = strings.ToLower(strings.ReplaceAll(*input.Name, " ", "-"))
	}
	if input.Description != nil {
		shop.Description = *input.Description
	}
	if input.Logo != nil {
		shop.Logo = *input.Logo
	}
	if input.Address != nil {
		shop.Address = *input.Address
	}
	if input.Phone != nil {
		shop.Phone = *input.Phone
	}
	if input.Status != nil {
		shop.Status = *input.Status
	}

	if err := s.repo.Update(shop); err != nil {
		return nil, err
	}
	return shop, nil
}

func (s *ShopService) Delete(id uuid.UUID) error {
	_, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("shop not found")
		}
		return err
	}
	return s.repo.Delete(id)
}
```

- [ ] **Step 3: Tạo ShopHandler**

```go
// internal/handlers/shop_handler.go
package handlers

import (
	"strconv"

	"go-fiber/internal/services"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ShopHandler struct {
	service *services.ShopService
}

func NewShopHandler(service *services.ShopService) *ShopHandler {
	return &ShopHandler{service: service}
}

func (h *ShopHandler) Create(c *fiber.Ctx) error {
	var input services.CreateShopInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	if errs := utils.Validate(input); errs != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", errs)
	}

	shop, err := h.service.Create(input)
	if err != nil {
		if utils.IsDuplicateEntry(err) {
			return utils.Error(c, 409, "DUPLICATE_ENTRY", err.Error())
		}
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, shop, "Shop created")
}

func (h *ShopHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	shop, err := h.service.GetByID(id)
	if err != nil {
		return utils.Error(c, 404, "NOT_FOUND", "Shop not found")
	}

	return utils.Success(c, shop, "")
}

func (h *ShopHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	shops, total, err := h.service.GetAll(page, limit)
	if err != nil {
		return utils.Error(c, 500, "INTERNAL_ERROR", "Failed to fetch shops")
	}

	return utils.SuccessWithPagination(c, shops, page, limit, total)
}

func (h *ShopHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	var input services.UpdateShopInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	shop, err := h.service.Update(id, input)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, shop, "Shop updated")
}

func (h *ShopHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	if err := h.service.Delete(id); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, nil, "Shop deleted")
}
```

- [ ] **Step 4: Đăng ký routes trong main.go**

```go
// cmd/server/main.go
shopRepo := repositories.NewShopRepository(db)
shopService := services.NewShopService(shopRepo, userRepo)
shopHandler := handlers.NewShopHandler(shopService)

// Public
shops := api.Group("/shops")
shops.Get("/", shopHandler.GetAll)
shops.Get("/:id", shopHandler.GetByID)

// Admin
adminShops := api.Group("/admin/shops", middleware.JWTAuth(cfg))
adminShops.Post("/", middleware.RequirePermission(userRepo, "shop:write"), shopHandler.Create)
adminShops.Put("/:id", middleware.RequirePermission(userRepo, "shop:write"), shopHandler.Update)
adminShops.Delete("/:id", middleware.RequirePermission(userRepo, "shop:delete"), shopHandler.Delete)
```

- [ ] **Step 5: Thêm permissions và commit**

```bash
# Thêm vào seedData:
# {Name: "shop:read", Description: "View shops"},
# {Name: "shop:write", Description: "Create/update shops"},
# {Name: "shop:delete", Description: "Delete shops"},

git add internal/repositories/shop_repo.go internal/services/shop_service.go internal/handlers/shop_handler.go cmd/server/main.go
git commit -m "feat: add Shop module (repo, service, handler, routes)"
```

---

### Task 9: Product Repository, Service, Handler & Routes

**Files:**
- Create: `internal/repositories/product_repo.go`
- Create: `internal/services/product_service.go`
- Create: `internal/handlers/product_handler.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Tạo ProductRepository**

```go
// internal/repositories/product_repo.go
package repositories

import (
	"go-fiber/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) Create(product *models.Product) error {
	return r.db.Create(product).Error
}

func (r *ProductRepository) FindByID(id uuid.UUID) (*models.Product, error) {
	var product models.Product
	err := r.db.Preload("Shop").Preload("Variants").Preload("Images").Preload("Categories").First(&product, "id = ?", id).Error
	return &product, err
}

func (r *ProductRepository) FindBySlug(slug string) (*models.Product, error) {
	var product models.Product
	err := r.db.Preload("Shop").Preload("Variants").Preload("Images").Preload("Categories").Where("slug = ?", slug).First(&product).Error
	return &product, err
}

func (r *ProductRepository) FindAll(shopID *uuid.UUID, categoryID *uuid.UUID, page, limit int) ([]models.Product, int64, error) {
	var products []models.Product
	var total int64

	query := r.db.Model(&models.Product{})
	if shopID != nil {
		query = query.Where("shop_id = ?", shopID)
	}
	if categoryID != nil {
		query = query.Joins("JOIN product_categories ON product_categories.product_id = products.id").
			Where("product_categories.category_id = ?", categoryID)
	}

	query.Count(&total)
	err := query.Preload("Shop").Preload("Variants").Preload("Images").
		Offset((page - 1) * limit).Limit(limit).Order("created_at DESC").Find(&products).Error
	return products, total, err
}

func (r *ProductRepository) Update(product *models.Product) error {
	return r.db.Save(product).Error
}

func (r *ProductRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Product{}, "id = ?", id).Error
}

func (r *ProductRepository) UpdateStock(variantID uuid.UUID, quantity int) error {
	return r.db.Model(&models.ProductVariant{}).Where("id = ?", variantID).
		Update("stock", gorm.Expr("stock - ?", quantity)).Error
}

func (r *ProductRepository) RestoreStock(variantID uuid.UUID, quantity int) error {
	return r.db.Model(&models.ProductVariant{}).Where("id = ?", variantID).
		Update("stock", gorm.Expr("stock + ?", quantity)).Error
}
```

- [ ] **Step 2: Tạo ProductService**

```go
// internal/services/product_service.go
package services

import (
	"errors"
	"strings"

	"go-fiber/internal/models"
	"go-fiber/internal/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductService struct {
	repo     *repositories.ProductRepository
	shopRepo *repositories.ShopRepository
}

func NewProductService(repo *repositories.ProductRepository, shopRepo *repositories.ShopRepository) *ProductService {
	return &ProductService{repo: repo, shopRepo: shopRepo}
}

type CreateProductInput struct {
	ShopID      string                  `json:"shop_id" validate:"required"`
	Name        string                  `json:"name" validate:"required"`
	Description string                  `json:"description"`
	Price       float64                 `json:"price" validate:"required,gt=0"`
	CategoryIDs []string                `json:"category_ids"`
	Variants    []CreateVariantInput    `json:"variants" validate:"required,min=1"`
	Images      []CreateImageInput      `json:"images"`
}

type CreateVariantInput struct {
	Name       string                 `json:"name" validate:"required"`
	SKU        string                 `json:"sku"`
	Price      float64                `json:"price" validate:"required,gt=0"`
	Stock      int                    `json:"stock" validate:"min=0"`
	Attributes map[string]interface{} `json:"attributes"`
}

type CreateImageInput struct {
	URL       string `json:"url" validate:"required"`
	SortOrder int    `json:"sort_order"`
}

type UpdateProductInput struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Price       *float64 `json:"price"`
	Status      *string  `json:"status"`
	CategoryIDs []string `json:"category_ids"`
}

func (s *ProductService) Create(input CreateProductInput) (*models.Product, error) {
	shopID, err := uuid.Parse(input.ShopID)
	if err != nil {
		return nil, errors.New("invalid shop_id")
	}

	_, err = s.shopRepo.FindByID(shopID)
	if err != nil {
		return nil, errors.New("shop not found")
	}

	slug := strings.ToLower(strings.TrimSpace(input.Name))
	slug = strings.ReplaceAll(slug, " ", "-")

	product := &models.Product{
		ShopID:      shopID,
		Name:        input.Name,
		Slug:        slug,
		Description: input.Description,
		Price:       input.Price,
		Status:      "active",
	}

	for _, v := range input.Variants {
		variant := models.ProductVariant{
			Name:       v.Name,
			SKU:        v.SKU,
			Price:      v.Price,
			Stock:      v.Stock,
			Attributes: v.Attributes,
		}
		product.Variants = append(product.Variants, variant)
	}

	for _, img := range input.Images {
		image := models.ProductImage{
			URL:       img.URL,
			SortOrder: img.SortOrder,
		}
		product.Images = append(product.Images, image)
	}

	if len(input.CategoryIDs) > 0 {
		for _, catIDStr := range input.CategoryIDs {
			catID, err := uuid.Parse(catIDStr)
			if err != nil {
				return nil, errors.New("invalid category_id")
			}
			product.Categories = append(product.Categories, models.Category{ID: catID})
		}
	}

	if err := s.repo.Create(product); err != nil {
		return nil, err
	}
	return product, nil
}

func (s *ProductService) GetByID(id uuid.UUID) (*models.Product, error) {
	product, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("product not found")
		}
		return nil, err
	}
	return product, nil
}

func (s *ProductService) GetAll(shopID, categoryID *string, page, limit int) ([]models.Product, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	var sid, cid *uuid.UUID
	if shopID != nil {
		id, err := uuid.Parse(*shopID)
		if err != nil {
			return nil, 0, errors.New("invalid shop_id")
		}
		sid = &id
	}
	if categoryID != nil {
		id, err := uuid.Parse(*categoryID)
		if err != nil {
			return nil, 0, errors.New("invalid category_id")
		}
		cid = &id
	}

	return s.repo.FindAll(sid, cid, page, limit)
}

func (s *ProductService) Update(id uuid.UUID, input UpdateProductInput) (*models.Product, error) {
	product, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("product not found")
		}
		return nil, err
	}

	if input.Name != nil {
		product.Name = *input.Name
		product.Slug = strings.ToLower(strings.ReplaceAll(*input.Name, " ", "-"))
	}
	if input.Description != nil {
		product.Description = *input.Description
	}
	if input.Price != nil {
		product.Price = *input.Price
	}
	if input.Status != nil {
		product.Status = *input.Status
	}

	if input.CategoryIDs != nil {
		var categories []models.Category
		for _, catIDStr := range input.CategoryIDs {
			catID, err := uuid.Parse(catIDStr)
			if err != nil {
				return nil, errors.New("invalid category_id")
			}
			categories = append(categories, models.Category{ID: catID})
		}
		product.Categories = categories
	}

	if err := s.repo.Update(product); err != nil {
		return nil, err
	}
	return product, nil
}

func (s *ProductService) Delete(id uuid.UUID) error {
	_, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("product not found")
		}
		return err
	}
	return s.repo.Delete(id)
}
```

- [ ] **Step 3: Tạo ProductHandler**

```go
// internal/handlers/product_handler.go
package handlers

import (
	"strconv"

	"go-fiber/internal/services"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ProductHandler struct {
	service *services.ProductService
}

func NewProductHandler(service *services.ProductService) *ProductHandler {
	return &ProductHandler{service: service}
}

func (h *ProductHandler) Create(c *fiber.Ctx) error {
	var input services.CreateProductInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	if errs := utils.Validate(input); errs != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", errs)
	}

	product, err := h.service.Create(input)
	if err != nil {
		if utils.IsDuplicateEntry(err) {
			return utils.Error(c, 409, "DUPLICATE_ENTRY", err.Error())
		}
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, product, "Product created")
}

func (h *ProductHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	product, err := h.service.GetByID(id)
	if err != nil {
		return utils.Error(c, 404, "NOT_FOUND", "Product not found")
	}

	return utils.Success(c, product, "")
}

func (h *ProductHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	shopID := c.Query("shop_id")
	categoryID := c.Query("category_id")

	var sid, cid *string
	if shopID != "" {
		sid = &shopID
	}
	if categoryID != "" {
		cid = &categoryID
	}

	products, total, err := h.service.GetAll(sid, cid, page, limit)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.SuccessWithPagination(c, products, page, limit, total)
}

func (h *ProductHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	var input services.UpdateProductInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	product, err := h.service.Update(id, input)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, product, "Product updated")
}

func (h *ProductHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	if err := h.service.Delete(id); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, nil, "Product deleted")
}
```

- [ ] **Step 4: Đăng ký routes trong main.go**

```go
// cmd/server/main.go
productRepo := repositories.NewProductRepository(db)
productService := services.NewProductService(productRepo, shopRepo)
productHandler := handlers.NewProductHandler(productService)

// Public
products := api.Group("/products")
products.Get("/", productHandler.GetAll)
products.Get("/:id", productHandler.GetByID)

// Admin
adminProducts := api.Group("/admin/products", middleware.JWTAuth(cfg))
adminProducts.Post("/", middleware.RequirePermission(userRepo, "product:write"), productHandler.Create)
adminProducts.Put("/:id", middleware.RequirePermission(userRepo, "product:write"), productHandler.Update)
adminProducts.Delete("/:id", middleware.RequirePermission(userRepo, "product:delete"), productHandler.Delete)
```

- [ ] **Step 5: Thêm permissions và commit**

```bash
# Thêm vào seedData:
# {Name: "product:read", Description: "View products"},
# {Name: "product:write", Description: "Create/update products"},
# {Name: "product:delete", Description: "Delete products"},

git add internal/repositories/product_repo.go internal/services/product_service.go internal/handlers/product_handler.go cmd/server/main.go
git commit -m "feat: add Product module (repo, service, handler, routes)"
```

---

### Task 10: Order Repository, Service, Handler & Routes

**Files:**
- Create: `internal/repositories/order_repo.go`
- Create: `internal/services/order_service.go`
- Create: `internal/handlers/order_handler.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Tạo OrderRepository**

```go
// internal/repositories/order_repo.go
package repositories

import (
	"fmt"
	"time"

	"go-fiber/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(order *models.Order) error {
	return r.db.Create(order).Error
}

func (r *OrderRepository) FindByID(id uuid.UUID) (*models.Order, error) {
	var order models.Order
	err := r.db.Preload("Customer").Preload("Shop").Preload("Items").Preload("StatusHistory").Preload("Payment").
		First(&order, "id = ?", id).Error
	return &order, err
}

func (r *OrderRepository) FindByOrderNumber(orderNumber string) (*models.Order, error) {
	var order models.Order
	err := r.db.Preload("Customer").Preload("Shop").Preload("Items").Preload("StatusHistory").Preload("Payment").
		Where("order_number = ?", orderNumber).First(&order).Error
	return &order, err
}

func (r *OrderRepository) FindByCustomerID(customerID uuid.UUID, page, limit int) ([]models.Order, int64, error) {
	var orders []models.Order
	var total int64

	query := r.db.Model(&models.Order{}).Where("customer_id = ?", customerID)
	query.Count(&total)
	err := query.Preload("Shop").Preload("Items").
		Offset((page - 1) * limit).Limit(limit).Order("created_at DESC").Find(&orders).Error
	return orders, total, err
}

func (r *OrderRepository) FindByShopID(shopID uuid.UUID, status *string, page, limit int) ([]models.Order, int64, error) {
	var orders []models.Order
	var total int64

	query := r.db.Model(&models.Order{}).Where("shop_id = ?", shopID)
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	query.Count(&total)
	err := query.Preload("Customer").Preload("Items").
		Offset((page - 1) * limit).Limit(limit).Order("created_at DESC").Find(&orders).Error
	return orders, total, err
}

func (r *OrderRepository) Update(order *models.Order) error {
	return r.db.Save(order).Error
}

func (r *OrderRepository) CreateStatusHistory(history *models.OrderStatusHistory) error {
	return r.db.Create(history).Error
}

func (r *OrderRepository) CreatePayment(payment *models.Payment) error {
	return r.db.Create(payment).Error
}

func (r *OrderRepository) UpdatePayment(payment *models.Payment) error {
	return r.db.Save(payment).Error
}

func (r *OrderRepository) FindPaymentByOrderID(orderID uuid.UUID) (*models.Payment, error) {
	var payment models.Payment
	err := r.db.Where("order_id = ?", orderID).First(&payment).Error
	return &payment, err
}

func (r *OrderRepository) GenerateOrderNumber() string {
	now := time.Now()
	count := int64(0)
	r.db.Model(&models.Order{}).Where("DATE(created_at) = CURRENT_DATE").Count(&count)
	return fmt.Sprintf("ORD-%s-%04d", now.Format("20060102"), count+1)
}
```

- [ ] **Step 2: Tạo OrderService**

```go
// internal/services/order_service.go
package services

import (
	"errors"
	"time"

	"go-fiber/internal/models"
	"go-fiber/internal/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderService struct {
	repo        *repositories.OrderRepository
	customerRepo *repositories.CustomerRepository
	productRepo  *repositories.ProductRepository
}

func NewOrderService(
	repo *repositories.OrderRepository,
	customerRepo *repositories.CustomerRepository,
	productRepo *repositories.ProductRepository,
) *OrderService {
	return &OrderService{
		repo:         repo,
		customerRepo: customerRepo,
		productRepo:  productRepo,
	}
}

type CreateOrderInput struct {
	CustomerID      string                 `json:"customer_id" validate:"required"`
	ShopID          string                 `json:"shop_id" validate:"required"`
	Items           []CreateOrderItemInput `json:"items" validate:"required,min=1"`
	ShippingFee     float64                `json:"shipping_fee"`
	ShippingAddress map[string]interface{} `json:"shipping_address" validate:"required"`
	Note            string                 `json:"note"`
	PaymentMethod   string                 `json:"payment_method" validate:"required,oneof=cod bank_transfer e_wallet"`
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

	orderNumber := s.repo.GenerateOrderNumber()

	order := &models.Order{
		CustomerID:      customerID,
		ShopID:          shopID,
		OrderNumber:     orderNumber,
		Status:          "pending",
		ShippingFee:     input.ShippingFee,
		ShippingAddress: input.ShippingAddress,
		Note:            input.Note,
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
			ProductID:   productID,
			ProductName: product.Name,
			Quantity:    item.Quantity,
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

			if variant.Stock < item.Quantity {
				return nil, errors.New("insufficient stock")
			}

			orderItem.VariantID = &variantID
			orderItem.VariantName = variant.Name
			orderItem.Price = variant.Price
		} else {
			orderItem.Price = product.Price
		}

		orderItem.Total = orderItem.Price * float64(item.Quantity)
		subTotal += orderItem.Total

		order.Items = append(order.Items, orderItem)
	}

	order.SubTotal = subTotal
	order.TotalAmount = subTotal + input.ShippingFee

	if err := s.repo.Create(order); err != nil {
		return nil, err
	}

	// Trừ stock
	for _, item := range order.Items {
		if item.VariantID != nil {
			if err := s.productRepo.UpdateStock(*item.VariantID, item.Quantity); err != nil {
				return nil, err
			}
		}
	}

	// Tạo payment
	payment := &models.Payment{
		OrderID: order.ID,
		Method:  input.PaymentMethod,
		Status:  "pending",
		Amount:  order.TotalAmount,
	}
	if err := s.repo.CreatePayment(payment); err != nil {
		return nil, err
	}

	// Tạo status history
	history := &models.OrderStatusHistory{
		OrderID: order.ID,
		Status:  "pending",
		Note:    "Order created",
	}
	if err := s.repo.CreateStatusHistory(history); err != nil {
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

	// Validate status transition
	validTransitions := map[string][]string{
		"pending":   {"confirmed", "cancelled"},
		"confirmed": {"shipping", "cancelled"},
		"shipping":  {"delivered"},
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

	order.Status = input.Status
	if err := s.repo.Update(order); err != nil {
		return nil, err
	}

	// Ghi status history
	history := &models.OrderStatusHistory{
		OrderID: order.ID,
		Status:  input.Status,
		Note:    input.Note,
	}
	if err := s.repo.CreateStatusHistory(history); err != nil {
		return nil, err
	}

	// Xử lý payment khi delivered (COD)
	if input.Status == "delivered" {
		payment, err := s.repo.FindPaymentByOrderID(order.ID)
		if err == nil && payment.Method == "cod" && payment.Status == "pending" {
			payment.Status = "paid"
			now := time.Now()
			payment.PaidAt = &now
			s.repo.UpdatePayment(payment)
		}
	}

	return order, nil
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

	order.Status = "cancelled"
	if err := s.repo.Update(order); err != nil {
		return nil, err
	}

	// Hoàn stock
	for _, item := range order.Items {
		if item.VariantID != nil {
			s.productRepo.RestoreStock(*item.VariantID, item.Quantity)
		}
	}

	// Cập nhật payment
	payment, err := s.repo.FindPaymentByOrderID(order.ID)
	if err == nil {
		if payment.Status == "paid" {
			payment.Status = "refunded"
		} else {
			payment.Status = "failed"
		}
		s.repo.UpdatePayment(payment)
	}

	// Ghi status history
	history := &models.OrderStatusHistory{
		OrderID: order.ID,
		Status:  "cancelled",
		Note:    note,
	}
	s.repo.CreateStatusHistory(history)

	return order, nil
}
```

- [ ] **Step 3: Tạo OrderHandler**

```go
// internal/handlers/order_handler.go
package handlers

import (
	"strconv"

	"go-fiber/internal/services"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type OrderHandler struct {
	service *services.OrderService
}

func NewOrderHandler(service *services.OrderService) *OrderHandler {
	return &OrderHandler{service: service}
}

func (h *OrderHandler) Create(c *fiber.Ctx) error {
	var input services.CreateOrderInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	if errs := utils.Validate(input); errs != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", errs)
	}

	order, err := h.service.Create(input)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, order, "Order created")
}

func (h *OrderHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	order, err := h.service.GetByID(id)
	if err != nil {
		return utils.Error(c, 404, "NOT_FOUND", "Order not found")
	}

	return utils.Success(c, order, "")
}

func (h *OrderHandler) GetByCustomer(c *fiber.Ctx) error {
	customerID, err := uuid.Parse(c.Params("customer_id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid customer ID")
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	orders, total, err := h.service.GetByCustomerID(customerID, page, limit)
	if err != nil {
		return utils.Error(c, 500, "INTERNAL_ERROR", "Failed to fetch orders")
	}

	return utils.SuccessWithPagination(c, orders, page, limit, total)
}

func (h *OrderHandler) GetByShop(c *fiber.Ctx) error {
	shopID, err := uuid.Parse(c.Params("shop_id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid shop ID")
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	status := c.Query("status")

	var st *string
	if status != "" {
		st = &status
	}

	orders, total, err := h.service.GetByShopID(shopID, st, page, limit)
	if err != nil {
		return utils.Error(c, 500, "INTERNAL_ERROR", "Failed to fetch orders")
	}

	return utils.SuccessWithPagination(c, orders, page, limit, total)
}

func (h *OrderHandler) UpdateStatus(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	var input services.UpdateOrderStatusInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	if errs := utils.Validate(input); errs != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", errs)
	}

	order, err := h.service.UpdateStatus(id, input)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, order, "Order status updated")
}

func (h *OrderHandler) Cancel(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	note := c.Query("note", "Cancelled")

	order, err := h.service.Cancel(id, note)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, order, "Order cancelled")
}
```

- [ ] **Step 4: Đăng ký routes trong main.go**

```go
// cmd/server/main.go
orderRepo := repositories.NewOrderRepository(db)
orderService := services.NewOrderService(orderRepo, customerRepo, productRepo)
orderHandler := handlers.NewOrderHandler(orderService)

// Customer routes
customerOrders := api.Group("/customer/orders", middleware.JWTAuth(cfg))
customerOrders.Post("/", orderHandler.Create)
customerOrders.Get("/", orderHandler.GetByCustomer)
customerOrders.Get("/:id", orderHandler.GetByID)
customerOrders.Post("/:id/cancel", orderHandler.Cancel)

// Admin routes
adminOrders := api.Group("/admin/orders", middleware.JWTAuth(cfg))
adminOrders.Get("/", middleware.RequirePermission(userRepo, "order:read"), orderHandler.GetByShop)
adminOrders.Get("/:id", middleware.RequirePermission(userRepo, "order:read"), orderHandler.GetByID)
adminOrders.Put("/:id/status", middleware.RequirePermission(userRepo, "order:write"), orderHandler.UpdateStatus)
```

- [ ] **Step 5: Thêm permissions và commit**

```bash
# Thêm vào seedData:
# {Name: "order:read", Description: "View orders"},
# {Name: "order:write", Description: "Update order status"},

git add internal/repositories/order_repo.go internal/services/order_service.go internal/handlers/order_handler.go cmd/server/main.go
git commit -m "feat: add Order module (repo, service, handler, routes)"
```

---

### Task 11: Final Integration & Verify

**Files:**
- Modify: `cmd/server/main.go` (final check)

- [ ] **Step 1: Verify all imports trong main.go**

Đảm bảo tất cả imports đã đúng và không có lỗi compile.

- [ ] **Step 2: Run build**

```bash
go build ./cmd/server/
```

Expected: Build successful

- [ ] **Step 3: Run server và test migration**

```bash
go run cmd/server/main.go
```

Expected: Server starts, all tables created in database

- [ ] **Step 4: Commit final changes**

```bash
git add .
git commit -m "feat: complete ecommerce system integration"
```
