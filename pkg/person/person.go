package person

import (
	"image"
	"image/color"
	"iter"
	"time"

	"github.com/Robogera/detect/pkg/gring"
	"github.com/Robogera/detect/pkg/gsma"
	"github.com/Robogera/detect/pkg/kalman"
	"github.com/muesli/gamut"
	"gocv.io/x/gocv"
)

var base_color = color.RGBA{255, 0, 0, 255}

func nextColor() color.RGBA {
	new_col := gamut.HueOffset(base_color, 150)
	r, g, b, a := new_col.RGBA()
	base_color = color.RGBA{
		uint8(r),
		uint8(g),
		uint8(b),
		uint8(a)}
	return base_color
}

func (a *Associator) NewPerson(t time.Time, box image.Rectangle, descriptor []float32) (*Person, error) {
	a.color = gamut.HueOffset(a.color, 133)
	r, g, b, _ := a.color.RGBA()
	descriptors := gring.NewRing[[]float32](3)
	descriptors.Push(descriptor)
	return &Person{
		id:                generateToken(4),
		last_update:       t,
		trajectory:        gring.NewRing[image.Point](40),
		color:             color.RGBA{uint8(r), uint8(g), uint8(b), 255},
		descriptors:       descriptors,
		filter:            kalman.NewFilter(center(box), t, a.proc_noise_cov, a.meas_noise_cov),
		sma:               gsma.NewSMA2d(a.sma_window),
		missed_frames:     0,
		max_missed_frames: a.frames_to_follow,
	}, nil
}

type Person struct {
	id                               string
	last_update                      time.Time
	trajectory                       *gring.Ring[image.Point]
	color                            color.RGBA
	descriptors                      *gring.Ring[[]float32]
	filter                           *kalman.Filter
	sma                              *gsma.SMA2d
	missed_frames, max_missed_frames int
}

func (p *Person) Id() string        { return p.id }
func (p *Person) Color() color.RGBA { return p.color }
func (p *Person) NotUpdatedFor(t time.Duration) bool {
	return p.last_update.Add(t).Before(time.Now())
}
func (p *Person) Trajectory() iter.Seq[image.Point] {
	return p.trajectory.All()
}

func (p *Person) Update(t time.Time, box image.Rectangle, descriptor []float32) error {
	p.missed_frames = 0
	p.trajectory.Push(p.sma.Recalc(p.filter.State()))
	p.descriptors.Push(descriptor)
	p.filter.Update(center(box), t)
	p.last_update = t
	return nil
}

func (p *Person) Predict(t time.Time) error {
	p.missed_frames++
	p.trajectory.Push(p.sma.Recalc(p.filter.State()))
	if p.missed_frames >= p.max_missed_frames {
		return nil
	}
	p.filter.Predict(t)
	return nil
}

func (p *Person) State() image.Point {
	return p.trajectory.Newest()
}

func (p *Person) DrawTrajectory(m *gocv.Mat, w int) {
	prev_point := p.State()
	for point := range p.Trajectory() {
		if (image.Point{}) != prev_point {
			gocv.Line(m, prev_point, point, p.Color(), w)
		}
		prev_point = point
	}
}

func (p *Person) DrawCross(m *gocv.Mat, w, r int) {
	gocv.Line(m, p.State().Add(image.Pt(-r, -r)), p.State().Add(image.Pt(r, r)), p.Color(), w)
	gocv.Line(m, p.State().Add(image.Pt(-r, r)), p.State().Add(image.Pt(r, -r)), p.Color(), w)
}
