package comparator

import (
	"sort"

	"github.com/Spiritreader/avior-go/config"
)

// toleranceFor returns the allowed shortfall in minutes for a recording of the
// given planned runtime. Steps are sorted ascending by BelowMinutes; a step
// without BelowMinutes is the catch-all for anything longer.
func toleranceFor(steps []config.ToleranceStep, runtimeMinutes int) int {
	sorted := make([]config.ToleranceStep, len(steps))
	copy(sorted, steps)
	sort.Slice(sorted, func(i, j int) bool {
		bi, bj := sorted[i].BelowMinutes, sorted[j].BelowMinutes
		if bi == 0 {
			return false // catch-all last
		}
		if bj == 0 {
			return true
		}
		return bi < bj
	})
	for _, step := range sorted {
		if step.BelowMinutes == 0 || runtimeMinutes < step.BelowMinutes {
			return step.Minutes
		}
	}
	last := 0
	for _, step := range sorted {
		if step.Minutes > last {
			last = step.Minutes
		}
	}
	return last
}
