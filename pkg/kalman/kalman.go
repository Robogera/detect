package kalman

import (
	"image"
	"time"

	"gocv.io/x/gocv"
)

func set(s [][]float32, m gocv.Mat) {
	for r_ind, r := range s {
		for c_ind, v := range r {
			m.SetFloatAt(r_ind, c_ind, v)
		}
	}
}

type Filter struct {
	filter      *gocv.KalmanFilter
	last_update time.Time
}

func (kf *Filter) Free() {
	kf.filter.Close()
}

func (kf *Filter) predict(dt float32) {
	tr_mat := kf.filter.GetTransitionMatrix()
	defer tr_mat.Close()
	tr_mat.SetFloatAt(0, 2, dt)
	tr_mat.SetFloatAt(1, 3, dt)
	// tr_mat.SetFloatAt(0, 4, dt*dt/2)
	// tr_mat.SetFloatAt(1, 5, dt*dt/2)
	// tr_mat.SetFloatAt(2, 4, dt)
	// tr_mat.SetFloatAt(3, 5, dt)
	// tr_mat.SetFloatAt(4, 4, .8)
	// tr_mat.SetFloatAt(5, 5, .8)
	pred := kf.filter.Predict()
	defer pred.Close()
}

func (kf *Filter) Predict(t time.Time) {
	dt := float32(t.Sub(kf.last_update).Seconds())
	kf.predict(dt)
}

func (kf *Filter) Update(meas image.Point, t time.Time) {
	dt := float32(t.Sub(kf.last_update).Seconds())
	kf.predict(dt)

	meas_mat := gocv.NewMatWithSize(2, 1, gocv.MatTypeCV32F)
	defer meas_mat.Close()
	meas_mat.SetFloatAt(0, 0, float32(meas.X))
	meas_mat.SetFloatAt(1, 0, float32(meas.Y))
	corr := kf.filter.Correct(meas_mat)
	defer corr.Close()
}

func (kf *Filter) State() image.Point {
	state := kf.filter.GetStatePost()
	defer state.Close()

	return image.Pt(
		int(state.GetFloatAt(0, 0)),
		int(state.GetFloatAt(0, 1)),
	)
}

func (kf *Filter) Noise() []float32 {
	noise_cov := kf.filter.GetMeasurementNoiseCov()
	defer noise_cov.Close()

	data, _ := noise_cov.DataPtrFloat32()
	raw_data := make([]float32, len(data))
	copy(raw_data, data)

	return raw_data
}

func (kf *Filter) Process() []float32 {
	noise_cov := kf.filter.GetProcessNoiseCov()
	defer noise_cov.Close()

	data, _ := noise_cov.DataPtrFloat32()
	raw_data := make([]float32, len(data))
	copy(raw_data, data)

	return raw_data
}

func (kf *Filter) Speed() image.Point {
	state := kf.filter.GetStatePost()
	defer state.Close()

	return image.Pt(
		int(state.GetFloatAt(0, 2)),
		int(state.GetFloatAt(0, 3)),
	)
}

func (kf *Filter) Close() {
	kf.filter.Close()
}

func NewFilter(p image.Point, t time.Time, proc_noise_cov, meas_noise_cov float64) *Filter {
	filter := gocv.NewKalmanFilter(4, 2)
	gocv.SetIdentity(filter.GetTransitionMatrix(), 1)
	gocv.SetIdentity(filter.GetMeasurementMatrix(), 1)
	gocv.SetIdentity(filter.GetProcessNoiseCov(), proc_noise_cov)
	gocv.SetIdentity(filter.GetMeasurementNoiseCov(), meas_noise_cov)
	gocv.SetIdentity(filter.GetErrorCovPost(), 1)
	mat := filter.GetStatePre()
	mat.SetFloatAt(0, 0, float32(p.X))
	mat.SetFloatAt(0, 1, float32(p.Y))
	filter.SetStatePre(mat)
	mat = filter.GetStatePost()
	mat.SetFloatAt(0, 0, float32(p.X))
	mat.SetFloatAt(0, 1, float32(p.Y))
	filter.SetStatePost(mat)
	return &Filter{
		filter: &filter, last_update: t}
}
