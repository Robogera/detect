package main

import (
	"context"
	"image"
	"log/slog"
	"runtime"

	"github.com/Robogera/detect/pkg/config"
	"github.com/Robogera/detect/pkg/yolo"
	gocvcommon "github.com/Robogera/detect/pkg/gocv-common"
	"github.com/Robogera/detect/pkg/indexed"
	"gocv.io/x/gocv"
)

type ProcessedFrame struct {
	Mat   *gocv.Mat
	Boxes []image.Rectangle
}

func detector(
	ctx context.Context,
	parent_logger *slog.Logger,
	cfg *config.ConfigFile,
	in_chan <-chan indexed.Indexed[gocv.Mat],
	out_chan chan<- indexed.Indexed[ProcessedFrame],
) error {

	// not sure if this helps
	runtime.LockOSThread()

	logger := parent_logger.With("coroutine", "detector")

	var net gocv.Net
	defer net.Close()

	// TODO: panic and recover when the CGO segfaults maybe?
	switch config.ModelFormat(cfg.Yolo.Format) {
	case config.ModelFormatCaffe:
		// TODO: test
		net = gocv.ReadNetFromCaffe(cfg.Yolo.ConfigPath, cfg.Yolo.Path)
	case config.ModelFormatONNX:
		net = gocv.ReadNetFromONNX(cfg.Yolo.Path)
	case config.ModelFormatOpenVINO:
		// TODO: test
		net = gocv.ReadNet(cfg.Yolo.Path, cfg.Yolo.ConfigPath)
	}

	if net.Empty() {
		logger.Error("Error reading network model")
		return ERR_BAD_MODEL
	}
	defer net.Close()

	output_layer_names := gocvcommon.GetOutputLayerNames(&net)
	if len(output_layer_names) == 0 {
		logger.Error("Can't read output layer name", "model", cfg.Yolo.Path)
		return ERR_BAD_MODEL
	}
	logger.Debug("Model info", "model", cfg.Yolo.Path, "output layers", output_layer_names)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Cancelled by context")
			return context.Canceled
		case frame := <-in_chan:
			img := frame.Value()
			boxes, _ := yolo.Detect(&net, &img, cfg, output_layer_names)

			select {
			case out_chan <- indexed.NewIndexed(frame.Id(), frame.Time(), ProcessedFrame{
				Mat: &img,
				Boxes: boxes,
			}):
			logger.Debug("Detected", "boxes", boxes)
			case <-ctx.Done():
				logger.Info("Cancelled by context")
				return context.Canceled
			}
		}
	}
}
