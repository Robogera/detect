package main

import (
	"context"
	"fmt"
	"image"

	// "image/color"
	"log/slog"
	"runtime"

	"github.com/Robogera/detect/pkg/config"
	"github.com/Robogera/detect/pkg/functions"
	"github.com/Robogera/detect/pkg/indexed"
	"github.com/Robogera/detect/pkg/person"
	"gocv.io/x/gocv"
)

func reidentificator(
	ctx context.Context,
	parent_logger *slog.Logger,
	cfg *config.ConfigFile,
	in_chan <-chan indexed.Indexed[ProcessedFrame],
	out_chan chan<- indexed.Indexed[ProcessedFrame],
	export_chan chan<- indexed.Indexed[[]*person.ExportedPerson],
) error {
	// not sure if this helps
	runtime.LockOSThread()
	logger := parent_logger.With("coroutine", "reidentificator")

	var net gocv.Net
	defer net.Close()

	// TODO: panic and recover when the CGO segfaults maybe?
	switch config.ModelFormat(cfg.Reid.Format) {
	case config.ModelFormatCaffe:
		// TODO: test
		net = gocv.ReadNetFromCaffe(cfg.Reid.ConfigPath, cfg.Reid.Path)
	case config.ModelFormatONNX:
		net = gocv.ReadNetFromONNX(cfg.Reid.Path)
	case config.ModelFormatOpenVINO:
		// TODO: test
		net = gocv.ReadNet(cfg.Reid.Path, cfg.Reid.ConfigPath)
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

	blob_conv_params := gocv.NewImageToBlobParams(
		1.0,
		image.Pt(128, 256),
		gocv.NewScalar(0, 0, 0, 0),
		false,
		gocv.MatTypeCV32F,
		gocv.DataLayoutNCHW,
		gocv.PaddingModeLetterbox,
		gocv.NewScalar(0, 0, 0, 0),
	)

	associator, err := person.NewAssociator(&net, &blob_conv_params, cfg)
	if err != nil {
		logger.Error("Can't init associator", "error", err)
		return fmt.Errorf("Can't init associator: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			logger.Info("Cancelled by context")
			return context.Canceled
		case frame := <-in_chan:
			dims := frame.Value().Mat.Size()
			associator.CleanUp(frame.Time(), image.Rect(0, 0, dims[1], dims[0]))
			associator.Associate(
				frame.Value().Mat, frame.Value().Boxes, frame.Time(),
				func(s, d float64) float64 {
					return functions.Gaussian(s, d, cfg.Reid.DistanceFactor)
				},
			)
			people := associator.EnumeratePeople()
			status := make(map[string]string, len(people))
			export := make([]*person.ExportedPerson, 0, len(people))
			for _, person := range people {
				export = append(export, person.Export())
				status[person.Id()] = string(person.Status())
				if person.IsValid() {
					person.DrawCross(frame.Value().Mat, 2, 9, 255)
				}
				person.DrawBox(frame.Value().Mat, 1)
				// person.DrawTrajectory(frame.Value().Mat, 1, alpha)
			}
			logger.Info("People", "status", status)
			select {
			case <-ctx.Done():
				logger.Info("Streamreader cancelled by context")
				return context.Canceled
			case out_chan <- frame:
			}
			select {
			case <-ctx.Done():
				logger.Info("Streamreader cancelled by context")
				return context.Canceled
			case export_chan <- indexed.NewIndexed(frame.Id(), frame.Time(), export):
			}
		}
	}
}
