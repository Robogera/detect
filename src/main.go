package main

import (
	// stdlib
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	// internal
	"github.com/Robogera/detect/pkg/config"
	"github.com/Robogera/detect/pkg/rpath"

	// external
	"github.com/lmittmann/tint"
	"golang.org/x/sync/errgroup"
)

const (
	default_cfg_path string = "../cfg/config.default.toml"
)

var cfg_path string
var exe_dir string

func init() {
	// I have to this or compiler goes crazy on the next line YIKES!
	var err error

	exe_dir, err = rpath.ExecutableDir()
	if err != nil {
		slog.Error("Can't find the executable's location", "error", err)
		return
	}

	flag.StringVar(
		&cfg_path, "config",
		default_cfg_path,
		"Path to config file")
}

func main() {

	// Configuration init

	flag.Parse()

	cfg, err := config.Unmarshal(cfg_path)
	if err != nil {
		slog.Error("Config file not loaded. Shutting down...", "provided path", cfg_path, "error", err)
		return
	}
	slog.Info("Config file loaded", "provided path", cfg_path)

	var log_level slog.Level

	switch config.LoggingLevel(cfg.Logging.Level) {
	case config.LoggingLevelDebug:
		log_level = slog.LevelDebug
	case config.LoggingLevelInfo:
		log_level = slog.LevelInfo
	case config.LoggingLevelWarn:
		log_level = slog.LevelWarn
	case config.LoggingLevelError:
		log_level = slog.LevelError
	default:
		slog.Warn(
			"No valid logging level provided. Defaulting to LevelError",
			"provided value", cfg.Logging.Level)
		log_level = slog.LevelError
	}

	logger := slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:      log_level,
		TimeFormat: time.RFC3339,
		AddSource:  true, // change to false on release version
	}))

	logger.Info("Starting...")

	ctx := context.Background()
	eg, child_ctx := errgroup.WithContext(ctx)

	// TODO: try buffering
	frames_chan := make(chan []byte, 8)

  stat_chan := make(chan Statistics, 8)

	eg.Go(func() error {
		return webplayer(child_ctx, logger, cfg, frames_chan)
	})

	eg.Go(func() error {
		return processor(child_ctx, logger, cfg, frames_chan, stat_chan)
	})

	eg.Go(func() error {
		return stat(
			child_ctx, logger, cfg, stat_chan)
	})

	eg.Go(func() error {
		return control(child_ctx, logger)
	})

	eg.Wait()

	logger.Info("All subroutines returned")
}

func control(ctx context.Context, logger *slog.Logger) error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt,
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT)

	select {
	case <-ctx.Done():
		logger.Info("Control cancelled by context")
		return context.Canceled
	case <-interrupt:
		logger.Info("Cancelled by user")
		return ERR_INTERRUPTED_BY_USER
	}
}
