package gring

import (
	"testing"
)

func TestRing(t *testing.T) {
	r := NewRing[string](5)
	r.Push("asdasd")
	r.Push("bababooey")
	r.Push("cenis")
	r.Push("dendy")
	r.Push("eepy")
	r.Push("faggotini")
	for s := range r.All() {
		t.Log(s)
	}
	for s := range r.All() {
		t.Log(s)
	}
}
