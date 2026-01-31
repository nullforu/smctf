package scoring

import "math"

// Ref: https://docs.ctfd.io/docs/custom-challenges/dynamic-value/
func DynamicPoints(initial, minimum, solveCount, decay int) int {
	if minimum > initial {
		minimum = initial
	}

	if decay <= 0 {
		return initial
	}

	value := (((float64(minimum)-float64(initial))/math.Pow(float64(decay), 2))*math.Pow(float64(solveCount), 2) + float64(initial))
	// Avoid floating point precision issues by subtracting a tiny value before ceiling
	value = math.Ceil(value - 1e-9)

	if int(value) < minimum {
		return minimum
	}

	return int(value)
}
