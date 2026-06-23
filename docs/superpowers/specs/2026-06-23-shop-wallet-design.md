# Shop Wallet Design

**Date:** 2026-06-23
**Status:** Approved for implementation planning

## Goal

Add shop wallet support so each shop can track held funds, withdrawable funds, and wallet history.

The wallet must support:

- Crediting order revenue into a temporary held balance when an order becomes `completed`.
- Releasing held balance into withdrawable balance 7 days after the order becomes `completed`.
- Recording every wallet balance movement in `shop_wallet_logs`.
- Letting shop owners request withdrawals.
- Letting admins approve or reject withdrawal requests.

## Current Context

The project is a Fiber v2 REST API using GORM and PostgreSQL. Models use UUID primary keys and are migrated through `database.Migrate()` with `AutoMigrate`.

Relevant existing behavior:

- `shops` has a one-to-one owner relationship through `shops.user_id`.
- `orders` belongs to a shop through `orders.shop_id`.
- Orders currently move from `delivered` to `completed` by cron after 7 days without complaint.
- Payments are separate from orders through `payments.order_id`.
- Existing API responses use `utils.Success`, `utils.SuccessWithPagination`, and `utils.Error`.

## Chosen Approach

Use a current-balance wallet table, immutable wallet logs, and a separate withdrawal request table.

This keeps wallet balance reads fast, keeps financial history auditable, and keeps withdrawal workflow separate from balance movement history.

## Data Model

### ShopWallet

One wallet belongs to one shop.

Fields:

- `id uuid primary key`
- `shop_id uuid unique not null index`
- `pending_balance decimal(12,2) not null default 0`
- `available_balance decimal(12,2) not null default 0`
- `withdrawn_balance decimal(12,2) not null default 0`
- `created_at`
- `updated_at`

No soft delete is needed because a wallet is financial state tied to a shop. Deleting wallet state would create audit risk.

Wallets are created lazily — a wallet row is created on first order completion or first withdrawal request if one does not already exist for the shop. The repository provides a `FindOrCreateByShopID` method for this purpose. There is no explicit wallet creation endpoint.

### ShopWalletLog

Append-only record of every wallet balance movement.

Fields:

- `id uuid primary key`
- `wallet_id uuid not null index`
- `shop_id uuid not null index`
- `order_id uuid null index`
- `withdrawal_request_id uuid null index`
- `type varchar(50) not null index`
- `amount decimal(12,2) not null`
- `available_before decimal(12,2) not null`
- `available_after decimal(12,2) not null`
- `pending_before decimal(12,2) not null`
- `pending_after decimal(12,2) not null`
- `withdrawn_before decimal(12,2) not null`
- `withdrawn_after decimal(12,2) not null`
- `status varchar(20) not null`
- `description text`
- `metadata jsonb`
- `created_at`

`order_id` is required (not null) for `order_completed_pending` and `pending_released` log types. `withdrawal_request_id` is required for `withdrawal_*` log types.

Log types and their status values:

| Log type | Status value | Balances affected |
|---|---|---|
| `order_completed_pending` | `completed` | pending ↑ |
| `pending_released` | `completed` | pending ↓, available ↑ |
| `withdrawal_hold` | `pending` | available ↓ |
| `withdrawal_approved` | `approved` | withdrawn ↑ |
| `withdrawal_rejected` | `rejected` | available ↑ |

`status` records the source entity state at the time of the movement. For order-related logs, it records the order status (`completed`). For withdrawal-related logs, it records the withdrawal request status (`pending`/`approved`/`rejected`). Logs are not updated or deleted by service code.

### ShopWithdrawalRequest

Tracks a shop withdrawal workflow separately from wallet logs.

Fields:

- `id uuid primary key`
- `shop_id uuid not null index`
- `amount decimal(12,2) not null`
- `status varchar(20) not null index`
- `bank_info jsonb not null`
- `note text`
- `admin_note text`
- `reviewed_by uuid null index`
- `reviewed_at timestamptz null`
- `created_at`
- `updated_at`

Statuses:

- `pending`
- `approved`
- `rejected`

`bank_info` is a snapshot JSON object supplied at request time, for example `bank_name`, `account_number`, `account_holder`, and optional metadata. It is intentionally snapshotted so later bank detail changes do not alter historical withdrawal requests.

Validation: `bank_info` must contain at minimum `bank_name`, `account_number`, and `account_holder`. The service rejects requests with missing required fields as `400 VALIDATION_ERROR`.

## Wallet Lifecycle

### Credit Pending Balance

