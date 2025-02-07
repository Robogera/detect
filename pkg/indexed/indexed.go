package indexed

import "time"

type Indexed[T any] struct {
	t     time.Time
	id    uint64
	value T
}

func NewIndexed[T any](id uint64, t time.Time, value T) Indexed[T] {
	return Indexed[T]{t, id, value}
}

func (i Indexed[T]) Less(other Indexed[T]) bool { return i.id < other.id }
func (i Indexed[T]) Id() uint64                 { return i.id }
func (i Indexed[T]) Time() time.Time            { return i.t }
func (i Indexed[T]) Value() T                   { return i.value }

type Timed[T any] struct {
	t     time.Time
	value T
}

func NewTimed[T any](t time.Time, value T) Timed[T] {
	return Timed[T]{t, value}
}

func (tm Timed[T]) Less(other Timed[T]) bool { return tm.t.Before(other.t) }
func (tm Timed[T]) Time() time.Time          { return tm.t }
func (tm Timed[T]) Value() T                 { return tm.value }
