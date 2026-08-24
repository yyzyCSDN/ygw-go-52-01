package walk_test

import (
	"context"
	"fmt"
	"testing"

	"graphstore/internal/model"
	"graphstore/internal/store"
	"graphstore/internal/walk"
)

// TestDepthLimitAcrossShards verifies the depth budget travels with a path
// when the walk crosses a shard boundary.
func TestDepthLimitAcrossShards(t *testing.T) {
	st, err := store.New(store.Options{EdgeBuckets: 4, ShardCapacity: 100, LabelCap: 32})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	for i := 0; i < 5; i++ {
		if err := st.PutVertex(model.NewVertex(fmt.Sprintf("v%d", i), "")); err != nil {
			t.Fatal(err)
		}
	}
	edges := []struct {
		id, from, to string
		bucket      int
	}{
		{"e01", "v0", "v1", 0},
		{"e12", "v1", "v2", 0},
		{"e23", "v2", "v3", 1},
		{"e34", "v3", "v4", 1},
	}
	for _, spec := range edges {
		if err := st.PutEdgeToBucket(model.NewEdge(spec.id, spec.from, spec.to, "feeds"), spec.bucket); err != nil {
			t.Fatal(err)
		}
	}
	walker := walk.New(st, walk.Options{BatchSize: 4})
	paths, err := walker.Walk(context.Background(), "v0", 3)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		hops := len(path.Vertices) - 1
		if hops > 3 {
			t.Fatalf("path %v exceeded max depth 3 with %d hops", path.Vertices, hops)
		}
	}
	for _, path := range paths {
		if len(path.Vertices) == 5 {
			t.Fatalf("depth-4 path crossed the shard boundary: %v", path.Vertices)
		}
	}
}
