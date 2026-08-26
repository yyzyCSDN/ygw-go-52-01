package walk_test

import (
	"context"
	"fmt"
	"testing"

	"graphstore/internal/model"
	"graphstore/internal/store"
	"graphstore/internal/walk"
)

// TestWalkVisitedMarkResetOnClose verifies a closed session resets its
// visited marks so the next traversal on the same walker is not truncated.
func TestWalkVisitedMarkResetOnClose(t *testing.T) {
	st, err := store.New(store.Options{EdgeBuckets: 4, ShardCapacity: 100, LabelCap: 32})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.PutVertex(model.NewVertex("c", "")); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if err := st.PutVertex(model.NewVertex(fmt.Sprintf("l%d", i), "")); err != nil {
			t.Fatal(err)
		}
		edge := model.NewEdge(fmt.Sprintf("e%d", i), "c", fmt.Sprintf("l%d", i), "link")
		if err := st.PutEdgeToBucket(edge, 0); err != nil {
			t.Fatal(err)
		}
	}
	walker := walk.New(st, walk.Options{BatchSize: 16})
	first, err := walker.Walk(context.Background(), "c", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 6 {
		t.Fatalf("expected 6 paths in the first walk, got %d", len(first))
	}
	walker.Close()
	second, err := walker.Walk(context.Background(), "c", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 6 {
		t.Fatalf("second walk reused stale visited marks: got %d paths, want 6", len(second))
	}
}
