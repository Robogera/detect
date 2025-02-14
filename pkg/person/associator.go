package person

import (
	"fmt"
	"image"
	"image/color"
	"iter"
	"slices"
	"time"

	"github.com/Robogera/detect/pkg/config"
	"github.com/Robogera/detect/pkg/gmat"
	gocvcommon "github.com/Robogera/detect/pkg/gocv-common"
	"github.com/Robogera/detect/pkg/seq"
	hung "github.com/arthurkushman/go-hungarian"
	"gocv.io/x/gocv"
)

type Associator struct {
	p                              map[string]*Person
	temp_enumeration               []*Person
	net                            *gocv.Net
	output_layer_name              string
	conv_params                    *gocv.ImageToBlobParams
	color                          color.Color
	speed_threshold                float64
	sma_window                     int
	frames_to_follow               int
	ttl                            float64
	cfg                            *config.ConfigFile
	proc_noise_cov, meas_noise_cov float64
}

func NewAssociator(net *gocv.Net, conv_params *gocv.ImageToBlobParams, cfg *config.ConfigFile) (*Associator, error) {
	if !gocvcommon.CheckLayerName(net, cfg.Reid.OutputLayerName) {
		return nil, fmt.Errorf("Model has no layer %s", cfg.Reid.OutputLayerName)
	}
	return &Associator{
		p:                 make(map[string]*Person, 0),
		net:               net,
		output_layer_name: cfg.Reid.OutputLayerName,
		conv_params:       conv_params,
		color:             color.RGBA{255, 0, 0, 255},
		speed_threshold:   cfg.Reid.SpeedThreshold,
		sma_window:        cfg.Reid.SMAWindow,
		frames_to_follow:  cfg.Reid.FramesToFollow,
		ttl:               cfg.Reid.TTL,
		proc_noise_cov:    cfg.Kalman.ProcessNoiseCov,
		meas_noise_cov:    cfg.Kalman.MeasNoiseCov,
	}, nil
}

func (a *Associator) ReenumeratePeople() {
	a.temp_enumeration = make([]*Person, len(a.p))
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

func (a *Associator) Associate(m *gocv.Mat, boxes []image.Rectangle, t time.Time, threshold float64) map[string]PersonStatus {
	size := m.Size()
	frame := image.Rect(0, 0, size[1], size[0])
	detected_descriptors := make([][]float32, 0, len(boxes))
	for _, box := range boxes {
		func() {
			if !box.In(frame) {
				box = box.Intersect(frame)
			}
			region := m.Region(box)
			defer region.Close()
			blob := gocv.BlobFromImageWithParams(region, *a.conv_params)
			defer blob.Close()
			a.net.SetInput(blob, "")
			output := a.net.Forward(a.output_layer_name)
			defer output.Close()
			ptr, err := output.DataPtrFloat32()
			if err != nil {
				return
			}
			raw_data := make([]float32, len(ptr))
			copy(raw_data, ptr)
			detected_descriptors = append(detected_descriptors, raw_data)
		}()
	}
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
	ass := hung.SolveMax(diss_mat.To2d())
	associated_boxes := make(map[int]struct{})
	updates := make(map[string]PersonStatus, len(a.p))
	for person_ind, person := range a.EnumeratedPeople() {
		assoc_box_ind := 0
		var score float64
		for assoc_box_ind, score = range ass[person_ind] {
			break
		}
		if assoc_box_ind >= len(detected_descriptors) {
			person.Predict(t)
			updates[person.Id()] = PersonStatusNoAss{}
		} else if score < threshold {
			person.Predict(t)
			updates[person.Id()] = PersonStatusNoAssLowScore{score: score}
		} else if moved, threshold := vecLen(person.State().Sub(center(boxes[assoc_box_ind]))), a.speed_threshold*time.Now().Sub(person.last_update).Seconds(); moved > threshold {
			updates[person.Id()] = PersonStatusNoAssTooFar{score: score, dst: moved}
			person.Predict(t)
		} else {
			associated_boxes[assoc_box_ind] = struct{}{}
			updates[person.Id()] = PersonStatusAssociated{ass: assoc_box_ind, dst: moved, score: score}
			person.Update(t, boxes[assoc_box_ind], detected_descriptors[assoc_box_ind])
			// gocv.Rectangle(m, boxes[assoc_box_ind], person.Color(), 1)
		}
	}
	for box_ind, box := range boxes {
		if _, associated := associated_boxes[box_ind]; !associated {
			new_person, _ := a.NewPerson(t, box, detected_descriptors[box_ind])
			updates[new_person.Id()] = PersonStatusNew{ass: box_ind}
			a.p[new_person.Id()] = new_person
		}
	}
	return updates
}

func (a *Associator) CleanUp(bounds image.Rectangle) map[string]PersonStatus {
	deletions := make(map[string]PersonStatus, 0)
	for _, person := range a.p {
		if threshold := time.Duration(a.ttl) * time.Second; person.NotUpdatedFor(threshold) {
			deletions[person.Id()] = PersonStatusDeletedNoUpdates{t: threshold}
			a.Del(person.Id())
		} else if !person.State().In(bounds) {
			deletions[person.Id()] = PersonStatusDeletedOOB{}
			a.Del(person.Id())
		}
	}
	return deletions
}
