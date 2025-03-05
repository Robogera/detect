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
	in_chan <-chan indexed.Indexed[*gocv.Mat],
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

	err := net.SetPreferableBackend(gocv.NetBackendOpenVINO)
	if err != nil {
		logger.Error("Can't set openvino backend", "model", cfg.Yolo.Path)
		return ERR_BAD_MODEL
	}
	err = net.SetPreferableTarget(gocv.NetTargetCPU)
	if err != nil {
		logger.Error("Can't set cpu backend", "model", cfg.Yolo.Path)
		return ERR_BAD_MODEL
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

	blob_conv_params := gocv.NewImageToBlobParams(
		1.0/cfg.Yolo.ScaleFactor,
		image.Pt(int(cfg.Yolo.W), int(cfg.Yolo.H)),
		gocv.NewScalar(0, 0, 0, 0),
		true,
		gocv.MatTypeCV32F,
		gocv.DataLayoutNCHW,
		gocv.PaddingModeLetterbox,
		gocv.NewScalar(0, 0, 0, 0),
	)
	for {
		select {
		case <-ctx.Done():
			logger.Info("Cancelled by context")
			return context.Canceled
		case frame := <-in_chan:
			boxes, err := yolo.Detect(&net, frame.Value(), cfg, output_layer_names, &blob_conv_params)
      if err != nil {
				logger.Error("Detection failure", "error", err)
      }
			select {
			case out_chan <- indexed.NewIndexed(frame.Id(), frame.Time(), ProcessedFrame{
				Mat: frame.Value(),
				Boxes: boxes,
			}):
			logger.Info("Detected", "boxes", boxes)
			case <-ctx.Done():
				logger.Info("Cancelled by context")
				return context.Canceled
			}
		}
	}
}
