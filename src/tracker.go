package main

import (
	"context"
	"image"
	"image/color"
	"log/slog"
	"time"

	"github.com/Robogera/detect/pkg/assoc"
	"github.com/Robogera/detect/pkg/config"
	"github.com/Robogera/detect/pkg/indexed"
	"github.com/Robogera/detect/pkg/seq"
	tr "github.com/Robogera/detect/pkg/tracker"
	"gocv.io/x/gocv"
)

func tracker(
	ctx context.Context,
	logger *slog.Logger,
	cfg *config.ConfigFile,
	in_chan <-chan indexed.Indexed[ProcessedFrame],
	out_chan chan<- indexed.Indexed[[]byte],
) error {

	trackers := make(map[string]*tr.Tracker, 0)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Tracker cancelled by context")
			return context.Canceled
		case frame := <-in_chan:
			old_points, tracker_ids := tr.GetTrackedPoints(trackers)
			det_points := seq.SMap[image.Point, image.Rectangle](frame.Value().Boxes, func(r image.Rectangle, i int) image.Point {
				return image.Pt(
					(r.Max.X-r.Min.X)/2,
					(r.Max.Y-r.Min.Y)/2,
				)
			})
			associated_points, lost_points, new_points :=
				assoc.Associate(old_points, det_points, 800)
			for old_point, det_point := range associated_points {
				trackers[tracker_ids[old_point]].Update(
					frame.Time(),
					det_points[det_point],
				)
			}
			for _, lost_point := range lost_points {
				trackers[tracker_ids[lost_point]].Predict(frame.Time())
			}
			for _, new_point := range new_points {
				tracker := tr.NewTracker(frame.Time(), det_points[new_point])
				trackers[tracker.Id()] = tracker
			}
			for id, tracker := range trackers {
				expired := tracker.OlderThan(time.Second / 5)
				gocv.Circle(frame.Value().Mat, tracker.State(), 3, color.RGBA{255, 0, 0, 255}, 2)
				// gocv.PutText(frame.Value().Mat, tracker.Id(), tracker.State(), gocv.FontHersheyComplex, 24, color.RGBA{255, 255, 0, 255}, 2)
				logger.Info(
					"Tracker", "id", id,
					"X", tracker.State().X,
					"Y", tracker.State().Y,
					"expired", expired,
				)
				if expired {
					delete(trackers, id)
				}
			}
			buf, err := gocv.IMEncode(gocv.JPEGFileExt, *frame.Value().Mat)
			if err != nil {
				logger.Error("Can't encode frame")
				return err
			}

			data := make([]byte, buf.Len())
			copy(data, buf.GetBytes()) // need to profile this and maybe not copy the entire frame every time
			select {
			case out_chan <- indexed.NewIndexed[[]byte](frame.Id(), frame.Time(), data):
			case <-ctx.Done():
				logger.Info("Tracker cancelled by context")
				return context.Canceled
			}
			buf.Close()
			frame.Value().Mat.Close()
		}
	}
}
