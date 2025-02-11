package ghung

import (
	"math/rand/v2"
	"testing"

	"github.com/Robogera/detect/pkg/gmat"
)

func coolMatrix(r, c int, scale float32) *gmat.Mat[float32] {
	m := gmat.NewMat[float32](r, c)
	for ind_r := range r {
		for ind_c := range c {
			m.Set(ind_r, ind_c, rand.Float32()*scale)
		}
	}
	return m
}

func TestSubMin(t *testing.T) {
	m := coolMatrix(4, 4, 400)
	t.Logf("m:\n%s\n", m)
	Solve(m)
	t.Logf("m:\n%s\n", m)
}

func TestSanity(t *testing.T) {
	m := coolMatrix(3, 3, 0.5)
	t.Logf("m:\n%s\n", m)
	s := Solve(m)
	t.Logf("2d:\n%v\n", m.To2d())
	t.Logf("m:\n%v\n", s)
}
