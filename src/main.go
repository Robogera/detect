package main

import (
	// stdlib
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// internal
	"github.com/Robogera/detect/pkg/config"
	"github.com/Robogera/detect/pkg/rpath"

	// external
	"github.com/hybridgroup/mjpeg"
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
  frames_chan := make(chan []byte)

	eg.Go(func() error {
		return webserver(
			child_ctx, logger, output_stream, cfg.Webserver.Port,
			cfg.Webserver.ReadTimeoutSec,
			cfg.Webserver.WriteTimeoutSec,
			cfg.Webserver.ShutdownTimeoutSec)
	})

	eg.Go(func() error {
		// return video(
		// 	child_ctx, logger,
		// 	rpath.Convert(exe_dir, cfg.Input.Path),
		// 	rpath.Convert(exe_dir, cfg.Model.Path),
		// 	output_stream, stats_chan)
    return processor(
    child_ctx, logger, cfg, frames_chan)
	})

	// eg.Go(func() error {
	// 	return stat(
	// 		child_ctx, logger, stats_chan,
	// 		cfg.Logging.StatPeriodSec)
	// })

	eg.Go(func() error {
		return control(child_ctx, logger)
	})

	eg.Wait()

	logger.Info("Stopped")
}

func stat(ctx context.Context, logger *slog.Logger, stats <-chan struct{}, stat_period_sec uint) error {
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
