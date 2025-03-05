package main

import (
	// stdlib
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	// internal
	"github.com/Robogera/detect/pkg/config"
	"github.com/Robogera/detect/pkg/indexed"
	"github.com/Robogera/detect/pkg/person"
	"github.com/Robogera/detect/pkg/rpath"
	"gocv.io/x/gocv"

	// external
	"github.com/lmittmann/tint"
	"golang.org/x/sync/errgroup"
)

const (
	default_cfg_path string = "../cfg/config.default.toml"
)

var cfg_path string
var exe_dir string
var create_default_config bool
var migrate_config bool

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

	flag.BoolVar(
		&create_default_config, "create",
		false,
		"Creates default config file in location specified in -config flag (or a default location if not specified)")

	flag.BoolVar(
		&migrate_config, "migrate",
		false,
		"Migrate config")
}

func main() {

	// Configuration init

	flag.Parse()

	logger := slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelInfo,
		TimeFormat: time.RFC3339,
		AddSource:  true,
	}))

	cfg_abs_path := filepath.Join(exe_dir, cfg_path)

	if create_default_config {
		if _, err := os.Stat(cfg_abs_path); os.IsNotExist(err) {
			err := config.CreateDefault(cfg_abs_path)
			if err != nil {
				slog.Error("Can't write default config file", "path", cfg_abs_path, "error", err)
			}
		} else {
			slog.Error("Will not overwrite existing file", "path", cfg_abs_path)
		}
		return
	} else if migrate_config {
		err := config.Migrate(cfg_abs_path)
		if err != nil {
			slog.Error("Can't migrate config file", "path", cfg_abs_path, "error", err)
		}
		return
	}

	cfg, err := config.Unmarshal(cfg_abs_path)
	if err != nil {
		slog.Error("Config file not loaded. Shutting down...", "provided path", cfg_abs_path, "error", err)
		return
	}
	slog.Info("Config file loaded", "provided path", cfg_abs_path)

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

	logger = slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:      log_level,
		TimeFormat: time.RFC3339,
		AddSource:  true, // change to false on release version
	}))

	logger.Info("Starting...")

	ctx := context.Background()
	eg, child_ctx := errgroup.WithContext(ctx)

	// TODO: try buffering
	unsorted_frames_chan := make(chan indexed.Indexed[ProcessedFrame], 8)
	sorted_frames_chan := make(chan indexed.Indexed[ProcessedFrame], 8)
	ident_frames_chan := make(chan indexed.Indexed[ProcessedFrame], 8)

	stat_chan := make(chan Statistics, 8)

	mat_chan := make(chan indexed.Indexed[*gocv.Mat], 8)

	export_chan := make(chan indexed.Indexed[[]*person.ExportedPerson], 8)

	eg.Go(func() error {
		return streamreader(child_ctx, logger, cfg, mat_chan)
	})

	for i := 0; i < int(cfg.Yolo.Threads); i++ {
		eg.Go(func() error {
			return detector(child_ctx, logger, cfg, mat_chan, unsorted_frames_chan)
		})
	}

	eg.Go(func() error {
		return sorter(child_ctx, logger, cfg, unsorted_frames_chan, sorted_frames_chan)
	})

	eg.Go(func() error {
		return reidentificator(child_ctx, logger, cfg, sorted_frames_chan, ident_frames_chan, export_chan)
	})

	eg.Go(func() error {
		return mqttclient(child_ctx, logger, cfg, export_chan)
	})

	eg.Go(func() error {
		return webplayer(child_ctx, logger, cfg, ident_frames_chan, stat_chan)
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
