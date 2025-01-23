package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/Robogera/detect/pkg/config"
	"github.com/Robogera/detect/pkg/gsma"
)

// WIP
type Statistics struct {
	inference_time time.Duration
}

// TODO: expand, add max_wait_time and sliding average for FPS
func stat(ctx context.Context, logger *slog.Logger, cfg *config.ConfigFile, stat_chan <-chan Statistics) error {
	sma, err := gsma.NewSMA[float64](100)
	if err != nil {
		logger.Error("Can't init an SMA accumulator", "error", err)
	}
	ticker := time.NewTicker(time.Second * time.Duration(cfg.Logging.StatPeriodSec))
	for {
		select {
		case <-ctx.Done():
			logger.Info("Stat cancelled by context")
			return context.Canceled
		case stats := <-stat_chan:
			sma.Recalc(stats.inference_time.Seconds())
		case <-ticker.C:
			logger.Info("Stats", "frame time SMA (sec)", sma.Show(), "avg FPS", 1.0/sma.Show())
		}
	}
}
