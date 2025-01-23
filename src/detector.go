package main

import (
	"context"
	"log/slog"
	"runtime"
	"time"

	"github.com/Robogera/detect/pkg/config"
	"github.com/Robogera/detect/pkg/indexed"
	"gocv.io/x/gocv"
)

func detector(
	ctx context.Context,
	logger *slog.Logger,
	cfg *config.ConfigFile,
	in_chan <-chan indexed.Indexed[gocv.Mat],
	out_chan chan<- indexed.Indexed[[]byte],
	stat_chan chan<- Statistics,
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
			inference_start := time.Now()
			img := frame.Value()
			boxed_img, _ := detectObjects(&net, &img, cfg, output_layer_names)

			stat_chan <- Statistics{time.Since(inference_start)}

			buf, err := gocv.IMEncode(gocv.JPEGFileExt, *boxed_img)
			if err != nil {
				logger.Error("Can't encode frame")
				return err
			}

			data := make([]byte, buf.Len())
			copy(data, buf.GetBytes()) // need to profile this and maybe not copy the entire frame every time
			select {
			case out_chan <- indexed.NewIndexed[[]byte](frame.Id(), data):
			default:
				logger.Warn("Frame channel full. Droping the frame...", "capacity", len(out_chan))
			}

			buf.Close()
			img.Close()
			boxed_img.Close()
		}
	}
}
