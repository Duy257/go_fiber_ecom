package services

import (
	"math"

	"go-fiber/internal/models"
)

type ProductResponse struct {
	ID             string                 `json:"id"`
	ShopID         string                 `json:"shop_id"`
	Shop           interface{}            `json:"shop,omitempty"`
	Name           string                 `json:"name"`
	Slug           string                 `json:"slug"`
	Description    string                 `json:"description,omitempty"`
	Price          float64                `json:"price"`
	DiscountType   string                 `json:"discount_type,omitempty"`
	DiscountValue  float64                `json:"discount_value"`
	DiscountedPrice float64               `json:"discounted_price"`
	DiscountAmount float64                `json:"discount_amount"`
	Images         []ProductImageResponse `json:"images,omitempty"`
	Variants       []ProductVariantResponse `json:"variants,omitempty"`
	Categories     []interface{}          `json:"categories,omitempty"`
	Status         string                 `json:"status"`
	CreatedAt      string                 `json:"created_at"`
	UpdatedAt      string                 `json:"updated_at"`
}

type ProductVariantResponse struct {
	ID              string `json:"id"`
	ProductID       string `json:"product_id"`
	Name            string `json:"name"`
	SKU             string `json:"sku,omitempty"`
	Price           float64 `json:"price"`
	DiscountedPrice float64 `json:"discounted_price"`
	DiscountAmount  float64 `json:"discount_amount"`
	Stock           int    `json:"stock"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

type ProductImageResponse struct {
	ID        string `json:"id"`
	ProductID string `json:"product_id"`
	URL       string `json:"url"`
	SortOrder int    `json:"sort_order"`
	CreatedAt string `json:"created_at"`
}

// CalculateDiscount computes the discounted price and discount amount for a given price and discount rule.
// The discount_amount is clamped so the final discounted_price is never below 0.
func CalculateDiscount(price float64, discountType string, discountValue float64) (discountedPrice float64, discountAmount float64) {
	if discountType == "" || discountValue <= 0 {
		return price, 0
	}

	var rawDiscount float64
	switch discountType {
	case models.ProductDiscountTypePercent:
		rawDiscount = price * discountValue / 100
	case models.ProductDiscountTypeFixedAmount:
		rawDiscount = discountValue
	default:
		return price, 0
	}

	discountAmount = math.Min(rawDiscount, price)
	discountedPrice = price - discountAmount
	return discountedPrice, discountAmount
}

// ToProductResponse maps a models.Product to a ProductResponse with computed discount fields.
func ToProductResponse(product *models.Product) *ProductResponse {
	if product == nil {
		return nil
	}

	discountedPrice, discountAmount := CalculateDiscount(product.Price, product.DiscountType, product.DiscountValue)

	resp := &ProductResponse{
		ID:    product.ID.String(),
		ShopID: product.ShopID.String(),
		Name:  product.Name,
		Slug:  product.Slug,
		Description: product.Description,
		Price: product.Price,
		DiscountType:  product.DiscountType,
		DiscountValue: product.DiscountValue,
		DiscountedPrice: discountedPrice,
		DiscountAmount:  discountAmount,
		Status: product.Status,
		CreatedAt: product.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: product.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if product.Shop.Name != "" {
		resp.Shop = map[string]interface{}{
			"id":   product.ShopID.String(),
			"name": product.Shop.Name,
		}
	}

	for _, img := range product.Images {
		resp.Images = append(resp.Images, ProductImageResponse{
			ID:        img.ID.String(),
			ProductID: img.ProductID.String(),
			URL:       img.URL,
			SortOrder: img.SortOrder,
			CreatedAt: img.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	for _, v := range product.Variants {
		vDiscountedPrice, vDiscountAmount := CalculateDiscount(v.Price, product.DiscountType, product.DiscountValue)
		vr := ProductVariantResponse{
			ID:    v.ID.String(),
			ProductID: v.ProductID.String(),
			Name:  v.Name,
			Price: v.Price,
			DiscountedPrice: vDiscountedPrice,
			DiscountAmount:  vDiscountAmount,
			Stock: v.Stock,
			CreatedAt: v.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: v.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		if v.SKU != nil {
			vr.SKU = *v.SKU
		}
		resp.Variants = append(resp.Variants, vr)
	}

	for _, cat := range product.Categories {
		resp.Categories = append(resp.Categories, map[string]interface{}{
			"id":   cat.ID.String(),
			"name": cat.Name,
		})
	}

	return resp
}
