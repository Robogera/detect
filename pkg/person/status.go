package person

import (
	"fmt"
	"image"
	"time"
)

// type PersonStatus string
//
// const (
//
//	EXPIRED        PersonStatus = "expired"
//	OOB            PersonStatus = "out of bounds"
//	LOST_NO_ASS    PersonStatus = "lost: no association"
//	LOST_LOW_SCORE PersonStatus = "lost: score too low"
//	LOST_HIGH_DST  PersonStatus = "lost: moved too far"
//	UPDATED        PersonStatus = "updated"
//	NEW            PersonStatus = "new"
//
// )
type PersonStatus interface {
	String() string
}

type PersonStatusNoAss struct{}

func (ps PersonStatusNoAss) String() string {
	return "No association found"
}

type PersonStatusNoAssLowScore struct {
	score float64
}

func (ps PersonStatusNoAssLowScore) String() string {
	return fmt.Sprintf("Not associated: low score: %.6f", ps.score)
}

type PersonStatusNoAssTooFar struct {
	state, dest image.Point
	score       float64
}

func (ps PersonStatusNoAssTooFar) String() string {
	return fmt.Sprintf("Not associated: too far. Position: %v, destination: %v, score: %.6f", ps.state, ps.dest, ps.score)
}

type PersonStatusAssociated struct {
	ass   int
	dst   float64
	score float64
}

func (ps PersonStatusAssociated) String() string {
	return fmt.Sprintf("Associated with %d. Moved %.2fpx, score: %.6f", ps.ass, ps.dst, ps.score)
}

type PersonStatusNew struct {
	coord image.Point
}

func (ps PersonStatusNew) String() string {
	return fmt.Sprintf("New: associated with coord: %dx%d", ps.coord.X, ps.coord.Y)
}

type PersonStatusDeletedOOB struct {
	t     time.Duration
	coord image.Point
}

func (ps PersonStatusDeletedOOB) String() string {
	return fmt.Sprintf("Deleted: OOB for %.2f sec. Last known coordinate: %dx%d", ps.t.Seconds(), ps.coord.X, ps.coord.Y)
}

type PersonStatusDeletedNoUpdates struct {
	t time.Duration
}

func (ps PersonStatusDeletedNoUpdates) String() string {
	return fmt.Sprintf("Deleted: lost for %.2f sec", ps.t.Seconds())
}
