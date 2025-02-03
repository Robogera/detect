package gmat

import (
	"math/rand/v2"
	"testing"
)

func coolMatrix(r, c, n int) *Mat[int] {
	m := NewMat[int](r, c)
	for ind_r := range r {
		for ind_c := range c {
			m.Set(ind_r, ind_c, rand.IntN(n))
		}
	}
	return m
}

func iterateOverVectors[T ~int | ~float64](t *testing.T, m *Mat[T], vertical Direction) T {
	var row_sum T
	for vec_ind, vec := range m.Vectors(vertical) {
		t.Logf("Vec ind: %d, vec: %v", vec_ind, vec.index)
		for ind, value := range vec.All() {
			t.Logf("value @ %d: %v", ind, value)
			row_sum += value
		}
	}
	return row_sum
}

func TestNewMat(t *testing.T) {
	m := coolMatrix(11, 13, 20)
	t.Log("m1:\n" + m.String())
	col_sum := iterateOverVectors(t, m, true)
	row_sum := iterateOverVectors(t, m, false)
	if row_sum != col_sum {
		t.Fatalf("Col and row iteration result varies (%d vs %d)", col_sum, row_sum)
	}
}

func TestMap(t *testing.T) {
	m := coolMatrix(10, 10, 100)
	mapped := Map(m, func(e int, r, c int) float64 {
		ret := float64(e) / 100.0
		if r == c {
			ret = 1.0
		}
		return ret
	})
	t.Log("\n" + m.String())
	t.Log("\n" + mapped.String())
}

func TestMask(t *testing.T) {
	m := coolMatrix(10, 10, 100)
	m2 := m.
		Mask(Vertical, 0, 1, 2).
		Mask(Horizontal, 0, 1, 2).
		Mask(Horizontal, 9, 8, 7)
	t.Log("m1:\n" + m.String())
	t.Log("m2:\n" + m2.String())
}
