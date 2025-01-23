package gsma

import (
	"errors"
	"fmt"

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
		s.average = s.average + (float32(new_value) - s.average)/float32(l + 1)
		s.data = append(s.data, new_value)
	} else {
		oldest_value := s.data[s.buffer_index]
		s.average = s.average + float32(new_value - oldest_value)/float32(c)
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
