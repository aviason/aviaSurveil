package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/telemetry"
)

type reminderScheduler interface {
	ScheduleDueReminders(context.Context) (int, error)
}

type reminderControllerConfig struct {
	Interval time.Duration
	Deadline time.Duration
	Ticks    <-chan time.Time
	Schedule reminderScheduler
	OnCycle  func(processed int, err error)
}

// runReminderController owns reminder cadence independently from the serial
// evidence/document/notification processors. A cycle is synchronous, so a
// queued tick can never overlap an active cycle; cancellation reaches a hung
// database cycle through its bounded child context.
func runReminderController(ctx context.Context, config reminderControllerConfig) {
	if config.Schedule == nil {
		return
	}
	interval := config.Interval
	if interval <= 0 {
		interval = time.Minute
	}
	deadline := config.Deadline
	if deadline <= 0 || deadline > interval {
		deadline = interval
	}
	ticks := config.Ticks
	var ticker *time.Ticker
	if ticks == nil {
		ticker = time.NewTicker(interval)
		defer ticker.Stop()
		ticks = ticker.C
	}

	cycle := func() {
		cycleContext, cancel := context.WithTimeout(ctx, deadline)
		processed, err := config.Schedule.ScheduleDueReminders(cycleContext)
		cancel()
		if config.OnCycle != nil {
			config.OnCycle(processed, err)
			return
		}
		if err != nil {
			slog.Warn("reminder scheduling cycle failed", "processed", processed, "errorClass", telemetry.ErrorClass(err))
		} else if processed > 0 {
			slog.Info("reminder scheduling cycle completed", "processed", processed)
		}
	}

	// Startup execution makes a restarted worker converge without waiting for
	// the first wall-clock tick.
	cycle()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
			cycle()
		}
	}
}
