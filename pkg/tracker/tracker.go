package tracker

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"image"
	"time"

	"github.com/rosshemsley/kalman"
	"github.com/rosshemsley/kalman/models"
	"gonum.org/v1/gonum/mat"
)

type Tracker struct {
	id                 string
	since, last_update time.Time
	model              *models.ConstantVelocityModel
	filter             *kalman.KalmanFilter
}

func NewTracker(timestamp time.Time, point image.Point) *Tracker {
	model := models.NewConstantVelocityModel(
		timestamp,
		pointToVec(point),
		models.ConstantVelocityModelConfig{
			InitialVariance: 0.01,
			ProcessVariance: 0.01,
		})
	return &Tracker{
		id:          generateToken(8),
		since:       timestamp,
		last_update: timestamp,
		model:       model,
		filter:      kalman.NewKalmanFilter(model),
	}
}

func (tr *Tracker) OlderThan(t time.Duration) bool {
	return tr.since.Add(t).Before(time.Now())
}

func (tr *Tracker) Id() string { return tr.id }

func (tr *Tracker) Update(t time.Time, point image.Point) (image.Point, error) {
	err := tr.filter.Update(t, tr.model.NewPositionMeasurement(pointToVec(point), 0.1))
	if err != nil {
		return image.Pt(0, 0), fmt.Errorf("Can't update tracker. Error: %w", err)
	}
	tr.last_update = t
	return vecToPoint(
		tr.model.Position(
			tr.filter.State())), nil
}

func (tr *Tracker) State() image.Point {
	return vecToPoint(
		tr.model.Position(
			tr.filter.State()))
}

func (tr *Tracker) Predict(t time.Time) (image.Point, error) {
	err := tr.filter.Predict(t)
	if err != nil {
		return image.Pt(0, 0), fmt.Errorf("Can't perform prediction for tracker %s. Error: %w", tr.id, err)
	}
	return vecToPoint(
		tr.model.Position(
			tr.filter.State())), nil
}

func pointToVec(point image.Point) mat.Vector {
	return mat.NewVecDense(2, []float64{
		float64(point.X),
		float64(point.Y),
	})
}

func vecToPoint(vec mat.Vector) image.Point {
	return image.Pt(
		int(vec.AtVec(0)),
		int(vec.AtVec(1)),
	)
}

func generateToken(l int) string {
	b := make([]byte, l)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

func GetTrackedPoints(trackers map[string]*Tracker) (points []image.Point, tracker_indices []string) {
	for _, tracker := range trackers {
		points = append(points, tracker.State())
		tracker_indices = append(tracker_indices, tracker.Id())
	}
	return
}