When an order becomes `completed`:

1. Calculate shop revenue as `TotalAmount - ShippingFee`.
2. Ensure the shop wallet exists.
3. In the same transaction as the order completion/status history update, lock the wallet row.
4. Add the revenue to `pending_balance`.
5. Insert a `shop_wallet_logs` row with type `order_completed_pending` and before/after balance snapshots.

The implementation should add `completed_at` to `orders` so release timing is based on the exact completion time. This avoids deriving completion time from status history text or cron execution time.

Idempotency requirement:

- Do not credit pending balance twice for the same order.
- Before crediting, check whether `shop_wallet_logs` already contains `type = order_completed_pending` for that `order_id`.

### Release Pending Balance

A wallet release cron moves held funds to withdrawable funds. The cron runs at 02:00 daily (same schedule as order auto-completion).

Eligibility:

- Order status is `completed`.
- `completed_at <= now - 7 days`.
- The order has an `order_completed_pending` wallet log.
- The order does not already have a `pending_released` wallet log.

For each eligible order, in a transaction:

1. Lock the wallet row.
2. Look up the `order_completed_pending` log for the order and read its `amount` (this is the revenue originally credited).
3. Verify `pending_balance >= amount`.
4. Subtract amount from `pending_balance`.
5. Add amount to `available_balance`.
6. Insert a `pending_released` log with before/after snapshots.

If pending balance is insufficient, fail the transaction and report the error. The service must not allow negative balances.

### Withdrawal Request

When a shop owner creates a withdrawal request:

1. Resolve the shop from `userID` in JWT through `shops.user_id`.
2. Validate `amount > 0` and `bank_info` is present.
3. Ensure wallet exists and lock the wallet row in a transaction.
4. Verify `available_balance >= amount`.
5. Create `shop_withdrawal_requests` with status `pending`.
6. Subtract amount from `available_balance` to hold funds immediately.
7. Insert `withdrawal_hold` log.

Holding funds immediately prevents multiple pending requests from overspending the same available balance.

### Withdrawal Approval

When an admin approves a pending withdrawal:

1. Lock the withdrawal request and wallet in a transaction.
2. Verify request status is `pending`.
3. Set status to `approved`, `reviewed_by`, and `reviewed_at`.
4. Add amount to `withdrawn_balance`.
5. Insert `withdrawal_approved` log.

Do not subtract available balance during approval because it was already held when the request was created.

### Withdrawal Rejection

When an admin rejects a pending withdrawal:

1. Lock the withdrawal request and wallet in a transaction.
2. Verify request status is `pending`.
3. Set status to `rejected`, `reviewed_by`, `reviewed_at`, and `admin_note`.
4. Add amount back to `available_balance`.
5. Insert `withdrawal_rejected` log.

## API Design

### Shop Owner Routes

Routes live under `/api/v1/shop` and use JWT auth. They resolve the shop from the token `userID`; clients do not pass `shop_id`.

- `GET /api/v1/shop/wallet`
  - Returns `pending_balance`, `available_balance`, `withdrawn_balance`.
  - If no wallet exists yet (shop has no completed orders), auto-create one with zero balances and return it. Never returns 404 for an existing shop.
- `GET /api/v1/shop/wallet/logs?page=&limit=&type=`
  - Returns paginated wallet logs for the current shop.
- `POST /api/v1/shop/withdrawals`
  - Body: `amount`, `bank_info`, optional `note`.
  - Creates a pending withdrawal and holds available balance immediately.
- `GET /api/v1/shop/withdrawals?page=&limit=&status=`
  - Returns paginated withdrawal requests for the current shop.
- `GET /api/v1/shop/withdrawals/:id`
  - Returns one withdrawal request if it belongs to the current shop.

### Admin Routes

- `GET /api/v1/admin/wallets?shop_id=&page=&limit=`
  - Permission: `wallet:read`.
- `GET /api/v1/admin/wallets/:shop_id/logs?page=&limit=&type=`
  - Permission: `wallet:read`.
- `GET /api/v1/admin/withdrawals?shop_id=&status=&page=&limit=`
  - Permission: `withdrawal:read`.
- `GET /api/v1/admin/withdrawals/:id`
  - Permission: `withdrawal:read`.
- `POST /api/v1/admin/withdrawals/:id/approve`
  - Permission: `withdrawal:write`.
- `POST /api/v1/admin/withdrawals/:id/reject`
  - Permission: `withdrawal:write`.
  - Body: optional `admin_note`.

## Permissions

Seed new permissions:

