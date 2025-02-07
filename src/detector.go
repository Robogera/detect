package main

import (
	"context"
	"image"
	"log/slog"
	"runtime"

	"github.com/Robogera/detect/pkg/config"
	"github.com/Robogera/detect/pkg/indexed"
	"gocv.io/x/gocv"
)

type ProcessedFrame struct {
	Mat   *gocv.Mat
	Boxes []image.Rectangle
}

func detector(
	ctx context.Context,
	logger *slog.Logger,
	cfg *config.ConfigFile,
	in_chan <-chan indexed.Indexed[gocv.Mat],
	out_chan chan<- indexed.Indexed[ProcessedFrame],
) error {

	// not sure if this helps
	runtime.LockOSThread()

	var net gocv.Net
	defer net.Close()

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

	for {
		select {
		case <-ctx.Done():
			logger.Info("Detector cancelled by context")
			return context.Canceled
		case frame := <-in_chan:
			img := frame.Value()
			boxes, _ := detectObjects(&net, &img, cfg, output_layer_names)

			select {
			case out_chan <- indexed.NewIndexed[ProcessedFrame](frame.Id(), frame.Time(), ProcessedFrame{
				Mat: &img,
				Boxes: boxes,
			}):
			default:
				logger.Warn("Frame channel full. Droping the frame...", "capacity", len(out_chan))
			}

			img.Close()
		}
	}
}
