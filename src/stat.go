package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/Robogera/detect/pkg/config"
)

// WIP
type Statistics struct {
	inference_time time.Duration
}

// TODO: expand, add max_wait_time and sliding average for FPS
func stat(ctx context.Context, logger *slog.Logger, cfg *config.ConfigFile, stat_chan <-chan Statistics) error {
	var frames uint = 0
	var frames_since_last_tick uint = 0
	var cum_avg float64 = 0.0
	ticker := time.NewTicker(time.Second * time.Duration(cfg.Logging.StatPeriodSec))
	for {
		select {
		case <-ctx.Done():
			logger.Info("Stat cancelled by context")
			return context.Canceled
		case stats := <-stat_chan:
			cum_avg = (stats.inference_time.Seconds() + float64(frames_since_last_tick)*cum_avg) / (float64(frames_since_last_tick) + 1)
			frames++
			frames_since_last_tick++
		case <-ticker.C:
			logger.Info("Stats", "frames processed", frames, "average inference time", cum_avg)
			frames_since_last_tick = 0
		}
	}
}