- `wallet:read`
- `wallet:write`
- `withdrawal:read`
- `withdrawal:write`

`super_admin` receives all permissions through the existing seed behavior. Do not add these permissions to `editor` or `viewer` by default because wallet and withdrawal actions are financial operations.

`wallet:write` is seeded for future wallet adjustment operations but is not used by this iteration. Manual admin wallet adjustment is out of scope.

## Components

New files:

- `internal/models/shop_wallet.go`
- `internal/repositories/shop_wallet_repo.go`
- `internal/services/shop_wallet_service.go`
- `internal/handlers/shop_wallet_handler.go`

Modified files:

- `internal/database/database.go`: add wallet models to `AutoMigrate`. Run migration SQL for existing completed orders (see [Migration](#migration)).
- `internal/models/order.go`: add `CompletedAt *time.Time`.
- `internal/repositories/order_repo.go`: set `completed_at` when completing orders. Support querying wallet-release-eligible completed orders (status = `completed`, `completed_at <= cutoff`, no `pending_released` log).
- `internal/services/order_service.go`: inject `ShopWalletService`, call it when an order becomes `completed`.
- `internal/cron`: accept wallet service dependency, run both order auto-completion and wallet pending-release jobs.
- `cmd/server/main.go`: wire repository, service, handler, routes, and seed permissions.

## Error Handling

Use existing response helpers and error conventions.

- No shop for current user: `404 NOT_FOUND`.
- Invalid amount or missing bank info: `400 VALIDATION_ERROR`.
- Insufficient available balance: `400 VALIDATION_ERROR`.
- Withdrawal request not owned by current shop: `404 NOT_FOUND`.
- Withdrawal request already processed: `400 VALIDATION_ERROR`.
- Unexpected database failure: `500 INTERNAL_ERROR` at handler boundary.

Service code must reject any operation that would make `pending_balance`, `available_balance`, or `withdrawn_balance` negative.

## Migration

When this feature is deployed to a database with existing completed orders, those orders will have `completed_at = NULL`. The wallet release cron only checks `completed_at <= now - 7 days`, so existing completed orders would never be released.

Run a one-time migration after `AutoMigrate` adds the `completed_at` column:

```sql
UPDATE orders SET completed_at = updated_at WHERE status = 'completed' AND completed_at IS NULL;
```

This sets `completed_at` to the order's last update timestamp for all existing completed orders, making them eligible for the release cron after the 7-day window from their original completion time.

Existing completed orders will NOT be credited to `pending_balance` automatically — only orders that transition to `completed` after the feature is deployed trigger the credit. Existing transactions before deployment are considered settled outside this system.

## Concurrency And Consistency

- All balance mutations run inside database transactions.
- Wallet row updates use row locking in PostgreSQL to prevent races between order completion, release cron, withdrawal creation, and admin review.
- Each balance mutation and its wallet log insert happen in the same transaction.
- Order credit and pending release operations are idempotent by checking existing wallet logs for the relevant `order_id` and log type.
- Withdrawal review is idempotency-protected by only allowing transitions from `pending`.

SQLite tests will verify business behavior, but PostgreSQL row locking cannot be fully simulated in SQLite. The repository should still express the intended lock behavior for production.

## Testing Plan

Add tests near the code they cover.

Model tests:

- Wallet log type constants.
- Withdrawal status constants.
- `CompletedAt` field on order.

Repository tests:

- Find or create wallet by shop.
- Query logs with pagination and type filter.
- Query withdrawal requests by shop/status.
- Find release-eligible completed orders.

Service tests:

- Completing an order credits `TotalAmount - ShippingFee` to pending balance.
- Completing the same order twice does not double-credit pending balance.
- Release cron moves eligible pending balance to available after 7 days from `completed_at`.
- Release cron does not release before 7 days.
- Release cron does not release twice.
- Creating a withdrawal holds available balance and creates a pending request.
- Approving a withdrawal increments withdrawn balance and does not subtract available twice.
- `withdrawal_approved` log captures `withdrawn_before` and `withdrawn_after` correctly.
- Rejecting a withdrawal returns held amount to available balance.
- Insufficient funds fails without creating a request or log.

Handler tests are optional for this iteration because existing coverage is concentrated in model, repository, and service layers. Add handler tests if the route wiring is straightforward after service coverage is in place.

## Out Of Scope

- External bank transfer integration.
- Shop bank account management table.
- Partial approval of withdrawal requests.
- Manual admin wallet adjustment endpoint.
- Customer-facing wallet or refund flows.
