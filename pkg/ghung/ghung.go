package ghung

import (
	"fmt"

	"github.com/Robogera/detect/pkg/gmat"
	"github.com/Robogera/detect/pkg/seq"
)

func Solve[T seq.Float](m *gmat.Mat[T]) map[int]map[int]T {
	perms := [][]int{
		{0, 1, 2},
		{0, 2, 1},
		{1, 0, 2},
		{1, 2, 0},
		{2, 0, 1},
		{2, 1, 0},
	}
	var min_sum T
	var min_perm []int
	set := false
	for _, perm := range perms {
		var sum T
		for r, c := range perm {
			sum += m.At(r, c)
		}
		if sum < min_sum || !set {
			min_sum = sum
			min_perm = perm
			set = true
		}
	}
	ass := make(map[int]map[int]T)
	for r, c := range min_perm {
		ass[r] = make(map[int]T)
		ass[r][c] = m.At(r,c)
	}
	return ass
}

func subtractMin[T seq.Float](m *gmat.Mat[T], d gmat.Direction) {
	for _, vec := range m.Vectors(d) {
		_, min_elem := seq.MinInd(vec.All())
		fmt.Printf("Min: %f", min_elem)
		for ind, value := range vec.All() {
			vec.Set(ind, value-min_elem)
		}
	}
}
