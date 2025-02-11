package main

import (
	// stdlib
	"context"
	"log/slog"
	"time"

	// internal
	"github.com/Robogera/detect/pkg/config"
	"github.com/Robogera/detect/pkg/indexed"

	// external
	"gocv.io/x/gocv"
)

func streamreader(
	ctx context.Context,
	logger *slog.Logger,
	cfg *config.ConfigFile,
	mat_chan chan<- indexed.Indexed[gocv.Mat],
) error {

	var input_stream *gocv.VideoCapture
	var err error

	switch config.InputType(cfg.Input.Type) {
	case config.InputTypeFile:
		input_stream, err = gocv.VideoCaptureFile(cfg.Input.Path)
	case config.InputTypeWebcam:
		// TODO: implement user supplied index/device address
		input_stream, err = gocv.VideoCaptureDevice(0)
	case config.InputTypeIPC:
		input_stream, err = gocv.OpenVideoCapture(cfg.Input.Path)
	default:
		slog.Error(
			"No valid input type provided. Shutting down...",
			"provided value", cfg.Input.Type)
		return ERR_INVALID_CONFIG
	}

	if err != nil {
		logger.Error(
			"Can't open input",
			"type", cfg.Input.Type,
			"address", cfg.Input.Path,
			"err", err)
		return ERR_BAD_INPUT
	}
	defer input_stream.Close()

	var frame_id uint64 = 0

	for {
		select {
		case <-ctx.Done():
			logger.Info("Streamreader cancelled by context")
			return context.Canceled
		default:
			// Reciever of this is responsible for closing
			img := gocv.NewMat()
			if !input_stream.Read(&img) {
				logger.Error("Can't read next frame. Shutting down...", "stream", cfg.Input.Path)
				return ERR_STREAM_ENDED
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
