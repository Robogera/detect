package gring

import (
	"iter"
)

type Ring[T any] struct {
	l   int
	s   []T
	pos int
}

func NewRing[T any](l int) *Ring[T] {
	return &Ring[T]{
		l:   0,
		s:   make([]T, l),
		pos: 0,
	}
}

func (r *Ring[T]) Size() int {
	return r.l
}

func (r *Ring[T]) IsFull() bool {
	return r.l == len(r.s)
}

func (r *Ring[T]) Push(e T) {
	r.s[r.pos] = e
	r.pos++
	if r.pos >= len(r.s) {
		r.pos = 0
	}
	if r.l < len(r.s) {
		r.l++
	}
}

func (r *Ring[T]) Newest() T {
	var ret T
	if r.l == 0 {
		return ret
	}
	pos := r.pos - 1
	if pos < 0 {
		pos = len(r.s) + pos
	}
	return r.s[pos]
}

func (r *Ring[T]) Oldest() T {
	var ret T
	if r.l == 0 {
		return ret
	} else if r.l < len(r.s) {
		return r.s[0]
	}
	return r.s[r.pos]
}

func (r *Ring[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		for i := range r.l {
			real_pos := r.pos - 1 - i
			if real_pos < 0 {
				real_pos = r.l + real_pos
			}
			if !yield(r.s[real_pos]) {
				return
			}
		}
	}
}
