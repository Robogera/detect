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
	a.next_color = gamut.HueOffset(a.next_color, 153)
	r, g, b, _ := a.next_color.RGBA()
	descriptors := gring.NewRing[[]float32](a.total_descriptors)
	descriptors.Push(descriptor)
	trajectory := gring.NewRing[image.Point](a.trajectory_points)
	trajectory.Push(center(box))
	return &Person{
		id:           generateToken(a.token_length),
		last_update:  t,
		trajectory:   trajectory,
		color:        color.RGBA{uint8(r), uint8(g), uint8(b), 255},
		descriptors:  descriptors,
		filter:       kalman.NewFilter(center(box), t, a.proc_noise_cov, a.meas_noise_cov),
		sma:          gsma.NewSMA2d(a.sma_window),
		total_hits:   0,
		total_misses: 0,
		valid:        false,
		last_box:     box,
	}, nil
}

type Person struct {
	id                       string
	created                  time.Time
	last_update              time.Time
	trajectory               *gring.Ring[image.Point]
	color                    color.RGBA
	descriptors              *gring.Ring[[]float32]
	filter                   *kalman.Filter
	sma                      *gsma.SMA2d
	total_hits, total_misses uint
	valid                    bool
	last_box                 image.Rectangle
}

func (p *Person) Id() string        { return p.id }
func (p *Person) Color() color.RGBA { return p.color }
func (p *Person) SinceUpdate(t time.Time) time.Duration {
	return t.Sub(p.last_update)
}
func (p *Person) Trajectory() iter.Seq[image.Point] {
	return p.trajectory.All()
}
func (p *Person) IsValid() bool {
	return p.valid
}

func (p *Person) Validate(t time.Time, validation_duration time.Duration, validation_ratio float64) {
	if !p.valid {
		hit_ratio := float64(p.total_hits) / float64(p.total_hits+p.total_misses)
		if t.Sub(p.created) > validation_duration && hit_ratio >= validation_ratio {
			p.valid = true
		}
	}
}

func (p *Person) Update(t time.Time, box image.Rectangle, descriptor []float32) error {
	p.total_hits++
	p.descriptors.Push(descriptor)
	p.filter.Update(center(box), t)
	p.trajectory.Push(p.sma.Recalc(p.filter.State()))
	p.last_update = t
	p.last_box = box
	return nil
}

func (p *Person) Predict(t time.Time, prediction_duration time.Duration) error {
	p.total_misses++
	if t.Sub(p.last_update) < prediction_duration {
		p.filter.Predict(t)
	}
	p.trajectory.Push(p.sma.Recalc(p.filter.State()))
	p.last_box = image.Rect(0, 0, 0, 0)
	return nil
}

func (p *Person) State() image.Point {
	return p.trajectory.Newest()
}

func (p *Person) DrawTrajectory(m *gocv.Mat, w int, alpha uint8) {
	c := p.Color()
	c.A = alpha
	prev_point := p.State()
	for point := range p.Trajectory() {
		if (image.Point{}) != prev_point {
			gocv.Line(m, prev_point, point, c, w)
		}
		prev_point = point
	}
}

func (p *Person) DrawCross(m *gocv.Mat, w, r int, alpha uint8) {
	c := p.Color()
	c.A = alpha
	gocv.Line(m, p.State().Add(image.Pt(-r, -r)), p.State().Add(image.Pt(r, r)), c, w)
	gocv.Line(m, p.State().Add(image.Pt(-r, r)), p.State().Add(image.Pt(r, -r)), c, w)
}

func (p *Person) DrawBox(m *gocv.Mat, w int) {
	if !p.last_box.Empty() {
		gocv.Rectangle(m, p.last_box, p.Color(), w)
	}
}

func (p *Person) Distance(box image.Rectangle) float64 {
	return vecLen(p.State().Sub(center(box)))
}

type ExportedPerson struct {
	Id string `json:"id"`
	X  uint   `json:"x"`
	Y  uint   `json:"y"`
}

func (p *Person) Export() *ExportedPerson {
	return &ExportedPerson{
		Id: p.Id(),
		X:  uint(p.State().X),
		Y:  uint(p.State().Y),
	}
}

