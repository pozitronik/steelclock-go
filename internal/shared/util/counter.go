package util

// CounterRate returns the per-second rate of a cumulative counter between two
// samples taken elapsed seconds apart. A counter lower than its previous value
// was reset (device reset or removed, a member of an aggregated total gone),
// so the sample reports 0 instead of a wrapped unsigned difference.
func CounterRate(current, previous uint64, elapsed float64) float64 {
	if current < previous || elapsed <= 0 {
		return 0
	}
	return float64(current-previous) / elapsed
}
