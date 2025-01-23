package gheap

// generic heap implementation
type Ordered[T any] interface {
	Less(T) bool
}

func Less[T Ordered[T]](v, u T) bool {
	return v.Less(u)
}

type Heap[T Ordered[T]] []T

func (h Heap[T]) down(u int) {
	v := u
	if 2*u+1 < len(h) && Less(h[2*u+1], h[v]) {
		v = 2*u + 1
	}
	if 2*u+2 < len(h) && Less(h[2*u+2], h[v]) {
		v = 2*u + 2
	}
	if v != u {
		h[v], h[u] = h[u], h[v]
		h.down(v)
	}
}

func (h Heap[T]) up(u int) {
	for u != 0 && Less(h[u], h[(u-1)/2]) {
		h[(u-1)/2], h[u] = h[u], h[(u-1)/2]
		u = (u - 1) / 2
	}
}

func (h Heap[T]) Len() int      { return len(h) }
func (h Heap[T]) IsEmpty() bool { return len(h) == 0 }

func (h Heap[T]) Init() {
	for i := (len(h) - 1) / 2; i >= 0; i-- {
		h.down(i)
	}
}

func (h *Heap[T]) Push(e T) {
	*h = append(*h, e)
	h.up(len(*h) - 1)
}

func (h *Heap[T]) Pop() T {
	x := (*h)[0]
	n := len(*h)
	(*h)[0], (*h)[n-1] = (*h)[n-1], (*h)[0]
	*h = (*h)[:n-1]
	h.down(0)
	return x
}

func (h Heap[T]) Peek() T {
	x := h[0]
	return x
}
