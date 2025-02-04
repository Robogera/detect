package assoc

import (
	"image"
	"math"

	"github.com/Robogera/detect/pkg/gmat"
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

	return assocs
}

func min_sq_dist1(m *gmat.Mat[float64]) (float64, []Assoc) {
	current_min := math.MaxFloat64
	edges := make([]Assoc, 0)

	if m.Size(gmat.Horizontal) < 1 {
		return 0, edges
	}
	var leftmost_vec gmat.Vector[float64]
	leftmost_ind_c := 0
	for ind_c, vec := range m.Vectors(gmat.Vertical) {
		leftmost_vec = vec
		leftmost_ind_c = ind_c
		break
	}
	for ind_r, value := range leftmost_vec.All() {
		sub_min, sub_edges := min_sq_dist1(
			m.Mask(gmat.Vertical, 0).
				Mask(gmat.Horizontal, ind_r))
		if new_min := value + sub_min; new_min < current_min {
			current_min = new_min
			edges = append(sub_edges, Assoc{Pred: ind_r, Det: leftmost_ind_c})
		}
	}
	return current_min, edges
}
