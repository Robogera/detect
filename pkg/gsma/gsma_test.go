package gsma

import (
	"fmt"
	"image"
	"iter"
	"testing"
)

var trajectory = []image.Point{
	image.Pt(0, 0),
	image.Pt(100, 100),
	image.Pt(200, 200),
	image.Pt(400, 400),
	image.Pt(500, 500),
	image.Pt(600, 600),
	image.Pt(700, 700),
	image.Pt(800, 800),
	image.Pt(900, 900),
	image.Pt(100, 100),
	image.Pt(100, 100),
	image.Pt(100, 100),
	image.Pt(100, 100),
	image.Pt(100, 100),
}

func Test2dSanity(t *testing.T) {
	sma := NewSMA2d(5)
	for _, p := range trajectory {
		t.Logf("SMA: %v, Data: %s, Newest: %v", sma.Recalc(p), prnt(sma.data.All()), sma.data.Newest())
	}
}

func prnt[T any](it iter.Seq[T]) string {
	s := "[ "
	for e := range it {
		s += fmt.Sprintf("%v ", e)
	}
	s += "]"
	return s
}
