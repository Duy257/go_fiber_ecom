package cron

import (
	"log"

	"go-fiber/internal/services"

	"github.com/robfig/cron/v3"
)

// orderCompletionCronSpec định nghĩa lịch chạy cron để tự động hoàn thành đơn hàng đã giao.
// Chạy lúc 02:00 sáng mỗi ngày.
const orderCompletionCronSpec = "0 2 * * *"

// Manager quản lý tất cả cron jobs trong ứng dụng.
type Manager struct {
	orderService *services.OrderService
	cronRunner   *cron.Cron
}

// NewManager tạo một cron Manager mới với các dependency được truyền vào.
func NewManager(orderService *services.OrderService) *Manager {
	return &Manager{
		orderService: orderService,
		cronRunner:   cron.New(),
	}
}

// Start đăng ký tất cả cron jobs và khởi động cron scheduler.
func (m *Manager) Start() error {
	_, err := m.cronRunner.AddFunc(orderCompletionCronSpec, func() {
		if m.orderService == nil {
			log.Printf("order completion cron skipped: order service is nil")
			return
		}

		completedCount, err := m.orderService.AutoCompleteDeliveredOrders()
		if err != nil {
			log.Printf("order completion cron failed: %v", err)
			return
		}

		log.Printf("order completion cron completed %d orders", completedCount)
	})
	if err != nil {
		return err
	}

	m.cronRunner.Start()
	return nil
}

// Stop dừng cron scheduler một cách an toàn.
func (m *Manager) Stop() {
	m.cronRunner.Stop()
}
