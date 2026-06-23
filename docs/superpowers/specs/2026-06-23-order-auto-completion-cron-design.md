# Thiết Kế Cron Tự Động Hoàn Tất Đơn Hàng

## Mục tiêu

Tự động chuyển đơn hàng từ `delivered` sang `completed` sau 7 ngày nếu đơn hàng không bị khiếu nại.

Cron job dùng thư viện `github.com/robfig/cron/v3` và chạy mỗi ngày lúc `02:00` với cron expression cố định `0 2 * * *`.

> **Yêu cầu triển khai**: Thêm `github.com/robfig/cron/v3` vào `go.mod`.

## Bối cảnh hiện tại

Project là backend Go Fiber dùng GORM và PostgreSQL. Luồng đơn hàng hiện nằm ở các file chính:

- `internal/models/order.go`: định nghĩa `Order`, `OrderItem`, `OrderStatusHistory`
- `internal/services/order_service.go`: xử lý tạo đơn, hủy đơn, cập nhật trạng thái
- `internal/repositories/order_repo.go`: query và lưu dữ liệu đơn hàng
- `cmd/server/main.go`: khởi tạo app và wiring dependency

Hiện chưa có model khiếu nại riêng. Trong phạm vi này, trạng thái khiếu nại được biểu diễn trực tiếp trên bảng `orders`.

## Hướng thiết kế đã chọn

Dùng cron job mỏng, gọi vào tầng service để xử lý nghiệp vụ:

- `cmd/server/main.go` khởi tạo robfig/cron sau khi tạo `OrderService`.
- Lịch chạy cố định trong code là `0 2 * * *`, tức `02:00` mỗi ngày.
- Cron handler gọi method mới `OrderService.AutoCompleteDeliveredOrders()`.
- Service gọi repository để tìm và cập nhật các đơn đủ điều kiện.
- Mỗi đơn được tự động hoàn tất sẽ có thêm một bản ghi `OrderStatusHistory`.

Cách này giữ phần lập lịch tách khỏi nghiệp vụ và bám sát cấu trúc service/repository hiện tại.

## Thay đổi model

Thêm 2 field vào `Order`:

- `DeliveredAt *time.Time \`gorm:"type:timestamptz" json:"delivered_at"\``: thời điểm đơn được chuyển sang `delivered`.
- `HasComplaint bool \`gorm:"default:false" json:"has_complaint"\``: mặc định `false`; nếu là `true`, cron sẽ bỏ qua đơn hàng.

> **Yêu cầu triển khai**: Cần GORM auto-migration (hoặc migration script) để thêm 2 cột `delivered_at` và `has_complaint` vào bảng `orders`.

Thêm hằng trạng thái:

- `OrderStatusCompleted = "completed"`

Trong worktree hiện có thay đổi chưa commit ở `internal/models/order.go` với lỗi trùng tên `OrderStatusDelivered` cho cả `"delivered"` và `"completed"` (dòng 15: `OrderStatusDelivered = "completed"`). **Khi triển khai**: rename dòng 15 thành `OrderStatusCompleted = "completed"`.

## Luồng trạng thái

Vòng đời đơn hàng sau thay đổi:

```text
pending -> confirmed -> shipping -> delivered -> completed
                      \-> cancelled
```

Quy tắc:

- Cập nhật thủ công vẫn cho phép chuyển `shipping -> delivered`.
- Khi status mới là `delivered`, service set `delivered_at = now` trong `UpdateStatus` (thêm vào block `if input.Status == "delivered"` hiện có, cùng transaction với `MarkAsPaid`).
- `delivered -> completed` **KHÔNG** được thêm vào `validTransitions` trong `UpdateStatus` — chỉ cron job thực hiện chuyển trạng thái này (qua method riêng `AutoCompleteDeliveredOrders`).
- Đơn có `has_complaint = true` sẽ giữ ở trạng thái `delivered`.
- Nếu sau này `has_complaint` được đổi lại thành `false`, cron kế tiếp có thể hoàn tất đơn nếu `delivered_at` đã quá 7 ngày (không restart timer).

## Hành vi cron

Mỗi ngày lúc `02:00`, cron chọn các đơn thỏa tất cả điều kiện:

- `status = delivered`
- `delivered_at <= now - 7 days` (SQL: `delivered_at <= CURRENT_TIMESTAMP - INTERVAL '7 days'`; Go: `time.Now().Add(-7 * 24 * time.Hour)`)
- `has_complaint = false`

Với mỗi đơn đủ điều kiện:

- cập nhật `status` thành `completed`
- tạo `OrderStatusHistory` với status `completed`
- dùng note hệ thống: `Auto-completed after 7 days without complaint`

Job có tính idempotent vì đơn đã `completed` không còn thỏa điều kiện `status = delivered`.

## Luồng dữ liệu

Luồng giao hàng thủ công:

```text
Admin PUT /admin/orders/:id/status
-> OrderHandler.UpdateStatus
-> OrderService.UpdateStatus
-> set status = delivered
-> set delivered_at = now
-> create OrderStatusHistory
-> giữ nguyên xử lý payment hiện tại
```

Luồng hoàn tất tự động:

```text
02:00 daily cron
-> OrderService.AutoCompleteDeliveredOrders
-> OrderRepository tìm các đơn delivered đủ điều kiện
-> service cập nhật từng đơn trong transaction riêng (per-order transaction, không gom chung một transaction)
-> service ghi OrderStatusHistory
```

## Xử lý lỗi và logging

Lỗi cron không được làm dừng HTTP server.

- Nếu cron task trả lỗi, log bằng `log.Printf`.
- Nếu một đơn lỗi trong transaction, rollback thay đổi của riêng đơn đó.
- Lần chạy cron kế tiếp có thể retry các đơn vẫn còn thỏa điều kiện.
- Vì cron expression cố định và hợp lệ, lỗi đăng ký lịch là lỗi bất thường và cần được log khi server khởi động.
- **Hành vi khi cron khởi động thất bại**: `cron.Start()` trả về error; nếu lỗi này xảy ra, server nên `log.Fatalf` để không chạy HTTP server thiếu cron job quan trọng.

## Testing

Test nên tập trung vào service và repository, không cần test nội bộ robfig/cron.

Các case bắt buộc:

- Đơn `delivered` có `delivered_at` cũ hơn 7 ngày và `has_complaint=false` được chuyển sang `completed`.
- Đơn `delivered` chưa đủ 7 ngày vẫn giữ `delivered`.
- Đơn `delivered` có `has_complaint=true` vẫn giữ `delivered`.
- Đơn đã `completed` không bị xử lý lại.
- Khi cập nhật đơn sang `delivered`, hệ thống set `delivered_at`.
- Auto-completion ghi `OrderStatusHistory`.

Có thể kiểm tra thủ công bằng dữ liệu seed hoặc test database với `delivered_at` được set sẵn. Lịch production vẫn cố định `02:00`.

## Ngoài phạm vi

- API hoặc workflow đầy đủ cho khiếu nại.
- Bảng riêng `order_complaints`.
- Cấu hình cron schedule qua `.env`.
- Thay đổi UI cho admin/customer.
