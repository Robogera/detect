package main

import (
	// stdlib
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"runtime"
	"time"

	// internal
	"github.com/Robogera/detect/pkg/config"
	"github.com/Robogera/detect/pkg/indexed"

	// external
	"gocv.io/x/gocv"
)

var frame_id uint64 = 0

func streamreader(
	ctx context.Context,
	parent_logger *slog.Logger,
	cfg *config.ConfigFile,
	mat_chan chan<- indexed.Indexed[*gocv.Mat],
) error {

	// not sure if this helps
	runtime.LockOSThread()

	logger := parent_logger.With("coroutine", "streamreader")
	for {
		select {
		case <-ctx.Done():
			logger.Info("Streamreader cancelled by context")
			return context.Canceled
		default:
			err := _streamreader(ctx, logger, cfg, mat_chan)
			if errors.Is(err, context.Canceled) {
				return err
			} else {
				logger.Warn("Restarting streamreader", "error", err)
			}
		}
	}
}

func _streamreader(
	ctx context.Context,
	logger *slog.Logger,
	cfg *config.ConfigFile,
	mat_chan chan<- indexed.Indexed[*gocv.Mat],
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

	var fill_zone gocv.PointsVector
	var do_fill bool

	if len(cfg.Mask.Contours) > 0 {
		contours := make([][]image.Point, 0, len(cfg.Mask.Contours))
		for _, points := range cfg.Mask.Contours {
			contour := make([]image.Point, 0, len(points))
			for _, point := range points {
				contour = append(contour, image.Pt(int(point.X), int(point.Y)))
			}
			contours = append(contours, contour)
		}
		fill_zone = gocv.NewPointsVectorFromPoints(contours)
		defer fill_zone.Close()
		do_fill = true
	}

	var crop_zone image.Rectangle
	var do_crop bool
	if cfg.Crop.A.X != 0 || cfg.Crop.A.Y != 0 ||
		cfg.Crop.B.X != 0 || cfg.Crop.B.Y != 0 {
		do_crop = true
		crop_zone = image.Rect(
			int(cfg.Crop.A.X),
			int(cfg.Crop.A.Y),
			int(cfg.Crop.B.X),
			int(cfg.Crop.B.Y),
		)
	}

	for {
		select {
		case <-ctx.Done():
			logger.Info("Cancelled by context")
			return context.Canceled
		default:
			// Reciever of this is responsible for closing
			var processed_img gocv.Mat
			err := func() error {
				img := gocv.NewMat()
				defer img.Close()
				if !input_stream.Read(&img) {
					logger.Error("Can't read next frame. Shutting down...", "stream", cfg.Input.Path)
					img.Close()
					return ERR_STREAM_ENDED
				}
				if img.Empty() {
					logger.Error("Empty frame received, skipping", "stream", cfg.Input.Path)
					img.Close()
					return nil
				}

				if do_crop {
					r, c := img.Rows(), img.Cols()
					frame := image.Rect(0, 0, c, r)
					if !crop_zone.In(frame) {
						logger.Warn("crop zone doesn't fit into frame", "frame", frame, "crop_zone", crop_zone)
						crop_zone = crop_zone.Union(frame)
					}
					if crop_zone.Dx() < 1 || crop_zone.Dy() < 1 {
						logger.Error("crop zone too small", "crop_zone", crop_zone)
            return fmt.Errorf("Crop zone too small")
					}
					region_ptr := img.Region(crop_zone)
					defer region_ptr.Close()
					processed_img = region_ptr.Clone()
				} else {
					processed_img = img.Clone()
				}

				if do_fill {
					gocv.FillPoly(&processed_img, fill_zone, color.RGBA{
						cfg.Mask.Color.R, cfg.Mask.Color.G, cfg.Mask.Color.B, 255})
				}

				return nil
			}()
			if err != nil {
				return err
			}

			select {
			case <-ctx.Done():
				logger.Info("Cancelled by context")
				return context.Canceled
			case mat_chan <- indexed.NewIndexed(frame_id, time.Now(), &processed_img):
				frame_id++
			}
		}
	}
}
