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
	"github.com/Robogera/detect/pkg/enums"
	"github.com/Robogera/detect/pkg/rpath"

	// external
	"github.com/hybridgroup/mjpeg"
	"github.com/lmittmann/tint"
	"gocv.io/x/gocv"
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

	// kill me
	switch *enums.LoggingLevels.Parse(cfg.Logging.Level) {
	case enums.LoggingLevelDebug:
		log_level = slog.LevelDebug
	case enums.LoggingLevelInfo:
		log_level = slog.LevelInfo
	case enums.LoggingLevelWarn:
		log_level = slog.LevelWarn
	case enums.LoggingLevelError:
		log_level = slog.LevelError
	default:
		slog.Warn(
			"No valid logging level provided. Defaulting to LevelError",
			"provided value", cfg.Logging.Level,
			"valid values", enums.LoggingLevels.Values())
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

	stats_chan := make(chan struct{}, 8)

	output_stream := mjpeg.NewStream()

	eg.Go(func() error {
		return webserver(
			child_ctx, logger, output_stream, cfg.Webserver.Port,
			cfg.Webserver.ReadTimeoutSec,
			cfg.Webserver.WriteTimeoutSec,
			cfg.Webserver.ShutdownTimeoutSec)
	})

	eg.Go(func() error {
		return video(
			child_ctx, logger,
			rpath.Convert(exe_dir, cfg.Input.Path),
			rpath.Convert(exe_dir, cfg.Model.Path),
			output_stream, stats_chan)
	})

	eg.Go(func() error {
		return stat(
			child_ctx, logger, stats_chan,
			cfg.Logging.StatPeriodSec)
	})

	eg.Go(func() error {
		return control(child_ctx, logger)
	})

	eg.Wait()

	logger.Info("Stopped")
}

func webserver(
	ctx context.Context,
	logger *slog.Logger,
	stream *mjpeg.Stream,
	port uint,
	read_timeout_sec, write_timeout_sec, shutdown_timeout_sec uint,
) error {

	http.Handle("/", stream)

	server := &http.Server{
		Addr:         fmt.Sprintf("127.0.0.1:%d", port),
		ReadTimeout:  time.Duration(read_timeout_sec) * time.Second,
		WriteTimeout: time.Duration(write_timeout_sec) * time.Second,
	}

	err_chan := make(chan error)

	go func() {
		err_chan <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logger.Info("Webserver cancelled by context. Shutting down...", "timeout (sec)", shutdown_timeout_sec)
		shutdown_context, cancel := context.WithTimeout(
			context.Background(),
			time.Second*time.Duration(shutdown_timeout_sec))
		defer cancel()
		shutdown_initiated_timestamp := time.Now()
		err := server.Shutdown(shutdown_context)
		logger.Info(
			"Webserver shut down successfully", "shutdown duration (sec)",
			time.Now().Sub(shutdown_initiated_timestamp).Seconds(), "error", err)
		return ERR_CANCELLED_BY_CONTEXT
	case err := <-err_chan:
		logger.Error("Server error", "port", port, "error", err)
		return err
	}
}

func video(
	ctx context.Context,
	logger *slog.Logger,
	file_path string,
	model_path string,
	output_stream *mjpeg.Stream,
	stats chan<- struct{}) error {

	logger.Info("Opening file", "address", file_path)

	input_stream, err := gocv.VideoCaptureFile(file_path)
	if err != nil {
		logger.Error("Can't open stream", "address", file_path, "err", err)
		return ERR_BAD_STREAM
	}
	defer input_stream.Close()

	net := gocv.ReadNetFromONNX(model_path)
	if net.Empty() {
		logger.Error("Error reading network model")
		return ERR_BAD_MODEL
	}
	defer net.Close()

	outputNames := getOutputNames(&net)
	if len(outputNames) == 0 {
		logger.Error("Error reading output layer names")
		return ERR_BAD_MODEL
	} else {
		for _, name := range outputNames {
			logger.Info("Model info", "outputNames", name)
		}
	}

	if err := net.SetPreferableBackend(gocv.NetBackendType(gocv.NetBackendOpenVINO)); err != nil {
		return ERR_CANT_SET_BACKEND
	}
	if err := net.SetPreferableTarget(gocv.NetTargetType(gocv.NetTargetCPU)); err != nil {
		return ERR_CANT_SET_TARGET
	}

	img := gocv.NewMat()
	defer img.Close()

	logger.Info("Video loop started")
	for {
		select {
		case <-ctx.Done():
			logger.Info("Video cancelled by context")
			return ERR_CANCELLED_BY_CONTEXT
		default:
			if !input_stream.Read(&img) {
				logger.Error("Can't read next frame", "stream", file_path)
				return ERR_STREAM_ENDED
			}
			if img.Empty() {
				logger.Error("Empty frame received, skipping", "stream", file_path)
				continue
			}

			detect(&net, &img, outputNames)

			stats <- struct{}{}
			buf, err := gocv.IMEncode(".jpg", img)
			if err != nil {
				logger.Error("Can't encode frame")
				return err
			}
			output_stream.UpdateJPEG(buf.GetBytes())
			buf.Close()
		}
	}
}

func stat(ctx context.Context, logger *slog.Logger, stats <-chan struct{}, stat_period_sec uint) error {
	var frames uint = 0
	var frames_since_last_tick uint = 0
	ticker := time.NewTicker(time.Second * time.Duration(stat_period_sec))
	for {
		select {
		case <-ctx.Done():
			logger.Info("Stat cancelled by context")
			return ERR_CANCELLED_BY_CONTEXT
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
		return ERR_CANCELLED_BY_CONTEXT
	case <-interrupt:
		logger.Info("Cancelled by user")
		return ERR_INTERRUPTED_BY_USER
	}
}
