package person

import (
	"math/rand/v2"
	"testing"

	"github.com/Robogera/detect/pkg/gmat"
	hung "github.com/arthurkushman/go-hungarian"
)

func coolMatrix(r, c int, scale float64) *gmat.Mat[float64] {
	m := gmat.NewMat[float64](r, c)
	for ind_r := range r {
		for ind_c := range c {
			m.Set(ind_r, ind_c, rand.Float64()*scale)
		}
	}
	return m
}

func TestAssociate(t *testing.T) {
	m := coolMatrix(3,3,1)
	t.Logf("m:\n%s\n", m)
	a := hung.SolveMin(m.To2d())
	t.Logf("a:\n%v\n", a)
}

func TestImage(t *testing.T) {

}
