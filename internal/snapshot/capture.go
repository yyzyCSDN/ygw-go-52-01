package snapshot

// Capturer records and restores walk session state against a snapshot of
// the graph. Deleted vertices are dropped from visited marks so a restored
// session never blocks on data that no longer exists.
type Capturer struct {
	alive func(vertexID string) bool
}

// NewCapturer builds a capturer whose alive callback reports whether a
// vertex still exists.
func NewCapturer(alive func(vertexID string) bool) *Capturer {
	return &Capturer{alive: alive}
}

// Capture builds a WalkState from the live visited marks.
func (c *Capturer) Capture(visited map[string]bool, frontier []string, depth int) WalkState {
	marks := make([]string, 0, len(visited))
	for vertexID := range visited {
		marks = append(marks, vertexID)
	}
	return WalkState{Visited: marks, Frontier: append([]string(nil), frontier...), Depth: depth}
}

// PruneDeletes removes visited marks for vertices that no longer exist.
func (c *Capturer) PruneDeletes(visited map[string]bool) int {
	if c.alive == nil {
		return 0
	}
	removed := 0
	for vertexID := range visited {
		if !c.alive(vertexID) {
			delete(visited, vertexID)
			removed++
		}
	}
	return removed
}
