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

func newProductServiceTestDB(t *testing.T) *gorm.DB {
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
		`CREATE TABLE "categories" (
			"id" text PRIMARY KEY,
			"name" text NOT NULL,
			"slug" text NOT NULL UNIQUE,
			"description" text,
			"image" text,
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
		`CREATE TABLE "product_images" (
			"id" text PRIMARY KEY,
			"product_id" text NOT NULL,
			"url" text NOT NULL,
			"sort_order" integer DEFAULT 0,
			"created_at" timestamp,
			"updated_at" timestamp
		)`,
		`CREATE TABLE "product_categories" (
			"product_id" text NOT NULL,
			"category_id" text NOT NULL,
			PRIMARY KEY ("product_id", "category_id")
		)`,
	}
	for _, sql := range createTableSQL {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatalf("create table: %v", err)
		}
	}

	return db
}

func createTestShop(t *testing.T, db *gorm.DB) models.Shop {
	t.Helper()
	userID := uuid.New()
	shop := models.Shop{
		ID:     uuid.New(),
		UserID: userID,
		Name:   "Test Shop " + uuid.New().String()[:8],
		Slug:   "test-shop-" + uuid.New().String()[:8],
		Status: "active",
	}
	if err := db.Create(&shop).Error; err != nil {
		t.Fatalf("create shop: %v", err)
	}
	return shop
}

// --- Product service discount tests ---

func TestCreateProductWithPercentDiscount(t *testing.T) {
	db := newProductServiceTestDB(t)
	shop := createTestShop(t, db)

	productRepo := repositories.NewProductRepository(db)
	shopRepo := repositories.NewShopRepository(db)
	categoryRepo := repositories.NewCategoryRepository(db)
	svc := NewProductService(productRepo, shopRepo, categoryRepo)

	input := CreateProductInput{
		ShopID:       shop.ID.String(),
		Name:         "Discounted Product",
		Price:        200000,
		DiscountType: "percent",
		DiscountValue: 10,
		Variants: []CreateVariantInput{
			{Name: "Default", Price: 250000, Stock: 10},
		},
	}

	product, err := svc.Create(input)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if product.DiscountType != "percent" {
		t.Fatalf("DiscountType = %q, want %q", product.DiscountType, "percent")
	}
	if product.DiscountValue != 10 {
		t.Fatalf("DiscountValue = %v, want 10", product.DiscountValue)
	}

	resp := ToProductResponse(product)
	if resp.DiscountedPrice != 180000 {
		t.Fatalf("DiscountedPrice = %v, want 180000", resp.DiscountedPrice)
	}
	if resp.DiscountAmount != 20000 {
		t.Fatalf("DiscountAmount = %v, want 20000", resp.DiscountAmount)
	}

	if len(resp.Variants) != 1 {
		t.Fatalf("variants = %d, want 1", len(resp.Variants))
	}
	if resp.Variants[0].DiscountedPrice != 225000 {
		t.Fatalf("variant discounted_price = %v, want 225000", resp.Variants[0].DiscountedPrice)
	}
	if resp.Variants[0].DiscountAmount != 25000 {
		t.Fatalf("variant discount_amount = %v, want 25000", resp.Variants[0].DiscountAmount)
	}
}

func TestCreateProductWithFixedAmountDiscount(t *testing.T) {
	db := newProductServiceTestDB(t)
	shop := createTestShop(t, db)

	productRepo := repositories.NewProductRepository(db)
	shopRepo := repositories.NewShopRepository(db)
	categoryRepo := repositories.NewCategoryRepository(db)
	svc := NewProductService(productRepo, shopRepo, categoryRepo)

	input := CreateProductInput{
		ShopID:       shop.ID.String(),
		Name:         "Fixed Discount Product",
		Price:        200000,
		DiscountType: "fixed_amount",
		DiscountValue: 50000,
		Variants: []CreateVariantInput{
			{Name: "Default", Price: 250000, Stock: 10},
		},
	}

	product, err := svc.Create(input)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	resp := ToProductResponse(product)
	if resp.DiscountedPrice != 150000 {
		t.Fatalf("DiscountedPrice = %v, want 150000", resp.DiscountedPrice)
	}
	if resp.DiscountAmount != 50000 {
		t.Fatalf("DiscountAmount = %v, want 50000", resp.DiscountAmount)
	}

	if resp.Variants[0].DiscountedPrice != 200000 {
		t.Fatalf("variant discounted_price = %v, want 200000", resp.Variants[0].DiscountedPrice)
	}
	if resp.Variants[0].DiscountAmount != 50000 {
		t.Fatalf("variant discount_amount = %v, want 50000", resp.Variants[0].DiscountAmount)
	}
}

func TestRejectUnknownDiscountType(t *testing.T) {
	db := newProductServiceTestDB(t)
	shop := createTestShop(t, db)

	productRepo := repositories.NewProductRepository(db)
	shopRepo := repositories.NewShopRepository(db)
	categoryRepo := repositories.NewCategoryRepository(db)
	svc := NewProductService(productRepo, shopRepo, categoryRepo)

	input := CreateProductInput{
		ShopID:       shop.ID.String(),
		Name:         "Bad Discount",
		Price:        100000,
		DiscountType: "invalid_type",
		DiscountValue: 10,
		Variants: []CreateVariantInput{
			{Name: "Default", Price: 150000, Stock: 10},
		},
	}

	_, err := svc.Create(input)
	if err == nil {
		t.Fatal("expected error for invalid discount type, got nil")
	}
}

func TestRejectPercentOver100(t *testing.T) {
	db := newProductServiceTestDB(t)
	shop := createTestShop(t, db)

	productRepo := repositories.NewProductRepository(db)
	shopRepo := repositories.NewShopRepository(db)
	categoryRepo := repositories.NewCategoryRepository(db)
	svc := NewProductService(productRepo, shopRepo, categoryRepo)

	input := CreateProductInput{
		ShopID:       shop.ID.String(),
		Name:         "Over 100% Discount",
		Price:        100000,
		DiscountType: "percent",
		DiscountValue: 150,
		Variants: []CreateVariantInput{
			{Name: "Default", Price: 150000, Stock: 10},
		},
	}

	_, err := svc.Create(input)
	if err == nil {
		t.Fatal("expected error for percent > 100, got nil")
	}
}

func TestNoDiscountResponseHasConsistentValues(t *testing.T) {
	db := newProductServiceTestDB(t)
	shop := createTestShop(t, db)

	productRepo := repositories.NewProductRepository(db)
	shopRepo := repositories.NewShopRepository(db)
	categoryRepo := repositories.NewCategoryRepository(db)
	svc := NewProductService(productRepo, shopRepo, categoryRepo)

	input := CreateProductInput{
		ShopID: shop.ID.String(),
		Name:   "No Discount Product",
		Price:  200000,
		Variants: []CreateVariantInput{
			{Name: "Default", Price: 250000, Stock: 10},
		},
	}

	product, err := svc.Create(input)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	resp := ToProductResponse(product)
	if resp.DiscountedPrice != 200000 {
		t.Fatalf("DiscountedPrice = %v, want 200000 (no discount)", resp.DiscountedPrice)
	}
	if resp.DiscountAmount != 0 {
		t.Fatalf("DiscountAmount = %v, want 0 (no discount)", resp.DiscountAmount)
	}
	if resp.Variants[0].DiscountedPrice != 250000 {
		t.Fatalf("variant discounted_price = %v, want 250000", resp.Variants[0].DiscountedPrice)
	}
	if resp.Variants[0].DiscountAmount != 0 {
		t.Fatalf("variant discount_amount = %v, want 0", resp.Variants[0].DiscountAmount)
	}
}

func TestDiscountValueWithoutTypeOnCreate(t *testing.T) {
	db := newProductServiceTestDB(t)
	shop := createTestShop(t, db)

	productRepo := repositories.NewProductRepository(db)
	shopRepo := repositories.NewShopRepository(db)
	categoryRepo := repositories.NewCategoryRepository(db)
	svc := NewProductService(productRepo, shopRepo, categoryRepo)

	input := CreateProductInput{
		ShopID:       shop.ID.String(),
		Name:         "Discount Value No Type",
		Price:        100000,
		DiscountValue: 10, // no DiscountType set
		Variants: []CreateVariantInput{
			{Name: "Default", Price: 150000, Stock: 10},
		},
	}

	_, err := svc.Create(input)
	if err == nil {
		t.Fatal("expected error for discount_value without discount_type, got nil")
	}
}

func TestUpdateProductWithDiscount(t *testing.T) {
	db := newProductServiceTestDB(t)
	shop := createTestShop(t, db)

	productRepo := repositories.NewProductRepository(db)
	shopRepo := repositories.NewShopRepository(db)
	categoryRepo := repositories.NewCategoryRepository(db)
	svc := NewProductService(productRepo, shopRepo, categoryRepo)

	// Create product without discount
	product, err := svc.Create(CreateProductInput{
		ShopID: shop.ID.String(),
		Name:   "Updatable Product",
		Price:  200000,
		Variants: []CreateVariantInput{
			{Name: "Default", Price: 250000, Stock: 10},
		},
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	// Update with percent discount
	discountType := "percent"
	discountValue := 15.0
	updated, err := svc.Update(product.ID, UpdateProductInput{
		DiscountType:  &discountType,
		DiscountValue: &discountValue,
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	if updated.DiscountType != "percent" {
		t.Fatalf("DiscountType = %q, want %q", updated.DiscountType, "percent")
	}
	if updated.DiscountValue != 15 {
		t.Fatalf("DiscountValue = %v, want 15", updated.DiscountValue)
	}

	resp := ToProductResponse(updated)
	if resp.DiscountedPrice != 170000 {
		t.Fatalf("DiscountedPrice = %v, want 170000", resp.DiscountedPrice)
	}
	if resp.DiscountAmount != 30000 {
		t.Fatalf("DiscountAmount = %v, want 30000", resp.DiscountAmount)
	}
}

func TestUpdateClearDiscountWithExplicitZero(t *testing.T) {
	db := newProductServiceTestDB(t)
	shop := createTestShop(t, db)

	productRepo := repositories.NewProductRepository(db)
	shopRepo := repositories.NewShopRepository(db)
	categoryRepo := repositories.NewCategoryRepository(db)
	svc := NewProductService(productRepo, shopRepo, categoryRepo)

	// Create product with discount
	product, err := svc.Create(CreateProductInput{
		ShopID:       shop.ID.String(),
		Name:         "Clearable Discount",
		Price:        200000,
		DiscountType: "percent",
		DiscountValue: 10,
		Variants: []CreateVariantInput{
			{Name: "Default", Price: 250000, Stock: 10},
		},
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	// Clear discount by sending empty type and zero value
	emptyType := ""
	zeroVal := 0.0
	updated, err := svc.Update(product.ID, UpdateProductInput{
		DiscountType:  &emptyType,
		DiscountValue: &zeroVal,
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

	resp := ToProductResponse(updated)
	if resp.DiscountedPrice != 200000 {
		t.Fatalf("DiscountedPrice = %v, want 200000", resp.DiscountedPrice)
	}
	if resp.DiscountAmount != 0 {
		t.Fatalf("DiscountAmount = %v, want 0", resp.DiscountAmount)
	}
}

func TestGetProductResponseReturnsComputedFields(t *testing.T) {
	db := newProductServiceTestDB(t)
	shop := createTestShop(t, db)

	productRepo := repositories.NewProductRepository(db)
	shopRepo := repositories.NewShopRepository(db)
	categoryRepo := repositories.NewCategoryRepository(db)
	svc := NewProductService(productRepo, shopRepo, categoryRepo)

	product, err := svc.Create(CreateProductInput{
		ShopID:       shop.ID.String(),
		Name:         "Response Test",
		Price:        100000,
		DiscountType: "fixed_amount",
		DiscountValue: 25000,
		Variants: []CreateVariantInput{
			{Name: "Default", Price: 150000, Stock: 10},
		},
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	resp, err := svc.GetProductResponse(product.ID)
	if err != nil {
		t.Fatalf("GetProductResponse returned error: %v", err)
	}

	if resp.DiscountedPrice != 75000 {
		t.Fatalf("DiscountedPrice = %v, want 75000", resp.DiscountedPrice)
	}
	if resp.DiscountAmount != 25000 {
		t.Fatalf("DiscountAmount = %v, want 25000", resp.DiscountAmount)
	}
	if resp.Variants[0].DiscountedPrice != 125000 {
		t.Fatalf("variant discounted_price = %v, want 125000", resp.Variants[0].DiscountedPrice)
	}
	if resp.Variants[0].DiscountAmount != 25000 {
		t.Fatalf("variant discount_amount = %v, want 25000", resp.Variants[0].DiscountAmount)
	}
}

func TestCalculateDiscountClampFixedAmount(t *testing.T) {
	// Fixed discount larger than price should clamp to 0
	discountedPrice, discountAmount := CalculateDiscount(50000, "fixed_amount", 100000)
	if discountedPrice != 0 {
		t.Fatalf("discountedPrice = %v, want 0", discountedPrice)
	}
	if discountAmount != 50000 {
		t.Fatalf("discountAmount = %v, want 50000", discountAmount)
	}
}

func TestCalculateDiscountPercentAbove100(t *testing.T) {
	// Percent > 100 should still be clamped
	discountedPrice, discountAmount := CalculateDiscount(50000, "percent", 200)
	if discountedPrice != 0 {
		t.Fatalf("discountedPrice = %v, want 0", discountedPrice)
	}
	if discountAmount != 50000 {
		t.Fatalf("discountAmount = %v, want 50000", discountAmount)
	}
}

func TestValidateDiscountPercentNegative(t *testing.T) {
	err := validateDiscount("percent", -10)
	if err == nil {
		t.Fatal("expected error for negative percent, got nil")
	}
}

func TestValidateDiscountFixedAmountNegative(t *testing.T) {
	err := validateDiscount("fixed_amount", -5000)
	if err == nil {
		t.Fatal("expected error for negative fixed_amount, got nil")
	}
}
