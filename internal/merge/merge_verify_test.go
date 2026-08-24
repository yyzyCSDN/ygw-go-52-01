package merge_test

import (
	"fmt"
	"testing"

	"graphstore/internal/merge"
	"graphstore/internal/model"
)

// TestMergeBatchNoOverlapNoGap verifies that paginated batches tile the path
// list exactly: no path repeats and none is skipped.
func TestMergeBatchNoOverlapNoGap(t *testing.T) {
	paths := make([]*model.Path, 0, 7)
	for i := 0; i < 7; i++ {
		path := model.NewPath(fmt.Sprintf("v%d", i))
		paths = append(paths, &path)
	}
	batches := merge.BatchPaths(paths, 3)
	if len(batches) != 3 {
		t.Fatalf("expected 3 batches, got %d", len(batches))
	}
	var flat []*model.Path
	for _, batch := range batches {
		flat = append(flat, batch...)
	}
	if len(flat) != len(paths) {
		t.Fatalf("batches contain %d paths, want %d (overlap or gap)", len(flat), len(paths))
	}
	seen := make(map[*model.Path]bool)
	for index, path := range flat {
		if seen[path] {
			t.Fatalf("path %d is duplicated across batches", index)
		}
		seen[path] = true
		if path != paths[index] {
			t.Fatalf("path %d out of order: got %v", index, path.Vertices)
		}
	}
}
