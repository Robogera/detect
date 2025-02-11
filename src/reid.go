package main

import (
	"context"
	"fmt"
	"image"
	"time"

	// "image/color"
	"log/slog"
	"runtime"

	"github.com/Robogera/detect/pkg/config"
	"github.com/Robogera/detect/pkg/indexed"
	"github.com/Robogera/detect/pkg/person"
	"gocv.io/x/gocv"
)

func reidentificator(
	ctx context.Context,
	logger *slog.Logger,
	cfg *config.ConfigFile,
	in_chan <-chan indexed.Indexed[ProcessedFrame],
	out_chan chan<- indexed.Indexed[ProcessedFrame],
) error {
	// not sure if this helps
	runtime.LockOSThread()

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

	if net.Empty() {
		logger.Error("Error reading network model")
		return ERR_BAD_MODEL
	}
	defer net.Close()

	blob_conv_params := gocv.NewImageToBlobParams(
		1.0,
		image.Pt(128, 256),
		gocv.NewScalar(0, 0, 0, 0),
		true,
		gocv.MatTypeCV32F,
		gocv.DataLayoutNCHW,
		gocv.PaddingModeLetterbox,
		gocv.NewScalar(0, 0, 0, 0),
	)

	associator, err := person.NewAssociator(&net, &blob_conv_params, cfg.Reid.OutputLayerName)
	if err != nil {
		logger.Error("Associator init error", "error", err)
		return fmt.Errorf("Can't create associator: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			logger.Info("Reidentificator cancelled by context")
			return context.Canceled
		case frame := <-in_chan:
			dims := frame.Value().Mat.Size()
			associator.CleanUp(time.Second * 3, image.Rect(0,0, dims[1], dims[0]))
			associator.Associate(frame.Value().Mat, frame.Value().Boxes, frame.Time())
			// for box_ind, box := range frame.Value().Boxes {
			// 	gocv.Rectangle(frame.Value().Mat, box, color.RGBA{0, 255, 0, 64}, 1)
			// 	gocv.PutText(frame.Value().Mat, fmt.Sprintf("%d", box_ind), box.Min, gocv.FontHersheyPlain, 4, color.RGBA{0, 255, 0, 255}, 2)
			// }
			for _, person := range associator.EnumeratedPeople() {
				gocv.Circle(frame.Value().Mat, person.State(), 5, person.Color(), 5)
				prev_point := person.State()
				for point := range person.Trajectory() {
					if (image.Point{}) != prev_point {
						gocv.Line(frame.Value().Mat, prev_point, point, person.Color(), 3)
					}
					prev_point = point
				}
			}
			select {
			case <-ctx.Done():
				logger.Info("Streamreader cancelled by context")
				return context.Canceled
			case out_chan <- frame:
			}
		}
	}

}
