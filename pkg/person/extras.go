package person

import (
	"crypto/rand"
	"encoding/hex"
	"image"
	"math"

	"gonum.org/v1/gonum/mat"
)

func vecToPoint(vec mat.Vector) image.Point {
	return image.Pt(
		int(vec.AtVec(0)),
		int(vec.AtVec(1)),
	)
}

func pointToVec(point image.Point) mat.Vector {
	return mat.NewVecDense(2, []float64{
		float64(point.X),
		float64(point.Y),
	})
}

func center(r image.Rectangle) image.Point {
	return image.Pt(
		(r.Max.X+r.Min.X)/2,
		(r.Max.Y+r.Min.Y)/2,
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

func vecLen(v image.Point) float64 {
	return math.Sqrt(math.Pow(float64(v.X), 2) + math.Pow(float64(v.Y), 2))
}
