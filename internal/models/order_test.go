package models

import (
	"testing"
	"time"
)

func TestOrderStatusCompletedConstant(t *testing.T) {
	if OrderStatusDelivered != "delivered" {
		t.Fatalf("OrderStatusDelivered = %q, want delivered", OrderStatusDelivered)
	}

	if OrderStatusCompleted != "completed" {
		t.Fatalf("OrderStatusCompleted = %q, want completed", OrderStatusCompleted)
	}
}

func TestOrderCompletionFields(t *testing.T) {
	deliveredAt := time.Now().UTC()
	order := Order{
		DeliveredAt:  &deliveredAt,
		HasComplaint: true,
	}

	if order.DeliveredAt == nil {
		t.Fatal("DeliveredAt is nil, want timestamp pointer")
	}

	if !order.DeliveredAt.Equal(deliveredAt) {
		t.Fatalf("DeliveredAt = %v, want %v", order.DeliveredAt, deliveredAt)
	}

	if !order.HasComplaint {
		t.Fatal("HasComplaint = false, want true")
	}
}
