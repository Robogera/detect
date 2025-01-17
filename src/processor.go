package main

import (
	// stdlib
	"context"
	"log/slog"

	// internal
	"github.com/Robogera/detect/pkg/config"

	// external
	"gocv.io/x/gocv"
)

func processor(
	ctx context.Context,
	logger *slog.Logger,
	cfg *config.ConfigFile,
	frames_chan chan<- []byte) error {

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

	var net gocv.Net

	// TODO: panic and recover when the CGO segfaults maybe?
	switch config.ModelFormat(cfg.Model.Format) {
	case config.ModelFormatCaffe:
		// TODO: test
		net = gocv.ReadNetFromCaffe(cfg.Model.ConfigPath, cfg.Model.Path)
	case config.ModelFormatONNX:
		net = gocv.ReadNetFromONNX(cfg.Model.Path)
	case config.ModelFormatOpenVINO:
		// TODO: test
		net = gocv.ReadNet(cfg.Model.Path, cfg.Model.ConfigPath)
	}

	if net.Empty() {
		logger.Error("Error reading network model")
		return ERR_BAD_MODEL
	}
	defer net.Close()

	output_layer_names := getOutputLayerNames(&net)
	if len(output_layer_names) == 0 {
		logger.Error("Can't read output layer name", "model", cfg.Model.Path)
		return ERR_BAD_MODEL
	}
	logger.Debug("Model info", "model", cfg.Model.Path, "output layers", output_layer_names)

	// TODO: select user provided backend/device
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
			return context.Canceled
		default:
			if !input_stream.Read(&img) {
				logger.Error("Can't read next frame. Shutting down...", "stream", cfg.Input.Path)
				return ERR_STREAM_ENDED
			}
			if img.Empty() {
				logger.Error("Empty frame received, skipping", "stream", cfg.Input.Path)
				continue
			}

			boxed_img, _, _ := detectObjects(&net, &img, output_layer_names)

			buf, err := gocv.IMEncode(".jpg", *boxed_img)
			if err != nil {
				logger.Error("Can't encode frame")
				return err
			}
			data := make([]byte, buf.Len())
			copy(data, buf.GetBytes()) // need to profile this and maybe not copy the entire frame every time
			select {
			case frames_chan <- data:
			default:
				logger.Warn("Frame channel full. Droping the frame...", "capacity", len(frames_chan))
			}
			buf.Close()
		}
	}
}
