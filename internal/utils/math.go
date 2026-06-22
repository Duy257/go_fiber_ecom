package utils

import "math"

// CeilToNearest rounds value up to the nearest multiple of nearest.
// Example: CeilToNearest(25300, 1000) → 26000
func CeilToNearest(value, nearest float64) float64 {
	return math.Ceil(value/nearest) * nearest
}