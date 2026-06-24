# Product Discount Design

Date: 2026-06-24

## Goal

Add product-level discounts to the ecommerce backend. A discount belongs to a product and applies to the product base price and every variant price. Orders must use the discounted price while preserving enough snapshot data to audit the original price and discount rule used at checkout.

## Current Context

Products currently store prices in two places:

- `Product.Price` is the base product price.
- `ProductVariant.Price` is the variant-specific price.

Order creation currently copies the selected product or variant price into `OrderItem.Price`, computes `OrderItem.Total` from that price, then uses the order subtotal for payment creation. There is no discount model or discount field today.

The project uses GORM `AutoMigrate`, so the discount change should be additive and safe for existing rows.

Product handlers currently return model values directly. This feature should introduce product response DTOs for computed discount fields instead of persisting derived values or adding ignored GORM fields to the model.

## Scope

In scope:

- Product-level discount fields.
- Percent and fixed amount discount types.
- Computed discounted price and discount amount in product responses.
- Discount application during order creation.
- Order item audit fields for original price and discount rule snapshot.
- Tests for validation, computation, and order snapshots.

Out of scope:

- Variant-specific discounts.
- Campaigns, coupons, stacking rules, usage limits, customer-specific discounts, and time windows.
- Discount history tables or admin reporting.

## Chosen Approach

Use discount fields directly on `Product` instead of a separate `ProductDiscount` table.

This fits the current requirement because each product can have at most one simple discount rule. It avoids extra joins and keeps the repository shape close to the current implementation. A separate table can be introduced later if discounts become campaign-like or need history.

## Data Model

Add these fields to `models.Product`:

```go
DiscountType  string  `gorm:"type:varchar(20)" json:"discount_type,omitempty"`
DiscountValue float64 `gorm:"type:decimal(12,2);default:0" json:"discount_value"`
```

Define discount type constants in the model package so validation and order calculation do not depend on repeated string literals:

```go
const (
	ProductDiscountTypePercent     = "percent"
	ProductDiscountTypeFixedAmount = "fixed_amount"
)
```

Valid values:

- Empty `DiscountType` with `DiscountValue = 0` means no discount.
- `percent` applies `DiscountValue` as a percentage from `0` to `100`.
- `fixed_amount` subtracts `DiscountValue` from the selected price.

`ProductVariant` will not store discount fields. Product-level discounts apply to every variant at calculation time.

Add these fields to `models.OrderItem`:

```go
OriginalPrice  float64 `gorm:"type:decimal(12,2);not null;default:0" json:"original_price"`
DiscountType   string  `gorm:"type:varchar(20)" json:"discount_type,omitempty"`
DiscountValue  float64 `gorm:"type:decimal(12,2);default:0" json:"discount_value"`
DiscountAmount float64 `gorm:"type:decimal(12,2);default:0" json:"discount_amount"`
```

Field meanings:

- `OriginalPrice`: selected product or variant price at checkout before discount.
- `Price`: final unit price after discount. This preserves the existing payment flow because current order totals already use `OrderItem.Price`.
- `DiscountType`: discount type snapshot at checkout.
- `DiscountValue`: configured discount value snapshot at checkout.
- `DiscountAmount`: per-unit amount subtracted from `OriginalPrice`, not multiplied by quantity.
- `Total`: final unit price after discount multiplied by quantity.

## Discount Calculation

The calculation should be centralized in the service layer so product responses and order creation use the same rules.

Rules:

- No discount: `discounted_price = original_price`, `discount_amount = 0`.
- Percent raw discount: `original_price * discount_value / 100`.
- Fixed amount raw discount: `discount_value`.
- Actual `discount_amount`: the smaller value between the raw discount and `original_price`.
- `discounted_price = original_price - discount_amount`.

The clamp rule applies to both product base price and variant prices.

## API And Validation

Product create and update inputs should accept optional discount fields:

```json
{
  "discount_type": "percent",
  "discount_value": 10
}
```

Input shape:

