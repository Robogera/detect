package main

import (
	// stdlib
	"context"
	"errors"
	"fmt"
	"image"
	// "image"
	"image/color"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// external
	"github.com/hybridgroup/mjpeg"
	"github.com/lmittmann/tint"
	"gocv.io/x/gocv"
	// "gocv.io/x/gocv/cuda"
	"golang.org/x/sync/errgroup"
)

const (
  file_path            string  = "/mnt/c/Users/gera/Downloads/greenscreen.mp4"
	http_port            uint    = 8080
	read_timeout_sec     uint    = 20
	write_timeout_sec    uint    = 20
	shutdown_timeout_sec uint    = 15
	stat_period_sec      uint    = 2
	resize_scale         float64 = 0.33
)

var (
	ERR_BAD_STREAM           error = errors.New("Can't read from stream")
	ERR_STREAM_ENDED         error = errors.New("Stream ended")
	ERR_CANCELLED_BY_CONTEXT error = errors.New("Cancelled via context")
	ERR_INTERRUPTED_BY_USER  error = errors.New("Interrupted by user")
)

func main() {
	logger := slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelInfo,
		TimeFormat: time.RFC3339,
		AddSource:  true,
	}))

	logger.Info("Starting...")

	ctx := context.Background()
	eg, child_ctx := errgroup.WithContext(ctx)

	stats_chan := make(chan struct{}, 8)

	output_stream := mjpeg.NewStream()

	eg.Go(func() error {
		return webserver(
			child_ctx, logger, output_stream, http_port,
			read_timeout_sec, write_timeout_sec, shutdown_timeout_sec)
	})

	eg.Go(func() error {
		return video(child_ctx, logger, file_path, resize_scale, output_stream, stats_chan)
	})

	eg.Go(func() error {
		return stat(child_ctx, logger, stats_chan, stat_period_sec)
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
	file_uri string,
	resize_scale float64,
	output_stream *mjpeg.Stream,
	stats chan<- struct{}) error {

	logger.Info("Opening file", "address", file_uri)

	input_stream, err := gocv.VideoCaptureFile(file_uri)
	if err != nil {
		logger.Error("Can't open stream", "address", file_uri, "err", err)
		return ERR_BAD_STREAM
	}
	defer input_stream.Close()

	img := gocv.NewMat()
	defer img.Close()

	hog := gocv.NewHOGDescriptor()
	err = hog.SetSVMDetector(gocv.HOGDefaultPeopleDetector())
	if err != nil {
		logger.Error("Can't set SVM detector", "err", err)
		return err
	}
	// hog := cuda.CreateHOG()
	// hog.SetSVMDetector(hog.GetDefaultPeopleDetector())

	// gpumat := cuda.NewGpuMat()
	// defer gpumat.Close()

	// gpumat_grey := cuda.NewGpuMat()
	// defer gpumat_grey.Close()

	logger.Info("Video loop started")
	for {
		select {
		case <-ctx.Done():
			logger.Info("Video cancelled by context")
			return ERR_CANCELLED_BY_CONTEXT
		default:
			if !input_stream.Read(&img) {
				logger.Error("Can't read next frame", "stream", file_uri)
				return ERR_STREAM_ENDED
			}
			if img.Empty() {
				logger.Error("Empty frame received, skipping", "stream", file_uri)
				continue
			}
			// gpumat.Upload(img)
			// cuda.CvtColor(gpumat, &gpumat_grey, gocv.ColorRGBToGray)
			// rectangles := hog.DetectMultiScale(gpumat_grey)
			gocv.CvtColor(img, &img, gocv.ColorRGBToGray)
			gocv.Resize(img, &img, image.Point{}, resize_scale, resize_scale, gocv.InterpolationLinear)
      rectangles := hog.DetectMultiScale(img)
			// rectangles := hog.DetectMultiScaleWithParams(
			// 	img, 0.1,
			// 	image.Point{2, 2}, image.Point{32, 32}, 1, 1, true)
			for _, rectangle := range rectangles {
				logger.Info(
					"Rectangle found",
					"min_x", rectangle.Min.X, "min_y", rectangle.Min.Y,
					"max_x", rectangle.Max.X, "max_y", rectangle.Max.Y)
				gocv.Rectangle(&img, rectangle, color.RGBA{0, 255, 0, 255}, 1)
			}
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
