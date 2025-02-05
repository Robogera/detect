package assoc

import (
	"image"
	"math"

	"github.com/Robogera/detect/pkg/gmat"
	"github.com/Robogera/detect/pkg/seq"
)

type Assoc struct{ Pred, Det int }

func Associate(predicted_points, detected_points []image.Point, threshold float64) []Assoc {
	var assocs []Assoc

	if len(detected_points) < 1 || len(predicted_points) < 1 {
		return assocs
	}

	rows, cols := len(predicted_points), len(detected_points)
	distance_mat := gmat.NewMat[float64](rows, cols)
	for row := range rows {
		for col := range cols {
			distance_mat.Set(row, col, math.Pow(float64(predicted_points[row].X-detected_points[col].X), 2)+math.Pow(float64(predicted_points[row].Y-detected_points[col].Y), 2))
		}
	}

	validity_mat := gmat.Map(
		distance_mat,
		func(v float64, r, c int) bool {
			return v <= threshold
		})

	for ind_r, vec := range validity_mat.Vectors(gmat.Horizontal) {
		total_edges_below_threshold := 0
		for _, value := range vec.All() {
			if value {
				total_edges_below_threshold++
			}
		}
		if total_edges_below_threshold == 0 {
			distance_mat = distance_mat.Mask(gmat.Horizontal, ind_r)
		}
	}

	for ind_c, vec := range validity_mat.Vectors(gmat.Vertical) {
		total_edges_below_threshold := 0
		for _, value := range vec.All() {
			if value {
				total_edges_below_threshold++
			}
		}
		if total_edges_below_threshold == 0 {
			distance_mat = distance_mat.Mask(gmat.Vertical, ind_c)
		}
	}

	_, assocs = min_sq_dist(distance_mat)

	return assocs
}

func min_sq_dist(m *gmat.Mat[float64]) (float64, []Assoc) {
	current_min := math.MaxFloat64
	edges := make([]Assoc, 0)

	if m.Size(gmat.Horizontal) == 1 {
		ind_c, vec := m.Head(gmat.Vertical)
		ind_r, value := seq.MinInd(vec.All())
		return value, []Assoc{Assoc{Pred: ind_r, Det: ind_c}}
	}

	if m.Size(gmat.Vertical) == 1 {
		ind_c, vec := m.Head(gmat.Horizontal)
		ind_r, value := seq.MinInd(vec.All())
		return value, []Assoc{Assoc{Pred: ind_r, Det: ind_c}}
	}

	leftmost_ind_c, leftmost_vec := m.Head(gmat.Vertical)

	for ind_r, value := range leftmost_vec.All() {
		sub_min, sub_edges := min_sq_dist(
			m.Mask(gmat.Vertical, leftmost_ind_c).
				Mask(gmat.Horizontal, ind_r))
		if new_min := value + sub_min; new_min < current_min {
			current_min = new_min
			edges = append(sub_edges, Assoc{Pred: ind_r, Det: leftmost_ind_c})
		}
	}
	return current_min, edges
}
