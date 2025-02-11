package main

import (
	// stdlib
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	// internal
	"github.com/Robogera/detect/pkg/config"
	"github.com/Robogera/detect/pkg/indexed"

	// external
	"gocv.io/x/gocv"
)

func debug_streamreader(
	ctx context.Context,
	logger *slog.Logger,
	cfg *config.ConfigFile,
	mat_chan chan<- indexed.Indexed[gocv.Mat],
) error {

	dir, err := os.Open(cfg.Input.Path)
	if err != nil {
		logger.Error("Can't open debug folder %s", "error", cfg.Input.Path)
		return err
	}

	files, err := dir.Readdir(-1)
	if err != nil {
		logger.Error("Can't read debug folder %s", "error", cfg.Input.Path)
		return err
	}

	names := make([]string, 0)
	for _, f := range files {
		if strings.Contains(f.Name(), "debug.jpg") {
			names = append(names, filepath.Join(cfg.Input.Path, f.Name()))
		}
	}

	slices.Sort(names)

	logger.Info("Debug mode", "images", names)

	var frame_id uint64 = 0

	counter := 0

	for {
		select {
		case <-ctx.Done():
			logger.Info("Streamreader cancelled by context")
			return context.Canceled
		default:
			img := gocv.IMRead(names[counter], gocv.IMReadColor)
			counter++
			if counter >= len(names) {
				counter = 0
			}

			if img.Empty() {
				logger.Error("Empty frame received, skipping", "stream", cfg.Input.Path)
				img.Close()
				continue
			}

			select {
			case <-ctx.Done():
				logger.Info("Streamreader cancelled by context")
				return context.Canceled
			case mat_chan <- indexed.NewIndexed(frame_id, time.Now(), img):
				frame_id++
			}
		}
	}
}
