package seq

import (
	"cmp"
	"iter"
)

type Int interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

type Uint interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

type Float interface {
	~float32 | ~float64
}

func seq(n int) []int {
	ret := make([]int, n)
	for i := range n {
		ret[i] = i
	}
	return ret
}

func Seq[T Int | Uint | Float](floor, ceiling, delta T) []T {
	seq := make([]T, 0, int(ceiling-floor/delta))
	for value := floor; value < ceiling; value += delta {
		seq = append(seq, value)
	}
	return seq
}

func SeqN[T Int | Uint](n T) []T {
	seq := make([]T, 0, int(n))
	var index T = 0
	for _ = range cap(seq) {
		seq = append(seq, index)
		index++
	}
	return seq
}

func MaxInd[I any, T cmp.Ordered](it iter.Seq2[I, T]) (I, T) {
	var set bool
	var current_max T
	var current_max_ind I
	for i, v := range it {
		if !set || v > current_max {
			current_max_ind = i
			current_max = v
		}
	}
	return current_max_ind, current_max
}

func MinInd[I any, T cmp.Ordered](it iter.Seq2[I, T]) (I, T) {
	var set bool
	var current_min T
	var current_min_ind I
	for i, v := range it {
		if !set || v < current_min {
			current_min_ind = i
			current_min = v
		}
	}
	return current_min_ind, current_min
}
