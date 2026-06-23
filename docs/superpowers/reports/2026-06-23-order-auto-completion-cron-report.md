# Order Auto-Completion Cron — Implementation Report

**Date:** 2026-06-23
**Branch:** `main`
**Plan:** `docs/superpowers/plans/2026-06-23-order-auto-completion-cron.md`

## Summary

Tự động chuyển đơn hàng từ `delivered` sang `completed` sau 7 ngày nếu đơn không bị khiếu nại. Cron chạy lúc 02:00 mỗi ngày.

## Changes

### Commits (5 commits)

| Commit | Message |
|--------|---------|
| `5369a44` | feat: add order completion fields |
| `4518c46` | feat: set delivered timestamp on order delivery |
| `9849cd8` | feat: add order auto-completion repository queries |
| `98b1c22` | feat: auto-complete delivered orders |
| `4c23f45` | feat: schedule order auto-completion cron |

### Files Modified/Created

| File | Action | Description |
|------|--------|-------------|
| `go.mod` / `go.sum` | Modified | Added `robfig/cron/v3`, `glebarez/sqlite`, `modernc.org/sqlite` |
| `internal/models/order.go` | Modified | Fixed duplicate `OrderStatusDelivered`, added `OrderStatusCompleted`, `DeliveredAt`, `HasComplaint` |
| `internal/models/order_test.go` | **New** | Tests for constants & new fields |
| `internal/services/order_service.go` | Modified | `UpdateStatus` sets `delivered_at`; added `AutoCompleteDeliveredOrders()`, `AutoCompleteDeliveredOrdersBefore()` |
| `internal/services/order_service_test.go` | **New** | Tests for delivered_at behavior & auto-completion logic |
| `internal/repositories/order_repo.go` | Modified | Added `FindAutoCompletableDelivered()`, `CompleteDeliveredOrder()` |
| `internal/repositories/order_repo_test.go` | **New** | Tests for repository query & update methods |
| `cmd/server/cron.go` | **New** | `startOrderCompletionCron()` — wraps `robfig/cron` with daily `0 2 * * *` schedule |
| `cmd/server/cron_test.go` | **New** | Tests for cron spec parsing & job registration |
| `cmd/server/main.go` | Modified | Wired cron startup after `OrderService` creation |

## Verification

- **`go test ./...`** — PASS
- **`go build ./...`** — PASS
- **`go fmt ./...`** — OK

## Architecture

```
cron (02:00 daily)
  └─ OrderService.AutoCompleteDeliveredOrders()
       └─ OrderRepository.FindAutoCompletableDelivered(cutoff)
            └─ WHERE status='delivered' AND delivered_at <= cutoff AND has_complaint=false
       └─ for each order: Transaction()
            ├─ OrderRepository.CompleteDeliveredOrder(tx, id)
            │    └─ UPDATE orders SET status='completed' WHERE id=? AND status='delivered'
            └─ INSERT order_status_history (status='completed', note='Auto-completed after 7 days without complaint')
```

## Key Design Decisions

- `completed` status **not** added to `validTransitions` in `UpdateStatus` — only cron can auto-complete
- Mỗi đơn xử lý trong transaction riêng (per-order isolation)
- Dùng `glebarez/sqlite` (CGO-free) cho unit/integration tests thay vì `gorm.io/driver/sqlite` (cần CGO/gcc)
