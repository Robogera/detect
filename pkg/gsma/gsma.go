package gsma

import (
	"errors"
	"fmt"
	"image"

	"github.com/Robogera/detect/pkg/gring"
	"golang.org/x/exp/constraints"
)

var (
	ERR_VALUE = errors.New("Bad value")
)

type Number interface {
	constraints.Float | constraints.Integer
}

type SMA[T Number] struct {
	data         []T
	buffer_index uint
	average      float32
}

func NewSMA[T Number](capacity uint) (*SMA[T], error) {
	if capacity < 3 {
		return nil, fmt.Errorf("Invalid capacity: %d. Error: %w", capacity, ERR_VALUE)
	}
	return &SMA[T]{
		data:         make([]T, 0, capacity),
		buffer_index: 0,
		average:      0,
	}, nil
}

func (s *SMA[T]) Recalc(new_value T) {
	l, c := len(s.data), cap(s.data)
	if l < c {
		s.average = s.average + (float32(new_value)-s.average)/float32(l+1)
		s.data = append(s.data, new_value)
	} else {
		oldest_value := s.data[s.buffer_index]
		s.average = s.average + float32(new_value-oldest_value)/float32(c)
		s.data[s.buffer_index] = new_value
		s.buffer_index++
		if s.buffer_index >= uint(l) {
			s.buffer_index = 0
		}
	}
}

func (s *SMA[T]) Show() float32 {
	return s.average
}

type SMA2d struct {
	data    *gring.Ring[image.Point]
	average image.Point
}

func NewSMA2d(n int) *SMA2d {
	return &SMA2d{
		average: image.Pt(0, 0),
		data:    gring.NewRing[image.Point](n),
	}
}

func (s2 *SMA2d) Recalc(p image.Point) image.Point {
	oldest := s2.data.Oldest()
	if !s2.data.IsFull() {
		s2.average = s2.average.Add(p.Sub(s2.average).Div(1+s2.data.Size()))
	} else {
		s2.average = s2.average.Add(p.Sub(oldest).Div(s2.data.Size()))
	}
	s2.data.Push(p)
	return s2.average
}

func (s2 *SMA2d) Peek() image.Point {
	return s2.data.Newest()
}
