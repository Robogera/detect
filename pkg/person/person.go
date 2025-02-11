package person

import (
	"fmt"
	"image"
	"image/color"
	"iter"
	"slices"
	"time"

	// "github.com/Robogera/detect/pkg/ghung"
	"github.com/Robogera/detect/pkg/gmat"
	gocvcommon "github.com/Robogera/detect/pkg/gocv-common"
	"github.com/Robogera/detect/pkg/gring"
	"github.com/Robogera/detect/pkg/seq"
	hung "github.com/arthurkushman/go-hungarian"
	"github.com/muesli/gamut"
	"github.com/rosshemsley/kalman"
	"github.com/rosshemsley/kalman/models"
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

type Associator struct {
	p                 map[string]*Person
	temp_enumeration  []*Person
	net               *gocv.Net
	output_layer_name string
	conv_params       *gocv.ImageToBlobParams
	color             color.Color
}

func NewAssociator(net *gocv.Net, conv_params *gocv.ImageToBlobParams, output_layer_name string) (*Associator, error) {
	if !gocvcommon.CheckLayerName(net, output_layer_name) {
		return nil, fmt.Errorf("Model has no layer %s", output_layer_name)
	}
	return &Associator{
		p:                 make(map[string]*Person, 0),
		net:               net,
		output_layer_name: output_layer_name,
		conv_params:       conv_params,
		color:             color.RGBA{255, 0, 0, 255},
	}, nil
}

func (a *Associator) ReenumeratePeople() {
	a.temp_enumeration = make([]*Person, len(a.p))
	// ks := make([]string, 0)
	// for k := range a.p {
	// 	ks = append(ks, k)
	// }
	// slices.Sort(ks)
	i := 0
	for _, p := range a.p {
		a.temp_enumeration[i] = p
		i++
	}
}

func (a *Associator) EnumeratedPeople() iter.Seq2[int, *Person] {
	return func(yield func(int, *Person) bool) {
		for i, person := range a.temp_enumeration {
			if !yield(i, person) {
				return
			}
		}
	}
}

func (a *Associator) GetByNumber(i int) *Person {
	if 0 < i || i < len(a.temp_enumeration) {
		return nil
	}
	return a.temp_enumeration[i]
}

func (a *Associator) Add(p *Person) {
	a.p[p.Id()] = p
}

func (a *Associator) Del(id string) {
	delete(a.p, id)
}

func (a *Associator) Associate(m *gocv.Mat, boxes []image.Rectangle, t time.Time) {
	detected_descriptors := seq.SMap(boxes,
		func(r image.Rectangle, i int) []float32 {
			region := m.Region(r)
			defer region.Close()
			blob := gocv.BlobFromImageWithParams(region, *a.conv_params)
			defer blob.Close()
			a.net.SetInput(blob, "")
			output := a.net.Forward(a.output_layer_name)
			defer output.Close()
			ptr, err := output.DataPtrFloat32()
			if err != nil {
				fmt.Printf("error: %s", err)
			}

			raw_data := make([]float32, len(ptr))
			copy(raw_data, ptr)
			return raw_data
		})
	a.ReenumeratePeople()
	diss_mat := gmat.NewMat[float64](max(len(a.p), len(detected_descriptors)), max(len(detected_descriptors), len(a.p)))
	for person_id, person := range a.EnumeratedPeople() {
		for detection_id, detected_descriptor := range detected_descriptors {
			dissimilarity := make([]float64, 0, person.descriptors.Size())
			for persons_descriptor := range person.descriptors.All() {
				dissimilarity = append(dissimilarity, float64(seq.CosSim(persons_descriptor, detected_descriptor)))
			}
			diss_mat.Set(person_id, detection_id, slices.Max(dissimilarity))
		}
	}
	// fmt.Printf("dissimilarity matrix:\n%s\n", diss_mat.Sprintf("%.4f"))
	ass := hung.SolveMax(diss_mat.To2d())
	// ass := ghung.Solve(diss_mat)
	associated_boxes := make(map[int]struct{})
	for person_ind, person := range a.EnumeratedPeople() {
		assoc_box_ind := 0
		score := float64(0)
		for assoc_box_ind, score = range ass[person_ind] {
			break
		}
		if assoc_box_ind >= len(detected_descriptors) {
			// fmt.Printf("Person %s has no real match (matched with index %d which was used for padding the matrix)\n", person.Id(), assoc_box_ind)
			person.Predict(t)
		} else {
			associated_boxes[assoc_box_ind] = struct{}{}
			fmt.Printf("Associated person %s (#%d) with box %d - score %.4f\n", person.Id(), person_ind, assoc_box_ind, score)
			person.Update(t, boxes[assoc_box_ind], detected_descriptors[assoc_box_ind])
			gocv.Rectangle(m, boxes[assoc_box_ind], person.Color(), 3)
		}
	}
	for box_ind, box := range boxes {
		if _, associated := associated_boxes[box_ind]; !associated {
			fmt.Printf("Adding a person for box #%d...\n", box_ind)
			new_person, _ := a.NewPerson(t, box, detected_descriptors[box_ind])
			a.p[new_person.Id()] = new_person
		}
	}
}

func (a *Associator) CleanUp(t time.Duration, bounds image.Rectangle) {
	for _, person := range a.p {
		if person.NotUpdatedFor(t) {
			fmt.Printf("Person %s: tracker expired after %s\n", person.Id(), t)
			a.Del(person.Id())
		} else if !person.State().In(bounds) {
			fmt.Printf("Person %s: out of bounds\n", person.Id())
			a.Del(person.Id())
		}
	}
}

func (a *Associator) NewPerson(t time.Time, box image.Rectangle, descriptor []float32) (*Person, error) {
	a.color = gamut.HueOffset(a.color, 133)
	r, g, b, _ := a.color.RGBA()
	model := models.NewConstantVelocityModel(
		t, pointToVec(center(box)),
		models.ConstantVelocityModelConfig{
			InitialVariance: 0.01,
			ProcessVariance: 0.01,
		})
	descriptors := gring.NewRing[[]float32](3)
	descriptors.Push(descriptor)
	return &Person{
		id:          generateToken(4),
		last_update: t,
		trajectory:  gring.NewRing[image.Point](40),
		color:       color.RGBA{uint8(r), uint8(g), uint8(b), 255},
		descriptors: descriptors,
		model:       model,
		filter:      kalman.NewKalmanFilter(model),
	}, nil
}

type Person struct {
	id          string
	last_update time.Time
	trajectory  *gring.Ring[image.Point]
	color       color.RGBA
	descriptors *gring.Ring[[]float32]
	model       *models.ConstantVelocityModel
	filter      *kalman.KalmanFilter
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
	p.trajectory.Push(vecToPoint(
		p.model.Position(
			p.filter.State())))
	p.descriptors.Push(descriptor)
	err := p.filter.Update(t, p.model.NewPositionMeasurement(pointToVec(center(box)), 0.1))
	if err != nil {
		return fmt.Errorf("Can't update person %s. Error: %w", p.id, err)
	}
	p.last_update = t
	return nil
}

func (p *Person) Predict(t time.Time) error {
	p.trajectory.Push(vecToPoint(
		p.model.Position(
			p.filter.State())))
	err := p.filter.Predict(t)
	if err != nil {
		return fmt.Errorf("Can't predict person %s. Error: %w", p.id, err)
	}
	return nil
}

func (p *Person) State() image.Point {
	return vecToPoint(
		p.model.Position(
			p.filter.State()))
}
