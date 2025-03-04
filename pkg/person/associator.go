package person

import (
	"fmt"
	"image"
	"image/color"
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
	p                   map[string]*Person
	net                 *gocv.Net
	output_layer_name   string
	conv_params         *gocv.ImageToBlobParams
	min_score           float64
	sma_window          uint
	validation_duration time.Duration
	prediction_duration time.Duration
	expiration_duration time.Duration
	proc_noise_cov      float64
	meas_noise_cov      float64
	total_descriptors   uint
	token_length        uint
	validation_ratio    float64
	// visual stuff
	trajectory_points uint
	next_color        color.Color
}

func NewAssociator(net *gocv.Net, conv_params *gocv.ImageToBlobParams, cfg *config.ConfigFile) (*Associator, error) {
	if !gocvcommon.CheckLayerName(net, cfg.Reid.OutputLayerName) {
		return nil, fmt.Errorf("Model has no layer %s", cfg.Reid.OutputLayerName)
	}
	return &Associator{
		p:                   make(map[string]*Person, 0),
		net:                 net,
		output_layer_name:   cfg.Reid.OutputLayerName,
		conv_params:         conv_params,
		token_length:        6,
		sma_window:          cfg.Reid.SMAWindow,
		proc_noise_cov:      cfg.Kalman.ProcessNoiseCov,
		meas_noise_cov:      cfg.Kalman.MeasNoiseCov,
		min_score:           cfg.Reid.ScoreThreshold,
		validation_duration: time.Duration(cfg.Reid.ValidateSec) * time.Second,
		expiration_duration: time.Duration(cfg.Reid.ExpireSec) * time.Second,
		prediction_duration: time.Duration(cfg.Reid.PredictSec) * time.Second,
		total_descriptors:   cfg.Reid.TotalDescriptors,
		trajectory_points:   25,
		next_color:          color.RGBA{255, 0, 0, 255},
		validation_ratio:    cfg.Reid.ValidationRatio,
	}, nil
}

func (a *Associator) EnumeratePeople() []*Person {
	people := make([]*Person, 0, len(a.p))
	for _, person := range a.p {
		people = append(people, person)
	}
	return people
}

func (a *Associator) Add(p *Person) {
	a.p[p.Id()] = p
}

func (a *Associator) Del(id string) {
	delete(a.p, id)
}

func (a *Associator) Associate(
	m *gocv.Mat,
	boxes []image.Rectangle,
	t time.Time,
	f func(score, dist float64) float64,
) map[string]PersonStatus {
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
	diss_mat := gmat.NewMat[float64](max(len(a.p), len(detected_descriptors)), max(len(detected_descriptors), len(a.p)))
	enumerated := a.EnumeratePeople()
	for person_id, person := range enumerated {
		for detection_id, detected_descriptor := range detected_descriptors {
			scores := make([]float64, 0, person.descriptors.Size())
			for persons_descriptor := range person.descriptors.All() {
				scores = append(scores, float64(seq.CosSim(persons_descriptor, detected_descriptor)))
			}
			score := f(slices.Max(scores), person.Distance(boxes[detection_id]))
			diss_mat.Set(person_id, detection_id, score)
		}
	}
	ass := hung.SolveMax(diss_mat.To2d())
	associated_boxes := make(map[int]struct{})
	updates := make(map[string]PersonStatus, len(a.p))
	for person_ind, person := range enumerated {
		assoc_box_ind := 0
		var score float64
		for assoc_box_ind, score = range ass[person_ind] {
			break
		}
		if assoc_box_ind >= len(detected_descriptors) {
			person.Predict(t, a.prediction_duration)
			updates[person.Id()] = PersonStatusNoAss{}
		} else if score < a.min_score {
			person.Predict(t, a.prediction_duration)
			updates[person.Id()] = PersonStatusNoAssLowScore{score: score}
		} else {
			associated_boxes[assoc_box_ind] = struct{}{}
			updates[person.Id()] = PersonStatusAssociated{ass: assoc_box_ind, dst: person.Distance(boxes[assoc_box_ind]), score: score}
			person.Update(t, boxes[assoc_box_ind], detected_descriptors[assoc_box_ind])
		}
		person.Validate(t, a.validation_duration, a.validation_ratio)
	}
	for box_ind, box := range boxes {
		if _, associated := associated_boxes[box_ind]; !associated {
			new_person, _ := a.NewPerson(t, box, detected_descriptors[box_ind])
			updates[new_person.Id()] = PersonStatusNew{coord: new_person.State()}
			a.p[new_person.Id()] = new_person
		}
	}
	return updates
}

func (a *Associator) CleanUp(t time.Time, bounds image.Rectangle) map[string]PersonStatus {
	deletions := make(map[string]PersonStatus, 0)
	for _, person := range a.p {
		if since_update := person.SinceUpdate(t); since_update > a.expiration_duration {
			deletions[person.Id()] = PersonStatusDeletedNoUpdates{t: since_update}
			a.Del(person.Id())
		} else if !person.State().In(bounds) {
			deletions[person.Id()] = PersonStatusDeletedOOB{}
			a.Del(person.Id())
		}
	}
	return deletions
}

func (a *Associator) TotalPeople() int {
	return len(a.p)
}
