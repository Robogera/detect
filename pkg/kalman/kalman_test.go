package kalman

import (
	"context"
	"image"
	"image/color"
	"math"
	"math/rand/v2"
	"net/http"
	"testing"
	"time"

	"github.com/Robogera/detect/pkg/gsma"
	"github.com/hybridgroup/mjpeg"
	"gocv.io/x/gocv"
)

func normSin(f, t float64) float64 { return (math.Sin(f*math.Pi*2*t)*0.8 + 1) / 2 }
func normCos(f, t float64) float64 { return (math.Cos(f*math.Pi*2*t)*0.8 + 1) / 2 }

func toJpeg(img gocv.Mat) []byte {
	buf, err := gocv.IMEncode(gocv.JPEGFileExt, img)
	if err != nil {
		return nil
	}
	data := make([]byte, buf.Len())
	copy(data, buf.GetBytes())
	return data
}

const W = 1920
const H = 1080

func TestSanity(t *testing.T) {
	t0 := time.Now()
	output_stream := mjpeg.NewStream()

	http.Handle("/", output_stream)

	server := &http.Server{
		Addr:         "0.0.0.0:8080",
		ReadTimeout:  0,
		WriteTimeout: 0,
	}

	err_chan := make(chan error)

	go func() {
		err_chan <- server.ListenAndServe()
	}()

	img := gocv.NewMatWithSize(H, W, gocv.MatTypeCV8UC3)
	defer img.Close()

	set := false
	var kf *Filter
	var avg, meas, noisy, prev image.Point
	sma := gsma.NewSMA2d(4)
	for i := range 5000 {
		<-time.NewTimer(time.Millisecond * time.Duration(80*(0.75+rand.Float32()/2))).C
		tm := time.Now().Sub(t0)
		meas = image.Pt(
			int(normSin(1.0/17.0, tm.Seconds())*W),
			int(normCos(1.0/5.0, tm.Seconds())*H),
		)
		noisy = meas.Add(image.Pt(int(rand.NormFloat64()*30), int(rand.NormFloat64()*30)))
		if !set {
			kf = NewFilter(noisy, time.Now(), 0.001, 900)
		}
		if i < 200 || 210 < i {
			kf.Update(noisy, time.Now())
		} else {
			kf.Predict(time.Now())
		}
		// t.Logf("Meas: %v, Filtered: %v", noisy, kf.State())
		// t.Logf("Noise: %v", kf.Noise())
		// t.Logf("Process: %v", kf.Process())
		img.SubtractUChar(0b100001)
		gocv.Circle(&img, noisy, 2, color.RGBA{255, 0, 0, 255}, 3)
		avg = sma.Recalc(kf.State())
		t.Logf("Avg: %v", avg)
		if set {
			gocv.Line(&img, prev, avg, color.RGBA{0, 0, 255, 255}, 2)
		}
		prev = avg
		set = true
		// gocv.Line(&img, kf.State(), kf.State().Add(kf.Speed().Mul(10)), color.RGBA{0, 255, 0, 255}, 1)
		output_stream.UpdateJPEG(toJpeg(img))
	}
	shutdown_context, cancel := context.WithTimeout(
		context.Background(),
		time.Second*1)
	defer cancel()
	server.Shutdown(shutdown_context)
}
