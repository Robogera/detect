package seq

import (
	"cmp"
	"iter"
	"math"
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
	for range cap(seq) {
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
			set = true
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
			set = true
		}
	}
	return current_min_ind, current_min
}

func SMap[I any, T any](s []T, f func(T, int) I) []I {
	ss := make([]I, 0)
	for i, v := range s {
		ss = append(ss, f(v, i))
	}
	return ss
}

func CosSim[T Float](a, b []T) T {
	if len(a) != len(b) {
		return 0
	}
	var sum_a, sum_b, sum_mul T
	for i := range len(a) {
		sum_mul += a[i] * b[i]
		sum_a += a[i] * a[i]
		sum_b += b[i] * b[i]
	}
	return sum_mul / sum_a / sum_b
}

func Sum[T cmp.Ordered](s []T) (sum T) {
	for _, v := range s {
		sum += v
	}
	return
}

func SqrtMean[T Float | Int](s []T) float64 {
	var mean float64
	for _, v := range s {
		mean += float64(v * v)
	}
	mean /= float64(len(s))
	return math.Sqrt(mean)
}
