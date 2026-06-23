package cron

import (
	"testing"
	"time"

	"github.com/robfig/cron/v3"
)

func TestOrderCompletionCronSpecRunsAtTwoAM(t *testing.T) {
	schedule, err := cron.ParseStandard(orderCompletionCronSpec)
	if err != nil {
		t.Fatalf("ParseStandard returned error: %v", err)
	}

	next := schedule.Next(time.Date(2026, 6, 23, 0, 0, 0, 0, time.Local))
	if next.Hour() != 2 || next.Minute() != 0 {
		t.Fatalf("next run = %v, want 02:00", next)
	}
}

func TestManagerStartRegistersOneJob(t *testing.T) {
	manager := NewManager(nil)
	err := manager.Start()
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer manager.Stop()

	entries := manager.cronRunner.Entries()
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
}
