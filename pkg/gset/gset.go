package gset

import (
	"cmp"
	"fmt"
	"iter"
	"strings"
)

type SetNode[T cmp.Ordered] struct {
	value T
	next  *SetNode[T]
}

type Set[T cmp.Ordered] struct {
	head *SetNode[T]
}

func (s *Set[T]) Add(values ...T) {
	for _, value := range values {
		s.add(value)
	}
}

func (s *Set[T]) add(value T) {
	if s.head == nil {
		new_node := &SetNode[T]{value: value, next: nil}
		s.head = new_node
		return
	}

	previous, current := (*SetNode[T])(nil), s.head
	for current != nil {
		if current.value == value {
			return
		} else if current.value > value {
			break
		}
		previous, current = current, current.next
	}

	new_node := &SetNode[T]{value: value, next: current}
	if previous == nil {
		s.head = new_node
	} else {
		previous.next = new_node
	}
}

func (s *Set[T]) Del(values ...T) {
	for _, value := range values {
		s.del(value)
	}
}

func (s *Set[T]) del(value T) {
	previous, current := (*SetNode[T])(nil), s.head
	for current != nil {
		if current.value == value {
			if previous == nil {
				s.head = current.next
			} else {
				previous.next = current.next
			}
			current = nil
			return
		}
		previous, current = current, current.next
	}
}

func (s *Set[T]) Contains(value T) bool {
	_, exists := s.Index(value)
	return exists
}

func (s *Set[T]) Index(value T) (int, bool) {
	for i, current := 0, s.head; current != nil; current, i = current.next, i+1 {
		if current.value == value {
			return i, true
		}
	}
	return 0, false
}

func (s *Set[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		for n := s.head; n != nil; n = n.next {
			if !yield(n.value) {
				return
			}
		}
	}
}

func (s *Set[T]) Sprintf(format string) string {
	b := new(strings.Builder)
	b.WriteString("[ ")
	for e := range s.All() {
		b.WriteString(fmt.Sprintf(format, e))
	}
	b.WriteString("]")
	return b.String()
}

func (s *Set[T]) String() string {
	return s.Sprintf("%v")
}
