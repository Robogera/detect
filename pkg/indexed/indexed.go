package indexed

type Indexed[T any] struct {
	id    uint64
	value T
}

func NewIndexed[T any](id uint64, value T) Indexed[T] {
	return Indexed[T]{id, value}
}

func (i Indexed[T]) Less(other Indexed[T]) bool { return i.id < other.id }
func (i Indexed[T]) Id() uint64                 { return i.id }
func (i Indexed[T]) Value() T                   { return i.value }
