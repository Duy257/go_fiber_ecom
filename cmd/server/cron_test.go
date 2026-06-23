package main

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

func TestStartOrderCompletionCronRegistersOneJob(t *testing.T) {
	cronRunner, err := startOrderCompletionCron(nil)
	if err != nil {
		t.Fatalf("startOrderCompletionCron returned error: %v", err)
	}
	defer cronRunner.Stop()

	entries := cronRunner.Entries()
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
}
