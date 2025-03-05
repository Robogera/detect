package person

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"github.com/Robogera/detect/pkg/config"
	"github.com/Robogera/detect/pkg/gmat"
	gocvcommon "github.com/Robogera/detect/pkg/gocv-common"
	"github.com/Robogera/detect/pkg/seq"
	hung "github.com/arthurkushman/go-hungarian"
	"gocv.io/x/gocv"
)

type Detection struct {
	Box        image.Rectangle
	Descriptor []float32
	Associated bool
}

type Associator struct {
	p           map[string]*Person
	net         *gocv.Net
	conv_params *gocv.ImageToBlobParams

	validation_duration          time.Duration
	prediction_duration          time.Duration
	expiration_duration          time.Duration
	nonvalid_expiration_duration time.Duration

	cfg *config.ConfigFile
	// visual stuff
	trajectory_points uint
	next_color        color.Color
}

func NewAssociator(net *gocv.Net, conv_params *gocv.ImageToBlobParams, cfg *config.ConfigFile) (*Associator, error) {
	if !gocvcommon.CheckLayerName(net, cfg.Reid.OutputLayerName) {
		return nil, fmt.Errorf("Model has no layer %s", cfg.Reid.OutputLayerName)
	}
	return &Associator{
		p:                            make(map[string]*Person, 0),
		net:                          net,
		conv_params:                  conv_params,
		validation_duration:          time.Duration(cfg.Reid.ValidateSec) * time.Second,
		expiration_duration:          time.Duration(cfg.Reid.ExpireSec) * time.Second,
		nonvalid_expiration_duration: time.Duration(cfg.Reid.NonValidExpireSec) * time.Second,
		prediction_duration:          time.Duration(cfg.Reid.PredictSec) * time.Second,
		cfg:                          cfg,
		trajectory_points:            25,
		next_color:                   color.RGBA{255, 0, 0, 255},
	}, nil
}

func (a *Associator) EnumeratePeople() []*Person {
	people := make([]*Person, 0, len(a.p))
	for _, person := range a.p {
		people = append(people, person)
	}
	return people
}

func (a *Associator) del(id string) {
	a.p[id].filter.Free()
	delete(a.p, id)
}

func (a *Associator) Associate(
	m *gocv.Mat,
	boxes []image.Rectangle,
	t time.Time,
	f func(score, dist float64) float64,
) {

	size := m.Size()
	frame := image.Rect(0, 0, size[1], size[0])
	detections := make([]*Detection, 0, len(boxes))

	for _, box := range boxes {
		func() {
			if !box.In(frame) {
				box = box.Intersect(frame)
			}
			// TODO: use a fraction of the frame's height or something
			// like that for a better threshold
			if box.Dx() < 1 || box.Dy() < 1 {
				return
			}
			region := m.Region(box)
			defer region.Close()
			fmt.Printf("Region:%v\n", region.Size())
			blob := gocv.BlobFromImageWithParams(region, *a.conv_params)
			defer blob.Close()
			a.net.SetInput(blob, "")
			output := a.net.Forward(a.cfg.Reid.OutputLayerName)
			defer output.Close()
			ptr, err := output.DataPtrFloat32()
			if err != nil {
				return
			}
			raw_data := make([]float32, len(ptr))
			copy(raw_data, ptr)
			detections = append(detections, &Detection{
				Box:        box,
				Descriptor: raw_data,
				Associated: false,
			})
		}()
	}

	matrix_size := max(len(a.p), len(detections))
	diss_mat := gmat.NewMat[float64](matrix_size, matrix_size)

	enumerated := a.EnumeratePeople()

	for pid, person := range enumerated {
		for did, detection := range detections {
			scores := make([]float64, 0, person.descriptors.Size())
			for descriptor := range person.descriptors.All() {
				scores = append(scores, float64(seq.CosSim(descriptor, detection.Descriptor)))
			}
			score := seq.SqrtMean(scores)
			// punishing distant pairs with lower score
			if overshoot := person.distance(detection.Box) - float64(a.cfg.Reid.DistanceThreshold); overshoot > 0 {
				score /= overshoot * a.cfg.Reid.DistanceFactor
			}
			diss_mat.Set(pid, did, score)
		}
	}

	associations := hung.SolveMax(diss_mat.To2d())

	for pid, person := range enumerated {
		var did int
		var score float64
		// ugly syntax but kept for compatibility with the third party
		// kahn algo packages
		for did, score = range associations[pid] {
			break
		}
		if did >= len(detections) || score < a.cfg.Reid.ScoreThreshold {
			// the first condition means he got associated with a "dummy"
			// detection column
			person.predict(t, a.prediction_duration)
		} else {
			detections[did].Associated = true
			person.update(t, detections[did].Box, detections[did].Descriptor)
		}
		person.validate(t, a.validation_duration, a.cfg.Reid.ValidationFrames)
	}
	for _, detection := range detections {
		if !detection.Associated {
			new_person, _ := a.NewPerson(t, detection.Box, detection.Descriptor)
			a.p[new_person.Id()] = new_person
		}
	}
}

func (a *Associator) CleanUp(t time.Time, bounds image.Rectangle) {
	for _, person := range a.p {
		if person.Status() == STATUS_EXPIRED {
		} else {
			since_update := person.SinceDetection(t)
			if !person.IsValid() && since_update > a.nonvalid_expiration_duration {
				person.last_status = STATUS_EXPIRED
			} else if person.IsValid() && since_update > a.expiration_duration {
				person.last_status = STATUS_EXPIRED
			}
		}
	}
}

func (a *Associator) TotalPeople() int {
	return len(a.p)
}
