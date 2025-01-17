package main

import (
	"context"
	"log/slog"
	"time"
)

// WIP
type Statistics struct {
  inference_time time.Duration
}

// TODO: expand, add max_wait_time and sliding average for FPS
func stat(ctx context.Context, logger *slog.Logger, stats <-chan Statistics, stat_period_sec uint) error {
	var frames uint = 0
	var frames_since_last_tick uint = 0
	ticker := time.NewTicker(time.Second * time.Duration(stat_period_sec))
	for {
		select {
		case <-ctx.Done():
			logger.Info("Stat cancelled by context")
			return context.Canceled
		case <-stats:
			frames++
			frames_since_last_tick++
		case <-ticker.C:
			logger.Info("Stats", "frames processed", frames, "frames per second", frames_since_last_tick/stat_period_sec)
			frames_since_last_tick = 0
		}
	}
}
