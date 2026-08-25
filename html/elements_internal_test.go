package html

import "testing"

// TestEndPriorityRankMatchesHtmlEndPriority guards against endPriorityLevels
// and endPriorityRank drifting away from htmlEndPriority: every above-default
// name in htmlEndPriority must have a rank, that rank must map back to the
// same priority value through endPriorityLevels, and endPriorityLevels itself
// must stay in ascending order (topmostAbovePriority relies on it).
func TestEndPriorityRankMatchesHtmlEndPriority(t *testing.T) {
	for i := 1; i < len(endPriorityLevels); i++ {
		if endPriorityLevels[i] <= endPriorityLevels[i-1] {
			t.Fatalf("endPriorityLevels not ascending at index %d: %v", i, endPriorityLevels)
		}
	}
	if numEndPriorityLevels != len(endPriorityLevels) {
		t.Fatalf("numEndPriorityLevels=%d, len(endPriorityLevels)=%d", numEndPriorityLevels, len(endPriorityLevels))
	}

	for name, priority := range htmlEndPriority {
		if priority <= 100 {
			t.Fatalf("htmlEndPriority[%q]=%d is not above the default 100", name, priority)
		}
		rank := endPriorityRank(name)
		if rank == -1 {
			t.Fatalf("endPriorityRank(%q) = -1, want a rank for htmlEndPriority[%q]=%d", name, name, priority)
		}
		if endPriorityLevels[rank] != priority {
			t.Fatalf("endPriorityLevels[endPriorityRank(%q)] = %d, want %d", name, endPriorityLevels[rank], priority)
		}
	}

	if got := endPriorityRank("p"); got != -1 {
		t.Fatalf("endPriorityRank(%q) = %d, want -1 (default priority, not in htmlEndPriority)", "p", got)
	}
}
