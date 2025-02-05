package assoc

import (
	"image"
	"image/color"
	"math/rand/v2"
	"testing"

	"gocv.io/x/gocv"
)

func BenchmarkAssoc(b *testing.B) {

	pred := make([]image.Point,0, 4)
	det := make([]image.Point,0, 8)

	for _ = range 4 {
		pred = append(pred, image.Pt(
			rand.IntN(600),
			rand.IntN(600),
		))
	}

	for _, point := range pred {
		det = append(
			det,
			point.Add(image.Pt(
				rand.IntN(12),
				rand.IntN(12),
			)),
		)
	}

	for _ = range 4 {
		det = append(det, image.Pt(
			rand.IntN(600),
			rand.IntN(600),
		))
	}
	b.ResetTimer()
	for range b.N {
		Associate(pred, det, 600)
	}
}

func TestAssoc(t *testing.T) {

	pred := make([]image.Point, 0)
	det := make([]image.Point, 0)

	for _ = range rand.IntN(4) {
		pred = append(pred, image.Pt(
			rand.IntN(600),
			rand.IntN(600),
		))
	}

	for _, point := range pred {
		det = append(
			det,
			point.Add(image.Pt(
				rand.IntN(12),
				rand.IntN(12),
			)),
		)
	}
	for _ = range rand.IntN(3) {
		det = append(det, image.Pt(
			rand.IntN(600),
			rand.IntN(600),
		))
	}

	assocs := Associate(pred, det, 129)
	for _, assoc := range assocs {
		t.Log(assoc.Pred, assoc.Det)
	}

	mat := gocv.NewMatWithSize(600, 600, gocv.MatTypeCV8UC3)
	for _, det_point := range det {
		gocv.Circle(&mat, det_point, 2, color.RGBA{255, 0, 0, 255}, 2)
	}
	for _, pred_point := range pred {
		gocv.Circle(&mat, pred_point, 2, color.RGBA{0, 255, 0, 255}, 2)
	}

	for _, assoc := range assocs {
		t.Logf("Found assoc %d:%d", assoc.Pred, assoc.Det)
		gocv.Line(&mat, pred[assoc.Pred], det[assoc.Det], color.RGBA{0, 0, 255, 127}, 1)
	}

	gocv.IMWrite("results.jpg", mat)
}