- `CreateProductInput.DiscountType` can be a string because an omitted value and an empty string both mean no discount.
- `CreateProductInput.DiscountValue` can be a float because an omitted value should default to `0`.
- `UpdateProductInput.DiscountType` should be `*string`.
- `UpdateProductInput.DiscountValue` should be `*float64` so the service can distinguish omitted value from an explicit `0` used to clear or change a discount.

Validation rules:

- Empty discount is valid only when `discount_value` is `0`.
- `percent` requires `0 <= discount_value <= 100`.
- `fixed_amount` requires `discount_value >= 0`.
- Unknown discount types return a validation error.
- Fixed discounts larger than the product or variant price are allowed; calculation clamps the final price to `0`.

Update behavior:

- Sending `discount_type: ""` with `discount_value: 0` clears the discount.
- Sending only `discount_type` reuses the existing `discount_value`.
- Sending only `discount_value` requires the product to already have a `discount_type`; otherwise the update returns a validation error.
- After applying a partial update to the existing product values, validate the final `discount_type` and `discount_value` pair using the same rules as create.

Product list and detail responses should include raw and computed values:

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

Computed fields must be represented with response DTOs instead of persisted model fields to avoid storing derived data. The service should map `models.Product` and `models.ProductVariant` to response structs before returning to handlers.

## Order Flow

During order creation:

1. Resolve the product as today.
2. If `variant_id` is provided, use `ProductVariant.Price` as `OriginalPrice`; otherwise use `Product.Price`.
3. Apply the product discount to `OriginalPrice`.
4. Set `OrderItem.OriginalPrice` to the selected price before discount.
5. Set `OrderItem.Price` to the discounted unit price.
6. Snapshot `DiscountType`, `DiscountValue`, and `DiscountAmount` on the order item.
7. Set `OrderItem.Total = OrderItem.Price * Quantity`.
8. Compute `Order.SubTotal` from discounted item totals.
9. Compute `Order.TotalAmount = SubTotal + ShippingFee`.
10. Create payment using the discounted `Order.TotalAmount`.

When there is no active discount, new order items should still set `OriginalPrice` to the selected product or variant price and set `DiscountAmount` to `0`. This keeps new order audit data complete even when no discount applies.

Existing orders are unaffected. For historical rows, `OriginalPrice` will default to `0` unless backfilled. Since existing rows already have `Price`, read paths should treat `OriginalPrice = 0` as legacy data rather than recalculating old discounts.

## Error Handling

Validation failures should follow existing API conventions:

- HTTP status: `400`
- Error code: `VALIDATION_ERROR`
- Message examples: `invalid discount_type`, `discount_value must be between 0 and 100 for percent discount`, `discount_type is required when setting discount_value`

Database uniqueness and not-found behavior remain unchanged.

## Testing Plan

Product service tests:

- Create product with percent discount.
- Create product with fixed amount discount.
- Reject unknown discount type.
- Reject percent value greater than `100`.
- Reject discount value without discount type for a product that does not already have one.
- Allow update with explicit `discount_value: 0` when clearing discount through pointer input.
- Validate the final discount pair after partial update.
- Return computed product and variant `discounted_price` and `discount_amount`.
- Keep no-discount responses consistent: `discounted_price` equals `price` and `discount_amount` equals `0`.

Order service tests:

- Product-level percent discount applies to a product item.
- Product-level fixed amount discount applies to a variant item.
- Fixed amount larger than selected price clamps the final unit price to `0`.
- Order item snapshots `OriginalPrice`, `DiscountType`, `DiscountValue`, and per-unit `DiscountAmount`.
- `SubTotal`, `TotalAmount`, and payment amount use discounted totals.

Regression tests:

- Existing no-discount product orders keep current behavior: `Price` equals original product or variant price and `DiscountAmount` is `0`.
- Existing product create/update flows still work when discount fields are omitted.

## Migration Notes

GORM `AutoMigrate` should add the new columns:

- `products.discount_type`
- `products.discount_value`
- `order_items.original_price`
- `order_items.discount_type`
- `order_items.discount_value`
- `order_items.discount_amount`

No new tables are required.

Existing product rows default to no discount. Existing order rows keep their current `price` and `total`; their new audit fields are legacy defaults.

Do not backfill existing order audit fields as part of this feature. Historical order totals should remain exactly as stored.
