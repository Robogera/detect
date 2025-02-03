package gmat

import (
	"fmt"
	"iter"
	"slices"
	"strings"
	"text/tabwriter"

	"gonum.org/v1/gonum/mat"
)

type Direction bool

const (
	Vertical   Direction = true
	Horizontal Direction = false
)

// Matrix with the ability to quickly
// delete (mask) rows or columns
type Mat[T any] struct {
	s                        []T
	masked_rows, masked_cols []bool
	stride                   int
}

// Vector backed by the data of the
// underlying matrix
type Vector[T any] struct {
	Mat[T]
	index     int
	direction Direction
}

func (m Mat[T]) Size(direction Direction) int {
	if direction == Vertical {
		return len(m.masked_rows)
	} else {
		return len(m.masked_cols)
	}
}

// Iterator over the unmasked rows/columns of the
// reciever as vectors
func (m Mat[T]) Vectors(direction Direction) iter.Seq2[int, Vector[T]] {
	return func(yield func(int, Vector[T]) bool) {
		iterate_over := m.masked_rows
		if direction == Vertical {
			iterate_over = m.masked_cols
		}
		for ind, masked := range iterate_over {
			if masked {
				continue
			}
			if !yield(ind, Vector[T]{
				Mat: m, index: ind,
				direction: direction,
			}) {
				return
			}
		}
	}
}

// Returns element of the receiver
// at index
func (v Vector[T]) At(index int) T {
	if v.direction {
		return v.Mat.s[v.Mat.stride*index+v.index]
	} else {
		return v.Mat.s[v.Mat.stride*v.index+index]
	}
}

// Iterate over the unmasked values of vector
func (v Vector[T]) All() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		iterate_over := v.Mat.masked_cols
		if v.direction {
			iterate_over = v.Mat.masked_rows
		}
		for ind, masked := range iterate_over {
			if masked {
				continue
			}
			if !yield(ind, v.At(ind)) {
				return
			}
		}
	}
}

// Returns a new matrix with pre-allocated
// backing slice
func NewMat[T any](r, c int) *Mat[T] {
	masked_rows := make([]bool, r)
	masked_cols := make([]bool, c)
	return &Mat[T]{
		s:           make([]T, r*c),
		masked_rows: masked_rows,
		masked_cols: masked_cols,
		stride:      c,
	}
}

// Returns a new matrix by mapping a mat.Dense with a provided function f
func NewMatFromDense[T any](m *mat.Dense, f func(float64) T) *Mat[T] {
	r, c := m.Dims()
	masked_rows := make([]bool, r)
	masked_cols := make([]bool, c)
	new_mat := &Mat[T]{
		s:           make([]T, r*c),
		masked_rows: masked_rows,
		masked_cols: masked_cols,
		stride:      c,
	}
	for ind_r := range r {
		for ind_c := range c {
			new_mat.Set(ind_r, ind_c, f(m.At(ind_r, ind_c)))
		}
	}
	return new_mat
}

// Maps an existing matrix into a new one via f
func Map[T, E any](m *Mat[T], f func(e T, r, c int) E) *Mat[E] {
	new_mat := &Mat[E]{
		s:           make([]E, len(m.masked_cols)*len(m.masked_rows)),
		masked_rows: slices.Clone(m.masked_rows),
		masked_cols: slices.Clone(m.masked_cols),
		stride:      m.stride,
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
	if r >= len(m.masked_rows) || c >= len(m.masked_cols) {
		return fmt.Errorf("Out of bounds")
	}
	m.s[m.stride*r+c] = v
	return nil
}

// Mask selected rows/columns
func (m Mat[T]) Mask(direction Direction, indices ...int) *Mat[T] {
	new_mat := &Mat[T]{
		s:           m.s,
		masked_rows: slices.Clone(m.masked_rows),
		masked_cols: slices.Clone(m.masked_cols),
		stride:      m.stride,
	}

	for _, ind := range indices {
		if direction == Vertical {
			new_mat.masked_cols[ind] = true
		} else {
			new_mat.masked_rows[ind] = true
		}
	}
	return new_mat
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
