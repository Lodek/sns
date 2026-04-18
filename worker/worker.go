package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/lodek/sns/notify"
	"github.com/lodek/sns/store"
)

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

type Worker struct {
	store     *store.Store
	notifiers []notify.Notifier
	interval  time.Duration
}

func New(s *store.Store, notifiers []notify.Notifier, interval time.Duration) *Worker {
	return &Worker{
		store:     s,
		notifiers: notifiers,
		interval:  interval,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	slog.Info("worker started", "interval", w.interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Run once immediately on startup.
	w.tick(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("worker stopped")
			return ctx.Err()
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *Worker) tick(ctx context.Context) {
	now := time.Now().UTC().Truncate(time.Minute)
	slog.Debug("worker tick", "now", now)

	w.processOneShots(ctx, now)
	w.processRecurring(ctx, now)
}

func (w *Worker) processOneShots(ctx context.Context, now time.Time) {
	alerts, err := w.store.ListOneShotAlerts(ctx)
	if err != nil {
		slog.Error("list oneshot alerts", "error", err)
		return
	}
	for _, alert := range alerts {
		fireAt := alert.FireAt.AsTime().Truncate(time.Minute)
		if !fireAt.After(now) {
			msg := formatMessage("one-shot", alert.Name, alert.Message)
			w.fanOut(ctx, msg)
			if err := w.store.DeleteOneShotAlert(ctx, alert.Id); err != nil {
				slog.Error("delete oneshot alert", "id", alert.Id, "error", err)
			}
		}
	}
}

func (w *Worker) processRecurring(ctx context.Context, now time.Time) {
	alerts, err := w.store.ListRecurringAlerts(ctx)
	if err != nil {
		slog.Error("list recurring alerts", "error", err)
		return
	}
	for _, alert := range alerts {
		sched, err := cronParser.Parse(alert.CronExpression)
		if err != nil {
			slog.Error("parse cron", "id", alert.Id, "expr", alert.CronExpression, "error", err)
			continue
		}
		// If the next fire time after (now - 1 minute) equals now, the cron matches.
		nextFire := sched.Next(now.Add(-time.Minute))
		if nextFire.Equal(now) {
			msg := formatMessage("recurring", alert.Name, alert.Message)
			w.fanOut(ctx, msg)
		}
	}
}

func (w *Worker) fanOut(ctx context.Context, message string) {
	for _, n := range w.notifiers {
		if err := n.Send(ctx, message); err != nil {
			slog.Error("send notification", "backend", n.Name(), "error", err)
		}
	}
}

func formatMessage(alertType, name, message string) string {
	return fmt.Sprintf("[%s] %s: %s", alertType, name, message)
}
