package main

import (
	"log"

	"go-fiber/internal/services"

	"github.com/robfig/cron/v3"
)

const orderCompletionCronSpec = "0 2 * * *"

func startOrderCompletionCron(orderService *services.OrderService) (*cron.Cron, error) {
	cronRunner := cron.New()

	_, err := cronRunner.AddFunc(orderCompletionCronSpec, func() {
		if orderService == nil {
			log.Printf("order completion cron skipped: order service is nil")
			return
		}

		completedCount, err := orderService.AutoCompleteDeliveredOrders()
		if err != nil {
			log.Printf("order completion cron failed: %v", err)
			return
		}

		log.Printf("order completion cron completed %d orders", completedCount)
	})
	if err != nil {
		return nil, err
	}

	cronRunner.Start()
	return cronRunner, nil
}
