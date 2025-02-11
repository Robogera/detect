package gmat

import (
	"errors"
	"fmt"
	"iter"
	"strings"
	"text/tabwriter"
)

var (
	ERR_OOB = errors.New("Index out of bounds")
)

type Direction bool

const (
	Vertical   Direction = true
	Horizontal Direction = false
)

// Matrix with the ability to quickly
// delete (mask) rows or columns
type Mat[T any] struct {
	s      []T
	stride int
}

// Vector backed by the data of the
// underlying matrix
type Vector[T any] struct {
	Mat[T]
	index     int
	direction Direction
}

func (m Mat[T]) At(r, c int) T {
	return m.s[r*m.stride+c]
}

func (m Mat[T]) Size(direction Direction) int {
	if direction == Vertical {
		return len(m.s) / m.stride
	}
	return m.stride
}

// Iterator over the unmasked rows/columns of the
// reciever as vectors
func (m Mat[T]) Vectors(direction Direction) iter.Seq2[int, Vector[T]] {
	return func(yield func(int, Vector[T]) bool) {
		iterate_over := m.stride
		if iterate_over < 1 {
			iterate_over = 0
		} else if direction == Horizontal {
			iterate_over = len(m.s) / m.stride
		}
		for ind := range iterate_over {
			if !yield(ind, Vector[T]{
				Mat: m, index: ind,
				direction: direction,
			}) {
				return
			}
		}
	}
}

func (m Mat[T]) Vector(direction Direction, index int) (Vector[T], error) {
	if (direction == Horizontal && index >= len(m.s)/m.stride) || (direction == Vertical && index >= m.stride) {
		return Vector[T]{}, ERR_OOB
	}
	return Vector[T]{
		Mat: m, index: index,
		direction: direction,
	}, nil
}

// Returns element of the receiver
// at index
func (v Vector[T]) At(index int) T {
	if v.direction == Vertical {
		return v.Mat.s[v.Mat.stride*index+v.index]
	} else {
		return v.Mat.s[v.Mat.stride*v.index+index]
	}
}

func (v *Vector[T]) Set(index int, value T) {
	if v.direction == Vertical {
		v.Mat.s[v.Mat.stride*index+v.index] = value
	} else {
		v.Mat.s[v.Mat.stride*v.index+index] = value
	}
}

// Iterate over the unmasked values of vector
func (v Vector[T]) All() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		iterate_over := v.Mat.stride
		if v.direction == Vertical {
			iterate_over = len(v.Mat.s) / v.Mat.stride
		}
		for ind := range iterate_over {
			if !yield(ind, v.At(ind)) {
				return
			}
		}
	}
}

// Returns a new matrix with pre-allocated
// backing slice
func NewMat[T any](r, c int) *Mat[T] {
	return &Mat[T]{
		s:      make([]T, r*c),
		stride: c,
	}
}

// Maps an existing matrix into a new one via f
func Map[T, E any](m *Mat[T], f func(e T, r, c int) E) *Mat[E] {
	new_mat := &Mat[E]{
		s:      make([]E, len(m.s)),
		stride: m.stride,
	}
	for ind_r, vec := range m.Vectors(false) {
		for ind_c, value := range vec.All() {
			new_mat.Set(ind_r, ind_c, f(value, ind_r, ind_c))
		}
	}
	return new_mat
}

// Set the value of element (r, c) in matrix m
func (m *Mat[T]) Set(r, c int, v T) error {
	if r >= len(m.s)/m.stride || c >= m.stride {
		return ERR_OOB
	}
	m.s[m.stride*r+c] = v
	return nil
}

// Pretty print
func (m Mat[T]) Sprintf(format string) string {
	b := new(strings.Builder)
	t := tabwriter.NewWriter(b, 3, 1, 1, ' ', 0)
	for _, vec := range m.Vectors(false) {
		for _, value := range vec.All() {
			fmt.Fprintf(t, format, value)
			fmt.Fprint(t, "\t")
		}
		fmt.Fprint(t, "\n")
	}
	t.Flush()
	return b.String()
}

func (m Mat[T]) String() string {
	return m.Sprintf("%v")
}

func (v Vector[T]) Sprintf(format string) string {
	b := new(strings.Builder)
	b.WriteString("[ ")
	for _, value := range v.All() {
		b.WriteString(fmt.Sprintf(format, value))
	}
	b.WriteString("]")
	return b.String()
}

func (v Vector[T]) String() string {
	return v.Sprintf("%v ")
}

func (m Mat[T]) To2d() [][]T {
	if m.stride < 1 {
		return make([][]T, 0)
	}
	s := make([][]T, len(m.s)/m.stride)
	for r := range len(s) {
		s[r] = m.s[m.stride*r : m.stride*(r+1)]
	}
	return s
}
