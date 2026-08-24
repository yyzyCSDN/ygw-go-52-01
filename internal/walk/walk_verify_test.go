package walk_test

import (
	"context"
	"testing"

	"graphstore/internal/store"
	"graphstore/internal/walk"
)

// TestEmptyWalkNoNilPanic verifies an empty graph traversal returns an empty
// non-nil result so downstream consumers never dereference a nil slice.
func TestEmptyWalkNoNilPanic(t *testing.T) {
	st, err := store.New(store.Options{EdgeBuckets: 4, ShardCapacity: 100, LabelCap: 32})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	walker := walk.New(st, walk.Options{BatchSize: 3})
	paths, err := walker.WalkFromLabel(context.Background(), "no-such-label", 3)
	if err != nil {
		t.Fatal(err)
	}
	if paths == nil {
		t.Fatal("empty walk returned a nil result")
	}
	if len(paths) != 0 {
		t.Fatalf("expected an empty result, got %d paths", len(paths))
	}
	var firstVertex string
	if len(paths) > 0 {
		firstVertex = paths[0].Vertices[0]
	}
	if firstVertex != "" {
		t.Fatalf("unexpected vertex %q", firstVertex)
	}
}
