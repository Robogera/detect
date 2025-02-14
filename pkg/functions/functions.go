package functions

import "math"

func Gaussian(score float64, distance float64, sigma float64) float64 {
	return score * math.Exp(-math.Pow(distance*2, 2)/2/math.Pow(sigma, 2))
}
